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
  compile     Compile a CEL expression against an environment configuration
  eval        Evaluate a CEL expression against test cases or bindings
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

func parseCommonEnvFlags(fs *flag.FlagSet) (envPath *string, fdsPath *string) {
	envPath = fs.String("env", "", "Path to environment JSON file or inline JSON string (alias: -environment)")
	fs.StringVar(envPath, "environment", "", "Alias for -env")

	fdsPath = fs.String("fds", "", "Path to binary FileDescriptorSet file (alias: -file_descriptor_set, -file_descriptors)")
	fs.StringVar(fdsPath, "file_descriptor_set", "", "Alias for -fds")
	fs.StringVar(fdsPath, "file_descriptors", "", "Alias for -fds")
	return envPath, fdsPath
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

func loadEnvAndOpts(envFlag, fdsFlag string, stdin io.Reader) (*tools.Config, []cel.EnvOption, error) {
	envContent, err := loadContentOrStdin(envFlag, stdin)
	if err != nil {
		return nil, nil, err
	}
	if envContent == "" {
		return nil, nil, errors.New("environment configuration is required (use -env <path|json>)")
	}

	cfg, err := mcp.LoadEnvConfig(envContent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load environment configuration: %w", err)
	}

	var opts []cel.EnvOption
	if fdsFlag != "" {
		fdOpt, err := mcp.LoadFileDescriptorSet(fdsFlag)
		if err != nil {
			return nil, nil, fmt.Errorf("failed loading file descriptor set: %w", err)
		}
		opts = append(opts, fdOpt)
	}

	return cfg, opts, nil
}

func runCompile(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag := parseCommonEnvFlags(fs)
	exprFlag := fs.String("expr", "", "CEL expression string")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr compile -env <path|json> [-expr <expr>] [expression]")
		fmt.Fprintln(stderr, "\nFlags:")
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

	cfg, opts, err := loadEnvAndOpts(*envFlag, *fdsFlag, stdin)
	if err != nil {
		return err
	}

	res, err := tools.CompileCEL(expr, cfg, opts...)
	if err != nil {
		return fmt.Errorf("compilation failed: %w", err)
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
	envFlag, fdsFlag := parseCommonEnvFlags(fs)
	exprFlag := fs.String("expr", "", "CEL expression string")
	testsFlag := fs.String("tests", "", "Path to test cases JSON file or inline JSON array")
	fs.StringVar(testsFlag, "test_cases", "", "Alias for -tests")
	bindingsFlag := fs.String("bindings", "", "Single test case variable bindings JSON file or inline JSON object")
	expectedFlag := fs.String("expected", "", "Expected output value for single test case binding (JSON literal or string)")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr eval -env <path|json> [-tests <path|json>] [-bindings <path|json>] [expression]")
		fmt.Fprintln(stderr, "\nFlags:")
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

	var testCases []tools.TestCase

	if *testsFlag != "" {
		rawTests, err := loadFileOrRawJSON(*testsFlag)
		if err != nil {
			return fmt.Errorf("loading test suite: %w", err)
		}
		if err := json.Unmarshal([]byte(rawTests), &testCases); err != nil {
			return fmt.Errorf("failed parsing test cases JSON: %w", err)
		}
	} else if *bindingsFlag != "" {
		rawBindings, err := loadFileOrRawJSON(*bindingsFlag)
		if err != nil {
			return fmt.Errorf("loading bindings: %w", err)
		}
		var bindings map[string]any
		if err := json.Unmarshal([]byte(rawBindings), &bindings); err != nil {
			return fmt.Errorf("failed parsing bindings JSON: %w", err)
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

	cfg, opts, err := loadEnvAndOpts(*envFlag, *fdsFlag, stdin)
	if err != nil {
		return err
	}

	res, err := tools.EvaluateCEL(expr, cfg, testCases, opts...)
	if err != nil {
		return fmt.Errorf("evaluation failed: %w", err)
	}

	hasExpected := (*testsFlag != "") || (*expectedFlag != "")

	for i := range res.TestResults {
		tr := &res.TestResults[i]
		if !hasExpected && strings.HasPrefix(tr.Status, "failed: got") {
			tr.Status = "pass"
		}
	}

	outBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return fmt.Errorf("failed formatting output JSON: %w", err)
	}

	fmt.Fprintln(stdout, string(outBytes))

	for _, tr := range res.TestResults {
		if tr.Status != "pass" {
			return fmt.Errorf("test case %q did not pass (status: %s)", tr.TestCase, tr.Status)
		}
	}

	return nil
}

func runEnv(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("env", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag := parseCommonEnvFlags(fs)

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

	cfg, opts, err := loadEnvAndOpts(*envFlag, *fdsFlag, stdin)
	if err != nil {
		return err
	}

	_, err = tools.EnvFromConfig(cfg, opts...)
	if err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
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
	envFlag, fdsFlag := parseCommonEnvFlags(fs)
	promptFlag := fs.String("prompt", "", "User requirement/prompt to generate CEL authoring prompt for")
	fs.StringVar(promptFlag, "user_prompt", "", "Alias for -prompt")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr prompt -env <path|json> [-prompt <requirement>] [requirement text]")
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

	cfg, opts, err := loadEnvAndOpts(*envFlag, *fdsFlag, stdin)
	if err != nil {
		return err
	}

	res, err := tools.GeneratePrompt(cfg, userPrompt, opts...)
	if err != nil {
		return fmt.Errorf("prompt generation failed: %w", err)
	}

	fmt.Fprintln(stdout, res)
	return nil
}

func runMCP(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envFlag, fdsFlag := parseCommonEnvFlags(fs)

	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: cel-expr mcp [-env <path|json>] [-fds <path>]")
		fmt.Fprintln(stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	s, err := mcp.SetupServer(*fdsFlag, *envFlag)
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
