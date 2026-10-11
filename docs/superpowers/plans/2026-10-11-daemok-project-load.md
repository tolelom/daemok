# 프로젝트 열기와 코드 분석 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `daemok serve` 화면에서 프로젝트를 열고, 설계 파일이 없는 Go 모듈은 코드에서 L2 설계 초안을 만들어 보여 주고 `design/`에 저장한다.

**Architecture:** 새 `internal/gosrc`가 codescan과 같은 규칙으로 Go 파일을 고르고, 새 `internal/infer`가 패키지와 import에서 초안(`Draft`)과 YAML을 만든다. serve는 설계 파일 유무로 대조 모드와 분석 모드를 정하고, 분석 모드에서는 초안 YAML을 `workspace.RunSources`로 기존 검사와 코드 대조에 그대로 넣어 같은 화면 JSON을 만든다. 쓰기는 `infer.Write` 하나를 `daemok init`과 `POST /api/init`이 같이 쓴다.

**Tech Stack:** Go 1.27 표준 라이브러리(`go/parser`, `go/doc`, `net/http`), 임베드된 단일 `index.html`(바닐라 JS).

**Spec:** `docs/superpowers/specs/2026-10-11-daemok-project-load-design.md`

## Global Constraints

- 외부 의존성 없음. Go 표준 라이브러리만 쓴다.
- 설계 파일 형식 `version: 0.4`. 초안은 이 저장소의 YAML 부분집합 파서(`internal/yaml`)로 읽혀야 한다. 큰따옴표 문자열의 이스케이프는 `\"`, `\\`, `\n`, `\t`만 지원한다.
- 노드 ID는 `^[a-z][a-z0-9]*(_[a-z0-9]+)*$`, 모듈 path 요소는 `^[A-Za-z][A-Za-z0-9_]*$`(`internal/check/names.go`).
- 초안을 쓴 직후 `daemok check` 오류 0. 파싱할 수 없는 Go 파일의 오류만 예외.
- 쓰기는 `design/` 생성과 파일 하나의 `O_EXCL` 생성뿐. 덮어쓰기 없음.
- 종료 코드: 0 성공, 1 문제 있음(이미 설계 파일 있음), 2 진행 불가.
- 진단·오류 메시지는 기존 문체를 따른다("설명: 대상", 평서형 "~다").
- 기존 테스트는 동작이 바뀌는 `TestFirstStateWithoutDesignFiles` 말고는 고치지 않는다. 이 테스트도 기대값(`fatal` 참)은 그대로 통과해야 한다.
- 커밋 메시지 끝에 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. Windows 경로: 화면에서 `E:\dev\projects\tolchain`처럼 역슬래시 경로와 끝의 `\`, 앞뒤 공백을 붙여넣어도 열린다. Task 8에서 공백·끝 구분자 테스트를 둔다.
2. 패키지 doc 주석에 큰따옴표, 역슬래시, 줄바꿈, `#`, `:`가 들어 있어도 초안이 파서로 다시 읽힌다. Task 4의 왕복 테스트에 특수 문자 summary를 넣는다.
3. 설계 파일을 저장하는 순간과 1초 감시 루프가 겹쳐도 화면이 분석 모드에 머물지 않는다. Task 8에서 `init` 뒤 상태의 `mode`가 `check`인지 확인한다.
4. 열기 실패(없는 경로, 파일 경로, go.mod도 설계도 없는 폴더)에서 현재 프로젝트가 바뀌지 않는다. Task 8 테스트.
5. 다른 사이트에서 오는 POST(Origin 다름, Origin 없음, `text/plain` 본문)는 열기·저장을 하지 못한다. Task 9 테스트.

---

## 파일 구조

| 파일 | 책임 |
|---|---|
| `internal/gosrc/gosrc.go` (새) | `Files`(디렉토리의 Go 파일 파싱), `SkipDir`, build 조건 판정 |
| `internal/gosrc/gosrc_test.go` (새) | 파일 선택 규칙 |
| `internal/codescan/scan.go` | `goFiles`가 `gosrc.Files`를 부른다. `ignored`, `satisfiable` 삭제 |
| `internal/workspace/workspace.go` | `SkipDir`이 `gosrc.SkipDir`을 부른다. `RunSources` 추가 |
| `internal/infer/collect.go` (새) | `go.mod` 읽기, 패키지 수집 |
| `internal/infer/draft.go` (새) | `Draft`, `Report`, `Analyze`, ID·tier 계산 |
| `internal/infer/render.go` (새) | `Render` |
| `internal/infer/write.go` (새) | `Write`, `ErrDesignExists` |
| `internal/infer/*_test.go`, `internal/infer/testdata/` (새) | fixture와 테스트 |
| `cmd/daemok/init.go` (새) | `daemok init` |
| `cmd/daemok/main.go` | 명령 분기와 사용법 |
| `cmd/daemok/serve.go` | `--workspace`, 루프백 판정, `Options` |
| `internal/serve/server.go` | 현재 프로젝트 전환, 모드, `Options` |
| `internal/serve/projects.go` (새) | `Open`, 최근 목록, 작업 폴더, `/api/projects`·`/api/open`·`/api/init` |
| `internal/serve/guard.go` (새) | Host·Origin 검사 |
| `internal/serve/web/index.html` | 프로젝트 버튼, 열기 서랍, 분석 모드 배지, 초안 탭 |
| `docs/design/02-slice-tolchain.md` | 초안과 손으로 쓴 설계의 비교 기록 |

---

### Task 1: `internal/gosrc`로 파일 선택 옮기기

**Files:**
- Create: `internal/gosrc/gosrc.go`, `internal/gosrc/gosrc_test.go`
- Modify: `internal/codescan/scan.go:3-17`(import), `:68-140`(`goFiles`, `ignored`, `satisfiable`)
- Modify: `internal/workspace/workspace.go:155-158`(`SkipDir`)

**Interfaces:**
- Produces:
  - `gosrc.Files(fset *token.FileSet, dir string, mode parser.Mode) ([]*ast.File, []gosrc.ParseError)`
  - `type gosrc.ParseError struct { Path string; Err error }`
  - `gosrc.SkipDir(name string) bool`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/gosrc/gosrc_test.go`:

```go
package gosrc

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package p\n")
	write(t, dir, "a_test.go", "package p\n")
	write(t, dir, "linux.go", "//go:build linux\n\npackage p\n")
	write(t, dir, "tool.go", "//go:build ignore\n\npackage main\n")
	write(t, dir, "broken.go", "package p\nfunc (\n")
	write(t, dir, "note.txt", "x")
	os.Mkdir(filepath.Join(dir, "sub.go"), 0o755)
	files, errs := Files(token.NewFileSet(), dir, parser.SkipObjectResolution)
	var names []string
	for _, f := range files {
		names = append(names, f.Name.Name)
	}
	if len(files) != 2 {
		t.Fatalf("a.go와 linux.go만 읽어야 한다: %d개 %v", len(files), names)
	}
	if len(errs) != 1 || filepath.Base(errs[0].Path) != "broken.go" {
		t.Fatalf("errs %v", errs)
	}
}

func TestFilesImportsOnlyKeepsDoc(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "// Package p는 예제다.\npackage p\n\nimport \"fmt\"\n\nfunc broken( {\n")
	files, errs := Files(token.NewFileSet(), dir, parser.ImportsOnly)
	if len(errs) != 0 || len(files) != 1 {
		t.Fatalf("files %d errs %v", len(files), errs)
	}
	if files[0].Doc == nil || len(files[0].Imports) != 1 {
		t.Fatal("ImportsOnly에서도 doc 주석과 import가 있어야 한다")
	}
}

func TestFilesMissingDir(t *testing.T) {
	files, errs := Files(token.NewFileSet(), filepath.Join(t.TempDir(), "none"), 0)
	if files != nil || errs != nil {
		t.Fatal("없는 디렉토리는 비어 있다")
	}
}

func TestSkipDir(t *testing.T) {
	for name, want := range map[string]bool{".git": true, "vendor": true, "testdata": true, "node_modules": true, "core": false, "_x": false} {
		if SkipDir(name) != want {
			t.Errorf("%s: %v", name, !want)
		}
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/gosrc/`
Expected: FAIL, `undefined: Files`

- [ ] **Step 3: 구현**

`internal/gosrc/gosrc.go`. `ignored`와 `satisfiable`은 `internal/codescan/scan.go`의 함수 본문을 그대로 옮긴다.

```go
// Package gosrc는 디렉토리의 Go 소스 파일을 고르고 읽는다. codescan과 infer가 같은 파일 집합을 보도록 한 곳에 둔다.
package gosrc

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ParseError는 읽거나 파싱하지 못한 파일이다.
type ParseError struct {
	Path string
	Err  error
}

// Files는 dir의 Go 파일을 문법 트리로 준다. _test.go와 //go:build ignore 파일은 뺀다.
// OS·아키텍처 조건은 모두 참으로 보아 어느 플랫폼의 파일이든 읽는다. 디렉토리가 없으면 비어 있다.
// build 조건을 보려고 mode에 parser.ParseComments를 더한다.
func Files(fset *token.FileSet, dir string, mode parser.Mode) ([]*ast.File, []ParseError) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var files []*ast.File
	var errs []ParseError
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		p := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, p, nil, mode|parser.ParseComments)
		if err != nil {
			errs = append(errs, ParseError{p, err})
			continue
		}
		if ignored(f) {
			continue
		}
		files = append(files, f)
	}
	return files, errs
}

// SkipDir는 코드를 읽지 않는 디렉토리다.
func SkipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules"
}

// ignored는 파일의 build 조건이 ignore 태그 없이는 참이 될 수 없는지다.
func ignored(f *ast.File) bool {
	// internal/codescan/scan.go의 ignored 본문 그대로
}

