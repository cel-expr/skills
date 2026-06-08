---
name: cel-authoring
description: >-
  Skill for authoring Google Common Expression Language (CEL) expressions.
  Use to configure and write a new policy or CEL rule.
---

# Google Common Expression Language (CEL) Authoring Skill

Use this skill to author CEL expressions, define environments (variables,
functions, types) via JSON configuration, test, and debug.

## Workflow

Follow these steps to author a CEL expression:

*   **Collect Requirements** - Determine what the CEL expressions need to
    accomplish: security, object transformation, filtering / routing?
*   **Determine the Environment** - Identify the variables, functions, and CEL
    extensions needed to satisfy the requirements, reusing an `{ENV}.json`
    config or generating a new one with the `cel_create_environment` tool.
*   **Generate an Authoring Prompt** - Generate an authoring prompt specific to
    the environment using the `cel_generate_prompt` tool.
*   **Generate an Expression** - Use the prompt to generate an expression, and
    validate it with the `cel_compile` tool.

### 1. Collect Requirements

Determine the use case, requirements, and relevant products.

If the following products are mentioned, use the following techniques to
determine variables and functions available:

-   **Google Cloud** - Query
    [Cloud Documentation](https://docs.cloud.google.com/docs)
-   **Kubernetes** - Read
    [CEL in Kubernetes](https://kubernetes.io/docs/reference/using-api/cel/)
-   Otherwise, use the built-in `googleSearch` tool to learn more.

### 2. Determine the Environment

Determine the variables, functions, and
[extensions](https://github.com/google/cel-go/tree/master/ext/README.md) needed
to satisfy the requirements. If an existing `{ENV}.json` file exists which meets
the needs exists, prefer using it. If no such `{ENV}.json` exists, generate one
and use the `cel_create_environment` tool to validate the config.

See `examples/network_env.json` and `examples/user_env.json` for environment
examples. Type references within the environment followed EBNF grammar defined
in `references/type_grammar_ebnf.txt`.

Example types:

*   Simple types: `bool`, `bytes`, `double`, `dyn`, `int`, `null_type`,
    `string`, `uint`
*   Parameterized types: `list<string>`, `list<~V>`, `map<string, dyn>`,
    `map<~K,~V>`, `type<list<string>>`, `optional_type<int>`, `map<string,
    google.rpc.Status>`
*   Namespaced types: `google.protobuf.Duration`, `.google.rpc.Status`

### 3. Generate an Authoring Prompt

Generate the authoring prompt by calling the `cel_generate_prompt` tool with the
`{ENV}.json` content as `envConfig` and a summary of the user's requirement as
`userPrompt`.

### 4. Generate an Expression

Determine if you know enough to author an expression. If not, ask the user for
more information to address missing variables, types, functions, or extensions.
If so, provide a summarized overview of the expression behavior and its expected
output type.

Generate a prompt using the `cel_generate_prompt` tool and save the result to
`{ENV}.prompt` for future reference. Use the returned `{ENV}.prompt` to generate
the expression, `{EXPR}.cel`.

Validate the expression compiles using `cel_compile` tool, providing the
`{EXPR}.cel` as the `expr` argument and `{ENV}.json` as the `envConfig`
argument.

On success, proceed to the [cel-testing](../cel-testing/SKILL.md) skill. On
failure, consult the [cel-debugging](../cel-debugging/SKILL.md) skill.

--------------------------------------------------------------------------------

## CEL Syntax & General Principles

### General Principles

1.  **Keep it simple:** CEL is deliberately simple. It doesn't support loops,
    statements, or state modification. Expressions must evaluate to a value.
2.  **Type safety:** CEL is strongly typed. Ensure your values match the types
    expected by operators and functions.
3.  **Dot notation:** Use dot notation for accessing fields of messages or maps,
    e.g., `user.name`.

### Standard Type Literals

-   **bool**: `true`, `false`
-   **bytes**: `b"abc"`, `b"\x41\x42"`
-   **double**: `3.14`
-   **int**: `42`, `-10`
-   **uint**: `42u`
-   **list**: `[1, 2, 3]`
-   **map**: `{"key": "value"}`
-   **null_type**: `null`
-   **string**: `"hello"`, `'world'`,

    ```cel
    """use for
       multi-line"""
    ```

### Common Operators

-   **Logical:** `&&`, `||`, `!`
-   **Comparison:** `==`, `!=`, `<`, `<=`, `>`, `>=`
-   **Arithmetic:** `+`, `-`, `*`, `/`, `%`
-   **String and List Concat:** `+`
-   **Membership:** `in` (e.g., `1 in [1, 2, 3]`)

### Standard Macros

-   **`has(message.field)`**: Checks if a field is present and has a non-default
    value.
-   **`exists(e, predicate)`**: Returns true if *at least one* element `e` in
    the collection satisfies the predicate.
    -   Example: `users.exists(u, u.age >= 18)`
-   **`all(e, predicate)`**: Returns true if *all* elements `e` in the
    collection satisfy the predicate.
    -   Example: `users.all(u, u.isActive)`
-   **`exists_one(e, predicate)`**: Returns true if *exactly one* element `e` in
    the collection satisfies the predicate.
    -   Example: `devices.exists_one(d, d.isPrimary == true)`
-   **`map(e, transform)`**: Applies a transformation to each element in a
    collection, producing a new list.
    -   Example: `users.map(u, u.name)` (returns a list of names)
-   **`filter(e, predicate)`**: Returns a new collection containing only
    elements that satisfy the predicate.
    -   Example: `users.filter(u, u.age >= 18)`

<!-- mdformat off (backslashes in codespans are mangled) -->
### Formatting and Escaping

-   Use consistent spacing around operators (e.g., `a == b` not `a==b`).
-   When writing multi-line strings, use `"""`.
-   Remember to escape special characters in strings if necessary (e.g., `\n`,
    `\"`, `\\`).
<!-- mdformat on -->
