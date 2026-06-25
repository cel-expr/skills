---
name: cel-authoring
description: >-
  Authoring, configuring, and testing Common Expression Language (CEL)
  expressions, policies, rules, and environment JSON configurations.
---

# Common Expression Language (CEL) Authoring Skill

Use this skill to guide authoring, compiling, and testing Common Expression
Language (CEL) expressions using the CEL MCP tools.

## Workflow

Follow this end-to-end workflow to author CEL expressions:

1.  **Locate or Create Environment**: Identify/write the CEL environment
    configuration JSON file (`envConfig`). Call `cel_create_environment` to
    validate the `envConfig`.
2.  **Generate Prompt**: Call `cel_generate_prompt` with the user request to
    create the CEL authoring prompt.
3.  **Create Expression**: Generate CEL using only the authoring prompt. Call
    `cel_compile` to validate the expression.
4.  **Evaluate and Test**: Call `cel_evaluate` to verify behavior and iterate
    on test coverage until there is >95% node and branch coverage.

## Locate or Create Environment

Find the environment definition containing the variables, functions, macros and
features supported in your CEL expression using `code_search`, `find_by_name`,
or `grep_search` and the `view_file` tool to read it.

If the environment does not exist, generate an `envConfig` object and call
`cel_create_environment` to validate it.

*   Namespaced variables with simple types improve correctness checks.
*   Use namespacing for related concepts, e.g. `request.path`, `request.time`.
*   Use protobuf object types for strongly typed structured data.
*   Only use `map<string, dyn>` the variable is a dynamic structure like JSON
*   Type names are formatted according to:
    [type_grammar_ebnf.txt](google3/third_party/cel/skills/skills/cel_authoring/references/type_grammar_ebnf.txt).

Save the `envConfig` json using `write_to_file` after validation.

## Generate Prompt

Provide the JSON environment to `cel_generate_prompt` with the user's original
request. Exclusively use the output CEL prompt for generating expressions.

## Create Expression

Use the CEL prompt to create the simplest possible expression which satisfies
the user's requirements. Call `cel_compile` to validate the CEL. Correct errors
with the
[cel-debugging](googl3/third_party/cel/skills/skills/cel_debugging/SKILL.md)
skill.

## Evaluate and Test

Call `cel_evaluate` with an expression and test cases to validate an expression.
Test cases must cover edge cases and missing fields to ensure robustness
and correctness. Prune tests which do not increase coverage.

Edge cases:

* Using a map where a qualified identifier with a simple type would be safer.
  * Before: `request: map<string, dyn>` and `request.path == 'value'`
  * After: `request.path: string` and `request.path == 'value'`
* Unguarded map key or list index accesses:
  * Before: `m[k]`, After: `k in m && m[k]` or `m[?k].orValue(<default>)`
  * Before: `l[i]`, After: `l.size() > i && l[i]` or `l[?i].orValue(<default>)`
Note: the `?` syntax requires enabling `optional` extension in the `envConfig`.

The coverage report for `cel_evaluate` indicates the node and branch coverage
percentages as a pre-formatted string (e.g., `"Node: %.2f%%, Branch: %.2f%%"`).
Generate additional test cases until you achieve >95% node and branch coverage.

Report the test results with node and branch coverage percentages.

## Complete Example

The following is a complete example of the artifacts generated from the prompt:
"Validate the request is authenticated by checking for a bearer token"

*   Environment:
    [network_env.json](google3/third_party/cel/skills/skills/cel_authoring/examples/network_env.json)
*   Expression:
    [network_headers.cel](google3/third_party/cel/skills/skills/cel_authoring/examples/network_headers.cel)
*   Tests:
    [network_headers_tests.json](google3/third_party/cel/skills/skills/cel_authoring/examples/network_headers_tests.json)
