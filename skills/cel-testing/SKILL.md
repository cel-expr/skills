---
name: cel-testing
description: >-
  Skill for testing Google Common Expression Language (CEL) expressions.
  Use to test or validate expressions.
---

# Google Common Expression Language (CEL) Testing Skill

Use this skill to test and validate CEL expressions with a variety of inputs to
ensure correctness and high coverage using the `cel-expr` CLI:

```bash
go install github.com/cel-expr/skills/cmd/cel-expr@latest
alias cel-expr="$(go env GOPATH)/bin/cel-expr"
```

## Workflow

Follow these steps to test a CEL expression:

*   **Compile the Expression** - use `cel-expr eval` (or its compile-only alias
    `cel-expr compile`) to validate that `{EXPR}.cel` compiles with `{ENV}.json`
    and inspect its `inputSchema` and `outputSchema`.
*   **Generate Tests** - Use the `inputSchema` and `outputSchema` to generate
    test inputs and outputs.
*   **Evaluate** - Evaluate the test cases with `cel-expr eval`.
*   **Improve Coverage** - Improve coverage until `cel-expr eval` indicates
    100% branch and node coverage across runnable test cases.

### 1. Compile the Expression

Provide the `{ENV}.json` and `{EXPR}.cel` to `cel-expr eval` (or its
compile-only alias `cel-expr compile`; pass `-proto_db` or `-fds` when protobuf
messages or enums are referenced—see
[protobuf_types.md](skills/cel-authoring/references/protobuf_types.md)):

```bash
cel-expr eval -env {ENV}.json -expr "{EXPR}"
```

If successful, the result will contain the `inputSchema` and `outputSchema`
associated with the expression.

If the compilation fails, use the [cel-debugging](skills/cel-debugging/SKILL.md)
skill to correct the expression.

### 2. Generate Test Input Fixtures

Create a test suite JSON matching the `cel-expr eval` `-tests` format defined in
[test_cases_schema.json](skills/cel-testing/references/test_cases_schema.json).
A test suite is a top-level JSON array (`[...]`) of test case objects where
`bindings` matches `inputSchema` (or is omitted for standalone expressions) and
`expected` matches `outputSchema`:

```json
[
  {
    "testCase": "is_admin",
    "bindings": {
      "request.auth.claims": {
        "groups": ["admin", "editor"]
      }
    },
    "expected": true
  }
]
```

If the test input schema contains an `additionalProperties` or `items` key be
sure to generate tests where the objects are populated and empty to validate the
robustness of the expression to unexpected inputs.

Reference examples in `examples/` if unsure:

*   [is_admin_policy.cel](skills/cel-testing/examples/is_admin_policy.cel)
*   [is_admin_env.json](skills/cel-testing/examples/is_admin_env.json)
*   [is_admin_test.json](skills/cel-testing/examples/is_admin_test.json)

### 3. Run the Tests

Run tests by calling `cel-expr eval` with an inline JSON array string (avoiding
temporary files), a test suite file, or single-case `-bindings` and `-expected`:

```bash
cel-expr eval -env {ENV}.json -expr "{EXPR}" \
  -tests '[{"testCase": "is_admin", "bindings": {"request.auth.claims": {"groups": ["admin"]}}, "expected": true}]'
# Or with a file or single binding:
cel-expr eval -env {ENV}.json -expr "{EXPR}" -tests {SUITE}.json
cel-expr eval -env {ENV}.json -expr "{EXPR}" -bindings '{"var": "val"}' -expected true
```

The `-env` flag may be omitted for standalone expressions. When `{ENV}.json`
declares custom `functions` without runtime implementations, `cel-expr eval`
evaluates all runnable cases and marks cases that depend on unknown functions as
`skipped` in `skippedTestCases`.

### 4. Evaluate Coverage and Iterate

Review test output for success/failure, `skippedTestCases`, and total evaluation
coverage. Pass multiple test cases in the test suite to increase coverage.
