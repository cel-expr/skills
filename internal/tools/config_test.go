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
	"strings"
	"testing"

	celenv "github.com/google/cel-go/common/env"
)

func TestConfigFromJSON(t *testing.T) {
	tests := []struct {
		name       string
		configJSON string
		want       *Config
		wantErr    bool
	}{
		{
			name: "valid basic config",
			configJSON: `{
				"name": "test_env",
				"description": "A test environment",
				"variables": [
					{"name": "user", "type": "User"}
				]
			}`,
			want: &Config{
				Name:        "test_env",
				Description: "A test environment",
				Variables: []*Variable{
					{Name: "user", Type: "User"},
				},
			},
			wantErr: false,
		},
		{
			name:       "invalid json",
			configJSON: `{"name": "test_env"`,
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConfigFromJSON(tc.configJSON)
			if (err != nil) != tc.wantErr {
				t.Errorf("ConfigFromJSON() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				gotJSON := mustJSONMarshal(t, got)
				wantJSON := mustJSONMarshal(t, tc.want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("ConfigFromJSON() = %v, want %v", string(gotJSON), string(wantJSON))
				}
			}
		})
	}
}

func mustJSONMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return data
}

func TestEnvFromConfig(t *testing.T) {
	tests := []struct {
		name    string
		envJSON *Config
		wantErr bool
	}{
		{
			name: "valid env config",
			envJSON: &Config{
				Name: "test_env",
				Variables: []*Variable{
					{Name: "user_name", Type: "string"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := EnvFromConfig(tt.envJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("EnvFromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && env == nil {
				t.Errorf("EnvFromJSON() returned nil env without error")
			}
		})
	}
}

func TestConfigToCEL(t *testing.T) {
	config := &Config{
		Name:        "test_config",
		Description: "A test config",
		Container:   "test.v1",
		Imports:     []*Import{{Name: "test.v1.TestMessage"}},
		StdLib:      &LibrarySubset{Disabled: false, DisableMacros: true},
		Extensions:  []*Extension{{Name: "strings", Version: "1"}},
		Variables: []*Variable{
			{Name: "user", Description: "user info", Type: "map<string, dyn>"},
		},
		Functions: []*Function{
			{Name: "myFunc", Description: "my func", Overloads: []*Overload{
				{ID: "myFunc_string", Args: []string{"string"}, Return: "bool"},
				{ID: "myFunc_target", Target: "string", Args: []string{"int"}, Return: "bool"},
			}},
		},
		Validators: []*Validator{
			{Name: "cel.homogenous_literals"},
		},
		Features: []*Feature{
			{Name: "enable_macro_call_tracking", Enabled: true},
		},
	}

	celConfig, err := config.ToCELConfig()
	if err != nil {
		t.Fatalf("ToCELConfig() failed: %v", err)
	}
	if celConfig == nil {
		t.Fatalf("ToCELConfig() returned nil")
	}

	if celConfig.Name != "test_config" || celConfig.Description != "A test config" || celConfig.Container != "test.v1" {
		t.Errorf("ToCELConfig() basic fields not mapped correctly")
	}

	if len(celConfig.Imports) != 1 || celConfig.Imports[0].Name != "test.v1.TestMessage" {
		t.Errorf("ToCELConfig() imports not mapped correctly")
	}

	if celConfig.StdLib == nil || celConfig.StdLib.DisableMacros != true {
		t.Errorf("ToCELConfig() StdLib not mapped correctly")
	}

	if len(celConfig.Extensions) != 1 || celConfig.Extensions[0].Name != "strings" {
		t.Errorf("ToCELConfig() Extensions not mapped correctly")
	}

	if len(celConfig.Variables) != 1 || celConfig.Variables[0].Name != "user" {
		t.Errorf("ToCELConfig() Variables not mapped correctly")
	}

	if len(celConfig.Functions) != 1 || celConfig.Functions[0].Name != "myFunc" || len(celConfig.Functions[0].Overloads) != 2 {
		t.Errorf("ToCELConfig() Functions not mapped correctly")
	}

	if len(celConfig.Validators) != 1 || celConfig.Validators[0].Name != "cel.homogenous_literals" {
		t.Errorf("ToCELConfig() Validators not mapped correctly")
	}

	if len(celConfig.Features) != 1 || celConfig.Features[0].Name != "enable_macro_call_tracking" {
		t.Errorf("ToCELConfig() Features not mapped correctly")
	}
}

func TestConfigNilReceivers(t *testing.T) {
	var c *Config
	if cfg, err := c.ToCELConfig(); err != nil || cfg != nil {
		t.Errorf("Expected nil")
	}
	var i *Import
	if i.ToCELImport() != nil {
		t.Errorf("Expected nil")
	}
	var v *Variable
	if vv, err := v.ToCELVariable(); err != nil || vv != nil {
		t.Errorf("Expected nil")
	}
	var cv *ContextVariable
	if got, err := cv.ToCELContextVariable(); got != nil || err != nil {
		t.Errorf("Expected nil")
	}
	var f *Function
	if fn, err := f.ToCELFunction(); err != nil || fn != nil {
		t.Errorf("Expected nil")
	}
	var o *Overload
	if ov, err := o.ToCELOverload(); err != nil || ov != nil {
		t.Errorf("Expected nil")
	}
	var e *Extension
	if e.ToCELExtension() != nil {
		t.Errorf("Expected nil")
	}
	var ls *LibrarySubset
	if ls.ToCELLibrarySubset() != nil {
		t.Errorf("Expected nil")
	}
	var fs *FunctionSubset
	if fs.ToCELFunction() != nil {
		t.Errorf("Expected nil")
	}
	var val *Validator
	if val.ToCELValidator() != nil {
		t.Errorf("Expected nil")
	}
	var feat *Feature
	if feat.ToCELFeature() != nil {
		t.Errorf("Expected nil")
	}
}

func TestToCELConfigErrors(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr string
	}{
		{
			name: "invalid context variable",
			config: &Config{
				ContextVariable: &ContextVariable{Type: "list<~"},
			},
			wantErr: "unexpected end of input",
		},
		{
			name: "invalid variable",
			config: &Config{
				Variables: []*Variable{{Name: "v", Type: "list<~"}},
			},
			wantErr: "unexpected end of input",
		},
		{
			name: "invalid function",
			config: &Config{
				Functions: []*Function{{Name: "f", Overloads: []*Overload{{ID: "id", Return: "list<~"}}}},
			},
			wantErr: "unexpected end of input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.config.ToCELConfig()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ToCELConfig() error = %v, wantErr %q", err, tt.wantErr)
			}
		})
	}
}

