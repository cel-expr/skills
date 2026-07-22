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

// Package main is the main package for the CEL MCP.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	descpb "google3/net/proto2/proto/descriptor_go_proto"
	"github.com/google/cel-go/cel"
	"google.golang.org/protobuf/proto"

	"github.com/cel-expr/skills/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	fileDescriptorSet := flag.String("file_descriptor_set", "", "Path to a binary FileDescriptorSet file containing protobuf definitions.")
	flag.String("file_descriptors", "", "Alias for -file_descriptor_set: path to a binary FileDescriptorSet file.")
	environmentFlag := flag.String("environment", "", "Path to a JSON file or JSON string containing the CEL environment configuration.")
	flag.String("env", "", "Alias for -environment: path to a JSON file or JSON string containing the CEL environment configuration.")
	flag.Parse()

	fdsPath := *fileDescriptorSet
	if fdsPath == "" {
		if f := flag.Lookup("file_descriptors"); f != nil {
			fdsPath = f.Value.String()
		}
	}

	envPath := *environmentFlag
	if envPath == "" {
		if f := flag.Lookup("env"); f != nil {
			envPath = f.Value.String()
		}
	}

	s, err := setupServer(fdsPath, envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		os.Exit(1)
	}

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func setupServer(fdsPath, envPath string) (*mcp.Server, error) {
	var opts []cel.EnvOption
	if fdsPath != "" {
		fdOpt, err := loadFileDescriptorSet(fdsPath)
		if err != nil {
			return nil, fmt.Errorf("failed loading file descriptor set %q: %w", fdsPath, err)
		}
		opts = append(opts, fdOpt)
	}

	var fixedEnv *tools.Config
	if envPath != "" {
		var err error
		fixedEnv, err = loadEnvConfig(envPath)
		if err != nil {
			return nil, fmt.Errorf("failed loading environment configuration %q: %w", envPath, err)
		}
	}

	return newServer(fixedEnv, opts...), nil
}

func loadFileDescriptorSet(path string) (cel.EnvOption, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fds descpb.FileDescriptorSet
	if err := proto.Unmarshal(data, &fds); err != nil {
		return nil, err
	}
	return cel.TypeDescs(&fds), nil
}

func loadEnvConfig(envPathOrJSON string) (*tools.Config, error) {
	if envPathOrJSON == "" {
		return nil, nil
	}
	var data []byte
	if strings.HasPrefix(strings.TrimSpace(envPathOrJSON), "{") {
		data = []byte(envPathOrJSON)
	} else {
		var err error
		data, err = os.ReadFile(envPathOrJSON)
		if err != nil {
			return nil, fmt.Errorf("failed reading environment file %q: %w", envPathOrJSON, err)
		}
	}
	return tools.ConfigFromJSON(string(data))
}

type toolsHandler struct {
	fixedEnv *tools.Config
	opts     []cel.EnvOption
}

