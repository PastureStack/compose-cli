package utils

import (
	"reflect"
	"testing"
)

type conversionSource struct {
	Name  string
	Tags  []string
	Attrs map[string]string
}

func TestConvertCompatibility(t *testing.T) {
	source := conversionSource{Name: "service", Tags: []string{"one"}, Attrs: map[string]string{"key": "value"}}

	for name, convert := range map[string]func(interface{}, interface{}) error{
		"json": ConvertByJSON,
		"yaml": Convert,
	} {
		t.Run(name, func(t *testing.T) {
			var target conversionSource
			if err := convert(source, &target); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(source, target) {
				t.Fatalf("conversion mismatch: %#v", target)
			}
		})
	}
}

func TestCopyHelpersAreIndependent(t *testing.T) {
	sourceSlice := []string{"one"}
	copySlice := CopySlice(sourceSlice)
	copySlice[0] = "two"
	if sourceSlice[0] != "one" {
		t.Fatal("CopySlice changed its source")
	}

	sourceMap := map[string]string{"key": "one"}
	copyMap := CopyMap(sourceMap)
	copyMap["key"] = "two"
	if sourceMap["key"] != "one" {
		t.Fatal("CopyMap changed its source")
	}

	if CopySlice(nil) != nil || CopyMap(nil) != nil {
		t.Fatal("nil inputs must remain nil")
	}
}
