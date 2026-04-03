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

package main

import (
	"context"
	"testing"

	"github.com/cel-expr/skills/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListTools(t *testing.T) {
	ctx := context.Background()
	s := newServer()

	// Connect the server and client using in-memory transports.
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := s.Connect(ctx, t1, nil); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, t2, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer session.Close()

	resp := session.Tools(ctx, nil)

	expectedTools := map[string]bool{
		"cel_compile":            true,
		"cel_evaluate":           true,
		"cel_generate_prompt":    true,
		"cel_create_environment": true,
	}

	foundTools := make(map[string]bool)
	for tool, err := range resp {
		if err != nil {
			t.Fatalf("iteration failed: %v", err)
		}
		foundTools[tool.Name] = true
	}

	for name := range expectedTools {
		if !foundTools[name] {
			t.Errorf("expected tool %s not found in ListTools response", name)
		}
	}
}

type EvaluationResults struct {
	TestCase string `json:"testCase"`
	Status   string `json:"status"`
}

type EvaluateExprOutputSchema struct {
	EvaluationResults []EvaluationResults `json:"evaluationResults"`
	Coverage          string              `json:"coverage"`
}

func TestHandleCreateEnvConfig(t *testing.T) {
	ctx := context.Background()

	args := CreateEnvConfigArgs{
		EnvConfig: &tools.Config{
			Variables: []*tools.Variable{
				{Name: "foo", Type: "string"},
			},
		},
	}

	res, _, err := handleCreateEnvConfig(ctx, &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleCreateEnvConfig failed: %v", err)
	}

	if res.IsError {
		t.Errorf("expected success, got error: %v", res.Content[0])
	}
}

func TestHandleCompile(t *testing.T) {
	ctx := context.Background()

	args := CompileArgs{
		EnvConfig: &tools.Config{
			Variables: []*tools.Variable{
				{Name: "foo", Type: "string"},
			},
		},
		Expr: "foo == 'bar'",
	}

	res, out, err := handleCompile(ctx, &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleCompile failed: %v", err)
	}

	if res != nil && res.IsError {
		t.Errorf("expected success, got error: %v", res.Content[0])
	}

	if out == nil {
		t.Fatal("expected output, got nil")
	}
}

func TestHandleEvaluate(t *testing.T) {
	ctx := context.Background()

	args := EvaluateArgs{
		EnvConfig: &tools.Config{
			Variables: []*tools.Variable{
				{Name: "foo", Type: "string"},
			},
		},
		Expr: "foo",
		TestCases: []tools.TestCase{
			{
				TestCase: "happy path",
				Bindings: map[string]any{"foo": "bar"},
				Expected: "bar",
			},
		},
	}

	res, out, err := handleEvaluate(ctx, &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleEvaluate failed: %v", err)
	}

	if res != nil && res.IsError {
		t.Errorf("expected success, got error: %v", res.Content[0])
	}

	if out == nil {
		t.Fatal("expected output, got nil")
	}

	tr := out.(*tools.EvaluateExprOutput)
	if len(tr.TestResults) == 0 {
		t.Fatal("expected evaluation results, got none")
	}

	if tr.TestResults[0].Status != "pass" {
		t.Errorf("expected 'pass', got '%s'", tr.TestResults[0].Status)
	}
}

func TestHandleGeneratePrompt(t *testing.T) {
	ctx := context.Background()

	args := GeneratePromptArgs{
		EnvConfig: &tools.Config{
			Variables: []*tools.Variable{
				{Name: "foo", Type: "string"},
			},
		},
		UserPrompt: "create a rule that checks if foo is 'bar'",
	}

	res, _, err := handleGeneratePrompt(ctx, &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleGeneratePrompt failed: %v", err)
	}

	if res.IsError {
		t.Errorf("expected success, got error: %v", res.Content[0])
	}

	if len(res.Content) == 0 {
		t.Error("expected non-empty output content")
	}
}

func TestHandleEvaluateUserAge(t *testing.T) {
	ctx := context.Background()

	args := EvaluateArgs{
		EnvConfig: &tools.Config{
			Variables: []*tools.Variable{
				{Name: "user.age", Type: "int"},
			},
		},
		Expr: "// Check if user age is over 18\nuser.age > 18",
		TestCases: []tools.TestCase{
			{
				TestCase: "Age is 19",
				Bindings: map[string]any{"user.age": 19},
				Expected: true,
			},
			{
				TestCase: "Age is exactly 18",
				Bindings: map[string]any{"user.age": 18},
				Expected: false,
			},
			{
				TestCase: "Age is under 18",
				Bindings: map[string]any{"user.age": 17},
				Expected: false,
			},
		},
	}

	res, out, err := handleEvaluate(ctx, &mcp.CallToolRequest{}, args)
	if err != nil {
		t.Fatalf("handleEvaluate failed: %v", err)
	}

	if res != nil && res.IsError {
		t.Errorf("expected success, got error: %v", res.Content[0])
	}

	if out == nil {
		t.Fatal("expected output, got nil")
	}

	expectedStatusFound := map[string]bool{
		"Age is 19":         false,
		"Age is exactly 18": false,
		"Age is under 18":   false,
	}

	tr := out.(*tools.EvaluateExprOutput)
	for _, result := range tr.TestResults {
		if result.Status != "pass" {
			t.Errorf("test case '%s' failed: %s", result.TestCase, result.Status)
		}
		expectedStatusFound[result.TestCase] = true
	}

	for tc, found := range expectedStatusFound {
		if !found {
			t.Errorf("expected test case result for '%s' not found", tc)
		}
	}
}
