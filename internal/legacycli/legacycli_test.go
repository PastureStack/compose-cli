package legacycli

import (
	"reflect"
	"testing"
)

func TestAdapterPreservesGlobalAndCommandValues(t *testing.T) {
	t.Setenv("PASTURESTACK_TEST_URL", "https://example.invalid")

	var beforeURL string
	var gotArgs []string
	var gotFiles []string
	var gotEnabled bool
	app := NewApp()
	app.Name = "pasturestack-test"
	app.Flags = []Flag{
		StringFlag{Name: "url,u", EnvVar: "PASTURESTACK_TEST_URL"},
	}
	app.Before = func(ctx *Context) error {
		beforeURL = ctx.GlobalString("url")
		return nil
	}
	app.Commands = []Command{{
		Name:      "deploy",
		ShortName: "d",
		Flags: []Flag{
			BoolTFlag{Name: "enabled"},
			StringSliceFlag{Name: "file,f"},
		},
		Action: func(ctx *Context) error {
			gotArgs = ctx.Args()
			gotFiles = ctx.StringSlice("file")
			gotEnabled = ctx.Bool("enabled")
			return nil
		},
	}}

	if err := app.Run([]string{"pasturestack-test", "d", "--file", "one.yml", "--file", "two.yml", "service"}); err != nil {
		t.Fatal(err)
	}
	if beforeURL != "https://example.invalid" {
		t.Fatalf("unexpected global URL: %q", beforeURL)
	}
	if !gotEnabled {
		t.Fatal("BoolTFlag default was not preserved")
	}
	if !reflect.DeepEqual(gotFiles, []string{"one.yml", "two.yml"}) {
		t.Fatalf("unexpected files: %#v", gotFiles)
	}
	if !reflect.DeepEqual(gotArgs, []string{"service"}) {
		t.Fatalf("unexpected args: %#v", gotArgs)
	}
}
