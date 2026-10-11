// Package gosrc는 디렉토리의 Go 소스 파일을 고르고 읽는다. codescan과 infer가 같은 파일 집합을 보도록 한 곳에 둔다.
package gosrc

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ParseError는 읽거나 파싱하지 못한 파일이다.
type ParseError struct {
	Path string
	Err  error
}

// Files는 dir의 Go 파일을 문법 트리로 준다. _test.go와 //go:build ignore 파일은 뺀다.
// OS·아키텍처 조건은 모두 참으로 보아 어느 플랫폼의 파일이든 읽는다. 디렉토리가 없으면 비어 있다.
// build 조건을 보려고 mode에 parser.ParseComments를 더한다.
func Files(fset *token.FileSet, dir string, mode parser.Mode) ([]*ast.File, []ParseError) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var files []*ast.File
	var errs []ParseError
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		p := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, p, nil, mode|parser.ParseComments)
		if err != nil {
			errs = append(errs, ParseError{p, err})
			continue
		}
		if ignored(f) {
			continue
		}
		files = append(files, f)
	}
	return files, errs
}

// ImportsOnly는 파일의 package 절과 import만 읽는다. 본문에 문법 오류가 있어도 읽힌다.
// 빌드 제약으로 빠지는 파일이면 nil이다.
func ImportsOnly(fset *token.FileSet, path string) (*ast.File, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil || ignored(f) {
		return nil, err
	}
	return f, nil
}

// SkipDir는 코드를 읽지 않는 디렉토리다.
func SkipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules"
}

// ignored는 파일의 build 조건이 ignore 태그 없이는 참이 될 수 없는지다.
func ignored(f *ast.File) bool {
	for _, cg := range f.Comments {
		if cg.Pos() >= f.Package {
			break
		}
		for _, c := range cg.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return false
			}
			return !satisfiable(expr)
		}
	}
	return false
}

// satisfiable은 ignore 태그를 거짓으로 두고 나머지 태그를 어떻게 정해 참이 될 수 있는지다.
// 태그가 많으면 판단하지 않고 참으로 본다.
func satisfiable(expr constraint.Expr) bool {
	var tags []string
	seen := map[string]bool{}
	var walk func(e constraint.Expr)
	walk = func(e constraint.Expr) {
		switch x := e.(type) {
		case *constraint.AndExpr:
			walk(x.X)
			walk(x.Y)
		case *constraint.OrExpr:
			walk(x.X)
			walk(x.Y)
		case *constraint.NotExpr:
			walk(x.X)
		case *constraint.TagExpr:
			if x.Tag != "ignore" && !seen[x.Tag] {
				seen[x.Tag] = true
				tags = append(tags, x.Tag)
			}
		}
	}
	walk(expr)
	if len(tags) > 12 {
		return true
	}
	for mask := 0; mask < 1<<len(tags); mask++ {
		ok := expr.Eval(func(tag string) bool {
			for i, t := range tags {
				if t == tag {
					return mask&(1<<i) != 0
				}
			}
			return false
		})
		if ok {
			return true
		}
	}
	return false
}
