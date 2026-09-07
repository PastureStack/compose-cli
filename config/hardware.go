package config

import (
	"fmt"
	"strconv"
)

type DeviceCount int

func (c DeviceCount) MarshalYAML() (interface{}, error) {
	if c == -1 {
		return "all", nil
	}
	if c < 1 {
		return nil, fmt.Errorf("invalid GPU count")
	}
	return int(c), nil
}
func (c *DeviceCount) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	if value == "all" {
		*c = -1
		return nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return fmt.Errorf("GPU count must be a positive integer or all")
	}
	*c = DeviceCount(n)
	return nil
}

type GPURequest struct {
	Driver       string            `yaml:"driver,omitempty"`
	Count        *DeviceCount      `yaml:"count,omitempty"`
	DeviceIDs    []string          `yaml:"device_ids,omitempty"`
	Capabilities []string          `yaml:"capabilities,omitempty"`
	Options      map[string]string `yaml:"options,omitempty"`
}
type GPURequests []GPURequest

func (g *GPURequests) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var scalar string
	if unmarshal(&scalar) == nil {
		if scalar != "all" {
			return fmt.Errorf("gpus scalar must be all")
		}
		count := DeviceCount(-1)
		*g = GPURequests{{Count: &count}}
		return nil
	}
	type plain GPURequests
	var requests plain
	if err := unmarshal(&requests); err != nil {
		return err
	}
	*g = GPURequests(requests)
	return nil
}

// Only the device reservation portion of deploy is implemented. The schema
// rejects unsupported deploy directives instead of silently ignoring them.
type HardwareDeployment struct {
	Resources struct {
		Reservations struct {
			Devices []GPURequest `yaml:"devices,omitempty"`
		} `yaml:"reservations,omitempty"`
	} `yaml:"resources,omitempty"`
}
