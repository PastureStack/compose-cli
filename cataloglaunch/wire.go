package cataloglaunch

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"

	client "github.com/PastureStack/compose-cli/internal/rancherclient/v2"
)

// These are mutation inputs, not generic API resources. Empty links remain an
// explicit business value, and link aliases/targets retain the native parser's values.
type serviceLinkWire struct {
	Name      string `json:"name,omitempty"`
	ServiceID string `json:"serviceId"`
}

type serviceLinksWire struct {
	ServiceLinks []serviceLinkWire `json:"serviceLinks"`
}

func catalogLinksWire(input *client.SetServiceLinksInput) *serviceLinksWire {
	if input == nil {
		return nil
	}
	result := &serviceLinksWire{ServiceLinks: make([]serviceLinkWire, 0, len(input.ServiceLinks))}
	for _, link := range input.ServiceLinks {
		result.ServiceLinks = append(result.ServiceLinks, serviceLinkWire{Name: link.Name, ServiceID: link.ServiceId})
	}
	return result
}

var apiResourceType = reflect.TypeOf(client.Resource{})

// Only this private preview producer projects SDK request structs. Preserve the
// native JSON encoding, then omit the SDK's typed Resource envelope throughout
// those structs. Maps are business data and are never traversed or key-stripped.
// Ordinary SDK/legacy requests keep their original serialization.
func catalogRequestWire(value interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, ErrUnsupported
	}
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&result) != nil || result == nil {
		return nil, ErrUnsupported
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
		if value.Type() == apiResourceType {
			// A named Resource property is data, not a promoted response envelope.
			return
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
				if embedded == apiResourceType {
					for resourceIndex := 0; resourceIndex < apiResourceType.NumField(); resourceIndex++ {
						key := strings.Split(apiResourceType.Field(resourceIndex).Tag.Get("json"), ",")[0]
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
