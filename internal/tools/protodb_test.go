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
	"strings"
	"testing"

	"cel.dev/cel-go/cel"
	_ "github.com/cel-expr/skills/internal/proto"
	descpb "google.golang.org/protobuf/types/descriptorpb"
)

func TestCollectWithDeps(t *testing.T) {
	// A depends on B and C; B depends on C; C has no deps.
	// Post-order result must have C before B, and both before A: [C, B, A].
	protoFiles := map[string]*descpb.FileDescriptorProto{
		"c.proto": {
			Name: protoStr("c.proto"),
		},
		"b.proto": {
			Name:       protoStr("b.proto"),
			Dependency: []string{"c.proto"},
		},
		"a.proto": {
			Name:       protoStr("a.proto"),
			Dependency: []string{"b.proto", "c.proto"},
		},
	}

	fileByName := func(name string) (*descpb.FileDescriptorProto, error) {
		if f, ok := protoFiles[name]; ok {
			return f, nil
		}
		return nil, fmt.Errorf("file %q not found", name)
	}

	res, err := CollectWithDeps(protoFiles["a.proto"], fileByName)
	if err != nil {
		t.Fatalf("CollectWithDeps failed: %v", err)
	}

	expected := []string{"c.proto", "b.proto", "a.proto"}
	if len(res) != len(expected) {
		t.Fatalf("expected %d files, got %d", len(expected), len(res))
	}
	for i, f := range res {
		if f.GetName() != expected[i] {
			t.Errorf("res[%d] = %q, want %q", i, f.GetName(), expected[i])
		}
	}

	// Missing dependency returns an error.
	brokenFile := &descpb.FileDescriptorProto{
		Name:       protoStr("broken.proto"),
		Dependency: []string{"missing.proto"},
	}
	if _, err := CollectWithDeps(brokenFile, fileByName); err == nil {
		t.Errorf("expected error for missing dependency, got nil")
	}
}

func TestExtendFileDescriptorSet_ResolutionModes(t *testing.T) {
	cfg := &Config{
		Imports: []*Import{
			{Name: "cel.skills.internal.proto.TestMessage"},
		},
		ContextVariable: &ContextVariable{
			Type: "cel.skills.internal.proto.TestMessage",
		},
		Variables: []*Variable{
			{Name: "direct", Type: "cel.skills.internal.proto.TestMessage"},
			{Name: "leading_dot", Type: ".cel.skills.internal.proto.TestMessage"},
			{Name: "nested", Type: "cel.skills.internal.proto.TestMessage.NestedMessage"},
			{Name: "list_generic", Type: "list<cel.skills.internal.proto.TestMessage>"},
			{Name: "map_generic", Type: "map<string, cel.skills.internal.proto.TestMessage>"},
			{Name: "ts", Type: "google.protobuf.Timestamp"},
			{Name: "dur", Type: "google.protobuf.Duration"},
			{Name: "st", Type: "google.protobuf.Struct"},
			{Name: "wr", Type: "google.protobuf.Int64Value"},
		},
		Functions: []*Function{
			{
				Name: "myFunc",
				Overloads: []*Overload{
					{
						ID:     "myFunc_msg",
						Args:   []string{"cel.skills.internal.proto.TestMessage"},
						Return: "cel.skills.internal.proto.TestMessage.NestedMessage",
					},
					{
						ID:     "myFunc_member",
						Target: "cel.skills.internal.proto.TestMessage",
						Return: "google.protobuf.Timestamp",
					},
				},
			},
		},
	}

	fds, warnings, err := ExtendFileDescriptorSet(nil, cfg, nil, "google.protobuf.DoubleValue")
	if err != nil {
		t.Fatalf("ExtendFileDescriptorSet failed: %v", err)
	}
	if len(warnings) > 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}

	var foundSchema, foundTimestamp, foundDuration, foundStruct, foundWrappers bool
	for _, f := range fds.GetFile() {
		if strings.HasSuffix(f.GetName(), "test_schema.proto") {
			foundSchema = true
		}
		if f.GetName() == "google/protobuf/timestamp.proto" {
			foundTimestamp = true
		}
		if f.GetName() == "google/protobuf/duration.proto" {
			foundDuration = true
		}
		if f.GetName() == "google/protobuf/struct.proto" {
			foundStruct = true
		}
		if f.GetName() == "google/protobuf/wrappers.proto" {
			foundWrappers = true
		}
	}

	if !foundSchema {
		t.Errorf("test_schema.proto not found in extended FDS: %v", fds.GetFile())
	}
	if !foundTimestamp {
		t.Errorf("google/protobuf/timestamp.proto not found in extended FDS")
	}
	if !foundDuration {
		t.Errorf("google/protobuf/duration.proto not found in extended FDS")
	}
	if !foundStruct {
		t.Errorf("google/protobuf/struct.proto not found in extended FDS")
	}
	if !foundWrappers {
		t.Errorf("google/protobuf/wrappers.proto not found in extended FDS")
	}

	// Verify deduplication: extending an existing FDS should not duplicate files.
	fds2, warnings2, err := ExtendFileDescriptorSet(fds, cfg, nil, "cel.skills.internal.proto.TestMessage")
	if err != nil {
		t.Fatalf("ExtendFileDescriptorSet on existing FDS failed: %v", err)
	}
	if len(warnings2) > 0 {
		t.Errorf("unexpected warnings: %v", warnings2)
	}
	if len(fds2.GetFile()) != len(fds.GetFile()) {
		t.Errorf("expected %d files after deduplication, got %d", len(fds.GetFile()), len(fds2.GetFile()))
	}
}

