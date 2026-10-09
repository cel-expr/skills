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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/common"
	"cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types"
	"cel.dev/cel-go/common/types/ref"
	"cel.dev/cel-go/ext"
	expr "cel.dev/expr"
	test2pb "cel.dev/expr/conformance/proto2"
	test3pb "cel.dev/expr/conformance/proto3"
	testpb "cel.dev/expr/conformance/test"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// ConformanceTestResult represents the result of evaluating a single conformance test case.
type ConformanceTestResult struct {
	Name        string `json:"name"`
	Status      string `json:"status"` // "pass" or detailed failure message
	Section     string `json:"section,omitempty"`
	File        string `json:"file,omitempty"`
	Expr        string `json:"expr,omitempty"`
	Description string `json:"description,omitempty"`
}

// Passed returns true if the test case passed.
func (r ConformanceTestResult) Passed() bool {
	return r.Status == "pass"
}

// ConformanceOutput represents the aggregated outcome of a conformance test suite evaluation.
type ConformanceOutput struct {
	Name        string                  `json:"name,omitempty"`
	Description string                  `json:"description,omitempty"`
	Total       int                     `json:"total"`
	Passed      int                     `json:"passed"`
	Failed      int                     `json:"failed"`
	Skipped     int                     `json:"skipped"`
	Results     []ConformanceTestResult `json:"results"`
	Coverage    string                  `json:"coverage,omitempty"`
}

// HasFailures returns true if one or more tests failed.
func (o *ConformanceOutput) HasFailures() bool {
	return o.Failed > 0
}

// ConformanceParams configures the conformance test evaluator.
type ConformanceParams struct {
	Tests        string   `json:"tests"`                  // Path to textproto file or inline textproto content
	Filter       string   `json:"filter,omitempty"`       // Optional substring filter for test names
	SkipTests    []string `json:"skipTests,omitempty"`    // Optional test prefixes/names to skip
	EnvConfig    *Config  `json:"envConfig,omitempty"`    // Optional environment configuration
	FallbackExpr string   `json:"fallbackExpr,omitempty"` // Fallback expression if test does not define one
}

// DefaultConformanceSkipTests contains the standard set of conformance test prefixes skipped by cel-go.
// These correspond to tests that require future specification enhancements, pending spec updates,
// or known cel-go platform limitations (derived from cel-expr/cel-spec/tests/simple/BUILD.bazel).
var defaultConformanceSkipTests = []string{
	// Failing conformance tests in cel-go
	"fields/qualified_identifier_resolution/map_key_float",
	"fields/qualified_identifier_resolution/map_key_null",
	"fields/qualified_identifier_resolution/map_value_repeat_key",
	"fields/qualified_identifier_resolution/map_value_repeat_key_heterogeneous",
	"timestamps/duration_converters/get_milliseconds",
	"optionals/optionals/map_optional_select_has",

	// Temporarily failing tests, need a spec update
	"string_ext/value_errors/indexof_out_of_range",
	"string_ext/value_errors/lastindexof_out_of_range",

	// Future enhancements
	"enums/strong_proto2",
	"enums/strong_proto3",

	// Type deductions
	"type_deductions/wrappers/wrapper_promotion_2",
	"type_deductions/legacy_nullable_types/null_assignable_to_message_parameter_candidate",
	"type_deductions/legacy_nullable_types/null_assignable_to_duration_parameter_candidate",
	"type_deductions/legacy_nullable_types/null_assignable_to_timestamp_parameter_candidate",
	"type_deductions/legacy_nullable_types/null_assignable_to_abstract_parameter_candidate",
}

// DefaultConformanceSkipTests returns a copy of the standard set of conformance test prefixes skipped by cel-go.
// These correspond to tests that require future specification enhancements, pending spec updates,
// or known cel-go platform limitations (derived from cel-expr/cel-spec/tests/simple/BUILD.bazel).
func DefaultConformanceSkipTests() []string {
	cp := make([]string, len(defaultConformanceSkipTests))
	copy(cp, defaultConformanceSkipTests)
	return cp
}

// ConformanceEnvOptions returns the standard CEL environment options required for CEL conformance testing.
func ConformanceEnvOptions() []cel.EnvOption {
	return []cel.EnvOption{
		cel.StdLib(),
		cel.ClearMacros(),
		cel.OptionalTypes(),
		cel.EagerlyValidateDeclarations(true),
		cel.EnableErrorOnBadPresenceTest(true),
		cel.Types(
			&test2pb.TestAllTypes{},
			&test2pb.Proto2ExtensionScopedMessage{},
			&test3pb.TestAllTypes{},
		),
		ext.Bindings(),
		ext.Encoders(),
		ext.Lists(),
		ext.Math(),
		ext.Protos(),
		ext.Strings(),
		ext.TwoVarComprehensions(),
		cel.Lib(celBlockLib{}),
		cel.EnableIdentifierEscapeSyntax(),
	}
}

