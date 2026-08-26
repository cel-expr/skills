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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testpb "github.com/cel-expr/skills/internal/proto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	descpb "google.golang.org/protobuf/types/descriptorpb"
)

func TestCLI_UsageAndHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", []string{}},
		{"help command", []string{"help"}},
		{"-h flag", []string{"-h"}},
		{"--help flag", []string{"--help"}},
		{"command help", []string{"help", "compile"}},
		{"eval help", []string{"help", "eval"}},
		{"env help", []string{"help", "env"}},
		{"prompt help", []string{"help", "prompt"}},
		{"mcp help", []string{"help", "mcp"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, &stdout, &stderr, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stdout.String(), "cel-expr") && !strings.Contains(stderr.String(), "cel-expr") {
				t.Errorf("expected usage output, got stdout=%q, stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestCLI_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"unknown_cmd"}, &stdout, &stderr, nil)
	if err == nil {
		t.Error("expected error for unknown command, got nil")
	}
}

func TestCLI_Compile(t *testing.T) {
	envJSON := `{"extensions":[{"name":"math"}],"variables":[{"name":"a","type":"int"},{"name":"b","type":"int"}]}`

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "compile valid expression with flags",
			args:    []string{"compile", "-env", envJSON, "-expr", "math.greatest(a, b) > 0"},
			wantErr: false,
		},
		{
			name:    "compile valid expression with positional argument",
			args:    []string{"compile", "-env", envJSON, "math.greatest(a, b) > 0"},
			wantErr: false,
		},
		{
			name:    "compile missing expression",
			args:    []string{"compile", "-env", envJSON},
			wantErr: true,
		},
		{
			name:    "compile missing env",
			args:    []string{"compile", "-expr", "1 + 1"},
			wantErr: true,
		},
		{
			name:    "compile invalid expression syntax",
			args:    []string{"compile", "-env", envJSON, "-expr", "1 +"},
			wantErr: true,
		},
		{
			name:    "compile invalid env json",
			args:    []string{"compile", "-env", "{invalid_json", "-expr", "1 == 1"},
			wantErr: true,
		},
		{
			name:    "compile invalid fds flag",
			args:    []string{"compile", "-env", envJSON, "-fds", "non_existent.pb", "-expr", "1 == 1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, &stdout, &stderr, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("run(compile) error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !strings.Contains(stdout.String(), "inputSchema") {
				t.Errorf("expected inputSchema in output, got %q", stdout.String())
			}
		})
	}
}

func TestCLI_Eval_TestSuite(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "env.json")
	if err := os.WriteFile(envPath, []byte(`{"extensions":[{"name":"math"}],"variables":[{"name":"a","type":"int"},{"name":"b","type":"int"},{"name":"c","type":"int"}]}`), 0644); err != nil {
		t.Fatalf("failed writing env: %v", err)
	}

	testsPath := filepath.Join(tmpDir, "tests.json")
	testsJSON := `[
		{
			"testCase": "find greatest",
			"bindings": {"a": 10, "b": 42, "c": 7},
			"expected": 42
		}
	]`
	if err := os.WriteFile(testsPath, []byte(testsJSON), 0644); err != nil {
		t.Fatalf("failed writing tests: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := run([]string{"eval", "-env", envPath, "-tests", testsPath, "math.greatest(a, b, c)"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("run(eval) failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), `"status": "pass"`) {
		t.Errorf("expected pass status in output, got: %s", stdout.String())
	}
}

