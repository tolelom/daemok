package codescan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/model"
)

func TestScan(t *testing.T) {
	root := filepath.Join("testdata", "scan")
	m, ds := model.Load(filepath.Join(root, "design"))
	if ds.HasErrors() {
		t.Fatalf("load: %v", ds)
	}
	if cds := check.Run(m); cds.HasErrors() {
		t.Fatalf("check: %v", cds)
	}
	var got []string
	for _, d := range Scan(m, root) {
		got = append(got, d.Level.String()+": "+d.Msg)
	}
	sort.Strings(got)
	want := []string{
		"경고: 발행자나 구독자로 적히지 않은 모듈이 이벤트를 쓴다: 모듈 b, 이벤트 deleted",
		"오류: Go 파일을 읽을 수 없다: " + filepath.Join(root, "lonely", "broken.go"),
		"오류: needs로만 이어진 모듈을 직접 import한다: feed → post",
		"오류: query 패키지가 data가 아닌 모듈을 import한다: post/query → store",
		"오류: 설계에 없는 의존을 import한다: b → a",
		"정보: 관리하지 않는 패키지를 import한다: post → example.com/scan/post/util",
		"정보: 설계상 의존하지만 import하지 않는다: lonely → store",
	}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("진단 수 %d, 기대 %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if !strings.HasPrefix(got[i], want[i]) {
			t.Errorf("got  %s\nwant %s", got[i], want[i])
		}
	}
}

func TestScanPositionsAreGoFiles(t *testing.T) {
	root := filepath.Join("testdata", "scan")
	m, _ := model.Load(filepath.Join(root, "design"))
	for _, d := range Scan(m, root) {
		if strings.Contains(d.Msg, "직접 import한다: feed → post") {
			if !strings.HasSuffix(d.Pos.File, filepath.Join("feed", "feed.go")) || d.Pos.Line != 6 {
				t.Fatalf("위치: %v", d.Pos)
			}
			return
		}
	}
	t.Fatal("진단이 없다")
}

func TestMissingDirsAreSilent(t *testing.T) {
	m, _ := model.Load(filepath.Join("testdata", "scan", "design"))
	if ds := Scan(m, t.TempDir()); len(ds) != 0 {
		t.Fatalf("코드가 없으면 진단이 없어야 한다: %v", ds)
	}
}

func TestDeclaredButUnreferenced(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"design/d.yaml": `version: 0.4
go_module: example.com/u
tiers: [a, b]
modules:
  events: { tier: a }
  pub:    { tier: b, depends_on: [events] }
  sub:    { tier: b, depends_on: [events] }
enums:
  event_type: { in: events, go: { type: EventType, prefix: Event }, values: [done] }
event_kinds:
  ev: { in: events, key: event_type, subscribers: [sub] }
events:
  done: { kind: ev, publishers: [pub] }
`,
		"events/events.go": "package events\n",
		"pub/pub.go":       "package pub\n\nimport _ \"example.com/u/events\"\n",
		"sub/sub.go":       "package sub\n\nimport _ \"example.com/u/events\"\n",
	}
	for p, c := range files {
		writeFile(t, filepath.Join(root, p), c)
	}
	m, ds := model.Load(filepath.Join(root, "design"))
	if ds.HasErrors() {
		t.Fatal(ds)
	}
	var msgs []string
	for _, d := range Scan(m, root) {
		msgs = append(msgs, d.Msg)
	}
	joined := strings.Join(msgs, "\n")
	for _, want := range []string{
		"이벤트 상수를 쓰지 않는다: 모듈 pub, 이벤트 done (EventDone)",
		"이벤트 목록 함수를 쓰지 않는다: 모듈 sub, 이벤트 종류 ev (EventTypeValues)",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("%q 없음:\n%s", want, joined)
		}
	}
}

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
