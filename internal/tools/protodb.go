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
	"strings"

	celenv "cel.dev/cel-go/common/env"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	descpb "google.golang.org/protobuf/types/descriptorpb"
)

// DescriptorDB resolves a fully-qualified protobuf symbol name to the
// FileDescriptorProto defining it along with its transitive file dependencies
// in topological (dependency-first) order.
type DescriptorDB interface {
	ResolveSymbol(symbol string) ([]*descpb.FileDescriptorProto, error)
}

type registryDescriptorDB struct{}

var defaultDescriptorDB DescriptorDB = registryDescriptorDB{}

// DefaultDescriptorDB returns the DescriptorDB used when none is supplied.
func DefaultDescriptorDB() DescriptorDB {
	return defaultDescriptorDB
}

// SetDefaultDescriptorDB replaces the DescriptorDB used when none is supplied.
//
// It is intended to be called during program initialization.
func SetDefaultDescriptorDB(db DescriptorDB) {
	defaultDescriptorDB = db
}

func (r registryDescriptorDB) ResolveSymbol(symbol string) ([]*descpb.FileDescriptorProto, error) {
	symbol = strings.TrimPrefix(symbol, ".")
	if symbol == "" {
		return nil, protoregistry.NotFound
	}

	fullName := protoreflect.FullName(symbol)
	var fd protoreflect.FileDescriptor

	if d, err := protoregistry.GlobalFiles.FindDescriptorByName(fullName); err == nil && d != nil {
		fd = d.ParentFile()
	} else if mt, err := protoregistry.GlobalTypes.FindMessageByName(fullName); err == nil && mt != nil {
		fd = mt.Descriptor().ParentFile()
	} else if et, err := protoregistry.GlobalTypes.FindEnumByName(fullName); err == nil && et != nil {
		fd = et.Descriptor().ParentFile()
	} else if xt, err := protoregistry.GlobalTypes.FindExtensionByName(fullName); err == nil && xt != nil {
		fd = xt.TypeDescriptor().ParentFile()
	}

	if fd == nil {
		return nil, protoregistry.NotFound
	}

	var result []*descpb.FileDescriptorProto
	seen := make(map[string]bool)

	var walk func(file protoreflect.FileDescriptor)
	walk = func(file protoreflect.FileDescriptor) {
		if file == nil || seen[file.Path()] {
			return
		}
		imports := file.Imports()
		for i := 0; i < imports.Len(); i++ {
			imp := imports.Get(i)
			if imp.FileDescriptor != nil {
				walk(imp.FileDescriptor)
			}
		}
		seen[file.Path()] = true
		result = append(result, protodesc.ToFileDescriptorProto(file))
	}

	walk(fd)
	return result, nil
}

// CollectWithDeps walks fdp.GetDependency() recursively in post-order using fileByName,
// deduplicating by GetName(), and appends fdp last.
func CollectWithDeps(fdp *descpb.FileDescriptorProto, fileByName func(string) (*descpb.FileDescriptorProto, error)) ([]*descpb.FileDescriptorProto, error) {
	if fdp == nil {
		return nil, nil
	}
	var result []*descpb.FileDescriptorProto
	seen := make(map[string]bool)

	var walk func(curr *descpb.FileDescriptorProto) error
	walk = func(curr *descpb.FileDescriptorProto) error {
		if curr == nil || seen[curr.GetName()] {
			return nil
		}
		for _, dep := range curr.GetDependency() {
			if seen[dep] {
				continue
			}
			depFDP, err := fileByName(dep)
			if err != nil {
				return fmt.Errorf("resolving dependency %q: %w", dep, err)
			}
			if err := walk(depFDP); err != nil {
				return err
			}
		}
		seen[curr.GetName()] = true
		result = append(result, curr)
		return nil
	}

	if err := walk(fdp); err != nil {
		return nil, err
	}
	return result, nil
}

var builtInCELTypes = map[string]bool{
	"bool":          true,
	"int":           true,
	"uint":          true,
	"double":        true,
	"bytes":         true,
	"string":        true,
	"list":          true,
	"map":           true,
	"type":          true,
	"null_type":     true,
	"dyn":           true,
	"any":           true,
	"optional_type": true,
	"error":         true,
}

