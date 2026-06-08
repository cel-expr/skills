// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/cel-go/cel"

	testpb "github.com/cel-expr/skills/internal/proto"
)

func TestComputeInputSchema(t *testing.T) {
	envJSON := &Config{
		Name: "basic",
		Variables: []*Variable{
			{Name: "user", Type: "map<string, dyn>"},
			{Name: "age", Type: "int"},
			{Name: "labels", Type: "list<string>"},
			{Name: "budget", Type: "double"},
			{Name: "timeout", Type: "google.protobuf.Duration"},
			{Name: "createdAt", Type: "google.protobuf.Timestamp"},
			{Name: "isActive", Type: "bool"},
			{Name: "nothing", Type: "null_type"},
			{Name: "count", Type: "uint"},
			{Name: "optName", Type: "optional_type<string>"},
			{Name: "defaultName", Type: "type"},
		},
	}

	env, err := EnvFromConfig(envJSON)
	if err != nil {
		t.Fatalf("Failed constructing env: %v", err)
	}
	env, err = env.Extend(
		cel.Types(&testpb.TestMessage{}),
		cel.Variable("msg", cel.ObjectType("cel.skills.internal.proto.TestMessage")),
	)
	if err != nil {
		t.Fatalf("Failed extending env: %v", err)
	}

	tests := []struct {
		name    string
		expr    string
		want    *SchemaNode
		wantErr bool
	}{
		{
			name: "basic access",
			expr: `user.name == "Alice" && age > 18`,
			want: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"user": {
						Type: "object",
						Properties: map[string]*SchemaNode{
							"name": {
								Type: "object",
							},
						},
						AdditionalProperties: &SchemaNode{Type: "object"},
					},
					"age": {
						Type:   "integer",
						Format: "int64",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "list access",
			expr: `labels.exists(l, l.matches("^foo-"))`,
			want: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"labels": {
						Type:  "array",
						Items: &SchemaNode{Type: "string"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "basic types access",
			expr: `budget > 100.0 && timeout > duration("1s") && createdAt > timestamp("2024-01-01T00:00:00Z") && isActive && nothing == null && count > 0u`,
			want: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"budget":    {Type: "number", Format: "double"},
					"timeout":   {Type: "string"},
					"createdAt": {Type: "string", Format: "date-time"},
					"isActive":  {Type: "boolean"},
					"nothing":   {Type: "null"},
					"count":     {Type: "integer", Format: "int64", Minimum: 0},
				},
			},
			wantErr: false,
		},
		{
			name: "opaque access",
			expr: `type(optName) == type(defaultName)`, // Evaluates without method access so opaque/type falls back
			want: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"optName":     {Type: "string"},
					"defaultName": {Type: "object", TypeID: "type"}, // Tests default empty case
				},
			},
			wantErr: false,
		},
		{
			name: "protobuf field graph leaf vs intermediate",
			expr: `msg.single_nested_message.bb == 42 && msg.single_int32 == 1`,
			want: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"msg": {
						Type: "object",
						Properties: map[string]*SchemaNode{
							"single_nested_message": {
								Type: "object",
								Properties: map[string]*SchemaNode{
									"bb": {Type: "integer", Format: "int64"},
								},
							},
							"single_int32": {Type: "integer", Format: "int64"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "protobuf nested msg",
			expr: `msg.single_nested_message`,
			want: &SchemaNode{
				Type: "object",
				Properties: map[string]*SchemaNode{
					"msg": {
						Type: "object",
						Properties: map[string]*SchemaNode{
							"single_nested_message": {
								Type: "object",
								Properties: map[string]*SchemaNode{
									"bb": {Type: "integer", Format: "int64"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, iss := env.Compile(tt.expr)
			if iss.Err() != nil {
				t.Fatalf("Failed compiling expression: %v", iss.Err())
			}
			got, err := ComputeInputSchema(env, ast)
			if (err != nil) != tt.wantErr {
				t.Errorf("ComputeInputSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)

			var gotMap, wantMap map[string]any
			json.Unmarshal(gotJSON, &gotMap)
			json.Unmarshal(wantJSON, &wantMap)

			// Helper to recursively strip ExprID
			var removeExprID func(any)
			removeExprID = func(v any) {
				if m, ok := v.(map[string]any); ok {
					delete(m, "ExprRefs")
					for _, val := range m {
						removeExprID(val)
					}
				} else if l, ok := v.([]any); ok {
					for _, val := range l {
						removeExprID(val)
					}
				}
			}
			removeExprID(gotMap)
			removeExprID(wantMap)

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Errorf("ComputeInputSchema()\nGot:  %v\nWant: %v", gotMap, wantMap)
			}
		})
	}
}
