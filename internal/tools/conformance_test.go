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
	"os"
	"path/filepath"
	"strings"
	"testing"

	expr "cel.dev/expr"
	testpb "cel.dev/expr/conformance/test"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const sampleTextproto = `
name: "sample_conformance"
description: "Sample test suite for CEL conformance evaluation"
section {
  name: "primitives"
  description: "Basic literal values"
  test {
    name: "int_zero"
    expr: "0"
    value { int64_value: 0 }
  }
  test {
    name: "uint_val"
    expr: "42u"
    value { uint64_value: 42 }
  }
  test {
    name: "float_val"
    expr: "3.14"
    value { double_value: 3.14 }
  }
  test {
    name: "string_val"
    expr: "'hello world'"
    value { string_value: "hello world" }
  }
  test {
    name: "bool_val"
    expr: "true"
    value { bool_value: true }
  }
  test {
    name: "default_bool_true"
    expr: "1 < 2"
  }
}
section {
  name: "bindings_and_env"
  test {
    name: "type_env_and_binding"
    expr: "x + y"
    type_env {
      name: "x"
      ident {
        type { primitive: INT64 }
      }
    }
    type_env {
      name: "y"
      ident {
        type { primitive: INT64 }
      }
    }
    bindings {
      key: "x"
      value {
        value { int64_value: 10 }
      }
    }
    bindings {
      key: "y"
      value {
        value { int64_value: 20 }
      }
    }
    value { int64_value: 30 }
  }
}
section {
  name: "errors_and_checks"
  test {
    name: "runtime_error"
    expr: "1 / 0"
    eval_error {
      errors { message: "division by zero" }
    }
  }
  test {
    name: "unchecked_unbound"
    expr: "unbound_var"
    disable_check: true
    eval_error {}
  }
  test {
    name: "check_only_deduced_type"
    expr: "size('test')"
    check_only: true
    typed_result {
      deduced_type {
        primitive: INT64
      }
    }
  }
}
section {
  name: "block_macros"
  test {
    name: "cel_block_test"
    expr: "cel.block([1, cel.index(0) + 2, cel.index(1) + 3], cel.index(2))"
    value { int64_value: 6 }
  }
}
`

func TestEvaluateConformance(t *testing.T) {
	out, err := EvaluateConformance(sampleTextproto, nil)
	if err != nil {
		t.Fatalf("EvaluateConformance failed: %v", err)
	}

	if out.Name != "sample_conformance" {
		t.Errorf("expected name sample_conformance, got %s", out.Name)
	}
	if out.Failed > 0 {
		for _, r := range out.Results {
			if !r.Passed() {
				t.Errorf("test %s failed: %s", r.Name, r.Status)
			}
		}
	}
	if out.Passed != out.Total {
		t.Errorf("expected all %d tests to pass, but passed: %d, failed: %d, skipped: %d", out.Total, out.Passed, out.Failed, out.Skipped)
	}
}

func TestEvaluateConformanceParams(t *testing.T) {
	// Test filtering
	out, err := EvaluateConformanceWithParams(ConformanceParams{
		Tests:  sampleTextproto,
		Filter: "primitives",
	})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if out.Total != 6 {
		t.Errorf("expected 6 filtered tests, got %d", out.Total)
	}

	// Test skip
	out, err = EvaluateConformanceWithParams(ConformanceParams{
		Tests:     sampleTextproto,
		SkipTests: []string{"sample_conformance/block_macros"},
	})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if out.Skipped != 1 {
		t.Errorf("expected 1 skipped test, got %d", out.Skipped)
	}
}

func TestEvaluateConformanceSingleTest(t *testing.T) {
	single := `
name: "single_test"
expr: "2 * 3"
value { int64_value: 6 }
`
	out, err := EvaluateConformance(single, nil)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if out.Total != 1 || out.Passed != 1 {
		t.Errorf("expected 1 passed test, got total=%d passed=%d", out.Total, out.Passed)
	}
}

func TestEvaluateConformanceSection(t *testing.T) {
	section := `
name: "section_only"
test {
  name: "t1"
  expr: "true && false"
  value { bool_value: false }
}
test {
  name: "t2"
  expr: "true || false"
  value { bool_value: true }
}
`
	out, err := EvaluateConformance(section, nil)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if out.Total != 2 || out.Passed != 2 {
		t.Errorf("expected 2 passed tests, got total=%d passed=%d", out.Total, out.Passed)
	}
}

