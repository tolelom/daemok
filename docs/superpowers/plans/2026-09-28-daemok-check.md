# daemok check Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `daemok check`가 설계 디렉토리의 YAML 부분집합을 읽어 모델로 합치고, 모델 문서의 설계 검사 전부를 돌려 `파일:줄:열: 수준: 메시지` 형식으로 출력한다. MVP 스펙의 1~3단계다.

**Architecture:** 세 패키지가 한 방향으로 의존한다. `internal/yaml`은 줄 번호와 열을 보존하는 트리를 만들고, `internal/model`은 그 트리를 노드 타입으로 옮기며 알 수 없는 키와 중복 ID를 잡고, `internal/check`는 모델만 보고 진단을 낸다. `cmd/daemok`은 인자 해석과 출력만 한다. 코드 생성(계획 2)과 코드 대조·서버(계획 3)는 이 세 패키지 위에 얹는다.

**Tech Stack:** Go 1.27 표준 라이브러리만. 외부 모듈 없음.

**Spec:** `docs/superpowers/specs/2026-09-27-daemok-mvp-design.md`, 형식과 검사 규칙은 `docs/design/01-model.md`.

## Global Constraints

- 외부 의존성 없음. `go.mod`에 `require`가 생기면 안 된다.
- 설계 파일 형식은 v0.3. 최상위 `version` 값이 `0.3`이 아니면 오류.
- 형식에 없는 키는 오류. 조용히 무시하지 않는다.
- 진단 출력 형식: `design/tx.yaml:12:5: 오류: 메시지`. 수준은 `오류`·`경고`·`정보`.
- 열은 바이트 기준 1-base(Go 도구 관례). 줄은 1-base.
- 종료 코드: 0 통과, 1 오류 있음(`--strict`면 경고 포함), 2 진행 불가(설계 파일을 읽지 못함 또는 모델 단계 오류).
- 메시지는 한국어 평서문. 마침표 없음.
- 커밋 메시지는 저장소 관례대로 `feat: 한국어 요약` 형식(`docs:`, `test:`도 같은 꼴).
- 모든 Go 코드는 `gofmt` 통과. 테스트는 `go test ./...` 통과.

## Review Focus

1. 빈 설계 디렉토리 또는 `*.yaml`이 없는 디렉토리: 종료 코드 2와 디렉토리 경로가 든 메시지 하나. Task 13에서 검사.
2. 한 줄 안에 `#`가 URL이나 문자열 안에 있는 값(`spec: "docs/a#b.yaml"`, `path: /x#y`): 따옴표 안 `#`은 주석이 아니고, 공백 없이 붙은 `#`도 주석이 아니다. Task 2에서 검사.
3. 여러 줄 `[`의 뒷줄에 주석이나 빈 줄이 섞인 경우: 주석·빈 줄은 건너뛰고 닫힘까지 이어 읽는다. Task 4에서 검사.
4. 같은 ID가 다른 종류에 함께 있는 경우(모듈 `events`와 인터페이스 `events`): 오류가 아니다. 반면 모듈과 멤버가 같은 ID면 `in`이 가리킬 대상이 둘이라 오류. Task 8에서 검사.
5. 의존 순환이 그룹 멤버의 예외를 거쳐 생기는 경우(`reward → asset`, `asset`이 `tx_modules`에 의존): 순환 오류에 경로가 나온다. Task 10에서 검사.

---

## File Structure

| 경로 | 책임 |
|---|---|
| `go.mod` | 모듈 `github.com/tolelom/daemok` |
| `cmd/daemok/main.go` | 인자 해석, 종료 코드 |
| `cmd/daemok/check.go` | `check` 명령: Load → Run → 출력 |
| `cmd/daemok/main_test.go` | 명령 단위 테스트(임시 디렉토리에 설계 파일을 쓰고 실행) |
| `internal/yaml/yaml.go` | `Pos`, `Error`, `Node`, `Entry`, 스칼라 도우미 |
| `internal/yaml/scan.go` | 줄 분리, 들여쓰기, 주석 제거, 탭·`---` 오류 |
| `internal/yaml/block.go` | `Parse`, 블록 맵·목록, `- key:` 인라인 맵 |
| `internal/yaml/scalar.go` | 따옴표 문자열 해석, 지원하지 않는 시작 문자 |
| `internal/yaml/flow.go` | `{}`·`[]` 파싱, 여러 줄 `[` |
| `internal/yaml/yaml_test.go` | 파서 테스트 |
| `internal/model/diag.go` | `Level`, `Diagnostic`, `Diagnostics` |
| `internal/model/model.go` | 노드 타입 |
| `internal/model/decode.go` | `decoder` 도우미(맵 확인, 허용 키, 참조 목록) |
| `internal/model/load.go` | `Load`, 파일 병합, 중복 ID, 이름공간 충돌 |
| `internal/model/decode_layers.go` | R·L0·L1·L2 디코드 |
| `internal/model/decode_l3.go` | L3 디코드 |
| `internal/model/decode_l4.go` | L4 디코드 |
| `internal/model/index.go` | ID 조회, `Owner`, `CanDepend`, `ResolveType` |
| `internal/model/load_test.go` | 디코드·병합 테스트 |
| `internal/model/testdata/tolchain/design/tolchain.yaml` | 프로토타입 tolchain 샘플(v0.3) |
| `internal/model/testdata/tolelog/design/tolelog.yaml` | 프로토타입 tolelog 샘플(v0.3) |
| `internal/check/check.go` | `Run`, `checker`, 충족 요소 색인 |
| `internal/check/refs.go` | 참조 해석 검사 |
| `internal/check/l0.go` | 인터페이스 구현·조립 검사 |
| `internal/check/l2.go` | tier 방향, 순환, 고아 모듈, 그룹 예외 |
| `internal/check/l3.go` | 타입 의존, 에러 이름·HTTP, 계약 `errors` |
| `internal/check/handlers.go` | 키 enum 값의 처리 멤버 수 |
| `internal/check/l4.go` | 함수 계약 검사 |
| `internal/check/req.go` | 요구사항 검사 |
| `internal/check/check_test.go` | 규칙별 통과·위반 테스트, 샘플 기대 진단 |

---

### Task 1: Go 모듈과 CLI 골격

**Files:**
- Create: `go.mod`, `cmd/daemok/main.go`, `cmd/daemok/main_test.go`

**Interfaces:**
- Produces: `func run(args []string, stdout, stderr io.Writer) int`. 뒤 Task는 `check` 명령을 여기 붙인다.

- [ ] **Step 1: 모듈 초기화**

```bash
cd /home/soungmin-kim/Desktop/develop/personal/daemok
go mod init github.com/tolelom/daemok
```

- [ ] **Step 2: 실패하는 테스트**

`cmd/daemok/main_test.go`:

```go
package main

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestNoArgsPrintsUsage(t *testing.T) {
	code, _, errOut := runCLI(t)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut, "daemok check") {
		t.Fatalf("usage missing: %q", errOut)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := runCLI(t, "frobnicate")
	if code != 2 || !strings.Contains(errOut, "frobnicate") {
		t.Fatalf("exit %d, err %q", code, errOut)
	}
}
```

- [ ] **Step 3: 실패 확인**

Run: `go test ./cmd/daemok/`
Expected: FAIL, `undefined: run`

- [ ] **Step 4: 구현**

`cmd/daemok/main.go`:

```go
// daemok은 설계 파일을 읽어 검사하고 코드를 생성하는 CLI다.
package main

import (
	"fmt"
	"io"
	"os"
)

const usage = `사용법:
  daemok check [--design DIR] [--root DIR] [--strict]
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "알 수 없는 명령 %q\n%s", args[0], usage)
	return 2
}
```

`cmd/daemok/check.go`(임시. Task 13에서 채운다):

```go
package main

import "io"

func runCheck(args []string, stdout, stderr io.Writer) int {
	return 2
}
```

- [ ] **Step 5: 통과 확인**

Run: `go test ./cmd/daemok/ && go vet ./...`
Expected: PASS

- [ ] **Step 6: 커밋**

```bash
git add go.mod cmd/
git commit -m "feat: Go 모듈과 daemok CLI 골격"
```

---

### Task 2: YAML 블록 구조

**Files:**
- Create: `internal/yaml/yaml.go`, `internal/yaml/scan.go`, `internal/yaml/block.go`, `internal/yaml/scalar.go`(이 Task에서는 `unquote`와 `unsupportedScalar`의 골격만), `internal/yaml/yaml_test.go`

**Interfaces:**
- Produces:
  - `type Pos struct{ File string; Line, Col int }`, `func (Pos) String() string` → `"file:line:col"`
  - `type Error struct{ Pos Pos; Msg string }`, `func (*Error) Error() string`
  - `type Kind int` (`Scalar`, `Map`, `Seq`)
  - `type Node struct{ Kind Kind; Pos Pos; Value string; Quoted bool; Entries []Entry; Items []*Node }`
  - `type Entry struct{ Key string; Pos Pos; Value *Node }`
  - `func Parse(file string, src []byte) (*Node, error)`. 실패 시 `error`는 항상 `*Error`.
  - `func (n *Node) Get(key string) *Node`, `IsNull() bool`, `Int() (int, bool)`, `Bool() (bool, bool)`

- [ ] **Step 1: 실패하는 테스트**

`internal/yaml/yaml_test.go`:

```go
package yaml

import (
	"strings"
	"testing"
)

func parse(t *testing.T, src string) *Node {
	t.Helper()
	n, err := Parse("d.yaml", []byte(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return n
}

func parseErr(t *testing.T, src string) *Error {
	t.Helper()
	_, err := Parse("d.yaml", []byte(src))
	if err == nil {
		t.Fatal("expected error")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("error type %T", err)
	}
	return e
}

func wantErr(t *testing.T, e *Error, line, col int, substr string) {
	t.Helper()
	if e.Pos.Line != line || e.Pos.Col != col || !strings.Contains(e.Msg, substr) {
		t.Fatalf("got %v, want %d:%d %q", e, line, col, substr)
	}
}

func TestBlockMapScalars(t *testing.T) {
	n := parse(t, "go_module: github.com/x/y\nts_out: web/gen.ts\n")
	if n.Kind != Map || len(n.Entries) != 2 {
		t.Fatalf("kind %v entries %d", n.Kind, len(n.Entries))
	}
	e := n.Entries[1]
	if e.Key != "ts_out" || e.Pos.Line != 2 || e.Pos.Col != 1 {
		t.Fatalf("entry %+v", e)
	}
	if e.Value.Value != "web/gen.ts" || e.Value.Pos.Col != 9 {
		t.Fatalf("value %+v", e.Value)
	}
}

func TestNestedBlockMap(t *testing.T) {
	n := parse(t, "modules:\n  core:\n    tier: domain\n  vm:\n    tier: execution\n")
	mods := n.Get("modules")
	if mods == nil || mods.Kind != Map || len(mods.Entries) != 2 {
		t.Fatalf("modules %+v", mods)
	}
	if got := mods.Get("vm").Get("tier").Value; got != "execution" {
		t.Fatalf("tier %q", got)
	}
	if mods.Get("vm").Pos.Line != 5 || mods.Get("vm").Pos.Col != 5 {
		t.Fatalf("pos %v", mods.Get("vm").Pos)
	}
}

func TestBlockSeq(t *testing.T) {
	n := parse(t, "tiers:\n  - foundation\n  - domain\n")
	s := n.Get("tiers")
	if s.Kind != Seq || len(s.Items) != 2 || s.Items[1].Value != "domain" {
		t.Fatalf("seq %+v", s)
	}
	if s.Items[1].Pos.Line != 3 || s.Items[1].Pos.Col != 5 {
		t.Fatalf("pos %v", s.Items[1].Pos)
	}
}

func TestSeqAtSameIndentAsKey(t *testing.T) {
	n := parse(t, "tiers:\n- a\n- b\nnext: 1\n")
	if len(n.Get("tiers").Items) != 2 || n.Get("next").Value != "1" {
		t.Fatalf("%+v", n)
	}
}

func TestSeqOfInlineMaps(t *testing.T) {
	src := "exceptions:\n  - depends_on: asset\n    reason: 재사용\n  - depends_on: core\n"
	s := parse(t, src).Get("exceptions")
	if len(s.Items) != 2 {
		t.Fatalf("items %d", len(s.Items))
	}
	first := s.Items[0]
	if first.Kind != Map || len(first.Entries) != 2 || first.Get("reason").Value != "재사용" {
		t.Fatalf("first %+v", first)
	}
	if first.Get("depends_on").Pos.Col != 17 {
		t.Fatalf("value col %d", first.Get("depends_on").Pos.Col)
	}
}

func TestNullValues(t *testing.T) {
	n := parse(t, "a:\nb: null\nc: x\n")
	if !n.Get("a").IsNull() || !n.Get("b").IsNull() || n.Get("c").IsNull() {
		t.Fatalf("%+v", n)
	}
}

func TestComments(t *testing.T) {
	n := parse(t, "# 머리\na: 1 # 뒤\nb: /x#y\nc: \"a#b\"\n")
	if n.Get("a").Value != "1" || n.Get("b").Value != "/x#y" || n.Get("c").Value != "a#b" {
		t.Fatalf("%+v", n)
	}
}

func TestEmptyFile(t *testing.T) {
	n := parse(t, "# only comment\n\n")
	if n.Kind != Map || len(n.Entries) != 0 {
		t.Fatalf("%+v", n)
	}
}

func TestErrTab(t *testing.T) {
	wantErr(t, parseErr(t, "a:\n\tb: 1\n"), 2, 1, "탭")
}

func TestErrMultiDoc(t *testing.T) {
	wantErr(t, parseErr(t, "a: 1\n---\nb: 2\n"), 2, 1, "여러 문서")
}

func TestErrDuplicateKey(t *testing.T) {
	e := parseErr(t, "a: 1\nb: 2\na: 3\n")
	wantErr(t, e, 3, 1, "중복 키")
	if !strings.Contains(e.Msg, "d.yaml:1:1") {
		t.Fatalf("first position missing: %q", e.Msg)
	}
}

func TestErrBadIndent(t *testing.T) {
	wantErr(t, parseErr(t, "a:\n  b: 1\n c: 2\n"), 3, 2, "들여쓰기")
}

func TestErrNoColon(t *testing.T) {
	wantErr(t, parseErr(t, "a: 1\njust text\n"), 2, 1, "키: 값")
}

func TestErrSeqInMap(t *testing.T) {
	wantErr(t, parseErr(t, "a: 1\n- b\n"), 2, 1, "목록 항목")
}

func TestErrMergeKey(t *testing.T) {
	wantErr(t, parseErr(t, "<<: base\n"), 1, 1, "병합 키")
}

func TestErrComplexKey(t *testing.T) {
	wantErr(t, parseErr(t, "? a\n"), 1, 1, "복합 키")
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/yaml/`
Expected: FAIL, `undefined: Parse`

- [ ] **Step 3: 타입**

`internal/yaml/yaml.go`:

```go
// Package yaml은 daemok 설계 파일이 쓰는 YAML 부분집합을 파싱한다.
// 모든 노드가 파일·줄·열 위치를 갖는다. 열은 바이트 기준 1-base다.
package yaml

import (
	"fmt"
	"strconv"
)

type Pos struct {
	File string
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col) }

type Error struct {
	Pos Pos
	Msg string
}

func (e *Error) Error() string { return e.Pos.String() + ": " + e.Msg }

type Kind int

const (
	Scalar Kind = iota
	Map
	Seq
)

type Node struct {
	Kind    Kind
	Pos     Pos
	Value   string // Scalar. 따옴표 문자열은 이스케이프를 풀어 담는다.
	Quoted  bool   // Scalar가 "..."였는지
	Entries []Entry
	Items   []*Node
}

type Entry struct {
	Key   string
	Pos   Pos // 키 위치
	Value *Node
}

func (n *Node) Get(key string) *Node {
	if n == nil || n.Kind != Map {
		return nil
	}
	for i := range n.Entries {
		if n.Entries[i].Key == key {
			return n.Entries[i].Value
		}
	}
	return nil
}

func (n *Node) IsNull() bool {
	return n != nil && n.Kind == Scalar && !n.Quoted && (n.Value == "" || n.Value == "null")
}

func (n *Node) Int() (int, bool) {
	if n == nil || n.Kind != Scalar || n.Quoted {
		return 0, false
	}
	v, err := strconv.Atoi(n.Value)
	return v, err == nil
}

func (n *Node) Bool() (bool, bool) {
	if n == nil || n.Kind != Scalar || n.Quoted {
		return false, false
	}
	switch n.Value {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}
```

- [ ] **Step 4: 스캐너**

`internal/yaml/scan.go`:

```go
package yaml

import "strings"

type line struct {
	num    int
	indent int    // 앞 공백 수
	text   string // 들여쓰기와 주석을 뺀 본문. 오른쪽 공백 제거
}

type parser struct {
	file  string
	lines []line
	i     int
}

func (p *parser) errAt(num, col int, msg string) error {
	return &Error{Pos: Pos{p.file, num, col}, Msg: msg}
}

func (p *parser) scan(src []byte) error {
	raw := strings.Split(strings.ReplaceAll(string(src), "\r\n", "\n"), "\n")
	for idx, r := range raw {
		num := idx + 1
		indent := 0
		for indent < len(r) && r[indent] == ' ' {
			indent++
		}
		if indent < len(r) && r[indent] == '\t' {
			return p.errAt(num, indent+1, "탭 들여쓰기는 지원하지 않는다")
		}
		text := strings.TrimRight(stripComment(r[indent:]), " ")
		if text == "" {
			continue
		}
		if text == "---" || text == "..." {
			return p.errAt(num, indent+1, "한 파일에 여러 문서는 지원하지 않는다")
		}
		p.lines = append(p.lines, line{num, indent, text})
	}
	return nil
}

// stripComment는 따옴표 밖에 있고 줄 처음이거나 앞이 공백인 '#'부터 자른다.
func stripComment(s string) string {
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch {
		case inQuote && s[i] == '\\':
			i++
		case s[i] == '"':
			inQuote = !inQuote
		case s[i] == '#' && !inQuote && (i == 0 || s[i-1] == ' '):
			return s[:i]
		}
	}
	return s
}
```

- [ ] **Step 5: 블록 파서**

`internal/yaml/block.go`:

```go
package yaml

import (
	"fmt"
	"strings"
)

// Parse는 src를 파싱한다. 빈 파일은 빈 맵이다. 실패하면 *Error를 돌려준다.
func Parse(file string, src []byte) (*Node, error) {
	p := &parser{file: file}
	if err := p.scan(src); err != nil {
		return nil, err
	}
	if len(p.lines) == 0 {
		return &Node{Kind: Map, Pos: Pos{file, 1, 1}}, nil
	}
	n, err := p.parseBlock(p.lines[0].indent)
	if err != nil {
		return nil, err
	}
	if p.i < len(p.lines) {
		ln := p.lines[p.i]
		return nil, p.errAt(ln.num, ln.indent+1, "예상하지 못한 들여쓰기")
	}
	return n, nil
}

func isSeqItem(text string) bool { return text == "-" || strings.HasPrefix(text, "- ") }

func (p *parser) parseBlock(indent int) (*Node, error) {
	if isSeqItem(p.lines[p.i].text) {
		return p.parseSeq(indent)
	}
	return p.parseMap(indent)
}

func (p *parser) parseMap(indent int) (*Node, error) {
	first := p.lines[p.i]
	n := &Node{Kind: Map, Pos: Pos{p.file, first.num, indent + 1}}
	seen := map[string]Pos{}
	for p.i < len(p.lines) {
		ln := p.lines[p.i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, p.errAt(ln.num, ln.indent+1, "예상하지 못한 들여쓰기")
		}
		if isSeqItem(ln.text) {
			return nil, p.errAt(ln.num, ln.indent+1, "맵 안에 목록 항목이 있다")
		}
		key, keyPos, rest, restCol, err := p.splitKey(ln)
		if err != nil {
			return nil, err
		}
		if prev, dup := seen[key]; dup {
			return nil, &Error{keyPos, fmt.Sprintf("중복 키 %q (처음: %s)", key, prev)}
		}
		seen[key] = keyPos
		p.i++
		var val *Node
		if rest == "" {
			switch {
			case p.i < len(p.lines) && p.lines[p.i].indent > indent:
				val, err = p.parseBlock(p.lines[p.i].indent)
			case p.i < len(p.lines) && p.lines[p.i].indent == indent && isSeqItem(p.lines[p.i].text):
				val, err = p.parseSeq(indent)
			default:
				val = &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, restCol}}
			}
		} else {
			val, err = p.parseInline(ln, rest, restCol)
		}
		if err != nil {
			return nil, err
		}
		n.Entries = append(n.Entries, Entry{key, keyPos, val})
	}
	return n, nil
}

