package rename

import (
	"maps"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// basicCopy는 생성기 테스트 설계를 임시 디렉토리로 복사한다.
func basicCopy(t *testing.T) (design, root string) {
	t.Helper()
	root = t.TempDir()
	design = filepath.Join(root, "design")
	if err := os.MkdirAll(design, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join("..", "gen", "golang", "testdata", "basic", "design", "basic.yaml")
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(design, "basic.yaml"), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return design, root
}

// genFiles는 생성 파일을 패키지 디렉토리와 내용으로 센다. 파일 이름은 코드 이름에 영향이 없다.
func genFiles(t *testing.T, design, root string) map[string]int {
	t.Helper()
	r := workspace.Run(design, root)
	if r.Fatal || r.Files == nil {
		t.Fatalf("생성하지 못했다: %v", r.Diagnostics)
	}
	out := map[string]int{}
	for _, f := range r.Files {
		out[path.Dir(f.Path)+"\n"+string(f.Content)]++
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

// 종류마다 기본 모드로 바꾸면 생성 결과가 그대로다.
func TestRunKeepsGeneratedCode(t *testing.T) {
	cases := []struct{ kind, old, new, pin string }{
		{"types", "account", "acct", "go: { name: Account }"},
		{"enums", "status", "acct_status", "go: { type: Status }"},
		{"enums", "tx_type", "tx_kind", ""},
		{"modules", "core", "base", "path: core"},
		{"groups", "tx_modules", "txs", ""},
		{"errors", "ErrNotFound", "ErrMissing", "go: { name: ErrNotFound }"},
		{"contracts", "ledger", "book", "go: { name: Ledger }"},
		{"consts", "max_tx_age", "tx_ttl", "go: { name: MaxTxAge }"},
		{"handler_kinds", "tx_handler", "tx_fn", ""},
		{"tiers", "domain", "core_tier", ""},
	}
	for _, c := range cases {
		t.Run(c.kind+"/"+c.old, func(t *testing.T) {
			design, root := basicCopy(t)
			before := genFiles(t, design, root)
			res, err := Run(design, root, c.kind, c.old, c.new, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Pinned != c.pin {
				t.Fatalf("고정 %q, 기대 %q", res.Pinned, c.pin)
			}
			if after := genFiles(t, design, root); !maps.Equal(before, after) {
				t.Fatal("생성 코드가 바뀌었다")
			}
			m, _ := model.Load(design)
			if exists(m, c.kind, c.old) || !exists(m, c.kind, c.new) {
				t.Fatalf("ID가 바뀌지 않았다:\n%s", readAll(t, design))
			}
		})
	}
}

func TestRunImpossibleNeedsCode(t *testing.T) {
	design, root := basicCopy(t)
	before := readAll(t, design)
	_, err := Run(design, root, "flows", "produce_block", "make_block", Options{})
	if err == nil || !strings.Contains(err.Error(), "--code") {
		t.Fatalf("%v", err)
	}
	if readAll(t, design) != before {
		t.Fatal("거부했는데 파일이 바뀌었다")
	}
}

func TestRunCodeRenamesGenerated(t *testing.T) {
	design, root := basicCopy(t)
	res, err := Run(design, root, "types", "account", "acct", Options{Code: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pinned != "" {
		t.Fatalf("--code인데 고정했다: %s", res.Pinned)
	}
	var all strings.Builder
	for _, f := range workspace.Run(design, root).Files {
		all.Write(f.Content)
	}
	if !strings.Contains(all.String(), "type Acct struct") {
		t.Fatal("생성 코드가 새 이름을 따르지 않는다")
	}
}

func TestRunCodeReportsMoves(t *testing.T) {
	design, root := basicCopy(t)
	res, err := Run(design, root, "members", "economy", "money", Options{Code: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Moves, ",") != "vm/modules/economy → vm/modules/money" {
		t.Fatalf("%v", res.Moves)
	}
	res, err = Run(design, root, "executables", "node", "full_node", Options{Code: true})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(res.Moves, ",") != "cmd/node → cmd/full_node" {
		t.Fatalf("%v", res.Moves)
	}
}

func TestRunCodePinsGenFalse(t *testing.T) {
	design, root := basicCopy(t)
	p := filepath.Join(design, "basic.yaml")
	b, _ := os.ReadFile(p)
	b = []byte(strings.Replace(string(b), "status:     { in: core, values", "status:     { in: core, gen: false, values", 1))
	os.WriteFile(p, b, 0o644)
	res, err := Run(design, root, "enums", "status", "acct_status", Options{Code: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Pinned != "go: { type: Status }" || len(res.Notes) == 0 {
		t.Fatalf("%+v", res)
	}
}

func TestRunRefuses(t *testing.T) {
	cases := []struct{ kind, old, new, msg string }{
		{"types", "account", "block", "이미 있는"},
		{"types", "account", "status", "이미 있는"},
		{"modules", "core", "store", "이미 있는"},
		{"modules", "core", "economy", "이미 있는"},
		{"types", "nope", "x", "없는 ID"},
		{"types", "account", "account", "같다"},
		{"types", "account", "a b", "ID"},
		{"bogus", "a", "b", "종류"},
		{"funcs", "handle_transfer", "do_transfer", "--code"},
	}
	for _, c := range cases {
		t.Run(c.kind+"/"+c.new, func(t *testing.T) {
			design, root := basicCopy(t)
			before := readAll(t, design)
			_, err := Run(design, root, c.kind, c.old, c.new, Options{})
			if err == nil || !strings.Contains(err.Error(), c.msg) {
				t.Fatalf("%v", err)
			}
			if readAll(t, design) != before {
				t.Fatal("거부했는데 파일이 바뀌었다")
			}
		})
	}
}

// 바꾼 결과가 검사 오류를 늘리면 쓰지 않는다. 대문자로 시작하는 타입 ID는 외부 타입으로 읽혀 검사가 잡지 못하므로
// 형식 규칙을 어기는 ID(타입 ID는 snake_case)로 확인한다.
func TestRunRefusesNewDiagnostics(t *testing.T) {
	design, root := basicCopy(t)
	before := readAll(t, design)
	if _, err := Run(design, root, "types", "account", "Account", Options{}); err == nil {
		t.Fatal("검사 오류를 늘리는 이름을 받았다")
	}
	if readAll(t, design) != before {
		t.Fatal("거부했는데 파일이 바뀌었다")
	}
}

const twoA = `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  core: { tier: a }
`

const twoB = `modules:
  web: { tier: b, depends_on: [core] }
`

func twoFiles(t *testing.T) (design, root string) {
	t.Helper()
	root = t.TempDir()
	design = filepath.Join(root, "design")
	os.MkdirAll(design, 0o755)
	os.WriteFile(filepath.Join(design, "a.yaml"), []byte(twoA), 0o644)
	os.WriteFile(filepath.Join(design, "b.yaml"), []byte(twoB), 0o644)
	return design, root
}

func TestRunMultiFile(t *testing.T) {
	design, root := twoFiles(t)
	res, err := Run(design, root, "modules", "core", "base", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 2 {
		t.Fatalf("%v", res.Files)
	}
	b, _ := os.ReadFile(filepath.Join(design, "b.yaml"))
	if !strings.Contains(string(b), "depends_on: [base]") {
		t.Fatalf("%s", b)
	}
	a, _ := os.ReadFile(filepath.Join(design, "a.yaml"))
	if !strings.Contains(string(a), "base: { tier: a, path: core }") {
		t.Fatalf("%s", a)
	}
}

func TestRunRollsBack(t *testing.T) {
	design, root := twoFiles(t)
	before := readAll(t, design)
	failAfter = 1
	defer func() { failAfter = -1 }()
	if _, err := Run(design, root, "modules", "core", "base", Options{}); err == nil {
		t.Fatal("쓰기 실패를 알리지 않았다")
	}
	if readAll(t, design) != before {
		t.Fatalf("되돌리지 않았다:\n%s", readAll(t, design))
	}
}

func TestRunConfinedRefusesOutsideLink(t *testing.T) {
	design, root := twoFiles(t)
	outside := filepath.Join(t.TempDir(), "b.yaml")
	os.Rename(filepath.Join(design, "b.yaml"), outside)
	if err := os.Symlink(outside, filepath.Join(design, "b.yaml")); err != nil {
		t.Skip(err)
	}
	if _, err := Run(design, root, "modules", "core", "base", Options{Confined: true}); err == nil {
		t.Fatal("설계 디렉토리 밖을 고쳤다")
	}
	a, _ := os.ReadFile(filepath.Join(design, "a.yaml"))
	if string(a) != twoA {
		t.Fatal("거부했는데 a.yaml이 바뀌었다")
	}
}

func TestRunReportsRenamedGeneratedFile(t *testing.T) {
	design, root := basicCopy(t)
	res, err := Run(design, root, "enums", "status", "acct_status", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(res.Notes, "\n"), "core/status.gen.go → core/acct_status.gen.go") {
		t.Fatalf("%v", res.Notes)
	}
}

// 설계 검사 오류로 생성하지 못하면 기본 모드는 코드가 그대로인지 확인할 수 없어 거부한다. --code는 받는다.
func TestRunDefaultNeedsGeneratableDesign(t *testing.T) {
	design, root := basicCopy(t)
	p := filepath.Join(design, "basic.yaml")
	b, _ := os.ReadFile(p)
	os.WriteFile(p, []byte(strings.Replace(string(b), "store:  { tier: foundation", "store:  { tier: nowhere", 1)), 0o644)
	if _, err := Run(design, root, "types", "account", "acct", Options{}); err == nil || !strings.Contains(err.Error(), "확인할 수 없다") {
		t.Fatalf("%v", err)
	}
	if _, err := Run(design, root, "types", "account", "acct", Options{Code: true}); err != nil {
		t.Fatal(err)
	}
}

// YAML이 값으로 읽는 이름은 참조를 빈 값으로 만들어 받지 않는다.
func TestRunRefusesYAMLKeywords(t *testing.T) {
	for _, id := range []string{"null", "true", "false", "Null", "TRUE"} {
		design, root := basicCopy(t)
		if _, err := Run(design, root, "events", "block_commit", id, Options{Code: true}); err == nil {
			t.Fatalf("%s를 받았다", id)
		}
	}
}

// ts_out 디렉토리가 없어도 TS 이름이 바뀌면 기본 모드는 거부한다. TS 이름은 ID에서 나온다.
func TestRunComparesTSWithoutDir(t *testing.T) {
	design, root := basicCopy(t)
	p := filepath.Join(design, "basic.yaml")
	b, _ := os.ReadFile(p)
	s := strings.Replace(string(b), "go_module: example.com/basic", "go_module: example.com/basic\nts_out: web/src/gen.ts", 1)
	s = strings.Replace(s, "status:     { in: core, values", "status:     { in: core, export: [ts], values", 1)
	os.WriteFile(p, []byte(s), 0o644)
	if _, err := Run(design, root, "enums", "status", "acct_status", Options{}); err == nil || !strings.Contains(err.Error(), "TS") {
		t.Fatalf("%v", err)
	}
}

// 파일 이름을 ID로 짓는 종류는 Go 빌드 제약 접미사로 끝나는 새 ID를 받지 않는다.
func TestRunRefusesBuildSuffix(t *testing.T) {
	for _, id := range []string{"status_windows", "status_arm64", "status_linux_amd64"} {
		design, root := basicCopy(t)
		if _, err := Run(design, root, "enums", "status", id, Options{}); err == nil || !strings.Contains(err.Error(), "빌드 제약") {
			t.Fatalf("%s: %v", id, err)
		}
	}
}
