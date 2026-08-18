# Common Expression Language (CEL) Skills

This repository provides tools and skills to author and debug CEL expressions.

## Provided Tools

The `cel-expr-mcp` MCP server provides the following tools:

- `cel_create_environment`: Defines the variables, functions, types for an
   expression.
- `cel_generate_prompt`: Generates an authoring prompt for an expression based
   on the configuration and requirement.
- `cel_compile`: Compiles a CEL expression to validate syntax, correctness, and
   type checking against an environment definition.
- `cel_evaluate`: Evaluates a compiled expression against provided test cases.

The server accepts startup flags to tailor the tool experience:

- `-environment` (alias `-env`): Path to a JSON file or JSON string containing
   the CEL environment configuration.
- `-file_descriptor_set` (alias `-file_descriptors`): Path to a binary
   `FileDescriptorSet` file containing protobuf definitions referenced as types
   within an environment.

## Available Skills

To understand how to best use these tools, please refer to the `cel_authoring`
and `cel_debugging` skills, which will be automatically available to the agent
when this extension is loaded.