// splitKey는 "key: rest"를 나눈다. rest가 비면 값이 다음 줄에 있거나 null이다.
// restCol은 rest가 시작하는 열이다.
func (p *parser) splitKey(ln line) (key string, keyPos Pos, rest string, restCol int, err error) {
	keyPos = Pos{p.file, ln.num, ln.indent + 1}
	text := ln.text
	switch {
	case strings.HasPrefix(text, "? "):
		return "", keyPos, "", 0, &Error{keyPos, "복합 키는 지원하지 않는다"}
	case strings.HasPrefix(text, "<<"):
		return "", keyPos, "", 0, &Error{keyPos, "병합 키는 지원하지 않는다"}
	}
	end := 0 // ':'의 인덱스
	if strings.HasPrefix(text, "\"") {
		s, n, msg := unquote(text)
		if msg != "" {
			return "", keyPos, "", 0, &Error{Pos{p.file, ln.num, ln.indent + 1 + n}, msg}
		}
		if n >= len(text) || text[n] != ':' {
			return "", keyPos, "", 0, &Error{Pos{p.file, ln.num, ln.indent + 1 + n}, "키 뒤에 콜론이 없다"}
		}
		key, end = s, n
	} else {
		end = colonAfterKey(text)
		if end <= 0 {
			return "", keyPos, "", 0, &Error{keyPos, "키: 값 형식이 아니다"}
		}
		key = text[:end]
	}
	rest = text[end+1:]
	if rest != "" && rest[0] != ' ' {
		return "", keyPos, "", 0, &Error{Pos{p.file, ln.num, ln.indent + 1 + end}, "콜론 뒤에 공백이 필요하다"}
	}
	trimmed := strings.TrimLeft(rest, " ")
	restCol = ln.indent + 1 + len(text) - len(trimmed)
	return key, keyPos, trimmed, restCol, nil
}

// colonAfterKey는 뒤에 공백이 오거나 줄 끝인 첫 ':'의 인덱스를 준다. 없으면 -1.
func colonAfterKey(text string) int {
	for i := 0; i < len(text); i++ {
		if text[i] == ':' && (i+1 == len(text) || text[i+1] == ' ') {
			return i
		}
	}
	return -1
}

func (p *parser) parseSeq(indent int) (*Node, error) {
	first := p.lines[p.i]
	n := &Node{Kind: Seq, Pos: Pos{p.file, first.num, indent + 1}}
	for p.i < len(p.lines) {
		ln := p.lines[p.i]
		if ln.indent < indent {
			break
		}
		if ln.indent > indent {
			return nil, p.errAt(ln.num, ln.indent+1, "예상하지 못한 들여쓰기")
		}
		if !isSeqItem(ln.text) {
			return nil, p.errAt(ln.num, ln.indent+1, "목록 안에 목록 항목이 아닌 줄이 있다")
		}
		body := strings.TrimLeft(ln.text[1:], " ")
		bodyIndent := ln.indent + len(ln.text) - len(body)
		if body == "" {
			p.i++
			if p.i < len(p.lines) && p.lines[p.i].indent > indent {
				item, err := p.parseBlock(p.lines[p.i].indent)
				if err != nil {
					return nil, err
				}
				n.Items = append(n.Items, item)
			} else {
				n.Items = append(n.Items, &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, bodyIndent + 1}})
			}
			continue
		}
		if isSeqItem(body) {
			return nil, p.errAt(ln.num, bodyIndent+1, "한 줄에 목록 항목을 겹쳐 쓸 수 없다")
		}
		if looksLikeKey(body) {
			// "- key: v"는 줄을 key부터 시작하는 것으로 바꿔 맵으로 읽는다.
			// 이어지는 줄은 같은 열에 맞춰야 한다.
			p.lines[p.i] = line{ln.num, bodyIndent, body}
			item, err := p.parseMap(bodyIndent)
			if err != nil {
				return nil, err
			}
			n.Items = append(n.Items, item)
			continue
		}
		p.i++
		item, err := p.parseInline(ln, body, bodyIndent+1)
		if err != nil {
			return nil, err
		}
		n.Items = append(n.Items, item)
	}
	return n, nil
}

func looksLikeKey(body string) bool {
	switch body[0] {
	case '{', '[', '"':
		return false
	}
	return colonAfterKey(body) > 0
}

// parseInline은 "key: " 뒤나 "- " 뒤의 값을 읽는다. 플로우 컬렉션, 따옴표 문자열, 평문 스칼라.
func (p *parser) parseInline(ln line, rest string, col int) (*Node, error) {
	switch rest[0] {
	case '{', '[':
		return p.parseFlow(ln, rest, col)
	case '"':
		s, n, msg := unquote(rest)
		if msg != "" {
			return nil, p.errAt(ln.num, col+n, msg)
		}
		if strings.TrimSpace(rest[n:]) != "" {
			return nil, p.errAt(ln.num, col+n, "닫는 따옴표 뒤에 문자가 있다")
		}
		return &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, col}, Value: s, Quoted: true}, nil
	}
	if msg := unsupportedScalar(rest); msg != "" {
		return nil, p.errAt(ln.num, col, msg)
	}
	return &Node{Kind: Scalar, Pos: Pos{p.file, ln.num, col}, Value: rest}, nil
}
```

- [ ] **Step 6: 스칼라 골격과 플로우 임시 구현**

`internal/yaml/scalar.go`(Task 3에서 완성):

```go
package yaml

// unquote는 s[0] == '"'인 문자열을 읽어 값, 소비한 바이트 수, 오류 메시지를 준다.
func unquote(s string) (val string, n int, msg string) {
	for i := 1; i < len(s); i++ {
		if s[i] == '"' {
			return s[1:i], i + 1, ""
		}
	}
	return "", len(s), "닫히지 않은 따옴표"
}

// unsupportedScalar는 평문 스칼라가 지원하지 않는 문법으로 시작하면 메시지를 준다.
func unsupportedScalar(s string) string { return "" }
```

`internal/yaml/flow.go`(Task 4에서 완성):

```go
package yaml

func (p *parser) parseFlow(ln line, rest string, col int) (*Node, error) {
	return nil, p.errAt(ln.num, col, "플로우 컬렉션은 아직 지원하지 않는다")
}
```

- [ ] **Step 7: 통과 확인**

Run: `gofmt -l ./internal/yaml/ && go test ./internal/yaml/`
Expected: gofmt 출력 없음, PASS

- [ ] **Step 8: 커밋**

```bash
git add internal/yaml/
git commit -m "feat: YAML 부분집합 파서: 블록 맵·목록과 위치 보존"
```

---

### Task 3: YAML 스칼라

**Files:**
- Modify: `internal/yaml/scalar.go`
- Test: `internal/yaml/yaml_test.go`

**Interfaces:**
- Produces: `unquote`가 `\"`, `\\`, `\n`, `\t`를 풀고 그 밖의 이스케이프는 오류. `unsupportedScalar`가 `'`, `&`, `*`, `!`, `|`, `>`, `%`, `@`, `` ` ``로 시작하는 값을 거부.

- [ ] **Step 1: 실패하는 테스트**

`yaml_test.go`에 추가:

```go
func TestQuotedScalar(t *testing.T) {
	n := parse(t, "a: \"x: y, z\"\nb: \"줄\\n탭\\t\\\"따옴표\\\" \\\\\"\n")
	if n.Get("a").Value != "x: y, z" || !n.Get("a").Quoted {
		t.Fatalf("a %+v", n.Get("a"))
	}
	if n.Get("b").Value != "줄\n탭\t\"따옴표\" \\" {
		t.Fatalf("b %q", n.Get("b").Value)
	}
}

func TestQuotedNumberStaysString(t *testing.T) {
	n := parse(t, "a: \"12\"\nb: 12\nc: 2s\n")
	if _, ok := n.Get("a").Int(); ok {
		t.Fatal("quoted number must not be int")
	}
	if v, ok := n.Get("b").Int(); !ok || v != 12 {
		t.Fatalf("b %v %v", v, ok)
	}
	if _, ok := n.Get("c").Int(); ok || n.Get("c").Value != "2s" {
		t.Fatalf("c %+v", n.Get("c"))
	}
}

func TestBoolScalar(t *testing.T) {
	n := parse(t, "a: true\nb: false\nc: yes\n")
	if v, ok := n.Get("a").Bool(); !ok || !v {
		t.Fatal("a")
	}
	if v, ok := n.Get("b").Bool(); !ok || v {
		t.Fatal("b")
	}
	if _, ok := n.Get("c").Bool(); ok {
		t.Fatal("yes is not bool")
	}
}

func TestErrUnterminatedQuote(t *testing.T) {
	wantErr(t, parseErr(t, "a: \"abc\n"), 1, 8, "닫히지 않은 따옴표")
}

func TestErrTrailingAfterQuote(t *testing.T) {
	wantErr(t, parseErr(t, "a: \"abc\" def\n"), 1, 9, "닫는 따옴표 뒤")
}

func TestErrBadEscape(t *testing.T) {
	wantErr(t, parseErr(t, "a: \"\\q\"\n"), 1, 5, "이스케이프")
}

func TestErrUnsupportedScalars(t *testing.T) {
	cases := []struct{ src, want string }{
		{"a: 'x'\n", "홑따옴표"},
		{"a: &anchor\n", "앵커"},
		{"a: *alias\n", "앵커"},
		{"a: !!str x\n", "태그"},
		{"a: |\n  text\n", "여러 줄 문자열"},
		{"a: >\n  text\n", "여러 줄 문자열"},
		{"a: %x\n", "지원하지 않는다"},
	}
	for _, c := range cases {
		wantErr(t, parseErr(t, c.src), 1, 4, c.want)
	}
}

func TestQuotedKey(t *testing.T) {
	n := parse(t, "\"a b\": 1\n")
	if n.Entries[0].Key != "a b" || n.Entries[0].Value.Value != "1" {
		t.Fatalf("%+v", n.Entries[0])
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/yaml/`
Expected: FAIL (`TestQuotedScalar`의 이스케이프, `TestErrUnsupportedScalars`)

- [ ] **Step 3: 구현**

`internal/yaml/scalar.go` 전체:

```go
package yaml

import (
	"fmt"
	"strings"
)

// unquote는 s[0] == '"'인 문자열을 읽어 값, 소비한 바이트 수, 오류 메시지를 준다.
// 오류일 때 n은 오류 위치 오프셋이다.
func unquote(s string) (val string, n int, msg string) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '"':
			return b.String(), i + 1, ""
		case '\\':
			if i+1 >= len(s) {
				return "", i, "닫히지 않은 따옴표"
			}
			i++
			switch s[i] {
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				return "", i - 1, fmt.Sprintf("지원하지 않는 이스케이프 \\%c", s[i])
			}
		default:
			b.WriteByte(s[i])
		}
	}
	return "", len(s), "닫히지 않은 따옴표"
}

// unsupportedScalar는 평문 스칼라가 지원하지 않는 문법으로 시작하면 메시지를 준다.
func unsupportedScalar(s string) string {
	switch s[0] {
	case '\'':
		return "홑따옴표 문자열은 지원하지 않는다. \"...\"를 쓴다"
	case '&', '*':
		return "앵커와 별칭은 지원하지 않는다"
	case '!':
		return "태그는 지원하지 않는다"
	case '|', '>':
		return "여러 줄 문자열은 지원하지 않는다"
	case '%', '@', '`':
		return fmt.Sprintf("%q로 시작하는 값은 지원하지 않는다", s[0])
	}
	return ""
}
```

- [ ] **Step 4: 통과 확인**

Run: `gofmt -l ./internal/yaml/ && go test ./internal/yaml/`
Expected: PASS. `TestErrUnterminatedQuote`의 열 8은 `a: "abc` 길이 7 다음이다.

- [ ] **Step 5: 커밋**

```bash
git add internal/yaml/
git commit -m "feat: YAML 스칼라: 따옴표 이스케이프와 지원하지 않는 문법 오류"
```

---

### Task 4: YAML 플로우 컬렉션

**Files:**
- Modify: `internal/yaml/flow.go`
- Test: `internal/yaml/yaml_test.go`

**Interfaces:**
- Produces: `{}`·`[]`가 서로 중첩되고, `[`는 닫힐 때까지 다음 줄을 이어 읽는다. `{`는 한 줄 안에서 닫혀야 한다. 빈 `{}`·`[]` 허용. 뒤에 항목 없는 쉼표는 오류.

- [ ] **Step 1: 실패하는 테스트**

```go
func TestFlowMap(t *testing.T) {
	n := parse(t, "m: { tier: domain, depends_on: [crypto, metrics], go: { type: TxType } }\n")
	m := n.Get("m")
	if m.Kind != Map || len(m.Entries) != 3 {
		t.Fatalf("%+v", m)
	}
	if m.Get("tier").Value != "domain" || m.Get("tier").Pos.Col != 12 {
		t.Fatalf("tier %+v", m.Get("tier"))
	}
	deps := m.Get("depends_on")
	if deps.Kind != Seq || len(deps.Items) != 2 || deps.Items[1].Value != "metrics" {
		t.Fatalf("deps %+v", deps)
	}
	if m.Get("go").Get("type").Value != "TxType" {
		t.Fatalf("go %+v", m.Get("go"))
	}
}

func TestFlowEmpty(t *testing.T) {
	n := parse(t, "a: {}\nb: []\nc: { req: {} }\n")
	if n.Get("a").Kind != Map || len(n.Get("a").Entries) != 0 {
		t.Fatal("a")
	}
	if n.Get("b").Kind != Seq || len(n.Get("b").Items) != 0 {
		t.Fatal("b")
	}
	if n.Get("c").Get("req").Kind != Map {
		t.Fatal("c")
	}
}

func TestFlowQuotedAndPaths(t *testing.T) {
	n := parse(t, "p: { ctx: \"*Context\", payload: json.RawMessage, path: /api/v1, spec: docs/a.yaml }\n")
	p := n.Get("p")
	if p.Get("ctx").Value != "*Context" || !p.Get("ctx").Quoted {
		t.Fatalf("ctx %+v", p.Get("ctx"))
	}
	if p.Get("path").Value != "/api/v1" || p.Get("spec").Value != "docs/a.yaml" {
		t.Fatalf("%+v", p)
	}
}

func TestFlowNullValue(t *testing.T) {
	n := parse(t, "g: { post: null, x: }\n")
	if !n.Get("g").Get("post").IsNull() || !n.Get("g").Get("x").IsNull() {
		t.Fatalf("%+v", n.Get("g"))
	}
}

func TestFlowSeqOfMaps(t *testing.T) {
	src := "cases:\n  - { name: \"금액 0\", given: { amount: 0 }, error: invalid }\n  - { name: ok, given: { a: 1 }, expect: { from: 7 } }\n"
	c := parse(t, src).Get("cases")
	if len(c.Items) != 2 || c.Items[0].Get("name").Value != "금액 0" {
		t.Fatalf("%+v", c)
	}
	if v, _ := c.Items[1].Get("expect").Get("from").Int(); v != 7 {
		t.Fatalf("expect %+v", c.Items[1].Get("expect"))
	}
	if c.Items[1].Pos.Line != 3 || c.Items[1].Pos.Col != 5 {
		t.Fatalf("pos %v", c.Items[1].Pos)
	}
}

func TestFlowMultiLineSeq(t *testing.T) {
	src := "values: [transfer, mint_asset,\n         # 주석 줄\n\n         grant_reward]\nnext: 1\n"
	n := parse(t, src)
	v := n.Get("values")
	if len(v.Items) != 3 || v.Items[2].Value != "grant_reward" {
		t.Fatalf("%+v", v)
	}
	if v.Items[2].Pos.Line != 4 || v.Items[2].Pos.Col != 10 {
		t.Fatalf("pos %v", v.Items[2].Pos)
	}
	if n.Get("next").Value != "1" {
		t.Fatal("next")
	}
}

func TestErrFlowMapMultiLine(t *testing.T) {
	wantErr(t, parseErr(t, "a: { x: 1,\n  y: 2 }\n"), 1, 4, "한 줄")
}

func TestErrFlowUnclosedSeq(t *testing.T) {
	wantErr(t, parseErr(t, "a: [x, y\n"), 1, 4, "닫히지 않은 [")
}

func TestErrFlowTrailingComma(t *testing.T) {
	wantErr(t, parseErr(t, "a: [x, y, ]\n"), 1, 11, "쉼표 뒤")
}

func TestErrFlowDuplicateKey(t *testing.T) {
	wantErr(t, parseErr(t, "a: { x: 1, x: 2 }\n"), 1, 12, "중복 키")
}

func TestErrFlowMissingColon(t *testing.T) {
	wantErr(t, parseErr(t, "a: { x 1 }\n"), 1, 10, "콜론")
}

func TestErrFlowTrailingText(t *testing.T) {
	wantErr(t, parseErr(t, "a: [x] y\n"), 1, 8, "닫는 괄호 뒤")
}

func TestErrFlowUnsupportedScalar(t *testing.T) {
	wantErr(t, parseErr(t, "a: [x, *y]\n"), 1, 8, "앵커")
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/yaml/`
Expected: FAIL, "플로우 컬렉션은 아직 지원하지 않는다"

- [ ] **Step 3: 구현**

`internal/yaml/flow.go` 전체:

