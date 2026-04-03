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
)

// CompileExprOutput is the output of a CEL compilation.
type CompileExprOutput struct {
	InputSchema  any `json:"inputSchema"`
	OutputSchema any `json:"outputSchema"`
}

// CompileCEL compiles a CEL expression against the provided JSON environment schema.
func CompileCEL(expr string, envConfig *Config) (*CompileExprOutput, error) {
	env, err := EnvFromConfig(envConfig)
	if err != nil {
		return nil, fmt.Errorf("failed constructing env: %w", err)
	}
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("compile error: %w", iss.Err())
	}
	schema, err := ComputeInputSchema(env, ast)
	if err != nil {
		return nil, fmt.Errorf("failed computing references: %w", err)
	}
	return &CompileExprOutput{
		InputSchema:  schema,
		OutputSchema: SchemaFromCELType(env, ast.OutputType()),
	}, nil
}
