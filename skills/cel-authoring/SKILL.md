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

### 1. Collect the Requirements

Ask the user about the kind of expression they want to create to determine the
variables and functions required to satisfy the request. This will inform how
the environment and the user's CEL expressions are created.

Information about the product being used may help inform the variables and
functions required:

-   For Google Cloud Products, query the Developer Knowledge MCP tool for
    supported variables and functions.
-   For Kubernetes policies, search kubernetes.io and GitHub for supported
    variables and functions.
-   Otherwise, offer to search Google for relevant concepts using the
    `googleSearch` tool.

### 2. Define the Environment

Create a `{ENV}.json` file matching the `cel_create_environment` tool's
`envConfig` argument schema with the variables, functions, and extensions
required to satisfy the request. Use the `cel_create_environment` tool to
validate the environment before continuing to the next step.

Types use the following syntax:

```
  TypeDesc            = NamespaceIdentifier [ "<" TypeList ">" ] ;
  NamespaceIdentifier = [ "." ] Identifier { "." Identifier } ;
  TypeList            = TypeElem { "," TypeElem } ;
  TypeElem            = TypeDesc | TypeParam
  TypeParam           = "~" Alpha ;
  Identifier          = ( Alpha | "_" ) { AlphaNumeric | "_" } ;

  (* Terminals *)
  Alpha               = "a"..."z" | "A"..."Z" ;
  Digit               = "0"..."9" ;
  AlphaNumeric        = Alpha | Digit ;
```

Examples:

* Simple types: `bool`, `bytes`, `double`, `dyn`, `int`, `null_type`, `string`,
  `uint`
* Parameterized types: `list<string>`, `list<~V>`, `map<string, dyn>`,
  `map<~K,~V>`, `type<list<string>>`, `optional_type<int>`,
  `map<string, google.rpc.Status>`
* Namespaced types: `google.protobuf.Duration`, `.google.rpc.Status`

These types may be combined into more complex type descriptions as needed, e.g.

```
// Optional type containing a map with string keys and protobuf values.
optional_type<map<string,google.rpc.context.AttributeContext.Resource>>
```

Reference examples in `examples/` for suggestions on documentation and structure
of the environment:

-   `examples/network_request_env.json`

CEL extension documentation:
https://github.com/google/cel-go/tree/master/ext/README.md

### 3. Generate an Authoring Prompt

Generate the authoring prompt by calling the `cel_generate_prompt` tool with the
`{ENV}.json` content as `envConfig` and the user's requirement as `userPrompt`.

### 4. Generate an Expression

Given the environment, determine if you know enough about the expression the
user wants to write. If not, ask the user for more information, and provide a
summarized overview of the expression behavior and its expected output type, if
one is provided.

Generate a prompt using the `cel_generate_prompt` tool and save the result to
`{ENV}.prompt` for future reference. Use the returned `{ENV}.prompt` to generate
the expression, `{EXPR}.cel`.

Validate the generated expression compiles by calling the `cel_compile` tool,
providing the generated expression in `{EXPR}.cel` as the `expr` argument and
the environment definition in `{ENV}.json` as the `envConfig` argument.

On success, proceed to the [cel-testing](../cel_testing/SKILL.md) skill. On
failure, consult the [cel-debugging](../cel_debugging/SKILL.md) skill.

---

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
```
    """use for
       multi-line"""
```

### Common Operators

-   **Logical:** `&&`, `||`, `!`
-   **Comparison:** `==`, `!=`, `<`, `<=`, `>`, `>=`
-   **Arithmetic:** `+`, `-`, `*`, `/`, `%`
-   **String and List Concat:** `+`
-   **Membership:** `in` (e.g., `1 in [1, 2, 3]`)

### Standard Macros and Functions

CEL provides utilities for working with collections and strings:

#### Collections (Lists and Maps)

-   **`list.size()`**: Returns the size of the list.
-   **`map.size()`**: Returns the number of entries in the map.
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

#### Strings

-   **`string.size()`**: Returns the number of code points the string.
-   **`string.startsWith(prefix)`**: Checks if the string starts with the
    prefix.
-   **`string.endsWith(suffix)`**: Checks if the string ends with the suffix.
-   **`string.contains(substring)`**: Checks if the string contains the
    substring.
-   **`string.matches(regex)`**: Checks if the string matches the RE2 regular
    expression.

### Formatting and Escaping

-   Use consistent spacing around operators (e.g., `a == b` not `a==b`).
-   When writing multi-line strings, use `"""`.
-   Remember to escape special characters in strings if necessary (e.g., `\n`,
    `\"`, `\\`).

### Example Expressions

**Basic Conditional:**

```cel
user.age >= 18 && user.country == "US"
```

**Collection Filtering and Macro:**

```cel
request.headers.exists(h, h.name == "Authorization" && h.value.startsWith("Bearer "))
```

**Checking Field Presence:**

```cel
has(request.payload.id) && request.payload.id != ""
```