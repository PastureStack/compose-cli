package platformapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/PastureStack/compose-cli/config"
	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
	"github.com/PastureStack/compose-cli/project"
)

func TestCatalogLinkSkeletonUsesNormalParserAndRejectsExternalScope(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schemas" {
			w.Header().Set("X-API-Schemas", server.URL+"/schemas")
			fmt.Fprintf(w, `{"data":[{"id":"service","collectionMethods":["GET"],"links":{"collection":%q}}]}`, server.URL+"/services")
			return
		}
		if r.URL.Path != "/services" || r.URL.Query().Get("stackId") != "1st7" || r.URL.Query().Get("name") != "db" {
			t.Error("unexpected link lookup")
		}
		fmt.Fprint(w, `{"data":[{"id":"1s8","name":"db"}]}`)
	}))
	defer server.Close()
	c, err := client.NewRancherClient(&client.ClientOpts{Url: server.URL + "/schemas"})
	if err != nil {
		t.Fatal("owner client failed")
	}
	cfg := &config.ServiceConfig{Links: []string{" db : chosen "}}
	ctx := &Context{Client: c, Stack: &client.Stack{Resource: client.Resource{Id: "1st7"}}, CatalogLaunchDelegation: "proof"}
	ctx.Project = project.NewProject(&ctx.Context)
	actual, err := NewService("app", cfg, ctx).getServiceLinks()
	if err != nil {
		t.Fatal("normal links failed")
	}
	skeleton, err := CatalogServiceLinkSkeleton(cfg.Links, map[string]string{"db": "", "helper": "app"})
	if err != nil {
		t.Fatal("typed links failed")
	}
	for i := range skeleton.ServiceLinks {
		skeleton.ServiceLinks[i].ServiceId = "1s8"
	}
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(skeleton.ServiceLinks)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("normal and typed link bodies differ")
	}
	for _, links := range [][]string{{"other/db"}, {"missing"}} {
		if _, err := CatalogServiceLinkSkeleton(links, map[string]string{"db": ""}); err == nil {
			t.Fatal("unbound link scope admitted")
		}
	}
}
