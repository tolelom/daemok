package lift

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/model"
)

// unit은 모듈 하나의 패키지에서 설계에 없는 exported 선언을 옮긴다. known은 설계가 만드는 이름이다.
func (c *collector) unit(module, dir string, known map[string]bool) {
	files, _ := gosrc.Files(c.fset, filepath.Join(c.root, filepath.FromSlash(dir)), parser.SkipObjectResolution)
	fresh := func(n *ast.Ident) bool { return n.IsExported() && !known[n.Name] }
	var hand []*ast.File
	for _, f := range files {
		if !generated(f) {
			hand = append(hand, f)
		}
	}
	for k, v := range docs(hand) {
		c.docs[k] = v
	}
	enumTypes, enumValues := c.enums(module, hand, fresh)
	// local은 이 패키지에서 설계 ID로 가리킬 수 있는 타입 이름이다. 설계에 있는 타입·enum과 이번에 함께 옮기는 것.
	local := c.designTypes(module)
	// 내장 타입 이름이 되는 ID는 설계 검사가 막으므로 가리키지 않는다.
	refable := func(name string) {
		if id, _, ok := snakeID(name); ok && model.Builtins[id] == "" {
			local[name] = id
		}
	}
	for name := range enumTypes {
		refable(name)
	}
	for _, f := range hand {
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, sp := range g.Specs {
					ts := sp.(*ast.TypeSpec)
					if _, isStruct := ts.Type.(*ast.StructType); isStruct && fresh(ts.Name) {
						refable(ts.Name.Name)
					}
				}
			}
		}
	}
	for _, f := range hand {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && fresh(d.Name) && d.Body != nil {
					c.pending = append(c.pending, pendingFunc{module, d, f})
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch sp := s.(type) {
					case *ast.TypeSpec:
						if fresh(sp.Name) && !enumTypes[sp.Name.Name] {
							c.structType(module, sp, f, local)
						}
					case *ast.ValueSpec:
						for i, n := range sp.Names {
							if !fresh(n) || enumValues[n.Name] {
								continue
							}
							if d.Tok == token.VAR {
								c.errorVar(module, n, sp, i)
							} else {
								c.constant(module, n, sp, i)
							}
						}
					}
				}
			}
		}
	}
}

// docs는 선언 이름마다 문서 주석이다. 블록 안 선언은 자기 주석, 괄호 없는 단독 선언은 선언 주석이다.
func docs(files []*ast.File) map[*ast.Ident]*ast.CommentGroup {
	out := map[*ast.Ident]*ast.CommentGroup{}
	for _, f := range files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				out[fd.Name] = fd.Doc
				continue
			}
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range g.Specs {
				switch sp := sp.(type) {
				case *ast.TypeSpec:
					out[sp.Name] = specDoc(g, sp.Doc)
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						out[n] = specDoc(g, sp.Doc)
					}
				}
			}
		}
	}
	return out
}

func specDoc(g *ast.GenDecl, doc *ast.CommentGroup) *ast.CommentGroup {
	if doc == nil && !g.Lparen.IsValid() {
		return g.Doc
	}
	return doc
}

// summary는 문서 주석 첫 단락의 첫 문장이다. 160자가 넘으면 요약으로 보지 않는다.
func summary(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	para, _, _ := strings.Cut(doc.Text(), "\n\n")
	text := strings.Join(strings.Fields(para), " ")
	if i := strings.Index(text, ". "); i >= 0 {
		text = text[:i+1]
	}
	if len([]rune(text)) > 160 {
		return ""
	}
	return text
}

func generated(f *ast.File) bool {
	return len(f.Comments) > 0 && f.Comments[0].List[0].Text == golang.Header && f.Comments[0].Pos() < f.Package
}
