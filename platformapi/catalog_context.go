package platformapi

import "github.com/PastureStack/compose-cli/project"

// CatalogProjectName exposes the existing executor's context normalization to
// its private preflight. It performs no client initialization or API request.
func CatalogProjectName(name string) string {
	context := &Context{Context: project.Context{ProjectName: name}}
	return context.sanitizedProjectName()
}

func CatalogSidekickOrder(p *project.Project, name string) ([]string, error) {
	info, err := NewSidekickInfo(p)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), info.primariesToSidekicks[name]...), nil
}

func CatalogSidekickPrimaries(p *project.Project) (map[string]string, error) {
	info, err := NewSidekickInfo(p)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	for name, primaries := range info.sidekickToPrimaries {
		if len(primaries) == 1 {
			result[name] = primaries[0]
		}
	}
	return result, nil
}