// buildConformanceEnvs constructs the custom CEL environments with and without macros.
func buildConformanceEnvs(cfg *Config, opts ...cel.EnvOption) (*cel.Env, *cel.Env, error) {
	baseOpts := ConformanceEnvOptions()
	if cfg != nil {
		celConfig, err := cfg.ToCELConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed constructing env from config: %w", err)
		}
		baseOpts = append(baseOpts, cel.FromConfig(celConfig, ext.ExtensionOptionFactory))
	}
	baseOpts = append(baseOpts, opts...)

	envNoMacros, err := cel.NewCustomEnv(baseOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed constructing env (no macros): %w", err)
	}
	envWithMacros, err := envNoMacros.Extend(cel.Macros(cel.StandardMacros...))
	if err != nil {
		return nil, nil, fmt.Errorf("failed constructing env (with macros): %w", err)
	}
	return envNoMacros, envWithMacros, nil
}

func formatTestName(file, section, test string) string {
	var parts []string
	if file != "" {
		parts = append(parts, file)
	}
	if section != "" {
		parts = append(parts, section)
	}
	parts = append(parts, test)
	return strings.Join(parts, "/")
}

// EvaluateConformance evaluates CEL conformance tests from a textproto file path or inline textproto string.
func EvaluateConformance(tests string, envConfig *Config, opts ...cel.EnvOption) (*ConformanceOutput, error) {
	return EvaluateConformanceWithParams(ConformanceParams{
		Tests:     tests,
		EnvConfig: envConfig,
	}, opts...)
}

// EvaluateConformanceWithParams evaluates CEL conformance tests with full parameter control.
func EvaluateConformanceWithParams(params ConformanceParams, opts ...cel.EnvOption) (*ConformanceOutput, error) {
	file, err := ParseConformanceFile(params.Tests)
	if err != nil {
		return nil, fmt.Errorf("failed parsing conformance tests: %w", err)
	}

	envNoMacros, envWithMacros, err := buildConformanceEnvs(params.EnvConfig, opts...)
	if err != nil {
		return nil, err
	}

	out := &ConformanceOutput{
		Name:        file.GetName(),
		Description: file.GetDescription(),
	}

	shouldSkip := func(name string) bool {
		for _, s := range params.SkipTests {
			if strings.HasPrefix(name, s) {
				rem := name[len(s):]
				if rem == "" || strings.HasPrefix(rem, "/") {
					return true
				}
			}
		}
		return false
	}

	for _, section := range file.GetSection() {
		for _, test := range section.GetTest() {
			fullName := formatTestName(file.GetName(), section.GetName(), test.GetName())

			if params.Filter != "" && !strings.Contains(fullName, params.Filter) {
				continue
			}

			out.Total++

			if shouldSkip(fullName) {
				out.Skipped++
				out.Results = append(out.Results, ConformanceTestResult{
					Name:        fullName,
					Section:     section.GetName(),
					File:        file.GetName(),
					Expr:        test.GetExpr(),
					Description: test.GetDescription(),
					Status:      "skipped",
				})
				continue
			}

			exprStr := test.GetExpr()
			if exprStr == "" {
				exprStr = params.FallbackExpr
			}

			res := evaluateSingleConformanceTest(test, exprStr, fullName, section.GetName(), file.GetName(), envNoMacros, envWithMacros)
			if res.Passed() {
				out.Passed++
			} else {
				out.Failed++
			}
			out.Results = append(out.Results, res)
		}
	}

	return out, nil
}

// EvaluateConformanceTest evaluates an individual SimpleTest protobuf message.
func EvaluateConformanceTest(test *testpb.SimpleTest, envConfig *Config, opts ...cel.EnvOption) (*ConformanceTestResult, error) {
	if test == nil {
		return nil, errors.New("nil SimpleTest")
	}
	envNoMacros, envWithMacros, err := buildConformanceEnvs(envConfig, opts...)
	if err != nil {
		return nil, err
	}

	res := evaluateSingleConformanceTest(test, test.GetExpr(), test.GetName(), "", "", envNoMacros, envWithMacros)
	return &res, nil
}

