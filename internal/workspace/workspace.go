// Package workspace는 설계 읽기, 설계 검사, 생성, 디스크의 생성 파일 대조를 묶는다.
// check·gen 명령과 serve가 같은 진단 목록을 쓰도록 한 곳에 둔다.
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

// Plan은 gen이 할 일이다. 경로는 코드 루트 기준 슬래시 경로다.
type Plan struct {
	Write  []golang.File // 없거나 내용이 다른 생성 파일
	Delete []string      // 생성 헤더가 있고 이번 결과에 없는 파일
	Refuse []Refusal     // 쓰지 않는 생성 경로
}

// Refusal은 생성 경로를 쓰지 않는 이유다.
type Refusal struct {
	Path   string
	Reason string // 생성 헤더가 없는 파일, 디렉토리, 심볼릭 링크
}

// Result는 한 번 돌린 결과다.
type Result struct {
	Model *model.Model
	Diags model.Diagnostics
	Fatal bool          // 설계 파일을 읽지 못했다
	Plan  *Plan         // 설계 검사와 생성이 성공했을 때만 있다
	Files []golang.File // 생성 파일 전부. 설계 검사와 생성이 성공했을 때만 있다
	// Others는 Diags에서 생성 파일 대조를 뺀 진단이다. gen이 출력한다.
	Others model.Diagnostics
	// Code는 Diags 가운데 코드를 읽어 낸 진단(생성 파일 대조와 codescan)이다.
	Code model.Diagnostics
}

// Run은 designDir의 설계를 읽고 검사하고 생성해서 root의 파일과 대조한다.
func Run(designDir, root string) Result {
	m, diags := model.Load(designDir)
	return run(m, diags, designDir, root)
}

// Check는 코드 대조까지 하는 Run이다. 코드 루트가 없으면 대조할 코드가 없어 통과로 보이므로 진행하지 않는다.
// gen은 없는 루트에 처음 생성할 수 있어야 하므로 Run을 그대로 쓴다.
func Check(designDir, root string) Result {
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		var diags model.Diagnostics
		diags.Errorf(yaml.Pos{File: root}, "코드 루트가 없다")
		return Result{Diags: diags, Fatal: true}
	}
	return Run(designDir, root)
}

// RunSources는 디스크가 아닌 설계 파일 내용으로 Run과 같은 일을 한다. 키는 진단 위치에 쓰는 파일 이름이다.
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
	var fresh model.Diagnostics
	plan := &Plan{}
	out := map[string]bool{}
	for _, f := range files {
		out[f.Path] = true
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
	var stale []string
	for p := range existing {
		if !out[p] {
			stale = append(stale, p)
		}
	}
	sort.Strings(stale)
	for _, p := range stale {
		plan.Delete = append(plan.Delete, p)
		fresh.Errorf(yaml.Pos{File: filepath.Join(root, filepath.FromSlash(p))}, "설계에 없는 생성 파일: %s (daemok gen)", p)
	}
	scanned := codescan.Scan(m, root)
	diags = append(diags, scanned...)
	diags.Sort()
	all := append(slices.Clone(diags), fresh...)
	all.Sort()
	code := append(slices.Clone(fresh), scanned...)
	return Result{Model: m, Diags: all, Plan: plan, Files: files, Others: diags, Code: code}
}

// withTS는 TS 생성 파일을 생성 목록에 더하고, 디스크의 것이 생성 파일이면 대조 대상에 넣는다.
// ts_out은 코드 루트 밖일 수 있어 ScanGenerated가 찾지 않는다. 상위 디렉토리가 없으면 프론트 저장소가
// 없는 곳으로 보고 쓰지 않는다. 경고는 gen도 출력하도록 대조 진단이 아닌 diags에 넣는다.
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

// blocked는 생성 경로에 쓰면 안 되는 이유를 준다. 쓸 수 있으면 빈 문자열이다.
// 코드 루트 아래 경로 요소 중 심볼릭 링크가 있거나, 그 자리에 디렉토리나 생성 헤더 없는 파일이 있으면 막는다.
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

// DefaultDesignDir는 설계 디렉토리를 따로 주지 않았을 때 쓰는 코드 루트 아래 design이다.
func DefaultDesignDir(root string) string { return filepath.Join(root, "design") }

// SkipDir는 생성 파일 범위와 코드 감시에서 빼는 디렉토리다.
func SkipDir(name string) bool { return gosrc.SkipDir(name) }

// ScanGenerated는 root 아래에서 첫 줄이 생성 헤더인 .go 파일을 찾아
// 슬래시 경로와 내용(CRLF를 LF로 바꾼 것)을 준다.
func ScanGenerated(root string) (map[string][]byte, error) {
	out := map[string][]byte{}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return out, nil // 아직 없는 코드 루트: 처음 생성할 때 만든다
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

// Apply는 계획대로 파일을 쓰고 지운다. 거부할 파일이 있으면 호출하지 않는다.
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

// removeEmptyDirs는 dir부터 코드 루트 바로 아래까지 비게 된 디렉토리를 지운다.
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
