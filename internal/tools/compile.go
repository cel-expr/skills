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
