---
name: cel-debugging
description: >-
  Diagnosing and resolving Common Expression Language (CEL) compilation and
  evaluation errors.
---

# Common Expression Language (CEL) Debugging Skill

Use this skill to guide diagnosing and resolving correcting issues with CEL
expressions.

## Workflow

1.  **Identify the Phase**: Determine whether the error is a Compilation
    error (syntax and types) or an evaluation error.
2.  **Locate Environment Configuration**: Verify the variables, types, and
    extensions declared in the environment configuration JSON file
    (`envConfig`).
3.  **Apply Fix**: Correct the expression or environment configuration based
    on the error pattern.
4.  **Verify**: Re-compile and re-evaluate the expression to ensure issues
    are resolved.

## Common Compilation Errors

These errors occur before execution due to typos, unknown symbols, or invalid
type overloading.

### 1. "Undeclared Reference"

Referencing an undefined variable, function, or field:
- Issue: `user.amin` (typo for `user.admin`, `user.is_admin`)
- Solution: Declare the variable in the environment configuration
  (`envConfig`), or correct typos in field/map key names.

### 2. "Type Mismatch" or "No Matching Overload"

Invoking functions or operators with incompatible argument types:
- Issue: `"123" + 456` or `request.path.startsWith(123)`
- Solution: Use explicit type conversion functions, e.g., `int("123")` or
  `string(456)`.

### 3. "Syntax Error"

Invalid syntax such as unclosed brackets and mismatched parentheses and
invalid operators:
- Issue: `user.age > 18 || (user.country == 'US'`
- Solution: `user.age > 18 || (user.country == 'US')`
- Issue: `request.time > now & request.path != "secret"`
- Solution: `request.time > now && request.path != "secret"`

## Evaluation Errors

These errors occur at runtime when valid compiled expressions encounter missing
or unexpected data bindings.

### 1. "No Such Field" or "No Such Key"

Accessing an unset Protobuf message field or missing map key at runtime when
dynamic data is omitted:
- Issue: `user.profile.website == "goog.com"` (fails if `profile` is unset)
- Solution A - Proto Presence `has()`: Check field presence via short-circuit:
  ```
  has(user.profile) && has(user.profile.website) &&
  user.profile.website == "goog.com"
  ```
- Solution B - Optional Chaining `.?`: Traverse unset fields with `.?` and
  `.orValue()` (requires `optional` extension in `envConfig`):
  ```
  user.?profile.?website.orValue("") == "goog.com"
  ```
- Solution C - Dynamic Map Lookup `'in'`: When selecting keys via brackets,
  guard access using the `in` operator:
  ```
  'profile' in user && 'website' in user['profile'] &&
  user['profile']['website'] == "goog.com"
  ```
- Solution D - Optional Chaining Keys `[?]`: When selecting keys via brackets,
  guard access using the `[?]` operator and `orValue()`:
  ```
  user[?'profile'][?'website'].orValue("") == "goog.com"
  ```

### 2. "Division by Zero"

Dividing by a denominator that evaluates to `0` at runtime:
- Issue: `x / y > 10` (fails at runtime if `y == 0`)
- Solution: `y != 0 && (x / y > 10)`

Alternatively, you may cast numeric types to doubles to avoid division
by zero errors, though you may end up with an infinite value:
- Issue: `x / y > 10` (fails at runtime if `y == 0`)
- Solution: `double(x) / double(y) > 10.0` (possibly infinite if `y == 0.0`)

### 3. "No Such Overload" (Runtime)

A function signature is declared in the environment configuration but is not
registered in the active CEL runtime, or the types at runtime do not agree
with the types expected by the function signature:

- Issue: `json.data.items.contains('hello')` results in a runtime error of
  no such overload: contains(list, string)
- Solution A - type guarding:
  ```
  type(json.data.items) == string && json.data.items.contains('hello') ||
  type(json.data.items) == list && 'hello' in json.data.items
  ```
- Solution B - extensions: Ensure required extensions (e.g., `strings`, `lists`)
  are present in the `extensions` array of the environment configuration.
- Solution C - narrow types: Replace `dyn` typed variable declarations with
  narrower types in the `envConfig` if possible.
  - Before: `json.data.items: dyn`, After: `json.data.items: list<dyn>`
  - Before: `user: map<string,dyn>`, After: `user.age: int, user.name: string`

## Strategies for Isolating Faults

1.  **Expression Decomposition**: Split complex `&&` / `||` compound rules into
    smaller sub-expressions to test each fragment independently.
2.  **Minimal Input Mocking**: Start with a minimal JSON binding object in test
    cases and add fields incrementally to isolate the runtime failure.
3.  **Dynamic Type Checks**: Use `type(val) == string` or `type(val) == int` to
    debug unexpected dynamic types at runtime.
