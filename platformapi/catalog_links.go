package platformapi

import (
	"sort"
	"strings"

	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/utils"
)

func parseServiceLink(raw string) Link {
	parts := strings.SplitN(raw, ":", 2)
	value := Link{ServiceName: strings.TrimSpace(parts[0])}
	if len(parts) == 2 {
		value.Alias = strings.TrimSpace(parts[1])
	}
	return value
}

func serviceLinksFromResolved(links map[Link]string) []client.ServiceLink {
	result := []client.ServiceLink{}
	for link, id := range links {
		if utils.IsRegionService(link.ServiceName) {
			result = append(result, client.ServiceLink{Name: link.Alias, Service: link.ServiceName})
		} else {
			result = append(result, client.ServiceLink{Name: link.Alias, ServiceId: id})
		}
	}
	return result
}

func sortCatalogLinks(links []client.ServiceLink) {
	sort.Slice(links, func(i, j int) bool {
		if links[i].ServiceId != links[j].ServiceId {
			return links[i].ServiceId < links[j].ServiceId
		}
		if links[i].Name != links[j].Name {
			return links[i].Name < links[j].Name
		}
		return links[i].Service < links[j].Service
	})
}

// Same parser and request DTO as normal setservicelinks. Only this new graph's
// services are supported; external/region/cross-stack links require a later
// typed reference adapter rather than broader service-Key reads.
func CatalogServiceLinkSkeleton(raw []string, graph map[string]string) (*client.SetServiceLinksInput, error) {
	resolved := map[Link]string{}
	for _, value := range raw {
		link := parseServiceLink(value)
		primary, exists := graph[link.ServiceName]
		if !exists || utils.IsRegionService(link.ServiceName) || len(link.Alias) > 128 || strings.ContainsAny(link.Alias, "\x00\r\n") {
			return nil, errCatalogGraph
		}
		if primary != "" {
			continue
		} // Sidekicks have no independent service row.
		resolved[link] = "$service:" + link.ServiceName
	}
	links := serviceLinksFromResolved(resolved)
	sortCatalogLinks(links)
	return &client.SetServiceLinksInput{ServiceLinks: links}, nil
}