func newServer(fixedEnv *tools.Config, opts ...cel.EnvOption) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "cel-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		SchemaCache: mcp.NewSchemaCache(),
	})

	h := &toolsHandler{
		fixedEnv: fixedEnv,
		opts:     opts,
	}

	if fixedEnv != nil {
		mcp.AddTool[FixedEnvGeneratePromptArgs, any](s, &mcp.Tool{
			Name:        "cel_generate_prompt",
			Description: "Generates an LLM authoring prompt explaining the exact variables, functions, and types available in the CEL environment.",
		}, h.handleFixedEnvGeneratePrompt)
		mcp.AddTool[FixedEnvCompileArgs, any](s, &mcp.Tool{
			Name:        "cel_compile",
			Description: "Compiles a CEL expression. Returns the expression's input and output JSON schemas if it compiles successfully.",
		}, h.handleFixedEnvCompile)
		mcp.AddTool[FixedEnvEvaluateArgs, any](s, &mcp.Tool{
			Name:        "cel_evaluate",
			Description: "Evaluates a CEL expression against provided test cases. Returns test case results and coverage.",
		}, h.handleFixedEnvEvaluate)
	} else {
		mcp.AddTool[CreateEnvConfigArgs, any](s, &mcp.Tool{
			Name:        "cel_create_environment",
			Description: "Creates a CEL environment configuration from a JSON object.",
		}, h.handleCreateEnvConfig)
		mcp.AddTool[GeneratePromptArgs, any](s, &mcp.Tool{
			Name:        "cel_generate_prompt",
			Description: "Generates an LLM authoring prompt explaining the exact variables, functions, and types available in the CEL environment.",
		}, h.handleGeneratePrompt)
		mcp.AddTool[CompileArgs, any](s, &mcp.Tool{
			Name:        "cel_compile",
			Description: "Compiles a CEL expression using a JSON environment configuration. Returns the expression's input and output JSON schemas if it compiles successfully.",
		}, h.handleCompile)
		mcp.AddTool[EvaluateArgs, any](s, &mcp.Tool{
			Name:        "cel_evaluate",
			Description: "Evaluates a CEL expression against provided test cases. Returns test case results and coverage.",
		}, h.handleEvaluate)
	}

	return s
}

// CreateEnvConfigArgs is the arguments for the cel_create_environment tool.
type CreateEnvConfigArgs struct {
	// EnvConfig is the JSON string representing the CEL environment configuration.
	EnvConfig *tools.Config `json:"envConfig" jsonschema_description:"The JSON string representing the CEL environment configuration."`
}

// GeneratePromptArgs is the arguments for the cel_generate_prompt tool.
type GeneratePromptArgs struct {
	// EnvConfig is the JSON string representing the CEL environment configuration.
	EnvConfig *tools.Config `json:"envConfig" jsonschema_description:"The JSON string representing the CEL environment schema."`

	// UserPrompt is the user prompt to generate the CEL expression for.
	UserPrompt string `json:"userPrompt" jsonschema_description:"The user prompt to generate the CEL expression for."`
}

// CompileArgs is the arguments for the cel_compile tool.
type CompileArgs struct {
	// EnvConfig is the JSON string representing the CEL environment configuration.
	EnvConfig *tools.Config `json:"envConfig" jsonschema_description:"The JSON string representing the CEL environment schema."`

	// Expr is the CEL expression to compile.
	Expr string `json:"expr" jsonschema_description:"The CEL expression to compile."`
}

// EvaluateArgs is the arguments for the cel_evaluate tool.
type EvaluateArgs struct {
	// EnvConfig is the JSON string representing the CEL environment configuration.
	EnvConfig *tools.Config `json:"envConfig" jsonschema_description:"The JSON string representing the CEL environment schema."`

	// Expr is the CEL expression to evaluate.
	Expr string `json:"expr" jsonschema_description:"The CEL expression to evaluate."`

	// TestCases is the test cases for evaluation.
	TestCases []tools.TestCase `json:"testCases" jsonschema_description:"The test cases for evaluation."`
}

// FixedEnvGeneratePromptArgs is the arguments for the cel_generate_prompt tool when a static environment is configured.
type FixedEnvGeneratePromptArgs struct {
	// UserPrompt is the user prompt to generate the CEL expression for.
	UserPrompt string `json:"userPrompt" jsonschema_description:"The user prompt to generate the CEL expression for."`
}

// FixedEnvCompileArgs is the arguments for the cel_compile tool when a static environment is configured.
type FixedEnvCompileArgs struct {
	// Expr is the CEL expression to compile.
	Expr string `json:"expr" jsonschema_description:"The CEL expression to compile."`
}

// FixedEnvEvaluateArgs is the arguments for the cel_evaluate tool when a static environment is configured.
type FixedEnvEvaluateArgs struct {
	// Expr is the CEL expression to evaluate.
	Expr string `json:"expr" jsonschema_description:"The CEL expression to evaluate."`

	// TestCases is the test cases for evaluation.
	TestCases []tools.TestCase `json:"testCases" jsonschema_description:"The test cases for evaluation."`
}