func TestCLI_Eval_Bindings(t *testing.T) {
	envJSON := `{"variables":[{"name":"x","type":"int"},{"name":"y","type":"int"}]}`
	bindingsJSON := `{"x": 10, "y": 20}`

	var stdout, stderr bytes.Buffer
	err := run([]string{"eval", "-env", envJSON, "-bindings", bindingsJSON, "-expected", "30", "x + y"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("run(eval with bindings) failed: %v", err)
	}

	if !strings.Contains(stdout.String(), `"status": "pass"`) {
		t.Errorf("expected pass status, got: %s", stdout.String())
	}
}

func TestCLI_Eval_DefaultEmptyBindings(t *testing.T) {
	envJSON := `{"variables":[]}`

	var stdout, stderr bytes.Buffer
	err := run([]string{"eval", "-env", envJSON, "1 + 1"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("run(eval default bindings) failed: %v", err)
	}
	if !strings.Contains(stdout.String(), `"status": "pass"`) {
		t.Errorf("expected pass status, got: %s", stdout.String())
	}
}

func TestCLI_Eval_Errors(t *testing.T) {
	envJSON := `{"variables":[{"name":"x","type":"int"}]}`

	tests := []struct {
		name string
		args []string
	}{
		{"missing expression", []string{"eval", "-env", envJSON}},
		{"invalid test suite json", []string{"eval", "-env", envJSON, "-tests", "{not_an_array", "x == 10"}},
		{"missing tests file", []string{"eval", "-env", envJSON, "-tests", "non_existent_tests.json", "x == 10"}},
		{"invalid bindings json", []string{"eval", "-env", envJSON, "-bindings", "{not_an_obj", "x == 10"}},
		{"missing bindings file", []string{"eval", "-env", envJSON, "-bindings", "non_existent_bindings.json", "x == 10"}},
		{"failing test evaluation mismatch", []string{"eval", "-env", envJSON, "-bindings", `{"x": 10}`, "-expected", "999", "x"}},
		{"eval compilation error", []string{"eval", "-env", envJSON, "1 +"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(tt.args, &stdout, &stderr, nil)
			if err == nil {
				t.Errorf("%s: expected error, got nil", tt.name)
			}
		})
	}
}

func TestCLI_EnvValidate(t *testing.T) {
	validEnv := `{"name":"test_env","variables":[{"name":"x","type":"int"}]}`
	invalidEnv := `{"name":"test_env","variables":[{"name":"x","type":"unknown_type_xxx"}]}`

	var stdout, stderr bytes.Buffer
	err := run([]string{"env", "-env", validEnv}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("expected valid env check, got error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"valid": true`) {
		t.Errorf("expected valid: true, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{"env", "-env", invalidEnv}, &stdout, &stderr, nil)
	if err == nil {
		t.Error("expected error for invalid variable type, got nil")
	}

	stdout.Reset()
	stderr.Reset()
	err = run([]string{"env"}, &stdout, &stderr, nil)
	if err == nil {
		t.Error("expected error when env flag is missing, got nil")
	}
}

func TestCLI_Prompt(t *testing.T) {
	envJSON := `{"name":"test_env","variables":[{"name":"user_id","type":"string"}]}`

	var stdout, stderr bytes.Buffer
	err := run([]string{"prompt", "-env", envJSON, "-prompt", "check if user_id is admin"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("run(prompt) failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "user_id") {
		t.Errorf("expected user_id in generated prompt, got %s", stdout.String())
	}

	// Positional prompt
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"prompt", "-env", envJSON, "check", "if", "user_id", "is", "admin"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("run(prompt positional) failed: %v", err)
	}

	// Missing prompt
	stdout.Reset()
	stderr.Reset()
	err = run([]string{"prompt", "-env", envJSON}, &stdout, &stderr, nil)
	if err == nil {
		t.Error("expected error when prompt is missing, got nil")
	}
}

func TestCLI_WithFileDescriptorSet(t *testing.T) {
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

	pbEnv := `{"variables":[{"name":"msg","type":"cel.skills.internal.proto.TestMessage"}]}`

	var stdout, stderr bytes.Buffer
	err = run([]string{"compile", "-env", pbEnv, "-fds", validFdsPath, "msg.single_int32 == 42"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("compile with FDS failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	bindingsJSON := `{"msg":{"singleInt32":42}}`
	err = run([]string{"eval", "-env", pbEnv, "-fds", validFdsPath, "-bindings", bindingsJSON, "-expected", "true", "msg.single_int32 == 42"}, &stdout, &stderr, nil)
	if err != nil {
		t.Fatalf("eval with FDS failed: %v", err)
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}

func TestCLI_Stdin(t *testing.T) {
	envJSON := `{"variables":[{"name":"x","type":"int"}]}`
	stdin := strings.NewReader(envJSON)

	var stdout, stderr bytes.Buffer
	err := run([]string{"compile", "-env", "-", "x > 0"}, &stdout, &stderr, stdin)
	if err != nil {
		t.Fatalf("compile reading env from stdin failed: %v", err)
	}

	// Test stdin read error
	err = run([]string{"compile", "-env", "-", "x > 0"}, &stdout, &stderr, errReader{})
	if err == nil {
		t.Error("expected error on stdin read failure, got nil")
	}
}

func TestCLI_MCP_Errors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"mcp", "-env", "{invalid_json"}, &stdout, &stderr, nil)
	if err == nil {
		t.Error("expected error when starting MCP server with invalid env, got nil")
	}
}