func checkDeducedType(want *expr.Type, outputType *cel.Type) string {
	outType, err := types.TypeToProto(outputType)
	if err != nil {
		return fmt.Sprintf("failed converting output type: %v", err)
	}
	if !equalTypes(want, outType) {
		return fmt.Sprintf("type mismatch: want %v, got %v", want, outType)
	}
	return ""
}

func evaluateSingleConformanceTest(test *testpb.SimpleTest, exprStr, fullName, sectionName, fileName string, envNoMacros, envWithMacros *cel.Env) ConformanceTestResult {
	res := ConformanceTestResult{
		Name:        fullName,
		Section:     sectionName,
		File:        fileName,
		Expr:        exprStr,
		Description: test.GetDescription(),
	}

	var baseEnv *cel.Env
	if test.GetDisableMacros() {
		baseEnv = envNoMacros
	} else {
		baseEnv = envWithMacros
	}

	src := common.NewStringSource(exprStr, fullName)
	ast, iss := baseEnv.ParseSource(src)
	if iss.Err() != nil {
		res.Status = fmt.Sprintf("parse error: %v", iss.Err())
		return res
	}

	var testOpts []cel.EnvOption
	if test.GetContainer() != "" {
		testOpts = append(testOpts, cel.Container(test.GetContainer()))
	}
	for _, d := range test.GetTypeEnv() {
		opt, err := cel.ProtoAsDeclaration(d)
		if err != nil {
			res.Status = fmt.Sprintf("proto declaration error: %v", err)
			return res
		}
		testOpts = append(testOpts, opt)
	}

	testEnv := baseEnv
	if len(testOpts) > 0 {
		var err error
		testEnv, err = baseEnv.Extend(testOpts...)
		if err != nil {
			res.Status = fmt.Sprintf("env extension error: %v", err)
			return res
		}
	}

	if !test.GetDisableCheck() {
		ast, iss = testEnv.Check(ast)
		if iss.Err() != nil {
			res.Status = fmt.Sprintf("check error: %v", iss.Err())
			return res
		}
	}

	// check_only tests verify the deduced output type without evaluation.
	if test.GetCheckOnly() {
		m, ok := test.GetResultMatcher().(*testpb.SimpleTest_TypedResult)
		if !ok || m.TypedResult == nil {
			res.Status = fmt.Sprintf("unexpected matcher kind for check-only test: %T", test.GetResultMatcher())
			return res
		}
		if status := checkDeducedType(m.TypedResult.GetDeducedType(), ast.OutputType()); status != "" {
			res.Status = status
			return res
		}
		res.Status = "pass"
		return res
	}

	prg, err := testEnv.Program(ast)
	if err != nil {
		res.Status = fmt.Sprintf("program creation error: %v", err)
		return res
	}

	bindings := make(map[string]any, len(test.GetBindings()))
	for k, v := range test.GetBindings() {
		val, err := exprValueToRefValue(testEnv.CELTypeAdapter(), v)
		if err != nil {
			res.Status = fmt.Sprintf("binding conversion error for %q: %v", k, err)
			return res
		}
		bindings[k] = val
	}

	ret, _, evalErr := prg.Eval(bindings)
	if evalErr == nil && ret != nil && types.IsError(ret) {
		if errVal, ok := ret.(*types.Err); ok {
			evalErr = errVal.Unwrap()
		} else {
			evalErr = fmt.Errorf("%v", ret)
		}
	}

	// Default result matcher is boolean true.
	matcher := test.GetResultMatcher()
	if matcher == nil {
		matcher = &testpb.SimpleTest_Value{
			Value: &expr.Value{
				Kind: &expr.Value_BoolValue{BoolValue: true},
			},
		}
	}

	switch m := matcher.(type) {
	case *testpb.SimpleTest_Value:
		if evalErr != nil {
			res.Status = fmt.Sprintf("unexpected eval error: %v", evalErr)
			return res
		}
		val, err := refValueToExprValue(ret)
		if err != nil {
			res.Status = fmt.Sprintf("failed converting result to value proto: %v", err)
			return res
		}
		if !equalValues(m.Value, val.GetValue()) {
			res.Status = fmt.Sprintf("value mismatch: want %v, got %v", m.Value, val.GetValue())
			return res
		}
		res.Status = "pass"
		return res

	case *testpb.SimpleTest_TypedResult:
		if evalErr != nil {
			res.Status = fmt.Sprintf("unexpected eval error: %v", evalErr)
			return res
		}
		val, err := refValueToExprValue(ret)
		if err != nil {
			res.Status = fmt.Sprintf("failed converting result to value proto: %v", err)
			return res
		}
		if m.TypedResult != nil {
			if m.TypedResult.GetResult() != nil && !equalValues(m.TypedResult.GetResult(), val.GetValue()) {
				res.Status = fmt.Sprintf("value mismatch: want %v, got %v", m.TypedResult.GetResult(), val.GetValue())
				return res
			}
			if m.TypedResult.GetDeducedType() != nil {
				if status := checkDeducedType(m.TypedResult.GetDeducedType(), ast.OutputType()); status != "" {
					res.Status = status
					return res
				}
			}
		}
		res.Status = "pass"
		return res

	case *testpb.SimpleTest_EvalError, *testpb.SimpleTest_AnyEvalErrors:
		if evalErr == nil {
			res.Status = fmt.Sprintf("expected eval error, got %v", ret)
			return res
		}
		res.Status = "pass"
		return res

	case *testpb.SimpleTest_Unknown, *testpb.SimpleTest_AnyUnknowns:
		if ret == nil || !types.IsUnknown(ret) {
			res.Status = fmt.Sprintf("expected unknown, got %v", ret)
			return res
		}
		res.Status = "pass"
		return res

	default:
		res.Status = fmt.Sprintf("unsupported result matcher: %T", matcher)
		return res
	}
}

