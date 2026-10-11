// Package edit는 설계 파일의 L2 모듈을 줄 단위로 고친다. 파일 전체를 다시 쓰지 않고
// 파서가 남긴 노드 시작 위치에서 원본 텍스트를 읽어 해당 토큰이나 줄만 바꾼다. 주석과 정렬은 그대로다.
package edit

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"

	"github.com/tolelom/daemok/internal/yaml"
)

var snakeName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// plainID는 따옴표 없이 YAML에 그대로 쓸 수 있는 ID다. 모델은 모듈 ID를 snake_case로 강제하지 않는다.
var plainID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

var (
	// ErrModuleNotInFile은 고칠 모듈이 이 파일에 없다는 뜻이다. 다른 설계 파일에 있을 수 있다.
	ErrModuleNotInFile = errors.New("이 파일에 모듈이 없다")
	// ErrNoModuleBlock은 이 파일에 모듈을 더할 modules 블록이 없다는 뜻이다. 다른 설계 파일에 있을 수 있다.
	ErrNoModuleBlock = errors.New("모듈을 더할 modules 블록이 없다")
)

// SetTier는 모듈의 tier 값을 바꾼다.
func SetTier(src []byte, file, module, tier string) ([]byte, error) {
	t, err := open(src, file, module)
	if err != nil {
		return nil, err
	}
	if !snakeName.MatchString(tier) {
		return nil, fmt.Errorf("tier 이름이 아니다: %q", tier)
	}
	n := t.val.Get("tier")
	if n == nil || n.Kind != yaml.Scalar {
		return nil, fmt.Errorf("tier가 없다: 모듈 %s", module)
	}
	if n.Value == "" && !n.Quoted {
		at := t.afterColon(entry(t.val, "tier"))
		return splice(src, at, at, " "+tier), nil
	}
	s := t.off(n.Pos)
	return splice(src, s, t.tokenEnd(s), tier), nil
}

// AddDep는 모듈의 depends_on에 dep를 더한다. depends_on이 없으면 만든다.
func AddDep(src []byte, file, module, dep string) ([]byte, error) {
	t, err := open(src, file, module)
	if err != nil {
		return nil, err
	}
	if !plainID.MatchString(dep) {
		return nil, fmt.Errorf("모듈 ID가 아니다: %q", dep)
	}
	e := entry(t.val, "depends_on")
	if e == nil {
		if t.flow(t.val) {
			end := t.closing(t.off(t.val.Pos))
			if len(t.val.Entries) == 0 {
				return splice(src, end, end, "depends_on: ["+dep+"]"), nil
			}
			i := end
			for i > 0 && t.src[i-1] == ' ' {
				i--
			}
			return splice(src, i, i, ", depends_on: ["+dep+"]"), nil
		}
		at := t.blockEnd(t.val.Pos.Line, t.val.Pos.Col-1)
		return t.insertLine(at, spaces(t.val.Pos.Col-1)+"depends_on: ["+dep+"]"), nil
	}
	seq := e.Value
	if seq.IsNull() {
		if seq.Value == "" {
			at := t.afterColon(e)
			return splice(src, at, at, " ["+dep+"]"), nil
		}
		s := t.off(seq.Pos)
		return splice(src, s, t.tokenEnd(s), "["+dep+"]"), nil
	}
	for _, it := range seq.Items {
		if it.Value == dep {
			return nil, fmt.Errorf("이미 있는 의존: %s → %s", module, dep)
		}
	}
	if t.flow(seq) {
		if len(seq.Items) == 0 {
			at := t.off(seq.Pos) + 1
			return splice(src, at, at, dep), nil
		}
		last := t.off(seq.Items[len(seq.Items)-1].Pos)
		at := t.tokenEnd(last)
		return splice(src, at, at, ", "+dep), nil
	}
	if len(seq.Items) == 0 {
		return nil, fmt.Errorf("depends_on 형식을 알 수 없다: 모듈 %s", module)
	}
	lastLine := seq.Items[len(seq.Items)-1].Pos.Line
	line := t.line(lastLine)
	dash := bytes.Index(line, []byte("- "))
	if dash < 0 {
		return nil, fmt.Errorf("depends_on 형식을 알 수 없다: 모듈 %s", module)
	}
	at := t.lineEnd(lastLine)
	return t.insertLine(at, string(line[:dash])+"- "+dep), nil
}

