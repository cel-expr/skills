# CEL Skills

Collection of skills and associated MCP server for working with CEL (Common
Expression Language).

To test your new skills in the Gemini CLI (or any MCP-compatible client), you
need to register the `cel-mcp` binary as an MCP server.

0.  Build the CEL MCP server

```bash
g4d -f cel-mcp-cli
blaze build //third_party/cel/skills/cmd/mcp
```

1.  Register the MCP Server

Add the following configuration to your mcp_config.json (usually located in
~/.gemini/settings.json):

```json
{
  "mcpServers": {
    "cel-skills": {
      // Use the absolute path to the binary if not in the same directory
      "command": "/google/src/cloud/{ldap}/{workspace}/third_party/cel/skills/cmd/mcp/mcp",
      "args": [],
      "env": {}
    }
  }
}
```

2.  Verify Skill Location

The Gemini CLI looks for skills in the `.agents/skills` (or `_agent/skills`)
directory relative to your workspace root. The agent will automatically discover
your skills when you start a session in `third_party/cel/skills`.

3.  Testing the Skill Once configured, you can invoke the skills by their name.
    For example, to test the authoring skill, you can ask the CLI:

"Use the cel-authoring skill to create a policy that checks if a user's age is
over 18."

The agent will then follow the updated workflow in your SKILL.md, calling
`cel_create_environment`, `cel_generate_prompt`, and `cel_compile` as needed.