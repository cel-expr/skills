# Google Common Expression Language (CEL) Extension

This extension provides the Gemini CLI with tools and skills to author, test,
and debug CEL expressions.

## Provided Tools

The `cel` MCP server provides the following tools:

- `cel_create_environment`: Defines the variables, functions, types for an
   expression.
- `cel_generate_prompt`: Generates an authoring prompt for an expression based
   on the configuration and requirement.
- `cel_compile`: Compiles a CEL expression to validate syntax, correctness, and
   type checking against an environment definition.
- `cel_evaluate`: Evaluates a compiled expression against provided test cases.

## Available Skills

To understand how to best use these tools, please refer to the `cel-authoring`,
`cel-testing`, and `cel-debugging` skills.
