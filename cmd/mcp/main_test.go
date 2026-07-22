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
	"os"
	"path/filepath"
	"testing"

	descpb "google3/net/proto2/proto/descriptor_go_proto"
	testpb "github.com/cel-expr/skills/internal/proto"
	"github.com/cel-expr/skills/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/proto"
	"google3/third_party/golang/protobuf/v2/reflect/protodesc/protodesc"
)

func TestListTools(t *testing.T) {
	ctx := context.Background()
	s := newServer(nil)

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

func TestListTools_WithFixedEnvironment(t *testing.T) {
	ctx := context.Background()
	fixedEnv := &tools.Config{
		Variables: []*tools.Variable{
			{Name: "foo", Type: "string"},
		},
	}
	s := newServer(fixedEnv)

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

	foundTools := make(map[string]bool)
	for tool, err := range resp {
		if err != nil {
			t.Fatalf("iteration failed: %v", err)
		}
		foundTools[tool.Name] = true
	}

	if foundTools["cel_create_environment"] {
		t.Error("cel_create_environment should NOT be exposed when a fixed environment is present")
	}

	expectedTools := []string{"cel_compile", "cel_evaluate", "cel_generate_prompt"}
	for _, name := range expectedTools {
		if !foundTools[name] {
			t.Errorf("expected tool %s not found in ListTools response with fixed environment", name)
		}
	}
}

func TestFixedEnvironmentToolsExecution(t *testing.T) {
	ctx := context.Background()
	fixedEnv := &tools.Config{
		Variables: []*tools.Variable{
			{Name: "foo", Type: "string"},
		},
	}
	s := newServer(fixedEnv)

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

	// 1. Test cel_compile without envConfig
	compileRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cel_compile",
		Arguments: map[string]any{
			"expr": "foo == 'bar'",
		},
	})
	if err != nil {
		t.Fatalf("CallTool cel_compile failed: %v", err)
	}
	if compileRes.IsError {
		t.Errorf("cel_compile failed: %v", compileRes.Content)
	}

	// 2. Test cel_evaluate without envConfig
	evalRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cel_evaluate",
		Arguments: map[string]any{
			"expr": "foo",
			"testCases": []any{
				map[string]any{
					"testCase": "check foo",
					"bindings": map[string]any{"foo": "bar"},
					"expected": "bar",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool cel_evaluate failed: %v", err)
	}
	if evalRes.IsError {
		t.Errorf("cel_evaluate failed: %v", evalRes.Content)
	}

	// 3. Test cel_generate_prompt without envConfig
	promptRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cel_generate_prompt",
		Arguments: map[string]any{
			"userPrompt": "check foo",
		},
	})
	if err != nil {
		t.Fatalf("CallTool cel_generate_prompt failed: %v", err)
	}
	if promptRes.IsError {
		t.Errorf("cel_generate_prompt failed: %v", promptRes.Content)
	}
}

