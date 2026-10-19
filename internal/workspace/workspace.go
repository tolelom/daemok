package workspace

import (
	"bytes"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/codescan"
	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gen/ts"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type Plan struct {
	Write  []golang.File
	Delete []string
	Refuse []Refusal
}

type Refusal struct {
	Path   string
	Reason string
}

type Result struct {
	Model  *model.Model
	Diags  model.Diagnostics
	Fatal  bool
	Plan   *Plan
	Files  []golang.File
	Others model.Diagnostics
	Code   model.Diagnostics
}

func Run(designDir, root string) Result {
	m, diags := model.Load(designDir)
	return run(m, diags, designDir, root)
}

func Check(designDir, root string) Result {
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		var diags model.Diagnostics
		diags.Errorf(yaml.Pos{File: root}, "코드 루트가 없다")
		return Result{Diags: diags, Fatal: true}
	}
	return Run(designDir, root)
}

func RunSources(files map[string][]byte, designDir, root string) Result {
	m, diags := model.LoadSources(files)
	return run(m, diags, designDir, root)
}

func run(m *model.Model, diags model.Diagnostics, designDir, root string) Result {
	if diags.HasErrors() {
		diags.Sort()
		return Result{Model: m, Diags: diags, Fatal: true}
	}
	diags = append(diags, check.Run(m)...)
	if diags.HasErrors() {
		diags.Sort()
		return Result{Model: m, Diags: diags}
	}
	files, err := golang.Generate(m)
	if err != nil {
		diags.Errorf(yaml.Pos{File: designDir}, "생성할 수 없다: %v", err)
		diags.Sort()
		return Result{Model: m, Diags: diags}
	}
	existing, err := ScanGenerated(root)
	if err != nil {
		diags.Errorf(yaml.Pos{File: root}, "코드 루트를 읽을 수 없다: %v", err)
		diags.Sort()
		return Result{Model: m, Diags: diags, Fatal: true}
	}
	files = withTS(m, root, files, existing, &diags)
	plan, fresh := planFiles(root, files, existing)
	scanned := codescan.Scan(m, root)
	diags = append(diags, scanned...)
	diags.Sort()
	all := append(slices.Clone(diags), fresh...)
	all.Sort()
	code := append(slices.Clone(fresh), scanned...)
	return Result{Model: m, Diags: all, Plan: plan, Files: files, Others: diags, Code: code}
}

func planFiles(root string, files []golang.File, existing map[string][]byte) (*Plan, model.Diagnostics) {
	var fresh model.Diagnostics
	plan := &Plan{}
	for _, f := range files {
		pos := yaml.Pos{File: filepath.Join(root, filepath.FromSlash(f.Path))}
		content, isGen := existing[f.Path]
		if reason := blocked(root, f.Path); reason != "" {
			plan.Refuse = append(plan.Refuse, Refusal{f.Path, reason})
			switch reason {
			case "심볼릭 링크":
				fresh.Errorf(pos, "생성 경로에 심볼릭 링크가 있다: %s", f.Path)
			case "디렉토리":
				fresh.Errorf(pos, "생성 경로에 디렉토리가 있다: %s", f.Path)
			default:
				fresh.Errorf(pos, "생성 헤더가 없는 파일이 생성 경로에 있다: %s", f.Path)
			}
			continue
		}
		switch {
		case !isGen:
			plan.Write = append(plan.Write, f)
			fresh.Errorf(pos, "생성 파일이 없다: %s (daemok gen)", f.Path)
		case !bytes.Equal(content, f.Content):
			plan.Write = append(plan.Write, f)
			fresh.Errorf(pos, "생성 파일이 설계와 다르다: %s (daemok gen)", f.Path)
		}
	}
	for _, p := range staleFiles(files, existing) {
		plan.Delete = append(plan.Delete, p)
		fresh.Errorf(yaml.Pos{File: filepath.Join(root, filepath.FromSlash(p))}, "설계에 없는 생성 파일: %s (daemok gen)", p)
	}
	return plan, fresh
}

func staleFiles(files []golang.File, existing map[string][]byte) []string {
	out := map[string]bool{}
	for _, f := range files {
		out[f.Path] = true
	}
	var stale []string
	for p := range existing {
		if !out[p] {
			stale = append(stale, p)
		}
	}
	sort.Strings(stale)
	return stale
}

func withTS(m *model.Model, root string, files []golang.File, existing map[string][]byte, diags *model.Diagnostics) []golang.File {
	content, ok := ts.Generate(m)
	if !ok {
		return files
	}
	p := path.Clean(m.TSOut)
	full := filepath.Join(root, filepath.FromSlash(p))
	if st, err := os.Stat(filepath.Dir(full)); err != nil || !st.IsDir() {
		diags.Warnf(m.TSOutPos, "ts_out 디렉토리가 없어 TS 파일을 쓰지 않는다: %s", p)
		return files
	}
	if hasHeader(full) {
		if src, err := os.ReadFile(full); err == nil {
			existing[p] = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
		}
	}
	return append(files, golang.File{Path: p, Content: content})
}

func blocked(root, slashPath string) string {
	p := root
	parts := strings.Split(slashPath, "/")
	for i, el := range parts {
		p = filepath.Join(p, el)
		st, err := os.Lstat(p)
		if err != nil {
			return ""
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return "심볼릭 링크"
		}
		if i == len(parts)-1 {
			if st.IsDir() {
				return "디렉토리"
			}
			if !hasHeader(p) {
				return "생성 헤더가 없는 파일"
			}
		}
	}
	return ""
}

func hasHeader(p string) bool {
	src, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	return golang.IsDaemokGenerated(src)
}

func DefaultDesignDir(root string) string { return filepath.Join(root, "design") }

func SkipDir(name string) bool { return gosrc.SkipDir(name) }

func ScanGenerated(root string) (map[string][]byte, error) {
	out := map[string][]byte{}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return out, nil
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && SkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
		if !golang.IsDaemokGenerated(src) {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = src
		return nil
	})
	return out, err
}

func Apply(root string, plan *Plan) error {
	for _, f := range plan.Write {
		p := filepath.Join(root, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, f.Content, 0o644); err != nil {
			return err
		}
	}
	for _, d := range plan.Delete {
		p := filepath.Join(root, filepath.FromSlash(d))
		if err := os.Remove(p); err != nil {
			return err
		}
		removeEmptyDirs(root, filepath.Dir(p))
	}
	return nil
}

func removeEmptyDirs(root, dir string) {
	rootAbs, _ := filepath.Abs(root)
	for {
		abs, _ := filepath.Abs(dir)
		if abs == rootAbs || !strings.HasPrefix(abs, rootAbs+string(filepath.Separator)) {
			return
		}
		if entries, err := os.ReadDir(dir); err != nil || len(entries) > 0 {
			return
		}
		if os.Remove(dir) != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