// ParseConformanceFile parses a conformance test from a file path or direct content.
// It first attempts to read from disk as a file; if successful, it unmarshals from textproto
// or protojson into a SimpleTestFile (or wrapped SimpleTest). Otherwise, it treats the content
// itself as a test case parsable from textproto or protojson.
func ParseConformanceFile(pathOrContent string) (*testpb.SimpleTestFile, error) {
	trimmed := strings.TrimSpace(pathOrContent)
	if len(trimmed) == 0 {
		return nil, errors.New("empty conformance test content")
	}

	isExplicitPath := strings.HasPrefix(trimmed, "/") ||
		strings.HasPrefix(trimmed, "./") ||
		strings.HasPrefix(trimmed, "../") ||
		strings.HasSuffix(trimmed, ".textproto") ||
		strings.HasSuffix(trimmed, ".proto") ||
		strings.HasSuffix(trimmed, ".json") ||
		strings.HasSuffix(trimmed, ".pb")

	if fi, err := os.Stat(pathOrContent); err == nil {
		if fi.IsDir() {
			return nil, fmt.Errorf("conformance path %q is a directory, expected a file", pathOrContent)
		}
		fileBytes, err := os.ReadFile(pathOrContent)
		if err != nil {
			return nil, fmt.Errorf("failed reading conformance test file %q: %w", pathOrContent, err)
		}
		return parseConformanceBytes(bytes.TrimSpace(fileBytes))
	} else if isExplicitPath {
		return nil, fmt.Errorf("failed reading conformance test file %q: %w", pathOrContent, err)
	}

	// Otherwise, treat content itself as parsable from textproto or protojson.
	return parseConformanceBytes([]byte(trimmed))
}

func parseConformanceBytes(data []byte) (*testpb.SimpleTestFile, error) {
	if len(data) == 0 {
		return nil, errors.New("empty conformance test content")
	}

	type unmarshalFunc func([]byte, proto.Message) error
	var formats []unmarshalFunc
	if bytes.HasPrefix(data, []byte("{")) {
		formats = []unmarshalFunc{protojson.Unmarshal, prototext.Unmarshal}
	} else {
		formats = []unmarshalFunc{prototext.Unmarshal, protojson.Unmarshal}
	}

	var lastErr error
	for _, unmarshal := range formats {
		// 1. Try SimpleTestFile
		file := &testpb.SimpleTestFile{}
		if err := unmarshal(data, file); err == nil && (len(file.GetSection()) > 0 || file.GetName() != "") {
			return file, nil
		} else if err != nil {
			lastErr = err
		}

		// 2. Try standalone SimpleTest
		singleTest := &testpb.SimpleTest{}
		if err := unmarshal(data, singleTest); err == nil && (singleTest.GetExpr() != "" || singleTest.GetName() != "") {
			return wrapSingleTest(singleTest), nil
		} else if err != nil {
			lastErr = err
		}

		// 3. Try SimpleTestSection
		section := &testpb.SimpleTestSection{}
		if err := unmarshal(data, section); err == nil && len(section.GetTest()) > 0 {
			return wrapSection(section), nil
		} else if err != nil {
			lastErr = err
		}
	}

	sample := string(data)
	if len(sample) > 200 {
		sample = sample[:200] + "... (truncated)"
	}
	if lastErr != nil {
		return nil, fmt.Errorf("unable to parse conformance test: %w (input sample: %q)", lastErr, sample)
	}
	return nil, fmt.Errorf("unable to parse conformance test (input sample: %q)", sample)
}

