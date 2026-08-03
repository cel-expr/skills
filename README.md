# CEL Skills

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

