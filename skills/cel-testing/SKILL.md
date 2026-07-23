---
name: cel-testing
description: >-
  Skill for testing Google Common Expression Language (CEL) expressions.
  Use to test or validate an existing CEL rule.
---

# Google Common Expression Language (CEL) Testing Skill

Use this skill to test and validate CEL expressions with a variety of inputs to
ensure correctness and high coverage.

## Workflow

Follow these steps to test a CEL expression:

*   **Compile the Expression** - use `cel_compile` to validate the `{EXPR}.cel`
    compiles with the `{ENV}.json`.
*   **Generate Tests** - Use the `inputSchema` and `outputSchema` from a
    successful `cel_compile` to generate test inputs and outputs to a
    `{SUITE}.json` file.
*   **Evaluate** - Evaluate the test cases with `cel_evaluate`.
*   **Improve Coverage** - Improve coverage until the `cel_evaluate` indicates
    100% branch and node coverage.

### 1. Compile the Expression

Provide the `{ENV}.json` and `{EXPR}.cel` to the `cel_compile` tool. If
successful, the result will contain the `inputSchema` and `outputSchema`
associated with the expression.

If the compilation fails, use the [cel-debugging](../cel-debugging/SKILL.md)
skill to correct the expression.

### 2. Generate Test Input Fixtures

Create a test suite JSON matching the `cel_evaluate` tool. A test suite is
composed of multiple test cases. Within a `testCase`, the `bindings` values must
match the `inputSchema` from the compile command. The `expected` value must
match the `outputSchema` from the compile command.

If the test input schema contains an `additionalProperties` or `items` key be
sure to generate tests where the objects are populated and empty to validate the
robustness of the expression to unexpected inputs.

Reference examples in `examples/` if unsure:

-   `examples/is_admin_policy.cel`
-   `examples/is_admin_env.json`
-   `examples/is_admin_test.json`

### 3. Run the Tests

Run tests by calling the `cel_evaluate` tool with the expression as `expr`, the
environment as `envConfig`, and the test suite content as `testCases`.

### 4. Evaluate Coverage and Iterate

Review test output for success/failure and total evaluation coverage. Pass
multiple test cases in the `testCases` to increase coverage.
