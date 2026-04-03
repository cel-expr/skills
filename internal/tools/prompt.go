// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