func TestExtendFileDescriptorSet_WarningsAndAugmentation(t *testing.T) {
	cfg := &Config{
		Container: "cel.skills.internal.proto",
		Variables: []*Variable{
			{Name: "unknownVar", Type: "com.example.NonExistentMessage"},
		},
	}

	fds, warnings, err := ExtendFileDescriptorSet(nil, cfg, nil)
	if err != nil {
		t.Fatalf("ExtendFileDescriptorSet failed: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], `failed to resolve type "com.example.NonExistentMessage"`) {
		t.Errorf("unexpected warning text: %q", warnings[0])
	}

	// Declared types must be fully qualified; the container is not applied.
	_, warnings2, err := ExtendFileDescriptorSet(nil, &Config{
		Container: "cel.skills.internal.proto",
		Variables: []*Variable{{Name: "short", Type: "TestMessage"}},
	}, nil)
	if err != nil {
		t.Fatalf("ExtendFileDescriptorSet failed: %v", err)
	}
	if len(warnings2) != 1 || !strings.Contains(warnings2[0], `failed to resolve type "TestMessage"`) {
		t.Errorf("expected unqualified type warning, got: %v", warnings2)
	}

	// Test AugmentErrorWithWarnings
	origErr := errors.New("type-check error")
	augmented := AugmentErrorWithWarnings(origErr, warnings)
	if augmented == nil || !strings.Contains(augmented.Error(), "proto_db warnings:") {
		t.Errorf("expected augmented error, got: %v", augmented)
	}
	if !errors.Is(augmented, origErr) {
		t.Errorf("expected augmented error to wrap origErr")
	}

	// AugmentErrorWithWarnings with nil error or empty warnings
	if AugmentErrorWithWarnings(nil, warnings) != nil {
		t.Errorf("expected nil when err is nil")
	}
	if got := AugmentErrorWithWarnings(origErr, nil); got != origErr {
		t.Errorf("expected original error when warnings is empty, got %v", got)
	}

	// Test compilation failure augmentation with EnvFromConfig / CompileCEL
	var opts []cel.EnvOption
	if len(fds.GetFile()) > 0 {
		opts = append(opts, cel.TypeDescs(fds))
	}
	_, compileErr := CompileCEL("unknownVar.foo == 'bar'", cfg, opts...)
	if compileErr == nil {
		t.Fatal("expected CompileCEL to fail with unknown type, got nil")
	}
	augCompileErr := AugmentErrorWithWarnings(compileErr, warnings)
	if !strings.Contains(augCompileErr.Error(), "proto_db warnings:") {
		t.Errorf("expected augmented compile error, got: %v", augCompileErr)
	}
}

type fakeDescriptorDB struct {
	files   map[string][]*descpb.FileDescriptorProto
	lookups []string
}

func (f *fakeDescriptorDB) ResolveSymbol(symbol string) ([]*descpb.FileDescriptorProto, error) {
	f.lookups = append(f.lookups, symbol)
	if files, ok := f.files[symbol]; ok {
		return files, nil
	}
	return nil, fmt.Errorf("symbol %q not found", symbol)
}

func TestSetDefaultDescriptorDB(t *testing.T) {
	orig := DefaultDescriptorDB()
	t.Cleanup(func() { SetDefaultDescriptorDB(orig) })

	fake := &fakeDescriptorDB{files: map[string][]*descpb.FileDescriptorProto{
		"example.Msg": {{Name: protoStr("example/msg.proto")}},
	}}
	SetDefaultDescriptorDB(fake)
	if got := DefaultDescriptorDB(); got != fake {
		t.Fatalf("DefaultDescriptorDB() = %v, want fake", got)
	}

	cfg := &Config{Variables: []*Variable{{Name: "msg", Type: "example.Msg"}}}
	fds, warnings, err := ExtendFileDescriptorSet(nil, cfg, nil)
	if err != nil {
		t.Fatalf("ExtendFileDescriptorSet failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if len(fds.GetFile()) != 1 || fds.GetFile()[0].GetName() != "example/msg.proto" {
		t.Errorf("ExtendFileDescriptorSet() files = %v, want [example/msg.proto]", fds.GetFile())
	}
	if len(fake.lookups) != 1 || fake.lookups[0] != "example.Msg" {
		t.Errorf("fake lookups = %v, want [example.Msg]", fake.lookups)
	}
}

func protoStr(s string) *string {
	return &s
}

func TestExtendFileDescriptorSet_ProtoTypes(t *testing.T) {
	cfg := &Config{
		ProtoTypes: []string{
			"cel.skills.internal.proto.TestEnum",
			"cel.skills.internal.proto.TestMessage.NestedEnum",
		},
		Variables: []*Variable{
			{Name: "code", Type: "int"},
		},
	}

	fds, warnings, err := ExtendFileDescriptorSet(nil, cfg, nil)
	if err != nil {
		t.Fatalf("ExtendFileDescriptorSet failed: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	if len(fds.GetFile()) == 0 {
		t.Fatal("expected test_schema.proto in extended FDS")
	}

	// Verify ProtoTypes does not add short-name aliases to ToCELConfig.
	celCfg, err := cfg.ToCELConfig()
	if err != nil {
		t.Fatalf("ToCELConfig failed: %v", err)
	}
	if len(celCfg.Imports) != 0 {
		t.Errorf("expected empty Imports in ToCELConfig, got %v", celCfg.Imports)
	}

	res, err := CompileCEL(
		"code == cel.skills.internal.proto.TestEnum.TEST_ENUM_FIRST && code != cel.skills.internal.proto.TestMessage.NestedEnum.BAR",
		cfg,
		cel.TypeDescs(fds),
	)
	if err != nil {
		t.Fatalf("CompileCEL with protoTypes FDS failed: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil CompileExprOutput")
	}
}
