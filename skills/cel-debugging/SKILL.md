---
name: cel-debugging
description: >-
  Skill for debugging Google Common Expression Language (CEL) expressions.
  Use when an expression fails to compile or evaluate properly.
---

# Google Common Expression Language (CEL) Debugging Skill

Use this skill to diagnose and resolve CEL compilation and evaluation errors.

## Understanding the Two Phases

CEL processing has two steps:

1.  **Compilation** Validating the syntax and type-correctness of an expression
    using the `cel_compile` tool with an `{ENV}.json` and `{EXPR}.cel`. Consult
    [cel-authoring](../cel-authoring/SKILL.md) for more information.

2.  **Evaluation:** Use `cel_evaluate` to evaluate an `{EXPR}.cel` against a set
    of `testCases` stored in a `{SUITE}.json` file. Consult
    [cel-testing](../cel-testing/SKILL.md) for more information.

## Common Compilation Errors

These occur before execution due to typos, unknown symbols, or invalid types.

### 1. "Undeclared Reference"

-   **Cause:** Using an undefined variable, function, or field.
-   **Example:** `user.admin` when the `User` message only has `id` and `name`.
-   **Solution:** Check schema/prototype. Ensure variables exist in the
    environment definition.

### 2. "Type Mismatch" or "No Matching Overload"

-   **Cause:** Calling a function or operator with incorrect types.
-   **Example:** `"123" + 456` (CEL doesn't automatically coerce strings to
    numbers).
-   **Example:** `string.startsWith(123)` (Expected string parameter).
-   **Solution:** Cast inputs (e.g., `string(456)`) or supply correct types.
    Verify function signatures.

### 3. "Syntax Error"

-   **Cause:** Invalid CEL syntax (e.g., mismatched parentheses).
-   **Example:** `user.age > 18 && (user.country == 'US'`
-   **Solution:** Fix grammar, match parentheses, quote strings.

## Common Evaluation Errors

Valid compiled expressions failing on runtime data.

### 1. "No Such Field"

-   **Cause:** Accessing a missing structural field (e.g., map key) at runtime.
-   **Solution:** Use the `has()` macro.
    -   *Incorrect:* `user.profile.website == "google.com"` (fails if `profile`
        isn't populated).
    -   *Correct:* `has(user.profile.website) && user.profile.website ==
        "google.com"`
-   **Alternative:** Enable the `optional` extension and use the `?` operator.
    -   *Incorrect:* `user.profile.website == "google.com"` (fails if `profile`
        isn't populated).
    -   *Correct:* `user.profile?.website.orValue("") == "google.com"`

### 2. "Division by Zero"

-   **Cause:** Dividing by 0.
-   **Solution:** Add conditional checks for dynamic denominators.
    -   *Better:* `y != 0 && (x / y > 10)`

### 3. "No Such Overload"

-   **Cause** A function has been declared in the `{ENV}.json`, but is not
    implemented in the CEL runtime.
-   **Solution** Determine if a there is another function which could be used to
    evaluate the desired functionality. Sometimes using more specific types will
    reveal a scenario where the type-checker did not identify the missing
    overload as the inputs to the function were marked as `dyn`.

## Strategies for Isolating Faults

To debug complex expressions:

1.  **Break it down:** Split `&&`/`||` expressions into chunks. Evaluate each
    chunk to isolate the failure.
2.  **Mock Inputs Minimally:** Test minimal JSON input, adding fields until
    failure occurs.
3.  **Verify AST:** Review AST to verify grouping and operator precedence.
4.  **Use Type Assertions:** Explicitly check dynamic types (e.g., `type(val) ==
    string`).
