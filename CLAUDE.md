# Common Expression Language (CEL) Skills

This repository provides tools and skills to author, test, and debug Common
Expression Language (CEL) expressions.

## Project Structure & Architecture

-   `cmd/mcp`: Entry point for the Model Context Protocol (MCP) server
    (`cel-expr-mcp`).
-   `internal/tools`: Internal implementation for environment loading,
    compilation, evaluation, and prompt generation.
-   `skills/`: Agent skills directory containing guidelines and workflows:
    -   `cel-authoring`: Authoring, configuring, and testing expressions and
        JSON environment definitions.
    -   `cel-debugging`: Diagnosing and resolving compilation and evaluation
        errors.

## Build and Test Commands

### Go CLI

-   **Build MCP server**: `go build ./cmd/mcp`
-   **Run tests**: `go test ./...`
-   **Run MCP server**: `go run ./cmd/mcp`

### Bazel

-   **Build all targets**: `bazel build //...`
-   **Run all tests**: `bazel test //...`

## Provided Tools

The `cel-expr-mcp` MCP server provides the following tools for AI agents:

-   `cel_create_environment`: Defines the variables, functions, and types for a
    CEL expression environment.
-   `cel_generate_prompt`: Generates an authoring prompt for a CEL expression
    based on the configuration and requirement.
-   `cel_compile`: Compiles a CEL expression to validate syntax, correctness,
    and type checking against an environment definition.
-   `cel_evaluate`: Evaluates a compiled expression against provided test cases.

### Startup Flags

The server accepts startup flags to tailor the tool experience:

-   `-environment` (alias `-env`): Path to a JSON file or JSON string containing
    the CEL environment configuration.
-   `-file_descriptor_set` (alias `-file_descriptors`): Path to a binary
    `FileDescriptorSet` file containing protobuf definitions referenced as types
    within an environment.

## Available Skills

To understand how to best use these tools, refer to the skills in `skills/`:

-   `cel-authoring`: Guidelines for creating CEL environment configurations,
    generating expressions, and compiling.
-   `cel-debugging`: Diagnostic steps for troubleshooting syntax, type, and
    evaluation errors.

## Release & Supply Chain Security

Releases are built for multiple platforms with Sigstore Cosign signing and SLSA Level 3 provenance:

-   Release Workflow: `.github/workflows/release.yml`
-   Signing: Keyless Cosign signatures (bundle, certificate, and signature) generated via GitHub OIDC token
-   Provenance: Generated via `slsa-framework/slsa-github-generator` (generic generator)