func collectTypesFromTypeDesc(td *celenv.TypeDesc, addType func(string)) {
	if td == nil || td.IsTypeParam {
		return
	}
	if !builtInCELTypes[td.TypeName] && td.TypeName != "" {
		addType(td.TypeName)
	}
	for _, p := range td.Params {
		collectTypesFromTypeDesc(p, addType)
	}
}

// collectDeclaredTypes passes the type names referenced by the context variable, variables, and
// function overloads in cfg to addType. It returns warnings for declarations that fail to parse.
func collectDeclaredTypes(cfg *Config, addType func(string)) []string {
	var warnings []string
	if cfg.ContextVariable != nil && cfg.ContextVariable.Type != "" {
		td, err := parseType(cfg.ContextVariable.Type)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("proto_db: failed to parse context variable type %q: %v", cfg.ContextVariable.Type, err))
		} else {
			collectTypesFromTypeDesc(td, addType)
		}
	}

	for _, v := range cfg.Variables {
		if v == nil {
			continue
		}
		celVar, err := v.ToCELVariable()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("proto_db: failed to convert variable %q: %v", v.Name, err))
			continue
		}
		if celVar != nil && celVar.TypeDesc != nil {
			collectTypesFromTypeDesc(celVar.TypeDesc, addType)
		}
	}

	for _, fn := range cfg.Functions {
		if fn == nil {
			continue
		}
		celFunc, err := fn.ToCELFunction()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("proto_db: failed to convert function %q: %v", fn.Name, err))
			continue
		}
		for _, o := range celFunc.Overloads {
			if o.Target != nil {
				collectTypesFromTypeDesc(o.Target, addType)
			}
			for _, arg := range o.Args {
				collectTypesFromTypeDesc(arg, addType)
			}
			if o.Return != nil {
				collectTypesFromTypeDesc(o.Return, addType)
			}
		}
	}
	return warnings
}

// ExtendFileDescriptorSet extends an existing FileDescriptorSet with message and enum definitions
// extracted from the environment configuration and extraTypes, resolving them from db.
func ExtendFileDescriptorSet(fds *descpb.FileDescriptorSet, cfg *Config, db DescriptorDB, extraTypes ...string) (*descpb.FileDescriptorSet, []string, error) {
	out := &descpb.FileDescriptorSet{}
	seen := make(map[string]bool)
	if fds != nil {
		for _, f := range fds.GetFile() {
			if f != nil {
				seen[f.GetName()] = true
				out.File = append(out.File, f)
			}
		}
	}

	if db == nil {
		db = DefaultDescriptorDB()
	}

	var warnings []string
	var typeNames []string
	seenTypes := make(map[string]bool)

	addType := func(t string) {
		t = strings.TrimSpace(t)
		if t == "" || seenTypes[t] {
			return
		}
		seenTypes[t] = true
		typeNames = append(typeNames, t)
	}

	if cfg != nil {
		for _, imp := range cfg.Imports {
			if imp != nil {
				addType(imp.Name)
			}
		}
		for _, pt := range cfg.ProtoTypes {
			addType(pt)
		}
		warnings = append(warnings, collectDeclaredTypes(cfg, addType)...)
	}

	for _, extra := range extraTypes {
		addType(extra)
	}

	// Types referenced in environment declarations must be fully qualified: cel-go does not apply
	// the container or import abbreviations when resolving them (those only apply to expressions).
	for _, typeName := range typeNames {
		fdps, err := db.ResolveSymbol(strings.TrimPrefix(typeName, "."))
		if err != nil || len(fdps) == 0 {
			warnings = append(warnings, fmt.Sprintf("proto_db: failed to resolve type %q", typeName))
			continue
		}
		for _, fdp := range fdps {
			if fdp != nil && !seen[fdp.GetName()] {
				seen[fdp.GetName()] = true
				out.File = append(out.File, fdp)
			}
		}
	}

	return out, warnings, nil
}

// AugmentErrorWithWarnings wraps an error with proto_db warnings if any exist.
func AugmentErrorWithWarnings(err error, warnings []string) error {
	if err == nil || len(warnings) == 0 {
		return err
	}
	return fmt.Errorf("%w (proto_db warnings: %s)", err, strings.Join(warnings, "; "))
}
