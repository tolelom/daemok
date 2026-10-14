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
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
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
  post: { in: post, summary: post 하나, fields: { p: "[]*post", s: post_status, m: "map[string]post" } }
  page: { in: web, fields: { first: { ref: post } } }
enums:
  post_status: { in: post, values: [post, draft] }
`

func changes(t *testing.T, m *model.Model, kind, old, new string) string {
	t.Helper()
	reps, err := Occurrences(m, kind, old, new)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range reps {
		got = append(got, r.Old+"->"+r.New)
	}
	sort.Strings(got)
	return strings.Join(got, " ")
}

func TestOccurrencesTypeOnly(t *testing.T) {
	got := changes(t, load(t, sample), "types", "post", "article")
	if want := "[]*post->[]*article map[string]post->map[string]article post->article post->article"; got != want {
		t.Fatalf("got %s", got)
	}
}

func TestOccurrencesModuleOnly(t *testing.T) {
	got := changes(t, load(t, sample), "modules", "post", "core")
	if want := "post->core post->core post->core post->core"; got != want {
		t.Fatalf("정의, depends_on, in 두 개여야 한다: %s", got)
	}
}

func TestOccurrencesMissing(t *testing.T) {
	if _, err := Occurrences(load(t, sample), "types", "nope", "x"); err == nil {
		t.Fatal("없는 ID를 받았다")
	}
}

func TestOccurrencesEventRenamesKeyValue(t *testing.T) {
	m := load(t, `version: 0.5
go_module: example.com/x
tiers: [a]
modules:
  core: { tier: a }
enums:
  ev: { in: core, values: [made, gone] }
event_kinds:
  evk: { in: core, key: ev }
events:
  made: { kind: evk, publishers: [core] }
  gone: { kind: evk, publishers: [core] }
flows:
  f: { in: core, on: made, steps: [{ id: s, emits: [made] }] }
`)
	if got := changes(t, m, "events", "made", "built"); got != "made->built made->built made->built made->built" {
		t.Fatalf("정의, enum 값, on, emits여야 한다: %s", got)
	}
}

func TestRetype(t *testing.T) {
	for in, want := range map[string]string{
		"post": "article", "[]*post": "[]*article", "map[post]post_status": "map[article]post_status",
		"post_status": "post_status", "map[string][]*post": "map[string][]*article", "x.post": "x.post",
	} {
		if got := retype(in, "post", "article"); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}