func (h *toolsHandler) handleCreateEnvConfig(ctx context.Context, request *mcp.CallToolRequest, args CreateEnvConfigArgs) (*mcp.CallToolResult, any, error) {
	_, err := tools.EnvFromConfig(args.EnvConfig, h.opts...)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Environment created successfully"}},
	}, nil, nil
}

func (h *toolsHandler) handleCompile(ctx context.Context, request *mcp.CallToolRequest, args CompileArgs) (*mcp.CallToolResult, any, error) {
	envConfig := args.EnvConfig
	if envConfig == nil {
		envConfig = h.fixedEnv
	}
	if envConfig == nil {
		return nil, nil, fmt.Errorf("environment configuration is required")
	}
	res, err := tools.CompileCEL(args.Expr, envConfig, h.opts...)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (h *toolsHandler) handleEvaluate(ctx context.Context, request *mcp.CallToolRequest, args EvaluateArgs) (*mcp.CallToolResult, any, error) {
	envConfig := args.EnvConfig
	if envConfig == nil {
		envConfig = h.fixedEnv
	}
	if envConfig == nil {
		return nil, nil, fmt.Errorf("environment configuration is required")
	}
	res, err := tools.EvaluateCEL(args.Expr, envConfig, args.TestCases, h.opts...)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (h *toolsHandler) handleGeneratePrompt(ctx context.Context, request *mcp.CallToolRequest, args GeneratePromptArgs) (*mcp.CallToolResult, any, error) {
	envConfig := args.EnvConfig
	if envConfig == nil {
		envConfig = h.fixedEnv
	}
	if envConfig == nil {
		return nil, nil, fmt.Errorf("environment configuration is required")
	}
	res, err := tools.GeneratePrompt(envConfig, args.UserPrompt, h.opts...)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: res}},
	}, nil, nil
}

func (h *toolsHandler) handleFixedEnvGeneratePrompt(ctx context.Context, request *mcp.CallToolRequest, args FixedEnvGeneratePromptArgs) (*mcp.CallToolResult, any, error) {
	return h.handleGeneratePrompt(ctx, request, GeneratePromptArgs{
		UserPrompt: args.UserPrompt,
	})
}

func (h *toolsHandler) handleFixedEnvCompile(ctx context.Context, request *mcp.CallToolRequest, args FixedEnvCompileArgs) (*mcp.CallToolResult, any, error) {
	return h.handleCompile(ctx, request, CompileArgs{
		Expr: args.Expr,
	})
}

func (h *toolsHandler) handleFixedEnvEvaluate(ctx context.Context, request *mcp.CallToolRequest, args FixedEnvEvaluateArgs) (*mcp.CallToolResult, any, error) {
	return h.handleEvaluate(ctx, request, EvaluateArgs{
		Expr:      args.Expr,
		TestCases: args.TestCases,
	})
}

func handleCreateEnvConfig(ctx context.Context, request *mcp.CallToolRequest, args CreateEnvConfigArgs) (*mcp.CallToolResult, any, error) {
	return (&toolsHandler{}).handleCreateEnvConfig(ctx, request, args)
}

func handleCompile(ctx context.Context, request *mcp.CallToolRequest, args CompileArgs) (*mcp.CallToolResult, any, error) {
	return (&toolsHandler{}).handleCompile(ctx, request, args)
}

func handleEvaluate(ctx context.Context, request *mcp.CallToolRequest, args EvaluateArgs) (*mcp.CallToolResult, any, error) {
	return (&toolsHandler{}).handleEvaluate(ctx, request, args)
}

func handleGeneratePrompt(ctx context.Context, request *mcp.CallToolRequest, args GeneratePromptArgs) (*mcp.CallToolResult, any, error) {
	return (&toolsHandler{}).handleGeneratePrompt(ctx, request, args)
}
