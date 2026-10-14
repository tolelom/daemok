# 설계 ID 이름 바꾸기 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `daemok rename <종류> <옛 ID> <새 ID> [--code]`와 MCP `edit`의 `rename` 작업으로 설계 ID와 모든 참조를 한 번에 바꾸고, 기본은 코드 이름을 고정 키로 유지한다.

**Architecture:** 텍스트 조작(위치의 토큰 바꾸기, 고정 키 더하기)은 `internal/edit`에 둔다. 모델에서 바꿀 자리를 모으고, 검증(읽기 오류, 진단 수, 생성 결과)과 쓰기·되돌리기를 하는 조율은 새 패키지 `internal/rename`(pipeline 층)에 둔다. CLI와 MCP는 `rename.Run`을 부른다.

**Tech Stack:** Go 표준 라이브러리, 기존 `model`·`check`·`workspace`·`edit`.

**Spec:** `docs/superpowers/specs/2026-10-14-daemok-rename-design.md`

## Global Constraints

- Go 표준 라이브러리만 쓴다.
- 기본 모드에서 바꾼 결과의 생성 파일 목록과 내용이 바꾸기 전과 같아야 한다. 다르면 쓰지 않는다.
- 바꾼 결과의 설계 검사 진단(메시지의 옛 ID를 새 ID로 바꾼 뒤 수준·메시지별 수)이 늘면 쓰지 않는다.
- 편집 전에 없던 읽기 오류가 생기면 쓰지 않는다.
- 여러 파일 쓰기 중 실패하면 이미 쓴 파일을 원래 내용으로 되돌린다.
- 고정 불가 종류(이벤트, 이벤트 종류, 흐름, 실행 단위, 그룹 멤버)는 `--code` 없이 거부한다.
- MCP는 설계 디렉토리 밖을 가리키는 심볼릭 링크 파일을 고치지 않는다.
- 커밋 메시지는 `feat:`/`fix:`/`docs:` + 한국어 요약 + `Claude-Session` 줄. 항목마다 `go vet ./...`, `go test ./...` 통과 후 커밋.

## Review Focus

- 따옴표로 감싼 참조(`"post"`)와 따옴표 없는 참조가 섞인 파일: 따옴표 스타일을 유지한 채 안쪽만 바뀐다.
- 같은 ID를 가진 다른 종류(모듈 `post`, 타입 `post`)가 있을 때: 지정한 종류와 그 참조만 바뀐다.
- 블록 형식 노드와 흐름 형식 노드, 이미 `go:` 맵이 있는 노드에 고정 키 더하기: 결과가 다시 읽히고 서식이 유지된다.
- CRLF 줄바꿈 파일: 더한 줄도 CRLF다.
- 타입 문자열 안에 옛 ID가 다른 ID의 일부로 들어간 경우(`post`를 바꿀 때 `post_status`): 바뀌지 않는다.

---

### Task 1: edit 토큰 바꾸기

**Files:**
- Create: `internal/edit/replace.go`
- Test: `internal/edit/replace_test.go`

**Interfaces:**
- Produces: `type Replacement struct { Pos yaml.Pos; Old, New string }`, `func Replace(src []byte, file string, reps []Replacement) ([]byte, error)`. 각 위치의 원문 토큰(따옴표면 안쪽)이 `Old`와 같아야 하고 `New`로 바꾼다. 따옴표 스타일은 유지한다. 위치는 뒤에서부터 바꾼다.

- [ ] **Step 1: 실패하는 테스트**

```go
package edit

import (
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/yaml"
)

func TestReplaceKeepsQuotes(t *testing.T) {
	src := "types:\n  post: { fields: { a: \"[]*post\", b: post_status } }\nmodules:\n  web: { depends_on: [\"core\"] }\n"
	out, err := Replace([]byte(src), "d.yaml", []Replacement{
		{Pos: yaml.Pos{File: "d.yaml", Line: 2, Col: 3}, Old: "post", New: "article"},
		{Pos: yaml.Pos{File: "d.yaml", Line: 2, Col: 25}, Old: "[]*post", New: "[]*article"},
		{Pos: yaml.Pos{File: "d.yaml", Line: 4, Col: 22}, Old: "core", New: "base"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "types:\n  article: { fields: { a: \"[]*article\", b: post_status } }\nmodules:\n  web: { depends_on: [\"base\"] }\n"
	if string(out) != want {
		t.Fatalf("got\n%s", out)
	}
}

func TestReplaceRefusesMismatch(t *testing.T) {
	_, err := Replace([]byte("a: x\n"), "d.yaml", []Replacement{{Pos: yaml.Pos{File: "d.yaml", Line: 1, Col: 4}, Old: "y", New: "z"}})
	if err == nil || !strings.Contains(err.Error(), "원문") {
		t.Fatalf("%v", err)
	}
}

func TestReplaceRefusesEscapes(t *testing.T) {
	_, err := Replace([]byte("a: \"x\\ty\"\n"), "d.yaml", []Replacement{{Pos: yaml.Pos{File: "d.yaml", Line: 1, Col: 4}, Old: "x\ty", New: "z"}})
	if err == nil {
		t.Fatal("이스케이프가 든 토큰을 바꿨다")
	}
}
```

- [ ] **Step 2: 실패 확인** `go test ./internal/edit -run TestReplace` → `undefined: Replace`

