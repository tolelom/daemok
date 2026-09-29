package golang

import (
	"path"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// stdPackages는 외부 타입 접두사 중 표준 라이브러리로 import하는 것이다.
var stdPackages = map[string]string{
	"json": "encoding/json", "time": "time", "http": "net/http", "context": "context", "io": "io",
}

// gen은 한 번의 생성 상태다.
type gen struct {
	m     *model.Model
	files map[string]*goFile
	order []string
}

// ownerPath는 모듈·멤버 ID의 코드 경로다.
func (g *gen) ownerPath(id string) string {
	o, _ := g.m.Owner(id)
	return o.Path
}

func (g *gen) importPath(dir string) string { return g.m.GoModule + "/" + dir }

// file은 경로의 파일을 만들거나 이미 있는 것을 준다. 패키지 이름은 디렉토리의 마지막 요소다.
func (g *gen) file(p string) *goFile {
	if f, ok := g.files[p]; ok {
		return f
	}
	f := &goFile{path: p, pkg: path.Base(path.Dir(p)), imports: map[string]string{}}
	if strings.HasPrefix(p, "cmd/") {
		f.pkg = "main"
	}
	g.files[p] = f
	g.order = append(g.order, p)
	return f
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}


// goType은 설계 타입 문자열을 from 디렉토리의 파일 f에서 쓸 Go 타입으로 바꾼다.
func (g *gen) goType(t string, fromDir string, f *goFile) string {
	prefix, base := model.SplitType(t)
	kind, _ := g.m.ResolveType(t)
	switch kind {
	case model.BuiltinType:
		name := model.Builtins[base]
		if strings.HasPrefix(name, "time.") {
			f.use("time")
		}
		return prefix + name
	case model.EnumType:
		e := g.m.Enum(base)
		return prefix + g.qualify(e.In.Name, e.GoType(), fromDir, f)
	case model.StructType:
		st := g.m.Type(base)
		return prefix + g.qualify(st.In.Name, st.GoName(), fromDir, f)
	}
	if pkg, name, ok := strings.Cut(base, "."); ok {
		if _, isOwner := g.m.Owner(pkg); isOwner {
			return prefix + g.qualify(pkg, name, fromDir, f)
		}
		if std, ok := stdPackages[pkg]; ok {
			return prefix + f.use(std) + "." + name
		}
	}
	return prefix + base
}

// qualify는 owner 모듈의 이름을 fromDir에서 쓸 수 있게 패키지를 붙인다.
func (g *gen) qualify(owner, name, fromDir string, f *goFile) string {
	dir := g.ownerPath(owner)
	if dir == fromDir {
		return name
	}
	return f.use(g.importPath(dir)) + "." + name
}

// argType은 계약 시그니처의 타입이다. 설계 struct는 접두사가 없으면 포인터로 쓴다.
func (g *gen) argType(t string, fromDir string, f *goFile) string {
	prefix, _ := model.SplitType(t)
	gt := g.goType(t, fromDir, f)
	if kind, _ := g.m.ResolveType(t); kind == model.StructType && prefix == "" {
		return "*" + gt
	}
	return gt
}
