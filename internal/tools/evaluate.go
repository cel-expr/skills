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
	"fmt"
	"reflect"

	protojson "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
)

// TestResult is the results of an evaluation.
type TestResult struct {
	TestCase string `json:"testCase"`
	Status   string `json:"status"`
}

// EvaluateExprOutput is the output of the EvaluateCEL function.
type EvaluateExprOutput struct {
	TestResults []TestResult `json:"testResults"`
	Coverage    string       `json:"coverage"`
}

// TestCase is a test case for evaluation.
type TestCase struct {
	TestCase string         `json:"testCase" jsonschema_description:"The name of the test case."`
	Bindings map[string]any `json:"bindings" jsonschema_description:"The variable bindings for the expression."`
	Expected any            `json:"expected" jsonschema_description:"The expected JSON output value of the expression."`
}

// EvaluateCEL evaluates a compiled CEL expression against provided variable bindings.
func EvaluateCEL(expr string, envConfig *Config, testCases []TestCase) (*EvaluateExprOutput, error) {
	env, err := EnvFromConfig(envConfig)
	if err != nil {
		return nil, fmt.Errorf("failed constructing env: %w", err)
	}

	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, fmt.Errorf("compile error: %w", iss.Err())
	}

	prg, err := env.Program(ast, cel.EvalOptions(cel.OptTrackState))
	if err != nil {
		return nil, fmt.Errorf("program creation error: %w", err)
	}
	coverageTracker := NewCoverageTracker(ast)
	var results []TestResult
	for _, tc := range testCases {
		out, details, err := prg.Eval(tc.Bindings)
		status := "undefined"
		if err != nil {
			results = append(results, TestResult{
				TestCase: tc.TestCase,
				Status:   err.Error(),
			})
			continue
		}
		if out != nil {
			coverageTracker.Record(details)
			val, err := out.ConvertToNative(types.JSONValueType)
			if err != nil {
				status = fmt.Sprintf("unexpected output type: %v", out.Value())
				results = append(results, TestResult{
					TestCase: tc.TestCase,
					Status:   status,
				})
				continue
			}
			valPB := protojson.Format(val.(proto.Message))
			var valJSON any
			err = json.Unmarshal([]byte(valPB), &valJSON)
			if err != nil {
				status = fmt.Sprintf("unexpected output type: %v", out.Value())
				results = append(results, TestResult{
					TestCase: tc.TestCase,
					Status:   status,
				})
				continue
			}
			eq := reflect.DeepEqual(valJSON, tc.Expected)
			if eq {
				status = "pass"
			} else {
				status = fmt.Sprintf("failed: got %v, expected %v", valJSON, tc.Expected)
			}
			results = append(results, TestResult{
				TestCase: tc.TestCase,
				Status:   status,
			})
		}
	}

	report := coverageTracker.GenerateReport()
	outputSchema := EvaluateExprOutput{
		TestResults: results,
		Coverage:    fmt.Sprintf("Node: %.2f%%, Branch: %.2f%%", report.NodeCoverage(), report.BranchCoverage()),
	}

	return &outputSchema, nil
}