```go
package yaml

import (
	"fmt"
	"strings"
)

// fc는 위치가 붙은 바이트다. 여러 줄 [ ]를 한 스트림으로 이어 읽기 위해 쓴다.
type fc struct {
	r   byte
	pos Pos
}

func (p *parser) parseFlow(ln line, rest string, col int) (*Node, error) {
	var cs []fc
	push := func(s string, num, col int) {
		for i := 0; i < len(s); i++ {
			cs = append(cs, fc{s[i], Pos{p.file, num, col + i}})
		}
	}
	push(rest, ln.num, col)
	depth, err := flowDepth(rest, 0)
	if err != "" {
		return nil, p.errAt(ln.num, col, err)
	}
	if depth > 0 && rest[0] == '{' {
		return nil, p.errAt(ln.num, col, "{ }는 한 줄 안에서 닫혀야 한다")
	}
	for depth > 0 {
		if p.i >= len(p.lines) {
			return nil, p.errAt(ln.num, col, "닫히지 않은 [")
		}
		nx := p.lines[p.i]
		p.i++
		cs = append(cs, fc{' ', Pos{p.file, nx.num, nx.indent}})
		push(nx.text, nx.num, nx.indent+1)
		if depth, err = flowDepth(nx.text, depth); err != "" {
			return nil, p.errAt(nx.num, nx.indent+1, err)
		}
	}
	f := &flow{cs: cs}
	n, e := f.value()
	if e != nil {
		return nil, e
	}
	f.skipSpace()
	if !f.eof() {
		return nil, f.err("닫는 괄호 뒤에 문자가 있다")
	}
	return n, nil
}

// flowDepth는 따옴표 밖의 괄호 깊이를 누적한다. 줄 끝에서 따옴표가 열려 있으면 오류다.
func flowDepth(s string, depth int) (int, string) {
	inQ := false
	for i := 0; i < len(s); i++ {
		switch {
		case inQ && s[i] == '\\':
			i++
		case s[i] == '"':
			inQ = !inQ
		case inQ:
		case s[i] == '{' || s[i] == '[':
			depth++
		case s[i] == '}' || s[i] == ']':
			depth--
		}
	}
	if inQ {
		return depth, "닫히지 않은 따옴표"
	}
	return depth, ""
}

type flow struct {
	cs []fc
	i  int
}

func (f *flow) eof() bool  { return f.i >= len(f.cs) }
func (f *flow) peek() byte { return f.cs[f.i].r }

func (f *flow) pos() Pos {
	if f.eof() {
		last := f.cs[len(f.cs)-1].pos
		last.Col++
		return last
	}
	return f.cs[f.i].pos
}

func (f *flow) err(msg string) error { return &Error{f.pos(), msg} }

func (f *flow) skipSpace() {
	for !f.eof() && f.peek() == ' ' {
		f.i++
	}
}

func (f *flow) rest() string {
	b := make([]byte, 0, len(f.cs)-f.i)
	for _, c := range f.cs[f.i:] {
		b = append(b, c.r)
	}
	return string(b)
}

func (f *flow) value() (*Node, error) {
	f.skipSpace()
	if f.eof() {
		return nil, f.err("값이 없다")
	}
	switch f.peek() {
	case '{':
		return f.mapping()
	case '[':
		return f.sequence()
	case '"':
		return f.quoted()
	}
	return f.plain()
}

func (f *flow) quoted() (*Node, error) {
	pos := f.pos()
	s, n, msg := unquote(f.rest())
	if msg != "" {
		f.i += n
		return nil, f.err(msg)
	}
	f.i += n
	return &Node{Kind: Scalar, Pos: pos, Value: s, Quoted: true}, nil
}

// plainText는 stops 중 하나 앞까지 읽고 오른쪽 공백을 뗀다.
func (f *flow) plainText(stops string) string {
	start := f.i
	for !f.eof() && strings.IndexByte(stops, f.peek()) < 0 {
		f.i++
	}
	b := make([]byte, 0, f.i-start)
	for _, c := range f.cs[start:f.i] {
		b = append(b, c.r)
	}
	return strings.TrimRight(string(b), " ")
}

func (f *flow) plain() (*Node, error) {
	pos := f.pos()
	s := f.plainText(",}]")
	if s == "" {
		return nil, &Error{pos, "값이 없다"}
	}
	if msg := unsupportedScalar(s); msg != "" {
		return nil, &Error{pos, msg}
	}
	return &Node{Kind: Scalar, Pos: pos, Value: s}, nil
}

// afterItem은 항목 뒤의 ',' 또는 닫는 괄호를 처리한다. 닫혔으면 true.
func (f *flow) afterItem(close byte) (bool, error) {
	f.skipSpace()
	if f.eof() {
		return false, f.err(fmt.Sprintf("닫히지 않은 %c", open(close)))
	}
	switch f.peek() {
	case ',':
		f.i++
		f.skipSpace()
		if !f.eof() && f.peek() == close {
			return false, f.err("쉼표 뒤에 항목이 없다")
		}
		return false, nil
	case close:
		f.i++
		return true, nil
	}
	return false, f.err(fmt.Sprintf("쉼표나 %c가 와야 한다", close))
}

func open(close byte) byte {
	if close == '}' {
		return '{'
	}
	return '['
}

func (f *flow) mapping() (*Node, error) {
	n := &Node{Kind: Map, Pos: f.pos()}
	f.i++
	seen := map[string]Pos{}
	f.skipSpace()
	if !f.eof() && f.peek() == '}' {
		f.i++
		return n, nil
	}
	for {
		f.skipSpace()
		if f.eof() {
			return nil, f.err("닫히지 않은 {")
		}
		keyPos := f.pos()
		var key string
		if f.peek() == '"' {
			q, err := f.quoted()
			if err != nil {
				return nil, err
			}
			key = q.Value
		} else {
			key = f.plainText(":,}]")
			if key == "" {
				return nil, &Error{keyPos, "키가 없다"}
			}
		}
		f.skipSpace()
		if f.eof() || f.peek() != ':' {
			return nil, f.err("키 뒤에 콜론이 없다")
		}
		f.i++
		if !f.eof() && f.peek() != ' ' && f.peek() != ',' && f.peek() != '}' {
			return nil, f.err("콜론 뒤에 공백이 필요하다")
		}
		if prev, dup := seen[key]; dup {
			return nil, &Error{keyPos, fmt.Sprintf("중복 키 %q (처음: %s)", key, prev)}
		}
		seen[key] = keyPos
		f.skipSpace()
		var val *Node
		if !f.eof() && (f.peek() == ',' || f.peek() == '}') {
			val = &Node{Kind: Scalar, Pos: f.pos()}
		} else {
			v, err := f.value()
			if err != nil {
				return nil, err
			}
			val = v
		}
		n.Entries = append(n.Entries, Entry{key, keyPos, val})
		closed, err := f.afterItem('}')
		if err != nil {
			return nil, err
		}
		if closed {
			return n, nil
		}
	}
}

func (f *flow) sequence() (*Node, error) {
	n := &Node{Kind: Seq, Pos: f.pos()}
	f.i++
	f.skipSpace()
	if !f.eof() && f.peek() == ']' {
		f.i++
		return n, nil
	}
	for {
		item, err := f.value()
		if err != nil {
			return nil, err
		}
		n.Items = append(n.Items, item)
		closed, err := f.afterItem(']')
		if err != nil {
			return nil, err
		}
		if closed {
			return n, nil
		}
	}
}
```

- [ ] **Step 4: 통과 확인**

Run: `gofmt -l ./internal/yaml/ && go vet ./internal/yaml/ && go test ./internal/yaml/`
Expected: PASS. `TestErrFlowMissingColon`은 `{ x 1 }`에서 키 `x 1`을 읽은 뒤 `}`에서 "키 뒤에 콜론이 없다"가 열 10에 나온다.

- [ ] **Step 5: 커밋**

```bash
git add internal/yaml/
git commit -m "feat: YAML 플로우 컬렉션: 중첩과 여러 줄 목록"
```

---

### Task 5: 모델 골격, 진단, 파일 병합, 요구사항

**Files:**
- Create: `internal/model/diag.go`, `internal/model/model.go`, `internal/model/decode.go`, `internal/model/load.go`, `internal/model/decode_layers.go`(이 Task에서는 `requirements`만), `internal/model/load_test.go`

**Interfaces:**
- Produces:
  - `type Level int` (`Info`, `Warning`, `Error`), `func (Level) String()` → `정보`·`경고`·`오류`
  - `type Diagnostic struct{ Pos yaml.Pos; Level Level; Msg string }`, `String()` → `file:line:col: 수준: 메시지`. `Pos.Line == 0`이면 `file: 수준: 메시지`
  - `type Diagnostics []Diagnostic`, `Add`, `Errorf`, `Warnf`, `Infof`, `HasErrors() bool`, `Count(Level) int`, `Sort()`
  - `func Load(dir string) (*Model, Diagnostics)`. 모델 단계 오류가 있어도 부분 모델을 돌려준다. 호출자는 `HasErrors()`로 진행 여부를 정한다.
  - `type Ref struct{ Name string; Pos yaml.Pos }`, `type Meta struct{ ID string; Pos yaml.Pos; Summary string; Satisfies []Ref }`
  - 노드 타입 전체(아래 `model.go`). 뒤 Task와 계획 2·3이 이 필드 이름을 그대로 쓴다.

- [ ] **Step 1: 실패하는 테스트**

`internal/model/load_test.go`:

```go
package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func load(t *testing.T, files map[string]string) (*Model, Diagnostics) {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Load(dir)
}

func loadOne(t *testing.T, src string) (*Model, Diagnostics) {
	t.Helper()
	return load(t, map[string]string{"d.yaml": "version: 0.3\n" + src})
}

func find(ds Diagnostics, level Level, substr string) *Diagnostic {
	for i := range ds {
		if ds[i].Level == level && strings.Contains(ds[i].Msg, substr) {
			return &ds[i]
		}
	}
	return nil
}

func wantDiag(t *testing.T, ds Diagnostics, level Level, substr string) *Diagnostic {
	t.Helper()
	d := find(ds, level, substr)
	if d == nil {
		t.Fatalf("no %v containing %q in:\n%s", level, substr, join(ds))
	}
	return d
}

func wantClean(t *testing.T, ds Diagnostics) {
	t.Helper()
	if len(ds) != 0 {
		t.Fatalf("unexpected diagnostics:\n%s", join(ds))
	}
}

func join(ds Diagnostics) string {
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func TestNoDesignFiles(t *testing.T) {
	_, ds := Load(t.TempDir())
	d := wantDiag(t, ds, Error, "설계 파일")
	if d.Pos.Line != 0 || !strings.HasSuffix(d.String(), ": 오류: 설계 파일(*.yaml)이 없다") {
		t.Fatalf("%q", d.String())
	}
}

func TestTopLevelScalars(t *testing.T) {
	m, ds := loadOne(t, "go_module: github.com/x/y\nts_out: web/gen.ts\n")
	wantClean(t, ds)
	if m.Version != "0.3" || m.GoModule != "github.com/x/y" || m.TSOut != "web/gen.ts" {
		t.Fatalf("%+v", m)
	}
}

func TestVersionMismatch(t *testing.T) {
	_, ds := load(t, map[string]string{"d.yaml": "version: 0.2\n"})
	wantDiag(t, ds, Error, "0.3")
}

func TestUnknownTopKey(t *testing.T) {
	_, ds := loadOne(t, "modulez: {}\n")
	d := wantDiag(t, ds, Error, `알 수 없는 키 "modulez"`)
	if d.Pos.Line != 2 || d.Pos.Col != 1 {
		t.Fatalf("pos %v", d.Pos)
	}
}

func TestParseErrorBecomesDiagnostic(t *testing.T) {
	_, ds := load(t, map[string]string{"a.yaml": "version: 0.3\n", "b.yaml": "x:\n\ty: 1\n"})
	d := wantDiag(t, ds, Error, "탭")
	if !strings.HasSuffix(d.Pos.File, "b.yaml") || d.Pos.Line != 2 {
		t.Fatalf("pos %v", d.Pos)
	}
}

func TestMergeAcrossFiles(t *testing.T) {
	m, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\nrequirements:\n  REQ-a: { text: A }\n",
		"b.yaml": "version: 0.3\nrequirements:\n  REQ-b: { text: B, status: held }\n",
	})
	wantClean(t, ds)
	if len(m.Requirements) != 2 || m.Requirements[0].ID != "REQ-a" || m.Requirements[1].Status != "held" {
		t.Fatalf("%+v", m.Requirements)
	}
	if m.Requirements[0].Status != "active" {
		t.Fatal("default status")
	}
}

func TestDuplicateIDAcrossFiles(t *testing.T) {
	_, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\nrequirements:\n  REQ-a: { text: A }\n",
		"b.yaml": "version: 0.3\nrequirements:\n  REQ-a: { text: B }\n",
	})
	d := wantDiag(t, ds, Error, `중복 ID 요구사항 "REQ-a"`)
	if !strings.Contains(d.Msg, "a.yaml:3:3") || !strings.HasSuffix(d.Pos.File, "b.yaml") {
		t.Fatalf("%v", d)
	}
}

func TestScalarConflictAcrossFiles(t *testing.T) {
	_, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\ngo_module: x\n",
		"b.yaml": "version: 0.3\ngo_module: y\n",
	})
	wantDiag(t, ds, Error, `"go_module"가 다른 파일`)
}

func TestRequirementFields(t *testing.T) {
	_, ds := loadOne(t, "requirements:\n  REQ-a: { text: A, status: maybe, extra: 1 }\n  REQ-b: { status: active }\n")
	wantDiag(t, ds, Error, "status는 active, held, dropped")
	wantDiag(t, ds, Error, `알 수 없는 키 "extra"`)
	wantDiag(t, ds, Error, `"text"가 필요하다`)
}

func TestDiagnosticsSort(t *testing.T) {
	ds := Diagnostics{
		{Pos: pos("b", 1, 1)}, {Pos: pos("a", 2, 1)}, {Pos: pos("a", 1, 5)}, {Pos: pos("a", 1, 2)},
	}
	ds.Sort()
	got := ds[0].Pos.String() + " " + ds[1].Pos.String() + " " + ds[2].Pos.String() + " " + ds[3].Pos.String()
	if got != "a:1:2 a:1:5 a:2:1 b:1:1" {
		t.Fatal(got)
	}
}
```

`pos`는 테스트 도우미다. 같은 파일 끝에 추가:

```go
func pos(file string, line, col int) yaml.Pos { return yaml.Pos{File: file, Line: line, Col: col} }
```

import에 `"github.com/tolelom/daemok/internal/yaml"`을 더한다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/model/`
Expected: FAIL, `undefined: Load`

- [ ] **Step 3: 진단**

`internal/model/diag.go`:

```go
package model

import (
	"fmt"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

type Level int

const (
	Info Level = iota
	Warning
	Error
)

func (l Level) String() string {
	switch l {
	case Error:
		return "오류"
	case Warning:
		return "경고"
	}
	return "정보"
}

type Diagnostic struct {
	Pos   yaml.Pos
	Level Level
	Msg   string
}

func (d Diagnostic) String() string {
	if d.Pos.Line == 0 {
		return fmt.Sprintf("%s: %s: %s", d.Pos.File, d.Level, d.Msg)
	}
	return fmt.Sprintf("%s: %s: %s", d.Pos, d.Level, d.Msg)
}

type Diagnostics []Diagnostic

func (ds *Diagnostics) Add(pos yaml.Pos, level Level, format string, args ...any) {
	*ds = append(*ds, Diagnostic{Pos: pos, Level: level, Msg: fmt.Sprintf(format, args...)})
}

func (ds *Diagnostics) Errorf(pos yaml.Pos, format string, args ...any) {
	ds.Add(pos, Error, format, args...)
}

func (ds *Diagnostics) Warnf(pos yaml.Pos, format string, args ...any) {
	ds.Add(pos, Warning, format, args...)
}

func (ds *Diagnostics) Infof(pos yaml.Pos, format string, args ...any) {
	ds.Add(pos, Info, format, args...)
}

func (ds Diagnostics) HasErrors() bool { return ds.Count(Error) > 0 }

func (ds Diagnostics) Count(level Level) int {
	n := 0
	for _, d := range ds {
		if d.Level == level {
			n++
		}
	}
	return n
}

func (ds Diagnostics) Sort() {
	sort.SliceStable(ds, func(i, j int) bool {
		a, b := ds[i].Pos, ds[j].Pos
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Col < b.Col
	})
}
```

- [ ] **Step 4: 노드 타입**

`internal/model/model.go`:

```go
// Package model은 설계 파일을 노드 타입으로 옮기고 여러 파일을 하나로 합친다.
package model

import "github.com/tolelom/daemok/internal/yaml"

// Ref는 다른 노드를 가리키는 이름과 그 위치다. 해석은 check가 한다.
type Ref struct {
	Name string
	Pos  yaml.Pos
}

type Meta struct {
	ID        string
	Pos       yaml.Pos // ID 키 위치
	Summary   string
	Satisfies []Ref
}

type Requirement struct {
	Meta
	Text   string
	Status string // active, held, dropped
}

type Actor struct{ Meta }

type Interface struct {
	Meta
	Protocol string
	Port     int
	Path     string
	Spec     string
	UsedBy   []Ref
}

type System struct {
	Meta
	Actors     []*Actor
	Interfaces []*Interface
}

type Executable struct {
	Meta
	Exposes  []Ref
	Composes []Ref
}

type Module struct {
	Meta
	Tier       Ref
	Path       string
	DependsOn  []Ref
	Implements []Ref
}

type Group struct {
	Meta
	Tier      Ref
	Kind      Ref
	Path      string
	DependsOn []Ref
	Rule      string
	Members   []*Member
}

type Member struct {
	Meta
	Group      *Group
	Handles    []Ref
	Exceptions []Exception
}

type Exception struct {
	DependsOn Ref
	Reason    string
}

type GoOpts struct {
	Type     string
	Name     string
	Prefix   string
	Register string
}

// Owned는 모듈이나 멤버에 속하는 L3·L4 노드의 공통 부분이다.
type Owned struct {
	Meta
	In  Ref
	Gen bool // 기본 true. false면 생성에서 뺀다
}

type Enum struct {
	Owned
	Values []Ref
	Go     GoOpts
	Export []string
}

type Const struct {
	Owned
	Type     string // int, string, duration
	Value    string
	ValuePos yaml.Pos
	Go       GoOpts
	Export   []string
}

type ErrorDef struct {
	Owned
	Message string
	HTTP    int // 0이면 없음
}

type Field struct {
	Name     string
	Pos      yaml.Pos
	Type     Ref // type 또는 ref 중 하나만 채워진다
	RefType  Ref
	Many     bool
	Optional bool
	Max      Ref // 상수 ID 또는 숫자
}

type Type struct {
	Owned
	Fields []*Field
	Go     GoOpts
}

type Param struct {
	Name string
	Pos  yaml.Pos
	Type Ref
}

type Method struct {
	Name      string
	Pos       yaml.Pos
	Params    []*Param
	Returns   []Ref
	Errors    []Ref
	HasErrors bool // errors 키가 있었는지
}

type Contract struct {
	Owned
	Go      GoOpts
	Methods []*Method
}

type HandlerKind struct {
	Owned
	Params  []*Param
	Returns []Ref
	Key     Ref
	Go      GoOpts
}

type Case struct {
	Name   string
	Pos    yaml.Pos
	Given  *yaml.Node // 맵
	Expect *yaml.Node // 맵 또는 nil
	Error  Ref        // 에러 ID, "invalid", 또는 비어 있음
}

type Func struct {
	Owned
	Kind    Ref
	Handles Ref
	Of      Ref // "계약.메서드"
	Errors  []Ref
	Cases   []*Case
}

type Model struct {
	Version  string
	GoModule string
	TSOut    string

	Requirements []*Requirement
	System       *System
	Executables  []*Executable
	Tiers        []Ref
	Modules      []*Module
	Groups       []*Group
	Enums        []*Enum
	Consts       []*Const
	Errors       []*ErrorDef
	Types        []*Type
	Contracts    []*Contract
	HandlerKinds []*HandlerKind
	Funcs        []*Func
}
```

- [ ] **Step 5: 디코더 도우미**

`internal/model/decode.go`:

```go
package model

import (
	"slices"

	"github.com/tolelom/daemok/internal/yaml"
)

type decoder struct {
	diags *Diagnostics
}

func (d *decoder) mapping(n *yaml.Node, what string) bool {
	if n == nil {
		return false
	}
	if n.Kind != yaml.Map {
		d.diags.Errorf(n.Pos, "%s는 맵이어야 한다", what)
		return false
	}
	return true
}

func (d *decoder) allow(n *yaml.Node, keys ...string) {
	for _, e := range n.Entries {
		if !slices.Contains(keys, e.Key) {
			d.diags.Errorf(e.Pos, "알 수 없는 키 %q", e.Key)
		}
	}
}

func (d *decoder) scalar(n *yaml.Node, key string) *yaml.Node {
	v := n.Get(key)
	if v == nil {
		return nil
	}
	if v.Kind != yaml.Scalar {
		d.diags.Errorf(v.Pos, "%q는 값 하나여야 한다", key)
		return nil
	}
	return v
}

func (d *decoder) str(n *yaml.Node, key string) string {
	if v := d.scalar(n, key); v != nil {
		return v.Value
	}
	return ""
}

func (d *decoder) need(n *yaml.Node, key string) bool {
	if n.Get(key) == nil {
		d.diags.Errorf(n.Pos, "%q가 필요하다", key)
		return false
	}
	return true
}

func (d *decoder) ref(n *yaml.Node, key string) Ref {
	v := d.scalar(n, key)
	if v == nil || v.IsNull() {
		return Ref{}
	}
	return Ref{Name: v.Value, Pos: v.Pos}
}

func (d *decoder) refs(n *yaml.Node, key string) []Ref {
	v := n.Get(key)
	if v == nil || v.IsNull() {
		return nil
	}
	if v.Kind != yaml.Seq {
		d.diags.Errorf(v.Pos, "%q는 목록이어야 한다", key)
		return nil
	}
	out := []Ref{}
	for _, it := range v.Items {
		if it.Kind != yaml.Scalar {
			d.diags.Errorf(it.Pos, "%q의 항목은 값 하나여야 한다", key)
			continue
		}
		out = append(out, Ref{Name: it.Value, Pos: it.Pos})
	}
	return out
}

func (d *decoder) strs(n *yaml.Node, key string) []string {
	var out []string
	for _, r := range d.refs(n, key) {
		out = append(out, r.Name)
	}
	return out
}

func (d *decoder) intv(n *yaml.Node, key string) (int, bool) {
	v := d.scalar(n, key)
	if v == nil {
		return 0, false
	}
	i, ok := v.Int()
	if !ok {
		d.diags.Errorf(v.Pos, "%q는 정수여야 한다", key)
	}
	return i, ok
}

func (d *decoder) boolv(n *yaml.Node, key string, def bool) bool {
	v := d.scalar(n, key)
	if v == nil {
		return def
	}
	b, ok := v.Bool()
	if !ok {
		d.diags.Errorf(v.Pos, "%q는 true 또는 false여야 한다", key)
		return def
	}
	return b
}

func (d *decoder) meta(id string, pos yaml.Pos, n *yaml.Node) Meta {
	return Meta{ID: id, Pos: pos, Summary: d.str(n, "summary"), Satisfies: d.refs(n, "satisfies")}
}

func (d *decoder) owned(id string, pos yaml.Pos, n *yaml.Node) Owned {
	o := Owned{Meta: d.meta(id, pos, n), Gen: d.boolv(n, "gen", true)}
	if d.need(n, "in") {
		o.In = d.ref(n, "in")
	}
	return o
}

func (d *decoder) goOpts(n *yaml.Node) GoOpts {
	g := n.Get("go")
	if g == nil || !d.mapping(g, "go") {
		return GoOpts{}
	}
	d.allow(g, "type", "name", "prefix", "register")
	return GoOpts{Type: d.str(g, "type"), Name: d.str(g, "name"), Prefix: d.str(g, "prefix"), Register: d.str(g, "register")}
}

// idMap은 ID를 키로 하는 맵을 돈다. 값이 맵이 아닌 항목은 오류로 남기고 건너뛴다.
func (d *decoder) idMap(n *yaml.Node, what string, fn func(id string, pos yaml.Pos, v *yaml.Node)) {
	if n == nil || n.IsNull() || !d.mapping(n, what) {
		return
	}
	for _, e := range n.Entries {
		if e.Value.Kind != yaml.Map {
			d.diags.Errorf(e.Pos, "%s %q는 맵이어야 한다", what, e.Key)
			continue
		}
		fn(e.Key, e.Pos, e.Value)
	}
}
```

- [ ] **Step 6: Load와 병합**

`internal/model/load.go`:

```go
package model

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

var topKeys = []string{
	"version", "go_module", "ts_out",
	"requirements", "system", "executables", "tiers", "modules", "groups",
	"enums", "consts", "errors", "types", "contracts", "handler_kinds", "funcs",
}

// Load는 dir의 *.yaml을 이름순으로 읽어 한 모델로 합친다.
// 오류가 있어도 읽은 만큼의 모델을 돌려준다.
func Load(dir string) (*Model, Diagnostics) {
	var diags Diagnostics
	paths, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if len(paths) == 0 {
		diags.Errorf(yaml.Pos{File: dir}, "설계 파일(*.yaml)이 없다")
		return nil, diags
	}
	sort.Strings(paths)
	mg := &merger{
		m:    &Model{},
		d:    &decoder{diags: &diags},
		ids:  map[string]map[string]yaml.Pos{},
		once: map[string]yaml.Pos{},
	}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			diags.Errorf(yaml.Pos{File: p}, "읽을 수 없다: %v", err)
			continue
		}
		root, perr := yaml.Parse(p, src)
		if perr != nil {
			e := perr.(*yaml.Error)
			diags.Add(e.Pos, Error, "%s", e.Msg)
			continue
		}
		mg.file(root)
	}
	mg.finish()
	return mg.m, diags
}

