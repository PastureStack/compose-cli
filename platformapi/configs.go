package platformapi

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/convert"
	"github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/utils"
	"github.com/PastureStack/compose-cli/yaml"
	"github.com/moby/moby/api/types/container"
	"github.com/sirupsen/logrus"
)

func createLaunchConfigs(r *PlatformService) (client.LaunchConfig, []client.SecondaryLaunchConfig, error) {
	secondaryLaunchConfigs := []client.SecondaryLaunchConfig{}
	launchConfig, err := createLaunchConfig(r, r.Name(), r.Config())
	if err != nil {
		return launchConfig, nil, err
	}
	launchConfig.HealthCheck = r.HealthCheck("")

	if secondaries, ok := r.Context().SidekickInfo.primariesToSidekicks[r.Name()]; ok {
		for _, secondaryName := range secondaries {
			serviceConfig, ok := r.Context().Project.ServiceConfigs.Get(secondaryName)
			if !ok {
				return launchConfig, nil, fmt.Errorf("Failed to find sidekick: %s", secondaryName)
			}

			launchConfig, err := createLaunchConfig(r, secondaryName, serviceConfig)
			if err != nil {
				return launchConfig, nil, err
			}
			launchConfig.HealthCheck = r.HealthCheck(secondaryName)

			var secondaryLaunchConfig client.SecondaryLaunchConfig
			utils.Convert(launchConfig, &secondaryLaunchConfig)
			secondaryLaunchConfig.Name = secondaryName

			if secondaryLaunchConfig.Labels == nil {
				secondaryLaunchConfig.Labels = map[string]interface{}{}
			}
			secondaryLaunchConfigs = append(secondaryLaunchConfigs, secondaryLaunchConfig)
		}
	}

	return launchConfig, secondaryLaunchConfigs, nil
}