func TestToCELContextVariableWithParams(t *testing.T) {
	cv := &ContextVariable{Type: "list<int>"}
	_, err := cv.ToCELContextVariable()
	if err == nil || !strings.Contains(err.Error(), "context variable cannot have type parameters") {
		t.Errorf("ToCELContextVariable() error = %v, wantErr %q", err, "context variable cannot have type parameters")
	}
}

func TestToCELVariableErrors(t *testing.T) {
	v := &Variable{Name: "v", Type: "list<~"}
	_, err := v.ToCELVariable()
	if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
		t.Errorf("ToCELVariable() error = %v, wantErr %q", err, "unexpected end of input")
	}
}

func TestToCELContextVariableErrors(t *testing.T) {
	cv := &ContextVariable{Type: "list<~"}
	_, err := cv.ToCELContextVariable()
	if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
		t.Errorf("ToCELContextVariable() error = %v, wantErr %q", err, "unexpected end of input")
	}
}

func TestToCELFunctionErrors(t *testing.T) {
	f := &Function{
		Name: "f",
		Overloads: []*Overload{
			{ID: "id", Return: "list<~"},
		},
	}
	_, err := f.ToCELFunction()
	if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
		t.Errorf("ToCELFunction() error = %v, wantErr %q", err, "unexpected end of input")
	}
}

func TestToCELOverloadErrors(t *testing.T) {
	tests := []struct {
		name     string
		overload *Overload
		wantErr  string
	}{
		{
			name:     "invalid args",
			overload: &Overload{ID: "id", Args: []string{"list<~"}},
			wantErr:  "unexpected end of input",
		},
		{
			name:     "invalid return",
			overload: &Overload{ID: "id", Return: "list<~"},
			wantErr:  "unexpected end of input",
		},
		{
			name:     "invalid target",
			overload: &Overload{ID: "id", Target: "list<~"},
			wantErr:  "unexpected end of input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.overload.ToCELOverload()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ToCELOverload() error = %v, wantErr %q", err, tt.wantErr)
			}
		})
	}
}

func TestEnvFromConfigError(t *testing.T) {
	cfg := &Config{
		Variables: []*Variable{{Name: "v", Type: "list<~"}},
	}
	_, err := EnvFromConfig(cfg)
	if err == nil || !strings.Contains(err.Error(), "unexpected end of input") {
		t.Errorf("EnvFromConfig() error = %v, wantErr %q", err, "unexpected end of input")
	}
}

func TestFunctionWithoutDescription(t *testing.T) {
	f := &Function{Name: "testFunc", Overloads: []*Overload{{ID: "testFunc"}}}
	celFunc, err := f.ToCELFunction()
	if err != nil {
		t.Fatalf("ToCELFunction() failed: %v", err)
	}
	if celFunc.Description != "" {
		t.Errorf("Expected empty description, got %q", celFunc.Description)
	}
}

