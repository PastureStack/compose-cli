// Package cataloglaunch is the Compose owner's private, non-networking
// materializer. It uses the normal parser and Docker conversion; it never
// authorizes a caller or creates a control-plane resource.
package cataloglaunch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/convert"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/lookup"
	"github.com/PastureStack/compose-cli/platformapi"
	"github.com/PastureStack/compose-cli/project"
	"github.com/PastureStack/compose-cli/template"
	"github.com/moby/moby/api/types/container"
)

const MaxInspectInputBytes = 1536 * 1024
const MaxInputBytes = 10 * 1024 * 1024
const MaxOutputBytes = 8 * 1024 * 1024
const MaxExpandedConfigBytes = 1024 * 1024
const Command = "--catalog-launch-preview"

var ErrInput = errors.New("CatalogLaunchInputInvalid")
var ErrUnsupported = errors.New("CatalogLaunchUnsupported")
var namePattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,63}$`)

// All input/output here is server-private, including normalized container
// configuration. Public plans only copy independently allowlisted counts.
type Input struct {
	Phase                    string                     `json:"phase,omitempty"`
	Name                     string                     `json:"name"`
	EnvironmentName          string                     `json:"environmentName"`
	DockerCompose            string                     `json:"dockerCompose"`
	RancherCompose           string                     `json:"rancherCompose"`
	Variables                map[string]interface{}     `json:"variables"`
	TransformedLaunchConfigs map[string]json.RawMessage `json:"transformedLaunchConfigs,omitempty"`
	ResolvedSecrets          map[string]string          `json:"resolvedSecrets,omitempty"`
}

type Inspect struct {
	Name       string                `json:"Name"`
	Config     *convert.DockerConfig `json:"Config"`
	HostConfig *container.HostConfig `json:"HostConfig"`
}

type Service struct {
	Name           string                        `json:"name"`
	PrimaryName    string                        `json:"primaryName,omitempty"`
	ResourceType   string                        `json:"resourceType,omitempty"`
	CreateSkeleton *platformapi.CompositeService `json:"createSkeleton,omitempty"`
	Config         *config.ServiceConfig         `json:"config"`
	Inspect        Inspect                       `json:"dockerInspect"`
}

type VolumeRequest struct {
	Name           string                 `json:"name"`
	ResourceType   string                 `json:"resourceType"`
	CreateSkeleton *client.VolumeTemplate `json:"createSkeleton"`
}

type Output struct {
	Phase           string                              `json:"phase"`
	FormatVersion   int                                 `json:"formatVersion"`
	DockerCompose   string                              `json:"dockerCompose"`
	RancherCompose  string                              `json:"rancherCompose"`
	Variables       map[string]interface{}              `json:"variables"`
	Services        []Service                           `json:"services"`
	Volumes         map[string]*config.VolumeConfig     `json:"volumes"`
	Networks        map[string]*config.NetworkConfig    `json:"networks"`
	Secrets         map[string]*config.SecretConfig     `json:"secrets"`
	Hosts           map[string]*config.HostConfig       `json:"hosts"`
	Dependencies    map[string]*config.DependencyConfig `json:"dependencies"`
	VolumeRequests  []VolumeRequest                     `json:"volumeRequests"`
	ServiceRequests []ServiceRequest                    `json:"serviceRequests"`
}

type ServiceRequest struct {
	Name            string                       `json:"name"`
	ResourceType    string                       `json:"resourceType"`
	CreateBody      interface{}                  `json:"createBody"`
	SetServiceLinks *client.SetServiceLinksInput `json:"setServiceLinks"`
}

type catalogEnvironment struct {
	source *lookup.MapEnvLookup
	used   int
	failed bool
}

func (e *catalogEnvironment) Variables() map[string]string { return e.source.Variables() }
func (e *catalogEnvironment) Lookup(key string, c *config.ServiceConfig) []string {
	values := e.source.Lookup(key, c)
	if len(values) == 0 {
		e.failed = true
		return []string{key + "="}
	}
	for _, value := range values {
		e.used += len(value)
		if e.used > MaxExpandedConfigBytes {
			e.failed = true
			return []string{key + "="}
		}
	}
	return values
}

func Prepare(input Input) (*Output, error) {
	if input.Phase != "" && input.Phase != "inspect" && input.Phase != "finalize" {
		return nil, ErrInput
	}
	if input.Phase != "finalize" && (len(input.TransformedLaunchConfigs) > 0 || len(input.ResolvedSecrets) > 0) {
		return nil, ErrInput
	}
	if !namePattern.MatchString(input.Name) || input.DockerCompose == "" || len(input.DockerCompose)+len(input.RancherCompose) > template.MaxCatalogTemplateBytes ||
		len(input.Variables) > 100 || len(input.EnvironmentName) > 512 {
		return nil, ErrInput
	}
	for name, value := range input.Variables {
		if name == "" || len(name) > 128 || strings.ContainsAny(name, "\x00\r\n") {
			return nil, ErrInput
		}
		switch v := value.(type) {
		case string:
			if len(v) > 16384 || strings.ContainsRune(v, '\x00') {
				return nil, ErrInput
			}
		case bool, json.Number, float64, int, int64:
		default:
			return nil, ErrInput
		}
	}
	explicitVariables := map[string]interface{}{}
	for name, value := range (&lookup.MapEnvLookup{Env: input.Variables}).Variables() {
		explicitVariables[name] = value
	}
	env := &catalogEnvironment{source: &lookup.MapEnvLookup{Env: explicitVariables}}
	catalogInfo, err := lookup.ParseCatalogConfig([]byte(input.RancherCompose))
	if err != nil {
		return nil, ErrUnsupported
	}
	ownerName := platformapi.CatalogProjectName(input.Name)
	stack := template.StackInfo{Name: ownerName, Version: catalogInfo.Version}
	environment := template.EnvironmentInfo{Name: input.EnvironmentName}
	docker, err := template.ApplyCatalog([]byte(input.DockerCompose), stack, environment, env.Variables())
	if err != nil {
		return nil, ErrUnsupported
	}
	rancher, err := template.ApplyCatalog([]byte(input.RancherCompose), stack, environment, env.Variables())
	if err != nil {
		return nil, ErrUnsupported
	}
	declaredSecrets := map[string]bool{}
	for _, frozen := range [][]byte{docker, rancher} {
		raw, err := config.CreateRawConfig(frozen)
		if err != nil {
			return nil, ErrUnsupported
		}
		for name, service := range raw.Services {
			// The normal parser does not support every YAML representation of
			// extends. Never approve a partially ignored external definition.
			if _, found := service["extends"]; found {
				return nil, ErrUnsupported
			}
			if references, found := service["secrets"]; found && references != nil {
				declaredSecrets[name] = true
			}
		}
	}
	ctx := project.Context{ProjectName: ownerName, EnvironmentName: input.EnvironmentName, Version: catalogInfo.Version,
		ComposeBytes: [][]byte{docker, rancher}, EnvironmentLookup: env, ResourceLookup: lookup.NewFileResourceLookup()}
	// Use the same parser and environment resolution used by executor projects.
	// No service/volume factory is installed, so parsing cannot call the API.
	p := project.NewProject(&ctx)
	if err := p.Parse(); err != nil {
		return nil, ErrUnsupported
	}
	if env.failed {
		return nil, ErrUnsupported
	}
	if p.ServiceConfigs.Len() == 0 || p.ServiceConfigs.Len() > 100 || p.ContainerConfigs.Len() > 0 || len(p.VolumeConfigs) > 100 ||
		len(p.NetworkConfigs) > 0 || len(p.SecretConfigs) > 100 || len(p.HostConfigs) > 0 || len(p.DependencyConfigs) > 0 {
		return nil, ErrUnsupported
	}
	for _, secret := range p.SecretConfigs {
		if secret == nil || secret.File != "" || secret.External == "" {
			return nil, ErrUnsupported
		}
	}
	primaries, err := platformapi.CatalogSidekickPrimaries(p)
	if err != nil {
		return nil, ErrUnsupported
	}
	for sidekick, primary := range primaries {
		if !p.ServiceConfigs.Has(sidekick) || !p.ServiceConfigs.Has(primary) || primaries[primary] != "" {
			return nil, ErrUnsupported
		}
	}
	graphNames := map[string]string{}
	for _, name := range p.ServiceConfigs.Keys() {
		graphNames[name] = primaries[name]
	}
	linkSkeletons := map[string]*client.SetServiceLinksInput{}
	names := p.ServiceConfigs.Keys()
	sort.Strings(names)
	result := &Output{Phase: "inspect", FormatVersion: 1, DockerCompose: string(docker), RancherCompose: string(rancher), Variables: explicitVariables,
		Services: []Service{}, Volumes: p.VolumeConfigs, Networks: p.NetworkConfigs, Secrets: p.SecretConfigs, Hosts: p.HostConfigs, Dependencies: p.DependencyConfigs, VolumeRequests: []VolumeRequest{}}
	volumes := make([]string, 0, len(p.VolumeConfigs))
	for name := range p.VolumeConfigs {
		volumes = append(volumes, name)
	}
	sort.Strings(volumes)
	for _, name := range volumes {
		if !namePattern.MatchString(name) {
			return nil, ErrUnsupported
		}
		result.VolumeRequests = append(result.VolumeRequests, VolumeRequest{Name: name, ResourceType: "volumeTemplate", CreateSkeleton: platformapi.CatalogVolumeCreateSkeleton(name, p.VolumeConfigs[name])})
	}
	for _, name := range names {
		if !namePattern.MatchString(name) {
			return nil, ErrUnsupported
		}
		service, _ := p.ServiceConfigs.Get(name)
		if service.Labels["io.rancher.service.selector.container"] != "" || service.Labels["io.rancher.service.selector.link"] != "" {
			return nil, ErrUnsupported
		}
		if len(service.ExternalLinks) > 0 {
			return nil, ErrUnsupported
		}
		links, err := platformapi.CatalogServiceLinkSkeleton(service.Links, graphNames)
		if err != nil {
			return nil, ErrUnsupported
		}
		linkSkeletons[name] = links
		if declaredSecrets[name] && len(service.Secrets) == 0 {
			return nil, ErrUnsupported
		}
		// KVM and external driver/provider lifecycles stay with their human UI.
		if service.Image == "" || service.Type != "" && service.Type != "container" || service.StorageDriver != nil || service.NetworkDriver != nil ||
			config.IsLoadBalancerServiceImage(service.Image) || service.LbConfig != nil || service.LegacyLoadBalancerConfig != nil || service.Build.Context != "" {
			return nil, ErrUnsupported
		}
		copy := *service
		copy.Environment = append([]string(nil), service.Environment...)
		if primaries[name] != "" {
			for _, value := range copy.Environment {
				if !strings.Contains(value, "=") {
					return nil, ErrUnsupported
				}
			}
		}
		if primaries[name] == "" {
			copy.Environment = resolveEnvironment(service.Environment, env, service)
		}
		if env.failed {
			return nil, ErrUnsupported
		}
		canonical, err := platformapi.CatalogCanonicalConfig(&copy)
		if err != nil {
			return nil, ErrUnsupported
		}
		c, h, err := convert.Convert(canonical, ctx)
		if err != nil {
			return nil, ErrUnsupported
		}
		// The existing /scripts/transform contract consumes Docker Inspect;
		// network launch references are retained separately in normalized config.
		h.NetworkMode = ""
		entry := Service{Name: name, PrimaryName: primaries[name], Config: canonical, Inspect: Inspect{Name: "/" + name, Config: c, HostConfig: h}}
		if entry.PrimaryName == "" {
			entry.ResourceType, entry.CreateSkeleton = platformapi.CatalogServiceCreateSkeleton(name, &copy)
		}
		result.Services = append(result.Services, entry)
	}
	if input.Phase == "finalize" {
		if len(input.TransformedLaunchConfigs) != len(result.Services) {
			return nil, ErrInput
		}
		parts := map[string]platformapi.CatalogLaunchPart{}
		neededSecrets := map[string]bool{}
		for _, entry := range result.Services {
			transformed, ok := input.TransformedLaunchConfigs[entry.Name]
			if !ok || len(transformed) == 0 || string(transformed) == "null" {
				return nil, ErrInput
			}
			parts[entry.Name] = platformapi.CatalogLaunchPart{Name: entry.Name, Config: entry.Config, Transform: transformed, VolumeDriver: entry.Inspect.HostConfig.VolumeDriver}
			if entry.PrimaryName == "" {
				for _, reference := range entry.Config.Secrets {
					neededSecrets[reference.Source] = true
				}
			}
		}
		if len(input.ResolvedSecrets) != len(neededSecrets) {
			return nil, ErrInput
		}
		for name, id := range input.ResolvedSecrets {
			if !neededSecrets[name] || id == "" || len(id) > 128 || strings.ContainsAny(id, "/\\: \t\r\n\x00") {
				return nil, ErrInput
			}
		}
		result.ServiceRequests = []ServiceRequest{}
		for _, entry := range result.Services {
			if entry.PrimaryName != "" {
				continue
			}
			order, err := platformapi.CatalogSidekickOrder(p, entry.Name)
			if err != nil {
				return nil, ErrUnsupported
			}
			secondary := []platformapi.CatalogLaunchPart{}
			for _, name := range order {
				secondary = append(secondary, parts[name])
			}
			kind, body, err := platformapi.CatalogFinalizeService(parts[entry.Name], secondary, input.ResolvedSecrets)
			if err != nil {
				return nil, ErrUnsupported
			}
			links := linkSkeletons[entry.Name]
			if kind == "externalService" {
				links = nil
			}
			result.ServiceRequests = append(result.ServiceRequests, ServiceRequest{Name: entry.Name, ResourceType: kind, CreateBody: platformapi.CatalogServiceRequestPayload(kind, body), SetServiceLinks: links})
		}
		result.Phase = "finalize"
	}
	encoded, err := json.Marshal(result)
	if err != nil || len(encoded) > MaxOutputBytes {
		return nil, ErrUnsupported
	}
	return result, nil
}

func resolveEnvironment(source []string, env config.EnvironmentLookup, service *config.ServiceConfig) []string {
	result := []string{}
	for _, entry := range source {
		if strings.Contains(entry, "=") {
			result = append(result, entry)
		} else {
			result = append(result, env.Lookup(entry, service)...)
		}
	}
	return result
}

// Run has no URL, path, credential or arbitrary command parameter. Its caller
// must enforce a subprocess deadline and discard stderr. Every failure has a
// fixed code, with no YAML, answer, parser text or host path echoed.
func Run(reader io.Reader, writer io.Writer) (status int) {
	defer func() {
		if recover() != nil {
			status = writeFailure(writer, ErrUnsupported)
		}
	}()
	raw, err := io.ReadAll(io.LimitReader(reader, MaxInputBytes+1))
	if err != nil || len(raw) > MaxInputBytes {
		return writeFailure(writer, ErrInput)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var input Input
	if decoder.Decode(&input) != nil {
		return writeFailure(writer, ErrInput)
	}
	if input.Phase != "finalize" && len(raw) > MaxInspectInputBytes {
		return writeFailure(writer, ErrInput)
	}
	var extra interface{}
	if decoder.Decode(&extra) != io.EOF {
		return writeFailure(writer, ErrInput)
	}
	result, err := Prepare(input)
	if err != nil {
		return writeFailure(writer, err)
	}
	if json.NewEncoder(writer).Encode(result) != nil {
		return 1
	}
	return 0
}

func writeFailure(writer io.Writer, err error) int {
	code := ErrUnsupported.Error()
	if errors.Is(err, ErrInput) {
		code = ErrInput.Error()
	}
	_ = json.NewEncoder(writer).Encode(struct {
		Code string `json:"code"`
	}{code})
	return 1
}
