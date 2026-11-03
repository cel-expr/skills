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

// Package main provides the unified CLI for Common Expression Language (CEL) tools and MCP server.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"cel.dev/cel-go/cel"
	"github.com/cel-expr/skills/internal/mcp"
	"github.com/cel-expr/skills/internal/tools"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const usageText = `cel-expr - Common Expression Language (CEL) CLI & MCP Server

Usage:
  cel-expr <command> [arguments]

Available Commands:
  eval        Compile and evaluate a CEL expression against an environment and optional test cases
  compile     Alias for compile-only validation (all eval flags except test cases/bindings)
  conformance Evaluate CEL conformance tests from cel-spec textproto files
  env         Validate or inspect an environment configuration
  prompt      Generate an LLM authoring prompt from an environment and requirement
  mcp         Start the Model Context Protocol (MCP) server over stdio
  help        Show help for cel-expr or a specific command

Run 'cel-expr <command> -help' for more information on a command.
`

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usageText)
		return nil
	}

	cmd := args[0]
	cmdArgs := args[1:]

	switch cmd {
	case "compile":
		return runCompile(cmdArgs, stdout, stderr, stdin)
	case "eval", "evaluate":
		return runEval(cmdArgs, stdout, stderr, stdin)
	case "conformance":
		return runConformance(cmdArgs, stdout, stderr, stdin)
	case "env":
		return runEnv(cmdArgs, stdout, stderr, stdin)
	case "prompt":
		return runPrompt(cmdArgs, stdout, stderr, stdin)
	case "mcp", "serve":
		return runMCP(cmdArgs, stdout, stderr)
	case "help", "-h", "--help", "-help":
		if len(cmdArgs) > 0 {
			return run([]string{cmdArgs[0], "-help"}, stdout, stderr, stdin)
		}
		fmt.Fprint(stdout, usageText)
		return nil
	default:
		return fmt.Errorf("unknown command %q. Run 'cel-expr help' for available commands", cmd)
	}
}

func parseCommonEnvFlags(fs *flag.FlagSet) (envPath *string, fdsPath *string, protoDB *bool) {
	envPath = fs.String("env", "", "Path to environment JSON file or inline JSON string (alias: -environment)")
	fs.StringVar(envPath, "environment", "", "Alias for -env")

	fdsPath = fs.String("fds", "", "Path to binary FileDescriptorSet file (alias: -file_descriptor_set, -file_descriptors)")
	fs.StringVar(fdsPath, "file_descriptor_set", "", "Alias for -fds")
	fs.StringVar(fdsPath, "file_descriptors", "", "Alias for -fds")

	protoDB = fs.Bool("proto_db", false, "Extend FileDescriptorSet with types resolved from the proto database")
	return envPath, fdsPath, protoDB
}

func loadContentOrStdin(val string, stdin io.Reader) (string, error) {
	if val == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}
	return val, nil
}

// loadEnvAndOpts resolves the environment configuration and CEL options from the
// command flags. When requireEnv is false, a missing environment yields a nil
// config, which results in a standard CEL environment suitable for standalone
// expressions.
func loadEnvAndOpts(envFlag, fdsFlag string, protoDBFlag bool, stdin io.Reader, requireEnv bool) (*tools.Config, []cel.EnvOption, []string, error) {
	envContent, err := loadContentOrStdin(envFlag, stdin)
	if err != nil {
		return nil, nil, nil, err
	}
	if envContent == "" && requireEnv {
		return nil, nil, nil, errors.New("environment configuration is required (use -env <path|json>)")
	}

	cfg, err := mcp.LoadEnvConfig(envContent)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load environment configuration: %w", err)
	}

	var opts []cel.EnvOption
	var warnings []string
	if fdsFlag != "" || protoDBFlag {
		opt, warns, err := mcp.BuildDescriptorEnvOption(fdsFlag, cfg, protoDBFlag)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed loading file descriptor set: %w", err)
		}
		if opt != nil {
			opts = append(opts, opt)
		}
		warnings = warns
	}

	return cfg, opts, warnings, nil
}

