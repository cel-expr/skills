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

// Package tools provides the implementation of the cel-skills library.
package tools

import (
	"encoding/json"
	"fmt"

	"github.com/google/cel-go/cel"
	celenv "github.com/google/cel-go/common/env"
	celext "github.com/google/cel-go/ext"
)

// Config is a configuration for a cel-go Env.
type Config struct {
	Name            string           `json:"name,omitempty"`
	Description     string           `json:"description,omitempty"`
	Container       string           `json:"container,omitempty"`
	Imports         []*Import        `json:"imports,omitempty"`
	StdLib          *LibrarySubset   `json:"stdlib,omitempty"`
	Extensions      []*Extension     `json:"extensions,omitempty"`
	ContextVariable *ContextVariable `json:"contextVariable,omitempty"`
	Variables       []*Variable      `json:"variables,omitempty"`
	Functions       []*Function      `json:"functions,omitempty"`
	Validators      []*Validator     `json:"validators,omitempty"`
	Features        []*Feature       `json:"features,omitempty"`
}

// ToCELConfig converts a Config to a celenv.Config.
func (c *Config) ToCELConfig() (*celenv.Config, error) {
	if c == nil {
		return nil, nil
	}
	res := celenv.NewConfig(c.Name)
	res.Description = c.Description
	res.Container = c.Container
	for _, imp := range c.Imports {
		res.Imports = append(res.Imports, imp.ToCELImport())
	}
	if c.StdLib != nil {
		res.StdLib = c.StdLib.ToCELLibrarySubset()
	}
	for _, ext := range c.Extensions {
		res.Extensions = append(res.Extensions, ext.ToCELExtension())
	}
	if c.ContextVariable != nil {
		celCtxVar, err := c.ContextVariable.ToCELContextVariable()
		if err != nil {
			return nil, err
		}
		res.ContextVariable = celCtxVar
	}
	for _, v := range c.Variables {
		celVar, err := v.ToCELVariable()
		if err != nil {
			return nil, err
		}
		res.Variables = append(res.Variables, celVar)
	}
	for _, f := range c.Functions {
		celFunc, err := f.ToCELFunction()
		if err != nil {
			return nil, err
		}
		res.Functions = append(res.Functions, celFunc)
	}
	for _, v := range c.Validators {
		res.Validators = append(res.Validators, v.ToCELValidator())
	}
	for _, f := range c.Features {
		res.Features = append(res.Features, f.ToCELFeature())
	}
	return res, res.Validate()
}

// Import is an import for a cel-go Env.
type Import struct {
	Name string `json:"name"`
}

// ToCELImport converts an Import to a celenv.Import.
func (i *Import) ToCELImport() *celenv.Import {
	if i == nil {
		return nil
	}
	return celenv.NewImport(i.Name)
}

// Variable is a variable for a cel-go Env.
type Variable struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type" jsonschema_description:"type name formatted as TypeName or namespace.TypeName with an optional set of type parameters in angle brackets <>"`
}

// ToCELVariable converts a Variable to a celenv.Variable.
func (v *Variable) ToCELVariable() (*celenv.Variable, error) {
	if v == nil {
		return nil, nil
	}
	td, err := parseType(v.Type)
	if err != nil {
		return nil, err
	}
	return &celenv.Variable{
		Name:        v.Name,
		Description: v.Description,
		TypeDesc:    td,
	}, nil
}

// ContextVariable is a context variable for a cel-go Env.
type ContextVariable struct {
	Type string `json:"type"`
}

// ToCELContextVariable converts a ContextVariable to a celenv.ContextVariable.
func (c *ContextVariable) ToCELContextVariable() (*celenv.ContextVariable, error) {
	if c == nil {
		return nil, nil
	}
	td, err := parseType(c.Type)
	if err != nil {
		return nil, err
	}
	if len(td.Params) != 0 {
		return nil, fmt.Errorf("context variable cannot have type parameters")
	}
	return &celenv.ContextVariable{TypeName: td.TypeName}, nil
}

// Function is a function for a cel-go Env.
type Function struct {
	Name        string      `json:"name" jsonschema_description:"camelCase function name, either as a standalone functionName or namespace.functionName"`
	Description string      `json:"description,omitempty"`
	Overloads   []*Overload `json:"overloads"`
}

