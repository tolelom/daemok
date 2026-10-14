package rename

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/edit"
	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gen/ts"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// Options는 이름 바꾸기 방식이다. Code면 고정 키 없이 생성 코드 이름도 새 ID를 따른다.
// Confined면 설계 디렉토리 밖을 가리키는 심볼릭 링크 파일을 고치지 않는다(MCP).
type Options struct {
	Code, Confined bool
}

// Result는 바꾼 결과다. Pinned는 더한 고정 키, Moves는 --code에서 옮겨야 할 디렉토리다.
type Result struct {
	Files  []string
	Pinned string
	Moves  []string
	Notes  []string
}

// yamlID는 따옴표 없이 YAML에 쓸 수 있는 ID다. 종류별 형식 규칙은 설계 검사가 본다.
var yamlID = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

// yamlKeyword는 따옴표 없이 쓰면 YAML이 문자열이 아닌 값(null, 불리언)으로 읽는 이름이다.
// 참조 자리에 쓰면 참조가 빈 값이 되어 검사도 잡지 못한다.
func yamlKeyword(id string) bool {
	switch strings.ToLower(id) {
	case "null", "true", "false", "yes", "no", "on", "off":
		return true
	}
	return false
}

// fileNamed는 생성 파일 이름을 ID로 짓는 종류다.
var fileNamed = map[string]bool{"enums": true, "handler_kinds": true, "event_kinds": true, "flows": true}

var (
	knownOS   = strings.Fields("aix android darwin dragonfly freebsd hurd illumos ios js linux nacl netbsd openbsd plan9 solaris wasip1 windows zos")
	knownArch = strings.Fields("386 amd64 amd64p32 arm armbe arm64 arm64be loong64 mips mipsle mips64 mips64le mips64p32 mips64p32le ppc ppc64 ppc64le riscv riscv64 s390 s390x sparc sparc64 wasm")
)

// buildSuffix는 id로 지은 파일 이름을 go/build가 GOOS·GOARCH 제약으로 읽는지다. 규칙은 go/build의 goodOSArchFile과 같다.
func buildSuffix(id string) bool {
	l := strings.Split(id, "_")
	if n := len(l); n > 0 && l[n-1] == "test" {
		l = l[:n-1]
	}
	n := len(l)
	if n >= 2 && slices.Contains(knownOS, l[n-2]) && slices.Contains(knownArch, l[n-1]) {
		return true
	}
	return n >= 2 && (slices.Contains(knownOS, l[n-1]) || slices.Contains(knownArch, l[n-1]))
}

// failAfter는 테스트에서 쓴 파일 수가 이 값이 되면 다음 쓰기를 실패로 본다. -1이면 끈다.
var failAfter = -1

// Run은 designDir의 설계에서 kind의 old 노드 ID와 그 참조를 new로 바꾼다.
// 편집 전에 없던 읽기 오류, 늘어난 설계 검사 진단, 기본 모드에서 달라진 생성 결과가 있으면 쓰지 않는다.
func Run(designDir, root, kind, old, new string, opt Options) (*Result, error) {
	pin, ok := kinds[kind]
	if !ok {
		return nil, fmt.Errorf("알 수 없는 종류 %q (가능: %s)", kind, strings.Join(Kinds(), ", "))
	}
	if pin == PinImpossible && !opt.Code {
		return nil, fmt.Errorf("%s는 코드 이름을 고정할 키가 없다. 생성 코드 이름까지 바꾸려면 --code를 쓴다", kind)
	}
	if old == new {
		return nil, fmt.Errorf("옛 ID와 새 ID가 같다: %s", old)
	}
	if !yamlID.MatchString(new) || yamlKeyword(new) {
		return nil, fmt.Errorf("ID로 쓸 수 없는 이름: %q", new)
	}
	if fileNamed[kind] && buildSuffix(new) {
		return nil, fmt.Errorf("새 ID가 Go 빌드 제약 접미사(_windows, _arm64 등)로 끝나 생성 파일이 다른 플랫폼에서만 빌드된다: %s", new)
	}
	files, err := readDesign(designDir)
	if err != nil {
		return nil, err
	}
	m, diags := model.LoadSources(files)
	if diags.HasErrors() {
		return nil, fmt.Errorf("설계를 읽을 수 없어 바꾸지 않는다: %s", firstError(diags))
	}
	reps, err := Occurrences(m, kind, old, new)
	if err != nil {
		return nil, err
	}
	for _, k := range namespace(kind) {
		if exists(m, k, new) {
			return nil, fmt.Errorf("이미 있는 ID: %s %q", k, new)
		}
	}
	res := &Result{}
	pinKey, pinSub, pinValue := pinFor(m, kind, old, pin, opt, res)
	if opt.Code {
		res.Moves = moves(m, kind, old, new)
	}

	byFile := map[string][]edit.Replacement{}
	for _, r := range reps {
		byFile[r.Pos.File] = append(byFile[r.Pos.File], r)
	}
	out := map[string][]byte{}
	for f, rs := range byFile {
		b, err := edit.Replace(files[f], f, rs)
		if err != nil {
			return nil, err
		}
		out[f] = b
	}
	if pinKey != "" {
		def, _ := definition(m, kind, old)
		b, err := edit.Pin(out[def.File], def.File, pinPath(m, kind, old, new), pinKey, pinSub, pinValue)
		switch {
		case errors.Is(err, edit.ErrKeyExists):
			// 경로를 ID와 같은 값으로 적어 둔 노드는 그 키가 이미 고정이다.
		case err != nil:
			return nil, err
		default:
			out[def.File] = b
			res.Pinned = pinText(pinKey, pinSub, pinValue)
		}
	}

	after := map[string][]byte{}
	for f, b := range files {
		after[f] = b
	}
	for f, b := range out {
		after[f] = b
	}
	renamed, err := verify(files, after, old, new, designDir, root, opt.Code)
	if err != nil {
		return nil, err
	}
	res.Notes = append(res.Notes, renamed...)
	for f := range out {
		if opt.Confined {
			if err := edit.Inside(designDir, f); err != nil {
				return nil, err
			}
		}
		res.Files = append(res.Files, f)
	}
	sort.Strings(res.Files)
	if err := writeAll(res.Files, out, files); err != nil {
		return nil, err
	}
	return res, nil
}

