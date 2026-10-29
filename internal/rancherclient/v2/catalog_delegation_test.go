package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogProofIsEventLocalAndIncludedOnBootstrapReadTransformAndChild(t *testing.T) {
	observed := []string{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = append(observed, r.Method+" "+r.URL.Path+" "+r.Header.Get(CatalogLaunchDelegationHeader))
		if r.URL.Path == "/v2-beta/schemas" {
			w.Header().Set("X-API-Schemas", server.URL+"/v2-beta/schemas")
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		fmt.Fprint(w, `{}`)
	}))
	defer server.Close()
	original, err := NewRancherClient(&ClientOpts{Url: server.URL + "/v2-beta/schemas", AccessKey: "service-key", SecretKey: "private-service-secret"})
	if err != nil {
		t.Fatal("original client failed")
	}
	proof := "server-only-proof"
	delegated, err := WithCatalogLaunchDelegation(original, proof)
	if err != nil {
		t.Fatal("event client failed")
	}
	var result map[string]interface{}
	if delegated.GetLink(Resource{Links: map[string]string{"self": server.URL + "/v2-beta/stacks/1st7"}}, "self", &result) != nil {
		t.Fatal("read failed")
	}
	for _, path := range []string{"/v2-beta/scripts/transform", "/v2-beta/projects/1a5/services"} {
		if delegated.Post(server.URL+path, map[string]interface{}{}, &result) != nil {
			t.Fatal("child failed")
		}
	}
	if original.GetOpts().CatalogLaunchDelegation != "" || original.GetOpts() == delegated.GetOpts() {
		t.Fatal("shared router client changed")
	}
	if !strings.HasSuffix(observed[0], " ") {
		t.Fatal("ordinary bootstrap gained proof")
	}
	for _, request := range observed[1:] {
		if !strings.HasSuffix(request, " "+proof) {
			t.Fatal("proof lost on event request")
		}
	}
}

func TestCatalogProofCannotFollowCrossOriginOrEchoPrivateFailureBodies(t *testing.T) {
	foreignCalls := 0
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { foreignCalls++; fmt.Fprint(w, `{}`) }))
	defer foreign.Close()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/schemas" {
			w.Header().Set("X-API-Schemas", server.URL+"/schemas")
			fmt.Fprint(w, `{"data":[]}`)
			return
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, foreign.URL, http.StatusFound)
			return
		}
		w.WriteHeader(403)
		fmt.Fprint(w, `{"message":"private-human-answer-and-proof"}`)
	}))
	defer server.Close()
	c, err := NewRancherClient(&ClientOpts{Url: server.URL + "/schemas", CatalogLaunchDelegation: "server-only-proof"})
	if err != nil {
		t.Fatal("client failed")
	}
	var result map[string]interface{}
	for _, path := range []string{"/redirect", "/deny"} {
		err := c.GetLink(Resource{Links: map[string]string{"self": server.URL + path}}, "self", &result)
		if err == nil || strings.Contains(err.Error(), "private-human") || strings.Contains(err.Error(), "server-only-proof") {
			t.Fatal("unsafe delegated response")
		}
	}
	if foreignCalls != 0 {
		t.Fatal("proof crossed origin")
	}
}