// ToCELFunction converts a Function to a celenv.Function.
func (f *Function) ToCELFunction() (*celenv.Function, error) {
	if f == nil {
		return nil, nil
	}
	var celOverloads []*celenv.Overload
	for _, o := range f.Overloads {
		celOverload, err := o.ToCELOverload()
		if err != nil {
			return nil, err
		}
		celOverloads = append(celOverloads, celOverload)
	}
	if f.Description != "" {
		return celenv.NewFunctionWithDoc(f.Name, f.Description, celOverloads...), nil
	}
	return celenv.NewFunction(f.Name, celOverloads...), nil
}

// Overload is an overload for a cel-go Env.
type Overload struct {
	ID       string   `json:"id" jsonschema_description:"overload ID in the format of function_name_type1_..._typeN for global functions and target_type_function_name_type1_..._typeN for member functions"`
	Examples []string `json:"examples,omitempty"`
	Target   string   `json:"target,omitempty" jsonschema_description:"receiver type name for member functions"`
	Args     []string `json:"args,omitempty" jsonschema_description:"argument type names"`
	Return   string   `json:"return" jsonschema_description:"return type name"`
}

// ToCELOverload converts an Overload to a celenv.Overload.
func (o *Overload) ToCELOverload() (*celenv.Overload, error) {
	if o == nil {
		return nil, nil
	}
	var args []*celenv.TypeDesc
	for _, a := range o.Args {
		td, err := parseType(a)
		if err != nil {
			return nil, err
		}
		args = append(args, td)
	}
	var ret *celenv.TypeDesc
	var err error
	if o.Return != "" {
		ret, err = parseType(o.Return)
		if err != nil {
			return nil, err
		}
	}
	var target *celenv.TypeDesc
	if o.Target != "" {
		target, err = parseType(o.Target)
		if err != nil {
			return nil, err
		}
	}

	if target != nil {
		return celenv.NewMemberOverload(o.ID, target, args, ret, o.Examples...), nil
	}
	return celenv.NewOverload(o.ID, args, ret, o.Examples...), nil
}

// Extension is an extension for a cel-go Env.
type Extension struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ToCELExtension converts an Extension to a celenv.Extension.
func (e *Extension) ToCELExtension() *celenv.Extension {
	if e == nil {
		return nil
	}
	return &celenv.Extension{Name: e.Name, Version: e.Version}
}

// LibrarySubset is a library subset for a cel-go Env.
type LibrarySubset struct {
	Disabled         bool              `json:"disabled,omitempty"`
	DisableMacros    bool              `json:"disableMacros,omitempty"`
	IncludeMacros    []string          `json:"includeMacros,omitempty"`
	ExcludeMacros    []string          `json:"excludeMacros,omitempty"`
	IncludeFunctions []*FunctionSubset `json:"includeFunctions,omitempty"`
	ExcludeFunctions []*FunctionSubset `json:"excludeFunctions,omitempty"`
}

// ToCELLibrarySubset converts a LibrarySubset to a celenv.LibrarySubset.
func (l *LibrarySubset) ToCELLibrarySubset() *celenv.LibrarySubset {
	if l == nil {
		return nil
	}
	res := celenv.NewLibrarySubset()
	res.Disabled = l.Disabled
	res.DisableMacros = l.DisableMacros
	res.IncludeMacros = append([]string{}, l.IncludeMacros...)
	res.ExcludeMacros = append([]string{}, l.ExcludeMacros...)
	for _, f := range l.IncludeFunctions {
		res.IncludeFunctions = append(res.IncludeFunctions, f.ToCELFunction())
	}
	for _, f := range l.ExcludeFunctions {
		res.ExcludeFunctions = append(res.ExcludeFunctions, f.ToCELFunction())
	}
	return res
}

// FunctionSubset is a function subset for a cel-go Env.
type FunctionSubset struct {
	Name        string   `json:"name"`
	OverloadIDs []string `json:"overloads,omitempty"`
}

// ToCELFunction converts a FunctionSubset to a celenv.FunctionSubset.
func (f *FunctionSubset) ToCELFunction() *celenv.Function {
	if f == nil {
		return nil
	}
	res := celenv.NewFunction(f.Name)
	for _, id := range f.OverloadIDs {
		res.Overloads = append(res.Overloads, celenv.NewOverload(id, nil, nil))
	}
	return res
}

// Validator is a validator for a cel-go Env.
type Validator struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config,omitempty"`
}

// ToCELValidator converts a Validator to a celenv.Validator.
func (v *Validator) ToCELValidator() *celenv.Validator {
	if v == nil {
		return nil
	}
	return celenv.NewValidator(v.Name).SetConfig(v.Config)
}

// Feature is a feature for a cel-go Env.
type Feature struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// ToCELFeature converts a Feature to a celenv.Feature.
func (f *Feature) ToCELFeature() *celenv.Feature {
	if f == nil {
		return nil
	}
	return celenv.NewFeature(f.Name, f.Enabled)
}

