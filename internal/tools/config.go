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
func (c *Config) ToCELConfig() *celenv.Config {
	if c == nil {
		return nil
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
		res.ContextVariable = c.ContextVariable.ToCELContextVariable()
	}
	for _, v := range c.Variables {
		res.Variables = append(res.Variables, v.ToCELVariable())
	}
	for _, f := range c.Functions {
		res.Functions = append(res.Functions, f.ToCELFunction())
	}
	for _, v := range c.Validators {
		res.Validators = append(res.Validators, v.ToCELValidator())
	}
	for _, f := range c.Features {
		res.Features = append(res.Features, f.ToCELFeature())
	}
	return res
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
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	TypeName    string      `json:"typeName,omitempty"`
	Params      []*TypeDesc `json:"params,omitempty"`
}

// ToCELVariable converts a Variable to a celenv.Variable.
func (v *Variable) ToCELVariable() *celenv.Variable {
	if v == nil {
		return nil
	}
	var params []*celenv.TypeDesc
	for _, p := range v.Params {
		params = append(params, p.ToCELTypeDesc())
	}
	return &celenv.Variable{
		Name:        v.Name,
		Description: v.Description,
		TypeDesc: &celenv.TypeDesc{
			TypeName: v.TypeName,
			Params:   params,
		},
	}
}

// ContextVariable is a context variable for a cel-go Env.
type ContextVariable struct {
	TypeName string `json:"typeName"`
}

// ToCELContextVariable converts a ContextVariable to a celenv.ContextVariable.
func (c *ContextVariable) ToCELContextVariable() *celenv.ContextVariable {
	if c == nil {
		return nil
	}
	return celenv.NewContextVariable(c.TypeName)
}

// Function is a function for a cel-go Env.
type Function struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Overloads   []*Overload `json:"overloads"`
}

// ToCELFunction converts a Function to a celenv.Function.
func (f *Function) ToCELFunction() *celenv.Function {
	if f == nil {
		return nil
	}
	var celOverloads []*celenv.Overload
	for _, o := range f.Overloads {
		celOverloads = append(celOverloads, o.ToCELOverload())
	}
	if f.Description != "" {
		return celenv.NewFunctionWithDoc(f.Name, f.Description, celOverloads...)
	}
	return celenv.NewFunction(f.Name, celOverloads...)
}

// Overload is an overload for a cel-go Env.
type Overload struct {
	ID       string      `json:"id"`
	Examples []string    `json:"examples,omitempty"`
	Target   *TypeDesc   `json:"target,omitempty"`
	Args     []*TypeDesc `json:"args,omitempty"`
	Return   *TypeDesc   `json:"return"`
}

// ToCELOverload converts an Overload to a celenv.Overload.
func (o *Overload) ToCELOverload() *celenv.Overload {
	if o == nil {
		return nil
	}
	var args []*celenv.TypeDesc
	for _, a := range o.Args {
		args = append(args, a.ToCELTypeDesc())
	}
	var ret *celenv.TypeDesc
	if o.Return != nil {
		ret = o.Return.ToCELTypeDesc()
	}
	var target *celenv.TypeDesc
	if o.Target != nil {
		target = o.Target.ToCELTypeDesc()
	}

	if target != nil {
		return celenv.NewMemberOverload(o.ID, target, args, ret, o.Examples...)
	}
	return celenv.NewOverload(o.ID, args, ret, o.Examples...)
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

// TypeDesc is a type descriptor for a cel-go Env.
type TypeDesc struct {
	TypeName    string            `json:"typeName"`
	Params      []*SimpleTypeDesc `json:"params,omitempty"`
	IsTypeParam bool              `json:"isTypeParam,omitempty"`
}

// ToCELTypeDesc converts a TypeDesc to a celenv.TypeDesc.
func (t *TypeDesc) ToCELTypeDesc() *celenv.TypeDesc {
	if t == nil {
		return nil
	}
	var params []*celenv.TypeDesc
	for _, p := range t.Params {
		params = append(params, p.ToCELTypeDesc())
	}
	res := celenv.NewTypeDesc(t.TypeName, params...)
	if t.IsTypeParam && len(t.Params) == 0 {
		return celenv.NewTypeParam(t.TypeName)
	}
	return res
}

// SimpleTypeDesc is a simple type descriptor for a cel-go Env.
type SimpleTypeDesc struct {
	TypeName    string `json:"typeName"`
	IsTypeParam bool   `json:"isTypeParam,omitempty"`
}

// ToCELTypeDesc converts a SimpleTypeDesc to a celenv.TypeDesc.
func (s *SimpleTypeDesc) ToCELTypeDesc() *celenv.TypeDesc {
	if s == nil {
		return nil
	}
	res := celenv.NewTypeDesc(s.TypeName)
	if s.IsTypeParam {
		return celenv.NewTypeParam(s.TypeName)
	}
	return res
}

// ConfigFromJSON converts a JSON string to a Config.
func ConfigFromJSON(configJSON string) (*Config, error) {
	var config Config
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("json.Unmarshal(configJSON) failed: %v", err)
	}
	return &config, nil
}

// EnvFromConfig takes the JSON string and converts it to a cel-go Env.
func EnvFromConfig(envConfig *Config) (*cel.Env, error) {
	celConfig := envConfig.ToCELConfig()
	return cel.NewEnv(cel.FromConfig(celConfig, celext.ExtensionOptionFactory))
}
