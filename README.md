# CEL Skills

[![SLSA 3](https://slsa.dev/images/gh-badge-level3.svg)](https://slsa.dev)
[![Go Reference](https://pkg.go.dev/badge/github.com/cel-expr/skills.svg)](https://pkg.go.dev/github.com/cel-expr/skills)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Collection of skills and associated MCP server for working with CEL (Common
Expression Language).

1.  Install and Configure Skills

To make these skills automatically discoverable and triggerable in Jetski/Gemini
Coder across any workspace, inherit the team-level configuration.

Add the following block to your personal configuration file at
`~/.gemini/skills.json` (create it if it doesn't exist), replacing
`<path-to-cel-skills>` with the path to your cloned repository:

```json
{
  "entries": [
    {
      "path": "<path-to-cel-skills>/skills/cel-authoring"
    },
    {
      "path": "<path-to-cel-skills>/skills/cel-debugging"
    },
    {
      "path": "<path-to-cel-skills>/skills/cel-testing"
    },
    {
      "path": "<path-to-cel-skills>/skills/cel-optimize"
    },
    {
      "path": "<path-to-cel-skills>/skills/cel-doc-generator"
    }
  ]
}
```

2.  Testing the Skill Once configured, you can invoke the skills by their name.
    For example, to test the authoring skill, you can ask your agent:

"Use the cel-authoring skill to create a policy that checks if a user's age is
over 18."

The agent will then follow the updated workflow in your SKILL.md, running
`cel-expr` CLI commands (`env`, `prompt`, `compile`, `eval`) as needed.

## CLI Tooling (`cel-expr`)

The skills use the standalone `cel-expr` CLI (`cmd/cel-expr`) to author, compile,
evaluate, and test CEL expressions:

- `cel-expr env -env <path|json>`: Validates environment configuration and type definitions.
- `cel-expr prompt -env <path|json> -prompt <requirement>`: Generates authoring prompt.
- `cel-expr compile -env <path|json> -expr <expr>`: Compiles and type-checks CEL expressions.
- `cel-expr eval -env <path|json> -expr <expr> -tests <path|json>`: Evaluates expressions against test cases and reports branch/node coverage.

### Installing `cel-expr`

Install using Go:

```bash
go install github.com/cel-expr/skills/cmd/cel-expr@latest
alias cel-expr="$(go env GOPATH)/bin/cel-expr"
```

Or build from source:

```bash
go build -o cel-expr ./cmd/cel-expr
```

Ensure `cel-expr` is on your `$PATH` (or aliased as above) so the agent can execute it.

## Formal Verification Tooling (`cel-verifier`)

The `cel-optimize` skill uses the standalone `cel-verifier` CLI to formally
prove semantic equivalence and invariants via Z3 SMT solving:

-   `cel-verifier verify-equiv --expr1 <expr> --expr2 <expr>`
    `--var <name>:<type>`: Proves logical equivalence between two CEL
    expressions across all inputs.
-   `cel-verifier check-sat --expr <expr> --var <name>:<type>`: Checks
    satisfiability and generates witness inputs.
-   `cel-verifier check-valid --expr <expr> --var <name>:<type>`: Proves
    validity (`isAlwaysTrue`) and generates counterexamples on violation.
-   `cel-verifier verify-policy --file <policy.yaml> --var <name>:<type>`:
    Verifies invariants on a YAML policy file.

### Installing `cel-verifier`

Configure the `cel-verifier` CLI alias:

```bash
# Download the cel-java verifier-cli JAR from Maven Central (requires Java 8+)
curl -fsSL --create-dirs -o "$HOME/.local/lib/cel/verifier-cli-0.14.0.jar" https://repo1.maven.org/maven2/dev/cel/verifier-cli/0.14.0/verifier-cli-0.14.0.jar
alias cel-verifier="java -jar $HOME/.local/lib/cel/verifier-cli-0.14.0.jar"
```

## Releases & Verification

Release binaries for `cel-expr` and `cel-expr-mcp` are built automatically on releases with
SLSA Level 3 provenance and signed using Sigstore (Cosign) keyless signatures.

### Verifying Signatures with Cosign

Using the Sigstore bundle:

```bash
cosign verify-blob \
  --bundle checksums.txt.bundle \
  --certificate-identity-regexp 'https://github.com/cel-expr/skills/.github/workflows/release.yml@refs/.*' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  checksums.txt
```

Or using the certificate and signature:

```bash
cosign verify-blob \
  --certificate checksums.txt.pem \
  --signature checksums.txt.sig \
  --certificate-identity-regexp 'https://github.com/cel-expr/skills/.github/workflows/release.yml@refs/.*' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  checksums.txt
```

Verify artifact hashes:

```bash
sha256sum --ignore-missing -c checksums.txt
```

### Verifying SLSA Level 3 Provenance

To verify the build provenance using `slsa-verifier`:

```bash
slsa-verifier verify-artifact <artifact-archive> \
  --provenance-path multiple.intoto.jsonl \
  --source-uri github.com/cel-expr/skills \
  --source-tag <tag>
```
