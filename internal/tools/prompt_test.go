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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePrompt(t *testing.T) {
	envConfig := &Config{
		Variables: []*Variable{
			{Name: "foo", Type: "string"},
		},
	}

	tests := []struct {
		name       string
		envConfig  *Config
		userPrompt string
		wantInRes  []string
		wantErr    bool
	}{
		{
			name:       "valid prompt",
			envConfig:  envConfig,
			userPrompt: "check if foo is bar",
			wantInRes:  []string{"foo", "string", "check if foo is bar"},
			wantErr:    false,
		},
		{
			name:       "nil env config",
			envConfig:  nil,
			userPrompt: "check if foo is bar",
			wantInRes:  nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GeneratePrompt(tt.envConfig, tt.userPrompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				for _, want := range tt.wantInRes {
					if !strings.Contains(got, want) {
						t.Errorf("GeneratePrompt() result missing %q", want)
					}
				}
			}
		})
	}
}

func TestGeneratePromptTestData(t *testing.T) {
	// 1. Read testdata/cloud_armor.json
	envData, err := os.ReadFile(filepath.Join("testdata", "cloud_armor.json"))
	if err != nil {
		t.Fatalf("failed to read env config: %v", err)
	}
	var envConfig Config
	if err := json.Unmarshal(envData, &envConfig); err != nil {
		t.Fatalf("failed to unmarshal env config: %v", err)
	}
	userPrompt := "Allow traffic from 10.0.0.0/8"
	// 2. Call GeneratePrompt
	got, err := GeneratePrompt(&envConfig, userPrompt)
	if err != nil {
		t.Fatalf("GeneratePrompt() error = %v", err)
	}

	if !strings.Contains(got, "Allow traffic from 10.0.0.0/8") {
		t.Errorf("GeneratePrompt() prompot missing %q", got)
	}
	if !strings.Contains(got, "origin.ip") {
		t.Errorf("GeneratePrompt() attribute missing %q", got)
	}
	if !strings.Contains(got, "inIpRange") {
		t.Errorf("GeneratePrompt() function missing %q", got)
	}
}