func createLaunchConfig(r *PlatformService, name string, serviceConfig *config.ServiceConfig) (client.LaunchConfig, error) {
	var result client.LaunchConfig
	if r.context.CatalogLaunchDelegation != "" {
		copy, err := CatalogCanonicalConfig(serviceConfig)
		if err != nil {
			return result, err
		}
		serviceConfig = copy
	}

	schemasUrl := strings.SplitN(r.Context().Client.GetSchemas().Links["self"], "/schemas", 2)[0]
	scriptsUrl := schemasUrl + "/scripts/transform"

	tempImage := serviceConfig.Image
	tempLabels := serviceConfig.Labels
	newLabels := yaml.SliceorMap{}
	if config.IsLoadBalancerServiceImage(serviceConfig.Image) {
		// Lookup default load balancer image
		lbImageSetting, err := r.Client().Setting.ById("lb.instance.image")
		if err != nil {
			return result, err
		}
		serviceConfig.Image = lbImageSetting.Value

		// Strip off legacy load balancer labels
		for k, v := range serviceConfig.Labels {
			if !strings.HasPrefix(k, "io.rancher.loadbalancer") && !strings.HasPrefix(k, "io.rancher.service.selector") {
				newLabels[k] = v
			}
		}
		serviceConfig.Labels = newLabels
	}

	config, hostConfig, err := convert.Convert(serviceConfig, r.context.Context)
	if err != nil {
		return result, err
	}

	serviceConfig.Image = tempImage
	serviceConfig.Labels = tempLabels

	dockerContainer := &ContainerInspect{
		Config:     config,
		HostConfig: hostConfig,
	}

	dockerContainer.HostConfig.NetworkMode = container.NetworkMode("")
	dockerContainer.Name = "/" + name

	data := map[string]interface{}{}
	err = r.Context().Client.Post(scriptsUrl, dockerContainer, &data)
	if err != nil {
		return result, err
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(buf, &result); err != nil {
		return result, err
	}

	if err = setupBuild(r, name, &result, serviceConfig); err != nil {
		return result, err
	}
	if err = setupSecrets(r, name, &result, serviceConfig); err != nil {
		return result, err
	}
	finishLaunchConfig(&result, serviceConfig, hostConfig.VolumeDriver)
	return result, err
}

func finishLaunchConfig(result *client.LaunchConfig, serviceConfig *config.ServiceConfig, volumeDriver string) {
	result.VolumeDriver = volumeDriver
	result.MilliCpuReservation = int64(serviceConfig.MilliCpuReservation)
	setupNetworking(serviceConfig.NetworkMode, result)
	setupVolumesFrom(serviceConfig.VolumesFrom, result)
	if result.Labels == nil {
		result.Labels = map[string]interface{}{}
	}

	result.Kind = serviceConfig.Type
	result.Vcpu = int64(serviceConfig.Vcpu)
	result.Userdata = serviceConfig.Userdata
	result.MemoryMb = int64(serviceConfig.Memory)
	result.Disks = serviceConfig.Disks
	result.DrainTimeoutMs = int64(serviceConfig.DrainTimeoutMs)

	if strings.EqualFold(result.Kind, "virtual_machine") || strings.EqualFold(result.Kind, "virtualmachine") {
		result.Kind = "virtualMachine"
	}

	if result.LogConfig == nil {
		result.LogConfig = &client.LogConfig{}
	}
	if result.LogConfig.Config == nil {
		result.LogConfig.Config = map[string]interface{}{}
	}

}

// Only delegated, prevalidated creation canonicalizes the semantically keyed
// environment list. Ordinary Compose order and duplicate behavior is unchanged.
func CatalogCanonicalConfig(source *config.ServiceConfig) (*config.ServiceConfig, error) {
	copy := *source
	copy.Environment = append([]string(nil), source.Environment...)
	seen := map[string]bool{}
	for _, entry := range copy.Environment {
		key := strings.SplitN(entry, "=", 2)[0]
		if seen[key] {
			return nil, fmt.Errorf("CatalogLaunchUnsupported")
		}
		seen[key] = true
	}
	sort.Strings(copy.Environment)
	return &copy, nil
}

func setupNetworking(netMode string, launchConfig *client.LaunchConfig) {
	if netMode == "" {
		launchConfig.NetworkMode = "managed"
	} else if container.IpcMode(netMode).IsContainer() {
		// For some reason NetworkMode object is gone runconfig, but IpcMode works the same for this
		launchConfig.NetworkMode = "container"
		launchConfig.NetworkLaunchConfig = strings.TrimPrefix(netMode, "container:")
	} else {
		launchConfig.NetworkMode = netMode
	}
}

func setupVolumesFrom(volumesFrom []string, launchConfig *client.LaunchConfig) {
	launchConfig.DataVolumesFromLaunchConfigs = volumesFrom
}

func setupBuild(r *PlatformService, name string, result *client.LaunchConfig, serviceConfig *config.ServiceConfig) error {
	if serviceConfig.Build.Context != "" {
		result.Build = &client.DockerBuild{
			Remote:     serviceConfig.Build.Context,
			Dockerfile: serviceConfig.Build.Dockerfile,
		}

		needBuild := true
		if config.IsValidRemote(serviceConfig.Build.Context) {
			needBuild = false
		}

		if needBuild {
			image, url, err := Upload(r.Context(), name)
			if err != nil {
				return err
			}
			logrus.Infof("Build for %s available at %s", name, url)
			serviceConfig.Build.Context = url

			if serviceConfig.Image == "" {
				serviceConfig.Image = image
			}

			result.Build = &client.DockerBuild{
				Context:    url,
				Dockerfile: serviceConfig.Build.Dockerfile,
			}
			result.ImageUuid = "docker:" + image
		} else if result.ImageUuid == "" {
			result.ImageUuid = fmt.Sprintf("docker:%s_%s_%d", r.Context().ProjectName, name, time.Now().UnixNano()/int64(time.Millisecond))
		}
	}

	return nil
}

func setupSecrets(r *PlatformService, name string, result *client.LaunchConfig, serviceConfig *config.ServiceConfig) error {
	return appendResolvedSecrets(result, r.serviceConfig.Secrets, func(source string) (string, error) {
		existingSecrets, err := r.Client().Secret.List(&client.ListOpts{
			Filters: map[string]interface{}{
				"name": source,
			},
		})
		if err != nil {
			return "", err
		}
		if len(existingSecrets.Data) == 0 {
			return "", fmt.Errorf("Failed to find secret %s", source)
		}
		return existingSecrets.Data[0].Id, nil
	})
}

func appendResolvedSecrets(result *client.LaunchConfig, references config.SecretReferences, resolve func(string) (string, error)) error {
	for _, secret := range references {
		id, err := resolve(secret.Source)
		if err != nil {
			return err
		}
		result.Secrets = append(result.Secrets, client.SecretReference{
			SecretId: id,
			Name:     secret.Target,
			Uid:      secret.Uid,
			Gid:      secret.Gid,
			Mode:     string(secret.Mode),
		})
	}
	return nil
}
