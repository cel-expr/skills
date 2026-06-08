# CEL Skills

Collection of skills and associated MCP server for working with CEL (Common
Expression Language).

The Gemini CLI looks for skills in the `.agents/skills` (or `_agent/skills`)
directory relative to your workspace root. 

Once configured, you can invoke the skills by their name. For example, to test the authoring skill, you can ask Gemini CLI:

```
/cel:cel-authoring create a policy that checks if a user's age is
over 18.
```

The agent will then follow the updated workflow in your SKILL.md, calling
`cel_create_environment`, `cel_generate_prompt`, and `cel_compile` as needed.