- [ ] **Step 3: 구현**

```go
package edit

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

// Replacement는 설계 파일 한 위치의 스칼라 토큰을 바꾸는 일이다. Old는 원문 토큰(따옴표면 안쪽)이다.
type Replacement struct {
	Pos      yaml.Pos
	Old, New string
}

// Replace는 위치마다 원문 토큰이 Old인지 확인하고 New로 바꾼다. 따옴표 스타일은 그대로 둔다.
// 이스케이프가 든 따옴표 토큰은 원문과 값이 달라 바꾸지 않는다. 뒤의 위치부터 바꿔 앞의 교체가 뒤 위치를 밀지 않는다.
func Replace(src []byte, file string, reps []Replacement) ([]byte, error) {
	t, err := parse(src, file)
	if err != nil {
		return nil, err
	}
	sorted := append([]Replacement(nil), reps...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[j].Pos.Less(sorted[i].Pos) })
	out := src
	for i, r := range sorted {
		if i > 0 && r.Pos == sorted[i-1].Pos {
			continue
		}
		s := t.off(r.Pos)
		e := t.tokenEnd(s)
		inner, at := src[s:e], s
		if len(inner) >= 2 && inner[0] == '"' {
			inner, at = inner[1:len(inner)-1], s+1
			if bytes.IndexByte(inner, '\\') >= 0 {
				return nil, fmt.Errorf("%s: 이스케이프가 든 토큰은 바꾸지 않는다: %s", r.Pos, src[s:e])
			}
		}
		if string(inner) != r.Old {
			return nil, fmt.Errorf("%s: 원문 토큰이 %q가 아니라 %q다", r.Pos, r.Old, inner)
		}
		out = splice(out, at, at+len(inner), r.New)
	}
	return out, nil
}
```

- [ ] **Step 4: 통과 확인** `go test ./internal/edit`
- [ ] **Step 5: 커밋** `feat: edit.Replace(위치의 스칼라 토큰을 원문 확인 뒤 바꾸고 따옴표 스타일 유지)`

### Task 2: edit 고정 키 더하기

**Files:**
- Create: `internal/edit/pin.go`
- Test: `internal/edit/pin_test.go`

**Interfaces:**
- Produces: `func Pin(src []byte, file string, path []string, key, sub, value string) ([]byte, error)`. `path`는 최상위 키부터 노드까지의 키 목록(`["types", "post"]`, `["groups", "g", "members", "m"]`). `sub`가 비면 `key: value`(예: `path: core`), 아니면 `key: { sub: value }`(예: `go: { name: Post }`)를 더한다. 노드에 `key` 맵이 이미 있으면 그 안에 `sub: value`를 더한다. 노드가 이 파일에 없으면 `ErrNodeNotInFile`.

- [ ] **Step 1: 실패하는 테스트**

```go
package edit

import (
	"errors"
	"testing"

	"github.com/tolelom/daemok/internal/yaml"
)

func pinCase(t *testing.T, src string, path []string, key, sub, value, want string) {
	t.Helper()
	out, err := Pin([]byte(src), "d.yaml", path, key, sub, value)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != want {
		t.Fatalf("got\n%s\nwant\n%s", out, want)
	}
	if _, err := yaml.Parse("d.yaml", out); err != nil {
		t.Fatal(err)
	}
}

func TestPinFlow(t *testing.T) {
	pinCase(t, "types:\n  article: { in: core, fields: { a: int } }\n", []string{"types", "article"}, "go", "name", "Post",
		"types:\n  article: { in: core, fields: { a: int }, go: { name: Post } }\n")
}

func TestPinFlowExistingGoMap(t *testing.T) {
	pinCase(t, "enums:\n  kind2: { in: core, values: [a], go: { prefix: K } }\n", []string{"enums", "kind2"}, "go", "type", "Kind",
		"enums:\n  kind2: { in: core, values: [a], go: { prefix: K, type: Kind } }\n")
}

func TestPinBlock(t *testing.T) {
	pinCase(t, "modules:\n  base:\n    tier: a\n    # 주석\n    depends_on: [x]\nother: 1\n", []string{"modules", "base"}, "path", "", "core",
		"modules:\n  base:\n    tier: a\n    # 주석\n    depends_on: [x]\n    path: core\nother: 1\n")
}

func TestPinBlockExistingGoMap(t *testing.T) {
	pinCase(t, "types:\n  a:\n    in: core\n    go:\n      register: R\n", []string{"types", "a"}, "go", "name", "B",
		"types:\n  a:\n    in: core\n    go:\n      register: R\n      name: B\n")
}

func TestPinCRLF(t *testing.T) {
	pinCase(t, "modules:\r\n  base:\r\n    tier: a\r\n", []string{"modules", "base"}, "path", "", "core",
		"modules:\r\n  base:\r\n    tier: a\r\n    path: core\r\n")
}

func TestPinMember(t *testing.T) {
	pinCase(t, "groups:\n  g:\n    tier: a\n    members:\n      m2: { handles: [x] }\n", []string{"groups", "g", "members", "m2"}, "go", "name", "M",
		"groups:\n  g:\n    tier: a\n    members:\n      m2: { handles: [x], go: { name: M } }\n")
}

func TestPinNotInFile(t *testing.T) {
	if _, err := Pin([]byte("types: {}\n"), "d.yaml", []string{"types", "x"}, "go", "name", "X"); !errors.Is(err, ErrNodeNotInFile) {
		t.Fatalf("%v", err)
	}
}

func TestPinRefusesExistingKey(t *testing.T) {
	if _, err := Pin([]byte("types:\n  a: { go: { name: A } }\n"), "d.yaml", []string{"types", "a"}, "go", "name", "B"); err == nil {
		t.Fatal("이미 있는 키를 덮었다")
	}
}
```

