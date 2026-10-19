package infer

import (
	"errors"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/gosrc"
)

var ErrNotModule = errors.New("Go 모듈이 아니다")

type pkg struct {
	dir     string
	main    bool
	imports []string
	ext     []string
	doc     string
	hints   []Hint
	name    string
	decls   map[string]string
	refs    []fileRef
}

type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

func modulePath(root string) (string, error) {
	module, _, err := readGoMod(root)
	return module, err
}

func readGoMod(root string) (module string, requires []string, err error) {
	src, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", nil, ErrNotModule
	}
	inBlock := false
	for _, line := range strings.Split(string(src), "\n") {
		code, comment, _ := strings.Cut(line, "//")
		f := strings.Fields(code)
		switch {
		case len(f) >= 2 && f[0] == "module":
			module = strings.Trim(f[1], `"`)
		case len(f) >= 2 && f[0] == "require" && f[1] == "(":
			inBlock = true
		case inBlock && len(f) == 1 && f[0] == ")":
			inBlock = false
		case len(f) >= 3 && f[0] == "require", inBlock && len(f) >= 2:
			if strings.TrimSpace(comment) == "indirect" {
				continue
			}
			if f[0] == "require" {
				f = f[1:]
			}
			requires = append(requires, strings.Trim(f[0], `"`))
		}
	}
	if module == "" {
		return "", nil, fmt.Errorf("go.mod에 module 줄이 없다")
	}
	sort.Strings(requires)
	return module, requires, nil
}

func collect(root, module string) ([]*pkg, []Skip, error) {
	var pkgs []*pkg
	var skips []Skip
	fset := token.NewFileSet()
	rel := func(p string) string {
		r, _ := filepath.Rel(root, p)
		return filepath.ToSlash(r)
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == root {
				return err
			}
			skips = append(skips, Skip{rel(p), "읽을 수 없다: " + err.Error()})
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if p != root {
			if gosrc.SkipPackageDir(d.Name()) {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				skips = append(skips, Skip{rel(p), "중첩 Go 모듈"})
				return filepath.SkipDir
			}
		}
		files, all, fileSkips := readPackage(fset, p, rel)
		skips = append(skips, fileSkips...)
		if len(all) == 0 {
			return nil
		}
		pkgs = append(pkgs, packageOf(fset, root, module, rel(p), files, all))
		return nil
	})
	return pkgs, skips, err
}

func readPackage(fset *token.FileSet, dir string, rel func(string) string) (files, all []*ast.File, skips []Skip) {
	files, errs := gosrc.Files(fset, dir, parser.SkipObjectResolution)
	all = files
	for _, e := range errs {
		if f, err := gosrc.ImportsOnly(fset, e.Path); err == nil {
			if f != nil {
				all = append(all, f)
			}
			skips = append(skips, Skip{rel(e.Path), "본문을 읽을 수 없어 import만 반영했다: " + e.Err.Error()})
			continue
		}
		skips = append(skips, Skip{rel(e.Path), "Go 파일을 읽을 수 없다: " + e.Err.Error()})
	}
	return files, all, skips
}

func packageOf(fset *token.FileSet, root, module, dir string, files, all []*ast.File) *pkg {
	pk := &pkg{dir: dir, main: hasMainFile(all), doc: packageSynopsis(all)}
	pk.imports, pk.ext = packageImports(all, module, dir)
	types := stringTypes(files)
	for _, f := range files {
		pk.hints = append(pk.hints, fileHints(fset, root, f, types)...)
		pk.name = f.Name.Name
		pk.refs = append(pk.refs, fileRefOf(f, module))
	}
	pk.decls = exportedDecls(files)
	return pk
}

func hasMainFile(files []*ast.File) bool {
	for _, f := range files {
		if f.Name.Name == "main" {
			return true
		}
	}
	return false
}

func packageSynopsis(files []*ast.File) string {
	synopsis := ""
	for _, f := range files {
		if synopsis == "" && f.Doc != nil {
			synopsis = strings.TrimSpace(new(doc.Package).Synopsis(f.Doc.Text()))
		}
	}
	return synopsis
}

func packageImports(files []*ast.File, module, dir string) (imports, ext []string) {
	seen := map[string]bool{}
	for _, f := range files {
		for _, im := range f.Imports {
			ip, _ := strconv.Unquote(im.Path.Value)
			var dep string
			switch {
			case ip == module:
				dep = "."
			case strings.HasPrefix(ip, module+"/"):
				dep = strings.TrimPrefix(ip, module+"/")
			default:
				if first, _, _ := strings.Cut(ip, "/"); strings.Contains(first, ".") && !seen["ext:"+ip] {
					seen["ext:"+ip] = true
					ext = append(ext, ip)
				}
				continue
			}
			if dep != dir && !seen[dep] {
				seen[dep] = true
				imports = append(imports, dep)
			}
		}
	}
	sort.Strings(imports)
	sort.Strings(ext)
	return imports, ext
}
