---
name: cel-authoring
description: >-
  Authoring, configuring, and testing Common Expression Language (CEL)
  expressions, policies, rules, and environment JSON configurations.
---

# Common Expression Language (CEL) Authoring Skill

Use this skill only when writing or modifying CEL expressions (do not invoke
`cel-expr` during read-only triage or investigation where no CEL expression is
authored or changed):

```bash
go install github.com/cel-expr/skills/cmd/cel-expr@latest
alias cel-expr="$(go env GOPATH)/bin/cel-expr"
alias proto-fds="protoc --include_imports --descriptor_set_out=/dev/stdout"
```

## Workflow

Follow this end-to-end workflow to author CEL expressions:

1.  **Locate or Create Environment**: Reuse an existing
    `<skill_dir>/resources/cel_env.json` or workspace `*_env.json`, or author
    and validate a new `envConfig` with `cel-expr env -env <env.json>`.
2.  **Generate Prompt (Optional)**: Skip if you have already read a concise
    `<env.json>`. Run `cel-expr prompt` only for large/extension-heavy
    environments or to expand protobuf (`-fds`) field paths.
3.  **Create, Evaluate, and Test**: Author the expression and run
    `cel-expr eval` with `-bindings` or `-tests` to compile, evaluate runnable
    test cases, and achieve >95% node and branch coverage (or `cel-expr compile`
    when no test bindings are needed).

## Locate or Create Environment

Check for an existing `<skill_dir>/resources/cel_env.json` or workspace
`*_env.json` first and read it before searching evaluator source code.

If no environment JSON exists, search the codebase for how the CEL environment
is configured (variables, functions, macros, and features), generate an
`envConfig` object conforming to
[env_config_schema.json](skills/cel-authoring/references/env_config_schema.json),
and validate it:

```bash
cel-expr env -env <env.json>
```

*   Namespaced variables with simple types improve correctness checks.
*   Use namespacing for related concepts, e.g. `request.path`, `request.time`.
*   Use protobuf object types for strongly typed structured data. Declare
    protobuf types with their **fully-qualified** package names (declare enum
    variables as `int`) and pass `-proto_db` or `-fds`:
    ```bash
    cel-expr env -env <env.json> -proto_db
    # or with an explicit FileDescriptorSet from protoc:
    cel-expr env -env <env.json> -fds <(proto-fds -I <proto_dir> <schema.proto>)
    ```
    When expressions reference protobuf enums or message literals not present in
    `variables`, infer the fully-qualified type from the expression on
    `undeclared reference` and add it to `"protoTypes"` in `<env.json>` (never
    add `"imports"` solely to load descriptors). See
    [protobuf_types.md](skills/cel-authoring/references/protobuf_types.md).
*   Only use `map<string, dyn>` when the variable is a dynamic structure like JSON.
*   Type names are formatted according to:
    [type_grammar_ebnf.txt](skills/cel-authoring/references/type_grammar_ebnf.txt).

Save the `envConfig` JSON to a file after validation.

## Generate Prompt (Optional)

If you have already read a concise `<env.json>`, skip `cel-expr prompt` and
proceed directly to `cel-expr eval`. Reserve `cel-expr prompt` for large or
extension-heavy environments, or when protobuf descriptor (`-fds`) field paths
need to be expanded:

```bash
cel-expr prompt -env <env.json> [-fds <(proto-fds -I <proto_dir> <schema.proto>)] \
  -prompt "<user request>"
```

## Create, Evaluate, and Test

Create the simplest possible expression that satisfies the user's requirements
and run `cel-expr eval` to compile, type-check, and evaluate in one step (use
`cel-expr compile` as an alias when no test cases or bindings are needed).
Correct any compilation or runtime errors with the
[cel-debugging](skills/cel-debugging/SKILL.md) skill.

For a single test case, pass `-bindings` and `-expected`:

```bash
cel-expr eval -env <env.json> -expr "<expression>" \
  -bindings '{"request.path": "/api/v1/users"}' -expected true
```

For a test suite, pass `-tests` as an inline JSON array string (avoiding
temporary files) or a file path. The `-tests` payload must be a top-level JSON
array (`[...]`) of `{"testCase", "bindings", "expected"}` objects conforming to
[test_cases_schema.json](skills/cel-testing/references/test_cases_schema.json):

```bash
cel-expr eval -env <env.json> -expr "<expression>" \
  -tests '[{"testCase": "valid path", "bindings": {"request.path": "/api/v1/users"}, "expected": true}, {"testCase": "invalid path", "bindings": {"request.path": "/other"}, "expected": false}]'
```

When `<env.json>` declares custom `functions` without runtime implementations,
`cel-expr eval` runs all runnable test cases and marks test cases whose result
depends on an unknown function as `skipped` (enumerated in `skippedTestCases`).

Test cases must cover edge cases and missing fields to ensure robustness
and correctness. Prune tests which do not increase coverage.

Edge cases:

*   Using a map where a qualified identifier with a simple type would be safer.
    *   Before: `request: map<string, dyn>` and `request.path == 'value'`
    *   After: `request.path: string` and `request.path == 'value'`
*   Unguarded map key or list index accesses:
    *   Before: `m[k]`, After: `k in m && m[k]` or `m[?k].orValue(<default>)`
    *   Before: `l[i]`, After: `l.size() > i && l[i]` or `l[?i].orValue(<default>)`
Note: the `?` syntax requires enabling `optional` extension in the `envConfig`.

The coverage report for `cel-expr eval` indicates the node and branch coverage
percentages as a pre-formatted string (e.g., `"Node: %.2f%%, Branch: %.2f%%"`).
Generate additional test cases until you achieve >95% node and branch coverage.

Report the test results with node and branch coverage percentages.

## Complete Example

The following is a complete example of the artifacts generated from the prompt:
"Validate the request is authenticated by checking for a bearer token"

*   Environment:
    [network_env.json](skills/cel-authoring/examples/network_env.json)
*   Expression:
    [network_headers.cel](skills/cel-authoring/examples/network_headers.cel)
*   Tests:
    [network_headers_tests.json](skills/cel-authoring/examples/network_headers_tests.json)