func TestEvaluateConformanceTestFunc(t *testing.T) {
	st := &testpb.SimpleTest{
		Name: "direct_test",
		Expr: "'abc' + 'def'",
		ResultMatcher: &testpb.SimpleTest_Value{
			Value: &expr.Value{
				Kind: &expr.Value_StringValue{StringValue: "abcdef"},
			},
		},
	}
	res, err := EvaluateConformanceTest(st, nil)
	if err != nil {
		t.Fatalf("EvaluateConformanceTest failed: %v", err)
	}
	if !res.Passed() {
		t.Errorf("expected pass, got %s", res.Status)
	}
}

func TestSimpleTestConversion(t *testing.T) {
	tc := TestCase{
		TestCase: "tc_conversion",
		Bindings: map[string]any{
			"str": "hello",
			"num": 42,
		},
		Expected: true,
	}

	st, err := TestCaseToSimpleTest(tc, "str == 'hello' && num == 42")
	if err != nil {
		t.Fatalf("TestCaseToSimpleTest failed: %v", err)
	}

	if st.GetName() != "tc_conversion" {
		t.Errorf("expected name tc_conversion, got %s", st.GetName())
	}
	if len(st.GetBindings()) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(st.GetBindings()))
	}

	convertedBack, err := SimpleTestToTestCase(st)
	if err != nil {
		t.Fatalf("SimpleTestToTestCase failed: %v", err)
	}
	if convertedBack.TestCase != tc.TestCase {
		t.Errorf("expected %s, got %s", tc.TestCase, convertedBack.TestCase)
	}

	text, err := FormatSimpleTest(st)
	if err != nil {
		t.Fatalf("FormatSimpleTest failed: %v", err)
	}
	if len(text) == 0 {
		t.Error("expected non-empty formatted textproto")
	}
}

func TestEqualValues(t *testing.T) {
	// List equality
	v1 := &expr.Value{
		Kind: &expr.Value_ListValue{
			ListValue: &expr.ListValue{
				Values: []*expr.Value{
					{Kind: &expr.Value_Int64Value{Int64Value: 1}},
					{Kind: &expr.Value_Int64Value{Int64Value: 2}},
				},
			},
		},
	}
	v2 := proto.Clone(v1).(*expr.Value)
	if !equalValues(v1, v2) {
		t.Error("identical list values should be equal")
	}

	// Map equality regardless of order
	m1 := &expr.Value{
		Kind: &expr.Value_MapValue{
			MapValue: &expr.MapValue{
				Entries: []*expr.MapValue_Entry{
					{
						Key:   &expr.Value{Kind: &expr.Value_StringValue{StringValue: "a"}},
						Value: &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: 1}},
					},
					{
						Key:   &expr.Value{Kind: &expr.Value_StringValue{StringValue: "b"}},
						Value: &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: 2}},
					},
				},
			},
		},
	}
	m2 := &expr.Value{
		Kind: &expr.Value_MapValue{
			MapValue: &expr.MapValue{
				Entries: []*expr.MapValue_Entry{
					{
						Key:   &expr.Value{Kind: &expr.Value_StringValue{StringValue: "b"}},
						Value: &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: 2}},
					},
					{
						Key:   &expr.Value{Kind: &expr.Value_StringValue{StringValue: "a"}},
						Value: &expr.Value{Kind: &expr.Value_Int64Value{Int64Value: 1}},
					},
				},
			},
		},
	}
	if !equalValues(m1, m2) {
		t.Error("map values with different entry ordering should be equal")
	}
}

func TestEvaluateConformanceFileOnDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.textproto")
	if err := os.WriteFile(path, []byte(sampleTextproto), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := EvaluateConformance(path, nil)
	if err != nil {
		t.Fatalf("EvaluateConformance with file path failed: %v", err)
	}
	if out.Passed != out.Total || out.Total == 0 {
		t.Errorf("expected all tests to pass, got total=%d passed=%d", out.Total, out.Passed)
	}
}

