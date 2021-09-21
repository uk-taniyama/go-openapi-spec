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
	"encoding/json"
	"strconv"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
	"github.com/uk-taniyama/go-openapi-spec/pkg/genspec"
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
