---
name: cel-testing
description: >-
  Skill for testing Google Common Expression Language (CEL) expressions.
  Use to test or validate an existing CEL rule.
---

# Google Common Expression Language (CEL) Testing Skill

Use this skill to test and validate CEL expressions against varied inputs to
ensure correctness and high coverage.

## Workflow

Follow these steps to test a CEL expression:

### 1. Identify Expression Dependencies

List referenced variables in the compiled expression to identify required
inputs:

The result of the compile command is the expression return type, and a JSON
schema describing the expected input. Use this schema to generate a test input
value. Determine the test output type from the compiled expression type.

Determine the required inputs by calling the `cel_compile` tool with the
expression as `expr` and the environment configuration as `envConfig`. The
result contains the expression return type and a JSON schema describing the
expected input.

### 2. Generate Test Input Fixtures

Create a test suite JSON matching the `cel_evaluate` tool's `testCases` argument
schema. A test suite is composed of multiple test sections with test cases.
Group related test cases into sections. One section should be populated with
inputs which should succeed, one should contain inputs which should fail.

Within a `testCase`, the `bindings` values must match the `inputSchema` from the
compile command. The `expected` value must match the `outputSchema` from the
compile command.

If the test input schema contains an `additionalProperties` or `items` key be
sure to generate tests where the objects are populated and empty to validate the
robustness of the expression to unexpected inputs.

Reference examples in `examples/` if unsure:

-   `examples/is_admin_policy.cel`
-   `examples/is_admin_env.yaml`
-   `examples/is_admin_test.celtest`

### 3. Run the Tests

Run tests by calling the `cel_evaluate` tool with the expression as `expr`, the
environment as `envConfig`, and the test suite content as `testCases`.

### 4. Evaluate Coverage and Iterate

Review test output for success/failure and total evaluation coverage. Pass
multiple test sections and test cases in the `testCases` to increase coverage.