package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestProgramFiltersMouseMotion checks main builds the program with every
// motion report (the preview's block handles need it) and with the filter
// that drops the motion that changes nothing. Program options are opaque
// functions, so the check reads the tea.NewProgram call in main.go.
func TestProgramFiltersMouseMotion(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var calls int
	var allMotion, filter bool
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || !isSelector(call.Fun, "tea", "NewProgram") {
			return true
		}
		calls++
		for _, arg := range call.Args[1:] {
			opt, ok := arg.(*ast.CallExpr)
			if !ok {
				continue
			}
			switch {
			case isSelector(opt.Fun, "tea", "WithMouseAllMotion"):
				allMotion = true
			case isSelector(opt.Fun, "tea", "WithFilter") && len(opt.Args) == 1 &&
				isSelector(opt.Args[0], "app", "MouseFilter"):
				filter = true
			}
		}
		return true
	})
	if calls != 1 {
		t.Fatalf("want one tea.NewProgram call in main.go, found %d", calls)
	}
	if !allMotion {
		t.Error("the program should keep tea.WithMouseAllMotion()")
	}
	if !filter {
		t.Error("the program should pass tea.WithFilter(app.MouseFilter)")
	}
}

func isSelector(e ast.Expr, pkg, name string) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}
