package infer

import (
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

// Use는 모듈·실행 단위 하나가 같은 Go 모듈의 다른 모듈에서 가져다 쓰는 export 이름이다.
type Use struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Names []Name `json:"names"`
}

// Name은 쓰인 이름과 그 선언의 종류(함수, 타입, 인터페이스, 상수, 에러, 변수)다.
type Name struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// fileRef는 파일 하나의 같은 모듈 import와 그 파일에 나온 "식별자.이름" 선택자다.
type fileRef struct {
	imports []fileImport
	sels    map[string]map[string]bool // 식별자 → 이름
}

type fileImport struct {
	dep   string // 패키지 디렉토리
	alias string // 없으면 ""
}

// exportedDecls는 패키지의 export된 최상위 선언 이름과 종류다. 메서드는 넣지 않는다.
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

// selectors는 파일에 나온 "식별자.이름" 쌍이다. 식별자가 지역 변수인지는 가리지 않고, 쓰는 쪽에서
// 대상 패키지의 export 선언과 맞는 이름만 남긴다.
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

// uses는 ids(디렉토리 → 모듈·실행 단위 ID)에 있는 패키지끼리 쓰는 이름을 모은다. 이름 없이 import만
// 하는 관계(빈 import 등)는 넣지 않는다.
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

// fileRefOf는 파일의 같은 모듈 import와 선택자를 모은다. "_"와 "." import는 이름으로 쓰이지 않아 뺀다.
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
