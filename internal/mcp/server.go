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

// Package mcp provides the Model Context Protocol (MCP) server for CEL expressions.
package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cel.dev/cel-go/cel"
	"google.golang.org/protobuf/proto"
	descpb "google.golang.org/protobuf/types/descriptorpb"

	"github.com/cel-expr/skills/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SetupServer configures and initializes an MCP Server with the given file descriptor set and
// environment. When useProtoDB is true, the file descriptor set is extended with types resolved
// from the proto database for each environment configuration.
func SetupServer(fdsPath, envPath string, useProtoDB bool) (*mcp.Server, error) {
	baseFDS, err := LoadFileDescriptorSetProto(fdsPath)
	if err != nil {
		return nil, fmt.Errorf("failed loading file descriptor set %q: %w", fdsPath, err)
	}

	fixedEnv, err := LoadEnvConfig(envPath)
	if err != nil {
		return nil, fmt.Errorf("failed loading environment configuration %q: %w", envPath, err)
	}

	var opts []cel.EnvOption
	if !useProtoDB && baseFDS != nil {
		opts = append(opts, cel.TypeDescs(baseFDS))
	}

	return NewServerWithOptions(fixedEnv, baseFDS, useProtoDB, opts...), nil
}

// LoadFileDescriptorSetProto reads a binary FileDescriptorSet proto from the filesystem.
func LoadFileDescriptorSetProto(path string) (*descpb.FileDescriptorSet, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var fds descpb.FileDescriptorSet
	if err := proto.Unmarshal(data, &fds); err != nil {
		return nil, err
	}
	return &fds, nil
}

// LoadFileDescriptorSet reads a binary FileDescriptorSet from the filesystem and returns a cel.EnvOption.
func LoadFileDescriptorSet(path string) (cel.EnvOption, error) {
	fds, err := LoadFileDescriptorSetProto(path)
	if err != nil {
		return nil, err
	}
	if fds == nil {
		return nil, nil
	}
	return cel.TypeDescs(fds), nil
}

// BuildDescriptorEnvOption resolves file descriptor set options for the CEL environment,
// optionally extending it from the descriptor DB.
func BuildDescriptorEnvOption(fdsPath string, cfg *tools.Config, useProtoDB bool) (cel.EnvOption, []string, error) {
	baseFDS, err := LoadFileDescriptorSetProto(fdsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed loading file descriptor set %q: %w", fdsPath, err)
	}

	if !useProtoDB {
		if baseFDS != nil && len(baseFDS.GetFile()) > 0 {
			return cel.TypeDescs(baseFDS), nil, nil
		}
		return nil, nil, nil
	}

	extendedFDS, warnings, err := tools.ExtendFileDescriptorSet(baseFDS, cfg, nil)
	if err != nil {
		return nil, warnings, err
	}
	if extendedFDS != nil && len(extendedFDS.GetFile()) > 0 {
		return cel.TypeDescs(extendedFDS), warnings, nil
	}
	return nil, warnings, nil
}

// LoadEnvConfig loads an environment Config from a file path or inline JSON string.
func LoadEnvConfig(envPathOrJSON string) (*tools.Config, error) {
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
	fixedEnv   *tools.Config
	opts       []cel.EnvOption
	baseFDS    *descpb.FileDescriptorSet
	useProtoDB bool
}

// NewServer creates a new MCP Server configured for CEL tools.
func NewServer(fixedEnv *tools.Config, opts ...cel.EnvOption) *mcp.Server {
	return NewServerWithOptions(fixedEnv, nil, false, opts...)
}

// NewServerWithOptions creates a new MCP Server configured for CEL tools and proto DB resolution.
func NewServerWithOptions(fixedEnv *tools.Config, baseFDS *descpb.FileDescriptorSet, useProtoDB bool, opts ...cel.EnvOption) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "cel-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		SchemaCache: mcp.NewSchemaCache(),
	})

	h := &toolsHandler{
		fixedEnv:   fixedEnv,
		opts:       opts,
		baseFDS:    baseFDS,
		useProtoDB: useProtoDB,
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
		mcp.AddTool[FixedEnvEvaluateConformanceArgs, any](s, &mcp.Tool{
			Name:        "cel_evaluate_conformance",
			Description: "Evaluates CEL conformance tests from a cel-spec textproto file or inline content. Returns test case results and status.",
		}, h.handleFixedEnvEvaluateConformance)
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
		mcp.AddTool[EvaluateConformanceArgs, any](s, &mcp.Tool{
			Name:        "cel_evaluate_conformance",
			Description: "Evaluates CEL conformance tests from a cel-spec textproto file or inline content. Returns test case results and status.",
		}, h.handleEvaluateConformance)
	}

	return s
}