// RemoveDep는 모듈의 depends_on에서 dep를 지운다. 비면 []로 남긴다.
func RemoveDep(src []byte, file, module, dep string) ([]byte, error) {
	t, err := open(src, file, module)
	if err != nil {
		return nil, err
	}
	e := entry(t.val, "depends_on")
	idx := -1
	if e != nil {
		for i, it := range e.Value.Items {
			if it.Value == dep {
				idx = i
			}
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("없는 의존: %s → %s", module, dep)
	}
	items := e.Value.Items
	if t.flow(e.Value) {
		s := t.off(items[idx].Pos)
		switch {
		case idx < len(items)-1:
			return splice(src, s, t.off(items[idx+1].Pos), ""), nil
		case idx > 0:
			return splice(src, t.tokenEnd(t.off(items[idx-1].Pos)), t.tokenEnd(s), ""), nil
		default:
			return splice(src, s, t.tokenEnd(s), ""), nil
		}
	}
	itemLine := items[idx].Pos.Line
	if len(items) == 1 {
		// 항목 줄만 지우고 키 뒤에 []를 넣는다. 키 줄과 사이의 주석은 그대로 둔다.
		at := t.afterColon(e)
		out := splice(src, t.lineStart(itemLine), t.lineEnd(itemLine), "")
		return splice(out, at, at, " []"), nil
	}
	return splice(src, t.lineStart(itemLine), t.lineEnd(itemLine), ""), nil
}

// AddModule은 modules 블록의 마지막 모듈 아래에 `id: { tier: T }` 줄을 더한다.
func AddModule(src []byte, file, id, tier string) ([]byte, error) {
	if !snakeName.MatchString(id) {
		return nil, fmt.Errorf("모듈 ID는 소문자 snake_case여야 한다: %q", id)
	}
	if !snakeName.MatchString(tier) {
		return nil, fmt.Errorf("tier 이름이 아니다: %q", tier)
	}
	t, err := parse(src, file)
	if err != nil {
		return nil, err
	}
	if t.mods == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoModuleBlock, file)
	}
	if t.mods.Kind != yaml.Map || t.flow(t.mods) || len(t.mods.Entries) == 0 {
		return nil, fmt.Errorf("%w(모듈이 하나 이상 있는 블록 형식이어야 한다): %s", ErrNoModuleBlock, file)
	}
	if t.mods.Get(id) != nil {
		return nil, fmt.Errorf("이미 있는 모듈: %s", id)
	}
	indent := t.mods.Pos.Col - 1
	at := t.blockEnd(t.mods.Pos.Line, indent)
	return t.insertLine(at, spaces(indent)+id+": { tier: "+tier+" }"), nil
}

type text struct {
	src    []byte
	starts []int // 줄마다 시작 바이트. BOM이 있으면 첫 줄은 BOM 뒤에서 시작한다(파서가 BOM을 떼고 열을 센다).
	mods   *yaml.Node
	val    *yaml.Node // 고칠 모듈의 값
}

func parse(src []byte, file string) (*text, error) {
	root, err := yaml.Parse(file, src)
	if err != nil {
		return nil, err
	}
	t := &text{src: src, starts: []int{0}, mods: root.Get("modules")}
	if bytes.HasPrefix(src, []byte("\xef\xbb\xbf")) {
		t.starts[0] = 3
	}
	for i, b := range src {
		if b == '\n' {
			t.starts = append(t.starts, i+1)
		}
	}
	return t, nil
}

func open(src []byte, file, module string) (*text, error) {
	t, err := parse(src, file)
	if err != nil {
		return nil, err
	}
	t.val = t.mods.Get(module)
	if t.val == nil || t.val.Kind != yaml.Map {
		return nil, fmt.Errorf("%w: %s", ErrModuleNotInFile, module)
	}
	return t, nil
}