func wrapSingleTest(st *testpb.SimpleTest) *testpb.SimpleTestFile {
	name := st.GetName()
	if name == "" {
		name = "standalone"
	}
	return &testpb.SimpleTestFile{
		Name: name,
		Section: []*testpb.SimpleTestSection{
			{
				Name: "default",
				Test: []*testpb.SimpleTest{st},
			},
		},
	}
}

func wrapSection(sec *testpb.SimpleTestSection) *testpb.SimpleTestFile {
	return &testpb.SimpleTestFile{
		Name:        sec.GetName(),
		Description: sec.GetDescription(),
		Section:     []*testpb.SimpleTestSection{sec},
	}
}

// SimpleTestToTestCase converts a SimpleTest to the unit test TestCase format.
func SimpleTestToTestCase(st *testpb.SimpleTest) (*TestCase, error) {
	if st == nil {
		return nil, errors.New("nil SimpleTest")
	}
	bindings := make(map[string]any, len(st.GetBindings()))
	for k, v := range st.GetBindings() {
		if v.GetValue() != nil {
			jsVal, err := valueToJSON(v.GetValue())
			if err != nil {
				return nil, fmt.Errorf("failed converting binding %q: %w", k, err)
			}
			bindings[k] = jsVal
		}
	}

	var expected any
	if v := st.GetValue(); v != nil {
		var err error
		expected, err = valueToJSON(v)
		if err != nil {
			return nil, fmt.Errorf("failed converting expected value: %w", err)
		}
	} else if tr := st.GetTypedResult(); tr != nil && tr.GetResult() != nil {
		var err error
		expected, err = valueToJSON(tr.GetResult())
		if err != nil {
			return nil, fmt.Errorf("failed converting expected value: %w", err)
		}
	}

	return &TestCase{
		TestCase: st.GetName(),
		Bindings: bindings,
		Expected: expected,
	}, nil
}

// TestCaseToSimpleTest converts a unit test TestCase into a SimpleTest.
func TestCaseToSimpleTest(tc TestCase, exprStr string) (*testpb.SimpleTest, error) {
	st := &testpb.SimpleTest{
		Name:     tc.TestCase,
		Expr:     exprStr,
		Bindings: make(map[string]*expr.ExprValue, len(tc.Bindings)),
	}

	for k, v := range tc.Bindings {
		pbVal, err := jsonToValue(v)
		if err != nil {
			return nil, fmt.Errorf("failed converting binding %q: %w", k, err)
		}
		st.Bindings[k] = &expr.ExprValue{
			Kind: &expr.ExprValue_Value{Value: pbVal},
		}
	}

	if tc.Expected != nil {
		pbVal, err := jsonToValue(tc.Expected)
		if err != nil {
			return nil, fmt.Errorf("failed converting expected value: %w", err)
		}
		st.ResultMatcher = &testpb.SimpleTest_Value{Value: pbVal}
	}

	return st, nil
}