// satisfiable은 ignore 태그를 거짓으로 두고 나머지 태그를 어떻게 정해 참이 될 수 있는지다.
// 태그가 많으면 판단하지 않고 참으로 본다.
func satisfiable(expr constraint.Expr) bool {
	// internal/codescan/scan.go의 satisfiable 본문 그대로
}
```

`internal/codescan/scan.go`: `ignored`, `satisfiable`을 지우고 `goFiles`를 바꾼다. import에서 `go/build/constraint`, `os`는 다른 곳에서 쓰지 않으면 지운다(`os`는 `dirExists`가 쓰므로 남는다).

```go
// goFiles는 모듈·멤버 디렉토리의 Go 파일이다. 파일 선택은 gosrc.Files를 따른다.
func (s *scanner) goFiles(dir string) []*ast.File {
	files, errs := gosrc.Files(s.fset, filepath.Join(s.root, filepath.FromSlash(dir)), parser.SkipObjectResolution)
	for _, e := range errs {
		s.diags.Errorf(yaml.Pos{File: e.Path}, "Go 파일을 읽을 수 없다: %v", e.Err)
	}
	return files
}
```

`internal/workspace/workspace.go`:

```go
// SkipDir는 생성 파일 범위와 코드 감시에서 빼는 디렉토리다.
func SkipDir(name string) bool { return gosrc.SkipDir(name) }
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/gosrc/ ./internal/codescan/ ./internal/workspace/ ./internal/serve/`
Expected: PASS. codescan 테스트 파일은 바꾸지 않는다.

- [ ] **Step 5: 커밋**

```bash
git add internal/gosrc internal/codescan/scan.go internal/workspace/workspace.go
git commit -m "refactor: Go 파일 선택을 gosrc로"
```

---

### Task 2: infer 패키지 수집

**Files:**
- Create: `internal/infer/collect.go`, `internal/infer/collect_test.go`
- Create fixture: `internal/infer/testdata/layers/...`, `internal/infer/testdata/rootmain/...`, `internal/infer/testdata/nomod/x.go`

**Interfaces:**
- Consumes: `gosrc.Files`, `gosrc.SkipDir`
- Produces (패키지 내부):
  - `var ErrNotModule = errors.New("Go 모듈이 아니다")`
  - `func modulePath(root string) (string, error)`
  - `type pkg struct { dir string; main bool; imports []string; doc string }` (`dir`과 `imports`는 루트 기준 슬래시 경로, 루트는 `"."`)
  - `type Skip struct { Path string \`json:"path"\`; Reason string \`json:"reason"\` }`
  - `func collect(root, module string) ([]*pkg, []Skip, error)`

- [ ] **Step 1: fixture 작성**

`internal/infer/testdata/layers/`:

| 파일 | 내용 |
|---|---|
| `go.mod` | `module example.com/layers` + 빈 줄 + `go 1.27` |
| `cmd/app/main.go` | `// app은 예제 실행 파일이다.` / `package main` / `import _ "example.com/layers/api"` / `func main() {}` |
| `api/api.go` | `package api` / `import ( _ "example.com/layers/core"; _ "example.com/layers/util"; _ "example.com/layers/bad-name"; _ "fmt" )` |
| `core/core.go` | `// Package core는 도메인 타입이다.` / `package core` / `import _ "example.com/layers/util"` |
| `core/core_test.go` | `package core` / `import _ "example.com/layers/api"` |
| `util/util.go` | `package util` |
| `util/ignore.go` | `//go:build ignore` / 빈 줄 / `package util` / `import _ "example.com/layers/core"` |
| `internal/util/util.go` | `package util` |
| `certGen/cert.go` | `package certGen` / `import _ "example.com/layers/internal/util"` |
| `bad-name/x.go` | `package badname` |
| `nested/go.mod` | `module example.com/nested` |
| `nested/x.go` | `package nested` |
| `_hidden/x.go` | `package hidden` |

`internal/infer/testdata/rootmain/`:

| 파일 | 내용 |
|---|---|
| `go.mod` | `module example.com/tool-kit/v2` |
| `main.go` | `package main` / `import _ "example.com/tool-kit/v2/lib"` / `func main() {}` |
| `lib/lib.go` | `// Package lib는 "따옴표"와 \역슬래시, #, : 를 쓴다.` / `package lib` |

`internal/infer/testdata/nomod/x.go`: `package x`

- [ ] **Step 2: 실패하는 테스트 작성**

`internal/infer/collect_test.go`:

```go
package infer

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestModulePath(t *testing.T) {
	if p, err := modulePath(filepath.Join("testdata", "layers")); err != nil || p != "example.com/layers" {
		t.Fatalf("%q %v", p, err)
	}
	if _, err := modulePath(filepath.Join("testdata", "nomod")); !errors.Is(err, ErrNotModule) {
		t.Fatalf("%v", err)
	}
}

func TestCollect(t *testing.T) {
	pkgs, skips, err := collect(filepath.Join("testdata", "layers"), "example.com/layers")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]pkg{}
	for _, p := range pkgs {
		got[p.dir] = *p
	}
	want := map[string]pkg{
		"api":           {dir: "api", imports: []string{"bad-name", "core", "util"}},
		"bad-name":      {dir: "bad-name"},
		"certGen":       {dir: "certGen", imports: []string{"internal/util"}},
		"cmd/app":       {dir: "cmd/app", main: true, imports: []string{"api"}, doc: "app은 예제 실행 파일이다."},
		"core":          {dir: "core", imports: []string{"util"}, doc: "Package core는 도메인 타입이다."},
		"internal/util": {dir: "internal/util"},
		"util":          {dir: "util"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	if !reflect.DeepEqual(skips, []Skip{{"nested", "중첩 Go 모듈"}}) {
		t.Fatalf("skips %v", skips)
	}
}

func TestCollectRootPackage(t *testing.T) {
	pkgs, _, err := collect(filepath.Join("testdata", "rootmain"), "example.com/tool-kit/v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 2 || pkgs[0].dir != "." || !pkgs[0].main || !reflect.DeepEqual(pkgs[0].imports, []string{"lib"}) {
		t.Fatalf("%+v", pkgs[0])
	}
}
```

- [ ] **Step 3: 실패 확인**

Run: `go test ./internal/infer/`
Expected: FAIL, `undefined: modulePath`

- [ ] **Step 4: 구현**

`internal/infer/collect.go`:

```go
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
	doc     string   // 패키지 doc 주석의 첫 문장
}

// Skip은 초안에 넣지 않은 디렉토리나 파일과 그 이유다.
type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

func modulePath(root string) (string, error) {
	src, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", ErrNotModule
	}
	for _, line := range strings.Split(string(src), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "module" {
			return strings.Trim(f[1], `"`), nil
		}
	}
	return "", fmt.Errorf("go.mod에 module 줄이 없다")
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
			return err
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
					continue
				}
				if dep != pk.dir && !seen[dep] {
					seen[dep] = true
					pk.imports = append(pk.imports, dep)
				}
			}
		}
		sort.Strings(pk.imports)
		pkgs = append(pkgs, pk)
		return nil
	})
	return pkgs, skips, err
}
```

- [ ] **Step 5: 통과 확인**

Run: `go test ./internal/infer/`
Expected: PASS. `new(doc.Package).Synopsis`의 결과가 기대 문장과 다르면(마침표 처리 등) 테스트 기대값이 아니라 실제 출력을 확인하고 원인을 적은 뒤 기대값을 맞춘다.

- [ ] **Step 6: 커밋**

```bash
git add internal/infer
git commit -m "feat: infer: go.mod와 패키지·import 수집"
```

---

### Task 3: infer 초안 계산

**Files:**
- Create: `internal/infer/draft.go`, `internal/infer/draft_test.go`

**Interfaces:**
- Consumes: Task 2의 `pkg`, `Skip`, `collect`, `modulePath`
- Produces:
  - `type Exe struct { ID, Dir, Summary string; Composes []string }`
  - `type Mod struct { ID, Path, Summary string; Tier int; DependsOn []string }` (`Path`는 ID와 다를 때만 비지 않음)
  - `type Draft struct { GoModule, System string; Exes []Exe; Tiers int; Mods []Mod }`
  - `type FanIn struct { ID string \`json:"id"\`; Count int \`json:"count"\` }`
  - `type Report struct { Packages int \`json:"packages"\`; Modules int \`json:"modules"\`; Executables int \`json:"executables"\`; Skipped []Skip \`json:"skipped"\`; FanIn []FanIn \`json:"fanIn"\` }`
  - `type Analysis struct { Draft *Draft; YAML []byte; Report Report }`
  - `func Analyze(root string) (*Analysis, error)` (이 태스크에서는 `YAML`을 비워 두고 Task 4에서 채운다)
  - `func snake(s string) string`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/infer/draft_test.go`:

```go
package infer

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSnake(t *testing.T) {
	for in, want := range map[string]string{
		"core": "core", "certGen": "cert_gen", "HTTPServer": "http_server", "v2": "v2",
		"a1B": "a1_b", "tool-kit": "tool_kit", "API": "api", "x__y_": "x_y",
	} {
		if got := snake(in); got != want {
			t.Errorf("snake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnalyzeLayers(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "layers"))
	if err != nil {
		t.Fatal(err)
	}
	want := &Draft{
		GoModule: "example.com/layers",
		System:   "layers",
		Exes:     []Exe{{ID: "app", Dir: "cmd/app", Summary: "app은 예제 실행 파일이다.", Composes: []string{"api"}}},
		Tiers:    3,
		Mods: []Mod{
			{ID: "internal_util", Path: "internal/util", Tier: 0},
			{ID: "util", Tier: 0},
			{ID: "cert_gen", Path: "certGen", Tier: 1, DependsOn: []string{"internal_util"}},
			{ID: "core", Tier: 1, Summary: "Package core는 도메인 타입이다.", DependsOn: []string{"util"}},
			{ID: "api", Tier: 2, DependsOn: []string{"core", "util"}},
		},
	}
	if !reflect.DeepEqual(a.Draft, want) {
		t.Fatalf("got  %+v\nwant %+v", a.Draft, want)
	}
	wantRep := Report{
		Packages: 7, Modules: 5, Executables: 1,
		Skipped: []Skip{{"bad-name", "Go 패키지 경로로 쓸 수 없는 디렉토리 이름"}, {"nested", "중첩 Go 모듈"}},
		FanIn:   []FanIn{{"util", 2}, {"api", 1}, {"core", 1}, {"internal_util", 1}},
	}
	if !reflect.DeepEqual(a.Report, wantRep) {
		t.Fatalf("got  %+v\nwant %+v", a.Report, wantRep)
	}
}

func TestAnalyzeRootMain(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "rootmain"))
	if err != nil {
		t.Fatal(err)
	}
	d := a.Draft
	if d.System != "tool_kit" || len(d.Exes) != 1 || d.Exes[0].ID != "tool_kit" || d.Exes[0].Dir != "." {
		t.Fatalf("%+v", d)
	}
	if len(d.Mods) != 1 || d.Mods[0].ID != "lib" || d.Mods[0].Summary != `Package lib는 "따옴표"와 \역슬래시, #, : 를 쓴다.` {
		t.Fatalf("%+v", d.Mods)
	}
}

func TestAnalyzeNotModule(t *testing.T) {
	if _, err := Analyze(filepath.Join("testdata", "nomod")); !errors.Is(err, ErrNotModule) {
		t.Fatalf("%v", err)
	}
}

func TestAssignIDs(t *testing.T) {
	got := assignIDs([]string{"a/util", "b/util", "util", "x/a/b_c", "x/a_b/c"}, "sys")
	want := map[string]string{"a/util": "a_util", "b/util": "b_util", "util": "util", "x/a/b_c": "x_a_b_c", "x/a_b/c": "x_a_b_c_2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}
```

`util`은 경로 요소가 하나라 더 늘릴 수 없어 그대로 남고, `a/util`, `b/util`만 늘어난다. `x/a/b_c`와 `x/a_b/c`는 경로 전체(`x_a_b_c`)로도 겹쳐서 디렉토리 순서대로 두 번째에 `_2`가 붙는다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/infer/`
Expected: FAIL, `undefined: snake`

- [ ] **Step 3: 구현**

`internal/infer/draft.go`:

```go
package infer

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	goIdent   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	snakeName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)
	version   = regexp.MustCompile(`^v[0-9]+$`)
)

// Exe는 초안의 실행 단위다.
type Exe struct {
	ID, Dir, Summary string
	Composes         []string
}

// Mod는 초안의 모듈이다. Path는 ID와 다를 때만 있다.
type Mod struct {
	ID, Path, Summary string
	Tier              int
	DependsOn         []string
}

// Draft는 코드에서 추정한 L2 설계다. tier 이름은 tier0부터 tier(Tiers-1)까지다.
type Draft struct {
	GoModule, System string
	Exes             []Exe
	Tiers            int
	Mods             []Mod
}

// FanIn은 모듈을 import하는 모듈·실행 단위의 수다.
type FanIn struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// Report는 분석 모드 화면과 daemok init이 보여 주는 메모다.
type Report struct {
	Packages    int     `json:"packages"`
	Modules     int     `json:"modules"`
	Executables int     `json:"executables"`
	Skipped     []Skip  `json:"skipped"`
	FanIn       []FanIn `json:"fanIn"`
}

// Analysis는 Analyze의 결과다.
type Analysis struct {
	Draft  *Draft
	YAML   []byte
	Report Report
}

// Analyze는 root의 go.mod와 Go 패키지에서 초안을 만든다. 파일을 쓰지 않는다.
func Analyze(root string) (*Analysis, error) {
	module, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	pkgs, skips, err := collect(root, module)
	if err != nil {
		return nil, fmt.Errorf("코드를 읽을 수 없다: %w", err)
	}
	d, rep := build(module, pkgs, skips)
	return &Analysis{Draft: d, Report: rep}, nil
}

func systemID(module string) string {
	els := strings.Split(module, "/")
	last := els[len(els)-1]
	if version.MatchString(last) && len(els) > 1 {
		last = els[len(els)-2]
	}
	if id := snake(last); snakeName.MatchString(id) {
		return id
	}
	return "system"
}

func representable(dir string) bool {
	if dir == "." {
		return false
	}
	for _, el := range strings.Split(dir, "/") {
		if !goIdent.MatchString(el) {
			return false
		}
	}
	return true
}

func build(module string, pkgs []*pkg, skips []Skip) (*Draft, Report) {
	d := &Draft{GoModule: module, System: systemID(module)}
	var libDirs, exeDirs []string
	byDir := map[string]*pkg{}
	for _, p := range pkgs {
		byDir[p.dir] = p
		switch {
		case p.main:
			exeDirs = append(exeDirs, p.dir)
		case p.dir == ".":
			skips = append(skips, Skip{".", "루트 패키지는 설계 형식으로 표현할 수 없다"})
		case !representable(p.dir):
			skips = append(skips, Skip{p.dir, "Go 패키지 경로로 쓸 수 없는 디렉토리 이름"})
		default:
			libDirs = append(libDirs, p.dir)
		}
	}
	libIDs := assignIDs(libDirs, d.System)
	exeIDs := assignIDs(exeDirs, d.System)

	tier := map[string]int{}
	var level func(dir string) int
	level = func(dir string) int {
		if t, ok := tier[dir]; ok {
			return t
		}
		tier[dir] = 0
		t := 0
		for _, dep := range byDir[dir].imports {
			if _, ok := libIDs[dep]; ok {
				if l := level(dep) + 1; l > t {
					t = l
				}
			}
		}
		tier[dir] = t
		return t
	}
	internalDeps := func(p *pkg) []string {
		var out []string
		for _, dep := range p.imports {
			if id, ok := libIDs[dep]; ok {
				out = append(out, id)
			}
		}
		sort.Strings(out)
		return out
	}
	fan := map[string]int{}
	for _, dir := range libDirs {
		p := byDir[dir]
		m := Mod{ID: libIDs[dir], Summary: p.doc, Tier: level(dir), DependsOn: internalDeps(p)}
		if m.ID != dir {
			m.Path = dir
		}
		for _, dep := range m.DependsOn {
			fan[dep]++
		}
		if m.Tier+1 > d.Tiers {
			d.Tiers = m.Tier + 1
		}
		d.Mods = append(d.Mods, m)
	}
	for _, dir := range exeDirs {
		id := exeIDs[dir]
		if !snakeName.MatchString(id) {
			skips = append(skips, Skip{dir, "실행 단위 ID로 쓸 수 없는 디렉토리 이름"})
			continue
		}
		p := byDir[dir]
		e := Exe{ID: id, Dir: dir, Summary: p.doc, Composes: internalDeps(p)}
		for _, dep := range e.Composes {
			fan[dep]++
		}
		d.Exes = append(d.Exes, e)
	}
	sort.Slice(d.Mods, func(i, j int) bool {
		if d.Mods[i].Tier != d.Mods[j].Tier {
			return d.Mods[i].Tier < d.Mods[j].Tier
		}
		return d.Mods[i].ID < d.Mods[j].ID
	})
	sort.Slice(d.Exes, func(i, j int) bool { return d.Exes[i].ID < d.Exes[j].ID })

	rep := Report{Packages: len(pkgs), Modules: len(d.Mods), Executables: len(d.Exes)}
	sort.Slice(skips, func(i, j int) bool { return skips[i].Path < skips[j].Path })
	rep.Skipped = skips
	for id, n := range fan {
		rep.FanIn = append(rep.FanIn, FanIn{id, n})
	}
	sort.Slice(rep.FanIn, func(i, j int) bool {
		if rep.FanIn[i].Count != rep.FanIn[j].Count {
			return rep.FanIn[i].Count > rep.FanIn[j].Count
		}
		return rep.FanIn[i].ID < rep.FanIn[j].ID
	})
	if len(rep.FanIn) > 5 {
		rep.FanIn = rep.FanIn[:5]
	}
	return d, rep
}

// assignIDs는 디렉토리마다 ID를 정한다. 마지막 경로 요소부터 쓰고, 겹치면 겹친 것끼리
// 경로 요소를 하나씩 앞으로 더 붙인다. 경로 전체로도 겹치면 디렉토리 순서대로 _2, _3을 붙인다.
// 루트(".")는 rootID다.
func assignIDs(dirs []string, rootID string) map[string]string {
	sort.Strings(dirs)
	parts := func(dir string) []string { return strings.Split(dir, "/") }
	cand := func(dir string, n int) string {
		if dir == "." {
			return rootID
		}
		els := parts(dir)
		if n > len(els) {
			n = len(els)
		}
		return snake(strings.Join(els[len(els)-n:], "_"))
	}
	level := map[string]int{}
	for _, d := range dirs {
		level[d] = 1
	}
	for {
		byID := map[string][]string{}
		for _, d := range dirs {
			c := cand(d, level[d])
			byID[c] = append(byID[c], d)
		}
		changed := false
		for _, ds := range byID {
			if len(ds) < 2 {
				continue
			}
			for _, d := range ds {
				if d != "." && level[d] < len(parts(d)) {
					level[d]++
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	ids := map[string]string{}
	used := map[string]int{}
	for _, d := range dirs {
		id := cand(d, level[d])
		used[id]++
		if used[id] > 1 {
			id = fmt.Sprintf("%s_%d", id, used[id])
		}
		ids[d] = id
	}
	return ids
}

// snake는 이름을 소문자 snake_case로 바꾼다. 소문자나 숫자 뒤의 대문자 앞, 대문자 연속 뒤에
// 소문자가 오면 그 연속의 마지막 대문자 앞에 _를 넣는다. -와 .은 _가 된다.
func snake(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsUpper(r) && i > 0 {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				b.WriteByte('_')
			}
		}
		if r == '-' || r == '.' {
			r = '_'
		}
		b.WriteRune(unicode.ToLower(r))
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return strings.Trim(out, "_")
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/infer/`
Expected: PASS

- [ ] **Step 5: 커밋**

```bash
git add internal/infer
git commit -m "feat: infer: 모듈·실행 단위 ID, 층 번호 tier, 분석 메모"
```

---

### Task 4: infer YAML 출력

**Files:**
- Create: `internal/infer/render.go`, `internal/infer/render_test.go`
- Modify: `internal/infer/draft.go`(`Analyze`가 `YAML: Render(d)`를 채운다)

**Interfaces:**
- Consumes: `Draft`
- Produces: `func Render(d *Draft) []byte`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/infer/render_test.go`:

```go
package infer

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

const layersYAML = `# daemok init이 만든 초안이다. 코드의 패키지와 import에서 추정했다.
# tier 이름, 그룹, summary는 사람이 정한다.
version: 0.4
go_module: example.com/layers

system:
  id: layers

executables:
  app: { summary: "app은 예제 실행 파일이다.", composes: [api] }

tiers: [tier0, tier1, tier2]

modules:
  internal_util: { tier: tier0, path: internal/util }
  util: { tier: tier0 }
  cert_gen: { tier: tier1, path: certGen, depends_on: [internal_util] }
  core: { tier: tier1, summary: "Package core는 도메인 타입이다.", depends_on: [util] }
  api: { tier: tier2, depends_on: [core, util] }
`

func TestRenderLayers(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "layers"))
	if err != nil {
		t.Fatal(err)
	}
	if string(a.YAML) != layersYAML {
		t.Fatalf("got\n%s", a.YAML)
	}
	b, _ := Analyze(filepath.Join("testdata", "layers"))
	if !bytes.Equal(a.YAML, b.YAML) {
		t.Fatal("같은 코드에서 다른 초안")
	}
}

func TestRenderRoundTrip(t *testing.T) {
	d := &Draft{
		GoModule: "example.com/x", System: "x", Tiers: 2,
		Exes: []Exe{{ID: "x", Dir: ".", Summary: "줄\n바꿈과\t탭"}},
		Mods: []Mod{
			{ID: "a", Tier: 0, Summary: `"따옴표"와 \역슬래시, # 샵: 콜론 {중괄호} [대괄호]`},
			{ID: "b", Path: "pkg/b", Tier: 1, DependsOn: []string{"a"}},
		},
	}
	m, diags := model.LoadSources(map[string][]byte{"x.yaml": Render(d)})
	if diags.HasErrors() {
		t.Fatalf("%v\n%s", diags, Render(d))
	}
	if m.GoModule != d.GoModule || m.System.ID != "x" || len(m.Tiers) != 2 || m.Tiers[1].Name != "tier1" {
		t.Fatalf("%+v", m)
	}
	if len(m.Executables) != 1 || m.Executables[0].Summary != "줄 바꿈과 탭" || len(m.Executables[0].Composes) != 0 {
		t.Fatalf("%+v", m.Executables[0])
	}
	a, b := m.Modules[0], m.Modules[1]
	if a.ID != "a" || a.Summary != d.Mods[0].Summary || a.Tier.Name != "tier0" {
		t.Fatalf("%+v", a)
	}
	if b.ID != "b" || b.Path != "pkg/b" || len(b.DependsOn) != 1 || b.DependsOn[0].Name != "a" {
		t.Fatalf("%+v", b)
	}
}

func TestRenderNoModules(t *testing.T) {
	d := &Draft{GoModule: "example.com/x", System: "x", Exes: []Exe{{ID: "x", Dir: "."}}}
	if _, diags := model.LoadSources(map[string][]byte{"x.yaml": Render(d)}); diags.HasErrors() {
		t.Fatalf("%v\n%s", diags, Render(d))
	}
}
```

`m.Modules[i].Path`가 path 키가 없을 때 ID로 채워지는지는 `internal/model`의 디코더가 정한다. `b.Path`만 비교하고 `a.Path`는 비교하지 않는다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/infer/ -run Render`
Expected: FAIL, `undefined: Render`

- [ ] **Step 3: 구현**

`internal/infer/render.go`:

```go
package infer

import (
	"bytes"
	"fmt"
	"strings"
)

// Render는 초안을 설계 파일 텍스트로 쓴다. 노드 하나를 한 줄 플로우 맵으로 쓴다.
func Render(d *Draft) []byte {
	var b bytes.Buffer
	b.WriteString("# daemok init이 만든 초안이다. 코드의 패키지와 import에서 추정했다.\n")
	b.WriteString("# tier 이름, 그룹, summary는 사람이 정한다.\n")
	b.WriteString("version: 0.4\n")
	fmt.Fprintf(&b, "go_module: %s\n", d.GoModule)
	fmt.Fprintf(&b, "\nsystem:\n  id: %s\n", d.System)
	if len(d.Exes) > 0 {
		b.WriteString("\nexecutables:\n")
		for _, e := range d.Exes {
			var kv []string
			if e.Summary != "" {
				kv = append(kv, "summary: "+quote(e.Summary))
			}
			kv = append(kv, "composes: "+list(e.Composes))
			fmt.Fprintf(&b, "  %s: { %s }\n", e.ID, strings.Join(kv, ", "))
		}
	}
	if len(d.Mods) > 0 {
		tiers := make([]string, d.Tiers)
		for i := range tiers {
			tiers[i] = tierName(i)
		}
		fmt.Fprintf(&b, "\ntiers: %s\n\nmodules:\n", list(tiers))
		for _, m := range d.Mods {
			kv := []string{"tier: " + tierName(m.Tier)}
			if m.Path != "" {
				kv = append(kv, "path: "+m.Path)
			}
			if m.Summary != "" {
				kv = append(kv, "summary: "+quote(m.Summary))
			}
			if len(m.DependsOn) > 0 {
				kv = append(kv, "depends_on: "+list(m.DependsOn))
			}
			fmt.Fprintf(&b, "  %s: { %s }\n", m.ID, strings.Join(kv, ", "))
		}
	}
	return b.Bytes()
}

func tierName(i int) string { return fmt.Sprintf("tier%d", i) }

func list(xs []string) string { return "[" + strings.Join(xs, ", ") + "]" }

// quote는 큰따옴표 문자열로 쓴다. 줄바꿈과 탭은 공백 하나로 바꾼다.
func quote(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
```

`draft.go`의 `Analyze` 마지막 줄을 바꾼다:

```go
	return &Analysis{Draft: d, YAML: Render(d), Report: rep}, nil
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/infer/`
Expected: PASS. `TestRenderNoModules`가 tiers 없는 설계를 오류로 보면, 원인 진단을 확인하고 `Render`가 모듈이 없을 때도 `tiers: [tier0]`을 쓰도록 바꾼 뒤 테스트 기대를 그대로 둔다.

- [ ] **Step 5: 커밋**

```bash
git add internal/infer
git commit -m "feat: infer: 초안 YAML 출력과 왕복 검사"
```

---

### Task 5: `workspace.RunSources`와 불변식

**Files:**
- Modify: `internal/workspace/workspace.go:47-55`
- Create: `internal/infer/invariant_test.go`

**Interfaces:**
- Produces: `func workspace.RunSources(files map[string][]byte, designDir, root string) workspace.Result`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/infer/invariant_test.go`:

```go
package infer_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/infer"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// draftErrors는 초안을 코드 루트에 대조한 오류 중 파싱 실패가 아닌 것이다.
func draftErrors(t *testing.T, root string) []string {
	t.Helper()
	a, err := infer.Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	design := filepath.Join(root, "design")
	r := workspace.RunSources(map[string][]byte{filepath.Join(design, a.Draft.System+".yaml"): a.YAML}, design, root)
	var out []string
	for _, d := range r.Diags {
		if d.Level == model.Error && !strings.Contains(d.Msg, "Go 파일을 읽을 수 없다") {
			out = append(out, d.String())
		}
	}
	if r.Fatal {
		out = append(out, "fatal")
	}
	return out
}

func TestDraftPassesCheck(t *testing.T) {
	for _, name := range []string{"layers", "rootmain"} {
		if errs := draftErrors(t, filepath.Join("testdata", name)); len(errs) > 0 {
			t.Errorf("%s: %v", name, errs)
		}
	}
}

// codescan fixture에서 설계를 빼고 go.mod를 더한 복사본에도 같은 불변식이 선다.
func TestDraftPassesCheckOnCodescanFixture(t *testing.T) {
	src := filepath.Join("..", "codescan", "testdata", "scan")
	dst := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			if d.Name() == "design" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dst, "go.mod"), []byte("module example.com/scan\n"), 0o644)
	if errs := draftErrors(t, dst); len(errs) > 0 {
		t.Fatal(errs)
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/infer/ -run DraftPasses`
Expected: FAIL, `undefined: workspace.RunSources`

- [ ] **Step 3: 구현**

`internal/workspace/workspace.go`의 `Run`을 둘로 나눈다. 기존 본문은 `run`으로 옮기고 첫 두 줄만 바뀐다.

```go
// Run은 designDir의 설계를 읽고 검사하고 생성해서 root의 파일과 대조한다.
func Run(designDir, root string) Result {
	m, diags := model.Load(designDir)
	return run(m, diags, designDir, root)
}

// RunSources는 디스크가 아닌 설계 파일 내용으로 Run과 같은 일을 한다. 키는 진단 위치에 쓰는 파일 이름이다.
func RunSources(files map[string][]byte, designDir, root string) Result {
	m, diags := model.LoadSources(files)
	return run(m, diags, designDir, root)
}

func run(m *model.Model, diags model.Diagnostics, designDir, root string) Result {
	if diags.HasErrors() {
		// 기존 Run의 나머지 본문 그대로
	}
	// ...
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/infer/ ./internal/workspace/ ./internal/serve/ ./cmd/...`
Expected: PASS. 불변식이 깨지면 진단 메시지를 보고 infer 규칙을 고친다. 테스트를 완화하지 않는다.

- [ ] **Step 5: 커밋**

```bash
git add internal/workspace/workspace.go internal/infer/invariant_test.go
git commit -m "feat: workspace.RunSources와 초안의 check 오류 0 불변식"
```

---

### Task 6: `infer.Write`와 `daemok init`

**Files:**
- Create: `internal/infer/write.go`, `internal/infer/write_test.go`, `cmd/daemok/init.go`, `cmd/daemok/init_test.go`
- Modify: `cmd/daemok/main.go`

**Interfaces:**
- Consumes: `infer.Analyze`, `infer.Analysis`, `infer.Report`
- Produces:
  - `var infer.ErrDesignExists`
  - `func infer.Write(designDir string, a *infer.Analysis) (string, error)`
  - `func infer.HasDesignFiles(designDir string) bool`
  - `func runInit(args []string, stdout, stderr io.Writer) int`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/infer/write_test.go`:

```go
package infer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	a := &Analysis{Draft: &Draft{System: "x"}, YAML: []byte("version: 0.4\n")}
	dir := filepath.Join(t.TempDir(), "design")
	p, err := Write(dir, a)
	if err != nil || p != filepath.Join(dir, "x.yaml") {
		t.Fatalf("%q %v", p, err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, a.YAML) {
		t.Fatal(string(b))
	}
	if !HasDesignFiles(dir) {
		t.Fatal("HasDesignFiles")
	}
	if _, err := Write(dir, &Analysis{Draft: &Draft{System: "y"}, YAML: []byte("z")}); !errors.Is(err, ErrDesignExists) {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "y.yaml")); err == nil {
		t.Fatal("설계 파일이 있으면 쓰지 않는다")
	}
}

// 모델은 *.yaml만 읽으므로 .yml 파일은 설계 파일로 세지 않는다.
func TestWriteIgnoresYml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.yml"), []byte("keep"), 0o644)
	if _, err := Write(dir, &Analysis{Draft: &Draft{System: "x"}, YAML: []byte("new")}); err != nil {
		t.Fatal(err)
	}
}
```

`cmd/daemok/init_test.go`:

```go
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n"), 0o644)
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", root}, &out, &errb); code != 0 {
		t.Fatalf("%d %s %s", code, out.String(), errb.String())
	}
	if _, err := os.Stat(filepath.Join(root, "design", "demo.yaml")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "패키지 1, 모듈 1, 실행 단위 0") {
		t.Fatal(out.String())
	}
	out.Reset()
	if code := run([]string{"check", "--design", filepath.Join(root, "design"), "--root", root}, &out, &errb); code != 0 {
		t.Fatalf("초안의 check: %d\n%s", code, out.String())
	}
	if code := run([]string{"init", "--root", root}, &out, &errb); code != 1 {
		t.Fatalf("이미 설계 파일이 있으면 1: %d", code)
	}
}

func TestInitNotModule(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", t.TempDir()}, &out, &errb); code != 2 {
		t.Fatalf("%d", code)
	}
	if !strings.Contains(errb.String(), "Go 모듈이 아니다") {
		t.Fatal(errb.String())
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/infer/ ./cmd/daemok/`
Expected: FAIL, `undefined: Write`

- [ ] **Step 3: 구현**

`internal/infer/write.go`:

```go
package infer

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrDesignExists = errors.New("설계 디렉토리에 이미 설계 파일이 있다")

// HasDesignFiles는 designDir에 *.yaml이 있는지다.
func HasDesignFiles(designDir string) bool {
	paths, _ := filepath.Glob(filepath.Join(designDir, "*.yaml"))
	return len(paths) > 0
}

// Write는 designDir에 *.yaml이 없을 때만 초안을 <system>.yaml로 새로 만든다. 만든 파일 경로를 준다.
func Write(designDir string, a *Analysis) (string, error) {
	if HasDesignFiles(designDir) {
		return "", ErrDesignExists
	}
	if err := os.MkdirAll(designDir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(designDir, a.Draft.System+".yaml")
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(a.YAML); err != nil {
		f.Close()
		return "", err
	}
	return p, f.Close()
}
```

`cmd/daemok/init.go`:

```go
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/infer"
)

// runInit은 설계 파일이 없는 Go 모듈의 코드에서 초안을 만들어 설계 디렉토리에 쓴다.
// 이미 설계 파일이 있으면 쓰지 않고 1, Go 모듈이 아니거나 쓰지 못하면 2다.
func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design := fs.String("design", "", "설계 디렉토리 (기본 <root>/design)")
	root := fs.String("root", ".", "코드 루트")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dir := *design
	if dir == "" {
		dir = filepath.Join(*root, "design")
	}
	a, err := infer.Analyze(*root)
	if err != nil {
		fmt.Fprintf(stderr, "분석할 수 없다: %v\n", err)
		return 2
	}
	p, err := infer.Write(dir, a)
	switch {
	case errors.Is(err, infer.ErrDesignExists):
		fmt.Fprintf(stderr, "%v: %s\n", err, dir)
		return 1
	case err != nil:
		fmt.Fprintf(stderr, "쓸 수 없다: %v\n", err)
		return 2
	}
	printReport(stdout, a.Report)
	fmt.Fprintf(stdout, "쓴 파일: %s\n", p)
	return 0
}

func printReport(w io.Writer, r infer.Report) {
	fmt.Fprintf(w, "패키지 %d, 모듈 %d, 실행 단위 %d\n", r.Packages, r.Modules, r.Executables)
	for _, s := range r.Skipped {
		fmt.Fprintf(w, "건너뜀: %s (%s)\n", s.Path, s.Reason)
	}
	if len(r.FanIn) > 0 {
		var xs []string
		for _, f := range r.FanIn {
			xs = append(xs, fmt.Sprintf("%s %d", f.ID, f.Count))
		}
		fmt.Fprintf(w, "많이 쓰이는 패키지: %s\n", strings.Join(xs, ", "))
	}
}
```

`cmd/daemok/main.go`: 사용법에 `daemok init  [--root DIR] [--design DIR]`, serve 줄에 `[--workspace DIR]`를 더하고 `case "init": return runInit(args[1:], stdout, stderr)`를 더한다.

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/infer/ ./cmd/daemok/`
Expected: PASS

- [ ] **Step 5: 커밋**

```bash
git add internal/infer cmd/daemok
git commit -m "feat: daemok init: 코드에서 L2 초안을 design/에 쓴다"
```

---

### Task 7: serve 모드(check, analyze, none)

**Files:**
- Modify: `internal/serve/server.go:25-77`
- Modify: `internal/serve/server_test.go`(테스트 추가만)

**Interfaces:**
- Consumes: `infer.Analyze`, `infer.HasDesignFiles`, `infer.Report`, `workspace.RunSources`
- Produces: 상태 JSON의 `mode`(`"check" | "analyze" | "none"`), `draft`(`{path, yaml, report, command}`), `problem`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/serve/server_test.go`에 추가:

```go
func goModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "api"), 0o755)
	os.WriteFile(filepath.Join(root, "api", "a.go"), []byte("package api\nimport _ \"example.com/demo/core\"\n"), 0o644)
	return root
}

func TestAnalyzeMode(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	st := getState(t, s)
	if st["mode"] != "analyze" || st["fatal"] != false {
		t.Fatalf("%v %v", st["mode"], st["fatal"])
	}
	mods := st["model"].(map[string]any)["mods"].([]any)
	if len(mods) != 2 {
		t.Fatalf("%v", mods)
	}
	d := st["draft"].(map[string]any)
	if !strings.Contains(d["yaml"].(string), "api: { tier: tier1, depends_on: [core] }") {
		t.Fatal(d["yaml"])
	}
	if d["report"].(map[string]any)["modules"].(float64) != 2 || !strings.HasPrefix(d["command"].(string), "daemok init --root ") {
		t.Fatalf("%v", d)
	}
	for _, it := range st["issues"].([]any) {
		if it.(map[string]any)["lv"] == "err" {
			t.Fatalf("초안에 오류: %v", it)
		}
	}
}

func TestAnalyzeModeSeesCodeChange(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	os.MkdirAll(filepath.Join(root, "web"), 0o755)
	os.WriteFile(filepath.Join(root, "web", "w.go"), []byte("package web\n"), 0o644)
	if !s.Poll() {
		t.Fatal("코드 변경을 감지하지 못했다")
	}
	if n := len(getState(t, s)["model"].(map[string]any)["mods"].([]any)); n != 3 {
		t.Fatalf("%d", n)
	}
}

func TestNoneMode(t *testing.T) {
	root := t.TempDir()
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	st := getState(t, s)
	if st["mode"] != "none" || st["fatal"] != true || !strings.Contains(st["problem"].(string), "Go 모듈이 아니다") {
		t.Fatalf("%v", st)
	}
}
```

`TestState`에 `st["mode"] != "check"`이면 실패하는 줄을 더한다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/serve/`
Expected: FAIL, `mode`가 `<nil>`

- [ ] **Step 3: 구현**

`state`에 필드를 더한다:

```go
	Mode    string  `json:"mode"` // check, analyze, none
	Draft   *vDraft `json:"draft,omitempty"`
	Problem string  `json:"problem,omitempty"`
```

```go
// vDraft는 분석 모드의 초안이다.
type vDraft struct {
	Path    string       `json:"path"`
	YAML    string       `json:"yaml"`
	Report  infer.Report `json:"report"`
	Command string       `json:"command"`
}
```

`Refresh`를 바꾼다. 경로는 잠금 아래에서 읽는다(Task 8에서 바뀔 수 있다).

```go
// Refresh는 설계나 코드를 다시 읽고 상태를 새로 만든 뒤 구독자에게 알린다.
// 설계 파일이 있으면 대조 모드, 없고 코드 루트가 Go 모듈이면 분석 모드다.
func (s *Server) Refresh() {
	design, root := s.paths()
	st := state{Design: design, Root: root, Updated: time.Now()}
	switch a, err := infer.Analyze(root); {
	case infer.HasDesignFiles(design):
		st.Mode = "check"
		fill(&st, workspace.Run(design, root), root)
		if files, ok, err := gitbase.Files(design); err == nil && ok {
			base, _ := model.LoadSources(files)
			st.Base, st.HasBase = toView(base), true
		}
	case err == nil:
		st.Mode = "analyze"
		name := filepath.Join(design, a.Draft.System+".yaml")
		fill(&st, workspace.RunSources(map[string][]byte{name: a.YAML}, design, root), root)
		st.Draft = &vDraft{Path: name, YAML: string(a.YAML), Report: a.Report, Command: "daemok init --root " + shellQuote(root)}
	default:
		st.Mode, st.Fatal, st.Problem = "none", true, err.Error()
	}
	s.publish(st)
}

func fill(st *state, r workspace.Result, root string) {
	st.Model, st.Fatal = toView(r.Model), r.Fatal
	st.Issues = toIssues(r.Model, root, r.Diags, r.Code)
	st.Files = toFiles(r.Model, r.Files)
	st.Relations = toRelations(r.Model)
}

// shellQuote는 공백이 있는 경로를 큰따옴표로 감싼다.
func shellQuote(p string) string {
	if strings.ContainsAny(p, " \t") {
		return `"` + p + `"`
	}
	return p
}

func (s *Server) paths() (design, root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.design, s.root
}
```

`switch` 초기문의 `infer.Analyze`는 대조 모드에서도 돌아 낭비다. 아래처럼 대조 모드를 먼저 가르고 그 밖에서만 부른다:

```go
	if infer.HasDesignFiles(design) {
		// check
	} else if a, err := infer.Analyze(root); err == nil {
		// analyze
	} else {
		// none
	}
```

기존 `Refresh`의 버전 증가·직렬화·구독자 알림 부분은 그대로 `publish(st state)`로 옮긴다.

`toView(nil)`이 이미 처리되는지 확인한다(none 모드는 `Model`이 nil). 기존 fatal 경로가 nil 모델을 쓰므로 처리된다.

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/serve/`
Expected: PASS(기존 `TestFirstStateWithoutDesignFiles` 포함)

- [ ] **Step 5: 커밋**

```bash
git add internal/serve
git commit -m "feat: serve 분석 모드: 설계 파일이 없으면 코드에서 만든 초안을 보여 준다"
```

---

### Task 8: 프로젝트 열기, 최근 목록, 작업 폴더, 저장 API

**Files:**
- Create: `internal/serve/projects.go`, `internal/serve/projects_test.go`
- Modify: `internal/serve/server.go`(`Server`에 `opts Options`, `Poll`과 `signature`가 경로를 잠금 아래에서 읽기, `Handler`에 라우트)

**Interfaces:**
- Consumes: `infer.Analyze`, `infer.Write`, `infer.ErrDesignExists`, `infer.HasDesignFiles`, `gosrc.SkipDir`
- Produces:
  - `type Options struct { Workspace string; ConfigDir string; Local bool }`
  - `func (s *Server) SetOptions(o Options)`
  - `func (s *Server) Open(path string) error`
  - `GET /api/projects` → `{"current": string, "recent": [{"path", "exists"}], "workspace": [string], "canOpen": bool}`
  - `POST /api/open` `{"path"}` → 200 `{"root"}` / 400·403 `{"error"}`
  - `POST /api/init` → 200 `{"path"}` / 403·409 `{"error"}`

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/serve/projects_test.go`:

```go
package serve

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func local(t *testing.T, s *Server) string {
	t.Helper()
	cfg := t.TempDir()
	s.SetOptions(Options{ConfigDir: cfg, Local: true})
	return cfg
}

func post(t *testing.T, s *Server, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Host = "127.0.0.1:7412"
	req.Header.Set("Origin", "http://127.0.0.1:7412")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestOpenSwitchesProject(t *testing.T) {
	s, _ := setup(t, design)
	cfg := local(t, s)
	root := goModule(t)
	body, _ := json.Marshal(map[string]string{"path": "  " + root + string(filepath.Separator) + "  "})
	if code, out := post(t, s, "/api/open", string(body)); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	st := getState(t, s)
	if st["mode"] != "analyze" || st["root"] != root {
		t.Fatalf("%v %v", st["mode"], st["root"])
	}
	b, _ := os.ReadFile(filepath.Join(cfg, "recent.json"))
	if !strings.Contains(string(b), filepath.Base(root)) {
		t.Fatal(string(b))
	}
}

func TestOpenRejectsAndKeepsProject(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	before := filepath.Dir(dir)
	file := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(file, nil, 0o644)
	for _, p := range []string{filepath.Join(before, "none"), file, t.TempDir(), ""} {
		body, _ := json.Marshal(map[string]string{"path": p})
		if code, _ := post(t, s, "/api/open", string(body)); code != 400 {
			t.Errorf("%q: %d", p, code)
		}
	}
	if st := getState(t, s); st["root"] != before || st["mode"] != "check" {
		t.Fatalf("현재 프로젝트가 바뀌었다: %v", st["root"])
	}
}

func TestInitWritesDraftAndSwitchesToCheck(t *testing.T) {
	root := goModule(t)
	s := New(filepath.Join(root, "design"), root)
	s.Poll()
	local(t, s)
	code, out := post(t, s, "/api/init", "{}")
	if code != 200 || !strings.HasSuffix(out["path"].(string), "demo.yaml") {
		t.Fatalf("%d %v", code, out)
	}
	if st := getState(t, s); st["mode"] != "check" {
		t.Fatalf("저장 뒤 대조 모드여야 한다: %v", st["mode"])
	}
	if code, _ := post(t, s, "/api/init", "{}"); code != 409 {
		t.Fatalf("%d", code)
	}
}

func TestOpenAndInitNeedLocal(t *testing.T) {
	s, _ := setup(t, design)
	s.SetOptions(Options{Local: false})
	for _, p := range []string{"/api/open", "/api/init"} {
		if code, _ := post(t, s, p, `{"path":"x"}`); code != 403 {
			t.Errorf("%s: %d", p, code)
		}
	}
}

func TestProjectsList(t *testing.T) {
	ws := t.TempDir()
	for _, p := range []string{"a", "g/b", "g/h/c", "g/h/i/d", ".hidden/e", "a/inner"} {
		os.MkdirAll(filepath.Join(ws, p), 0o755)
		os.WriteFile(filepath.Join(ws, p, "go.mod"), []byte("module x\n"), 0o644)
	}
	s, _ := setup(t, design)
	cfg := t.TempDir()
	s.SetOptions(Options{Workspace: ws, ConfigDir: cfg, Local: true})
	os.WriteFile(filepath.Join(cfg, "recent.json"), []byte(`["`+strings.ReplaceAll(filepath.Join(ws, "gone"), `\`, `\\`)+`"]`), 0o644)
	req := httptest.NewRequest("GET", "/api/projects", nil)
	req.Host = "localhost:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out struct {
		Recent []struct {
			Path   string `json:"path"`
			Exists bool   `json:"exists"`
		} `json:"recent"`
		Workspace []string `json:"workspace"`
		CanOpen   bool     `json:"canOpen"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	want := []string{filepath.Join(ws, "a"), filepath.Join(ws, "g", "b"), filepath.Join(ws, "g", "h", "c")}
	if strings.Join(out.Workspace, "|") != strings.Join(want, "|") {
		t.Fatalf("깊이 3까지, 숨김 제외, 모듈 아래로 내려가지 않음: %v", out.Workspace)
	}
	if len(out.Recent) != 1 || out.Recent[0].Exists || !out.CanOpen {
		t.Fatalf("%+v", out)
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/serve/`
Expected: FAIL, `undefined: Options`

- [ ] **Step 3: 구현**

`internal/serve/server.go`의 `Server`에 `opts Options`를 더하고, `signature`가 인자로 경로를 받게 바꾼다. `Poll`:

```go
// Poll은 설계나 코드가 바뀌었으면 Refresh하고 true를 준다. 도중에 프로젝트가 바뀌면 아무것도 하지 않는다.
func (s *Server) Poll() bool {
	design, root := s.paths()
	sig := signature(design, root)
	s.mu.Lock()
	if s.design != design || s.root != root {
		s.mu.Unlock()
		return false
	}
	changed := sig != s.sig
	s.sig = sig
	s.mu.Unlock()
	if changed {
		s.Refresh()
	}
	return changed
}
```

`Handler`:

```go
	mux.HandleFunc("GET /api/projects", s.projects)
	mux.HandleFunc("POST /api/open", s.open)
	mux.HandleFunc("POST /api/init", s.initDraft)
	return s.guard(mux)
```

`guard`는 Task 9에서 만든다. 이 태스크에서는 `func (s *Server) guard(h http.Handler) http.Handler { return h }`를 `guard.go`에 두고 Task 9에서 채운다.

`internal/serve/projects.go`:

```go
package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/infer"
)

// Options는 프로젝트 열기에 쓰는 설정이다.
type Options struct {
	Workspace string // 프로젝트를 찾을 작업 폴더. 비면 목록이 없다
	ConfigDir string // recent.json을 둘 디렉토리. 비면 저장하지 않는다
	Local     bool   // 루프백 주소에서 들을 때 참. 참일 때만 열기·저장을 허용하고 Host·Origin을 검사한다
}

const recentMax = 10

func (s *Server) SetOptions(o Options) {
	s.mu.Lock()
	s.opts = o
	s.mu.Unlock()
}

func (s *Server) options() Options {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opts
}

// Open은 path를 현재 프로젝트로 연다. 설계 디렉토리는 path/design이다.
// 디렉토리가 아니거나 설계 파일과 go.mod가 모두 없으면 바꾸지 않고 오류를 준다.
func (s *Server) Open(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return errors.New("경로가 없다")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if st, err := os.Stat(abs); err != nil || !st.IsDir() {
		return fmt.Errorf("디렉토리가 아니다: %s", abs)
	}
	design := filepath.Join(abs, "design")
	if !infer.HasDesignFiles(design) {
		if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
			return fmt.Errorf("설계 파일도 go.mod도 없다: %s", abs)
		}
	}
	s.mu.Lock()
	s.design, s.root, s.sig = design, abs, "\x00"
	s.mu.Unlock()
	s.remember(abs)
	s.Poll()
	return nil
}

func (s *Server) recentPath() string {
	if d := s.options().ConfigDir; d != "" {
		return filepath.Join(d, "recent.json")
	}
	return ""
}

func (s *Server) recent() []string {
	p := s.recentPath()
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var list []string
	if json.Unmarshal(b, &list) != nil {
		return nil
	}
	return list
}

func (s *Server) remember(path string) {
	p := s.recentPath()
	if p == "" {
		return
	}
	list := []string{path}
	for _, x := range s.recent() {
		if x != path && len(list) < recentMax {
			list = append(list, x)
		}
	}
	b, _ := json.Marshal(list)
	if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
		os.WriteFile(p, b, 0o644)
	}
}

// findModules는 dir 아래 깊이 maxDepth까지에서 go.mod가 있는 디렉토리를 찾는다. 모듈 아래로는 내려가지 않는다.
func findModules(dir string, maxDepth int) []string {
	var out []string
	base := filepath.Clean(dir)
	filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if p != base && (gosrc.SkipDir(d.Name()) || strings.HasPrefix(d.Name(), "_")) {
			return filepath.SkipDir
		}
		if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
			out = append(out, p)
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(base, p)
		if rel != "." && len(strings.Split(filepath.ToSlash(rel), "/")) >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	return out
}

type vProject struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	o := s.options()
	_, root := s.paths()
	body := struct {
		Current   string     `json:"current"`
		Recent    []vProject `json:"recent"`
		Workspace []string   `json:"workspace"`
		CanOpen   bool       `json:"canOpen"`
	}{Current: root, Recent: []vProject{}, Workspace: []string{}, CanOpen: o.Local}
	for _, p := range s.recent() {
		st, err := os.Stat(p)
		body.Recent = append(body.Recent, vProject{p, err == nil && st.IsDir()})
	}
	if o.Workspace != "" {
		body.Workspace = append(body.Workspace, findModules(o.Workspace, 3)...)
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) open(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄웠을 때만 열 수 있다")
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "요청 본문을 읽을 수 없다")
		return
	}
	if err := s.Open(req.Path); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	_, root := s.paths()
	writeJSON(w, http.StatusOK, map[string]string{"root": root})
}

func (s *Server) initDraft(w http.ResponseWriter, r *http.Request) {
	if !s.options().Local {
		writeErr(w, http.StatusForbidden, "루프백 주소로 띄웠을 때만 저장할 수 있다")
		return
	}
	design, root := s.paths()
	if infer.HasDesignFiles(design) {
		writeErr(w, http.StatusConflict, "design/에 이미 설계 파일이 있다")
		return
	}
	a, err := infer.Analyze(root)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	p, err := infer.Write(design, a)
	switch {
	case errors.Is(err, infer.ErrDesignExists):
		writeErr(w, http.StatusConflict, "design/에 이미 설계 파일이 있다")
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.Poll()
	writeJSON(w, http.StatusOK, map[string]string{"path": p})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
```

`filepath.Abs`는 끝의 구분자를 지운다(`Clean`). `TestOpenSwitchesProject`의 끝 구분자와 공백 입력이 이 동작에 기대므로 테스트로 확인된다.

- [ ] **Step 4: 통과 확인**

Run: `go test ./internal/serve/`
Expected: PASS

- [ ] **Step 5: 커밋**

```bash
git add internal/serve
git commit -m "feat: serve 프로젝트 열기, 최근 목록, 작업 폴더, 초안 저장 API"
```

---

### Task 9: Host·Origin 검사와 `serve` 명령 옵션

**Files:**
- Modify: `internal/serve/guard.go`
- Create: `internal/serve/guard_test.go`
- Modify: `cmd/daemok/serve.go`

**Interfaces:**
- Consumes: `Options.Local`
- Produces: `func (s *Server) guard(h http.Handler) http.Handler`, `func isLoopback(a net.Addr) bool`(cmd)

- [ ] **Step 1: 실패하는 테스트 작성**

`internal/serve/guard_test.go`:

```go
package serve

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuard(t *testing.T) {
	s, _ := setup(t, design)
	s.SetOptions(Options{Local: true})
	cases := []struct {
		name, method, path, host, origin, ctype string
		want                                     int
	}{
		{"다른 Host의 GET", "GET", "/api/state", "evil.example:7412", "", "", 403},
		{"localhost GET", "GET", "/api/state", "localhost:7412", "", "", 200},
		{"IPv6 루프백 GET", "GET", "/api/state", "[::1]:7412", "", "", 200},
		{"Origin 없는 POST", "POST", "/api/init", "127.0.0.1:7412", "", "application/json", 403},
		{"다른 Origin POST", "POST", "/api/init", "127.0.0.1:7412", "http://evil.example", "application/json", 403},
		{"text/plain POST", "POST", "/api/init", "127.0.0.1:7412", "http://127.0.0.1:7412", "text/plain", 415},
		{"같은 Origin POST", "POST", "/api/init", "127.0.0.1:7412", "http://127.0.0.1:7412", "application/json", 409},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader("{}"))
		req.Host = c.host
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		if c.ctype != "" {
			req.Header.Set("Content-Type", c.ctype)
		}
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: %d, want %d", c.name, rec.Code, c.want)
		}
	}
}

func TestGuardOffWhenNotLocal(t *testing.T) {
	s, _ := setup(t, design)
	req := httptest.NewRequest("GET", "/api/state", nil)
	req.Host = "192.168.0.5:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d", rec.Code)
	}
}
```

`같은 Origin POST`는 `setup`의 프로젝트에 설계 파일이 있으므로 검사를 통과한 뒤 `init`이 409를 준다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/serve/ -run Guard`
Expected: FAIL, 다른 Host의 GET이 200

- [ ] **Step 3: 구현**

`internal/serve/guard.go`:

```go
package serve

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// guard는 루프백에서 들을 때 다른 사이트가 브라우저를 통해 보내는 요청을 막는다.
// Host가 루프백이 아니면 DNS rebinding으로 보고, POST의 Origin이 Host와 다르면 CSRF로 본다.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.options().Local {
			if !loopbackHost(r.Host) {
				http.Error(w, "허용하지 않는 Host", http.StatusForbidden)
				return
			}
			if r.Method == http.MethodPost {
				u, err := url.Parse(r.Header.Get("Origin"))
				if err != nil || u.Host == "" || u.Host != r.Host {
					http.Error(w, "허용하지 않는 Origin", http.StatusForbidden)
					return
				}
				if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
					http.Error(w, "JSON 본문만 받는다", http.StatusUnsupportedMediaType)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func loopbackHost(hostport string) bool {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		h = hostport
	}
	h = strings.Trim(h, "[]")
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
```

`cmd/daemok/serve.go`: 플래그와 옵션을 더한다.

```go
	workspaceDir := fs.String("workspace", "", "프로젝트를 찾을 작업 폴더")
	// ... Listen 뒤
	local := isLoopback(ln.Addr())
	cfg := ""
	if d, err := os.UserConfigDir(); err == nil {
		cfg = filepath.Join(d, "daemok")
	}
	s.SetOptions(serve.Options{Workspace: *workspaceDir, ConfigDir: cfg, Local: local})
	if !local {
		fmt.Fprintln(stderr, "루프백이 아닌 주소라 프로젝트 열기와 초안 저장을 끈다")
	}
```

```go
func isLoopback(a net.Addr) bool {
	t, ok := a.(*net.TCPAddr)
	return ok && t.IP.IsLoopback()
}
```

- [ ] **Step 4: 통과 확인**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: 커밋**

```bash
git add internal/serve cmd/daemok/serve.go
git commit -m "feat: serve Host·Origin 검사와 --workspace"
```

---

### Task 10: 화면

**Files:**
- Modify: `internal/serve/web/index.html`

**Interfaces:**
- Consumes: 상태 JSON의 `mode`, `draft`, `problem`, `root`. `GET /api/projects`, `POST /api/open`, `POST /api/init`.

이 파일에는 JS 테스트 틀이 없다. 서버 테스트 `TestIndexServed`가 계속 통과하는지와 Step 4의 headless 캡처로 확인한다.

- [ ] **Step 1: 머리와 탭**

`index.html:217`의 brand와 탭을 바꾼다:

```html
    <div class="brand"><b>daemok</b><button type="button" class="projbtn" id="projbtn" aria-label="프로젝트 열기"></button><span id="modebadge" class="modebadge" hidden>분석 모드 · 설계 파일 없음</span><span id="status">불러오는 중</span></div>
    <nav class="tabs" aria-label="보기">
      <button type="button" data-view="map">지도</button>
      <button type="button" data-view="module">모듈</button>
      <button type="button" data-view="flow">흐름</button>
      <button type="button" data-view="reqs">요구사항</button>
      <button type="button" data-view="draft" hidden>초안</button>
    </nav>
```

CSS를 `/* 머리 */` 절 끝에 더한다:

```css
.projbtn { border: 0; background: none; padding: 0; cursor: pointer; font: 600 14px var(--mono); color: var(--ink); text-decoration: underline; text-decoration-color: var(--rule-2); text-underline-offset: 3px; }
.projbtn:hover { color: var(--accent); }
.modebadge { color: var(--warn) !important; font-size: 13px; }
.toast { position: fixed; right: 24px; bottom: 24px; z-index: 20; background: var(--bg-3); border: 1px solid var(--rule-2); padding: 10px 14px; border-radius: 6px; font-size: 14px; }
pre.yaml { background: var(--bg-2); border: 1px solid var(--rule); padding: 14px 16px; overflow-x: auto; font: 13px/1.6 var(--mono); }
.draftbar { display: flex; flex-wrap: wrap; align-items: center; gap: 10px 14px; margin: 18px 0 8px; }
.draftbar code { font: 13px var(--mono); color: var(--ink-2); overflow-wrap: anywhere; }
.draftbar button, #openform button { border: 1px solid var(--rule-2); background: var(--bg-2); padding: 4px 12px; border-radius: 4px; cursor: pointer; }
.draftbar button.primary { border-color: var(--accent); color: var(--accent); }
#openform { display: flex; gap: 8px; }
#openform input { flex: 1; min-width: 0; font: 14px var(--mono); color: var(--ink); background: var(--bg-2); border: 1px solid var(--rule-2); padding: 5px 8px; border-radius: 4px; }
#openerr { color: var(--err); margin: 6px 0 0; min-height: 1.2em; font-size: 13px; }
```

- [ ] **Step 2: 상태 그리기**

`renderHeader`를 바꾼다:

```js
function renderHeader() {
  const t = new Date(S.updated);
  const name = (S.root || '').replace(/[\\/]+$/, '').split(/[\\/]/).pop() || '프로젝트 열기';
  $('#projbtn').textContent = name;
  $('#projbtn').title = S.root || '';
  $('#modebadge').hidden = S.mode !== 'analyze';
  const status = S.mode === 'none' ? [S.problem] : [t.toLocaleTimeString(), S.fatal ? '설계 파일을 읽지 못했다' : '', S.mode === 'check' && !meta.hasBase ? '기준 없음' : ''];
  $('#status').textContent = status.filter(Boolean).join('  ');
  $('#status').className = S.fatal ? 'bad' : '';
  const shown = { flow: S.mode === 'check', reqs: S.mode === 'check', draft: S.mode === 'analyze' };
  document.querySelectorAll('nav.tabs button').forEach(b => {
    b.hidden = S.mode === 'none' || (b.dataset.view in shown && !shown[b.dataset.view]);
    b.setAttribute('aria-current', String(b.dataset.view === view));
  });
  const c = { err: 0, warn: 0, info: 0 };
  S.issues.forEach(i => c[i.lv]++);
  $('#issuesbtn').innerHTML = [['e', 'err'], ['w', 'warn'], ['n', 'info']].map(([cl, lv]) => `<span class="${cl} ${c[lv] ? 'on' : ''}">${SEV[lv]} ${c[lv]}</span>`).join('');
}
```

`render`를 바꾼다:

```js
const VIEWS = { check: ['map', 'module', 'flow', 'reqs'], analyze: ['map', 'module', 'draft'], none: [] };
function render() {
  if (!(VIEWS[S.mode] || VIEWS.check).includes(view)) view = 'map';
  renderHeader();
  if (S.mode === 'none') { setSide(''); $('#main').innerHTML = `<p class="muted">열 프로젝트를 고른다. 설계 파일(design/*.yaml)이나 go.mod가 있는 폴더를 연다.</p>`; return; }
  ({ map: renderMap, module: renderModule, flow: renderFlow, reqs: renderReqs, draft: renderDraft })[view]();
}
```

초안 탭:

```js
/* ---------- 초안 ---------- */
function renderDraft() {
  setSide('');
  const d = S.draft, r = d.report;
  const skipped = (r.skipped || []).map(x => `<li><span class="id">${esc(x.path)}</span> <span class="muted">${esc(x.reason)}</span></li>`).join('') || '<li class="muted">없음</li>';
  const fan = (r.fanIn || []).map(x => `${unitLink(x.id)} ${x.count}`).join(', ') || '<span class="muted">없음</span>';
  $('#main').innerHTML = `<h1 class="plain">초안</h1>
<p class="sub">설계 파일이 없어 코드의 패키지와 import에서 추정했다. tier 이름과 묶음은 사람이 정한다.</p>
<dl class="facts"><dt>패키지</dt><dd>${r.packages}</dd><dt>모듈</dt><dd>${r.modules}</dd><dt>실행 단위</dt><dd>${r.executables}</dd><dt>많이 쓰이는 패키지</dt><dd>${fan}</dd></dl>
<h3>건너뛴 것</h3><ul>${skipped}</ul>
<div class="draftbar"><button type="button" class="primary" id="savedraft">design/에 저장</button><code>${esc(d.command)}</code><button type="button" id="copycmd">명령 복사</button></div>
<p id="savemsg" class="muted"></p>
<pre class="yaml">${esc(d.yaml)}</pre>`;
}
```

`.facts`가 지금 CSS에 없으면 `dl.facts { display: grid; grid-template-columns: max-content 1fr; gap: 4px 16px; } dl.facts dt { color: var(--ink-3); } dl.facts dd { margin: 0; }`를 더한다(`factRows`가 이미 `facts` 클래스를 쓰므로 먼저 `grep -n "\.facts" index.html`로 확인한다).

- [ ] **Step 3: 프로젝트 열기와 저장**

```js
/* ---------- 프로젝트 ---------- */
let autoOpened = false;
async function post(url, body) {
  try {
    const r = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) });
    const j = await r.json().catch(() => ({}));
    return r.ok ? { ok: true, ...j } : { ok: false, error: j.error || `요청이 거부되었다 (${r.status})` };
  } catch (e) { return { ok: false, error: '서버에 연결할 수 없다' }; }
}
function toast(msg) {
  const t = document.createElement('div');
  t.className = 'toast'; t.setAttribute('role', 'status'); t.textContent = msg;
  document.body.appendChild(t);
  setTimeout(() => t.remove(), 4000);
}
async function openProjects() {
  drawerOpen = '#projects';
  let p = { recent: [], workspace: [], canOpen: false };
  try { p = await (await fetch('/api/projects', { cache: 'no-store' })).json(); } catch (e) {}
  const item = (path, ok) => `<button type="button" data-openpath="${esc(path)}" ${ok ? '' : 'disabled'}><span class="id">${esc(path)}</span></button>`;
  const form = p.canOpen
    ? `<form id="openform"><input id="openpath" type="text" placeholder="E:\\dev\\projects\\tolchain" autocomplete="off" spellcheck="false" aria-label="프로젝트 경로"><button type="submit">열기</button></form><p id="openerr"></p>`
    : `<p class="muted">루프백이 아닌 주소로 띄워서 프로젝트를 열 수 없다</p>`;
  $('#drawer').innerHTML = `<div class="scrim" data-close="1"></div><aside class="drawer" role="dialog" aria-label="프로젝트 열기"><div class="head"><h2>프로젝트 열기</h2><button type="button" class="close" data-close="1">닫기</button></div>
${form}
<div><h3>최근</h3><div class="issuelist">${p.recent.map(x => item(x.path, x.exists && p.canOpen)).join('') || '<span class="muted">없음</span>'}</div></div>
${p.workspace.length ? `<div><h3>작업 폴더</h3><div class="issuelist">${p.workspace.map(x => item(x, p.canOpen)).join('')}</div></div>` : ''}
</aside>`;
  ($('#openpath') || $('#drawer .close')).focus();
}
async function openProject(path) {
  const res = await post('/api/open', { path });
  if (!res.ok) { const e = $('#openerr'); if (e) e.textContent = res.error; else toast(res.error); return; }
  closeDrawer();
  view = 'map'; current = ''; flowSel = ''; flowCompare = ''; failAt = ''; mapFocus = '';
  history.replaceState(null, '', '#map');
  await load();
}
async function saveDraft() {
  const btn = $('#savedraft'); if (btn) btn.disabled = true;
  const res = await post('/api/init', {});
  if (!res.ok) { $('#savemsg').textContent = res.error; if (btn) btn.disabled = false; return; }
  toast(`${res.path}을 만들었다`);
  await load();
}
```

클릭 처리기 맨 앞(`const tab = ...` 위)에 더한다:

```js
  if (t.closest('#projbtn')) return openProjects();
  const op = t.closest('[data-openpath]'); if (op) return openProject(op.dataset.openpath);
  if (t.closest('#savedraft')) return saveDraft();
  if (t.closest('#copycmd')) { navigator.clipboard.writeText(S.draft.command).then(() => toast('명령을 복사했다'), () => toast('복사할 수 없다')); return; }
```

폼 제출:

```js
document.addEventListener('submit', e => {
  if (e.target.id !== 'openform') return;
  e.preventDefault();
  openProject($('#openpath').value.trim());
});
```

`load()`의 마지막 줄을 바꾼다. 열기 서랍은 다시 그리지 않는다(입력 중인 경로가 지워진다):

```js
  if (drawerOpen === '#issues') openIssues(); else if (drawerOpen && drawerOpen[0] !== '#') openDrawer(drawerOpen);
  if (S.mode === 'none' && !autoOpened) { autoOpened = true; openProjects(); }
```

해시 처리 블록의 허용 목록에 `'draft'`를 더한다.

- [ ] **Step 4: 확인**

Run: `go test ./internal/serve/`
Expected: PASS

그 뒤 headless 캡처로 아래 상태를 확인한다. 캡처 스크립트는 세션 scratchpad의 `shot.mjs`(Edge `--remote-debugging-port`, CDP `Page.captureScreenshot`)를 쓴다. 서버는 `go run ./cmd/daemok serve --root <임시 디렉토리>`로 띄운다.

| 상태 | 확인 |
|---|---|
| Go 모듈이 아닌 루트 | 열기 서랍이 자동으로 열리고 탭이 없다 |
| 경로 입력 실패 | 서랍 안에 이유 문구 |
| Go 모듈 열기 | 분석 모드 배지, 지도에 `tier0…`, 탭은 지도·모듈·초안 |
| 초안 탭 | 메모, YAML, 저장 버튼, 명령 |
| 저장 | 알림, 대조 모드로 바뀌고 배지가 사라짐 |

- [ ] **Step 5: 커밋**

```bash
git add internal/serve/web/index.html
git commit -m "feat: 화면에서 프로젝트 열기, 분석 모드 배지, 초안 탭과 저장"
```

---

### Task 11: 실제 프로젝트 확인과 기록

**Files:**
- Modify: `docs/design/02-slice-tolchain.md`(절 추가)

원래 저장소에는 쓰지 않는다. 확인은 모두 세션 scratchpad의 복사본에서 한다.

- [ ] **Step 1: tolchain 복사본**

```bash
git -C E:/dev/projects/games/A301_GAME/tolchain rev-parse --short HEAD   # 29c1d3a 확인
mkdir -p <scratch>/tolchain && git -C E:/dev/projects/games/A301_GAME/tolchain archive 29c1d3a | tar -x -C <scratch>/tolchain
go run ./cmd/daemok init --root <scratch>/tolchain
go run ./cmd/daemok check --design <scratch>/tolchain/design --root <scratch>/tolchain
```

Expected: `check`의 오류 0(종료 코드 0 또는 경고만 있어 0).

- [ ] **Step 2: 비교**

초안(`<scratch>/tolchain/design/tolchain.yaml`)과 `internal/model/testdata/tolchain/design/tolchain.yaml`을 모듈 단위로 비교해 센다.

| 항목 | 세는 규칙 |
|---|---|
| tier 배치를 바꿔야 하는 모듈 | 손으로 쓴 설계의 tier 순서(foundation < domain < persistence < execution < service)로 바꿨을 때, 초안의 층 순서와 상대 순서가 어긋나는 모듈 |
| 그룹으로 묶어야 하는 패키지 | 손으로 쓴 설계에서 그룹 멤버인 패키지 |
| ID를 바꿔야 하는 모듈 | 같은 경로인데 ID가 다른 모듈 |

- [ ] **Step 3: fiber_api_server와 합성 fixture 화면**

```bash
mkdir -p <scratch>/fiber && git -C E:/dev/projects/web/tolelom/fiber_api_server archive HEAD | tar -x -C <scratch>/fiber
```

합성 fixture: `<scratch>/big/`에 `go.mod`(`module example.com/big`)와 `p00`…`p59` 패키지를 만들고, `pNN`이 `p(NN-1)`과 `p(NN-7)`을 import하게 한다(층이 여러 개, 한 층에 패키지가 여러 개). `cmd/app`이 `p59`를 import한다.

세 복사본을 `daemok serve --root ...`로 띄워 지도와 초안 탭을 headless로 캡처해 읽을 수 있는지 본다. 지도가 넘치면 고치지 않고 위험으로 보고한다(스펙의 위험 표).

- [ ] **Step 4: 슬라이스 문서에 기록**

`docs/design/02-slice-tolchain.md` 끝에 절을 더한다. 기준 커밋, 세는 규칙, 표(항목, 수, 해당 모듈)만 쓴다. 문체는 이 문서의 기존 문체(평서형 "~다")를 따른다.

```markdown
## 코드에서 만든 초안과의 비교

`daemok init`을 `main` `29c1d3a`에 돌려 만든 초안과 이 문서의 설계 파일을 모듈 단위로 비교한다.

| 항목 | 수 | 모듈 |
|---|---|---|
| tier 배치를 바꿔야 하는 모듈 | <측정값> | <목록> |
| 그룹으로 묶어야 하는 패키지 | <측정값> | <목록> |
| ID를 바꿔야 하는 모듈 | <측정값> | <목록> |
```

`<측정값>`과 `<목록>`은 Step 2에서 센 실제 값으로 채운다. 채우지 못하면 이 단계를 끝내지 않는다.

- [ ] **Step 5: 전체 확인과 커밋**

Run: `go vet ./... && go test ./...`
Expected: PASS

```bash
git add docs/design/02-slice-tolchain.md
git commit -m "docs: tolchain 초안과 손으로 쓴 설계 비교"
```