func runCompile(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag, protoDBFlag := parseCommonEnvFlags(fs)
	exprFlag := fs.String("expr", "", "CEL expression string")

	fs.Usage = func() {
		fmt.Fprintln(stderr, `Usage: cel-expr compile [-env <path|json>] [-fds <path>] [-proto_db] [-expr <expr>] [expression]

Alias for compile-only validation: supports all environment, descriptor, and expression flags
that 'cel-expr eval' supports, without test cases or bindings. Outputs inputSchema and outputSchema.

Flags:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	expr := *exprFlag
	if expr == "" && fs.NArg() > 0 {
		expr = strings.Join(fs.Args(), " ")
	}
	if expr == "" {
		return errors.New("expression is required (use -expr <expr> or pass as argument)")
	}

	cfg, opts, warnings, err := loadEnvAndOpts(*envFlag, *fdsFlag, *protoDBFlag, stdin, false)
	if err != nil {
		return tools.AugmentErrorWithWarnings(err, warnings)
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	res, err := tools.CompileCEL(expr, cfg, opts...)
	if err != nil {
		return tools.AugmentErrorWithWarnings(fmt.Errorf("compilation failed: %w", err), warnings)
	}

	outBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format output JSON: %w", err)
	}

	fmt.Fprintln(stdout, string(outBytes))
	return nil
}

func runEval(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag, protoDBFlag := parseCommonEnvFlags(fs)
	exprFlag := fs.String("expr", "", "CEL expression string")
	testsFlag := fs.String("tests", "", "Path to test cases JSON file or inline JSON array: '[{\"testCase\": \"...\", \"bindings\": {...}, \"expected\": ...}]'")
	fs.StringVar(testsFlag, "test_cases", "", "Alias for -tests")
	bindingsFlag := fs.String("bindings", "", "Single test case variable bindings JSON file or inline JSON object: '{\"var\": \"val\"}'")
	expectedFlag := fs.String("expected", "", "Expected output value for single test case binding (JSON literal or string)")
	conformanceFlag := fs.String("conformance", "", "Path to conformance textproto file or inline textproto content")

	fs.Usage = func() {
		fmt.Fprintln(stderr, `Usage: cel-expr eval [-env <path|json>] [-fds <path>] [-tests <path|json>] [-bindings <path|json>] [-expected <json|string>] [-conformance <path|textproto>] [-expr <expr>] [expression]

Compiles and evaluates a CEL expression against an environment configuration:
  1. Compiles and type-checks the expression (reporting inputSchema and outputSchema).
  2. Evaluates all runnable test cases from -tests, -bindings/-expected, or -conformance
     (cel-spec SimpleTestFile/SimpleTest textproto suites) and reports AST coverage.
  3. Automatically marks test cases that cannot be evaluated at runtime (e.g., branches calling
     custom functions declared in -env without runtime implementations, or expressions requiring
     variables when no bindings are provided) as "skipped: ..." and lists them in skippedTestCases.
  4. Exits 0 when compilation succeeds and all runnable test cases pass; exits non-zero if
     compilation fails or any runnable test case fails.

Examples:
  # Single test case with inline bindings and expected value:
  cel-expr eval -env env.json -bindings '{"x": 10}' -expected true 'x > 5'

  # Test suite with inline JSON array:
  cel-expr eval -env env.json -tests '[{"testCase": "positive", "bindings": {"x": 10}, "expected": true}]' 'x > 5'

  # Conformance test suite from textproto:
  cel-expr eval -env env.json -conformance tests.textproto

Flags:`)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	expr := *exprFlag
	if expr == "" && fs.NArg() > 0 {
		expr = strings.Join(fs.Args(), " ")
	}

	if *conformanceFlag != "" {
		conformanceArgs := []string{"-tests", *conformanceFlag}
		if *envFlag != "" {
			conformanceArgs = append(conformanceArgs, "-env", *envFlag)
		}
		if *fdsFlag != "" {
			conformanceArgs = append(conformanceArgs, "-fds", *fdsFlag)
		}
		if *protoDBFlag {
			conformanceArgs = append(conformanceArgs, "-proto_db")
		}
		if expr != "" {
			conformanceArgs = append(conformanceArgs, "-expr", expr)
		}
		return runConformance(conformanceArgs, stdout, stderr, stdin)
	}

	if expr == "" {
		return errors.New("expression is required (use -expr <expr> or pass as argument)")
	}

	var testCases []tools.TestCase

	if *testsFlag != "" {
		rawTests, err := loadFileOrRawJSON(*testsFlag)
		if err != nil {
			return fmt.Errorf("loading test suite: %w", err)
		}
		if err := json.Unmarshal([]byte(rawTests), &testCases); err != nil {
			return fmt.Errorf("failed parsing test cases JSON: %w", err)
		}
	} else if *bindingsFlag != "" || *expectedFlag != "" {
		// Bindings are optional here: a standalone expression may only declare
		// an expected result.
		bindings := map[string]any{}
		if *bindingsFlag != "" {
			rawBindings, err := loadFileOrRawJSON(*bindingsFlag)
			if err != nil {
				return fmt.Errorf("loading bindings: %w", err)
			}
			if err := json.Unmarshal([]byte(rawBindings), &bindings); err != nil {
				return fmt.Errorf("failed parsing bindings JSON: %w", err)
			}
		}

		var expected any
		if *expectedFlag != "" {
			if err := json.Unmarshal([]byte(*expectedFlag), &expected); err != nil {
				// Treat as plain string if not valid JSON literal
				expected = *expectedFlag
			}
		}

		testCases = append(testCases, tools.TestCase{
			TestCase: "CLI evaluation",
			Bindings: bindings,
			Expected: expected,
		})
	} else {
		// Default empty bindings evaluation
		testCases = append(testCases, tools.TestCase{
			TestCase: "default",
			Bindings: map[string]any{},
		})
	}

	cfg, opts, warnings, err := loadEnvAndOpts(*envFlag, *fdsFlag, *protoDBFlag, stdin, false)
	if err != nil {
		return tools.AugmentErrorWithWarnings(err, warnings)
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	res, err := tools.EvaluateCEL(expr, cfg, testCases, opts...)
	if err != nil {
		return tools.AugmentErrorWithWarnings(fmt.Errorf("evaluation failed: %w", err), warnings)
	}

	hasExplicitTests := (*testsFlag != "") || (*bindingsFlag != "") || (*expectedFlag != "")
	hasExpected := (*testsFlag != "") || (*expectedFlag != "")

	for i := range res.TestResults {
		tr := &res.TestResults[i]
		if !hasExpected && strings.HasPrefix(tr.Status, "failed: got") {
			tr.Status = "pass"
		} else if !hasExplicitTests && strings.HasPrefix(tr.Status, "no such attribute(s):") {
			tr.Status = fmt.Sprintf("skipped: missing bindings (%s)", tr.Status)
			res.SkippedTestCases = append(res.SkippedTestCases, tr.TestCase)
		}
	}

	outBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("failed formatting output JSON: %w", err)
	}

	fmt.Fprintln(stdout, string(outBytes))

	for _, tr := range res.TestResults {
		if tr.Status != "pass" && !strings.HasPrefix(tr.Status, "skipped:") {
			return fmt.Errorf("test case %q did not pass (status: %s)", tr.TestCase, tr.Status)
		}
	}

	return nil
}

func runConformance(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag, protoDBFlag := parseCommonEnvFlags(fs)
	testsFlag := fs.String("tests", "", "Path to conformance textproto file or inline textproto content")
	filterFlag := fs.String("filter", "", "Optional substring filter for test names")
	skipFlag := fs.String("skip", "", "Comma-separated test prefixes to skip")
	exprFlag := fs.String("expr", "", "Optional fallback expression if test does not define one")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr conformance [-tests <path|textproto>] [-filter <name>] [-skip <prefixes>] [-env <path|json>] [-fds <path>] [tests]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	testsVal := *testsFlag
	if testsVal == "" && fs.NArg() > 0 {
		testsVal = strings.Join(fs.Args(), " ")
	}
	if testsVal == "" {
		return errors.New("conformance tests are required (use -tests <path|textproto> or pass as argument)")
	}

	rawTests, err := loadContentOrStdin(testsVal, stdin)
	if err != nil {
		return fmt.Errorf("reading conformance tests: %w", err)
	}

	var cfg *tools.Config
	var opts []cel.EnvOption
	var warnings []string
	if *envFlag != "" || *fdsFlag != "" || *protoDBFlag {
		var err error
		cfg, opts, warnings, err = loadEnvAndOpts(*envFlag, *fdsFlag, *protoDBFlag, stdin, false)
		if err != nil {
			return tools.AugmentErrorWithWarnings(err, warnings)
		}
		for _, w := range warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
	}

	var skipTests []string
	if *skipFlag != "" {
		for _, s := range strings.Split(*skipFlag, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				skipTests = append(skipTests, s)
			}
		}
	}

	res, err := tools.EvaluateConformanceWithParams(tools.ConformanceParams{
		Tests:        rawTests,
		Filter:       *filterFlag,
		SkipTests:    skipTests,
		EnvConfig:    cfg,
		FallbackExpr: *exprFlag,
	}, opts...)
	if err != nil {
		return tools.AugmentErrorWithWarnings(fmt.Errorf("conformance evaluation failed: %w", err), warnings)
	}

	outBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("failed formatting output JSON: %w", err)
	}

	fmt.Fprintln(stdout, string(outBytes))

	if res.HasFailures() {
		return fmt.Errorf("%d of %d conformance tests failed", res.Failed, res.Total)
	}

	return nil
}

func runEnv(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag, protoDBFlag := parseCommonEnvFlags(fs)

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr env -env <path|json> [validate]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	cfg, opts, warnings, err := loadEnvAndOpts(*envFlag, *fdsFlag, *protoDBFlag, stdin, true)
	if err != nil {
		return tools.AugmentErrorWithWarnings(err, warnings)
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	_, err = tools.EnvFromConfig(cfg, opts...)
	if err != nil {
		return tools.AugmentErrorWithWarnings(fmt.Errorf("environment validation failed: %w", err), warnings)
	}

	summary := map[string]any{
		"valid":       true,
		"name":        cfg.Name,
		"variables":   len(cfg.Variables),
		"extensions":  len(cfg.Extensions),
		"functions":   len(cfg.Functions),
		"description": cfg.Description,
	}

	outBytes, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, string(outBytes))
	return nil
}

func runPrompt(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag, protoDBFlag := parseCommonEnvFlags(fs)
	promptFlag := fs.String("prompt", "", "User requirement/prompt to generate CEL authoring prompt for")
	fs.StringVar(promptFlag, "user_prompt", "", "Alias for -prompt")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr prompt [-env <path|json>] [-prompt <requirement>] [requirement text]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	userPrompt := *promptFlag
	if userPrompt == "" && fs.NArg() > 0 {
		userPrompt = strings.Join(fs.Args(), " ")
	}
	if userPrompt == "" {
		return errors.New("prompt requirement is required (use -prompt <text> or pass as arguments)")
	}

	cfg, opts, warnings, err := loadEnvAndOpts(*envFlag, *fdsFlag, *protoDBFlag, stdin, false)
	if err != nil {
		return tools.AugmentErrorWithWarnings(err, warnings)
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "warning: %s\n", w)
	}

	res, err := tools.GeneratePrompt(cfg, userPrompt, opts...)
	if err != nil {
		return tools.AugmentErrorWithWarnings(fmt.Errorf("prompt generation failed: %w", err), warnings)
	}

	fmt.Fprintln(stdout, res)
	return nil
}

func runMCP(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag, protoDBFlag := parseCommonEnvFlags(fs)

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr mcp [-env <path|json>] [-fds <path>] [-proto_db]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	s, err := mcp.SetupServer(*fdsFlag, *envFlag, *protoDBFlag)
	if err != nil {
		return fmt.Errorf("setup server error: %w", err)
	}

	return s.Run(context.Background(), &sdk.StdioTransport{})
}

func loadFileOrRawJSON(pathOrJSON string) (string, error) {
	trimmed := strings.TrimSpace(pathOrJSON)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}
	data, err := os.ReadFile(pathOrJSON)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
