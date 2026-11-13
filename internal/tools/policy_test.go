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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cel.dev/cel-go/cel"
	_ "github.com/cel-expr/skills/internal/proto"
)

func TestPolicy_CompileAndEvaluate(t *testing.T) {
	inlinePolicy := `name: access_policy
display_name: Access Control Policy
description: Evaluates role and enum-based access rules.
rule:
  variables:
    - name: is_first_tier
      expression: code == cel.skills.internal.proto.TestEnum.TEST_ENUM_FIRST
  match:
    - condition: role == "admin" || variables.is_first_tier
      output: '"allow"'
    - condition: role == "guest"
      output: '"deny"'
`
	cfgWithoutProto := &Config{
		Extensions: []*Extension{{Name: "bindings"}},
		Variables: []*Variable{
			{Name: "role", Type: "string"},
			{Name: "code", Type: "int"},
		},
	}

	// 1. Without protoTypes/FDS, compilation fails with undeclared reference.
	_, err := CompileCELWithOptions(inlinePolicy, cfgWithoutProto, CompileOptions{Policy: true})
	if err == nil || !strings.Contains(err.Error(), "undeclared reference") {
		t.Fatalf("expected undeclared reference error without protoTypes, got: %v", err)
	}

	// 2. Adding TestEnum to protoTypes extends the FDS and compiles/evaluates cleanly.
	cfg := &Config{
		ProtoTypes: []string{"cel.skills.internal.proto.TestEnum"},
		Extensions: []*Extension{{Name: "bindings"}},
		Variables: []*Variable{
			{Name: "role", Type: "string"},
			{Name: "code", Type: "int"},
		},
	}
	fds, warnings, err := ExtendFileDescriptorSet(nil, cfg, nil)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("ExtendFileDescriptorSet failed: warnings=%v, err=%v", warnings, err)
	}
	envOpts := []cel.EnvOption{cel.TypeDescs(fds)}

	compOut, err := CompileCELWithOptions(inlinePolicy, cfg, CompileOptions{
		EnvOpts: envOpts,
		Policy:  true,
	})
	if err != nil {
		t.Fatalf("CompileCELWithOptions(policy) failed: %v", err)
	}
	if len(compOut.Warnings) != 1 || !strings.Contains(compOut.Warnings[0], `"display_name"`) {
		t.Errorf("expected display_name policy warning, got: %v", compOut.Warnings)
	}
	if schema, ok := compOut.InputSchema.(*SchemaNode); ok {
		for prop := range schema.Properties {
			if strings.HasPrefix(prop, "@") || strings.Contains(prop, "TEST_ENUM") || strings.HasPrefix(prop, "variables") {
				t.Errorf("unexpected synthetic or enum property %q in inputSchema: %+v", prop, schema.Properties)
			}
		}
		if schema.Properties["role"] == nil || schema.Properties["code"] == nil {
			t.Errorf("expected role and code in inputSchema, got: %+v", schema.Properties)
		}
	}

	// Evaluate policy covering matched branches and unmatched optional.none() (expected: nil).
	tests := []TestCase{
		{
			TestCase: "admin allowed",
			Bindings: map[string]any{"role": "admin", "code": 0},
			Expected: "allow",
		},
		{
			TestCase: "first tier enum allowed",
			Bindings: map[string]any{"role": "member", "code": 1},
			Expected: "allow",
		},
		{
			TestCase: "guest denied",
			Bindings: map[string]any{"role": "guest", "code": 0},
			Expected: "deny",
		},
		{
			TestCase: "unmatched returns optional.none (null)",
			Bindings: map[string]any{"role": "member", "code": 2},
			Expected: nil,
		},
	}
	evalOut, err := EvaluateCELWithOptions(inlinePolicy, cfg, tests, CompileOptions{
		EnvOpts: envOpts,
		Policy:  true,
	})
	if err != nil {
		t.Fatalf("EvaluateCELWithOptions(policy) failed: %v", err)
	}
	for _, tr := range evalOut.TestResults {
		if tr.Status != "pass" {
			t.Errorf("test case %q failed: %+v", tr.TestCase, tr)
		}
	}
}

