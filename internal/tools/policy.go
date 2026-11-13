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
	"errors"
	"fmt"
	"os"
	"strings"

	"cel.dev/cel-go/cel"
	celext "cel.dev/cel-go/ext"
	"cel.dev/cel-go/policy"
	"go.yaml.in/yaml/v3"
)

// LoadPolicySource loads a .celpolicy YAML document from a file path if it exists, or treats
// exprOrPath as inline policy YAML.
func LoadPolicySource(exprOrPath string) (string, string, error) {
	data, err := os.ReadFile(exprOrPath)
	if err == nil {
		return string(data), exprOrPath, nil
	}
	if !strings.Contains(exprOrPath, "\n") && !strings.Contains(exprOrPath, ":") {
		return "", "", fmt.Errorf("reading policy file %q: %w", exprOrPath, err)
	}
	if !errors.Is(err, os.ErrNotExist) && !strings.Contains(exprOrPath, "\n") {
		return "", "", fmt.Errorf("reading policy file %q: %w", exprOrPath, err)
	}
	return exprOrPath, "<inline>", nil
}

type lenientPolicyTagVisitor struct {
	base     policy.TagVisitor
	warnings []string
	seen     map[string]bool
}

func (v *lenientPolicyTagVisitor) PolicyTag(ctx policy.ParserContext, id int64, tagName string, node *yaml.Node, p *policy.Policy) {
	if tagName == "description" {
		return
	}
	if node != nil {
		p.SetMetadata(tagName, node.Value)
	}
	if !v.seen[tagName] {
		v.seen[tagName] = true
		v.warnings = append(v.warnings, fmt.Sprintf("policy: ignoring unsupported policy tag %q", tagName))
	}
}

func (v *lenientPolicyTagVisitor) RuleTag(ctx policy.ParserContext, id int64, tagName string, node *yaml.Node, p *policy.Policy, r *policy.Rule) {
	v.base.RuleTag(ctx, id, tagName, node, p, r)
}

func (v *lenientPolicyTagVisitor) MatchTag(ctx policy.ParserContext, id int64, tagName string, node *yaml.Node, p *policy.Policy, m *policy.Match) {
	v.base.MatchTag(ctx, id, tagName, node, p, m)
}

func (v *lenientPolicyTagVisitor) VariableTag(ctx policy.ParserContext, id int64, tagName string, node *yaml.Node, p *policy.Policy, pv *policy.Variable) {
	v.base.VariableTag(ctx, id, tagName, node, p, pv)
}

// ParsePolicy parses a .celpolicy YAML document into a canonical policy.Policy. Unknown top-level
// policy metadata tags (such as display_name) are recorded as warnings rather than errors.
func ParsePolicy(content, location string) (*policy.Policy, []string, error) {
	if location == "" {
		location = "<inline>"
	}
	visitor := &lenientPolicyTagVisitor{
		base: policy.DefaultTagVisitor(),
		seen: make(map[string]bool),
	}
	parser, err := policy.NewParser(policy.CustomTagVisitor(visitor))
	if err != nil {
		return nil, nil, err
	}
	p, iss := parser.Parse(policy.StringSource(content, location))
	if iss.Err() != nil {
		return nil, visitor.warnings, iss.Err()
	}
	return p, visitor.warnings, nil
}

func compilePolicy(exprOrPath string, env *cel.Env, opts CompileOptions) (*cel.Env, *cel.Ast, []string, error) {
	content, loc, err := LoadPolicySource(exprOrPath)
	if err != nil {
		return nil, nil, nil, err
	}
	if opts.Location != "" {
		loc = opts.Location
	}

	pol, warnings, err := ParsePolicy(content, loc)
	if err != nil {
		return nil, nil, warnings, fmt.Errorf("policy parse error: %w", err)
	}

	// Policy composition uses optional.of/optional.none for partial rules and cel.@block for
	// policy variables. Extending after EnvFromConfig adds OptionalTypes when absent while
	// preserving any optional version already declared in envConfig via SingletonLibrary.
	env, err = env.Extend(cel.OptionalTypes(), celext.Bindings())
	if err != nil {
		return nil, nil, warnings, fmt.Errorf("failed configuring policy environment: %w", err)
	}

	astVal, iss := policy.Compile(env, pol)
	if iss.Err() != nil {
		return nil, nil, warnings, fmt.Errorf("compile error: %w", iss.Err())
	}
	return env, astVal, warnings, nil
}
