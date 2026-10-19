package infer

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

type Use struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Names []Name `json:"names"`
}

type Name struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type fileRef struct {
	imports []fileImport
	sels    map[string]map[string]bool
}

type fileImport struct {
	dep   string
	alias string
}

func exportedDecls(files []*ast.File) map[string]string {
	out := map[string]string{}
	for _, f := range files {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					out[d.Name.Name] = "함수"
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch s := s.(type) {
					case *ast.TypeSpec:
						if !s.Name.IsExported() {
							continue
						}
						if _, ok := s.Type.(*ast.InterfaceType); ok {
							out[s.Name.Name] = "인터페이스"
						} else {
							out[s.Name.Name] = "타입"
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							switch {
							case !n.IsExported():
							case d.Tok == token.CONST:
								out[n.Name] = "상수"
							case strings.HasPrefix(n.Name, "Err"):
								out[n.Name] = "에러"
							default:
								out[n.Name] = "변수"
							}
						}
					}
				}
			}
		}
	}
	return out
}

func selectors(f *ast.File) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok {
			if out[x.Name] == nil {
				out[x.Name] = map[string]bool{}
			}
			out[x.Name][sel.Sel.Name] = true
		}
		return true
	})
	return out
}

func uses(byDir map[string]*pkg, ids map[string]string, libIDs map[string]string) []Use {
	var out []Use
	for dir, from := range ids {
		names := map[string]map[string]bool{}
		for _, fr := range byDir[dir].refs {
			for _, im := range fr.imports {
				if _, ok := libIDs[im.dep]; !ok || im.dep == dir {
					continue
				}
				target := byDir[im.dep]
				local := im.alias
				if local == "" {
					local = target.name
				}
				for n := range fr.sels[local] {
					if _, ok := target.decls[n]; ok {
						if names[im.dep] == nil {
							names[im.dep] = map[string]bool{}
						}
						names[im.dep][n] = true
					}
				}
			}
		}
		for dep, set := range names {
			u := Use{From: from, To: libIDs[dep]}
			for n := range set {
				u.Names = append(u.Names, Name{n, byDir[dep].decls[n]})
			}
			sort.Slice(u.Names, func(i, j int) bool { return u.Names[i].Name < u.Names[j].Name })
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func fileRefOf(f *ast.File, module string) fileRef {
	fr := fileRef{sels: selectors(f)}
	for _, im := range f.Imports {
		ip := strings.Trim(im.Path.Value, "`\"")
		var dep string
		switch {
		case ip == module:
			dep = "."
		case strings.HasPrefix(ip, module+"/"):
			dep = strings.TrimPrefix(ip, module+"/")
		default:
			continue
		}
		alias := ""
		if im.Name != nil {
			alias = im.Name.Name
		}
		if alias == "_" || alias == "." {
			continue
		}
		fr.imports = append(fr.imports, fileImport{dep, alias})
	}
	return fr
}
