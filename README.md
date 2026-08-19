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
    }
  ]
}
```

2.  Testing the Skill Once configured, you can invoke the skills by their name.
    For example, to test the authoring skill, you can ask your agent:

"Use the cel-authoring skill to create a policy that checks if a user's age is
over 18."

The agent will then follow the updated workflow in your SKILL.md, calling
`cel_create_environment`, `cel_generate_prompt`, and `cel_compile` as needed.

## Releases & Verification

Release binaries for `cel-expr-mcp` are built automatically on releases with SLSA Level 3 provenance and signed using Sigstore (Cosign) keyless signatures.

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
slsa-verifier verify-artifact <cel-expr-mcp-archive> \
  --provenance-path multiple.intoto.jsonl \
  --source-uri github.com/cel-expr/skills \
  --source-tag <tag>
```