// FormatSimpleTestFile formats a SimpleTestFile as textproto.
func FormatSimpleTestFile(file *testpb.SimpleTestFile) (string, error) {
	if file == nil {
		return "", errors.New("nil SimpleTestFile")
	}
	opts := prototext.MarshalOptions{Multiline: true, Indent: "  "}
	b, err := opts.Marshal(file)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FormatSimpleTest formats a SimpleTest as textproto.
func FormatSimpleTest(test *testpb.SimpleTest) (string, error) {
	if test == nil {
		return "", errors.New("nil SimpleTest")
	}
	opts := prototext.MarshalOptions{Multiline: true, Indent: "  "}
	b, err := opts.Marshal(test)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// refValueToExprValue converts a CEL ref.Val result into an ExprValue protobuf message.
// Derived from cel-expr/cel-go/conformance/conformance_test.go (refValueToExprValue).
func refValueToExprValue(res ref.Val) (*expr.ExprValue, error) {
	if types.IsUnknown(res) {
		return &expr.ExprValue{
			Kind: &expr.ExprValue_Unknown{
				Unknown: &expr.UnknownSet{
					Exprs: res.Value().([]int64),
				},
			},
		}, nil
	}
	v, err := cel.ValueAsProto(res)
	if err != nil {
		return nil, err
	}
	return &expr.ExprValue{
		Kind: &expr.ExprValue_Value{Value: v},
	}, nil
}

// exprValueToRefValue converts an ExprValue protobuf message to a CEL ref.Val.
// Derived from cel-expr/cel-go/conformance/conformance_test.go (exprValueToRefValue).
func exprValueToRefValue(adapter types.Adapter, ev *expr.ExprValue) (ref.Val, error) {
	if ev == nil {
		return nil, nil
	}
	switch k := ev.Kind.(type) {
	case *expr.ExprValue_Value:
		return cel.ProtoAsValue(adapter, k.Value)
	case *expr.ExprValue_Error:
		return types.NewErr("evaluation error"), nil
	case *expr.ExprValue_Unknown:
		var unk *types.Unknown
		for _, id := range k.Unknown.GetExprs() {
			if unk == nil {
				unk = types.NewUnknown(id, nil)
			} else {
				unk = types.MergeUnknowns(types.NewUnknown(id, nil), unk)
			}
		}
		return unk, nil
	default:
		return nil, fmt.Errorf("unknown ExprValue kind: %T", ev.Kind)
	}
}

// equalValues compares expected and actual expr.Value protobuf messages for semantic equality.
// Handles IEEE-754 NaN semantics, list order preservation, and order-insensitive map entry comparison.
// Derived from cel-expr/cel-go/conformance/conformance_test.go (diffValue / equalValues).
func equalValues(want, got *expr.Value) bool {
	if want == nil || got == nil {
		return want == got
	}
	switch w := want.Kind.(type) {
	case *expr.Value_NullValue:
		g, ok := got.Kind.(*expr.Value_NullValue)
		return ok && w.NullValue == g.NullValue
	case *expr.Value_BoolValue:
		g, ok := got.Kind.(*expr.Value_BoolValue)
		return ok && w.BoolValue == g.BoolValue
	case *expr.Value_Int64Value:
		g, ok := got.Kind.(*expr.Value_Int64Value)
		return ok && w.Int64Value == g.Int64Value
	case *expr.Value_Uint64Value:
		g, ok := got.Kind.(*expr.Value_Uint64Value)
		return ok && w.Uint64Value == g.Uint64Value
	case *expr.Value_DoubleValue:
		g, ok := got.Kind.(*expr.Value_DoubleValue)
		if !ok {
			return false
		}
		if math.IsNaN(w.DoubleValue) && math.IsNaN(g.DoubleValue) {
			return true
		}
		return w.DoubleValue == g.DoubleValue
	case *expr.Value_StringValue:
		g, ok := got.Kind.(*expr.Value_StringValue)
		return ok && w.StringValue == g.StringValue
	case *expr.Value_BytesValue:
		g, ok := got.Kind.(*expr.Value_BytesValue)
		return ok && bytes.Equal(w.BytesValue, g.BytesValue)
	case *expr.Value_TypeValue:
		g, ok := got.Kind.(*expr.Value_TypeValue)
		return ok && w.TypeValue == g.TypeValue
	case *expr.Value_EnumValue:
		g, ok := got.Kind.(*expr.Value_EnumValue)
		return ok && proto.Equal(w.EnumValue, g.EnumValue)
	case *expr.Value_ObjectValue:
		g, ok := got.Kind.(*expr.Value_ObjectValue)
		if !ok {
			return false
		}
		if proto.Equal(w.ObjectValue, g.ObjectValue) {
			return true
		}
		if w.ObjectValue == nil || g.ObjectValue == nil {
			return w.ObjectValue == g.ObjectValue
		}
		wMsg, wErr := w.ObjectValue.UnmarshalNew()
		gMsg, gErr := g.ObjectValue.UnmarshalNew()
		if wErr == nil && gErr == nil {
			return proto.Equal(wMsg, gMsg)
		}
		return false
	case *expr.Value_ListValue:
		g, ok := got.Kind.(*expr.Value_ListValue)
		if !ok {
			return false
		}
		wVals := w.ListValue.GetValues()
		gVals := g.ListValue.GetValues()
		if len(wVals) != len(gVals) {
			return false
		}
		for i, elem := range wVals {
			if !equalValues(elem, gVals[i]) {
				return false
			}
		}
		return true
	case *expr.Value_MapValue:
		g, ok := got.Kind.(*expr.Value_MapValue)
		if !ok {
			return false
		}
		wEntries := w.MapValue.GetEntries()
		gEntries := g.MapValue.GetEntries()
		if len(wEntries) != len(gEntries) {
			return false
		}
		matched := make([]bool, len(gEntries))
		for _, we := range wEntries {
			found := false
			for j, ge := range gEntries {
				if !matched[j] && equalValues(we.GetKey(), ge.GetKey()) && equalValues(we.GetValue(), ge.GetValue()) {
					matched[j] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	return proto.Equal(want, got)
}

// equalTypes compares expected and actual expr.Type protobuf messages for equality.
// Derived from cel-expr/cel-go/conformance/conformance_test.go (diffType).
func equalTypes(want, got *expr.Type) bool {
	return proto.Equal(want, got)
}

func valueToJSON(v *expr.Value) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch k := v.Kind.(type) {
	case *expr.Value_NullValue:
		return nil, nil
	case *expr.Value_BoolValue:
		return k.BoolValue, nil
	case *expr.Value_Int64Value:
		return k.Int64Value, nil
	case *expr.Value_Uint64Value:
		return k.Uint64Value, nil
	case *expr.Value_DoubleValue:
		return k.DoubleValue, nil
	case *expr.Value_StringValue:
		return k.StringValue, nil
	case *expr.Value_BytesValue:
		return k.BytesValue, nil
	case *expr.Value_TypeValue:
		return k.TypeValue, nil
	case *expr.Value_ListValue:
		res := make([]any, 0, len(k.ListValue.GetValues()))
		for _, elem := range k.ListValue.GetValues() {
			jv, err := valueToJSON(elem)
			if err != nil {
				return nil, err
			}
			res = append(res, jv)
		}
		return res, nil
	case *expr.Value_MapValue:
		entries := k.MapValue.GetEntries()
		res := make(map[string]any, len(entries))
		for _, entry := range entries {
			var keyStr string
			if keyVal := entry.GetKey(); keyVal != nil {
				if keyVal.GetStringValue() != "" {
					keyStr = keyVal.GetStringValue()
				} else {
					kJSON, err := valueToJSON(keyVal)
					if err == nil && kJSON != nil {
						keyStr = fmt.Sprintf("%v", kJSON)
					} else {
						keyStr = fmt.Sprintf("%v", keyVal)
					}
				}
			}
			jv, err := valueToJSON(entry.GetValue())
			if err != nil {
				return nil, err
			}
			res[keyStr] = jv
		}
		return res, nil
	case *expr.Value_EnumValue:
		if k.EnumValue == nil {
			return nil, nil
		}
		return k.EnumValue.GetValue(), nil
	case *expr.Value_ObjectValue:
		if k.ObjectValue == nil {
			return nil, nil
		}
		msg, err := k.ObjectValue.UnmarshalNew()
		if err != nil {
			return nil, fmt.Errorf("failed unmarshaling ObjectValue: %w", err)
		}
		jsonBytes, err := protojson.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("failed marshaling ObjectValue to JSON: %w", err)
		}
		var jsVal any
		if err := json.Unmarshal(jsonBytes, &jsVal); err != nil {
			return nil, fmt.Errorf("failed unmarshaling ObjectValue JSON: %w", err)
		}
		return jsVal, nil
	default:
		return nil, fmt.Errorf("unsupported value type: %T", v.Kind)
	}
}

func jsonToValue(val any) (*expr.Value, error) {
	if val == nil {
		return &expr.Value{Kind: &expr.Value_NullValue{}}, nil
	}
	switch v := val.(type) {
	case bool:
		return &expr.Value{Kind: &expr.Value_BoolValue{BoolValue: v}}, nil
	case int:
		return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: int64(v)}}, nil
	case int64:
		return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: v}}, nil
	case uint:
		return &expr.Value{Kind: &expr.Value_Uint64Value{Uint64Value: uint64(v)}}, nil
	case uint64:
		return &expr.Value{Kind: &expr.Value_Uint64Value{Uint64Value: v}}, nil
	case float64:
		if math.Floor(v) == v && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= math.MinInt64 && v <= math.MaxInt64 {
			// Integer value represented in JSON float
			return &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: int64(v)}}, nil
		}
		return &expr.Value{Kind: &expr.Value_DoubleValue{DoubleValue: v}}, nil
	case string:
		return &expr.Value{Kind: &expr.Value_StringValue{StringValue: v}}, nil
	case []byte:
		return &expr.Value{Kind: &expr.Value_BytesValue{BytesValue: v}}, nil
	case []any:
		elements := make([]*expr.Value, 0, len(v))
		for _, elem := range v {
			pbVal, err := jsonToValue(elem)
			if err != nil {
				return nil, err
			}
			elements = append(elements, pbVal)
		}
		return &expr.Value{Kind: &expr.Value_ListValue{ListValue: &expr.ListValue{Values: elements}}}, nil
	case map[string]any:
		entries := make([]*expr.MapValue_Entry, 0, len(v))
		for k, elem := range v {
			pbKey := &expr.Value{Kind: &expr.Value_StringValue{StringValue: k}}
			pbVal, err := jsonToValue(elem)
			if err != nil {
				return nil, err
			}
			entries = append(entries, &expr.MapValue_Entry{Key: pbKey, Value: pbVal})
		}
		return &expr.Value{Kind: &expr.Value_MapValue{MapValue: &expr.MapValue{Entries: entries}}}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON value type: %T", val)
	}
}