func findCelSpecTestdataDir() string {
	if dir := os.Getenv("CEL_SPEC_TESTDATA"); dir != "" {
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return dir
		}
	}
	if dir := os.Getenv("CEL_SPEC_DIR"); dir != "" {
		sub := filepath.Join(dir, "tests", "simple", "testdata")
		if fi, err := os.Stat(sub); err == nil && fi.IsDir() {
			return sub
		}
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	candidates := []string{
		filepath.Join(gopath, "src", "github.com", "google", "cel-spec", "tests", "simple", "testdata"),
		filepath.Join("..", "..", "..", "google", "cel-spec", "tests", "simple", "testdata"),
		filepath.Join("..", "..", "cel-spec", "tests", "simple", "testdata"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	return ""
}

func TestExternalCelSpecBasicTextproto(t *testing.T) {
	dir := findCelSpecTestdataDir()
	if dir == "" {
		t.Skip("cel-spec testdata directory not found")
	}
	celSpecBasic := filepath.Join(dir, "basic.textproto")
	if _, err := os.Stat(celSpecBasic); os.IsNotExist(err) {
		t.Skip("cel-spec basic.textproto not available in local environment")
	}

	out, err := EvaluateConformance(celSpecBasic, nil)
	if err != nil {
		t.Fatalf("failed evaluating cel-spec basic.textproto: %v", err)
	}

	t.Logf("Evaluated cel-spec basic.textproto: %d tests, %d passed, %d failed", out.Total, out.Passed, out.Failed)
	if out.Failed > 0 {
		for _, r := range out.Results {
			if !r.Passed() {
				t.Errorf("test %s failed: %s", r.Name, r.Status)
			}
		}
	}
	if out.Passed != out.Total {
		t.Errorf("expected all %d tests in basic.textproto to pass, got %d passed", out.Total, out.Passed)
	}
}

func TestExternalCelSpecBlockExtTextproto(t *testing.T) {
	dir := findCelSpecTestdataDir()
	if dir == "" {
		t.Skip("cel-spec testdata directory not found")
	}
	celSpecBlock := filepath.Join(dir, "block_ext.textproto")
	if _, err := os.Stat(celSpecBlock); os.IsNotExist(err) {
		t.Skip("cel-spec block_ext.textproto not available in local environment")
	}

	out, err := EvaluateConformance(celSpecBlock, nil)
	if err != nil {
		t.Fatalf("failed evaluating cel-spec block_ext.textproto: %v", err)
	}

	t.Logf("Evaluated cel-spec block_ext.textproto: %d tests, %d passed, %d failed", out.Total, out.Passed, out.Failed)
	if out.Failed > 0 {
		for _, r := range out.Results {
			if !r.Passed() {
				t.Errorf("test %s failed: %s", r.Name, r.Status)
			}
		}
	}
	if out.Passed != out.Total {
		t.Errorf("expected all %d tests in block_ext.textproto to pass, got %d passed", out.Total, out.Passed)
	}
}

func TestEvaluateConformanceProtoJSON(t *testing.T) {
	// Single test in ProtoJSON format
	jsonSingle := `{
		"name": "json_single_test",
		"expr": "10 * 20",
		"value": { "int64Value": "200" }
	}`
	out, err := EvaluateConformance(jsonSingle, nil)
	if err != nil {
		t.Fatalf("EvaluateConformance with protojson single test failed: %v", err)
	}
	if out.Total != 1 || out.Passed != 1 {
		t.Errorf("expected 1 passed test, got total=%d passed=%d", out.Total, out.Passed)
	}

	// Full SimpleTestFile in ProtoJSON format
	jsonFile := `{
		"name": "json_file_suite",
		"description": "Suite in ProtoJSON",
		"section": [
			{
				"name": "arithmetic",
				"test": [
					{
						"name": "add",
						"expr": "5 + 5",
						"value": { "int64Value": "10" }
					},
					{
						"name": "sub",
						"expr": "15 - 5",
						"value": { "int64Value": "10" }
					}
				]
			}
		]
	}`
	out, err = EvaluateConformance(jsonFile, nil)
	if err != nil {
		t.Fatalf("EvaluateConformance with protojson suite failed: %v", err)
	}
	if out.Total != 2 || out.Passed != 2 {
		t.Errorf("expected 2 passed tests, got total=%d passed=%d", out.Total, out.Passed)
	}
}

func TestEndToEndAllCelSpecConformance(t *testing.T) {
	celSpecDir := findCelSpecTestdataDir()
	if celSpecDir == "" {
		t.Skip("cel-spec testdata directory not found")
	}

	files, err := filepath.Glob(filepath.Join(celSpecDir, "*.textproto"))
	if err != nil {
		t.Fatalf("failed globbing cel-spec textproto files: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no cel-spec textproto files found in %s", celSpecDir)
	}

	totalAll := 0
	passedAll := 0
	failedAll := 0
	skippedAll := 0

	for _, file := range files {
		fileName := filepath.Base(file)
		// Skip network_ext as it tests IP extensions not included in cel-go core
		if fileName == "network_ext.textproto" {
			t.Logf("Skipping %s (requires network extensions outside cel-go core)", fileName)
			continue
		}

		t.Run(fileName, func(t *testing.T) {
			out, err := EvaluateConformanceWithParams(ConformanceParams{
				Tests:     file,
				SkipTests: DefaultConformanceSkipTests(),
			})
			if err != nil {
				t.Fatalf("failed evaluating %s: %v", fileName, err)
			}

			totalAll += out.Total
			passedAll += out.Passed
			failedAll += out.Failed
			skippedAll += out.Skipped

			t.Logf("[%s] Total: %d, Passed: %d, Failed: %d, Skipped: %d",
				fileName, out.Total, out.Passed, out.Failed, out.Skipped)

			if out.Failed > 0 {
				for _, r := range out.Results {
					if !r.Passed() {
						t.Errorf("FAIL in %s: %s -> %s", fileName, r.Name, r.Status)
					}
				}
			}
		})
	}

	t.Logf("OVERALL CONFORMANCE SUMMARY: Total: %d, Passed: %d, Skipped: %d, Failed: %d across %d files",
		totalAll, passedAll, skippedAll, failedAll, len(files)-1)
	if failedAll > 0 {
		t.Errorf("conformance evaluation had %d failures out of %d tests", failedAll, totalAll)
	}
}

func TestConformanceReviewEdgeCases(t *testing.T) {
	t.Run("equalValues_nil_ObjectValue", func(t *testing.T) {
		vNil := &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: nil}}
		vVal := &expr.Value{Kind: &expr.Value_ObjectValue{ObjectValue: &anypb.Any{TypeUrl: "test"}}}

		// Must not panic
		if equalValues(vNil, vVal) {
			t.Error("nil ObjectValue should not equal non-nil ObjectValue")
		}
		if equalValues(vVal, vNil) {
			t.Error("non-nil ObjectValue should not equal nil ObjectValue")
		}
		if !equalValues(vNil, vNil) {
			t.Error("nil ObjectValue should equal nil ObjectValue")
		}
	})

	t.Run("formatTestName_symmetry", func(t *testing.T) {
		tests := []struct {
			file, section, test string
			want                string
		}{
			{"suite", "sec", "t1", "suite/sec/t1"},
			{"suite", "", "t1", "suite/t1"},
			{"", "sec", "t1", "sec/t1"},
			{"", "", "t1", "t1"},
		}
		for _, tc := range tests {
			got := formatTestName(tc.file, tc.section, tc.test)
			if got != tc.want {
				t.Errorf("formatTestName(%q, %q, %q) = %q, want %q", tc.file, tc.section, tc.test, got, tc.want)
			}
		}
	})

	t.Run("DefaultConformanceSkipTests_defensive_copy", func(t *testing.T) {
		s1 := DefaultConformanceSkipTests()
		s2 := DefaultConformanceSkipTests()
		if len(s1) == 0 {
			t.Fatal("expected non-empty skip list")
		}
		s1[0] = "mutated_value"
		if s2[0] == "mutated_value" {
			t.Error("DefaultConformanceSkipTests did not return an independent defensive copy")
		}
	})

	t.Run("parseConformanceBytes_error_diagnostics", func(t *testing.T) {
		longInvalid := "invalid_syntax_key: {" + strings.Repeat("x", 500)
		_, err := ParseConformanceFile(longInvalid)
		if err == nil {
			t.Fatal("expected error on invalid syntax, got nil")
		}
		errMsg := err.Error()
		if len(errMsg) > 600 {
			t.Errorf("error message unexpectedly large (%d bytes), expected bounded sample", len(errMsg))
		}
		if !strings.Contains(errMsg, "truncated") {
			t.Errorf("expected truncated notice in error message, got: %s", errMsg)
		}
	})

	t.Run("jsonToValue_integer_bounds", func(t *testing.T) {
		v, err := jsonToValue(float64(42))
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := v.Kind.(*expr.Value_Int64Value); !ok {
			t.Errorf("expected Int64Value for whole number float, got %T", v.Kind)
		}

		v2, err := jsonToValue(3.14)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := v2.Kind.(*expr.Value_DoubleValue); !ok {
			t.Errorf("expected DoubleValue for fractional float, got %T", v2.Kind)
		}
	})

	t.Run("FormatSimpleTestFile", func(t *testing.T) {
		f := &testpb.SimpleTestFile{
			Name: "fmt_test",
			Section: []*testpb.SimpleTestSection{
				{
					Name: "s1",
					Test: []*testpb.SimpleTest{
						{Name: "t1", Expr: "true"},
					},
				},
			},
		}
		formatted, err := FormatSimpleTestFile(f)
		if err != nil {
			t.Fatalf("FormatSimpleTestFile failed: %v", err)
		}
		if !strings.Contains(formatted, "fmt_test") {
			t.Errorf("expected formatted output to contain fmt_test, got: %s", formatted)
		}
		if _, err := FormatSimpleTestFile(nil); err == nil {
			t.Error("expected error for nil SimpleTestFile, got nil")
		}
	})

	t.Run("HasFailures", func(t *testing.T) {
		oPass := &ConformanceOutput{Passed: 5, Failed: 0}
		if oPass.HasFailures() {
			t.Error("expected HasFailures() = false when Failed is 0")
		}
		oFail := &ConformanceOutput{Passed: 4, Failed: 1}
		if !oFail.HasFailures() {
			t.Error("expected HasFailures() = true when Failed > 0")
		}
	})

	t.Run("EvaluateConformanceTest_nil", func(t *testing.T) {
		_, err := EvaluateConformanceTest(nil, nil)
		if err == nil {
			t.Error("expected error evaluating nil SimpleTest, got nil")
		}
	})

	t.Run("ParseConformanceFile_explicit_path_errors", func(t *testing.T) {
		// Non-existent explicit file path
		_, err := ParseConformanceFile("non_existent_file.textproto")
		if err == nil {
			t.Error("expected error for missing .textproto file, got nil")
		} else if !strings.Contains(err.Error(), "failed reading conformance test file") {
			t.Errorf("expected 'failed reading conformance test file' error, got: %v", err)
		}

		// Directory path
		tmpDir := t.TempDir()
		_, err = ParseConformanceFile(tmpDir)
		if err == nil {
			t.Error("expected error for directory path, got nil")
		} else if !strings.Contains(err.Error(), "is a directory") {
			t.Errorf("expected 'is a directory' error, got: %v", err)
		}
	})

	t.Run("valueToJSON_exhaustive", func(t *testing.T) {
		tests := []struct {
			name string
			val  *expr.Value
		}{
			{"null", &expr.Value{Kind: &expr.Value_NullValue{}}},
			{"uint", &expr.Value{Kind: &expr.Value_Uint64Value{Uint64Value: 100}}},
			{"double", &expr.Value{Kind: &expr.Value_DoubleValue{DoubleValue: 2.718}}},
			{"bytes", &expr.Value{Kind: &expr.Value_BytesValue{BytesValue: []byte("bytes_test")}}},
			{"type", &expr.Value{Kind: &expr.Value_TypeValue{TypeValue: "int"}}},
			{"enum", &expr.Value{Kind: &expr.Value_EnumValue{EnumValue: &expr.EnumValue{Type: "test.Enum", Value: 1}}}},
		}
		for _, tc := range tests {
			jv, err := valueToJSON(tc.val)
			if err != nil {
				t.Errorf("%s: valueToJSON failed: %v", tc.name, err)
			}
			if tc.name != "null" && jv == nil {
				t.Errorf("%s: expected non-nil result", tc.name)
			}
		}
	})
}


