package tools

import (
	"encoding/json"
	"reflect"
	"testing"
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
					{"name": "user", "typeName": "User"}
				]
			}`,
			want: &Config{
				Name:        "test_env",
				Description: "A test environment",
				Variables: []*Variable{
					{Name: "user", TypeName: "User"},
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConfigFromJSON(tt.configJSON)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConfigFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				gotJSON := mustJSONMarshal(t, got)
				wantJSON := mustJSONMarshal(t, tt.want)
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
					{Name: "user_name", TypeName: "string"},
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
		Name:            "test_config",
		Description:     "A test config",
		Container:       "test.v1",
		Imports:         []*Import{{Name: "test.v1.TestMessage"}},
		StdLib:          &LibrarySubset{Disabled: false, DisableMacros: true},
		Extensions:      []*Extension{{Name: "strings", Version: "1"}},
		ContextVariable: &ContextVariable{TypeName: "test.v1.TestMessage"},
		Variables: []*Variable{
			{Name: "user", Description: "user info", TypeName: "map", Params: []*TypeDesc{{TypeName: "string"}, {TypeName: "dyn"}}},
		},
		Functions: []*Function{
			{Name: "myFunc", Description: "my func", Overloads: []*Overload{
				{ID: "myFunc_string", Args: []*TypeDesc{{TypeName: "string"}}, Return: &TypeDesc{TypeName: "bool"}},
				{ID: "myFunc_target", Target: &TypeDesc{TypeName: "string"}, Args: []*TypeDesc{{TypeName: "int"}}, Return: &TypeDesc{TypeName: "bool"}},
			}},
		},
		Validators: []*Validator{
			{Name: "cel.homogenous_literals"},
		},
		Features: []*Feature{
			{Name: "enable_macro_call_tracking", Enabled: true},
		},
	}

	celConfig := config.ToCELConfig()
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

	if celConfig.ContextVariable == nil || celConfig.ContextVariable.TypeName != "test.v1.TestMessage" {
		t.Errorf("ToCELConfig() ContextVariable not mapped correctly")
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
	if c.ToCELConfig() != nil {
		t.Errorf("Expected nil")
	}
	var i *Import
	if i.ToCELImport() != nil {
		t.Errorf("Expected nil")
	}
	var v *Variable
	if v.ToCELVariable() != nil {
		t.Errorf("Expected nil")
	}
	var cv *ContextVariable
	if cv.ToCELContextVariable() != nil {
		t.Errorf("Expected nil")
	}
	var f *Function
	if f.ToCELFunction() != nil {
		t.Errorf("Expected nil")
	}
	var o *Overload
	if o.ToCELOverload() != nil {
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
	var val *Validator
	if val.ToCELValidator() != nil {
		t.Errorf("Expected nil")
	}
	var feat *Feature
	if feat.ToCELFeature() != nil {
		t.Errorf("Expected nil")
	}
	var td *TypeDesc
	if td.ToCELTypeDesc() != nil {
		t.Errorf("Expected nil")
	}
}

func TestFunctionWithoutDescription(t *testing.T) {
	f := &Function{Name: "testFunc", Overloads: []*Overload{{ID: "testFunc"}}}
	celFunc := f.ToCELFunction()
	if celFunc.Description != "" {
		t.Errorf("Expected empty description, got %q", celFunc.Description)
	}
}

func TestTypeParam(t *testing.T) {
	td := &TypeDesc{TypeName: "T", IsTypeParam: true}
	celTd := td.ToCELTypeDesc()
	if celTd.TypeName != "T" || !celTd.IsTypeParam {
		t.Errorf("Expected TypeParam T, got %v", celTd)
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