func TestPolicy_OptionalAlreadyDeclaredInEnvJSON(t *testing.T) {
	partialPolicy := `name: optional_coexistence_policy
rule:
  match:
    - condition: headers[?"x-role"].orValue("") == "admin"
      output: '"allow"'
`
	optFlatMapPolicy := `name: optional_v1_optflatmap_policy
rule:
  match:
    - condition: headers[?"x-role"].optFlatMap(r, optional.of(r)).orValue("") == "admin"
      output: "true"
    - output: "false"
`

	cases := []struct {
		name               string
		envJSON            string
		wantOptFlatMapPass bool
	}{
		{
			name:               "unversioned optional in env.json",
			envJSON:            `{"extensions":[{"name":"optional"}],"variables":[{"name":"headers","type":"map<string, string>"},{"name":"items","type":"list<string>"}]}`,
			wantOptFlatMapPass: false,
		},
		{
			name:               "optional version 0 in env.json",
			envJSON:            `{"extensions":[{"name":"optional","version":"0"}],"variables":[{"name":"headers","type":"map<string, string>"},{"name":"items","type":"list<string>"}]}`,
			wantOptFlatMapPass: false,
		},
		{
			name:               "optional version 1 in env.json",
			envJSON:            `{"extensions":[{"name":"optional","version":"1"}],"variables":[{"name":"headers","type":"map<string, string>"},{"name":"items","type":"list<string>"}]}`,
			wantOptFlatMapPass: true,
		},
		{
			name:               "optional version latest in env.json",
			envJSON:            `{"extensions":[{"name":"optional","version":"latest"}],"variables":[{"name":"headers","type":"map<string, string>"},{"name":"items","type":"list<string>"}]}`,
			wantOptFlatMapPass: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ConfigFromJSON(tc.envJSON)
			if err != nil {
				t.Fatalf("ConfigFromJSON failed: %v", err)
			}

			// Re-adding cel.OptionalTypes() during policy compilation must succeed without conflict.
			evalOut, err := EvaluateCELWithOptions(partialPolicy, cfg, []TestCase{
				{
					TestCase: "matched admin",
					Bindings: map[string]any{"headers": map[string]any{"x-role": "admin"}, "items": []any{}},
					Expected: "allow",
				},
				{
					TestCase: "unmatched returns optional.none",
					Bindings: map[string]any{"headers": map[string]any{}, "items": []any{}},
					Expected: nil,
				},
			}, CompileOptions{Policy: true})
			if err != nil {
				t.Fatalf("EvaluateCELWithOptions failed when optional already in env.json: %v", err)
			}
			for _, tr := range evalOut.TestResults {
				if tr.Status != "pass" {
					t.Errorf("test case %q failed: %+v", tr.TestCase, tr)
				}
			}

			// Verify the version declared in env.json is preserved (v0 rejects optFlatMap, v1+ allows it).
			_, v1Err := CompileCELWithOptions(optFlatMapPolicy, cfg, CompileOptions{Policy: true})
			if tc.wantOptFlatMapPass && v1Err != nil {
				t.Errorf("expected optFlatMap policy to compile for %s, got: %v", tc.name, v1Err)
			}
			if !tc.wantOptFlatMapPass && v1Err == nil {
				t.Errorf("expected optFlatMap policy to be rejected when optional is pinned to v0 in %s", tc.name)
			}
		})
	}
}

func TestPolicy_FileAndImportsAndErrors(t *testing.T) {
	tmpDir := t.TempDir()
	policyContent := `name: import_policy
imports:
  - name: cel.skills.internal.proto.TestMessage
rule:
  match:
    - condition: 'TestMessage{single_int32: x}.single_int32 == 42'
      output: "true"
    - output: "false"
`
	policyPath := filepath.Join(tmpDir, "sample.celpolicy")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed writing policy file: %v", err)
	}

	// Without protoTypes/FDS, compilation fails with undeclared reference.
	cfgMissing := &Config{
		Variables: []*Variable{{Name: "x", Type: "int"}},
	}
	_, err := CompileCELWithOptions(policyPath, cfgMissing, CompileOptions{Policy: true})
	if err == nil || !strings.Contains(err.Error(), "undeclared reference") {
		t.Fatalf("expected undeclared reference error without protoTypes, got: %v", err)
	}

	cfg := &Config{
		ProtoTypes: []string{"cel.skills.internal.proto.TestMessage"},
		Variables:  []*Variable{{Name: "x", Type: "int"}},
	}
	fds, warnings, err := ExtendFileDescriptorSet(nil, cfg, nil)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("ExtendFileDescriptorSet failed: warnings=%v, err=%v", warnings, err)
	}
	evalOut, err := EvaluateCELWithOptions(policyPath, cfg, []TestCase{
		{TestCase: "match", Bindings: map[string]any{"x": 42}, Expected: true},
		{TestCase: "fallback", Bindings: map[string]any{"x": 1}, Expected: false},
	}, CompileOptions{
		EnvOpts: []cel.EnvOption{cel.TypeDescs(fds)},
		Policy:  true,
	})
	if err != nil {
		t.Fatalf("EvaluateCELWithOptions(file policy) failed: %v", err)
	}
	for _, tr := range evalOut.TestResults {
		if tr.Status != "pass" {
			t.Errorf("test case %q failed: %+v", tr.TestCase, tr)
		}
	}

	// Non-existent policy file path returns a clear file read error.
	_, err = CompileCELWithOptions(filepath.Join(tmpDir, "missing.celpolicy"), cfg, CompileOptions{Policy: true})
	if err == nil || !strings.Contains(err.Error(), "reading policy file") {
		t.Errorf("expected reading policy file error, got: %v", err)
	}

	// Unsupported tag inside rule/match still fails policy parsing.
	badRuleTag := `name: bad_rule
rule:
  unknown_rule_tag: true
  match:
    - output: "true"
`
	_, err = CompileCELWithOptions(badRuleTag, cfg, CompileOptions{Policy: true})
	if err == nil || !strings.Contains(err.Error(), "unsupported rule tag") {
		t.Errorf("expected unsupported rule tag error, got: %v", err)
	}
}
