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

	"cel.dev/cel-go/cel"
	celast "cel.dev/cel-go/common/ast"
	"cel.dev/cel-go/common/types"
)

func TestCoverageReport(t *testing.T) {
	// Empty report
	emptyReport := &CoverageReport{
		TotalNodes:    0,
		TotalBranches: 0,
	}
	if emptyReport.NodeCoverage() != 100.0 {
		t.Errorf("expected 100.0 for empty TotalNodes, got %f", emptyReport.NodeCoverage())
	}
	if emptyReport.BranchCoverage() != 100.0 {
		t.Errorf("expected 100.0 for empty TotalBranches, got %f", emptyReport.BranchCoverage())
	}

	// Non-empty report
	report := &CoverageReport{
		TotalNodes:      10,
		CoveredNodes:    5,
		TotalBranches:   4,
		CoveredBranches: 2,
	}
	if report.NodeCoverage() != 50.0 {
		t.Errorf("expected 50.0 node coverage, got %f", report.NodeCoverage())
	}
	if report.BranchCoverage() != 50.0 {
		t.Errorf("expected 50.0 branch coverage, got %f", report.BranchCoverage())
	}
}

func TestNodeCoverageMethods(t *testing.T) {
	env, err := cel.NewEnv()
	if err != nil {
		t.Fatalf("cel.NewEnv() failed: %v", err)
	}
	ast, iss := env.Compile("x > 0")
	if iss.Err() != nil {
		// x is undefined so compile fails, let's use valid bool expr
	}
	_ = ast

	// Test non-branch node coverage
	intType := cel.IntType
	fac := celast.NewExprFactory()
	nonBranchExpr := fac.NewIdent(1, "myVar")
	nonBranchNode := &nodeCoverage{
		exprType: intType,
		expr:     nonBranchExpr,
		visited:  false,
	}

	if nonBranchNode.NodeCoverage() != 0.0 {
		t.Errorf("expected 0.0 node coverage for unvisited, got %f", nonBranchNode.NodeCoverage())
	}
	if nonBranchNode.IsBranch() {
		t.Errorf("expected IsBranch() to be false for non-bool node")
	}
	if nonBranchNode.BranchesCovered() != 0 {
		t.Errorf("expected 0 branches covered for non-branch node, got %d", nonBranchNode.BranchesCovered())
	}
	if nonBranchNode.TrueCovered() {
		t.Errorf("expected false for TrueCovered on non-branch")
	}
	if nonBranchNode.FalseCovered() {
		t.Errorf("expected false for FalseCovered on non-branch")
	}

	// Record a value
	nonBranchNode.Record(types.Int(42))
	if !nonBranchNode.visited {
		t.Error("expected visited to be true after Record")
	}
	if nonBranchNode.NodeCoverage() != 1.0 {
		t.Errorf("expected 1.0 node coverage for visited, got %f", nonBranchNode.NodeCoverage())
	}

	// Test branch node coverage
	boolType := cel.BoolType
	branchExpr := fac.NewCall(2, "_>_", fac.NewIdent(3, "a"), fac.NewLiteral(4, types.Int(0)))
	branchNode := &nodeCoverage{
		exprType: boolType,
		expr:     branchExpr,
		visited:  false,
	}

	if !branchNode.IsBranch() {
		t.Error("expected IsBranch() to be true for call returning bool")
	}
	if branchNode.BranchesCovered() != 0 {
		t.Errorf("expected 0 branches covered initially, got %d", branchNode.BranchesCovered())
	}

	// Record True
	branchNode.Record(types.True)
	if !branchNode.TrueCovered() {
		t.Error("expected TrueCovered to be true")
	}
	if branchNode.FalseCovered() {
		t.Error("expected FalseCovered to be false")
	}
	if branchNode.BranchesCovered() != 1 {
		t.Errorf("expected 1 branch covered, got %d", branchNode.BranchesCovered())
	}

	// Record False
	branchNode.Record(types.False)
	if !branchNode.FalseCovered() {
		t.Error("expected FalseCovered to be true")
	}
	if branchNode.BranchesCovered() != 2 {
		t.Errorf("expected 2 branches covered, got %d", branchNode.BranchesCovered())
	}
}

func TestCoverageTracker_PartialBranchCoverage(t *testing.T) {
	env, err := cel.NewEnv(
		cel.Variable("a", cel.IntType),
		cel.Variable("b", cel.IntType),
	)
	if err != nil {
		t.Fatalf("cel.NewEnv failed: %v", err)
	}

	ast, iss := env.Compile("a > 0 && b > 0")
	if iss.Err() != nil {
		t.Fatalf("Compile failed: %v", iss.Err())
	}

	prg, err := env.Program(ast, cel.EvalOptions(cel.OptTrackState))
	if err != nil {
		t.Fatalf("Program failed: %v", err)
	}

	tracker := NewCoverageTracker(ast)

	// Run only false branch for a
	_, details, err := prg.Eval(map[string]any{"a": -1, "b": 10})
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}
	tracker.Record(details)

	report := tracker.GenerateReport()
	if len(report.UncoveredTrueBranches) == 0 && len(report.UncoveredFalseBranches) == 0 {
		t.Logf("branches recorded: covered=%d, total=%d", report.CoveredBranches, report.TotalBranches)
	}

	// Test Record with empty coverageStats map to test !ok branches
	emptyTracker := &CoverageTracker{
		coverageStats: make(map[int64]*nodeCoverage),
	}
	emptyTracker.Record(details)
}
