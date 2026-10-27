# Common Expression Language (CEL) Skills

This repository provides tools and skills to author, test, debug, and optimize
CEL expressions.

## Provided Tools

### `cel-expr` CLI (Recommended)

The standalone `cel-expr` CLI requires no special agent configuration and runs directly via standard terminal commands:

- `cel-expr env -env <path|json>`: Defines/validates variables, functions, and types.
- `cel-expr prompt -env <path|json> -prompt <requirement>`: Generates authoring prompt.
- `cel-expr compile -env <path|json> -expr <expr>`: Compiles and type-checks CEL expressions.
- `cel-expr eval -env <path|json> -expr <expr> -tests <path|json>`: Evaluates expressions against test cases.

In open source, build the CLI with `go build -o cel-expr ./cmd/cel-expr` or install via `go install github.com/cel-expr/skills/cmd/cel-expr@latest` (and `alias cel-expr="$(go env GOPATH)/bin/cel-expr"` if `$(go env GOPATH)/bin` is not on your `PATH`).

### `cel-verifier` CLI

The `cel-verifier` CLI provides formal Z3 SMT verification for CEL expressions
and policies:

-   `cel-verifier verify-equiv --expr1 <expr> --expr2 <expr>`
    `--var <name>:<type>`: Proves logical equivalence between two CEL
    expressions across all inputs.
-   `cel-verifier check-sat --expr <expr> --var <name>:<type>`: Checks
    satisfiability and generates witness inputs.
-   `cel-verifier check-valid --expr <expr> --var <name>:<type>`: Proves
    validity (`isAlwaysTrue`) and generates counterexamples on violation.
-   `cel-verifier verify-policy --file <policy.yaml> --var <name>:<type>`:
    Verifies invariants on a YAML policy file.

```bash
# Download the cel-java verifier-cli JAR from Maven Central (requires Java 8+)
curl -fsSL --create-dirs -o "$HOME/.local/lib/cel/verifier-cli-0.14.0.jar" https://repo1.maven.org/maven2/dev/cel/verifier-cli/0.14.0/verifier-cli-0.14.0.jar
alias cel-verifier="java -jar $HOME/.local/lib/cel/verifier-cli-0.14.0.jar"
```

### `cel-expr-mcp` MCP Server

The `cel-expr-mcp` MCP server provides matching tools (`cel_create_environment`,
`cel_generate_prompt`, `cel_compile`, `cel_evaluate`) for MCP-compatible clients.

The server accepts startup flags to tailor the tool experience:

- `-environment` (alias `-env`): Path to a JSON file or JSON string containing
   the CEL environment configuration.
- `-file_descriptor_set` (alias `-file_descriptors`): Path to a binary
   `FileDescriptorSet` file containing protobuf definitions referenced as types
   within an environment.

## Available Skills

To understand how to best use these tools, please refer to the `cel-authoring`,
`cel-debugging`, `cel-testing`, and `cel-optimize` skills, which will be
automatically available to the agent when this configuration is loaded.
