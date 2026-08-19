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

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	protojson "google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
func EvaluateCEL(expr string, envConfig *Config, testCases []TestCase, opts ...cel.EnvOption) (*EvaluateExprOutput, error) {
	env, err := EnvFromConfig(envConfig, opts...)
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
	provider := env.CELTypeProvider()
	var results []TestResult
	for _, tc := range testCases {
		bindings := make(map[string]any, len(tc.Bindings))
		var bindingErr error
		for k, v := range tc.Bindings {
			if _, isProto := v.(proto.Message); !isProto && v != nil {
				var typeName string
				if envConfig != nil {
					for _, varCfg := range envConfig.Variables {
						if varCfg.Name == k {
							typeName = varCfg.Type
							break
						}
					}
				}
				if typeName != "" {
					if td, err := parseType(typeName); err == nil && td != nil {
						typeName = td.TypeName
					}
				}
				if typeName != "" {
					if _, found := provider.FindStructType(typeName); found {
						emptyVal := provider.NewValue(typeName, map[string]ref.Val{})
						if emptyVal != nil && !types.IsError(emptyVal) {
							if pbMsg, ok := emptyVal.Value().(proto.Message); ok && pbMsg != nil {
								jsonBytes, err := json.Marshal(v)
								if err != nil {
									bindingErr = fmt.Errorf("failed marshaling JSON value for binding %q: %w", k, err)
									break
								}
								msg := pbMsg.ProtoReflect().New().Interface()
								unopts := protojson.UnmarshalOptions{DiscardUnknown: true}
								if err := unopts.Unmarshal(jsonBytes, msg); err != nil {
									bindingErr = fmt.Errorf("failed unmarshaling protobuf input from JSON value for binding %q (type %s): %w", k, typeName, err)
									break
								}
								bindings[k] = msg
								continue
							}
						}
					}
				}
			}
			bindings[k] = v
		}
		if bindingErr != nil {
			results = append(results, TestResult{
				TestCase: tc.TestCase,
				Status:   bindingErr.Error(),
			})
			continue
		}
		out, details, err := prg.Eval(bindings)
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
