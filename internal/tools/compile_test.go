package tools

import (
	"testing"
)

func TestCompileCEL(t *testing.T) {
	envConfig := &Config{
		Name: "basic",
		Variables: []*Variable{
			{Name: "user", TypeName: "string"},
			{Name: "age", TypeName: "int"},
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
			envConfig: &Config{Variables: []*Variable{{Name: "a", TypeName: "invalid"}}}, // invalid type
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
			envConfig: &Config{Variables: []*Variable{{Name: "user", TypeName: "string"}, {Name: "user", TypeName: "int"}}}, // duplicate variable
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