type merger struct {
	m    *Model
	d    *decoder
	ids  map[string]map[string]yaml.Pos // 종류 → ID → 처음 위치
	once map[string]yaml.Pos            // 한 번만 나올 수 있는 최상위 키
}

func (mg *merger) file(root *yaml.Node) {
	d := mg.d
	if !d.mapping(root, "설계 파일") {
		return
	}
	d.allow(root, topKeys...)
	mg.scalarOnce(root, "version", &mg.m.Version)
	if v := root.Get("version"); v != nil && v.Kind == yaml.Scalar && v.Value != "0.3" {
		d.diags.Errorf(v.Pos, "지원하는 형식 버전은 0.3이다")
	}
	mg.scalarOnce(root, "go_module", &mg.m.GoModule)
	mg.scalarOnce(root, "ts_out", &mg.m.TSOut)
	mg.requirements(root.Get("requirements"))
	mg.system(root.Get("system"))
	mg.executables(root.Get("executables"))
	mg.tiers(root.Get("tiers"))
	mg.modules(root.Get("modules"))
	mg.groups(root.Get("groups"))
	mg.enums(root.Get("enums"))
	mg.consts(root.Get("consts"))
	mg.errors(root.Get("errors"))
	mg.types(root.Get("types"))
	mg.contracts(root.Get("contracts"))
	mg.handlerKinds(root.Get("handler_kinds"))
	mg.funcs(root.Get("funcs"))
}

func (mg *merger) scalarOnce(root *yaml.Node, key string, dst *string) {
	v := mg.d.scalar(root, key)
	if v == nil {
		return
	}
	if prev, ok := mg.once[key]; ok && *dst != v.Value {
		mg.d.diags.Errorf(v.Pos, "%q가 다른 파일(%s)과 다르다", key, prev)
		return
	}
	mg.once[key] = v.Pos
	*dst = v.Value
}

// unique는 종류 안의 ID 중복을 잡는다. 처음 보는 ID면 true.
func (mg *merger) unique(kind, id string, pos yaml.Pos) bool {
	if mg.ids[kind] == nil {
		mg.ids[kind] = map[string]yaml.Pos{}
	}
	if prev, dup := mg.ids[kind][id]; dup {
		mg.d.diags.Errorf(pos, "중복 ID %s %q (처음: %s)", kind, id, prev)
		return false
	}
	mg.ids[kind][id] = pos
	return true
}

// finish는 한 참조 필드가 받는 종류끼리 ID가 겹치는지 본다.
func (mg *merger) finish() {
	mg.collide("모듈", "그룹")
	mg.collide("모듈", "멤버")
	mg.collide("그룹", "멤버")
	mg.collide("enum", "타입")
}

func (mg *merger) collide(a, b string) {
	for id, posB := range mg.ids[b] {
		if posA, ok := mg.ids[a][id]; ok {
			mg.d.diags.Errorf(posB, "%s %q가 %s(%s)와 같은 이름이라 참조가 모호하다", b, id, a, posA)
		}
	}
}
```

- [ ] **Step 7: 요구사항 디코드와 나머지 계층의 빈 함수**

`internal/model/decode_layers.go`:

```go
package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) requirements(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "requirements", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "text", "status", "summary")
		if !mg.unique("요구사항", id, pos) {
			return
		}
		r := &Requirement{Meta: Meta{ID: id, Pos: pos, Summary: d.str(v, "summary")}}
		if d.need(v, "text") {
			r.Text = d.str(v, "text")
		}
		r.Status = d.str(v, "status")
		if r.Status == "" {
			r.Status = "active"
		}
		switch r.Status {
		case "active", "held", "dropped":
		default:
			d.diags.Errorf(v.Get("status").Pos, "status는 active, held, dropped 중 하나여야 한다")
		}
		mg.m.Requirements = append(mg.m.Requirements, r)
	})
}

func (mg *merger) system(n *yaml.Node)      {}
func (mg *merger) executables(n *yaml.Node) {}
func (mg *merger) tiers(n *yaml.Node)       {}
func (mg *merger) modules(n *yaml.Node)     {}
func (mg *merger) groups(n *yaml.Node)      {}
```

`internal/model/decode_l3.go`:

```go
package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) enums(n *yaml.Node)        {}
func (mg *merger) consts(n *yaml.Node)       {}
func (mg *merger) errors(n *yaml.Node)       {}
func (mg *merger) types(n *yaml.Node)        {}
func (mg *merger) contracts(n *yaml.Node)    {}
func (mg *merger) handlerKinds(n *yaml.Node) {}
```

`internal/model/decode_l4.go`:

```go
package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) funcs(n *yaml.Node) {}
```

- [ ] **Step 8: 통과 확인**

Run: `gofmt -l ./internal/ && go vet ./internal/... && go test ./internal/model/`
Expected: PASS

- [ ] **Step 9: 커밋**

```bash
git add internal/model/
git commit -m "feat: 모델 골격: 진단, 파일 병합, 중복 ID, 요구사항"
```

---

### Task 6: L0·L1·L2 디코드

**Files:**
- Modify: `internal/model/decode_layers.go`
- Test: `internal/model/load_test.go`

**Interfaces:**
- Produces: `Model.System`, `Executables`, `Tiers`, `Modules`, `Groups`가 채워진다. `Module.Path`·`Group.Path`는 비어 있으면 ID.

- [ ] **Step 1: 실패하는 테스트**

`load_test.go`에 추가:

```go
const l2Sample = `
system:
  id: tolchain
  summary: 블록체인
  actors:
    game_server: { summary: "게임" }
  interfaces:
    json_rpc: { protocol: jsonrpc2-http, port: 8545, used_by: [game_server] }
    events:   { protocol: sse, path: /events, spec: docs/events.yaml }
executables:
  node: { exposes: [json_rpc, events], composes: [core, vm, tx_modules] }
  genkey: { summary: 키 생성, composes: [] }
tiers: [foundation, domain, execution]
modules:
  crypto: { tier: foundation }
  core:   { tier: domain, depends_on: [crypto], path: pkg/core }
  vm:     { tier: execution, depends_on: [core], implements: [json_rpc, events] }
groups:
  tx_modules:
    tier: execution
    kind: tx_handler
    path: vm/modules
    depends_on: [core, vm]
    rule: no_sibling_deps
    members:
      economy: { handles: [transfer], satisfies: [REQ-a] }
      reward:
        handles: [grant_reward]
        exceptions:
          - depends_on: asset
            reason: "스키마 재사용"
`

func TestDecodeL2(t *testing.T) {
	m, ds := loadOne(t, l2Sample)
	wantClean(t, ds)
	if m.System.ID != "tolchain" || len(m.System.Actors) != 1 || len(m.System.Interfaces) != 2 {
		t.Fatalf("system %+v", m.System)
	}
	rpc := m.System.Interfaces[0]
	if rpc.Port != 8545 || rpc.UsedBy[0].Name != "game_server" || m.System.Interfaces[1].Spec != "docs/events.yaml" {
		t.Fatalf("interfaces %+v %+v", rpc, m.System.Interfaces[1])
	}
	if len(m.Executables) != 2 || len(m.Executables[0].Composes) != 3 || m.Executables[1].Composes == nil {
		t.Fatalf("executables %+v", m.Executables)
	}
	if len(m.Tiers) != 3 || m.Tiers[2].Name != "execution" {
		t.Fatalf("tiers %+v", m.Tiers)
	}
	core := m.Modules[1]
	if core.ID != "core" || core.Tier.Name != "domain" || core.Path != "pkg/core" || core.DependsOn[0].Name != "crypto" {
		t.Fatalf("core %+v", core)
	}
	if m.Modules[0].Path != "crypto" {
		t.Fatalf("default path %q", m.Modules[0].Path)
	}
	g := m.Groups[0]
	if g.Kind.Name != "tx_handler" || g.Rule != "no_sibling_deps" || g.Tier.Name != "execution" || len(g.Members) != 2 {
		t.Fatalf("group %+v", g)
	}
	reward := g.Members[1]
	if reward.Group != g || reward.Handles[0].Name != "grant_reward" || len(reward.Exceptions) != 1 {
		t.Fatalf("member %+v", reward)
	}
	if reward.Exceptions[0].DependsOn.Name != "asset" || reward.Exceptions[0].Reason != "스키마 재사용" {
		t.Fatalf("exception %+v", reward.Exceptions[0])
	}
	if g.Members[0].Satisfies[0].Name != "REQ-a" {
		t.Fatal("member satisfies")
	}
}

func TestDecodeL2Errors(t *testing.T) {
	_, ds := loadOne(t, `
modules:
  core: { summary: x }
  vm: { tier: a, tierr: b }
groups:
  g: { kind: k, rule: whatever, members: { m: { handlez: [] } } }
  h: { members: {} }
`)
	wantDiag(t, ds, Error, `"tier"가 필요하다`)
	wantDiag(t, ds, Error, `알 수 없는 키 "tierr"`)
	wantDiag(t, ds, Error, "rule은 no_sibling_deps")
	wantDiag(t, ds, Error, `알 수 없는 키 "handlez"`)
	wantDiag(t, ds, Error, `"kind"가 필요하다`)
}

func TestSecondSystemAndTiers(t *testing.T) {
	_, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\nsystem: { id: a }\ntiers: [x]\n",
		"b.yaml": "version: 0.3\nsystem: { id: b }\ntiers: [y]\n",
	})
	wantDiag(t, ds, Error, "system은 한 파일에만")
	wantDiag(t, ds, Error, "tiers는 한 파일에만")
}

func TestExceptionNeedsDependsOn(t *testing.T) {
	_, ds := loadOne(t, "groups:\n  g:\n    kind: k\n    members:\n      m:\n        exceptions:\n          - reason: x\n")
	wantDiag(t, ds, Error, `"depends_on"가 필요하다`)
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/model/ -run 'TestDecodeL2|TestSecondSystem|TestException'`
Expected: FAIL (nil System)

- [ ] **Step 3: 구현**

`decode_layers.go`의 빈 함수 다섯 개를 아래로 바꾼다:

```go
func (mg *merger) system(n *yaml.Node) {
	if n == nil {
		return
	}
	d := mg.d
	if !d.mapping(n, "system") {
		return
	}
	if mg.m.System != nil {
		d.diags.Errorf(n.Pos, "system은 한 파일에만 있어야 한다 (처음: %s)", mg.m.System.Pos)
		return
	}
	d.allow(n, "id", "summary", "actors", "interfaces")
	s := &System{Meta: Meta{ID: d.str(n, "id"), Pos: n.Pos, Summary: d.str(n, "summary")}}
	if s.ID == "" {
		d.diags.Errorf(n.Pos, "system에 id가 필요하다")
	}
	d.idMap(n.Get("actors"), "actors", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies")
		if mg.unique("액터", id, pos) {
			s.Actors = append(s.Actors, &Actor{Meta: d.meta(id, pos, v)})
		}
	})
	d.idMap(n.Get("interfaces"), "interfaces", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "protocol", "port", "path", "spec", "used_by")
		if !mg.unique("인터페이스", id, pos) {
			return
		}
		it := &Interface{
			Meta:     d.meta(id, pos, v),
			Protocol: d.str(v, "protocol"),
			Path:     d.str(v, "path"),
			Spec:     d.str(v, "spec"),
			UsedBy:   d.refs(v, "used_by"),
		}
		it.Port, _ = d.intv(v, "port")
		s.Interfaces = append(s.Interfaces, it)
	})
	mg.m.System = s
}

func (mg *merger) executables(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "executables", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "exposes", "composes")
		if !mg.unique("실행 단위", id, pos) {
			return
		}
		mg.m.Executables = append(mg.m.Executables, &Executable{
			Meta:     d.meta(id, pos, v),
			Exposes:  d.refs(v, "exposes"),
			Composes: d.refs(v, "composes"),
		})
	})
}

func (mg *merger) tiers(n *yaml.Node) {
	if n == nil {
		return
	}
	d := mg.d
	if prev, ok := mg.once["tiers"]; ok {
		d.diags.Errorf(n.Pos, "tiers는 한 파일에만 있어야 한다 (처음: %s)", prev)
		return
	}
	mg.once["tiers"] = n.Pos
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "tiers는 목록이어야 한다")
		return
	}
	for _, it := range n.Items {
		if it.Kind != yaml.Scalar {
			d.diags.Errorf(it.Pos, "tier 이름은 값 하나여야 한다")
			continue
		}
		if mg.unique("tier", it.Value, it.Pos) {
			mg.m.Tiers = append(mg.m.Tiers, Ref{Name: it.Value, Pos: it.Pos})
		}
	}
}

func (mg *merger) modules(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "modules", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "tier", "path", "depends_on", "implements")
		if !mg.unique("모듈", id, pos) {
			return
		}
		m := &Module{
			Meta:       d.meta(id, pos, v),
			Path:       d.str(v, "path"),
			DependsOn:  d.refs(v, "depends_on"),
			Implements: d.refs(v, "implements"),
		}
		if d.need(v, "tier") {
			m.Tier = d.ref(v, "tier")
		}
		if m.Path == "" {
			m.Path = id
		}
		mg.m.Modules = append(mg.m.Modules, m)
	})
}

func (mg *merger) groups(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "groups", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "tier", "kind", "path", "depends_on", "rule", "members")
		if !mg.unique("그룹", id, pos) {
			return
		}
		g := &Group{
			Meta:      d.meta(id, pos, v),
			Tier:      d.ref(v, "tier"),
			Path:      d.str(v, "path"),
			DependsOn: d.refs(v, "depends_on"),
			Rule:      d.str(v, "rule"),
		}
		if d.need(v, "kind") {
			g.Kind = d.ref(v, "kind")
		}
		if g.Path == "" {
			g.Path = id
		}
		if g.Rule != "" && g.Rule != "no_sibling_deps" {
			d.diags.Errorf(v.Get("rule").Pos, "rule은 no_sibling_deps만 지원한다")
		}
		if d.need(v, "members") {
			d.idMap(v.Get("members"), "members", func(mid string, mpos yaml.Pos, mv *yaml.Node) {
				d.allow(mv, "summary", "satisfies", "handles", "exceptions")
				if !mg.unique("멤버", mid, mpos) {
					return
				}
				mem := &Member{Meta: d.meta(mid, mpos, mv), Group: g, Handles: d.refs(mv, "handles")}
				mem.Exceptions = mg.exceptions(mv.Get("exceptions"))
				g.Members = append(g.Members, mem)
			})
		}
		mg.m.Groups = append(mg.m.Groups, g)
	})
}