// CreateEnvConfigArgs is the arguments for the cel_create_environment tool.
type CreateEnvConfigArgs struct {
	EnvConfig *tools.Config `json:"envConfig" jsonschema_description:"The JSON string representing the CEL environment configuration."`
}

// GeneratePromptArgs is the arguments for the cel_generate_prompt tool.
type GeneratePromptArgs struct {
	EnvConfig  *tools.Config `json:"envConfig,omitempty" jsonschema_description:"The JSON string representing the CEL environment schema. May be omitted for standalone expressions which only use the CEL standard library."`
	UserPrompt string        `json:"userPrompt" jsonschema_description:"The user prompt to generate the CEL expression for."`
}

// CompileArgs is the arguments for the cel_compile tool.
type CompileArgs struct {
	EnvConfig *tools.Config `json:"envConfig,omitempty" jsonschema_description:"The JSON string representing the CEL environment schema. May be omitted for standalone expressions which only use the CEL standard library."`
	Expr      string        `json:"expr" jsonschema_description:"The CEL expression to compile."`
}

// EvaluateArgs is the arguments for the cel_evaluate tool.
type EvaluateArgs struct {
	EnvConfig *tools.Config    `json:"envConfig,omitempty" jsonschema_description:"The JSON string representing the CEL environment schema. May be omitted for standalone expressions which only use the CEL standard library."`
	Expr      string           `json:"expr" jsonschema_description:"The CEL expression to evaluate."`
	TestCases []tools.TestCase `json:"testCases" jsonschema_description:"The test cases for evaluation."`
}

// EvaluateConformanceArgs is the arguments for the cel_evaluate_conformance tool.
type EvaluateConformanceArgs struct {
	Tests        string        `json:"tests" jsonschema_description:"Path to conformance textproto file or inline textproto content."`
	Filter       string        `json:"filter,omitempty" jsonschema_description:"Optional substring filter for test names."`
	SkipTests    []string      `json:"skipTests,omitempty" jsonschema_description:"Optional list of test name prefixes to skip."`
	EnvConfig    *tools.Config `json:"envConfig,omitempty" jsonschema_description:"Optional JSON environment configuration."`
	FallbackExpr string        `json:"fallbackExpr,omitempty" jsonschema_description:"Optional fallback CEL expression."`
}

// FixedEnvGeneratePromptArgs is the arguments for the cel_generate_prompt tool when a static environment is configured.
type FixedEnvGeneratePromptArgs struct {
	UserPrompt string `json:"userPrompt" jsonschema_description:"The user prompt to generate the CEL expression for."`
}

// FixedEnvCompileArgs is the arguments for the cel_compile tool when a static environment is configured.
type FixedEnvCompileArgs struct {
	Expr string `json:"expr" jsonschema_description:"The CEL expression to compile."`
}

// FixedEnvEvaluateArgs is the arguments for the cel_evaluate tool when a static environment is configured.
type FixedEnvEvaluateArgs struct {
	Expr      string           `json:"expr" jsonschema_description:"The CEL expression to evaluate."`
	TestCases []tools.TestCase `json:"testCases" jsonschema_description:"The test cases for evaluation."`
}

// FixedEnvEvaluateConformanceArgs is the arguments for the cel_evaluate_conformance tool when a static environment is configured.
type FixedEnvEvaluateConformanceArgs struct {
	Tests        string   `json:"tests" jsonschema_description:"Path to conformance textproto file or inline textproto content."`
	Filter       string   `json:"filter,omitempty" jsonschema_description:"Optional substring filter for test names."`
	SkipTests    []string `json:"skipTests,omitempty" jsonschema_description:"Optional list of test name prefixes to skip."`
	FallbackExpr string   `json:"fallbackExpr,omitempty" jsonschema_description:"Optional fallback CEL expression."`
}