- [ ] **Step 2: 실패 확인** `go test ./internal/edit -run TestPin` → `undefined: Pin`

- [ ] **Step 3: 구현**

```go
package edit

import (
	"errors"
	"fmt"

	"github.com/tolelom/daemok/internal/yaml"
)

// ErrNodeNotInFile은 고정 키를 더할 노드가 이 파일에 없다는 뜻이다.
var ErrNodeNotInFile = errors.New("이 파일에 노드가 없다")

// Pin은 path가 가리키는 맵 노드에 key: value(sub가 비면) 또는 key: { sub: value }를 더한다.
// key 맵이 이미 있으면 그 안에 sub: value를 더한다. 같은 키가 이미 있으면 거부한다.
// 흐름 형식이면 닫는 괄호 앞에, 블록 형식이면 노드의 마지막 내용 줄 뒤에 같은 들여쓰기로 넣는다.
func Pin(src []byte, file string, path []string, key, sub, value string) ([]byte, error) {
	t, err := parse(src, file)
	if err != nil {
		return nil, err
	}
	root, _ := yaml.Parse(file, src)
	n := root
	for _, k := range path {
		n = n.Get(k)
		if n == nil {
			return nil, fmt.Errorf("%w: %v", ErrNodeNotInFile, path)
		}
	}
	if n.Kind != yaml.Map {
		return nil, fmt.Errorf("맵 노드가 아니다: %v", path)
	}
	target, k, v := n, key, value
	if sub != "" {
		if inner := n.Get(key); inner != nil {
			if inner.Kind != yaml.Map {
				return nil, fmt.Errorf("%s가 맵이 아니다: %v", key, path)
			}
			target, k = inner, sub
		} else {
			v = "{ " + sub + ": " + value + " }"
		}
	}
	if target.Get(k) != nil {
		return nil, fmt.Errorf("이미 있는 키 %s: %v", k, path)
	}
	if t.flow(target) {
		end := t.closing(t.off(target.Pos))
		at := end
		for at > 0 && src[at-1] == ' ' {
			at--
		}
		sep := ", "
		if len(target.Entries) == 0 {
			sep = " "
		}
		return splice(src, at, end, sep+k+": "+v+" "), nil
	}
	if len(target.Entries) == 0 {
		return nil, fmt.Errorf("빈 블록 맵에는 더하지 않는다: %v", path)
	}
	indent := target.Entries[0].Pos.Col - 1
	at := t.blockEnd(target.Entries[0].Pos.Line, indent)
	return t.insertLine(at, spaces(indent)+k+": "+v), nil
}
```

`blockEnd`는 첫 항목 줄부터 들여쓰기 `indent` 이상인 줄을 노드 내용으로 본다. 블록 맵의 `Pos`가 첫 항목 위치이므로 들여쓰기는 첫 항목 열에서 얻는다.

- [ ] **Step 4: 통과 확인** `go test ./internal/edit`
- [ ] **Step 5: 커밋** `feat: edit.Pin(노드에 고정 키를 흐름·블록 서식대로 더함)`

### Task 3: rename 종류 표와 바꿀 자리 모으기

**Files:**
- Create: `internal/rename/kinds.go`, `internal/rename/occur.go`
- Test: `internal/rename/occur_test.go`

**Interfaces:**
- Consumes: `edit.Replacement`
- Produces:
  - `type Kind struct { Name string; Pin PinKind }`, `type PinKind int` 상수 `PinNone`, `PinGoName`, `PinGoType`, `PinPath`, `PinImpossible`. `func KindOf(name string) (Kind, bool)`.
  - `func Occurrences(m *model.Model, kind, old, new string) ([]edit.Replacement, error)`: 정의 키와 참조 자리. 옛 ID가 없으면 오류.
  - `func retype(s, old, new string) string`: 타입 문자열에서 식별자 토큰이 정확히 `old`인 것만 바꾼다.

종류와 참조의 대응은 `check/refs.go`의 해석을 따른다.

| 종류 | 정의 위치 | 참조 |
|---|---|---|
| requirements | `Requirement.Pos` | 모든 노드의 `Satisfies` |
| actors | `Actor.Pos` | `Interface.UsedBy` |
| interfaces | `Interface.Pos` | `Executable.Exposes`, `Module.Implements` |
| executables | `Executable.Pos` | 없음 |
| tiers | `Model.Tiers[i].Pos` | `Module.Tier`, `Group.Tier` |
| modules, groups, members | 각 `Pos` | `Executable.Composes`, `Module.DependsOn`, `Group.DependsOn`, `Exception.DependsOn`, `EventKind.Subscribers`, `Event.Publishers`, `Event.Subscribers`, 그리고 modules·members는 `Owned.In` |
| enums, types | 각 `Pos` | 타입 문자열(`Field.Type`, `Param.Type`, `Method.Returns`, `HandlerKind.Returns`), types는 `Field.RefType`, enums는 `HandlerKind.Key`, `EventKind.Key` |
| consts | `Pos` | `Field.Max` |
| errors | `Pos` | `Method.Errors`, `Func.Errors`, `Case.Error` |
| contracts | `Pos` | `Need.Contract`, `Func.Of`·`Step.Calls`의 `계약.` 앞부분 |
| handler_kinds | `Pos` | `Group.Kind`, `Func.Kind` |
| event_kinds | `Pos` | `Event.Kind` |
| events | `Pos` | `Flow.On`, `Step.Emits` |
| funcs, flows | `Pos` | 없음 |

