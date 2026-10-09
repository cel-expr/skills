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

*   **Compile the Expression** - use `cel-expr compile` to validate that
    `{EXPR}.cel` compiles with the `{ENV}.json`.
*   **Generate Tests** - Use the `inputSchema` and `outputSchema` from a
    successful `cel-expr compile` to generate test inputs and outputs to a
    `{SUITE}.json` file.
*   **Evaluate** - Evaluate the test cases with `cel-expr eval`.
*   **Improve Coverage** - Improve coverage until `cel-expr eval` indicates
    100% branch and node coverage.

### 1. Compile the Expression

Provide the `{ENV}.json` and `{EXPR}.cel` to `cel-expr compile`:

```bash
cel-expr compile -env {ENV}.json -expr "{EXPR}"
```

If successful, the result will contain the `inputSchema` and `outputSchema`
associated with the expression.

If the compilation fails, use the [cel-debugging](../cel-debugging/SKILL.md)
skill to correct the expression.

### 2. Generate Test Input Fixtures

Create a test suite JSON matching the `cel-expr eval` format. A test suite is
composed of an array of test case objects. Within each test case, the `bindings`
values must match the `inputSchema` from the compile command. The `expected`
value must match the `outputSchema` from the compile command.

If the expression is standalone, meaning it does not reference any variables,
the `bindings` may be omitted or left empty.

If the test input schema contains an `additionalProperties` or `items` key be
sure to generate tests where the objects are populated and empty to validate the
robustness of the expression to unexpected inputs.

Reference examples in `examples/` if unsure:

*   `examples/is_admin_policy.cel`
*   `examples/is_admin_env.json`
*   `examples/is_admin_test.json`

### 3. Run the Tests

Run tests by calling `cel-expr eval`:

```bash
cel-expr eval -env {ENV}.json -expr "{EXPR}" -tests {SUITE}.json
```

Or test a single case with inline bindings:

```bash
cel-expr eval -env {ENV}.json -expr "{EXPR}" -bindings '{"var": "val"}'
```

The `env` flag may be omitted for standalone expressions which only rely on the CEL
standard library.

### 4. Evaluate Coverage and Iterate

Review test output for success/failure and total evaluation coverage. Pass
multiple test cases in the test suite to increase coverage.
