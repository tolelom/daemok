// Package infer는 설계 파일이 없는 Go 모듈의 코드에서 L2 설계 초안을 만든다.
// 기존 프로젝트를 처음 가져올 때만 쓴다.
package infer

import (
	"errors"
	"fmt"
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

// pkg는 Go 파일이 있는 디렉토리 하나다. 경로는 루트 기준 슬래시 경로이고 루트는 "."다.
type pkg struct {
	dir     string
	main    bool
	imports []string // 같은 모듈 안의 패키지 디렉토리, 정렬
	ext     []string // 표준 라이브러리가 아닌 다른 모듈의 import 경로, 정렬
	doc     string   // 패키지 doc 주석의 첫 문장
}

// Skip은 초안에 넣지 않은 디렉토리나 파일과 그 이유다.
type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

func modulePath(root string) (string, error) {
	module, _, err := readGoMod(root)
	return module, err
}

// readGoMod는 go.mod의 module 경로와 직접 require(// indirect가 아닌 것)를 준다.
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
			// 하위 디렉토리 하나를 읽지 못해도 나머지는 분석한다.
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
			if gosrc.SkipDir(d.Name()) || strings.HasPrefix(d.Name(), "_") {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				skips = append(skips, Skip{rel(p), "중첩 Go 모듈"})
				return filepath.SkipDir
			}
		}
		files, errs := gosrc.Files(fset, p, parser.ImportsOnly)
		for _, e := range errs {
			skips = append(skips, Skip{rel(e.Path), "Go 파일을 읽을 수 없다: " + e.Err.Error()})
		}
		if len(files) == 0 {
			return nil
		}
		pk := &pkg{dir: rel(p)}
		seen := map[string]bool{}
		for _, f := range files {
			if f.Name.Name == "main" {
				pk.main = true
			}
			if pk.doc == "" && f.Doc != nil {
				pk.doc = strings.TrimSpace(new(doc.Package).Synopsis(f.Doc.Text()))
			}
			for _, im := range f.Imports {
				ip, _ := strconv.Unquote(im.Path.Value)
				var dep string
				switch {
				case ip == module:
					dep = "."
				case strings.HasPrefix(ip, module+"/"):
					dep = strings.TrimPrefix(ip, module+"/")
				default:
					// 첫 경로 요소에 점이 없으면 표준 라이브러리다.
					if first, _, _ := strings.Cut(ip, "/"); strings.Contains(first, ".") && !seen["ext:"+ip] {
						seen["ext:"+ip] = true
						pk.ext = append(pk.ext, ip)
					}
					continue
				}
				if dep != pk.dir && !seen[dep] {
					seen[dep] = true
					pk.imports = append(pk.imports, dep)
				}
			}
		}
		sort.Strings(pk.imports)
		sort.Strings(pk.ext)
		pkgs = append(pkgs, pk)
		return nil
	})
	return pkgs, skips, err
}