func entry(n *yaml.Node, key string) *yaml.Entry {
	for i := range n.Entries {
		if n.Entries[i].Key == key {
			return &n.Entries[i]
		}
	}
	return nil
}

func (t *text) off(p yaml.Pos) int { return t.starts[p.Line-1] + p.Col - 1 }

func (t *text) lineStart(line int) int { return t.starts[line-1] }

// lineEnd는 줄바꿈까지 포함한 줄의 끝이다.
func (t *text) lineEnd(line int) int {
	if line < len(t.starts) {
		return t.starts[line]
	}
	return len(t.src)
}

func (t *text) line(line int) []byte {
	return bytes.TrimRight(t.src[t.lineStart(line):t.lineEnd(line)], "\r\n")
}

// flow는 노드가 { 나 [ 로 시작하는 플로우 형식인지다.
func (t *text) flow(n *yaml.Node) bool {
	o := t.off(n.Pos)
	return o < len(t.src) && (t.src[o] == '{' || t.src[o] == '[')
}

// tokenEnd는 s에서 시작하는 스칼라의 끝이다. 따옴표 문자열은 닫는 따옴표 뒤다.
func (t *text) tokenEnd(s int) int {
	if t.src[s] == '"' {
		for i := s + 1; i < len(t.src); i++ {
			switch t.src[i] {
			case '\\':
				i++
			case '"':
				return i + 1
			}
		}
		return len(t.src)
	}
	i := s
	for i < len(t.src) && !bytes.ContainsRune([]byte(",]}#\r\n \t"), rune(t.src[i])) {
		i++
	}
	return i
}

// closing은 open 위치의 { 나 [ 와 짝이 맞는 닫는 괄호 위치다.
func (t *text) closing(open int) int {
	depth := 0
	for i := open; i < len(t.src); i++ {
		switch t.src[i] {
		case '"':
			i = t.tokenEnd(i) - 1
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return len(t.src)
}

// blockEnd는 first 줄에서 시작하는 들여쓰기 indent 이상의 블록에서 마지막 내용 줄의 끝이다.
// 빈 줄과 주석 줄은 블록을 끝내지 않지만 내용으로 세지 않는다.
func (t *text) blockEnd(first, indent int) int {
	last := first
	for ln := first + 1; ln <= len(t.starts); ln++ {
		l := t.line(ln)
		trimmed := bytes.TrimLeft(l, " ")
		if len(trimmed) == 0 || trimmed[0] == '#' {
			continue
		}
		if len(l)-len(trimmed) < indent {
			break
		}
		last = ln
	}
	return t.lineEnd(last)
}

// insertLine은 at에 한 줄을 넣는다. 파일 끝에 줄바꿈이 없으면 먼저 줄을 바꾼다.
// 줄바꿈은 넣는 자리 앞줄의 것을 따른다.
func (t *text) insertLine(at int, line string) []byte {
	nl := "\n"
	if i := bytes.LastIndexByte(t.src[:at], '\n'); i > 0 && t.src[i-1] == '\r' {
		nl = "\r\n"
	}
	if at == len(t.src) && len(t.src) > 0 && t.src[len(t.src)-1] != '\n' {
		line = nl + line
	}
	return splice(t.src, at, at, line+nl)
}

// afterColon은 항목 키 뒤 콜론의 바로 다음 위치다.
func (t *text) afterColon(e *yaml.Entry) int {
	i := t.off(e.Pos)
	if t.src[i] == '"' {
		i = t.tokenEnd(i)
	} else {
		i += len(e.Key)
	}
	for i < len(t.src) && t.src[i] != ':' {
		i++
	}
	return i + 1
}

func splice(src []byte, s, e int, repl string) []byte {
	out := make([]byte, 0, len(src)+len(repl))
	out = append(out, src[:s]...)
	out = append(out, repl...)
	return append(out, src[e:]...)
}

func spaces(n int) string { return string(bytes.Repeat([]byte(" "), n)) }
