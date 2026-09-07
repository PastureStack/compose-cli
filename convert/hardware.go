package convert

import (
	"fmt"
	"strings"

	"github.com/PastureStack/compose-cli/config"
	"github.com/moby/moby/api/types/container"
)

func hardwareRequests(c *config.ServiceConfig) ([]container.DeviceRequest, error) {
	if c.ShmSize < 0 || (c.ShmSize > 0 && c.Ipc != "" && c.Ipc != "private" && c.Ipc != "shareable") {
		return nil, fmt.Errorf("shm_size must be non-negative and requires private or shareable IPC")
	}
	if strings.Trim(c.Runtime, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-") != "" {
		return nil, fmt.Errorf("runtime must be a registered runtime name")
	}
	if c.PidsLimit != nil && *c.PidsLimit < -1 {
		return nil, fmt.Errorf("pids_limit must be non-negative or -1")
	}
	requests := []container.DeviceRequest{}
	items := []config.GPURequest(c.GPUs)
	if c.Deploy != nil {
		if len(items) > 0 && len(c.Deploy.Resources.Reservations.Devices) > 0 {
			return nil, fmt.Errorf("use either gpus or deploy device reservations, not both")
		}
		items = append(items, c.Deploy.Resources.Reservations.Devices...)
	}
	for _, item := range items {
		if item.Count != nil && len(item.DeviceIDs) > 0 {
			return nil, fmt.Errorf("GPU count and device_ids are mutually exclusive")
		}
		count := -1
		if len(item.DeviceIDs) > 0 {
			count = 0
		} else if item.Count != nil {
			count = int(*item.Count)
		}
		if count < -1 || (count == 0 && len(item.DeviceIDs) == 0) {
			return nil, fmt.Errorf("invalid GPU count")
		}
		caps := append([]string{}, item.Capabilities...)
		if len(c.GPUs) > 0 {
			gpu := false
			for _, cap := range caps {
				if cap == "gpu" {
					gpu = true
				}
			}
			if !gpu {
				caps = append(caps, "gpu")
			}
		}
		if len(caps) == 0 {
			return nil, fmt.Errorf("device reservations require capabilities")
		}
		for _, cap := range caps {
			if strings.TrimSpace(cap) == "" {
				return nil, fmt.Errorf("capabilities cannot be blank")
			}
		}
		seen := map[string]bool{}
		for _, id := range item.DeviceIDs {
			if strings.TrimSpace(id) == "" || seen[id] {
				return nil, fmt.Errorf("device_ids must be unique and non-empty")
			}
			seen[id] = true
		}
		requests = append(requests, container.DeviceRequest{Driver: item.Driver, Count: count, DeviceIDs: item.DeviceIDs, Capabilities: [][]string{caps}, Options: item.Options})
	}
	return requests, nil
}
