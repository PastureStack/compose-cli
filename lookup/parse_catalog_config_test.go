package lookup

import (
	"reflect"
	"testing"
)

func testParseCatalog(t *testing.T, contents string, expectedCatalogConfig *CatalogConfig) {
	catalogConfig, err := ParseCatalogConfig([]byte(contents))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedCatalogConfig, catalogConfig) {
		t.Fail()
	}
}

func TestParseCatalog(t *testing.T) {
	testParseCatalog(t, `
.catalog:
  name: test`, &CatalogConfig{
		Name: "test",
	})

	testParseCatalog(t, `
version: '2'
catalog:
  name: test`, &CatalogConfig{
		Name: "test",
	})

	testParseCatalog(t, `
version: '2'
.catalog:
  name: test`, &CatalogConfig{
		Name: "test",
	})

	testParseCatalog(t, `
version: '2'
services:
  .catalog:
    name: test`, &CatalogConfig{
		Name: "test",
	})
}
