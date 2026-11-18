---
name: cel-debugging
description: >-
  Diagnosing and resolving Common Expression Language (CEL) compilation and
  evaluation errors.
---

# Common Expression Language (CEL) Debugging Skill

Use this skill only when diagnosing or fixing a broken CEL expression (do not
invoke `cel-expr` during read-only triage or investigation where no CEL
expression is being written or debugged):

```bash
go install github.com/cel-expr/skills/cmd/cel-expr@latest
alias cel-expr="$(go env GOPATH)/bin/cel-expr"
alias proto-fds="protoc --include_imports --descriptor_set_out=/dev/stdout"
```

## Workflow

1.  **Identify the Phase**: Determine whether the error is a compilation
    error (syntax and types) or an evaluation error.
2.  **Locate Environment Configuration**: Inspect `<skill_dir>/resources/cel_env.json`
    or workspace `*_env.json` (conforming to
    [env_config_schema.json](skills/cel-authoring/references/env_config_schema.json))
    before searching codebase evaluator implementations. If only a `*_env.yaml`
    exists, convert it to a working JSON file first with
    `cel-expr convert-env -env <env.yaml> -out <working_env.json>`.
3.  **Apply Fix**: Correct the expression, `.celpolicy`, or working JSON
    environment configuration based on the error pattern.
4.  **Verify**: Re-validate using `cel-expr eval` (or `cel-expr compile` when no
    test bindings are needed; pass `-policy` for `.celpolicy` files or inline
    YAML policies).

## Common Compilation Errors

These errors occur before execution due to typos, unknown symbols, or invalid
type overloading.

### 1. "Undeclared Reference" or Missing Protobuf Types

Referencing an undefined variable, function, field, or unloaded protobuf
message/enum:
- Issue (typo): `user.amin` (typo for `user.admin`, `user.is_admin`)
- Solution: Declare the variable in `envConfig`, or correct typos in field/map
  key names.
- Issue (missing protobuf message/enum in expression or `envConfig`):
  `undeclared reference to 'acme'` (on a qualified enum like
  `acme.v1.Verdict.ALLOW`, where `cel-go` reports only the root identifier),
  `undeclared reference to 'acme.v1.Controls'`, or `undefined type name`
- Solution: Inspect the full dotted path or struct literal at `<line>:<col>` in
  the expression (stripping trailing `UPPER_SNAKE_CASE` enum constant names and
  expanding `container` / `imports`), add the fully-qualified message or enum
  name to `"protoTypes"` in `envConfig`, and pass `-proto_db` (or `-fds` for
  local `.proto` files). Declare enum variables with `"type": "int"`. **Do not**
  add entries to `"imports"` just to trigger type lookup, as `"imports"` alters
  expression name resolution—see
  [protobuf_types.md](skills/cel-authoring/references/protobuf_types.md).

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

The runtime types of dynamic (`dyn`) values do not agree with the types expected
by the function signature, or a required extension is not enabled in the
environment:

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

## Verification with `cel-expr` CLI

Validate corrected expressions using `cel-expr eval` (pass
`-fds <(proto-fds -I <proto_dir> <schema.proto>)` when protobuf message types are
referenced; see
[test_cases_schema.json](skills/cel-testing/references/test_cases_schema.json)
for the `-tests` schema):

```bash
cel-expr eval -env <env.json> -expr "<expression>" -bindings '{"var": "val"}' -expected true
# or with an inline test suite array / file:
cel-expr eval -env <env.json> -expr "<expression>" \
  -tests '[{"testCase": "case1", "bindings": {"var": "val"}, "expected": true}]'
```

`cel-expr eval` compiles the expression, runs all runnable test cases, and marks
any cases whose result depends on unimplemented custom `functions` as `skipped`
in `skippedTestCases` (use `cel-expr compile` when no test bindings are needed).
