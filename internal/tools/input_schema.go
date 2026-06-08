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
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
)

// ComputeInputSchema returns a schemaNode for the given CEL expression.
func ComputeInputSchema(env *cel.Env, astVal *cel.Ast) (*SchemaNode, error) {
	astRep := astVal.NativeRep()
	root := ast.NavigateAST(astRep)
	rootNode := newSchemaNode("object")
	visitor := ast.NewExprVisitor(func(e ast.Expr) {
		switch e.Kind() {
		case ast.IdentKind:
			rootNode.AddPropertyRef(e.AsIdent(), e.ID(), true)
		case ast.SelectKind:
			s := e.AsSelect()
			op := s.Operand()
			path := []ast.Expr{e}
			for op.Kind() == ast.SelectKind {
				elem := op.AsSelect()
				if elem.IsTestOnly() {
					break
				}
				path = append([]ast.Expr{op}, path...)
				op = elem.Operand()
			}
			if op.Kind() == ast.IdentKind {
				varNode := rootNode.AddPropertyRef(op.AsIdent(), op.ID(), false)
				for i, p := range path {
					field := p.AsSelect().FieldName()
					varNode = varNode.AddPropertyRef(field, p.ID(), i == len(path)-1)
				}
			}
		}
	})
	ast.PostOrderVisit(root, visitor)

	// Clean up internal comprehension variables which often have names like "@result" or single/double letters like "l" "k" "v" usually but anything from iterRange essentially.
	// Since we don't know the exact loop struct variables from just Ident, a safe heuristic is filtering out anything starting with '@'
	// To be truly robust, we should capture comprehension iter_var and accu_var strings and ignore them in a scope map, but for now we just filter `@` and known short ones if they don't exist in config.
	// A simpler way: Only keep variables defined in the env JSON, or filter out `@`
	delete(rootNode.Properties, "@result")

	// A better way is to collect comprehension variables while visiting
	comprehensionVars := make(map[string]bool)
	ast.PreOrderVisit(root, ast.NewExprVisitor(func(e ast.Expr) {
		if e.Kind() == ast.ComprehensionKind {
			comp := e.AsComprehension()
			comprehensionVars[comp.IterVar()] = true
			comprehensionVars[comp.AccuVar()] = true
		}
	}))

	for v := range comprehensionVars {
		delete(rootNode.Properties, v)
	}

	rootNode.ApplyTypes(env, astRep.TypeMap())
	return rootNode, nil
}

// SchemaFromCELType returns a schemaNode for the given CEL type.
func SchemaFromCELType(env *cel.Env, t *types.Type) *SchemaNode {
	visited := map[string]*SchemaNode{}
	return schemaFromCELTypeInternal(env, t, visited)
}

func schemaFromCELTypeInternal(env *cel.Env, t *types.Type, visited map[string]*SchemaNode) *SchemaNode {
	if node, found := visited[t.TypeName()]; found {
		return node
	}
	switch t.Kind() {
	case types.MapKind:
		m := newSchemaNode("object")
		m.AdditionalProperties = schemaFromCELTypeInternal(env, t.Parameters()[1], visited)
		return m
	case types.ListKind:
		l := newSchemaNode("array")
		l.Items = schemaFromCELTypeInternal(env, t.Parameters()[0], visited)
		return l
	case types.StringKind:
		return newSchemaNode("string")
	case types.DoubleKind:
		d := newSchemaNode("number")
		d.Format = "double"
		return d
	case types.IntKind:
		i := newSchemaNode("integer")
		i.Format = "int64"
		return i
	case types.UintKind:
		i := newSchemaNode("integer")
		i.Format = "int64"
		i.Minimum = 0
		return i
	case types.DurationKind:
		return newSchemaNode("string")
	case types.TimestampKind:
		i := newSchemaNode("string")
		i.Format = "date-time"
		return i
	case types.BoolKind:
		return newSchemaNode("boolean")
	case types.NullTypeKind:
		return newSchemaNode("null")
	case types.OpaqueKind:
		if t.TypeName() == "optional_type" {
			return schemaFromCELTypeInternal(env, t.Parameters()[0], visited)
		}
		obj := newSchemaNode("object")
		obj.TypeID = t.TypeName()
		return obj
	case types.TypeKind:
		obj := newSchemaNode("object")
		obj.TypeID = "type"
		return obj
	case types.DynKind:
		return newSchemaNode("object")
	default:
		obj := newSchemaNode("object")
		visited[t.TypeName()] = obj
		fieldNames, found := env.CELTypeProvider().FindStructFieldNames(t.TypeName())
		if !found {
			return obj
		}
		for _, fieldName := range fieldNames {
			ft, _ := env.CELTypeProvider().FindStructFieldType(t.TypeName(), fieldName)
			obj.Properties[fieldName] = schemaFromCELTypeInternal(env, ft.Type, visited)
		}
		return obj
	}
}