필드는 타입(`Owned`의 모든 L3·L4 노드, 이벤트 data)마다 모두 본다.

- [ ] **Step 1: 실패하는 테스트**

```go
package rename

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

func load(t *testing.T, src string) *model.Model {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(src), 0o644)
	m, ds := model.Load(dir)
	if ds.HasErrors() {
		t.Fatalf("%v", ds)
	}
	return m
}

const sample = `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  post: { tier: a }
  web: { tier: b, depends_on: [post] }
types:
  post: { in: post, fields: { p: "[]*post", s: post_status, m: "map[string]post" } }
  page: { in: web, fields: { first: { ref: post } } }
enums:
  post_status: { in: post, values: [post, draft] }
`

func places(reps []string) string { sort.Strings(reps); return strings.Join(reps, " ") }

func TestOccurrencesTypeOnly(t *testing.T) {
	m := load(t, sample)
	reps, err := Occurrences(m, "types", "post", "article")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range reps {
		got = append(got, r.Old+"->"+r.New)
	}
	want := "[]*post->[]*article map[string]post->map[string]article post->article post->article"
	if places(got) != want {
		t.Fatalf("got %s", places(got))
	}
}

func TestOccurrencesModuleOnly(t *testing.T) {
	m := load(t, sample)
	reps, err := Occurrences(m, "modules", "post", "core")
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 5 {
		t.Fatalf("정의 1, depends_on 1, in 3이어야 한다: %+v", reps)
	}
}

func TestOccurrencesMissing(t *testing.T) {
	if _, err := Occurrences(load(t, sample), "types", "nope", "x"); err == nil {
		t.Fatal("없는 ID를 받았다")
	}
}

func TestRetype(t *testing.T) {
	for in, want := range map[string]string{
		"post": "article", "[]*post": "[]*article", "map[post]post_status": "map[article]post_status",
		"post_status": "post_status", "map[string][]*post": "map[string][]*article",
	} {
		if got := retype(in, "post", "article"); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}
```

- [ ] **Step 2: 실패 확인** `go test ./internal/rename` → `undefined: Occurrences`

- [ ] **Step 3: 구현** `kinds.go`

```go
// Package rename은 설계 노드 하나의 ID와 그 참조를 설계 파일 전체에서 바꾼다.
// 기본은 코드 이름을 고정 키로 유지하고, 생성 결과와 진단이 그대로인지 확인한 뒤 쓴다.
package rename

// PinKind는 코드 이름을 옛 값으로 고정하는 방법이다.
type PinKind int

const (
	PinNone       PinKind = iota // 코드와 무관
	PinGoName                    // go: { name: ... }
	PinGoType                    // go: { type: ... }
	PinPath                      // path: ...
	PinImpossible                // 형식에 코드 이름 키가 없음
)

// Kind는 이름을 바꿀 수 있는 노드 종류다. Name은 설계 파일의 키다.
type Kind struct {
	Name string
	Pin  PinKind
}

var kinds = map[string]PinKind{
	"requirements": PinNone, "actors": PinNone, "interfaces": PinNone, "tiers": PinNone,
	"types": PinGoName, "contracts": PinGoName, "handler_kinds": PinGoName, "funcs": PinGoName, "errors": PinGoName, "consts": PinGoName,
	"enums": PinGoType, "modules": PinPath, "groups": PinPath,
	"events": PinImpossible, "event_kinds": PinImpossible, "flows": PinImpossible, "executables": PinImpossible, "members": PinImpossible,
}

// KindOf는 설계 파일 키로 종류를 찾는다.
func KindOf(name string) (Kind, bool) {
	p, ok := kinds[name]
	return Kind{name, p}, ok
}
```

`occur.go`

