package convert

import (
	"encoding/json"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	"github.com/PastureStack/compose-cli/utils"
	"gopkg.in/yaml.v3"
)

func TestHardwareComposeRoundTrip(t *testing.T) {
	for _, value := range []string{
		"gpus: all\n",
		"gpus:\n - driver: nvidia\n   count: 2\n",
		"deploy:\n resources:\n  reservations:\n   devices:\n    - driver: nvidia\n      device_ids: [GPU-example]\n      capabilities: [gpu]\n",
	} {
		var before config.ServiceConfigV1
		if err := yaml.Unmarshal([]byte("image: example\nruntime: nvidia\nshm_size: 2g\nports: ['127.0.0.1:5903:5901']\ngroup_add: ['993']\n"+value), &before); err != nil {
			t.Fatal(err)
		}
		converted, err := config.ConvertServices(map[string]*config.ServiceConfigV1{"app": &before})
		if err != nil {
			t.Fatal(err)
		}
		request, err := hardwareRequests(converted["app"])
		if err != nil || len(request) != 1 {
			t.Fatalf("%+v %v", request, err)
		}
		var after config.ServiceConfig
		if err := utils.Convert(converted["app"], &after); err != nil {
			t.Fatal(err)
		}
		again, err := hardwareRequests(&after)
		a, _ := json.Marshal(request)
		b, _ := json.Marshal(again)
		if err != nil || string(a) != string(b) || after.ShmSize != 2147483648 || after.Ports[0] != "127.0.0.1:5903:5901" || after.Runtime != "nvidia" || after.GroupAdd[0] != "993" {
			t.Fatalf("roundtrip lost options: %s %s %v", a, b, err)
		}
	}
}

func TestHardwareComposeRejectsContradictions(t *testing.T) {
	for _, value := range []string{
		"gpus:\n - count: 1\n   device_ids: [GPU-example]\n",
		"deploy:\n resources:\n  reservations:\n   devices:\n    - count: all\n",
		"gpus: all\ndeploy:\n resources:\n  reservations:\n   devices:\n    - count: all\n      capabilities: [gpu]\n",
		"shm_size: 2g\nipc: host\n", "pids_limit: -2\n", "runtime: 'runc;other'\n",
	} {
		var c config.ServiceConfig
		err := yaml.Unmarshal([]byte(value), &c)
		if err == nil {
			_, err = hardwareRequests(&c)
		}
		if err == nil {
			t.Errorf("accepted %s", value)
		}
	}
}
