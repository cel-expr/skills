package tools

import (
	"fmt"

	"github.com/google/cel-go/cel"
)

// GeneratePrompt generates an LLM authoring prompt explaining the exact variables and functions available.
func GeneratePrompt(envConfig *Config, userPrompt string) (string, error) {
	if envConfig == nil {
		return "", fmt.Errorf("envConfig cannot be nil")
	}
	env, err := EnvFromConfig(envConfig)
	if err != nil {
		return "", fmt.Errorf("EnvFromConfig(envConfig) failed: %v", err)
	}
	prompt, err := cel.AuthoringPrompt(env)
	if err != nil {
		return "", fmt.Errorf("cel.AuthoringPrompt(env) failed: %v", err)
	}
	return prompt.Render(userPrompt), nil
}
