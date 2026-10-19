package golang

import (
	"path"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

var stdPackages = map[string]string{
	"json": "encoding/json", "time": "time", "http": "net/http", "context": "context", "io": "io",
}

func StdImport(importPath string) bool {
	for _, p := range stdPackages {
		if p == importPath {
			return true
		}
	}
	return false
}

type gen struct {
	m     *model.Model
	files map[string]*goFile
	order []string
}

func (g *gen) ownerPath(id string) string {
	o, _ := g.m.Owner(id)
	return o.Path
}

func (g *gen) importPath(dir string) string { return g.m.GoModule + "/" + dir }

func (g *gen) file(p string) *goFile {
	if f, ok := g.files[p]; ok {
		return f
	}
	f := &goFile{path: p, pkg: path.Base(path.Dir(p)), imports: map[string]string{}}
	g.files[p] = f
	g.order = append(g.order, p)
	return f
}

func (g *gen) mainFile(p string) *goFile {
	f := g.file(p)
	f.pkg = "main"
	return f
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

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
	case model.MapType:
		key, value, _ := model.MapParts(base)
		return prefix + "map[" + g.goType(key, fromDir, f) + "]" + g.goType(value, fromDir, f)
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

func (g *gen) qualify(owner, name, fromDir string, f *goFile) string {
	dir := g.ownerPath(owner)
	if dir == fromDir {
		return name
	}
	return f.use(g.importPath(dir)) + "." + name
}

func (g *gen) argType(t string, fromDir string, f *goFile) string {
	prefix, _ := model.SplitType(t)
	gt := g.goType(t, fromDir, f)
	if kind, _ := g.m.ResolveType(t); kind == model.StructType && prefix == "" {
		return "*" + gt
	}
	return gt
}