func (mg *merger) exceptions(n *yaml.Node) []Exception {
	if n == nil || n.IsNull() {
		return nil
	}
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "exceptions는 목록이어야 한다")
		return nil
	}
	var out []Exception
	for _, it := range n.Items {
		if !d.mapping(it, "예외") {
			continue
		}
		d.allow(it, "depends_on", "reason")
		if !d.need(it, "depends_on") {
			continue
		}
		out = append(out, Exception{DependsOn: d.ref(it, "depends_on"), Reason: d.str(it, "reason")})
	}
	return out
}
```

- [ ] **Step 4: 통과 확인**

Run: `gofmt -l ./internal/ && go test ./internal/model/`
Expected: PASS

- [ ] **Step 5: 커밋**

```bash
git add internal/model/
git commit -m "feat: 모델 L0·L1·L2 디코드"
```

---

### Task 7: L3 디코드

**Files:**
- Modify: `internal/model/decode_l3.go`
- Test: `internal/model/load_test.go`

**Interfaces:**
- Produces: `Model.Enums`, `Consts`, `Errors`, `Types`, `Contracts`, `HandlerKinds`. `Const.Type` 기본 `int`. `Field`는 `Type`과 `RefType` 중 하나만 채운다.

- [ ] **Step 1: 실패하는 테스트**

```go
const l3Sample = `
enums:
  tx_type:
    in: core
    go: { type: TxType, prefix: Tx }
    values: [transfer, mint_asset]
    export: [ts]
consts:
  max_block_txs: { in: config, type: int, value: 500, go: { name: DefaultMaxBlockTxs } }
  max_tx_age:    { in: config, type: duration, value: 1h }
  page_size:     { in: httputil, value: 10 }
errors:
  ErrNotFound: { in: core, message: "not found", http: 404 }
  ErrOverflow: { in: core, message: "overflow" }
types:
  transfer_payload:
    in: core
    gen: false
    fields: { to: address, amount: uint64 }
  post:
    in: model
    fields:
      title:  { type: string, max: post_title_max }
      tags:   { ref: tag, many: true }
      series: { ref: series, optional: true }
      note:   { type: string, max: 80 }
contracts:
  post_service:
    in: post
    go: { name: Service }
    methods:
      get_post: { params: { post_id: uint, user_id: "*uint" }, returns: [post], errors: [ErrNotFound] }
      list:     { params: {}, returns: [] }
handler_kinds:
  tx_handler:
    in: vm
    go: { name: Handler, register: Register }
    params: { ctx: "*Context", payload: json.RawMessage }
    returns: [error]
    key: tx_type
`

func TestDecodeL3(t *testing.T) {
	m, ds := loadOne(t, l3Sample)
	wantClean(t, ds)
	e := m.Enums[0]
	if e.ID != "tx_type" || e.In.Name != "core" || !e.Gen || e.Go.Prefix != "Tx" || len(e.Values) != 2 || e.Export[0] != "ts" {
		t.Fatalf("enum %+v", e)
	}
	if m.Consts[0].Go.Name != "DefaultMaxBlockTxs" || m.Consts[0].Value != "500" || m.Consts[1].Type != "duration" || m.Consts[2].Type != "int" {
		t.Fatalf("consts %+v %+v %+v", m.Consts[0], m.Consts[1], m.Consts[2])
	}
	if m.Errors[0].HTTP != 404 || m.Errors[1].HTTP != 0 || m.Errors[1].Message != "overflow" {
		t.Fatalf("errors %+v %+v", m.Errors[0], m.Errors[1])
	}
	tp := m.Types[0]
	if tp.Gen || len(tp.Fields) != 2 || tp.Fields[1].Type.Name != "uint64" || tp.Fields[1].RefType.Name != "" {
		t.Fatalf("type %+v", tp)
	}
	post := m.Types[1]
	if post.Fields[0].Max.Name != "post_title_max" || !post.Fields[1].Many || post.Fields[1].RefType.Name != "tag" || !post.Fields[2].Optional || post.Fields[3].Max.Name != "80" {
		t.Fatalf("post fields %+v", post.Fields)
	}
	c := m.Contracts[0]
	if c.Go.Name != "Service" || len(c.Methods) != 2 || c.Methods[0].Params[1].Type.Name != "*uint" || !c.Methods[0].HasErrors || c.Methods[1].HasErrors {
		t.Fatalf("contract %+v", c)
	}
	hk := m.HandlerKinds[0]
	if hk.Key.Name != "tx_type" || hk.Go.Register != "Register" || hk.Params[0].Type.Name != "*Context" || hk.Returns[0].Name != "error" {
		t.Fatalf("handler kind %+v", hk)
	}
}

func TestDecodeL3Errors(t *testing.T) {
	_, ds := loadOne(t, `
enums:
  e: { values: [a] }
consts:
  c: { in: m, type: float, value: 1 }
  d: { in: m, value: abc }
errors:
  E: { in: m }
types:
  t: { in: m, fields: { a: { type: x, ref: y }, b: { many: true } } }
contracts:
  k: { in: m }
handler_kinds:
  h: { in: m, params: {} }
`)
	wantDiag(t, ds, Error, `"in"가 필요하다`)
	wantDiag(t, ds, Error, "type은 int, string, duration")
	wantDiag(t, ds, Error, "value는 정수여야 한다")
	wantDiag(t, ds, Error, `"message"가 필요하다`)
	wantDiag(t, ds, Error, "type과 ref 중 하나만")
	wantDiag(t, ds, Error, "type이나 ref가 필요하다")
	wantDiag(t, ds, Error, `"methods"가 필요하다`)
	wantDiag(t, ds, Error, `"key"가 필요하다`)
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/model/ -run TestDecodeL3`
Expected: FAIL (index out of range)

- [ ] **Step 3: 구현**

`internal/model/decode_l3.go` 전체:

```go
package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) enums(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "enums", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "values", "go", "export")
		if !mg.unique("enum", id, pos) {
			return
		}
		e := &Enum{Owned: d.owned(id, pos, v), Go: d.goOpts(v), Export: d.strs(v, "export")}
		if d.need(v, "values") {
			e.Values = d.refs(v, "values")
		}
		mg.m.Enums = append(mg.m.Enums, e)
	})
}

func (mg *merger) consts(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "consts", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "type", "value", "go", "export")
		if !mg.unique("상수", id, pos) {
			return
		}
		c := &Const{Owned: d.owned(id, pos, v), Type: d.str(v, "type"), Go: d.goOpts(v), Export: d.strs(v, "export")}
		if c.Type == "" {
			c.Type = "int"
		}
		switch c.Type {
		case "int", "string", "duration":
		default:
			d.diags.Errorf(v.Get("type").Pos, "type은 int, string, duration 중 하나여야 한다")
		}
		if d.need(v, "value") {
			if val := d.scalar(v, "value"); val != nil {
				c.Value, c.ValuePos = val.Value, val.Pos
				if _, ok := val.Int(); c.Type == "int" && !ok {
					d.diags.Errorf(val.Pos, "value는 정수여야 한다")
				}
			}
		}
		mg.m.Consts = append(mg.m.Consts, c)
	})
}

func (mg *merger) errors(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "errors", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "message", "http")
		if !mg.unique("에러", id, pos) {
			return
		}
		e := &ErrorDef{Owned: d.owned(id, pos, v)}
		if d.need(v, "message") {
			e.Message = d.str(v, "message")
		}
		e.HTTP, _ = d.intv(v, "http")
		mg.m.Errors = append(mg.m.Errors, e)
	})
}

func (mg *merger) types(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "types", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "fields", "go")
		if !mg.unique("타입", id, pos) {
			return
		}
		t := &Type{Owned: d.owned(id, pos, v), Go: d.goOpts(v)}
		if d.need(v, "fields") {
			t.Fields = mg.fields(v.Get("fields"))
		}
		mg.m.Types = append(mg.m.Types, t)
	})
}

func (mg *merger) fields(n *yaml.Node) []*Field {
	d := mg.d
	if !d.mapping(n, "fields") {
		return nil
	}
	var out []*Field
	for _, e := range n.Entries {
		f := &Field{Name: e.Key, Pos: e.Pos}
		switch e.Value.Kind {
		case yaml.Scalar:
			f.Type = Ref{Name: e.Value.Value, Pos: e.Value.Pos}
		case yaml.Map:
			v := e.Value
			d.allow(v, "type", "ref", "many", "optional", "max")
			f.Type, f.RefType = d.ref(v, "type"), d.ref(v, "ref")
			switch {
			case f.Type.Name != "" && f.RefType.Name != "":
				d.diags.Errorf(e.Pos, "필드 %q에는 type과 ref 중 하나만 쓴다", e.Key)
			case f.Type.Name == "" && f.RefType.Name == "":
				d.diags.Errorf(e.Pos, "필드 %q에 type이나 ref가 필요하다", e.Key)
			}
			f.Many, f.Optional = d.boolv(v, "many", false), d.boolv(v, "optional", false)
			f.Max = d.ref(v, "max")
		default:
			d.diags.Errorf(e.Pos, "필드 %q는 타입 이름이나 맵이어야 한다", e.Key)
			continue
		}
		out = append(out, f)
	}
	return out
}

func (mg *merger) params(n *yaml.Node) []*Param {
	d := mg.d
	p := n.Get("params")
	if p == nil || p.IsNull() || !d.mapping(p, "params") {
		return nil
	}
	var out []*Param
	for _, e := range p.Entries {
		if e.Value.Kind != yaml.Scalar {
			d.diags.Errorf(e.Pos, "파라미터 %q는 타입 이름이어야 한다", e.Key)
			continue
		}
		out = append(out, &Param{Name: e.Key, Pos: e.Pos, Type: Ref{Name: e.Value.Value, Pos: e.Value.Pos}})
	}
	return out
}

func (mg *merger) contracts(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "contracts", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "go", "methods")
		if !mg.unique("계약", id, pos) {
			return
		}
		c := &Contract{Owned: d.owned(id, pos, v), Go: d.goOpts(v)}
		if d.need(v, "methods") {
			d.idMap(v.Get("methods"), "methods", func(name string, mpos yaml.Pos, mv *yaml.Node) {
				d.allow(mv, "params", "returns", "errors")
				c.Methods = append(c.Methods, &Method{
					Name:      name,
					Pos:       mpos,
					Params:    mg.params(mv),
					Returns:   d.refs(mv, "returns"),
					Errors:    d.refs(mv, "errors"),
					HasErrors: mv.Get("errors") != nil,
				})
			})
		}
		mg.m.Contracts = append(mg.m.Contracts, c)
	})
}

func (mg *merger) handlerKinds(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "handler_kinds", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "params", "returns", "key", "go")
		if !mg.unique("핸들러 종류", id, pos) {
			return
		}
		hk := &HandlerKind{Owned: d.owned(id, pos, v), Params: mg.params(v), Returns: d.refs(v, "returns"), Go: d.goOpts(v)}
		if d.need(v, "key") {
			hk.Key = d.ref(v, "key")
		}
		mg.m.HandlerKinds = append(mg.m.HandlerKinds, hk)
	})
}
```

- [ ] **Step 4: 통과 확인**

Run: `gofmt -l ./internal/ && go test ./internal/model/`
Expected: PASS

- [ ] **Step 5: 커밋**

```bash
git add internal/model/
git commit -m "feat: 모델 L3 디코드"
```

---

### Task 8: L4 디코드, 색인, 샘플 설계 파일

**Files:**
- Modify: `internal/model/decode_l4.go`
- Create: `internal/model/index.go`, `internal/model/testdata/tolchain/design/tolchain.yaml`, `internal/model/testdata/tolelog/design/tolelog.yaml`
- Test: `internal/model/load_test.go`

**Interfaces:**
- Produces:
  - `Model.Funcs`
  - 조회: `Module(id)`, `Group(id)`, `Member(id)`, `Enum(id)`, `Const(id)`, `Error(id)`, `Type(id)`, `Contract(id)`, `HandlerKind(id)`, `Func(id)`, `Requirement(id)`, `Interface(id)`, `Actor(id)`, `Executable(id)`. 없으면 nil.
  - `TierIndex(name string) int`(없으면 -1)
  - `type Owner struct{ ID, Path, Tier string; Module *Module; Member *Member }`, `func (m *Model) Owner(id string) (Owner, bool)`
  - `func (m *Model) CanDepend(from, to string) bool`
  - `type TypeKind int` (`BuiltinType`, `EnumType`, `StructType`, `ExternalType`, `UnknownType`), `var Builtins map[string]string`, `func SplitType(t string) (prefix, base string)`, `func (m *Model) ResolveType(t string) (TypeKind, string)`
  - `func (mem *Member) Path() string` → 그룹 path + `/` + 멤버 ID

- [ ] **Step 1: 실패하는 테스트**

```go
func TestDecodeL4(t *testing.T) {
	m, ds := loadOne(t, `
funcs:
  handle_transfer:
    in: economy
    kind: tx_handler
    handles: transfer
    errors: [ErrInsufficientBalance]
    cases:
      - { name: "금액 0", given: { amount: 0 }, error: invalid }
      - { name: "정상", given: { balance: 10, req: { title: "새 제목" } }, expect: { from: 7 } }
  update_post:
    in: post
    of: post_service.update_post
    errors: []
    cases: []
`)
	wantClean(t, ds)
	f := m.Funcs[0]
	if f.Kind.Name != "tx_handler" || f.Handles.Name != "transfer" || len(f.Cases) != 2 {
		t.Fatalf("%+v", f)
	}
	if f.Cases[0].Error.Name != "invalid" || f.Cases[0].Expect != nil || f.Cases[0].Given.Get("amount").Value != "0" {
		t.Fatalf("case0 %+v", f.Cases[0])
	}
	if f.Cases[1].Error.Name != "" || f.Cases[1].Expect.Get("from").Value != "7" {
		t.Fatalf("case1 %+v", f.Cases[1])
	}
	g := m.Funcs[1]
	if g.Of.Name != "post_service.update_post" || g.Errors == nil || len(g.Errors) != 0 {
		t.Fatalf("%+v", g)
	}
}

func TestDecodeL4Errors(t *testing.T) {
	_, ds := loadOne(t, `
funcs:
  a: { in: m, kind: k, handles: h, of: c.m }
  b: { in: m }
  c: { in: m, kind: k }
  d:
    in: m
    of: c.m
    cases:
      - { given: {} }
      - { name: x, given: {}, expect: {}, error: E }
      - { name: y, given: {} }
      - { name: z, given: [1], error: E }
      - { name: w, given: { a: [1] }, error: E }
`)
	wantDiag(t, ds, Error, "kind+handles와 of 중 하나만")
	wantDiag(t, ds, Error, "kind+handles 또는 of가 필요하다")
	wantDiag(t, ds, Error, `"handles"가 필요하다`)
	wantDiag(t, ds, Error, `"name"가 필요하다`)
	wantDiag(t, ds, Error, "expect와 error 중 하나만")
	wantDiag(t, ds, Error, "expect나 error가 필요하다")
	wantDiag(t, ds, Error, "given은 맵이어야 한다")
	wantDiag(t, ds, Error, "given의 값은 값 하나 또는 맵이어야 한다")
}

func TestIndexAndOwner(t *testing.T) {
	m, ds := loadOne(t, l2Sample+`
enums:
  tx_type: { in: core, values: [transfer] }
types:
  payload: { in: core, fields: { to: address } }
`)
	wantClean(t, ds)
	if m.Module("core") == nil || m.Module("nope") != nil || m.Member("reward") == nil || m.Group("tx_modules") == nil {
		t.Fatal("lookup")
	}
	if m.TierIndex("execution") != 2 || m.TierIndex("x") != -1 {
		t.Fatal("tier index")
	}
	o, ok := m.Owner("reward")
	if !ok || o.Path != "vm/modules/reward" || o.Tier != "execution" || o.Member == nil {
		t.Fatalf("owner %+v", o)
	}
	o, _ = m.Owner("core")
	if o.Path != "pkg/core" || o.Tier != "domain" || o.Module == nil {
		t.Fatalf("owner %+v", o)
	}
	if _, ok := m.Owner("tx_modules"); ok {
		t.Fatal("group is not an owner")
	}
	cases := []struct {
		from, to string
		want     bool
	}{
		{"core", "core", true}, {"core", "crypto", true}, {"crypto", "core", false},
		{"reward", "core", true}, {"reward", "asset", true}, {"economy", "asset", false},
		{"vm", "economy", false}, {"tx_modules", "vm", true},
	}
	for _, c := range cases {
		if got := m.CanDepend(c.from, c.to); got != c.want {
			t.Errorf("CanDepend(%s, %s) = %v", c.from, c.to, got)
		}
	}
	if k, base := m.ResolveType("*uint"); k != BuiltinType || base != "uint" {
		t.Fatal("builtin")
	}
	if k, _ := m.ResolveType("[]tx_type"); k != EnumType {
		t.Fatal("enum")
	}
	if k, _ := m.ResolveType("payload"); k != StructType {
		t.Fatal("struct")
	}
	if k, _ := m.ResolveType("json.RawMessage"); k != ExternalType {
		t.Fatal("external dotted")
	}
	if k, _ := m.ResolveType("*Context"); k != ExternalType {
		t.Fatal("external upper")
	}
	if k, _ := m.ResolveType("nothing"); k != UnknownType {
		t.Fatal("unknown")
	}
}

func TestMemberViaGroupDependency(t *testing.T) {
	m, ds := loadOne(t, l2Sample+"\nmodules_extra: {}\n")
	_ = m
	wantDiag(t, ds, Error, `알 수 없는 키 "modules_extra"`)
	m, ds = loadOne(t, strings.Replace(l2Sample, "vm:     { tier: execution, depends_on: [core]", "vm:     { tier: execution, depends_on: [core, tx_modules]", 1))
	wantClean(t, ds)
	if !m.CanDepend("vm", "economy") {
		t.Fatal("depending on a group allows its members")
	}
}

func TestNamespaceCollisions(t *testing.T) {
	_, ds := loadOne(t, `
modules:
  events: { tier: a }
  economy: { tier: a }
groups:
  g: { kind: k, members: { economy: {} } }
system:
  id: s
  interfaces:
    events: {}
enums:
  post: { in: m, values: [a] }
types:
  post: { in: m, fields: {} }
`)
	wantDiag(t, ds, Error, `멤버 "economy"가 모듈`)
	wantDiag(t, ds, Error, `타입 "post"가 enum`)
	if find(ds, Error, `"events"`) != nil {
		t.Fatal("module and interface may share an ID")
	}
}

func TestSamplesLoadClean(t *testing.T) {
	for _, s := range []string{"tolchain", "tolelog"} {
		m, ds := Load(filepath.Join("testdata", s, "design"))
		if len(ds) != 0 {
			t.Fatalf("%s:\n%s", s, join(ds))
		}
		if m.GoModule == "" || len(m.Modules) == 0 {
			t.Fatalf("%s: empty model", s)
		}
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/model/`
Expected: FAIL (`undefined: m.Module` 등)

- [ ] **Step 3: L4 디코드**

`internal/model/decode_l4.go` 전체:

```go
package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) funcs(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "funcs", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "kind", "handles", "of", "errors", "cases")
		if !mg.unique("함수", id, pos) {
			return
		}
		f := &Func{Owned: d.owned(id, pos, v), Kind: d.ref(v, "kind"), Handles: d.ref(v, "handles"), Of: d.ref(v, "of"), Errors: d.refs(v, "errors")}
		hasKind, hasOf := f.Kind.Name != "" || f.Handles.Name != "", f.Of.Name != ""
		switch {
		case hasKind && hasOf:
			d.diags.Errorf(pos, "함수 %q에는 kind+handles와 of 중 하나만 쓴다", id)
		case !hasKind && !hasOf:
			d.diags.Errorf(pos, "함수 %q에 kind+handles 또는 of가 필요하다", id)
		case hasKind:
			d.need(v, "kind")
			d.need(v, "handles")
		}
		f.Cases = mg.cases(v.Get("cases"))
		mg.m.Funcs = append(mg.m.Funcs, f)
	})
}

func (mg *merger) cases(n *yaml.Node) []*Case {
	if n == nil || n.IsNull() {
		return nil
	}
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "cases는 목록이어야 한다")
		return nil
	}
	var out []*Case
	for _, it := range n.Items {
		if !d.mapping(it, "케이스") {
			continue
		}
		d.allow(it, "name", "given", "expect", "error")
		c := &Case{Pos: it.Pos, Error: d.ref(it, "error")}
		if d.need(it, "name") {
			c.Name = d.str(it, "name")
		}
		if d.need(it, "given") {
			c.Given = mg.caseMap(it.Get("given"), "given")
		}
		if e := it.Get("expect"); e != nil {
			c.Expect = mg.caseMap(e, "expect")
		}
		switch {
		case c.Expect != nil && c.Error.Name != "":
			d.diags.Errorf(it.Pos, "케이스에는 expect와 error 중 하나만 쓴다")
		case c.Expect == nil && c.Error.Name == "":
			d.diags.Errorf(it.Pos, "케이스에 expect나 error가 필요하다")
		}
		out = append(out, c)
	}
	return out
}