// celBlockLib simulates indexed arguments and test-only macros for cel.block conformance tests.
// Derived from cel-expr/cel-go/conformance/conformance_test.go (celBlockLib, celBlock, celIndex, celCompreVar).
type celBlockLib struct{}

func (celBlockLib) LibraryName() string {
	return "cel.lib.ext.cel.block.conformance"
}

func (celBlockLib) CompileOptions() []cel.EnvOption {
	maxIndices := 30
	indexOpts := make([]cel.EnvOption, maxIndices)
	for i := 0; i < maxIndices; i++ {
		indexOpts[i] = cel.Variable(fmt.Sprintf("@index%d", i), cel.DynType)
	}
	return append([]cel.EnvOption{
		cel.Macros(
			// cel.block([args], expr)
			cel.ReceiverMacro("block", 2, celBlock),
			// cel.index(int)
			cel.ReceiverMacro("index", 1, celIndex),
			// cel.iterVar(int, int)
			cel.ReceiverMacro("iterVar", 2, celCompreVar("cel.iterVar", "@it")),
			// cel.accuVar(int, int)
			cel.ReceiverMacro("accuVar", 2, celCompreVar("cel.accuVar", "@ac")),
		),
	}, indexOpts...)
}

func (celBlockLib) ProgramOptions() []cel.ProgramOption {
	return []cel.ProgramOption{}
}

