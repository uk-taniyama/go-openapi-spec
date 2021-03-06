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

package genspec_test

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
	"github.com/uk-taniyama/go-openapi-spec/pkg/genspec"
	"gopkg.in/yaml.v2"
)

type TestStruct struct {
	key1 string `scheme:"min:4,max:4,pattern:'\\\\s'"`
	key2 string `{min:4,max:4,pattern:'\\s'}`
}

func TestParseTag(t *testing.T) {
	type caseT struct {
		tag   string
		empty bool
	}
	cases := []caseT{
		{tag: `{min:4,max:4,pattern:"\\s"}`},
		{tag: `min:4,max:4,pattern:"\\s"`, empty: true},
		{tag: `scheme:"{min:4,max:4,pattern:\"\\\\s\"}"`},
		{tag: `scheme:"min:4,max:4,pattern:\"\\\\s\""`},
		{tag: `scheme:"min:4,max:4,pattern:'\\\\s'"`},
	}
	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			kv := genspec.ParseTag(c.tag)
			if c.empty {
				require.Empty(t, kv)
				return
			}
			require.Equal(t, kv["min"], float64(4))
			require.Equal(t, kv["max"], float64(4))
			require.Equal(t, kv["pattern"], "\\s")
		})
	}
}

func TestToScheme(t *testing.T) {
	type caseT struct {
		schema *openapi3.Schema
		kv     genspec.KeyValue
		json   string
	}
	cases := []caseT{
		{
			schema: openapi3.NewIntegerSchema(),
			kv:     genspec.KeyValue{"minimum": float64(4)},
			json:   `{"type":"integer","minimum":4}`,
		},
		{
			schema: openapi3.NewIntegerSchema(),
			kv:     genspec.KeyValue{"min": float64(4)},
			json:   `{"type":"integer","minimum":4}`,
		},
		{
			schema: openapi3.NewIntegerSchema(),
			kv:     genspec.KeyValue{"max": float64(4)},
			json:   `{"type":"integer","maximum":4}`,
		},
		{
			schema: openapi3.NewIntegerSchema(),
			kv:     genspec.KeyValue{"gt": float64(4)},
			json:   `{"type":"integer","minimum":4,"exclusiveMinimum":true}`,
		},
		{
			schema: openapi3.NewIntegerSchema(),
			kv:     genspec.KeyValue{"lt": float64(4)},
			json:   `{"type":"integer","maximum":4,"exclusiveMaximum":true}`,
		},
		{
			schema: openapi3.NewStringSchema(),
			kv:     genspec.KeyValue{"min": float64(4)},
			json:   `{"type":"string","minLength":4}`,
		},
		{
			schema: openapi3.NewStringSchema(),
			kv:     genspec.KeyValue{"max": float64(4)},
			json:   `{"type":"string","maxLength":4}`,
		},
		{
			schema: openapi3.NewStringSchema(),
			kv:     genspec.KeyValue{"lt": float64(4)},
			json:   `{"type":"string"}`,
		},
		{
			schema: openapi3.NewStringSchema(),
			kv:     genspec.KeyValue{"gt": float64(4)},
			json:   `{"type":"string"}`,
		},
		{
			schema: openapi3.NewArraySchema(),
			kv:     genspec.KeyValue{"min": float64(4)},
			json:   `{"type":"array","minItems":4}`,
		},
		{
			schema: openapi3.NewArraySchema(),
			kv:     genspec.KeyValue{"max": float64(4)},
			json:   `{"type":"array","maxItems":4}`,
		},
		{
			schema: openapi3.NewArraySchema(),
			kv:     genspec.KeyValue{"lt": float64(4)},
			json:   `{"type":"array"}`,
		},
		{
			schema: openapi3.NewArraySchema(),
			kv:     genspec.KeyValue{"gt": float64(4)},
			json:   `{"type":"array"}`,
		},
	}
	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			genspec.ExpandTagForScheme(c.schema, c.kv)
			b, err := json.Marshal(c.schema)
			require.NoError(t, err)
			require.JSONEq(t, string(b), c.json)
		})
	}
}