func readDesign(dir string) (map[string][]byte, error) {
	paths, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if len(paths) == 0 {
		return nil, fmt.Errorf("설계 파일(*.yaml)이 없다: %s", dir)
	}
	files := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", edit.ErrIO, err)
		}
		files[p] = b
	}
	return files, nil
}

func firstError(ds model.Diagnostics) string {
	for _, d := range ds {
		if d.Level == model.Error {
			return d.Pos.String() + ": " + d.Msg
		}
	}
	return ""
}

// pinFor는 더할 고정 키다. 코드 이름 키가 이미 있거나 --code(gen: false 제외)면 비운다.
func pinFor(m *model.Model, kind, old string, pin PinKind, opt Options, res *Result) (key, sub, value string) {
	var goOpts *model.GoOpts
	gen := true
	switch kind {
	case "types":
		x := m.Type(old)
		goOpts, gen = &x.Go, x.Gen
	case "contracts":
		x := m.Contract(old)
		goOpts, gen = &x.Go, x.Gen
	case "handler_kinds":
		x := m.HandlerKind(old)
		goOpts, gen = &x.Go, x.Gen
	case "funcs":
		x := m.Func(old)
		goOpts, gen = &x.Go, x.Gen
		if x.Kind.Name != "" || x.Of.Name != "" {
			// go.name은 일반 함수에만 쓴다. 핸들러·계약 함수의 코드 이름은 생성 결과 비교가 지킨다.
			return "", "", ""
		}
	case "errors":
		x := m.Error(old)
		goOpts, gen = &x.Go, x.Gen
	case "consts":
		x := m.Const(old)
		goOpts, gen = &x.Go, x.Gen
	case "enums":
		x := m.Enum(old)
		goOpts, gen = &x.Go, x.Gen
	}
	if opt.Code && gen {
		return "", "", ""
	}
	switch pin {
	case PinGoName:
		if goOpts.Name != "" {
			return "", "", ""
		}
		value = model.GoName(old)
		if kind == "errors" {
			value = old
		}
		key, sub = "go", "name"
	case PinGoType:
		if goOpts.Type != "" {
			return "", "", ""
		}
		key, sub, value = "go", "type", model.GoName(old)
	case PinPath:
		path := ""
		if kind == "modules" {
			path = m.Module(old).Path
		} else {
			path = m.Group(old).Path
		}
		if path != old {
			return "", "", ""
		}
		key, value = "path", old
	default:
		return "", "", ""
	}
	if opt.Code {
		res.Notes = append(res.Notes, "gen: false 노드라 손으로 쓴 선언의 이름을 바꿀 수 없어 코드 이름을 고정했다")
	}
	return key, sub, value
}

func pinText(key, sub, value string) string {
	if sub == "" {
		return key + ": " + value
	}
	return key + ": { " + sub + ": " + value + " }"
}

// pinPath는 바꾼 뒤 노드까지의 키 목록이다.
func pinPath(m *model.Model, kind, old, new string) []string {
	if kind == "members" {
		return []string{"groups", m.Member(old).Group.ID, "members", new}
	}
	return []string{kind, new}
}

// moves는 --code에서 경로가 ID에서 나오는 노드의 옮길 디렉토리다. 손으로 쓴 파일은 daemok이 옮기지 않는다.
func moves(m *model.Model, kind, old, new string) []string {
	switch kind {
	case "executables":
		return []string{"cmd/" + old + " → cmd/" + new}
	case "members":
		p := m.Member(old).Group.Path
		return []string{p + "/" + old + " → " + p + "/" + new}
	case "modules":
		if m.Module(old).Path == old {
			return []string{old + " → " + new}
		}
	case "groups":
		if m.Group(old).Path == old {
			return []string{old + " → " + new}
		}
	}
	return nil
}