func celBlock(mef cel.MacroExprFactory, target ast.Expr, args []ast.Expr) (ast.Expr, *cel.Error) {
	if !isCELNamespace(target) {
		return nil, nil
	}
	bindings := args[0]
	if bindings.Kind() != ast.ListKind {
		return bindings, mef.NewError(bindings.ID(), "cel.block requires the first arg to be a list literal")
	}
	return mef.NewCall("cel.@block", args...), nil
}

func celIndex(mef cel.MacroExprFactory, target ast.Expr, args []ast.Expr) (ast.Expr, *cel.Error) {
	if !isCELNamespace(target) {
		return nil, nil
	}
	index := args[0]
	if !isNonNegativeInt(index) {
		return index, mef.NewError(index.ID(), "cel.index requires a single non-negative int constant arg")
	}
	indexVal := index.AsLiteral().(types.Int)
	return mef.NewIdent(fmt.Sprintf("@index%d", indexVal)), nil
}

func celCompreVar(funcName, varPrefix string) cel.MacroFactory {
	return func(mef cel.MacroExprFactory, target ast.Expr, args []ast.Expr) (ast.Expr, *cel.Error) {
		if !isCELNamespace(target) {
			return nil, nil
		}
		depth := args[0]
		if !isNonNegativeInt(depth) {
			return depth, mef.NewError(depth.ID(), fmt.Sprintf("%s requires two non-negative int constant args", funcName))
		}
		unique := args[1]
		if !isNonNegativeInt(unique) {
			return unique, mef.NewError(unique.ID(), fmt.Sprintf("%s requires two non-negative int constant args", funcName))
		}
		depthVal := depth.AsLiteral().(types.Int)
		uniqueVal := unique.AsLiteral().(types.Int)
		return mef.NewIdent(fmt.Sprintf("%s:%d:%d", varPrefix, depthVal, uniqueVal)), nil
	}
}

func isCELNamespace(target ast.Expr) bool {
	return target.Kind() == ast.IdentKind && target.AsIdent() == "cel"
}

func isNonNegativeInt(expr ast.Expr) bool {
	if expr.Kind() != ast.LiteralKind {
		return false
	}
	val := expr.AsLiteral()
	return val.Type() == cel.IntType && val.(types.Int) >= 0
}