```go
package rename

import (
	"fmt"
	"strings"

	"github.com/tolelom/daemok/internal/edit"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type collector struct {
	old, new string
	reps     []edit.Replacement
}

func (c *collector) at(pos yaml.Pos, old, new string) {
	c.reps = append(c.reps, edit.Replacement{Pos: pos, Old: old, New: new})
}

// ref는 이름이 옛 ID인 참조를 바꾼다.
func (c *collector) ref(r model.Ref) {
	if r.Name == c.old {
		c.at(r.Pos, c.old, c.new)
	}
}

func (c *collector) refs(rs []model.Ref) {
	for _, r := range rs {
		c.ref(r)
	}
}

// typ은 타입 문자열 안의 옛 ID 토큰을 바꾼다.
func (c *collector) typ(r model.Ref) {
	if r.Name == "" {
		return
	}
	if n := retype(r.Name, c.old, c.new); n != r.Name {
		c.at(r.Pos, r.Name, n)
	}
}

// method는 "계약.메서드" 참조의 계약 부분을 바꾼다.
func (c *collector) method(r model.Ref) {
	if ct, meth, ok := strings.Cut(r.Name, "."); ok && ct == c.old {
		c.at(r.Pos, r.Name, c.new+"."+meth)
	}
}

// retype은 타입 문자열에서 식별자 토큰이 정확히 old인 것만 new로 바꾼다. 경계는 [ ] * , 와 문자열 끝이다.
func retype(s, old, new string) string {
	var b strings.Builder
	tok := 0
	flush := func(end int) {
		if s[tok:end] == old {
			b.WriteString(new)
		} else {
			b.WriteString(s[tok:end])
		}
	}
	for i := 0; i < len(s); i++ {
		if strings.IndexByte("[]*,", s[i]) >= 0 {
			flush(i)
			b.WriteByte(s[i])
			tok = i + 1
		}
	}
	flush(len(s))
	return b.String()
}

// Occurrences는 kind의 old 노드 정의 키와 그 노드를 가리키는 모든 참조 자리를 준다.
func Occurrences(m *model.Model, kind, old, new string) ([]edit.Replacement, error) {
	def, ok := definition(m, kind, old)
	if !ok {
		return nil, fmt.Errorf("없는 ID: %s %q", kind, old)
	}
	c := &collector{old: old, new: new}
	c.at(def, old, new)
	switch kind {
	case "requirements":
		eachMeta(m, func(meta *model.Meta) { c.refs(meta.Satisfies) })
	case "actors":
		if m.System != nil {
			for _, it := range m.System.Interfaces {
				c.refs(it.UsedBy)
			}
		}
	case "interfaces":
		for _, e := range m.Executables {
			c.refs(e.Exposes)
		}
		for _, mod := range m.Modules {
			c.refs(mod.Implements)
		}
	case "tiers":
		for _, mod := range m.Modules {
			c.ref(mod.Tier)
		}
		for _, g := range m.Groups {
			c.ref(g.Tier)
		}
	case "modules", "groups", "members":
		deps(m, c)
		if kind != "groups" {
			eachOwned(m, func(o *model.Owned) { c.ref(o.In) })
		}
	case "enums", "types":
		eachFields(m, func(f *model.Field) {
			c.typ(f.Type)
			if kind == "types" {
				c.ref(f.RefType)
			}
		})
		for _, ct := range m.Contracts {
			for _, me := range ct.Methods {
				for _, p := range me.Params {
					c.typ(p.Type)
				}
				for _, r := range me.Returns {
					c.typ(r)
				}
			}
		}
		for _, hk := range m.HandlerKinds {
			for _, p := range hk.Params {
				c.typ(p.Type)
			}
			for _, r := range hk.Returns {
				c.typ(r)
			}
			if kind == "enums" {
				c.ref(hk.Key)
			}
		}
		if kind == "enums" {
			for _, ek := range m.EventKinds {
				c.ref(ek.Key)
			}
		}
	case "consts":
		eachFields(m, func(f *model.Field) { c.ref(f.Max) })
	case "errors":
		for _, ct := range m.Contracts {
			for _, me := range ct.Methods {
				c.refs(me.Errors)
			}
		}
		for _, f := range m.Funcs {
			c.refs(f.Errors)
			for _, cs := range f.Cases {
				c.ref(cs.Error)
			}
		}
	case "contracts":
		for _, mod := range m.Modules {
			needs(mod.Needs, c)
		}
		for _, g := range m.Groups {
			needs(g.Needs, c)
			for _, mem := range g.Members {
				needs(mem.Needs, c)
			}
		}
		for _, f := range m.Funcs {
			c.method(f.Of)
		}
		for _, fl := range m.Flows {
			for _, s := range fl.Steps {
				for _, r := range s.Calls {
					c.method(r)
				}
			}
		}
	case "handler_kinds":
		for _, g := range m.Groups {
			c.ref(g.Kind)
		}
		for _, f := range m.Funcs {
			c.ref(f.Kind)
		}
	case "event_kinds":
		for _, e := range m.Events {
			c.ref(e.Kind)
		}
	case "events":
		for _, fl := range m.Flows {
			c.ref(fl.On)
			for _, s := range fl.Steps {
				c.refs(s.Emits)
			}
		}
	}
	return c.reps, nil
}

func needs(ns []model.Need, c *collector) {
	for _, n := range ns {
		c.ref(n.Contract)
	}
}

// deps는 모듈·그룹·멤버를 함께 가리키는 참조다. 셋은 ID가 겹치지 않는다.
func deps(m *model.Model, c *collector) {
	for _, e := range m.Executables {
		c.refs(e.Composes)
	}
	for _, mod := range m.Modules {
		c.refs(mod.DependsOn)
	}
	for _, g := range m.Groups {
		c.refs(g.DependsOn)
		for _, mem := range g.Members {
			for _, x := range mem.Exceptions {
				c.ref(x.DependsOn)
			}
		}
	}
	for _, ek := range m.EventKinds {
		c.refs(ek.Subscribers)
	}
	for _, e := range m.Events {
		c.refs(e.Publishers)
		c.refs(e.Subscribers)
	}
}
```

`definition`, `eachMeta`, `eachOwned`, `eachFields`는 `occur.go`에 같이 둔다.