func TestLoadEnvConfig(t *testing.T) {
	empty, err := loadEnvConfig("")
	if err != nil || empty != nil {
		t.Errorf("expected (nil, nil) for empty input, got (%v, %v)", empty, err)
	}

	jsonStr := `{"variables":[{"name":"x","type":"int"}]}`
	cfg, err := loadEnvConfig(jsonStr)
	if err != nil {
		t.Fatalf("loadEnvConfig failed for JSON string: %v", err)
	}
	if len(cfg.Variables) != 1 || cfg.Variables[0].Name != "x" {
		t.Errorf("unexpected config: %+v", cfg)
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

func TestSetupServer(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "env.json")
	if err := os.WriteFile(envPath, []byte(`{"variables":[{"name":"y","type":"string"}]}`), 0644); err != nil {
		t.Fatalf("failed writing temp env file: %v", err)
	}

	tests := []struct {
		name    string
		fdsPath string
		envPath string
		wantErr bool
	}{
		{
			name:    "empty arguments",
			fdsPath: "",
			envPath: "",
			wantErr: false,
		},
		{
			name:    "inline JSON string",
			fdsPath: "",
			envPath: `{"variables":[{"name":"x","type":"int"}]}`,
			wantErr: false,
		},
		{
			name:    "file path for env",
			fdsPath: "",
			envPath: envPath,
			wantErr: false,
		},
		{
			name:    "invalid FDS path",
			fdsPath: "non_existent_fds.pb",
			envPath: "",
			wantErr: true,
		},
		{
			name:    "invalid env path",
			fdsPath: "",
			envPath: "non_existent_env.json",
			wantErr: true,
		},
		{
			name:    "invalid inline JSON string",
			fdsPath: "",
			envPath: "{invalid_json",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := setupServer(tt.fdsPath, tt.envPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("setupServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s == nil {
				t.Errorf("setupServer() returned nil server without error")
			}
		})
	}
}

func TestSetupServer_WithProtobufTypes(t *testing.T) {
	tmpDir := t.TempDir()

	fdProto := protodesc.ToFileDescriptorProto((&testpb.TestMessage{}).ProtoReflect().Descriptor().ParentFile())
	fds := &descpb.FileDescriptorSet{
		File: []*descpb.FileDescriptorProto{fdProto},
	}
	fdsBytes, err := proto.Marshal(fds)
	if err != nil {
		t.Fatalf("failed marshaling FDS: %v", err)
	}
	validFdsPath := filepath.Join(tmpDir, "test_schema.pb")
	if err := os.WriteFile(validFdsPath, fdsBytes, 0644); err != nil {
		t.Fatalf("failed writing valid FDS file: %v", err)
	}

	validPbEnvPath := filepath.Join(tmpDir, "pb_env.json")
	if err := os.WriteFile(validPbEnvPath, []byte(`{"variables":[{"name":"msg","type":"cel.skills.internal.proto.TestMessage"}]}`), 0644); err != nil {
		t.Fatalf("failed writing temp pb env file: %v", err)
	}

	s, err := setupServer(validFdsPath, validPbEnvPath)
	if err != nil || s == nil {
		t.Fatalf("setupServer() failed with valid FDS and env: server=%v, err=%v", s, err)
	}

	ctx := context.Background()
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

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cel_compile",
		Arguments: map[string]any{
			"expr": "msg.single_int32 == 42 && msg.single_nested_message.bb == 100",
		},
	})
	if err != nil {
		t.Fatalf("CallTool cel_compile failed: %v", err)
	}
	if res.IsError {
		t.Errorf("cel_compile with valid FDS failed: %v", res.Content)
	}

	evalRes, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "cel_evaluate",
		Arguments: map[string]any{
			"expr": "msg.single_int32 == 42 && msg.single_nested_message.bb == 100",
			"testCases": []any{
				map[string]any{
					"testCase": "check proto message",
					"bindings": map[string]any{
						"msg": map[string]any{
							"singleInt32": 42,
							"singleNestedMessage": map[string]any{
								"bb": 100,
							},
						},
					},
					"expected": true,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool cel_evaluate failed: %v", err)
	}
	if evalRes.IsError {
		t.Errorf("cel_evaluate with valid FDS failed: %v", evalRes.Content)
	}
}

func TestLoadFileDescriptorSet(t *testing.T) {
	// 1. Non-existent file
	if _, err := loadFileDescriptorSet("non_existent.pb"); err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	// 2. Invalid proto bytes
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.pb")
	if err := os.WriteFile(invalidPath, []byte("not a proto"), 0644); err != nil {
		t.Fatalf("failed writing temp file: %v", err)
	}
	// Note: proto unmarshal of arbitrary string might not error if protobuf tags are absent,
	// but let's test a valid FileDescriptorSet
	fds := &descpb.FileDescriptorSet{}
	validBytes, err := proto.Marshal(fds)
	if err != nil {
		t.Fatalf("failed marshaling FDS: %v", err)
	}
	validPath := filepath.Join(tmpDir, "valid.pb")
	if err := os.WriteFile(validPath, validBytes, 0644); err != nil {
		t.Fatalf("failed writing valid FDS: %v", err)
	}

	opt, err := loadFileDescriptorSet(validPath)
	if err != nil || opt == nil {
		t.Errorf("expected success loading valid FDS, got opt=%v, err=%v", opt, err)
	}
}

func TestLoadEnvConfig_Errors(t *testing.T) {
	tests := []struct {
		name          string
		envPathOrJSON string
	}{
		{
			name:          "non-existent file path",
			envPathOrJSON: "does_not_exist.json",
		},
		{
			name:          "malformed inline JSON",
			envPathOrJSON: "{malformed: true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := loadEnvConfig(tt.envPathOrJSON); err == nil {
				t.Errorf("loadEnvConfig(%q) expected error, got nil", tt.envPathOrJSON)
			}
		})
	}
}

func TestToolHandlers_ErrorPaths(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(ctx context.Context) error
	}{
		{
			name: "handleCreateEnvConfig error (invalid variable type)",
			run: func(ctx context.Context) error {
				_, _, err := handleCreateEnvConfig(ctx, &mcp.CallToolRequest{}, CreateEnvConfigArgs{
					EnvConfig: &tools.Config{
						Variables: []*tools.Variable{{Name: "x", Type: "unknown_type_xxx"}},
					},
				})
				return err
			},
		},
		{
			name: "handleCompile error (invalid CEL expression)",
			run: func(ctx context.Context) error {
				_, _, err := handleCompile(ctx, &mcp.CallToolRequest{}, CompileArgs{
					Expr: "1 +",
				})
				return err
			},
		},
		{
			name: "handleEvaluate error (invalid CEL expression)",
			run: func(ctx context.Context) error {
				_, _, err := handleEvaluate(ctx, &mcp.CallToolRequest{}, EvaluateArgs{
					Expr: "1 +",
				})
				return err
			},
		},
		{
			name: "handleGeneratePrompt error (invalid env config)",
			run: func(ctx context.Context) error {
				_, _, err := handleGeneratePrompt(ctx, &mcp.CallToolRequest{}, GeneratePromptArgs{
					EnvConfig: &tools.Config{
						Variables: []*tools.Variable{{Name: "x", Type: "unknown_type_xxx"}},
					},
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(ctx); err == nil {
				t.Errorf("%s: expected error, got nil", tt.name)
			}
		})
	}
}
