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
	"fmt"
	"os"

	"github.com/cel-expr/skills/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	s := newServer()

	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func newServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "cel-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		SchemaCache: mcp.NewSchemaCache(),
	})

	mcp.AddTool[CreateEnvConfigArgs, any](s, &mcp.Tool{
		Name:        "cel_create_environment",
		Description: "Creates a CEL environment configuration from a JSON object.",
	}, handleCreateEnvConfig)
	mcp.AddTool[GeneratePromptArgs, any](s, &mcp.Tool{
		Name:        "cel_generate_prompt",
		Description: "Generates an LLM authoring prompt explaining the exact variables, functions, and types available in the CEL environment.",
	}, handleGeneratePrompt)
	mcp.AddTool[CompileArgs, any](s, &mcp.Tool{
		Name:        "cel_compile",
		Description: "Compiles a CEL expression using a JSON environment configuration. Returns the expression's input and output JSON schemas if it compiles successfully.",
	}, handleCompile)
	mcp.AddTool[EvaluateArgs, any](s, &mcp.Tool{
		Name:        "cel_evaluate",
		Description: "Evaluates a CEL expression against provided test cases. Returns test case results and coverage.",
	}, handleEvaluate)

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

func handleCreateEnvConfig(ctx context.Context, request *mcp.CallToolRequest, args CreateEnvConfigArgs) (*mcp.CallToolResult, any, error) {
	_, err := tools.EnvFromConfig(args.EnvConfig)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Environment created successfully"}},
	}, nil, nil
}

func handleCompile(ctx context.Context, request *mcp.CallToolRequest, args CompileArgs) (*mcp.CallToolResult, any, error) {
	res, err := tools.CompileCEL(args.Expr, args.EnvConfig)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func handleEvaluate(ctx context.Context, request *mcp.CallToolRequest, args EvaluateArgs) (*mcp.CallToolResult, any, error) {
	res, err := tools.EvaluateCEL(args.Expr, args.EnvConfig, args.TestCases)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func handleGeneratePrompt(ctx context.Context, request *mcp.CallToolRequest, args GeneratePromptArgs) (*mcp.CallToolResult, any, error) {
	res, err := tools.GeneratePrompt(args.EnvConfig, args.UserPrompt)
	if err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: res}},
	}, nil, nil
}