```go
// definition은 kind의 id 노드의 ID 키 위치다.
func definition(m *model.Model, kind, id string) (yaml.Pos, bool) {
	var found *model.Meta
	eachKind(m, kind, func(meta *model.Meta) {
		if meta.ID == id && found == nil {
			found = meta
		}
	})
	if kind == "tiers" {
		for _, t := range m.Tiers {
			if t.Name == id {
				return t.Pos, true
			}
		}
	}
	if found == nil {
		return yaml.Pos{}, false
	}
	return found.Pos, true
}

// eachKind는 kind의 노드마다 Meta를 준다. tiers는 Meta가 없어 주지 않는다.
func eachKind(m *model.Model, kind string, fn func(*model.Meta)) {
	switch kind {
	case "requirements":
		for _, x := range m.Requirements {
			fn(&x.Meta)
		}
	case "actors", "interfaces":
		if m.System == nil {
			return
		}
		if kind == "actors" {
			for _, x := range m.System.Actors {
				fn(&x.Meta)
			}
		} else {
			for _, x := range m.System.Interfaces {
				fn(&x.Meta)
			}
		}
	case "executables":
		for _, x := range m.Executables {
			fn(&x.Meta)
		}
	case "modules":
		for _, x := range m.Modules {
			fn(&x.Meta)
		}
	case "groups":
		for _, x := range m.Groups {
			fn(&x.Meta)
		}
	case "members":
		for _, g := range m.Groups {
			for _, x := range g.Members {
				fn(&x.Meta)
			}
		}
	case "enums":
		for _, x := range m.Enums {
			fn(&x.Meta)
		}
	case "consts":
		for _, x := range m.Consts {
			fn(&x.Meta)
		}
	case "errors":
		for _, x := range m.Errors {
			fn(&x.Meta)
		}
	case "types":
		for _, x := range m.Types {
			fn(&x.Meta)
		}
	case "contracts":
		for _, x := range m.Contracts {
			fn(&x.Meta)
		}
	case "handler_kinds":
		for _, x := range m.HandlerKinds {
			fn(&x.Meta)
		}
	case "event_kinds":
		for _, x := range m.EventKinds {
			fn(&x.Meta)
		}
	case "events":
		for _, x := range m.Events {
			fn(&x.Meta)
		}
	case "funcs":
		for _, x := range m.Funcs {
			fn(&x.Meta)
		}
	case "flows":
		for _, x := range m.Flows {
			fn(&x.Meta)
		}
	}
}

func eachMeta(m *model.Model, fn func(*model.Meta)) {
	for k := range kinds {
		eachKind(m, k, fn)
	}
	if m.System != nil && m.System.ID != "" {
		fn(&m.System.Meta)
	}
}

func eachOwned(m *model.Model, fn func(*model.Owned)) {
	for _, x := range m.Enums {
		fn(&x.Owned)
	}
	for _, x := range m.Consts {
		fn(&x.Owned)
	}
	for _, x := range m.Errors {
		fn(&x.Owned)
	}
	for _, x := range m.Types {
		fn(&x.Owned)
	}
	for _, x := range m.Contracts {
		fn(&x.Owned)
	}
	for _, x := range m.HandlerKinds {
		fn(&x.Owned)
	}
	for _, x := range m.EventKinds {
		fn(&x.Owned)
	}
	for _, x := range m.Funcs {
		fn(&x.Owned)
	}
	for _, x := range m.Flows {
		fn(&x.Owned)
	}
}

func eachFields(m *model.Model, fn func(*model.Field)) {
	for _, t := range m.Types {
		for _, f := range t.Fields {
			fn(f)
		}
	}
	for _, e := range m.Events {
		for _, f := range e.Data {
			fn(f)
		}
	}
}
```

- [ ] **Step 4: 통과 확인** `go test ./internal/rename`
- [ ] **Step 5: 커밋** `feat: rename 종류 표와 모델에서 바꿀 자리 모으기`

### Task 4: rename.Run(검증, 고정, 쓰기, 되돌리기)

**Files:**
- Create: `internal/rename/run.go`
- Test: `internal/rename/run_test.go`

**Interfaces:**
- Consumes: `Occurrences`, `KindOf`, `edit.Replace`, `edit.Pin`, `workspace.RunSources`, `model.LoadSources`, `check.Run`
- Produces:
  - `type Options struct { Code, Confined bool }`
  - `type Result struct { Files []string; Pinned string; Moves []string; Notes []string }`: 고친 파일, 더한 고정 키(`go.name: Post` 같은 글), 옮길 디렉토리(`cmd/a → cmd/b`), 안내.
  - `func Run(designDir, root, kind, old, new string, opt Options) (*Result, error)`

동작 순서:

1. `KindOf(kind)`가 없으면 오류. `PinImpossible`이고 `!opt.Code`면 거부(`--code`를 안내).
2. 설계 파일(`designDir/*.yaml`)을 읽어 `model.LoadSources`. 읽기 오류가 있으면 거부.
3. `Occurrences`. 고정 값 계산(바꾸기 전 모델에서):
   - `PinGoName`: 노드에 `go.name`이 이미 있으면 고정 없음. 에러는 옛 ID, 그 밖은 `model.GoName(old)`.
   - `PinGoType`: `go.type`이 있으면 없음. 아니면 `model.GoName(old)`.
   - `PinPath`: `path` 키가 원문에 있으면 없음(모델은 비면 ID로 채우므로 `Path != ID`이거나 원문 노드에 `path` 키가 있는지로 본다. 원문 확인은 `edit.Pin`이 이미 있는 키를 거부하는 것을 이용해 `HasKey`로 따로 두지 않고, 모델의 `Path == old`일 때만 고정한다).
   - `--code`이고 노드가 `gen: false`가 아니면 고정 없음. `gen: false`면 고정하고 Notes에 이유를 적는다.
   - executables·members는 `--code`면 Moves에 `cmd/<옛> → cmd/<새>`, `<그룹 경로>/<옛> → <그룹 경로>/<새>`.
4. 파일마다 `edit.Replace`(그 파일의 Replacement만). 고정이 있으면 노드 정의 파일에 `edit.Pin`(path는 `[kind, new]`, members는 `["groups", 그룹 ID, "members", new]`).
5. 검증: 바꾼 내용으로 `model.LoadSources`의 오류가 늘면 거부. `check.Run` 진단을 `Level + Msg`(옛 ID를 새 ID로 바꾼 메시지) 수로 세어 어느 하나라도 늘면 거부. `!opt.Code`면 `workspace.RunSources` 전후의 `Files`(경로·내용) 비교, 다르면 거부.
6. `opt.Confined`면 고칠 파일마다 `edit` 패키지의 설계 디렉토리 확인을 거친다. 이를 위해 `edit.Inside(dir, path string) error`를 공개한다(Task 4에서 `inside`를 `Inside`로 이름 바꿈, `apply.go` 호출부 함께).
7. 쓰기: 파일마다 `edit.WriteAtomic`(Task 4에서 `writeAtomic`을 공개). 실패하면 이미 쓴 파일을 원래 내용으로 다시 쓰고 오류.

- [ ] **Step 1: 실패하는 테스트** `run_test.go`

```go
package rename

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/workspace"
)

// basicCopy는 생성기 테스트 설계를 임시 디렉토리로 복사한다.
func basicCopy(t *testing.T) (design, root string) {
	t.Helper()
	root = t.TempDir()
	design = filepath.Join(root, "design")
	os.MkdirAll(design, 0o755)
	src := filepath.Join("..", "gen", "golang", "testdata", "basic", "design")
	ents, _ := os.ReadDir(src)
	for _, e := range ents {
		b, _ := os.ReadFile(filepath.Join(src, e.Name()))
		os.WriteFile(filepath.Join(design, e.Name()), b, 0o644)
	}
	return design, root
}

func genFiles(t *testing.T, design, root string) map[string]string {
	t.Helper()
	r := workspace.Run(design, root)
	out := map[string]string{}
	for _, f := range r.Files {
		out[f.Path] = string(f.Content)
	}
	return out
}

// 종류마다 기본 모드로 바꾸면 생성 결과가 그대로다.
func TestRunKeepsGeneratedCode(t *testing.T) {
	cases := [][3]string{
		{"types", "account", "acct"},
		{"enums", "status", "acct_status"},
		{"modules", "core", "base"},
		{"errors", "ErrNotFound", "ErrMissing"},
		{"contracts", "ledger", "book"},
		{"tiers", "service", "svc"},
	}
	for _, c := range cases {
		t.Run(c[0]+"/"+c[1], func(t *testing.T) {
			design, root := basicCopy(t)
			before := genFiles(t, design, root)
			if _, err := Run(design, root, c[0], c[1], c[2], Options{}); err != nil {
				t.Fatal(err)
			}
			after := genFiles(t, design, root)
			if len(before) == 0 || len(before) != len(after) {
				t.Fatalf("생성 파일 수 %d → %d", len(before), len(after))
			}
			for p, b := range before {
				if after[p] != b {
					t.Fatalf("%s가 바뀌었다", p)
				}
			}
		})
	}
}

func TestRunImpossibleNeedsCode(t *testing.T) {
	design, root := basicCopy(t)
	if _, err := Run(design, root, "flows", "produce_block", "make_block", Options{}); err == nil || !strings.Contains(err.Error(), "--code") {
		t.Fatalf("%v", err)
	}
}

func TestRunCodeRenamesGenerated(t *testing.T) {
	design, root := basicCopy(t)
	if _, err := Run(design, root, "types", "account", "acct", Options{Code: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(mapValues(genFiles(t, design, root)), ""), "type Acct struct") {
		t.Fatal("생성 코드가 새 이름을 따르지 않는다")
	}
}

func TestRunRefusesCollision(t *testing.T) {
	design, root := basicCopy(t)
	before := readAll(t, design)
	if _, err := Run(design, root, "types", "account", "address", Options{}); err == nil {
		t.Fatal("겹치는 ID를 받았다")
	}
	if readAll(t, design) != before {
		t.Fatal("거부했는데 파일이 바뀌었다")
	}
}

func TestRunRollsBack(t *testing.T) {
	design, root := basicCopy(t)
	before := readAll(t, design)
	failAfter = 1 // 두 번째 파일 쓰기에서 실패
	defer func() { failAfter = -1 }()
	if _, err := Run(design, root, "modules", "core", "base", Options{}); err == nil {
		t.Skip("설계 파일이 하나뿐이면 두 번째 쓰기가 없다")
	}
	if readAll(t, design) != before {
		t.Fatal("되돌리지 않았다")
	}
}

func mapValues(m map[string]string) []string {
	var out []string
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func readAll(t *testing.T, dir string) string {
	t.Helper()
	ents, _ := os.ReadDir(dir)
	var b strings.Builder
	for _, e := range ents {
		c, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		b.WriteString(e.Name() + "\n" + string(c))
	}
	return b.String()
}
```

