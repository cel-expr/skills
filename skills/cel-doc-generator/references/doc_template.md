# CEL Library Documentation Template

Use this reference template when generating or updating a `README.md` for a CEL
extension library from the perspective of a single target language (e.g.,
**Go**, **C++**, or **Java**), matching the canonical format established in
`third_party/cel/go/ext/README.md`.

---

## 1.Document Template

When adding or updating a library's `README.md`:

```markdown
## <LibraryName>

<Summary paragraph explaining what the library provides and its target use cases.>

<Language-specific enablement snippet and package import description.>

<Optional namespace note: e.g. "Note, all macros and functions use the '<namespace>' namespace. If you are currently using a variable named '<namespace>', the function will likely work just as intended; however, there is some chance for collision.">

### <Namespace>.<FunctionName>

**Introduced in version <version_number> (cost support in version <cost_version>)**

<Detailed paragraph explaining the function or macro's behavior, argument constraints, type conversions, and edge cases.>

    <receiver_type>.<function_name>(<param_1_type>, <param_2_type>) -> <return_type>
    <namespace>.<function_name>(<param_1_type>, <param_2_type>) -> <return_type>

Examples:

    <namespace>.<function_name>(<arg1>, <arg2>) // <return_value>
    <receiver>.<function_name>(<arg1>)         // <return_value>
    <namespace>.<function_name>(<bad_arg>)     // error

### <TypeName>

*   **Description**: <Explanation of the custom CEL type.>
*   **Field Selection**: `<type>.<field>` returns `<field_type>`.
*   **Methods**: `<type>.<method>()` returns `<return_type>`.
```

## 2. Formatting Rules Reference

### 2.1 Headings

*   **H2 (`## <LibraryName>`)**: Capitalized library name (e.g., `## Math`, `## Strings`, `## Encoders`, `## JWT`).
*   **H3 (`### <Namespace>.<FunctionName>`)**: Capitalized namespace and function name in title case (e.g., `### Math.Greatest`, `### Base64.Decode`, `### JWT.Parse`, `### CharAt`).

### 2.2 Version Annotations

Use one of the two canonical formats right below the H3 heading:

*   `**Introduced in version <V> (cost support in version <C>)**`
*   `Introduced at version: <V>`

### 2.3 Signature Blocks

*   Enclose in standard triple backticks without a language specifier (` ``` `).
*   Use `<type>` angle bracket notation for parameters:
    *   Receiver syntax: `<string>.charAt(<int>) -> <string>`
    *   Global / Namespaced syntax: `math.greatest(<arg>, ...) -> <double|int|uint>`
    *   Comprehension syntax: `<list>.all(indexVar, valueVar, <predicate>) -> bool`
*   Multiple overloads should be listed on consecutive lines.

### 2.4 Example Blocks

*   Enclose in triple backticks with standard trailing inline comments:
    *   Successful returns: `// <value>` or `// returns <value>` (e.g., `// returns 42`, `// return b'hello'`)
    *   Boolean comparison: `// returns true` or `== true`
    *   Errors: `// error`, `// parse error`, `// check-time error`, `// runtime error`
