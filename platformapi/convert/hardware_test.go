package convert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/PastureStack/compose-cli/config"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/project"
	yaml "gopkg.in/yaml.v3"
)

func TestHardwareTransformHTTPContract(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusUnprocessableEntity} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var inspected ContainerInspect
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v2-beta/scripts/transform" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&inspected); err != nil {
					t.Error(err)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if status != http.StatusOK {
					_, _ = w.Write([]byte(`{"type":"error","message":"invalid hardware"}`))
					return
				}
				_, _ = w.Write([]byte(`{"runtime":"nvidia","shmSize":2147483648,"deviceRequests":[{"driver":"nvidia","deviceIds":["GPU-one"],"capabilities":[["gpu"]],"options":{"mode":"test"}}],"groupAdd":["993"],"ports":["127.0.0.1:5903:5901/tcp"],"restartPolicy":{"name":"unless-stopped"}}`))
			}))
			defer server.Close()
			c := &client.RancherClient{RancherBaseClient: &client.RancherBaseClientImpl{
				Opts:    &client.ClientOpts{Url: server.URL, Timeout: time.Second},
				Schemas: &client.Schemas{Collection: client.Collection{Links: map[string]string{"self": server.URL + "/v2-beta/schemas"}}},
			}}
			var service config.ServiceConfig
			if err := yaml.Unmarshal([]byte("image: example\nruntime: nvidia\nshm_size: 2g\nrestart: unless-stopped\ngroup_add: ['993']\nports: ['127.0.0.1:5903:5901']\ngpus:\n - driver: nvidia\n   device_ids: [GPU-one]\n   options: {mode: test}\n"), &service); err != nil {
				t.Fatal(err)
			}
			result, err := CreateLaunchConfig("gpu", &service, c, project.Context{})
			if status != http.StatusOK {
				if err == nil {
					t.Fatal("transform rejection was silently discarded")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if inspected.HostConfig.Runtime != "nvidia" || inspected.HostConfig.ShmSize != 2147483648 || len(inspected.HostConfig.DeviceRequests) != 1 || inspected.HostConfig.DeviceRequests[0].DeviceIDs[0] != "GPU-one" {
				t.Fatalf("lost outgoing hardware: %+v", inspected.HostConfig)
			}
			if result.Runtime != "nvidia" || result.ShmSize != 2147483648 || len(result.DeviceRequests) != 1 || result.DeviceRequests[0].DeviceIDs[0] != "GPU-one" || result.DeviceRequests[0].Options["mode"] != "test" {
				t.Fatalf("lost incoming hardware: %+v", result)
			}
			if result.Ports[0] != "127.0.0.1:5903:5901/tcp" || inspected.HostConfig.RestartPolicy.Name != "unless-stopped" {
				t.Fatal("unrelated launch fields changed")
			}
		})
	}
}
