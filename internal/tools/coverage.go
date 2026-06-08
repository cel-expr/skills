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
	celast "github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types"
)

// NewCoverageTracker creates a new CoverageTracker.
func NewCoverageTracker(ast *cel.Ast) *CoverageTracker {
	rootExpr := celast.NavigateAST(ast.NativeRep())
	coverageStats := make(map[int64]*nodeCoverage)
	celast.PreOrderVisit(rootExpr, celast.NewExprVisitor(func(e celast.Expr) {
		coverageStats[e.ID()] = &nodeCoverage{
			exprType: ast.NativeRep().TypeMap()[e.ID()],
			expr:     e,
		}
	}))
	return &CoverageTracker{
		rootExpr:      celast.NavigateAST(ast.NativeRep()),
		coverageStats: coverageStats,
	}
}

// CoverageTracker is a tracker for coverage stats.
type CoverageTracker struct {
	rootExpr      celast.NavigableExpr
	coverageStats map[int64]*nodeCoverage
}

// Record records the coverage stats for the given expression ID.
func (c *CoverageTracker) Record(details *cel.EvalDetails) {
	state := details.State()
	for _, id := range state.IDs() {
		node, ok := c.coverageStats[id]
		if !ok {
			continue
		}
		value, ok := state.Value(id)
		if !ok {
			continue
		}
		node.Record(value)
	}
}

// GenerateReport generates a coverage report for the given coverage stats.
func (c *CoverageTracker) GenerateReport() *CoverageReport {
	report := &CoverageReport{
		TotalNodes: len(c.coverageStats),
	}
	for _, node := range c.coverageStats {
		if node.visited {
			report.CoveredNodes++
		} else {
			report.UncoveredNodes = append(report.UncoveredNodes, node.expr)
		}
		if node.IsBranch() {
			report.TotalBranches += 2
			if node.TrueCovered() {
				report.CoveredBranches++
			} else if node.visited {
				report.UncoveredTrueBranches = append(report.UncoveredTrueBranches, node.expr)
			}
			if node.FalseCovered() {
				report.CoveredBranches++
			} else if node.visited {
				report.UncoveredFalseBranches = append(report.UncoveredFalseBranches, node.expr)
			}
		}
	}
	return report
}

// CoverageReport documents node and branch coverage for a CEL expression.
type CoverageReport struct {
	TotalNodes             int
	CoveredNodes           int
	TotalBranches          int
	CoveredBranches        int
	UncoveredNodes         []celast.Expr
	UncoveredFalseBranches []celast.Expr
	UncoveredTrueBranches  []celast.Expr
}

// NodeCoverage returns the node coverage for the expression.
func (c *CoverageReport) NodeCoverage() float64 {
	if c.TotalNodes == 0 {
		return 100.0
	}
	return float64(c.CoveredNodes) / float64(c.TotalNodes) * 100.0
}

// BranchCoverage returns the branch coverage for the expression.
func (c *CoverageReport) BranchCoverage() float64 {
	if c.TotalBranches == 0 {
		return 100.0
	}
	return float64(c.CoveredBranches) / float64(c.TotalBranches) * 100.0
}

// nodeCoverage is a node coverage tracker.
type nodeCoverage struct {
	exprType *cel.Type
	expr     celast.Expr
	visited  bool
	values   []ref.Val
}

// Record records the coverage stats for the given value.
func (n *nodeCoverage) Record(value ref.Val) {
	n.visited = true
	n.values = append(n.values, value)
}

// NodeCoverage returns the node coverage for the expression.
func (n *nodeCoverage) NodeCoverage() float64 {
	if n.visited {
		return 1.0
	}
	return 0.0
}

// IsBranch returns true if the expression is a branch.
func (n *nodeCoverage) IsBranch() bool {
	return n.exprType.Kind() == types.BoolKind && n.expr.Kind() != celast.LiteralKind
}

// BranchesCovered returns the branch coverage for the expression.
func (n *nodeCoverage) BranchesCovered() int {
	if !n.IsBranch() {
		return 0
	}
	coverage := 0
	if n.TrueCovered() {
		coverage++
	}
	if n.FalseCovered() {
		coverage++
	}
	return coverage
}

func (n *nodeCoverage) TrueCovered() bool {
	if !n.IsBranch() {
		return false
	}
	for _, value := range n.values {
		if value == types.True {
			return true
		}
	}
	return false
}

func (n *nodeCoverage) FalseCovered() bool {
	if !n.IsBranch() {
		return false
	}
	for _, value := range n.values {
		if value == types.False {
			return true
		}
	}
	return false
}