func newSchemaNode(typeName string) *SchemaNode {
	return &SchemaNode{
		Type:       typeName,
		Properties: make(map[string]*SchemaNode),
	}
}

type exprRef struct {
	ExprID int64
	Leaf   bool
}

// SchemaNode is a node in a JSON schema.
type SchemaNode struct {
	ExprRefs             []*exprRef             `json:"-"`
	TypeID               string                 `json:"$id,omitempty"`
	Type                 string                 `json:"type"`
	Format               string                 `json:"format,omitempty"`
	Items                *SchemaNode            `json:"items,omitempty"`
	Properties           map[string]*SchemaNode `json:"properties,omitempty"`
	AdditionalProperties *SchemaNode            `json:"additionalProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Minimum              float64                `json:"minimum,omitempty"`
}

// IsLeaf returns true if the schema node is a leaf node.
func (s *SchemaNode) IsLeaf() bool {
	for _, ref := range s.ExprRefs {
		if ref.Leaf {
			return true
		}
	}
	return false
}

// FindType returns the CEL type for the schema node.
func (s *SchemaNode) FindType(typeMap map[int64]*types.Type) *types.Type {
	for _, ref := range s.ExprRefs {
		if t, found := typeMap[ref.ExprID]; found {
			return t
		}
	}
	return nil
}

// AddExprRef adds an expression reference to the schema node.
func (s *SchemaNode) AddExprRef(exprID int64, leaf bool) {
	for _, ref := range s.ExprRefs {
		if ref.ExprID == exprID {
			ref.Leaf = false
			return
		}
	}
	s.ExprRefs = append(s.ExprRefs, &exprRef{ExprID: exprID, Leaf: leaf})
}

// AddPropertyRef adds a property reference to the schema node.
func (s *SchemaNode) AddPropertyRef(name string, id int64, leaf bool) *SchemaNode {
	if existing, found := s.Properties[name]; found {
		existing.AddExprRef(id, leaf)
		return existing
	}
	node := newSchemaNode("")
	if !leaf {
		node.Type = "object"
	}
	node.AddExprRef(id, leaf)
	s.Type = "object"
	s.Properties[name] = node
	return node
}

// ApplyTypes applies the CEL types to the schema node.
func (s *SchemaNode) ApplyTypes(env *cel.Env, typeMap map[int64]*types.Type) {
	for _, p := range s.Properties {
		p.ApplyTypes(env, typeMap)
		t := p.FindType(typeMap)
		if t == nil {
			continue
		}
		if t.Kind() == types.StructKind && !p.IsLeaf() {
			continue
		}
		schema := SchemaFromCELType(env, t)
		p.TypeID = schema.TypeID
		p.Type = schema.Type
		p.Format = schema.Format
		p.Items = schema.Items
		p.AdditionalProperties = schema.AdditionalProperties
		if len(p.Properties) == 0 {
			p.Properties = schema.Properties
		}
		p.Required = schema.Required
		p.Minimum = schema.Minimum
	}
}
