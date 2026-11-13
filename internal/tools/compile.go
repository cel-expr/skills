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
	"fmt"

	"cel.dev/cel-go/cel"
	celext "cel.dev/cel-go/ext"
)

// CompileExprOutput is the output of a CEL compilation.
type CompileExprOutput struct {
	InputSchema  any      `json:"inputSchema"`
	OutputSchema any      `json:"outputSchema"`
	Warnings     []string `json:"warnings,omitempty"`
}

// CompileOptions configures CEL compilation and evaluation behavior beyond cel.EnvOption.
type CompileOptions struct {
	EnvOpts  []cel.EnvOption
	Policy   bool
	Location string
}

// CompileCEL compiles a CEL expression against the provided environment configuration.
func CompileCEL(expr string, envConfig *Config, opts ...cel.EnvOption) (*CompileExprOutput, error) {
	return CompileCELWithOptions(expr, envConfig, CompileOptions{EnvOpts: opts})
}

// CompileCELWithOptions compiles a CEL expression or .celpolicy with extended options.
func CompileCELWithOptions(expr string, envConfig *Config, opts CompileOptions) (*CompileExprOutput, error) {
	envOpts := opts.EnvOpts
	if opts.Policy {
		// Prepend Bindings (v1+, providing cel.@block) before FromConfig so an unversioned
		// bindings entry in envConfig does not omit cel.@block during policy composition.
		envOpts = append([]cel.EnvOption{celext.Bindings()}, envOpts...)
	}
	env, err := EnvFromConfig(envConfig, envOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed constructing env: %w", err)
	}
	var ast *cel.Ast
	var warnings []string
	if opts.Policy {
		env, ast, warnings, err = compilePolicy(expr, env, opts)
		if err != nil {
			return nil, err
		}
	} else {
		var iss *cel.Issues
		ast, iss = env.Compile(expr)
		if iss.Err() != nil {
			return nil, fmt.Errorf("compile error: %w", iss.Err())
		}
	}
	schema, err := ComputeInputSchema(env, ast)
	if err != nil {
		return nil, fmt.Errorf("failed computing references: %w", err)
	}
	return &CompileExprOutput{
		InputSchema:  schema,
		OutputSchema: SchemaFromCELType(env, ast.OutputType()),
		Warnings:     warnings,
	}, nil
}
