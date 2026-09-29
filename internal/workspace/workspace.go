// Package workspace는 설계 읽기, 설계 검사, 생성, 디스크의 생성 파일 대조를 묶는다.
// check·gen 명령과 serve가 같은 진단 목록을 쓰도록 한 곳에 둔다.
package workspace

import (
	"bufio"
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/codescan"
	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

// Plan은 gen이 할 일이다. 경로는 코드 루트 기준 슬래시 경로다.
type Plan struct {
	Write  []golang.File // 없거나 내용이 다른 생성 파일
	Delete []string      // 생성 헤더가 있고 이번 결과에 없는 파일
	Refuse []string      // 생성 경로에 있는 생성 헤더 없는 파일
}

// Result는 한 번 돌린 결과다.
type Result struct {
	Model *model.Model
	Diags model.Diagnostics
	Fatal bool  // 설계 파일을 읽지 못했다
	Plan  *Plan // 설계 검사와 생성이 성공했을 때만 있다
}

// Run은 designDir의 설계를 읽고 검사하고 생성해서 root의 파일과 대조한다.
func Run(designDir, root string) Result {
	m, diags := model.Load(designDir)
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
	plan := &Plan{}
	out := map[string]bool{}
	for _, f := range files {
		out[f.Path] = true
		pos := yaml.Pos{File: filepath.Join(root, filepath.FromSlash(f.Path))}
		content, isGen := existing[f.Path]
		switch {
		case !isGen && fileExists(root, f.Path):
			plan.Refuse = append(plan.Refuse, f.Path)
			diags.Errorf(pos, "생성 헤더가 없는 파일이 생성 경로에 있다: %s", f.Path)
		case !isGen:
			plan.Write = append(plan.Write, f)
			diags.Errorf(pos, "생성 파일이 없다: %s (daemok gen)", f.Path)
		case !bytes.Equal(content, f.Content):
			plan.Write = append(plan.Write, f)
			diags.Errorf(pos, "생성 파일이 설계와 다르다: %s (daemok gen)", f.Path)
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
		diags.Errorf(yaml.Pos{File: filepath.Join(root, filepath.FromSlash(p))}, "설계에 없는 생성 파일: %s (daemok gen)", p)
	}
	diags = append(diags, codescan.Scan(m, root)...)
	diags.Sort()
	return Result{Model: m, Diags: diags, Plan: plan}
}

func fileExists(root, slashPath string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(slashPath)))
	return err == nil
}

// skipDir는 생성 파일 범위에서 빼는 디렉토리다.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules"
}

// ScanGenerated는 root 아래에서 첫 줄이 생성 헤더인 .go 파일을 찾아
// 슬래시 경로와 내용(CRLF를 LF로 바꾼 것)을 준다.
func ScanGenerated(root string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
		first, _, _ := bufio.NewReader(bytes.NewReader(src)).ReadLine()
		if string(first) != golang.Header {
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
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(d))); err != nil {
			return err
		}
	}
	return nil
}
