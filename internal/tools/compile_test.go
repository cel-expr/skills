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
	"testing"
)

func TestCompileCEL(t *testing.T) {
	envConfig := &Config{
		Name: "basic",
		Variables: []*Variable{
			{Name: "user", Type: "string"},
			{Name: "age", Type: "int"},
		},
	}

	tests := []struct {
		name      string
		expr      string
		envConfig *Config
		wantErr   bool
	}{
		{
			name:      "valid expression",
			expr:      `user == "Alice" && age > 18`,
			envConfig: envConfig,
			wantErr:   false,
		},
		{
			name:      "invalid expression syntax",
			expr:      `user == `,
			envConfig: envConfig,
			wantErr:   true,
		},
		{
			name:      "failed constructing env",
			expr:      `user == "Alice"`,
			envConfig: &Config{Variables: []*Variable{{Name: "a", Type: "invalid"}}}, // invalid type
			wantErr:   true,
		},
		{
			name:      "compile error",
			expr:      `invalid_var == "Alice"`,
			envConfig: envConfig,
			wantErr:   true,
		},
		{
			name:      "another failed constructing env (duplicate variable)",
			expr:      `user == "Alice"`,
			envConfig: &Config{Variables: []*Variable{{Name: "user", Type: "string"}, {Name: "user", Type: "int"}}}, // duplicate variable
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompileCEL(tt.expr, tt.envConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("CompileCEL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("CompileCEL() returned nil result without error")
			}
		})
	}
}