`basic` 설계의 실제 ID(`account`, `status`, `core`, `ErrNotFound`, `ledger`, `service`, `address`, `produce_block`)는 구현 전에 `testdata/basic/design/basic.yaml`에서 확인하고, 다르면 테스트의 ID를 그 파일의 것으로 바꾼다. 설계 파일이 하나면 되돌리기 테스트용으로 `basicCopy`가 두 번째 파일(`extra.yaml`, 바꾸는 ID를 참조하는 모듈 하나)을 만든다.

- [ ] **Step 2: 실패 확인** `go test ./internal/rename -run TestRun` → `undefined: Run`

- [ ] **Step 3: 구현** `run.go`. 위 동작 순서대로 쓴다. 테스트용 실패 주입은 `var failAfter = -1`(쓴 파일 수가 이 값이 되면 다음 쓰기를 실패로 본다).

- [ ] **Step 4: 통과 확인** `go vet ./... && go test ./...`
- [ ] **Step 5: 커밋** `feat: rename.Run(기본은 코드 이름 고정, 진단·생성 결과 확인, 실패 시 되돌리기)`

### Task 5: CLI `daemok rename`

**Files:**
- Create: `cmd/daemok/rename.go`
- Modify: `cmd/daemok/main.go`(명령 등록과 사용법)
- Test: `cmd/daemok/rename_test.go`

**Interfaces:**
- Consumes: `rename.Run`, `rename.Options`, `rename.Result`
- Produces: `daemok rename <종류> <옛 ID> <새 ID> [--code] [--design D] [--root R]`. 종료 코드 0 성공, 1 거부, 2 인자 오류. 출력: `고친 파일: ...`, `고정: ...`, `옮길 디렉토리: ...`, `안내: ...`.

- [ ] **Step 1: 실패하는 테스트**: 기존 `cmd/daemok`의 명령 테스트 형식(`run(args, stdout, stderr) int` 같은 진입 함수)을 따라, basic 설계 복사본에서 `rename types account acct`가 0을 내고 출력에 `go: { name: Account }`가 있는지, 인자가 셋이 아니면 2인지 확인한다.
- [ ] **Step 2: 실패 확인**
- [ ] **Step 3: 구현**: 기존 명령의 플래그 처리(`--design` 기본값 `<root>/design`, 위치 인자 거부 규칙)를 그대로 따른다.
- [ ] **Step 4: 통과 확인**
- [ ] **Step 5: 커밋** `feat: daemok rename 명령`

### Task 6: MCP `edit`의 `rename`

**Files:**
- Modify: `internal/mcp/server.go`, `internal/mcp/tools.go`(edit 스키마에 `kind`, `code`, op 목록에 `rename`)
- Test: `internal/mcp/write_test.go`

- [ ] **Step 1: 실패하는 테스트**: `project(t)` 설계에서 `{"op":"rename","kind":"modules","id":"core","value":"base"}`가 성공하고 파일에 `base`와 `path: core`가 있는지. 심볼릭 링크가 설계 디렉토리 밖을 가리키면 `isError`.
- [ ] **Step 2: 실패 확인**
- [ ] **Step 3: 구현**: `args.Op == "rename"`이면 `rename.Run(s.design, s.root, args.Kind, args.ID, args.Value, rename.Options{Code: args.Code, Confined: true})`.
- [ ] **Step 4: 통과 확인**
- [ ] **Step 5: 커밋** `feat: MCP edit에 rename 작업`

### Task 7: 문서와 자기 설계, 실사례 확인

**Files:**
- Modify: `design/daemok.yaml`(모듈 `rename`, tier pipeline, depends_on), `README.md`(명령 목록), `docs/design/04-open-items.md`(화면 이름 바꾸기 미룬 작업), `docs/design/00-overview.md`(구조 절의 명령 목록이 README를 가리키므로 필요 없으면 그대로)
- Test: `internal/rename/real_test.go`

- [ ] **Step 1**: `real_test.go`에 tolchain(`~/src/tolchain`)과 fiber_api_server(`~/src/fiber_api_server`)가 있으면 설계를 임시 디렉토리로 복사해 종류마다 첫 노드 하나를 기본 모드로 바꾸고 생성 결과가 같은지 확인하는 테스트(클론이 없으면 건너뜀).
- [ ] **Step 2**: 테스트 실행, 실패하면 원인 수정(빠진 참조 종류는 Task 3 표에 더함).
- [ ] **Step 3**: `design/daemok.yaml`에 `rename: { tier: pipeline, path: internal/rename, depends_on: [check, edit, model, workspace, yaml] }`(실제 import에 맞춤), `mcp`의 depends_on에 `rename`. `go run ./cmd/daemok check` 오류 0.
- [ ] **Step 4**: README와 미결 문서. `tools/doc-lint.sh`.
- [ ] **Step 5: 커밋** `docs: rename 명령과 자기 설계 반영`