func TestParseTypeDesc(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *celenv.TypeDesc
		wantErr string
	}{
		{
			name:  "simple type",
			input: "int",
			want:  celenv.NewTypeDesc("int"),
		},
		{
			name:  "namespaced identifier",
			input: "google.protobuf.Struct",
			want:  celenv.NewTypeDesc("google.protobuf.Struct"),
		},
		{
			name:  "leading dot",
			input: ".foo.bar",
			want:  celenv.NewTypeDesc(".foo.bar"),
		},
		{
			name:  "nested type",
			input: "list<int>",
			want:  celenv.NewTypeDesc("list", celenv.NewTypeDesc("int")),
		},
		{
			name:  "nested namespaced",
			input: "list<google.rpc.Status>",
			want:  celenv.NewTypeDesc("list", celenv.NewTypeDesc("google.rpc.Status")),
		},
		{
			name:  "whitespace",
			input: "  list < int >  ",
			want:  celenv.NewTypeDesc("list", celenv.NewTypeDesc("int")),
		},
		{
			name:  "bare type param",
			input: "~T",
			want:  celenv.NewTypeParam("T"),
		},
		{
			name:  "complex nested",
			input: "map<string, list<~V>>",
			want:  celenv.NewTypeDesc("map", celenv.NewTypeDesc("string"), celenv.NewTypeDesc("list", celenv.NewTypeParam("V"))),
		},
		{
			name:  "multiple type params",
			input: "map<string, map<int, bool>>",
			want:  celenv.NewTypeDesc("map", celenv.NewTypeDesc("string"), celenv.NewTypeDesc("map", celenv.NewTypeDesc("int"), celenv.NewTypeDesc("bool"))),
		},
		{
			name:  "underscore and numbers",
			input: "my_type_1",
			want:  celenv.NewTypeDesc("my_type_1"),
		},
		{
			name:    "invalid syntax",
			input:   "list<int",
			wantErr: "expected ',' or '>'",
		},
		{
			name:    "missing comma",
			input:   "map<string int>",
			wantErr: "expected ',' or '>'",
		},
		{
			name:    "invalid identifier start",
			input:   "1type",
			wantErr: "identifier is expected, but '1' was found",
		},
		{
			name:    "invalid identifier character",
			input:   "int-type",
			wantErr: "unexpected character '-'",
		},
		{
			name:    "invalid type parameter multiple chars",
			input:   "list<~ABC>",
			wantErr: "invalid type param, must have a single alphabetic character",
		},
		{
			name:    "empty generic",
			input:   "list<>",
			wantErr: "identifier is expected, but '>' was found",
		},
		{
			name:    "incomplete generic",
			input:   "map<int,>",
			wantErr: "identifier is expected, but '>' was found",
		},
		{
			name:    "trailing characters",
			input:   "int bool",
			wantErr: "unexpected character 'b'",
		},
		{
			name:    "double dots",
			input:   "google..protobuf.Struct",
			wantErr: "identifier is expected, but '.' was found",
		},
		{
			name:    "missing identifier before generic",
			input:   "<int>",
			wantErr: "missing identifier at position 0",
		},
		{
			name:    "incomplete type param",
			input:   "list<~",
			wantErr: "unexpected end of input",
		},
		{
			name:    "incomplete identifier",
			input:   "google.",
			wantErr: "unexpected end of input",
		},
		{
			name:    "invalid type parameter identifier",
			input:   "list<~1>",
			wantErr: "invalid type parameter identifier '1'",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseType(tc.input)
			if tc.wantErr != "" {
				if err == nil {
					t.Errorf("parseTypeDesc() error = nil, wantErr %q", tc.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("parseTypeDesc() error = %v, wantErr %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTypeDesc() error = %v, wantErr nil", err)
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseTypeDesc(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestLibrarySubsetFunctions(t *testing.T) {
	ls := &LibrarySubset{
		IncludeFunctions: []*FunctionSubset{{Name: "inc", OverloadIDs: []string{"inc"}}},
		ExcludeFunctions: []*FunctionSubset{{Name: "exc", OverloadIDs: []string{"exc"}}},
	}
	celLs := ls.ToCELLibrarySubset()
	if len(celLs.IncludeFunctions) != 1 || celLs.IncludeFunctions[0].Name != "inc" {
		t.Errorf("expected 1 include function 'inc', got %v", celLs.IncludeFunctions)
	}
	if len(celLs.ExcludeFunctions) != 1 || celLs.ExcludeFunctions[0].Name != "exc" {
		t.Errorf("expected 1 exclude function 'exc', got %v", celLs.ExcludeFunctions)
	}
}