// verify는 바꾼 설계가 읽기 오류나 검사 진단을 늘리지 않는지, 기본 모드면 생성 코드가 같은지 본다.
// 기본 모드에서 이름만 바뀌는 생성 파일 목록을 준다.
func verify(before, after map[string][]byte, old, new, designDir, root string, code bool) ([]string, error) {
	bm, bd := model.LoadSources(before)
	am, ad := model.LoadSources(after)
	if ad.HasErrors() {
		return nil, fmt.Errorf("%w: %s", edit.ErrBreaksLoad, firstError(ad))
	}
	bd = append(bd, check.Run(bm)...)
	ad = append(ad, check.Run(am)...)
	count := map[string]int{}
	for _, d := range bd {
		count[d.Level.String()+" "+replaceWord(d.Msg, old, new)]++
	}
	var added []string
	for _, d := range ad {
		k := d.Level.String() + " " + d.Msg
		if count[k] == 0 {
			added = append(added, d.Pos.String()+": "+d.Msg)
		}
		count[k]--
	}
	if len(added) > 0 {
		sort.Strings(added)
		return nil, fmt.Errorf("바꾸면 설계 검사 진단이 늘어난다: %s", strings.Join(added, "; "))
	}
	if code {
		return nil, nil
	}
	bg := workspace.RunSources(before, designDir, root)
	if bg.Files == nil {
		return nil, fmt.Errorf("바꾸기 전 설계로 생성하지 못해 생성 코드가 그대로인지 확인할 수 없다: %s. 오류를 고치거나 --code를 쓴다", firstError(bg.Diags))
	}
	// TS 파일은 ts_out 디렉토리가 없으면 Files에 빠지므로 따로 비교한다. TS 이름은 고정 키가 아니라 ID에서 나온다.
	bt, _ := ts.Generate(bm)
	at, _ := ts.Generate(am)
	if !bytes.Equal(bt, at) {
		return nil, fmt.Errorf("바꾸면 TS 생성 이름이 달라진다(%s). 생성 코드 이름까지 바꾸려면 --code를 쓴다", bm.TSOut)
	}
	renamed, err := sameCode(bg.Files, workspace.RunSources(after, designDir, root).Files)
	if err != nil {
		return nil, err
	}
	return renamed, nil
}

// sameCode는 생성 파일이 패키지 디렉토리와 내용으로 같은지 본다. 파일 이름만 바뀐 것은 코드 이름에 영향이
// 없어 받고(enum, 핸들러 종류 파일은 ID로 이름을 짓는다), 그 목록을 준다.
func sameCode(before, after []golang.File) ([]string, error) {
	key := func(f golang.File) string { return path.Dir(f.Path) + "\x00" + string(f.Content) }
	left := map[string][]string{}
	for _, f := range before {
		left[key(f)] = append(left[key(f)], f.Path)
	}
	var renamed []string
	for _, f := range after {
		k := key(f)
		if len(left[k]) == 0 {
			return nil, fmt.Errorf("바꾸면 생성 코드가 달라진다: %s. 생성 코드 이름까지 바꾸려면 --code를 쓴다", f.Path)
		}
		if was := left[k][0]; was != f.Path {
			renamed = append(renamed, "생성 파일 이름이 바뀐다(daemok gen으로 반영): "+was+" → "+f.Path)
		}
		left[k] = left[k][1:]
	}
	for _, ps := range left {
		if len(ps) > 0 {
			return nil, fmt.Errorf("바꾸면 생성 파일이 사라진다: %s", ps[0])
		}
	}
	sort.Strings(renamed)
	return renamed, nil
}

// replaceWord는 s에서 앞뒤가 식별자 글자가 아닌 old만 new로 바꾼다.
func replaceWord(s, old, new string) string {
	word := func(b byte) bool {
		return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], old) && (i == 0 || !word(s[i-1])) && (i+len(old) == len(s) || !word(s[i+len(old)])) {
			b.WriteString(new)
			i += len(old)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// writeAll은 파일마다 원자적으로 쓰고, 실패하면 이미 쓴 파일을 원래 내용으로 되돌린다.
func writeAll(paths []string, out, orig map[string][]byte) error {
	for i, p := range paths {
		var err error
		if i == failAfter {
			err = errors.New("테스트용 쓰기 실패")
		} else {
			err = edit.WriteAtomic(p, out[p])
		}
		if err != nil {
			var lost []string
			for _, q := range paths[:i] {
				if rerr := edit.WriteAtomic(q, orig[q]); rerr != nil {
					lost = append(lost, q)
				}
			}
			if len(lost) > 0 {
				return fmt.Errorf("%w: %s 쓰기 실패(%v), 되돌리지 못한 파일: %s", edit.ErrIO, p, err, strings.Join(lost, ", "))
			}
			return fmt.Errorf("%w: %s 쓰기 실패(%v), 앞서 쓴 파일은 되돌렸다", edit.ErrIO, p, err)
		}
	}
	return nil
}