// caseMap은 given·expect가 스칼라 또는 맵만 값으로 갖는 맵인지 확인한다.
func (mg *merger) caseMap(n *yaml.Node, what string) *yaml.Node {
	d := mg.d
	if n.Kind != yaml.Map {
		d.diags.Errorf(n.Pos, "%s은 맵이어야 한다", what)
		return nil
	}
	for _, e := range n.Entries {
		switch e.Value.Kind {
		case yaml.Scalar:
		case yaml.Map:
			mg.caseMap(e.Value, what)
		default:
			d.diags.Errorf(e.Value.Pos, "%s의 값은 값 하나 또는 맵이어야 한다", what)
		}
	}
	return n
}
```

- [ ] **Step 4: 색인**

`internal/model/index.go`:

```go
package model

import "strings"

func (m *Model) Requirement(id string) *Requirement {
	for _, r := range m.Requirements {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (m *Model) Actor(id string) *Actor {
	if m.System == nil {
		return nil
	}
	for _, a := range m.System.Actors {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func (m *Model) Interface(id string) *Interface {
	if m.System == nil {
		return nil
	}
	for _, i := range m.System.Interfaces {
		if i.ID == id {
			return i
		}
	}
	return nil
}

func (m *Model) Executable(id string) *Executable {
	for _, e := range m.Executables {
		if e.ID == id {
			return e
		}
	}
	return nil
}

func (m *Model) Module(id string) *Module {
	for _, x := range m.Modules {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Group(id string) *Group {
	for _, g := range m.Groups {
		if g.ID == id {
			return g
		}
	}
	return nil
}

func (m *Model) Member(id string) *Member {
	for _, g := range m.Groups {
		for _, mem := range g.Members {
			if mem.ID == id {
				return mem
			}
		}
	}
	return nil
}

func (m *Model) Enum(id string) *Enum {
	for _, x := range m.Enums {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Const(id string) *Const {
	for _, x := range m.Consts {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Error(id string) *ErrorDef {
	for _, x := range m.Errors {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Type(id string) *Type {
	for _, x := range m.Types {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Contract(id string) *Contract {
	for _, x := range m.Contracts {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) HandlerKind(id string) *HandlerKind {
	for _, x := range m.HandlerKinds {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Func(id string) *Func {
	for _, x := range m.Funcs {
		if x.ID == id {
			return x
		}
	}
	return nil
}

// TierIndex는 tiers 목록에서의 순서다. 작을수록 아래 tier. 없으면 -1.
func (m *Model) TierIndex(name string) int {
	for i, t := range m.Tiers {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (mem *Member) Path() string { return mem.Group.Path + "/" + mem.ID }

// Owner는 L3·L4 노드가 속하는 모듈 또는 그룹 멤버다.
type Owner struct {
	ID     string
	Path   string
	Tier   string
	Module *Module // 모듈이면 채워진다
	Member *Member // 멤버면 채워진다
}

func (m *Model) Owner(id string) (Owner, bool) {
	if mod := m.Module(id); mod != nil {
		return Owner{ID: id, Path: mod.Path, Tier: mod.Tier.Name, Module: mod}, true
	}
	if mem := m.Member(id); mem != nil {
		return Owner{ID: id, Path: mem.Path(), Tier: mem.Group.Tier.Name, Member: mem}, true
	}
	return Owner{}, false
}

func hasRef(rs []Ref, name string) bool {
	for _, r := range rs {
		if r.Name == name {
			return true
		}
	}
	return false
}

// CanDepend는 설계상 from이 to를 import해도 되는지다. from·to는 모듈, 그룹, 멤버 ID.
// 그룹에 의존하면 그 멤버에도 의존할 수 있다.
func (m *Model) CanDepend(from, to string) bool {
	if from == to {
		return true
	}
	var deps []Ref
	switch {
	case m.Module(from) != nil:
		deps = m.Module(from).DependsOn
	case m.Group(from) != nil:
		deps = m.Group(from).DependsOn
	case m.Member(from) != nil:
		mem := m.Member(from)
		deps = mem.Group.DependsOn
		for _, e := range mem.Exceptions {
			if e.DependsOn.Name == to {
				return true
			}
		}
	default:
		return false
	}
	if hasRef(deps, to) {
		return true
	}
	if mem := m.Member(to); mem != nil && hasRef(deps, mem.Group.ID) {
		return true
	}
	return false
}

type TypeKind int

const (
	UnknownType TypeKind = iota
	BuiltinType
	EnumType
	StructType
	ExternalType
)

// Builtins는 설계 타입 이름과 Go 타입의 대응이다.
var Builtins = map[string]string{
	"string": "string", "int": "int", "uint": "uint", "int64": "int64", "uint64": "uint64",
	"bool": "bool", "float64": "float64", "duration": "time.Duration", "time": "time.Time",
	"address": "string", "bytes": "[]byte", "error": "error",
}

// SplitType은 "*[]post"를 ("*[]", "post")로 나눈다.
func SplitType(t string) (prefix, base string) {
	base = t
	for {
		switch {
		case strings.HasPrefix(base, "*"):
			prefix, base = prefix+"*", base[1:]
		case strings.HasPrefix(base, "[]"):
			prefix, base = prefix+"[]", base[2:]
		default:
			return prefix, base
		}
	}
}

// ResolveType은 타입 이름의 종류와 접두사를 뗀 이름을 준다.
// 대문자로 시작하거나 '.'이 있으면 외부 타입으로 본다.
func (m *Model) ResolveType(t string) (TypeKind, string) {
	_, base := SplitType(t)
	switch {
	case Builtins[base] != "":
		return BuiltinType, base
	case m.Enum(base) != nil:
		return EnumType, base
	case m.Type(base) != nil:
		return StructType, base
	case base != "" && (strings.Contains(base, ".") || base[0] >= 'A' && base[0] <= 'Z'):
		return ExternalType, base
	}
	return UnknownType, base
}
```

- [ ] **Step 5: 샘플 설계 파일**

프로토타입 `SAMPLES`에서 두 설계 파일을 뽑는다. 줄 범위는 `prototype/daemok-v2.html`의 tolchain 본문(272~394줄)과 tolelog 본문(404~526줄)이다.

```bash
mkdir -p internal/model/testdata/tolchain/design internal/model/testdata/tolelog/design
{ echo 'version: 0.3'; sed -n '272,394p' prototype/daemok-v2.html; } > internal/model/testdata/tolchain/design/tolchain.yaml
{ echo 'version: 0.3'; sed -n '404,526p' prototype/daemok-v2.html; } > internal/model/testdata/tolelog/design/tolelog.yaml
head -3 internal/model/testdata/tolchain/design/tolchain.yaml
tail -2 internal/model/testdata/tolelog/design/tolelog.yaml
```

첫 줄이 `version: 0.3`, 둘째 줄이 `go_module: ...`이고 마지막 줄이 백틱 없이 케이스 행으로 끝나야 한다. 다르면 줄 번호를 `grep -n 'text: \`\|^\`}' prototype/daemok-v2.html`로 다시 잡는다.

tolchain 파일의 `transfer_payload`에 `gen: false`를 넣는다:

```bash
sed -i 's/^  transfer_payload:$/  transfer_payload:\n    gen: false/' internal/model/testdata/tolchain/design/tolchain.yaml
grep -n -A3 'transfer_payload:' internal/model/testdata/tolchain/design/tolchain.yaml
```

결과가 아래여야 한다:

```yaml
  transfer_payload:
    gen: false
    in: core
    fields: { to: address, amount: uint64 }
```

- [ ] **Step 6: 통과 확인**

Run: `gofmt -l ./internal/ && go vet ./internal/... && go test ./internal/model/`
Expected: PASS. `TestSamplesLoadClean`이 실패하면 진단 메시지가 가리키는 샘플 줄을 보고, 샘플이 형식을 벗어난 것인지 디코더가 형식을 잘못 읽는 것인지 `docs/design/01-model.md`로 판정한다. 샘플 쪽 문제면 샘플을 고치고 이 계획 파일의 Step 5에 그 수정도 적는다.

- [ ] **Step 7: 커밋**

```bash
git add internal/model/
git commit -m "feat: 모델 L4 디코드, 색인, 샘플 설계 파일"
```

---

### Task 9: 검사 골격과 참조 해석

**Files:**
- Create: `internal/check/check.go`, `internal/check/refs.go`, `internal/check/check_test.go`
- Modify: `internal/model/index.go`(`Contract.Method`, `Enum.HasValue` 추가)
- 나머지 검사 파일은 빈 함수로 만든다: `internal/check/l0.go`, `l2.go`, `l3.go`, `handlers.go`, `l4.go`, `req.go`

**Interfaces:**
- Consumes: `model.Load`, `model.Model`의 조회 함수, `model.Diagnostics`
- Produces:
  - `func Run(m *model.Model) model.Diagnostics`. 위치순 정렬.
  - `func (c *Contract) Method(name string) *Method`, `func (e *Enum) HasValue(name string) bool`(model 패키지)
  - 검사 내부 도우미: `checker.errorf/warnf/infof`, `checker.depsOf(id) []string`, `has(rs []model.Ref, name string) bool`

- [ ] **Step 1: 실패하는 테스트**

`internal/check/check_test.go`:

```go
package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

// run은 설계 텍스트 하나를 읽어 검사 진단을 준다. 모델 단계 오류가 있으면 실패다.
func run(t *testing.T, src string) model.Diagnostics {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.3\n"+src), 0o644); err != nil {
		t.Fatal(err)
	}
	m, ds := model.Load(dir)
	if ds.HasErrors() {
		t.Fatalf("model errors:\n%s", join(ds))
	}
	return Run(m)
}

func join(ds model.Diagnostics) string {
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func want(t *testing.T, ds model.Diagnostics, level model.Level, substr string) model.Diagnostic {
	t.Helper()
	for _, d := range ds {
		if d.Level == level && strings.Contains(d.Msg, substr) {
			return d
		}
	}
	t.Fatalf("no %v containing %q in:\n%s", level, substr, join(ds))
	return model.Diagnostic{}
}

func wantNone(t *testing.T, ds model.Diagnostics, level model.Level) {
	t.Helper()
	if n := ds.Count(level); n != 0 {
		t.Fatalf("want no %v, got %d:\n%s", level, n, join(ds))
	}
}

// base는 모든 검사를 통과하는 작은 설계다. 각 테스트가 한 곳씩 바꿔 위반을 만든다.
const base = `
requirements:
  REQ-a: { text: A }
system:
  id: s
  actors:
    game: {}
  interfaces:
    rpc: { protocol: http, used_by: [game] }
executables:
  node: { exposes: [rpc], composes: [core, vm, tx_modules, rpc_mod] }
tiers: [foundation, domain, execution, service]
modules:
  crypto:  { tier: foundation }
  core:    { tier: domain, depends_on: [crypto], satisfies: [REQ-a] }
  vm:      { tier: execution, depends_on: [core] }
  rpc_mod: { tier: service, depends_on: [core, vm], implements: [rpc] }
groups:
  tx_modules:
    tier: execution
    kind: tx_handler
    depends_on: [core, vm]
    rule: no_sibling_deps
    members:
      economy: { handles: [transfer] }
      asset:   { handles: [mint] }
enums:
  tx_type: { in: core, values: [transfer, mint] }
errors:
  ErrNotFound: { in: core, message: "not found", http: 404 }
  ErrLow:      { in: core, message: "low" }
types:
  payload: { in: core, fields: { to: address, kind: tx_type } }
contracts:
  store:
    in: core
    methods:
      get: { params: { id: uint }, returns: [payload], errors: [ErrNotFound] }
handler_kinds:
  tx_handler:
    in: vm
    params: { payload: payload }
    returns: [error]
    key: tx_type
funcs:
  handle_transfer:
    in: economy
    kind: tx_handler
    handles: transfer
    errors: [ErrLow]
    cases:
      - { name: low, given: { balance: 0 }, error: ErrLow }
      - { name: ok, given: { balance: 9 }, expect: { balance: 8 } }
`

func replace(t *testing.T, old, new string) string {
	t.Helper()
	if !strings.Contains(base, old) {
		t.Fatalf("base does not contain %q", old)
	}
	return strings.Replace(base, old, new, 1)
}

func TestBaseIsClean(t *testing.T) {
	ds := run(t, base)
	if len(ds) != 0 {
		t.Fatalf("base must be clean:\n%s", join(ds))
	}
}

func TestRefErrors(t *testing.T) {
	cases := []struct{ old, new, msg string }{
		{"satisfies: [REQ-a]", "satisfies: [REQ-x]", `없는 요구사항 "REQ-x"`},
		{"used_by: [game]", "used_by: [nobody]", `없는 액터 "nobody"`},
		{"exposes: [rpc]", "exposes: [grpc]", `없는 인터페이스 "grpc"`},
		{"composes: [core, vm, tx_modules, rpc_mod]", "composes: [core, ghost]", `없는 모듈 "ghost"`},
		{"tier: domain,", "tier: cloud,", `없는 tier "cloud"`},
		{"depends_on: [crypto]", "depends_on: [cryptoo]", `없는 모듈 "cryptoo"`},
		{"implements: [rpc]", "implements: [rest]", `없는 인터페이스 "rest"`},
		{"kind: tx_handler\n    depends_on", "kind: nope\n    depends_on", `없는 핸들러 종류 "nope"`},
		{"handles: [mint]", "handles: [burn]", `tx_type에 없는 값 "burn"`},
		{"in: vm\n    params", "in: nowhere\n    params", `"nowhere"가 없다`},
		{"to: address", "to: adress", `알 수 없는 타입 "adress"`},
		{"returns: [payload], errors: [ErrNotFound]", "returns: [payload], errors: [ErrGone]", `없는 에러 "ErrGone"`},
		{"key: tx_type", "key: tx_kind", `없는 enum "tx_kind"`},
		{"handles: transfer\n", "handles: refund\n", `tx_type에 없는 값 "refund"`},
		{"error: ErrLow }", "error: ErrNope }", `없는 에러 "ErrNope"`},
	}
	for _, c := range cases {
		ds := run(t, replace(t, c.old, c.new))
		want(t, ds, model.Error, c.msg)
	}
}

func TestRefFieldRefAndMax(t *testing.T) {
	ds := run(t, replace(t, "kind: tx_type }", "kind: tx_type, owner: { ref: user }, n: { type: uint, max: big } }"))
	want(t, ds, model.Error, `없는 타입 "user"`)
	want(t, ds, model.Error, `max "big"는 상수 ID나 숫자`)
	ds = run(t, replace(t, "kind: tx_type }", "kind: tx_type, n: { type: uint, max: 80 } }"))
	wantNone(t, ds, model.Error)
}

func TestRefOf(t *testing.T) {
	ds := run(t, replace(t, "kind: tx_handler\n    handles: transfer\n", "of: store.put\n"))
	want(t, ds, model.Error, `계약 store에 없는 메서드 "put"`)
	ds = run(t, replace(t, "kind: tx_handler\n    handles: transfer\n", "of: shop.get\n"))
	want(t, ds, model.Error, `없는 계약 "shop"`)
	ds = run(t, replace(t, "kind: tx_handler\n    handles: transfer\n", "of: storeget\n"))
	want(t, ds, model.Error, "계약.메서드 형식")
}

func TestExceptionTargetMustExist(t *testing.T) {
	ds := run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: ghost }] }"))
	want(t, ds, model.Error, `없는 모듈 "ghost"`)
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/check/`
Expected: FAIL, `undefined: Run`

- [ ] **Step 3: model 도우미**

`internal/model/index.go` 끝에 추가:

```go
func (c *Contract) Method(name string) *Method {
	for _, m := range c.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

func (e *Enum) HasValue(name string) bool { return hasRef(e.Values, name) }
```

- [ ] **Step 4: 검사 골격**

`internal/check/check.go`:

```go
// Package check는 모델만 보고 설계 검사를 한다. 코드는 읽지 않는다.
package check

import (
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type satisfier struct {
	id  string
	ref model.Ref
}

type checker struct {
	m          *model.Model
	diags      model.Diagnostics
	satisfiers map[string][]satisfier // 요구사항 ID → 충족한다고 적은 요소
}

// Run은 설계 검사 전부를 돌려 위치순으로 정렬한 진단을 준다.
func Run(m *model.Model) model.Diagnostics {
	c := &checker{m: m, satisfiers: map[string][]satisfier{}}
	c.eachMeta(func(meta *model.Meta) {
		for _, r := range meta.Satisfies {
			c.satisfiers[r.Name] = append(c.satisfiers[r.Name], satisfier{meta.ID, r})
		}
	})
	c.refs()
	c.l0()
	c.l2()
	c.l3()
	c.handlers()
	c.l4()
	c.requirements()
	c.diags.Sort()
	return c.diags
}

func (c *checker) errorf(pos yaml.Pos, format string, args ...any) {
	c.diags.Errorf(pos, format, args...)
}

func (c *checker) warnf(pos yaml.Pos, format string, args ...any) {
	c.diags.Warnf(pos, format, args...)
}

func (c *checker) infof(pos yaml.Pos, format string, args ...any) {
	c.diags.Infof(pos, format, args...)
}

// eachMeta는 satisfies를 가질 수 있는 모든 노드를 돈다.
func (c *checker) eachMeta(fn func(*model.Meta)) {
	m := c.m
	if m.System != nil {
		for _, a := range m.System.Actors {
			fn(&a.Meta)
		}
		for _, i := range m.System.Interfaces {
			fn(&i.Meta)
		}
	}
	for _, e := range m.Executables {
		fn(&e.Meta)
	}
	for _, x := range m.Modules {
		fn(&x.Meta)
	}
	for _, g := range m.Groups {
		fn(&g.Meta)
		for _, mem := range g.Members {
			fn(&mem.Meta)
		}
	}
	for _, x := range m.Enums {
		fn(&x.Meta)
	}
	for _, x := range m.Consts {
		fn(&x.Meta)
	}
	for _, x := range m.Errors {
		fn(&x.Meta)
	}
	for _, x := range m.Types {
		fn(&x.Meta)
	}
	for _, x := range m.Contracts {
		fn(&x.Meta)
	}
	for _, x := range m.HandlerKinds {
		fn(&x.Meta)
	}
	for _, x := range m.Funcs {
		fn(&x.Meta)
	}
}

func has(rs []model.Ref, name string) bool {
	for _, r := range rs {
		if r.Name == name {
			return true
		}
	}
	return false
}

// depsOf는 id가 설계상 의존하는 ID 목록이다. 멤버는 그룹의 의존과 자기 예외를 따른다.
func (c *checker) depsOf(id string) []string {
	m := c.m
	var out []string
	add := func(rs []model.Ref) {
		for _, r := range rs {
			out = append(out, r.Name)
		}
	}
	switch {
	case m.Module(id) != nil:
		add(m.Module(id).DependsOn)
	case m.Group(id) != nil:
		add(m.Group(id).DependsOn)
	case m.Member(id) != nil:
		mem := m.Member(id)
		add(mem.Group.DependsOn)
		for _, e := range mem.Exceptions {
			out = append(out, e.DependsOn.Name)
		}
	}
	return out
}

// unitPos는 모듈·그룹·멤버 ID의 정의 위치다.
func (c *checker) unitPos(id string) yaml.Pos {
	m := c.m
	switch {
	case m.Module(id) != nil:
		return m.Module(id).Pos
	case m.Group(id) != nil:
		return m.Group(id).Pos
	case m.Member(id) != nil:
		return m.Member(id).Pos
	}
	return yaml.Pos{}
}

func (c *checker) tierOf(id string) string {
	m := c.m
	switch {
	case m.Module(id) != nil:
		return m.Module(id).Tier.Name
	case m.Group(id) != nil:
		return m.Group(id).Tier.Name
	case m.Member(id) != nil:
		return m.Member(id).Group.Tier.Name
	}
	return ""
}
```

빈 파일들:

```go
// internal/check/l0.go
package check

func (c *checker) l0() {}
```

```go
// internal/check/l2.go
package check

func (c *checker) l2() {}
```

```go
// internal/check/l3.go
package check

func (c *checker) l3() {}
```

```go
// internal/check/handlers.go
package check

func (c *checker) handlers() {}
```

```go
// internal/check/l4.go
package check

func (c *checker) l4() {}
```

```go
// internal/check/req.go
package check

func (c *checker) requirements() {}
```

- [ ] **Step 5: 참조 해석**

`internal/check/refs.go`:

```go
package check

import (
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// refs는 모든 참조 필드가 존재하는, 받을 수 있는 종류의 대상을 가리키는지 본다.
func (c *checker) refs() {
	m := c.m
	c.eachMeta(func(meta *model.Meta) {
		for _, r := range meta.Satisfies {
			if m.Requirement(r.Name) == nil {
				c.errorf(r.Pos, "없는 요구사항 %q", r.Name)
			}
		}
	})
	if m.System != nil {
		for _, it := range m.System.Interfaces {
			for _, r := range it.UsedBy {
				if m.Actor(r.Name) == nil {
					c.errorf(r.Pos, "없는 액터 %q", r.Name)
				}
			}
		}
	}
	for _, e := range m.Executables {
		for _, r := range e.Exposes {
			c.iface(r)
		}
		for _, r := range e.Composes {
			c.dep(r)
		}
	}
	for _, mod := range m.Modules {
		c.tier(mod.Tier)
		for _, r := range mod.DependsOn {
			c.dep(r)
		}
		for _, r := range mod.Implements {
			c.iface(r)
		}
	}
	for _, g := range m.Groups {
		c.tier(g.Tier)
		if g.Kind.Name != "" && m.HandlerKind(g.Kind.Name) == nil {
			c.errorf(g.Kind.Pos, "없는 핸들러 종류 %q", g.Kind.Name)
		}
		for _, r := range g.DependsOn {
			c.dep(r)
		}
		for _, mem := range g.Members {
			for _, r := range mem.Handles {
				c.handleValue(g.Kind, r)
			}
			for _, e := range mem.Exceptions {
				c.dep(e.DependsOn)
			}
		}
	}
	for _, x := range m.Enums {
		c.in(x.In)
	}
	for _, x := range m.Consts {
		c.in(x.In)
	}
	for _, x := range m.Errors {
		c.in(x.In)
	}
	for _, t := range m.Types {
		c.in(t.In)
		for _, f := range t.Fields {
			if f.Type.Name != "" {
				c.typeRef(f.Type)
			}
			if f.RefType.Name != "" && m.Type(f.RefType.Name) == nil {
				c.errorf(f.RefType.Pos, "없는 타입 %q", f.RefType.Name)
			}
			if f.Max.Name != "" {
				if _, err := strconv.Atoi(f.Max.Name); err != nil && m.Const(f.Max.Name) == nil {
					c.errorf(f.Max.Pos, "max %q는 상수 ID나 숫자여야 한다", f.Max.Name)
				}
			}
		}
	}
	for _, ct := range m.Contracts {
		c.in(ct.In)
		for _, me := range ct.Methods {
			c.sig(me.Params, me.Returns)
			for _, r := range me.Errors {
				c.errRef(r)
			}
		}
	}
	for _, hk := range m.HandlerKinds {
		c.in(hk.In)
		c.sig(hk.Params, hk.Returns)
		if hk.Key.Name != "" && m.Enum(hk.Key.Name) == nil {
			c.errorf(hk.Key.Pos, "없는 enum %q", hk.Key.Name)
		}
	}
	for _, f := range m.Funcs {
		c.in(f.In)
		if f.Kind.Name != "" {
			if m.HandlerKind(f.Kind.Name) == nil {
				c.errorf(f.Kind.Pos, "없는 핸들러 종류 %q", f.Kind.Name)
			} else if f.Handles.Name != "" {
				c.handleValue(f.Kind, f.Handles)
			}
		}
		if f.Of.Name != "" {
			c.ofRef(f.Of)
		}
		for _, r := range f.Errors {
			c.errRef(r)
		}
		for _, cs := range f.Cases {
			if cs.Error.Name != "" && cs.Error.Name != "invalid" {
				c.errRef(cs.Error)
			}
		}
	}
}

func (c *checker) iface(r model.Ref) {
	if c.m.Interface(r.Name) == nil {
		c.errorf(r.Pos, "없는 인터페이스 %q", r.Name)
	}
}

func (c *checker) dep(r model.Ref) {
	m := c.m
	if m.Module(r.Name) == nil && m.Group(r.Name) == nil && m.Member(r.Name) == nil {
		c.errorf(r.Pos, "없는 모듈 %q", r.Name)
	}
}

func (c *checker) tier(r model.Ref) {
	if r.Name != "" && c.m.TierIndex(r.Name) < 0 {
		c.errorf(r.Pos, "없는 tier %q", r.Name)
	}
}

func (c *checker) in(r model.Ref) {
	if r.Name == "" {
		return
	}
	if _, ok := c.m.Owner(r.Name); !ok {
		c.errorf(r.Pos, "in이 가리키는 모듈이나 멤버 %q가 없다", r.Name)
	}
}

func (c *checker) typeRef(r model.Ref) {
	if k, base := c.m.ResolveType(r.Name); k == model.UnknownType {
		c.errorf(r.Pos, "알 수 없는 타입 %q", base)
	}
}

func (c *checker) sig(params []*model.Param, returns []model.Ref) {
	for _, p := range params {
		c.typeRef(p.Type)
	}
	for _, r := range returns {
		c.typeRef(r)
	}
}

func (c *checker) errRef(r model.Ref) {
	if c.m.Error(r.Name) == nil {
		c.errorf(r.Pos, "없는 에러 %q", r.Name)
	}
}

// handleValue는 handles 값이 핸들러 종류의 키 enum에 있는지 본다.
// 핸들러 종류나 enum이 없으면 다른 곳에서 이미 오류이므로 조용히 넘긴다.
func (c *checker) handleValue(kind model.Ref, r model.Ref) {
	hk := c.m.HandlerKind(kind.Name)
	if hk == nil {
		return
	}
	en := c.m.Enum(hk.Key.Name)
	if en == nil {
		return
	}
	if !en.HasValue(r.Name) {
		c.errorf(r.Pos, "%s에 없는 값 %q", en.ID, r.Name)
	}
}

func (c *checker) ofRef(r model.Ref) {
	ct, meth, ok := strings.Cut(r.Name, ".")
	if !ok {
		c.errorf(r.Pos, "of는 계약.메서드 형식이어야 한다")
		return
	}
	contract := c.m.Contract(ct)
	if contract == nil {
		c.errorf(r.Pos, "없는 계약 %q", ct)
		return
	}
	if contract.Method(meth) == nil {
		c.errorf(r.Pos, "계약 %s에 없는 메서드 %q", ct, meth)
	}
}
```

- [ ] **Step 6: 통과 확인**

Run: `gofmt -l ./internal/ && go vet ./internal/... && go test ./internal/check/ ./internal/model/`
Expected: PASS

- [ ] **Step 7: 커밋**

```bash
git add internal/check/ internal/model/index.go
git commit -m "feat: 설계 검사 골격과 참조 해석"
```

---

### Task 10: L0·L2 검사

**Files:**
- Modify: `internal/check/l0.go`, `internal/check/l2.go`
- Test: `internal/check/check_test.go`

**Interfaces:**
- Produces: 인터페이스 구현·조립 검사, tier 방향, 의존 순환, 고아 모듈, 그룹 예외 정보.

- [ ] **Step 1: 실패하는 테스트**

```go
func TestL0(t *testing.T) {
	ds := run(t, replace(t, "implements: [rpc]", "implements: []"))
	want(t, ds, model.Error, "인터페이스 rpc를 구현하는 모듈이 없다")

	ds = run(t, replace(t, "composes: [core, vm, tx_modules, rpc_mod]", "composes: [core, vm, tx_modules]"))
	want(t, ds, model.Error, "실행 단위 node가 rpc를 노출하지만 구현 모듈 rpc_mod를 조립하지 않는다")

	// composes에 없어도 depends_on으로 닿으면 조립된 것이다.
	ds = run(t, replace(t, "composes: [core, vm, tx_modules, rpc_mod]", "composes: [rpc_mod, tx_modules]"))
	wantNone(t, ds, model.Error)
}

func TestTierDirection(t *testing.T) {
	ds := run(t, replace(t, "core:    { tier: domain, depends_on: [crypto]", "core:    { tier: domain, depends_on: [crypto, vm]"))
	d := want(t, ds, model.Error, "상위 tier 의존: core(domain) → vm(execution)")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
	// 같은 tier는 허용
	ds = run(t, replace(t, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, depends_on: [hash] }\n  hash:    { tier: foundation }"))
	wantNone(t, ds, model.Error)
	// 멤버 예외도 tier를 따른다
	ds = run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: rpc_mod }] }"))
	want(t, ds, model.Error, "상위 tier 의존: asset(execution) → rpc_mod(service)")
}

func TestCycle(t *testing.T) {
	ds := run(t, replace(t, "vm:      { tier: execution, depends_on: [core] }", "vm:      { tier: execution, depends_on: [core, tx_modules] }"))
	want(t, ds, model.Error, "의존 순환: vm → tx_modules → vm")
	if n := ds.Count(model.Error); n != 1 {
		t.Fatalf("one cycle error, got %d:\n%s", n, join(ds))
	}
}

func TestCycleThroughException(t *testing.T) {
	src := replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: helper }] }")
	src = strings.Replace(src, "  crypto:  { tier: foundation }", "  crypto:  { tier: foundation }\n  helper:  { tier: execution, depends_on: [tx_modules] }", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "의존 순환: helper → tx_modules → helper")
	if n := ds.Count(model.Error); n != 1 {
		t.Fatalf("one cycle error, got %d:\n%s", n, join(ds))
	}
}

func TestOrphanModule(t *testing.T) {
	src := strings.Replace(base, "  crypto:  { tier: foundation }", "  crypto:  { tier: foundation }\n  lonely:  { tier: foundation }", 1)
	ds := run(t, src)
	want(t, ds, model.Warning, "어느 실행 단위에도 없는 모듈 lonely")
	// 실행 단위가 없으면 검사하지 않는다
	ds = run(t, strings.Replace(src, "executables:\n  node: { exposes: [rpc], composes: [core, vm, tx_modules, rpc_mod] }\n", "", 1))
	wantNone(t, ds, model.Warning)
}

func TestGroupExceptions(t *testing.T) {
	ds := run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [transfer], exceptions: [{ depends_on: asset, reason: \"검증 재사용\" }, { depends_on: core }] }"))
	want(t, ds, model.Info, "그룹 규칙 예외: economy → asset (검증 재사용)")
	want(t, ds, model.Warning, "예외 core는 그룹 tx_modules가 이미 의존한다")
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/check/ -run 'TestL0|TestTier|TestCycle|TestOrphan|TestGroup'`
Expected: FAIL

- [ ] **Step 3: L0**

`internal/check/l0.go` 전체:

```go
package check

import "github.com/tolelom/daemok/internal/model"

// l0은 인터페이스마다 구현 모듈이 있고, 노출하는 실행 단위가 그 모듈을 조립하는지 본다.
func (c *checker) l0() {
	m := c.m
	if m.System == nil {
		return
	}
	implementers := map[string][]string{}
	for _, mod := range m.Modules {
		for _, r := range mod.Implements {
			implementers[r.Name] = append(implementers[r.Name], mod.ID)
		}
	}
	for _, it := range m.System.Interfaces {
		if len(implementers[it.ID]) == 0 {
			c.errorf(it.Pos, "인터페이스 %s를 구현하는 모듈이 없다", it.ID)
		}
	}
	for _, e := range m.Executables {
		reach := c.reachable(e)
		for _, r := range e.Exposes {
			for _, impl := range implementers[r.Name] {
				if !reach[impl] {
					c.errorf(r.Pos, "실행 단위 %s가 %s를 노출하지만 구현 모듈 %s를 조립하지 않는다", e.ID, r.Name, impl)
				}
			}
		}
	}
}

// reachable은 composes에서 의존을 따라 닿는 모듈·그룹·멤버 ID 집합이다. 그룹은 멤버를 포함한다.
func (c *checker) reachable(e *model.Executable) map[string]bool {
	seen := map[string]bool{}
	var visit func(id string)
	visit = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		for _, d := range c.depsOf(id) {
			visit(d)
		}
		if g := c.m.Group(id); g != nil {
			for _, mem := range g.Members {
				visit(mem.ID)
			}
		}
	}
	for _, r := range e.Composes {
		visit(r.Name)
	}
	return seen
}
```

- [ ] **Step 4: L2**

`internal/check/l2.go` 전체:

```go
package check

import (
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

func (c *checker) l2() {
	c.tierDirection()
	c.cycles()
	c.orphans()
	c.exceptions()
}

// tierDirection은 자기보다 위 tier에 의존하면 오류다. 같은 tier는 허용한다.
func (c *checker) tierDirection() {
	m := c.m
	check := func(fromID, fromTier string, deps []model.Ref) {
		fi := m.TierIndex(fromTier)
		if fi < 0 {
			return
		}
		for _, d := range deps {
			toTier := c.tierOf(d.Name)
			if m.TierIndex(toTier) > fi {
				c.errorf(d.Pos, "상위 tier 의존: %s(%s) → %s(%s)", fromID, fromTier, d.Name, toTier)
			}
		}
	}
	for _, mod := range m.Modules {
		check(mod.ID, mod.Tier.Name, mod.DependsOn)
	}
	for _, g := range m.Groups {
		check(g.ID, g.Tier.Name, g.DependsOn)
		for _, mem := range g.Members {
			for _, e := range mem.Exceptions {
				check(mem.ID, g.Tier.Name, []model.Ref{e.DependsOn})
			}
		}
	}
}

// cycles는 모듈과 그룹의 의존 그래프에서 순환을 찾는다. 순환마다 한 번 보고한다.
// 멤버는 그룹으로 접는다: 멤버를 가리키는 의존은 그 그룹으로, 멤버의 예외는 그룹의 의존으로 본다.
// 그룹에 의존하면 멤버에도 의존할 수 있으므로 멤버 예외를 거치는 순환도 여기서 잡힌다.
func (c *checker) cycles() {
	m := c.m
	var ids []string
	for _, mod := range m.Modules {
		ids = append(ids, mod.ID)
	}
	for _, g := range m.Groups {
		ids = append(ids, g.ID)
	}
	collapse := func(id string) string {
		if mem := m.Member(id); mem != nil {
			return mem.Group.ID
		}
		return id
	}
	// 형제 멤버로 가는 예외는 그룹 자신으로 접히므로 뺀다.
	edges := func(id string) []string {
		var out []string
		add := func(d string) {
			if d = collapse(d); d != id {
				out = append(out, d)
			}
		}
		for _, d := range c.depsOf(id) {
			add(d)
		}
		if g := m.Group(id); g != nil {
			for _, mem := range g.Members {
				for _, e := range mem.Exceptions {
					add(e.DependsOn.Name)
				}
			}
		}
		return out
	}
	const (
		white = iota
		gray
		black
	)
	color := map[string]int{}
	var stack []string
	var visit func(id string)
	visit = func(id string) {
		color[id] = gray
		stack = append(stack, id)
		for _, d := range edges(id) {
			switch color[d] {
			case white:
				if slices.Contains(ids, d) {
					visit(d)
				}
			case gray:
				i := slices.Index(stack, d)
				cyc := append(slices.Clone(stack[i:]), d)
				c.errorf(c.unitPos(id), "의존 순환: %s", strings.Join(cyc, " → "))
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
	}
	for _, id := range ids {
		if color[id] == white {
			visit(id)
		}
	}
}

// orphans는 어느 실행 단위에서도 닿지 않는 모듈·그룹을 경고한다. 실행 단위가 없으면 건너뛴다.
func (c *checker) orphans() {
	m := c.m
	if len(m.Executables) == 0 {
		return
	}
	reach := map[string]bool{}
	for _, e := range m.Executables {
		for id := range c.reachable(e) {
			reach[id] = true
		}
	}
	for _, mod := range m.Modules {
		if !reach[mod.ID] {
			c.warnf(mod.Pos, "어느 실행 단위에도 없는 모듈 %s", mod.ID)
		}
	}
	for _, g := range m.Groups {
		if !reach[g.ID] {
			c.warnf(g.Pos, "어느 실행 단위에도 없는 그룹 %s", g.ID)
		}
	}
}

// exceptions는 멤버 예외를 정보로 남기고, 그룹이 이미 의존하는 대상이면 경고한다.
func (c *checker) exceptions() {
	for _, g := range c.m.Groups {
		for _, mem := range g.Members {
			for _, e := range mem.Exceptions {
				if has(g.DependsOn, e.DependsOn.Name) {
					c.warnf(e.DependsOn.Pos, "예외 %s는 그룹 %s가 이미 의존한다", e.DependsOn.Name, g.ID)
					continue
				}
				msg := "그룹 규칙 예외: " + mem.ID + " → " + e.DependsOn.Name
				if e.Reason != "" {
					msg += " (" + e.Reason + ")"
				}
				c.infof(e.DependsOn.Pos, "%s", msg)
			}
		}
	}
}
```

- [ ] **Step 5: 통과 확인**

Run: `gofmt -l ./internal/ && go test ./internal/check/`
Expected: PASS. `TestCycleThroughException`은 멤버 `asset`의 예외가 그룹으로 접혀 `helper → tx_modules → helper` 하나로 보고된다. 멤버마다 따로 보고하면 멤버 수만큼 같은 순환이 반복되므로 접는다.

- [ ] **Step 6: 커밋**

```bash
git add internal/check/
git commit -m "feat: L0·L2 설계 검사: 구현·조립, tier 방향, 순환, 고아, 예외"
```

---

### Task 11: L3 검사와 핸들러 키 검사

**Files:**
- Modify: `internal/check/l3.go`, `internal/check/handlers.go`
- Test: `internal/check/check_test.go`

- [ ] **Step 1: 실패하는 테스트**

```go
func TestTypeDependency(t *testing.T) {
	// crypto(foundation)의 타입이 core의 enum을 쓰면 crypto는 core에 의존하지 않으므로 오류
	src := strings.Replace(base, "types:\n", "types:\n  sig: { in: crypto, fields: { kind: tx_type } }\n", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "crypto가 core의 타입 tx_type를 쓰지만 core에 의존하지 않는다")
	// 멤버는 그룹 의존을 따른다
	src = strings.Replace(base, "types:\n", "types:\n  memo: { in: economy, fields: { p: payload } }\n", 1)
	wantNone(t, run(t, src), model.Error)
	// handler_kind의 key enum도 의존 대상이다
	src = replace(t, "vm:      { tier: execution, depends_on: [core] }", "vm:      { tier: execution, depends_on: [] }")
	src = strings.Replace(src, "params: { payload: payload }", "params: {}", 1)
	ds = run(t, src)
	want(t, ds, model.Error, "vm가 core의 타입 tx_type를 쓰지만 core에 의존하지 않는다")
}

func TestErrorHTTP(t *testing.T) {
	ds := run(t, replace(t, `http: 404 }`, `http: 400 }`))
	want(t, ds, model.Warning, "ErrNotFound의 이름은 404를 뜻하지만 http가 400다")
	ds = run(t, replace(t, `ErrLow:      { in: core, message: "low" }`, `ErrLow:      { in: core, message: "low" }
  ErrUnauthorized: { in: core, message: "u", http: 403 }
  ErrForbidden:    { in: core, message: "f", http: 403 }`))
	want(t, ds, model.Warning, "ErrUnauthorized의 이름은 401을 뜻하지만 http가 403다")
	if strings.Contains(join(ds), "ErrForbidden") {
		t.Fatal("ErrForbidden 403 is consistent")
	}
}

func TestContractWithoutErrors(t *testing.T) {
	ds := run(t, replace(t, "returns: [payload], errors: [ErrNotFound] }", "returns: [payload] }"))
	want(t, ds, model.Warning, "계약 store의 메서드 get에 errors가 없다")
	ds = run(t, replace(t, "returns: [payload], errors: [ErrNotFound] }", "returns: [payload], errors: [] }"))
	wantNone(t, ds, model.Warning)
}

func TestHandlerCoverage(t *testing.T) {
	ds := run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [] }"))
	want(t, ds, model.Error, "tx_type.transfer를 처리하는 멤버가 없다")
	ds = run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint, transfer] }"))
	d := want(t, ds, model.Error, "tx_type.transfer를 economy, asset가 함께 처리한다")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
}
```

`TestHandlerCoverage`의 첫 경우는 `handle_transfer`의 `handles: transfer`가 멤버 `handles`에 없다는 L4 오류도 함께 낸다. 그건 Task 12에서 확인한다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/check/ -run 'TestTypeDep|TestErrorHTTP|TestContract|TestHandler'`
Expected: FAIL

- [ ] **Step 3: L3**

`internal/check/l3.go` 전체:

```go
package check

import (
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

func (c *checker) l3() {
	c.typeDeps()
	c.errorHTTP()
	c.contractErrors()
}

// typeDeps는 노드가 쓰는 enum·타입의 모듈에 그 노드의 모듈이 의존하는지 본다.
func (c *checker) typeDeps() {
	m := c.m
	use := func(owner string, r model.Ref) {
		if r.Name == "" || owner == "" {
			return
		}
		kind, base := m.ResolveType(r.Name)
		var typeOwner string
		switch kind {
		case model.EnumType:
			typeOwner = m.Enum(base).In.Name
		case model.StructType:
			typeOwner = m.Type(base).In.Name
		default:
			return
		}
		if typeOwner != "" && !m.CanDepend(owner, typeOwner) {
			c.errorf(r.Pos, "%s가 %s의 타입 %s를 쓰지만 %s에 의존하지 않는다", owner, typeOwner, base, typeOwner)
		}
	}
	for _, t := range m.Types {
		for _, f := range t.Fields {
			use(t.In.Name, f.Type)
			use(t.In.Name, f.RefType)
		}
	}
	for _, ct := range m.Contracts {
		for _, me := range ct.Methods {
			for _, p := range me.Params {
				use(ct.In.Name, p.Type)
			}
			for _, r := range me.Returns {
				use(ct.In.Name, r)
			}
		}
	}
	for _, hk := range m.HandlerKinds {
		for _, p := range hk.Params {
			use(hk.In.Name, p.Type)
		}
		for _, r := range hk.Returns {
			use(hk.In.Name, r)
		}
		use(hk.In.Name, hk.Key)
	}
}

var httpBySuffix = []struct {
	suffix string
	code   int
	josa   string // "를"·"을"
}{
	{"NotFound", 404, "를"},
	{"Unauthorized", 401, "을"},
	{"Forbidden", 403, "을"},
	{"AlreadyExists", 409, "를"},
	{"Conflict", 409, "를"},
}

// errorHTTP는 에러 이름의 접미사가 뜻하는 HTTP 상태와 http 값이 다르면 경고한다.
func (c *checker) errorHTTP() {
	for _, e := range c.m.Errors {
		if e.HTTP == 0 {
			continue
		}
		for _, s := range httpBySuffix {
			if strings.HasSuffix(e.ID, s.suffix) && e.HTTP != s.code {
				c.warnf(e.Pos, "%s의 이름은 %d%s 뜻하지만 http가 %d다", e.ID, s.code, s.josa, e.HTTP)
			}
		}
	}
}

func (c *checker) contractErrors() {
	for _, ct := range c.m.Contracts {
		for _, me := range ct.Methods {
			if !me.HasErrors {
				c.warnf(me.Pos, "계약 %s의 메서드 %s에 errors가 없다", ct.ID, me.Name)
			}
		}
	}
}
```

- [ ] **Step 4: 핸들러 키**

`internal/check/handlers.go` 전체:

```go
package check

import (
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// handlers는 핸들러 종류의 키 enum 값마다 처리하는 멤버가 정확히 하나인지 본다.
func (c *checker) handlers() {
	m := c.m
	for _, hk := range m.HandlerKinds {
		en := m.Enum(hk.Key.Name)
		if en == nil {
			continue
		}
		type handler struct {
			member string
			ref    model.Ref
		}
		byValue := map[string][]handler{}
		for _, g := range m.Groups {
			if g.Kind.Name != hk.ID {
				continue
			}
			for _, mem := range g.Members {
				for _, r := range mem.Handles {
					byValue[r.Name] = append(byValue[r.Name], handler{mem.ID, r})
				}
			}
		}
		for _, v := range en.Values {
			hs := byValue[v.Name]
			switch {
			case len(hs) == 0:
				c.errorf(v.Pos, "%s.%s를 처리하는 멤버가 없다", en.ID, v.Name)
			case len(hs) > 1:
				var names []string
				for _, h := range hs {
					names = append(names, h.member)
				}
				c.errorf(hs[1].ref.Pos, "%s.%s를 %s가 함께 처리한다", en.ID, v.Name, strings.Join(names, ", "))
			}
		}
	}
}
```

- [ ] **Step 5: 통과 확인**

Run: `gofmt -l ./internal/ && go test ./internal/check/`
Expected: PASS

- [ ] **Step 6: 커밋**

```bash
git add internal/check/
git commit -m "feat: L3 설계 검사와 핸들러 키 검사"
```

---

### Task 12: L4·요구사항 검사와 샘플 기대 진단

**Files:**
- Modify: `internal/check/l4.go`, `internal/check/req.go`
- Test: `internal/check/check_test.go`

- [ ] **Step 1: 실패하는 테스트**

```go
func TestL4Handler(t *testing.T) {
	ds := run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [] }"))
	want(t, ds, model.Error, "멤버 economy의 handles에 transfer가 없다")

	ds = run(t, replace(t, "in: economy\n    kind: tx_handler", "in: core\n    kind: tx_handler"))
	want(t, ds, model.Error, "핸들러 함수 handle_transfer는 그룹 멤버에 있어야 한다")
}

func TestL4Of(t *testing.T) {
	src := replace(t, "in: economy\n    kind: tx_handler\n    handles: transfer\n", "in: vm\n    of: store.get\n")
	ds := run(t, src)
	want(t, ds, model.Error, "계약 store는 core에 있는데 함수 handle_transfer는 vm에 있다")
}

func TestL4CaseErrors(t *testing.T) {
	ds := run(t, replace(t, "error: ErrLow }", "error: ErrNotFound }"))
	want(t, ds, model.Error, `케이스 "low"의 에러 ErrNotFound가 함수 handle_transfer의 errors에 없다`)
	want(t, ds, model.Warning, "함수 handle_transfer의 에러 ErrLow를 확인하는 케이스가 없다")
	// invalid는 errors에 없어도 된다
	ds = run(t, replace(t, "error: ErrLow }", "error: invalid }"))
	wantNone(t, ds, model.Error)
}

func TestRequirements(t *testing.T) {
	src := strings.Replace(base, "  REQ-a: { text: A }\n", "  REQ-a: { text: A }\n  REQ-b: { text: B }\n  REQ-c: { text: C, status: held }\n", 1)
	src = strings.Replace(src, "vm:      { tier: execution, depends_on: [core] }", "vm:      { tier: execution, depends_on: [core], satisfies: [REQ-c] }", 1)
	ds := run(t, src)
	want(t, ds, model.Warning, "요구사항 REQ-b를 충족하는 설계 요소가 없다")
	d := want(t, ds, model.Warning, "held 요구사항 REQ-c에 vm가 매달려 있다")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
	if strings.Contains(join(ds), "REQ-a") {
		t.Fatal("REQ-a is satisfied")
	}
}

func TestSamples(t *testing.T) {
	cases := []struct {
		name     string
		errors   []string
		warnings []string
		infos    []string
	}{
		{
			name:     "tolchain",
			warnings: []string{"held 요구사항 REQ-random에 random가 매달려 있다", "함수 handle_transfer의 에러 ErrBalanceOverflow를 확인하는 케이스가 없다"},
			infos:    []string{"그룹 규칙 예외: reward → asset (보상 지급 시 자산 스키마 검증 재사용)"},
		},
		{
			name:     "tolelog",
			errors:   []string{"상위 tier 의존: post(service) → dto(transport)"},
			warnings: []string{"ErrUnauthorized의 이름은 401을 뜻하지만 http가 403다", "요구사항 REQ-oauth를 충족하는 설계 요소가 없다"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, ds := model.Load(filepath.Join("..", "model", "testdata", c.name, "design"))
			if len(ds) != 0 {
				t.Fatalf("load:\n%s", join(ds))
			}
			ds = Run(m)
			for level, msgs := range map[model.Level][]string{model.Error: c.errors, model.Warning: c.warnings, model.Info: c.infos} {
				if got := ds.Count(level); got != len(msgs) {
					t.Errorf("%v: want %d, got %d\n%s", level, len(msgs), got, join(ds))
				}
				for _, msg := range msgs {
					want(t, ds, level, msg)
				}
			}
		})
	}
}
```

- [ ] **Step 2: 실패 확인**

Run: `go test ./internal/check/ -run 'TestL4|TestRequirements|TestSamples'`
Expected: FAIL

- [ ] **Step 3: L4**

`internal/check/l4.go` 전체:

```go
package check

import "strings"

// l4는 함수 계약이 소속 멤버·계약과 맞고, 케이스의 에러가 errors와 맞는지 본다.
func (c *checker) l4() {
	m := c.m
	for _, f := range m.Funcs {
		owner, ok := m.Owner(f.In.Name)
		if ok && f.Kind.Name != "" {
			switch {
			case owner.Member == nil:
				c.errorf(f.In.Pos, "핸들러 함수 %s는 그룹 멤버에 있어야 한다", f.ID)
			case owner.Member.Group.Kind.Name != f.Kind.Name:
				c.errorf(f.Kind.Pos, "멤버 %s의 그룹은 %s 종류인데 함수 %s는 %s다", owner.ID, owner.Member.Group.Kind.Name, f.ID, f.Kind.Name)
			case f.Handles.Name != "" && !has(owner.Member.Handles, f.Handles.Name):
				c.errorf(f.Handles.Pos, "멤버 %s의 handles에 %s가 없다", owner.ID, f.Handles.Name)
			}
		}
		if ok && f.Of.Name != "" {
			ct, _, _ := strings.Cut(f.Of.Name, ".")
			if contract := m.Contract(ct); contract != nil && contract.In.Name != f.In.Name {
				c.errorf(f.Of.Pos, "계약 %s는 %s에 있는데 함수 %s는 %s에 있다", ct, contract.In.Name, f.ID, f.In.Name)
			}
		}
		covered := map[string]bool{}
		for _, cs := range f.Cases {
			e := cs.Error.Name
			if e == "" || e == "invalid" {
				continue
			}
			covered[e] = true
			if !has(f.Errors, e) {
				c.errorf(cs.Error.Pos, "케이스 %q의 에러 %s가 함수 %s의 errors에 없다", cs.Name, e, f.ID)
			}
		}
		for _, r := range f.Errors {
			if !covered[r.Name] {
				c.warnf(r.Pos, "함수 %s의 에러 %s를 확인하는 케이스가 없다", f.ID, r.Name)
			}
		}
	}
}
```

- [ ] **Step 4: 요구사항**

`internal/check/req.go` 전체:

```go
package check

// requirements는 active 요구사항에 충족 요소가 있는지, held·dropped 요구사항에 요소가 매달려 있지 않은지 본다.
func (c *checker) requirements() {
	for _, r := range c.m.Requirements {
		sats := c.satisfiers[r.ID]
		if r.Status == "active" {
			if len(sats) == 0 {
				c.warnf(r.Pos, "요구사항 %s를 충족하는 설계 요소가 없다", r.ID)
			}
			continue
		}
		for _, s := range sats {
			c.warnf(s.ref.Pos, "%s 요구사항 %s에 %s가 매달려 있다", r.Status, r.ID, s.id)
		}
	}
}
```

- [ ] **Step 5: 통과 확인**

Run: `gofmt -l ./internal/ && go vet ./... && go test ./...`
Expected: PASS. `TestSamples`가 기대와 다른 진단을 내면 먼저 샘플과 `docs/design/01-model.md`로 어느 쪽이 맞는지 판정한다. 검사 규칙이 문서와 다르면 검사를 고치고, 샘플이 문서를 어기면 샘플을 고친 뒤 이 테스트의 기대 목록을 갱신한다. 기대 목록만 늘려서 통과시키지 않는다.

- [ ] **Step 6: 커밋**

```bash
git add internal/check/
git commit -m "feat: L4·요구사항 검사와 샘플 기대 진단"
```

---

### Task 13: `daemok check` 명령

**Files:**
- Modify: `cmd/daemok/check.go`, `cmd/daemok/main_test.go`

**Interfaces:**
- Consumes: `model.Load`, `check.Run`, `model.Diagnostics.Sort/Count/HasErrors`
- Produces: `daemok check [--design DIR] [--root DIR] [--strict]`. 진단을 한 줄씩 stdout에, 마지막에 `오류 N, 경고 N, 정보 N`. `--root`는 받기만 하고 계획 3의 코드 대조가 쓴다.

- [ ] **Step 1: 실패하는 테스트**

`cmd/daemok/main_test.go`에 추가:

```go
func writeDesign(t *testing.T, src string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "design")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const cleanDesign = `version: 0.3
tiers: [a]
modules:
  core: { tier: a }
`

func TestCheckClean(t *testing.T) {
	dir := writeDesign(t, cleanDesign)
	code, out, _ := runCLI(t, "check", "--design", dir)
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if strings.TrimSpace(out) != "오류 0, 경고 0, 정보 0" {
		t.Fatalf("out %q", out)
	}
}

func TestCheckMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	code, out, _ := runCLI(t, "check", "--design", missing)
	if code != 2 || !strings.Contains(out, missing) || !strings.Contains(out, "설계 파일(*.yaml)이 없다") {
		t.Fatalf("exit %d out %q", code, out)
	}
}

func TestCheckModelErrorExits2(t *testing.T) {
	dir := writeDesign(t, "version: 0.3\nmodulez: {}\n")
	code, out, _ := runCLI(t, "check", "--design", dir)
	if code != 2 || !strings.Contains(out, "d.yaml:2:1: 오류: 알 수 없는 키 \"modulez\"") {
		t.Fatalf("exit %d out %q", code, out)
	}
}

func TestCheckDesignErrorExits1(t *testing.T) {
	dir := writeDesign(t, "version: 0.3\ntiers: [a, b]\nmodules:\n  low:  { tier: a, depends_on: [high] }\n  high: { tier: b }\n")
	code, out, _ := runCLI(t, "check", "--design", dir)
	if code != 1 || !strings.Contains(out, "d.yaml:4:32: 오류: 상위 tier 의존: low(a) → high(b)") || !strings.Contains(out, "오류 1, 경고 0, 정보 0") {
		t.Fatalf("exit %d out %q", code, out)
	}
}

func TestCheckStrict(t *testing.T) {
	dir := writeDesign(t, "version: 0.3\nrequirements:\n  REQ-a: { text: A }\n")
	code, _, _ := runCLI(t, "check", "--design", dir)
	if code != 0 {
		t.Fatalf("warning only: exit %d", code)
	}
	code, out, _ := runCLI(t, "check", "--design", dir, "--strict")
	if code != 1 || !strings.Contains(out, "경고: 요구사항 REQ-a") {
		t.Fatalf("strict: exit %d out %q", code, out)
	}
}

func TestCheckOutputSorted(t *testing.T) {
	dir := writeDesign(t, "version: 0.3\nrequirements:\n  REQ-b: { text: B }\n  REQ-a: { text: A }\n")
	_, out, _ := runCLI(t, "check", "--design", dir, "--strict")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], ":3:3:") || !strings.Contains(lines[1], ":4:3:") {
		t.Fatalf("%q", out)
	}
}

func TestCheckBadFlag(t *testing.T) {
	code, _, errOut := runCLI(t, "check", "--bogus")
	if code != 2 || !strings.Contains(errOut, "bogus") {
		t.Fatalf("exit %d err %q", code, errOut)
	}
}
```

import에 `"os"`, `"path/filepath"`를 더한다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./cmd/daemok/`
Expected: FAIL (모든 check 테스트가 exit 2)

- [ ] **Step 3: 구현**

`cmd/daemok/check.go` 전체:

```go
package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/model"
)

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design := fs.String("design", "design", "설계 디렉토리")
	fs.String("root", ".", "코드 루트 (코드 대조가 쓴다)")
	strict := fs.Bool("strict", false, "경고도 실패로 본다")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	m, diags := model.Load(*design)
	if diags.HasErrors() {
		diags.Sort()
		printDiags(stdout, diags)
		return 2
	}
	diags = append(diags, check.Run(m)...)
	diags.Sort()
	printDiags(stdout, diags)
	if diags.HasErrors() || (*strict && diags.Count(model.Warning) > 0) {
		return 1
	}
	return 0
}

func printDiags(w io.Writer, diags model.Diagnostics) {
	for _, d := range diags {
		fmt.Fprintln(w, d.String())
	}
	fmt.Fprintf(w, "오류 %d, 경고 %d, 정보 %d\n", diags.Count(model.Error), diags.Count(model.Warning), diags.Count(model.Info))
}
```

- [ ] **Step 4: 통과 확인**

Run: `gofmt -l . && go vet ./... && go test ./...`
Expected: PASS. `TestCheckDesignErrorExits1`의 열 32는 `  low:  { tier: a, depends_on: [high] }`에서 `high`의 첫 글자다.

- [ ] **Step 5: 샘플에 직접 실행**

```bash
go run ./cmd/daemok check --design internal/model/testdata/tolchain/design
go run ./cmd/daemok check --design internal/model/testdata/tolelog/design
```

Expected: tolchain은 종료 코드 0, 경고 2·정보 1. tolelog는 종료 코드 1, 오류 1·경고 2. 출력 줄이 `파일:줄:열: 수준: 메시지` 형식이고 줄·열이 샘플 파일의 해당 위치를 가리킨다(`sed -n '<줄>p'`로 확인).

- [ ] **Step 6: 커밋**

```bash
git add cmd/
git commit -m "feat: daemok check 명령: 진단 출력과 종료 코드"
```

---

## 계획 밖

이 계획이 끝나면 스펙의 3단계까지다. 남은 단계는 두 계획으로 나눈다.

| 계획 | 범위 | 스펙 단계 |
|---|---|---|
| 2 | `internal/gen/golang`, `internal/gen/ts`, `daemok gen`, 골든 파일과 임시 모듈 빌드 | 4·5 |
| 3 | `internal/codescan`, 생성 파일 최신 여부, `internal/gitbase`, `internal/serve`, 웹 화면 이식, tolchain 도입 | 6·7·8 |

계획 3의 tolchain 도입 전에 `github.com/tolelom/tolchain` main을 클론해 슬라이스 문서의 코드 위치(`core/transaction.go`, `vm/registry.go`)를 다시 확인한다. 슬라이스의 기준 시점은 2026-06-12다.
