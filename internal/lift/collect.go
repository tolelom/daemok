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

func (c *collector) unit(module, dir string, known map[string]bool) {
	files, perrs := gosrc.Files(c.fset, filepath.Join(c.root, filepath.FromSlash(dir)), parser.SkipObjectResolution)
	for _, pe := range perrs {
		rel, err := filepath.Rel(c.root, pe.Path)
		if err != nil {
			rel = pe.Path
		}
		c.res.Skipped = append(c.res.Skipped, Skip{filepath.ToSlash(rel), "", "파일을 파싱하지 못해 선언을 옮기지 않았다: " + pe.Err.Error()})
	}
	fresh := func(n *ast.Ident) bool { return n.IsExported() && !known[n.Name] }
	var hand []*ast.File
	for _, f := range files {
		if !golang.IsDaemokGeneratedFile(f) {
			hand = append(hand, f)
		}
	}
	for k, v := range docs(hand) {
		c.docs[k] = v
	}
	enumTypes, enumValues := c.enums(module, hand, fresh)
	local := c.designTypes(module)
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
