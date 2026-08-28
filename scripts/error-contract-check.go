package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type violation struct {
	position token.Position
	reason   string
}

func main() {
	strict := flag.Bool("strict", false, "exit non-zero when violations are found")
	flag.Parse()

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "get working directory: %v\n", err)
		os.Exit(2)
	}

	violations, err := findViolations(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error contract audit: %v\n", err)
		os.Exit(2)
	}
	if len(violations) == 0 {
		fmt.Println("error contract audit passed")
		return
	}

	for _, violation := range violations {
		fmt.Printf("%s:%d: %s\n", violation.position.Filename, violation.position.Line, violation.reason)
	}

	if *strict {
		fmt.Fprintln(os.Stderr, "error contract check failed")
		os.Exit(1)
	}
	fmt.Println("error contract audit found migration work")
}

func findViolations(projectRoot string) ([]violation, error) {
	var violations []violation
	appRoot := filepath.Join(projectRoot, "app")
	err := filepath.WalkDir(appRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if createsDatabaseCodeWithoutCause(call) {
				violations = append(violations, newViolation(projectRoot, fileSet.Position(call.Pos()), "database error code must wrap the original cause"))
			}
			if classifiesErrorByText(call) {
				violations = append(violations, newViolation(projectRoot, fileSet.Position(call.Pos()), "classify errors with errors.Is or errors.As instead of err.Error() text"))
			}
			return true
		})
		return nil
	})
	return violations, err
}

func newViolation(projectRoot string, position token.Position, reason string) violation {
	if relative, err := filepath.Rel(projectRoot, position.Filename); err == nil {
		position.Filename = relative
	}
	return violation{position: position, reason: reason}
}

func createsDatabaseCodeWithoutCause(call *ast.CallExpr) bool {
	_, name, ok := selector(call.Fun)
	if !ok || name != "NewCode" || len(call.Args) < 2 || !isErrcodeDatabase(call.Args[0]) {
		return false
	}
	return containsErrorText(call.Args[1])
}

func classifiesErrorByText(call *ast.CallExpr) bool {
	receiver, name, ok := selector(call.Fun)
	if !ok || receiver != "strings" || (name != "Contains" && name != "EqualFold") {
		return false
	}
	for _, argument := range call.Args {
		if containsErrorText(argument) {
			return true
		}
	}
	return false
}

func isErrcodeDatabase(expr ast.Expr) bool {
	receiver, name, ok := selector(expr)
	return ok && receiver == "errcode" && name == "ErrDatabase"
}

func containsErrorText(expr ast.Expr) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		receiver, name, ok := selector(call.Fun)
		if ok && ((receiver == "fmt" && name == "Sprintf") || name == "Error") {
			found = true
			return false
		}
		return true
	})
	return found
}

func selector(expr ast.Expr) (string, string, bool) {
	selection, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", "", false
	}
	receiver, ok := selection.X.(*ast.Ident)
	if !ok {
		return "", "", false
	}
	return receiver.Name, selection.Sel.Name, true
}