// ConfigFromJSON converts a JSON string to a Config.
func ConfigFromJSON(configJSON string) (*Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(configJSON) failed: %v", err)
	}
	return &config, nil
}

// EnvFromConfig takes a Config and converts it to a cel-go Env.
func EnvFromConfig(envConfig *Config) (*cel.Env, error) {
	celConfig, err := envConfig.ToCELConfig()
	if err != nil {
		return nil, err
	}
	return cel.NewEnv(cel.FromConfig(celConfig, celext.ExtensionOptionFactory))
}

type typeParser struct {
	text   string
	pos    int
	length int
}

func parseType(text string) (*celenv.TypeDesc, error) {
	p := &typeParser{text: text, length: len(text)}
	res, err := p.parseTypeDesc()
	if err != nil {
		return nil, fmt.Errorf("failed to parse type %q: %v", text, err)
	}
	p.skipWhitespace()
	if p.pos < p.length {
		return nil, fmt.Errorf("unexpected character %q at position %d in %q", p.text[p.pos], p.pos, text)
	}
	return res, nil
}

func (p *typeParser) parseTypeDesc() (*celenv.TypeDesc, error) {
	id, err := p.parseNamespaceIdentifier()
	if err != nil {
		return nil, err
	}
	if p.pos < p.length && p.text[p.pos] == '<' {
		p.pos++ // consume '<'
		var params []*celenv.TypeDesc
		for {
			p.skipWhitespace()
			param, err := p.parseTypeElem()
			if err != nil {
				return nil, err
			}
			params = append(params, param)
			p.skipWhitespace()
			if p.pos < p.length && p.text[p.pos] == ',' {
				p.pos++ // consume ','
				continue
			}
			if p.pos < p.length && p.text[p.pos] == '>' {
				p.pos++ // consume '>'
				break
			}
			return nil, fmt.Errorf("expected ',' or '>' at position %d", p.pos)
		}
		return celenv.NewTypeDesc(id, params...), nil
	}
	return celenv.NewTypeDesc(id), nil
}

func (p *typeParser) parseTypeElem() (*celenv.TypeDesc, error) {
	p.skipWhitespace()
	if p.pos < p.length && p.text[p.pos] == '~' {
		p.pos++ // consume '~'
		id, err := p.parseTypeParamIdent()
		if err != nil {
			return nil, err
		}
		return celenv.NewTypeParam(id), nil
	}
	return p.parseTypeDesc()
}

func (p *typeParser) parseNamespaceIdentifier() (string, error) {
	p.skipWhitespace()
	var id string
	for p.pos < p.length && p.text[p.pos] != '<' {
		c := p.text[p.pos]
		if c == '.' {
			id += "."
			p.pos++ // consume '.'
		}
		ident, err := p.parseIdentifier()
		if err != nil {
			return "", err
		}
		id += ident
		p.skipWhitespace()
		if p.pos < p.length && p.text[p.pos] != '.' {
			break
		}
	}
	if id == "" {
		return "", fmt.Errorf("missing identifier at position %d", p.pos)
	}
	return id, nil
}

func (p *typeParser) parseIdentifier() (string, error) {
	p.skipWhitespace()
	if p.pos >= p.length {
		return "", fmt.Errorf("unexpected end of input")
	}
	start := p.pos
	c := p.text[p.pos]
	if !isAlpha(c) && c != '_' {
		return "", fmt.Errorf("identifier is expected, but %q was found at position %d", c, p.pos)
	}
	p.pos++
	for p.pos < p.length {
		c := p.text[p.pos]
		if !isAlphaNumeric(c) && c != '_' {
			break
		}
		p.pos++
	}
	return p.text[start:p.pos], nil
}

func (p *typeParser) parseTypeParamIdent() (string, error) {
	p.skipWhitespace()
	if p.pos >= p.length {
		return "", fmt.Errorf("unexpected end of input")
	}
	c := p.text[p.pos]
	if !isAlpha(c) {
		return "", fmt.Errorf("invalid type parameter identifier %q at position %d, must be a single character from A-Z", c, p.pos)
	}
	p.pos++
	if p.pos < p.length && isAlpha(p.text[p.pos]) {
		return "", fmt.Errorf("invalid type param, must have a single alphabetic character at position %d", p.pos)
	}
	return string(c), nil
}

func (p *typeParser) skipWhitespace() {
	for p.pos < p.length && p.text[p.pos] == ' ' {
		p.pos++
	}
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || (c >= '0' && c <= '9')
}
