# CEL Skills

Collection of skills and associated MCP server for working with CEL (Common
Expression Language).

1.  Install the extension

To install the stable version of CEL skills run the following command line:

```
gemini extension link /google/src/files/head/depot/google3/third_party/cel/skills
```

To install the workspace version of CEL skills run this instead:

```
gemini extensions uninstall cel && \
gemini extensions link "$(blaze info workspace)/third_party/cel/skills"
```

2.  Install and Configure Skills

To make these skills automatically discoverable and triggerable in Jetski/Gemini
Coder across any workspace, inherit the team-level configuration.

Add the following block to your personal configuration file at
`configs/users/<username>/_agents/skills.json` (create it if it doesn't exist):

```json
{
  "inherits": [
    {
      "path": "google3/third_party/cel/skills/_agents/skills.json"
    }
  ]
}
```

3.  Testing the Skill Once configured, you can invoke the skills by their name.
    For example, to test the authoring skill, you can ask the CLI:

"Use the cel-authoring skill to create a policy that checks if a user's age is
over 18."

The agent will then follow the updated workflow in your SKILL.md, calling
`cel_create_environment`, `cel_generate_prompt`, and `cel_compile` as needed.

## Evaluation

Use go/evalin to evaluate skill activation and behavior. Since CEL uses an MCP
server make sure to use the `--agent=third_party/cel/skills/config.yaml` agent
configuration that enables the stdio MCP server before testing the skills.

```sh
alias evalin='/google/bin/releases/gemini-agents-evalin/evalin.par'

evalin run third_party/cel/skills/skills/cel_debugging/EVAL.txtpb \
  --agent=third_party/cel/skills/config.yaml
```