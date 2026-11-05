package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

var errCatalogMutationRequest = errors.New("CatalogLaunchRequestInvalid")

func (rancherClient *RancherBaseClientImpl) marshalMutationRequest(input interface{}) ([]byte, error) {
	if rancherClient.Opts == nil || rancherClient.Opts.CatalogLaunchDelegation == "" {
		return json.Marshal(input)
	}
	wire, err := CatalogMutationRequest(input)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire)
}

var catalogResourceType = reflect.TypeOf(Resource{})

// CatalogMutationRequest projects only typed SDK Resource envelopes. Both the
// private preview and delegated HTTP mutations use this owner. Business maps
// are never traversed or key-stripped; ordinary SDK requests do not call it.
func CatalogMutationRequest(value interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, errCatalogMutationRequest
	}
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&result) != nil || result == nil {
		return nil, errCatalogMutationRequest
	}
	projectRequestStruct(reflect.ValueOf(value), result)
	return result, nil
}

func projectRequestStruct(value reflect.Value, encoded interface{}) {
	for value.IsValid() && (value.Kind() == reflect.Ptr || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return
	}
	switch value.Kind() {
	case reflect.Struct:
		object, ok := encoded.(map[string]interface{})
		if !ok {
			return
		}
		if value.Type() == catalogResourceType {
			// A named Resource property is data, not a promoted response envelope.
			return
		}
		if value.Type() == reflect.TypeOf(SetServiceLinksInput{}) {
			// Empty links are an explicit approved action, not an omitted field.
			if _, present := object["serviceLinks"]; !present {
				object["serviceLinks"] = []interface{}{}
			}
		}
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if field.PkgPath != "" || name == "-" {
				continue
			}
			if field.Anonymous && name == "" {
				embedded := field.Type
				if embedded.Kind() == reflect.Ptr {
					embedded = embedded.Elem()
				}
				if embedded == catalogResourceType {
					for resourceIndex := 0; resourceIndex < catalogResourceType.NumField(); resourceIndex++ {
						key := strings.Split(catalogResourceType.Field(resourceIndex).Tag.Get("json"), ",")[0]
						delete(object, key)
					}
				} else {
					projectRequestStruct(value.Field(index), object)
				}
			} else {
				if name == "" {
					name = field.Name
				}
				projectRequestStruct(value.Field(index), object[name])
			}
		}
	case reflect.Array, reflect.Slice:
		items, ok := encoded.([]interface{})
		if ok {
			for index := 0; index < value.Len() && index < len(items); index++ {
				projectRequestStruct(value.Index(index), items[index])
			}
		}
	}
}