func TestParseOpeDoc(t *testing.T) {
	const doc = `Description
	
(GET /pets)
200: pet response
default: unexpected error
`
	opeDoc := genspec.ParseOpeDoc(doc)
	require.NotNil(t, opeDoc)
	require.Equal(t, opeDoc.Desc, "Description")
	require.Equal(t, opeDoc.Method, "GET")
	require.Equal(t, opeDoc.Path, "/pets")
	require.Equal(t, opeDoc.KV, genspec.KeyValue{
		"200":     "pet response",
		"default": "unexpected error",
	})
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func MustParseMapSlice(str string) *yaml.MapSlice {
	ms, err := genspec.ParseMapSlice(str)
	Must(err)
	return ms
}

//go:embed testdata/mergeyaml.yaml
var testdataMergeyaml string

func TestMergeMapSlice(t *testing.T) {
	testdata := MustParseMapSlice(testdataMergeyaml)
	data := yaml.MapSlice{}
	for _, item := range *testdata {
		// key:
		//   in:
		//   out:
		key := item.Key.(string)
		value := item.Value.(yaml.MapSlice)
		in := value[0].Value.(yaml.MapSlice)
		out := value[1].Value.(yaml.MapSlice)
		t.Run(key, func(t *testing.T) {
			genspec.MergeMapSlice(&data, &in)
			require.Equal(t, &out, &data)
		})
	}
}

//go:embed testdata/parsesecurity.yaml
var testdataParseSecurity []byte

func MustJSONStringify(obj interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func TestGenerateSecurityScheme(t *testing.T) {
	testdata := genspec.KeyValue{}
	err := yaml.Unmarshal(testdataParseSecurity, &testdata)
	require.NoError(t, err)
	for key, v := range testdata {
		value := v.(map[string]interface{})
		in := value["in"].(string)
		name := value["name"].(string)
		out := value["out"].(map[string]interface{})
		t.Run(key, func(t *testing.T) {
			n, s, err := genspec.GenerateSecurityScheme(in)
			require.NoError(t, err)
			require.Equal(t, name, n)
			require.Equal(
				t,
				MustJSONStringify(out),
				MustJSONStringify(s),
			)
		})
	}
}

func TestGenerateSecuritySchemes(t *testing.T) {
	text := `
basic: basic
oauth3:
  type: oauth2
  flows:
    authorizationCode:
      authorizationUrl: 'https://api.example.com/oauth2/authorize'
      refreshUrl: 'https://api.example.com/oauth2/refresh'
      tokenUrl: 'https://api.example.com/oauth2/token'
      scopes:
        read_pets: read your pets
        write_pets: modify pets in your account
oauth2:
  flow: authorizationCode
  authUrl: https://api.example.com/oauth2/authorize
  tokenUrl: https://api.example.com/oauth2/token
  refreshUrl: https://api.example.com/oauth2/refresh
  scopes:
    read_pets: read your pets
    write_pets: modify pets in your account
`
	s, err := genspec.GenerateSecuritySchemes(text)
	require.NoError(t, err)
	require.JSONEq(t, MustJSONStringify(s), `
	{
		"basic": {
			"type": "http",
			"scheme": "basic"
		},
		"oauth2": {
		  "type": "oauth2",
		  "flows": {
			"authorizationCode": {
			  "authorizationUrl": "https://api.example.com/oauth2/authorize",
			  "tokenUrl": "https://api.example.com/oauth2/token",
			  "refreshUrl": "https://api.example.com/oauth2/refresh",
			  "scopes": {
				"read_pets": "read your pets",
				"write_pets": "modify pets in your account"
			  }
			}
		  }
		},
		"oauth3": {
		  "type": "oauth2",
		  "flows": {
			"authorizationCode": {
			  "authorizationUrl": "https://api.example.com/oauth2/authorize",
			  "tokenUrl": "https://api.example.com/oauth2/token",
			  "refreshUrl": "https://api.example.com/oauth2/refresh",
			  "scopes": {
				"read_pets": "read your pets",
				"write_pets": "modify pets in your account"
			  }
			}
		  }
		}
	}
	`)
}

func TestCSVMergeLines(t *testing.T) {
	type caseT struct {
		in  []string
		out []string
	}
	cases := []caseT{
		{
			in:  []string{"A,B", ",C"},
			out: []string{"A,B,C"},
		},
		{
			in:  []string{"A,B", ",C", ",D"},
			out: []string{"A,B,C,D"},
		},
		{
			in:  []string{"A,B", ",C", ",D", "E"},
			out: []string{"A,B,C,D", "E"},
		},
		{
			in:  []string{"A,B", ",C", ",D", "E", ",F"},
			out: []string{"A,B,C,D", "E,F"},
		},
		{
			in:  []string{"", "A,B", ",C"},
			out: []string{"", "A,B,C"},
		},
	}
	for _, c := range cases {
		src := c.in
		dst := genspec.CSVMergeLines(src)
		require.Equal(t, c.out, dst, c.in)
	}
}

func TestKeyValueUnmarshial(t *testing.T) {
	kv := genspec.KeyValue{}
	err := yaml.Unmarshal([]byte("xxx"), &kv)
	require.EqualError(t, err, "KeyValue.UnmarshalYAML:cannot unmarshal")
	text := `
a1:
  b1: c
a2:
  b2: c
`
	err = yaml.Unmarshal([]byte(text), &kv)
	require.NoError(t, err)
	d, err := json.Marshal(&kv)
	require.NoError(t, err)
	require.JSONEq(t, string(d), `{"a1":{"b1":"c"},"a2":{"b2":"c"}}`)
}
