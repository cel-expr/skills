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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvaluateCEL(t *testing.T) {
	envJSON := &Config{
		Name: "basic",
		Variables: []*Variable{
			{Name: "user", Type: "string"},
			{Name: "age", Type: "int"},
		},
	}
	namespaceEnvJSON := &Config{
		Name: "namespace",
		Variables: []*Variable{
			{Name: "request.name", Type: "string"},
			{Name: "request.path", Type: "string"},
			{Name: "request.method", Type: "string"},
		},
	}

	tests := []struct {
		name         string
		expr         string
		envConfig    *Config
		testCases    []TestCase
		wantContains string
		wantCoverage string
		wantErr      bool
	}{
		{
			name:         "successful evaluation",
			expr:         `user == "Alice" && age > 18`,
			envConfig:    envJSON,
			testCases:    []TestCase{{TestCase: "test-1", Bindings: map[string]any{"user": "Alice", "age": 20}, Expected: true}},
			wantContains: `"testCase":"test-1","status":"pass"`,
			wantErr:      false,
			wantCoverage: "Node: 100.00%, Branch: 50.00%",
		},
		{
			name:         "failed evaluation (returns false)",
			expr:         `user == "Alice" && age > 18`,
			envConfig:    envJSON,
			testCases:    []TestCase{{TestCase: "test-2", Bindings: map[string]any{"user": "Bob", "age": 20}, Expected: false}},
			wantContains: `"testCase":"test-2","status":"pass"`,
			wantErr:      false,
			wantCoverage: "Node: 57.14%, Branch: 33.33%",
		},
		{
			name:         "invalid bindings json",
			expr:         `user == "Alice"`,
			envConfig:    envJSON,
			testCases:    []TestCase{{TestCase: "test-3", Bindings: map[string]any{"user": "Bob"}, Expected: false}},
			wantErr:      false,
			wantCoverage: "Node: 100.00%, Branch: 50.00%",
		},
		{
			name:         "missing variables in binding",
			expr:         `user == "Alice" && age > 18`,
			envConfig:    envJSON,
			testCases:    []TestCase{{TestCase: "test-4", Bindings: map[string]any{"user": "Alice"}, Expected: false}},
			wantContains: `"status":"no such attribute(s): age"`,
			wantErr:      false, // Evaluate is resilient to per-test failures but catches them in the result status
		},
		{
			name:      "compile error due to bad syntax",
			expr:      `user ==`,
			envConfig: envJSON,
			testCases: []TestCase{},
			wantErr:   true,
		},
		{
			name:         "evaluation returns null",
			expr:         `null`,
			envConfig:    envJSON,
			testCases:    []TestCase{{TestCase: "test-null", Bindings: map[string]any{}, Expected: nil}},
			wantContains: `"status":"pass"`,
			wantErr:      false,
		},
		{
			name:         "failed constructing env",
			expr:         "true",
			envConfig:    &Config{Variables: []*Variable{{Name: "a", Type: "invalid"}}},
			testCases:    []TestCase{},
			wantErr:      true,
			wantCoverage: "Node: 100.00%, Branch: 100.00%",
		},
		{
			name:         "namespace evaluation",
			expr:         `request.name == "test"`,
			envConfig:    namespaceEnvJSON,
			testCases:    []TestCase{{TestCase: "test-namespace", Bindings: map[string]any{"request.name": "test"}, Expected: true}},
			wantContains: `"status":"pass"`,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateCEL(tt.expr, tt.envConfig, tt.testCases)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateCEL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.wantContains != "" {
				gotBytes, _ := json.Marshal(got)
				gotStr := string(gotBytes)
				if !strings.Contains(gotStr, tt.wantContains) {
					t.Errorf("EvaluateCEL() got = %v, want it to contain %v", gotStr, tt.wantContains)
				}
			}
			if !tt.wantErr && tt.wantCoverage != "" {
				if got.Coverage != tt.wantCoverage {
					t.Errorf("EvaluateCEL() coverage = %v, want %v", got.Coverage, tt.wantCoverage)
				}
			}
		})
	}
}

func TestEvaluateTestData(t *testing.T) {
	// 1. Read testdata/request_headers_env.json
	envData, err := os.ReadFile(filepath.Join("testdata", "request_headers_env.json"))
	if err != nil {
		t.Fatalf("failed to read env config: %v", err)
	}
	var envConfig Config
	if err := json.Unmarshal(envData, &envConfig); err != nil {
		t.Fatalf("failed to unmarshal env config: %v", err)
	}

	// 2. Read testdata/user_agent_mozilla.cel
	exprData, err := os.ReadFile(filepath.Join("testdata", "user_agent_mozilla.cel"))
	if err != nil {
		t.Fatalf("failed to read expression: %v", err)
	}
	expr := string(exprData)

	// 3. Read testdata/user_agent_test.json
	testCasesData, err := os.ReadFile(filepath.Join("testdata", "user_agent_test.json"))
	if err != nil {
		t.Fatalf("failed to read test cases: %v", err)
	}
	var testCases []TestCase
	if err := json.Unmarshal(testCasesData, &testCases); err != nil {
		t.Fatalf("failed to unmarshal test cases: %v", err)
	}

	// 4. Call EvaluateCEL
	got, err := EvaluateCEL(expr, &envConfig, testCases)
	if err != nil {
		t.Fatalf("EvaluateCEL() error = %v", err)
	}

	// 5. Verify the output matches the expectations
	// We expect all 3 test cases to pass (status: "pass")
	for _, res := range got.TestResults {
		if res.Status != "pass" {
			t.Errorf("test case '%s' failed: status = %s", res.TestCase, res.Status)
		}
	}

	// Verify count
	if len(got.TestResults) != 3 {
		t.Errorf("expected 3 evaluation results, got %d", len(got.TestResults))
	}
}