func (h *toolsHandler) resolveEnvOpts(cfg *tools.Config) ([]cel.EnvOption, []string, error) {
	var opts []cel.EnvOption
	opts = append(opts, h.opts...)
	if !h.useProtoDB {
		return opts, nil, nil
	}
	extendedFDS, warnings, err := tools.ExtendFileDescriptorSet(h.baseFDS, cfg, nil)
	if err != nil {
		return nil, warnings, err
	}
	if extendedFDS != nil && len(extendedFDS.GetFile()) > 0 {
		opts = append(opts, cel.TypeDescs(extendedFDS))
	}
	return opts, warnings, nil
}

func (h *toolsHandler) handleCreateEnvConfig(ctx context.Context, request *mcp.CallToolRequest, args CreateEnvConfigArgs) (*mcp.CallToolResult, any, error) {
	opts, warnings, err := h.resolveEnvOpts(args.EnvConfig)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	_, err = tools.EnvFromConfig(args.EnvConfig, opts...)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: "Environment created successfully"}},
	}, nil, nil
}

// resolveEnvConfig returns the environment configuration to use for a request.
//
// Request-supplied configurations take precedence over the server's fixed
// environment. When neither is present an empty configuration is returned so
// that standalone expressions relying only on the CEL standard library may
// still be compiled and evaluated.
func (h *toolsHandler) resolveEnvConfig(envConfig *tools.Config) *tools.Config {
	if envConfig != nil {
		return envConfig
	}
	if h.fixedEnv != nil {
		return h.fixedEnv
	}
	return &tools.Config{}
}

func (h *toolsHandler) handleCompile(ctx context.Context, request *mcp.CallToolRequest, args CompileArgs) (*mcp.CallToolResult, any, error) {
	cfg := h.resolveEnvConfig(args.EnvConfig)
	opts, warnings, err := h.resolveEnvOpts(cfg)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	res, err := tools.CompileCEL(args.Expr, cfg, opts...)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	return nil, res, nil
}

func (h *toolsHandler) handleEvaluate(ctx context.Context, request *mcp.CallToolRequest, args EvaluateArgs) (*mcp.CallToolResult, any, error) {
	cfg := h.resolveEnvConfig(args.EnvConfig)
	opts, warnings, err := h.resolveEnvOpts(cfg)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	res, err := tools.EvaluateCEL(args.Expr, cfg, args.TestCases, opts...)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	return nil, res, nil
}

func (h *toolsHandler) handleGeneratePrompt(ctx context.Context, request *mcp.CallToolRequest, args GeneratePromptArgs) (*mcp.CallToolResult, any, error) {
	cfg := h.resolveEnvConfig(args.EnvConfig)
	opts, warnings, err := h.resolveEnvOpts(cfg)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	res, err := tools.GeneratePrompt(cfg, args.UserPrompt, opts...)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
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

func (h *toolsHandler) handleEvaluateConformance(ctx context.Context, request *mcp.CallToolRequest, args EvaluateConformanceArgs) (*mcp.CallToolResult, any, error) {
	envConfig := args.EnvConfig
	if envConfig == nil {
		envConfig = h.fixedEnv
	}
	opts, warnings, err := h.resolveEnvOpts(envConfig)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	res, err := tools.EvaluateConformanceWithParams(tools.ConformanceParams{
		Tests:        args.Tests,
		Filter:       args.Filter,
		SkipTests:    args.SkipTests,
		EnvConfig:    envConfig,
		FallbackExpr: args.FallbackExpr,
	}, opts...)
	if err != nil {
		return nil, nil, tools.AugmentErrorWithWarnings(err, warnings)
	}
	return nil, res, nil
}

func (h *toolsHandler) handleFixedEnvEvaluateConformance(ctx context.Context, request *mcp.CallToolRequest, args FixedEnvEvaluateConformanceArgs) (*mcp.CallToolResult, any, error) {
	return h.handleEvaluateConformance(ctx, request, EvaluateConformanceArgs{
		Tests:        args.Tests,
		Filter:       args.Filter,
		SkipTests:    args.SkipTests,
		FallbackExpr: args.FallbackExpr,
	})
}
