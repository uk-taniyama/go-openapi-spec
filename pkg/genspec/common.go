// Copyright (c) 2021 uk-taniyama.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package genspec

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/flynn/json5"
	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v2"
)

type KeyValue map[string]interface{}

func ParseTag(tag string) KeyValue {
	kv := KeyValue{}
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, "{") {
		tag = reflect.StructTag(tag).Get("scheme")
		tag = strings.TrimSpace(tag)
		if !strings.HasPrefix(tag, "{") {
			tag = "{" + tag
		}
	}
	if !strings.HasSuffix(tag, "}") {
		tag = tag + "}"
	}
	_ = json5.Unmarshal([]byte(tag), &kv)
	return kv
}

func Convert(src, dst interface{}) error {
	b, err := json.Marshal(src)
	if err != nil {
		return nil
	}
	return json.Unmarshal(b, dst)
}

func ExpandTagForScheme(scheme *openapi3.Schema, tag KeyValue) {
	ty := scheme.Type
	work := KeyValue{}
	Convert(scheme, &work)

	for k, v := range tag {
		switch k {
		case "min":
			switch ty {
			case "integer":
				work["minimum"] = v
			case "string":
				work["minLength"] = v
			case "array":
				work["minItems"] = v
			}
		case "max":
			switch ty {
			case "integer":
				work["maximum"] = v
			case "string":
				work["maxLength"] = v
			case "array":
				work["maxItems"] = v
			}
		case "gt":
			switch ty {
			case "integer":
				work["minimum"] = v
				work["exclusiveMinimum"] = true
			}
		case "lt":
			switch ty {
			case "integer":
				work["maximum"] = v
				work["exclusiveMaximum"] = true
			}
		default:
			work[k] = v
		}
	}
	Convert(&work, scheme)
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustParseMapSlice(str string) *yaml.MapSlice {
	ms, err := ParseMapSlice(str)
	Must(err)
	return ms
}

func ParseMapSlice(str string) (*yaml.MapSlice, error) {
	ms := &yaml.MapSlice{}
	err := yaml.Unmarshal([]byte(str), ms)
	return ms, err
}

func StringifyMapSlice(obj *yaml.MapSlice) (string, error) {
	bytes, err := yaml.Marshal(obj)
	if err != nil {
		return "", err
	}
	return string(bytes), err
}

func MergeMapSlice(a *yaml.MapSlice, b *yaml.MapSlice) {
NEXT_B:
	for _, itemB := range *b {
		for i, itemA := range *a {
			if itemA.Key != itemB.Key {
				continue
			}

			itemValueA, ok := itemA.Value.(yaml.MapSlice)
			if ok {
				itemValueB, ok := itemB.Value.(yaml.MapSlice)
				if ok {
					MergeMapSlice(&itemValueA, &itemValueB)
					// Update by index !!!
					(*a)[i].Value = itemValueA
					continue NEXT_B
				}
			}

			// Update by index !!!
			(*a)[i].Value = itemB.Value
			continue NEXT_B
		}

		*a = append(*a, itemB)
	}
}
