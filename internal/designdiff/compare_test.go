package designdiff

import (
	"strings"
	"testing"
)

// brief는 Report를 비교하기 쉬운 한 줄씩으로 쓴다.
func brief(r *Report) string {
	var out []string
	for _, s := range r.Settings {
		out = append(out, "설정 "+s)
	}
	for _, n := range r.Added {
		out = append(out, "+"+n.Kind+"/"+n.ID)
	}
	for _, n := range r.Renamed {
		out = append(out, "="+n.Kind+"/"+n.From+">"+n.To)
	}
	for _, m := range r.Modified {
		out = append(out, "~"+m.Kind+"/"+m.ID+": "+strings.Join(m.Lines, "; "))
	}
	for _, n := range r.Removed {
		out = append(out, "-"+n.Kind+"/"+n.ID)
	}
	return strings.Join(out, "\n")
}

func cmp(t *testing.T, base, work map[string]string) string {
	t.Helper()
	b, w := map[string][]byte{}, map[string][]byte{}
	for k, v := range base {
		b[k] = []byte(v)
	}
	for k, v := range work {
		w[k] = []byte(v)
	}
	r, err := Compare(b, w)
	if err != nil {
		t.Fatal(err)
	}
	return brief(r)
}

func one(t *testing.T, base, work string) string {
	t.Helper()
	return cmp(t, map[string]string{"a.yaml": base}, map[string]string{"a.yaml": work})
}

func TestCompareIgnoresFormat(t *testing.T) {
	base := "tiers: [a, b]\nmodules:\n  core: { tier: a, depends_on: [x] }\n  web: { tier: b }\n"
	work := map[string]string{
		"a.yaml": "# 주석\ntiers: [a, b]\nmodules:\n  core:\n    tier: \"a\"\n    depends_on:\n      - x\n",
		"b.yaml": "modules:\n  web: { tier: b }\n",
	}
	if got := cmp(t, map[string]string{"a.yaml": base}, work); got != "" {
		t.Fatalf("변경이 없어야 한다:\n%s", got)
	}
}

func TestCompareValues(t *testing.T) {
	cases := []struct{ name, base, work, want string }{
		{"스칼라", "modules:\n  web: { tier: b }\n", "modules:\n  web: { tier: c }\n", "~modules/web: tier: b → c"},
		{"키 추가", "types:\n  p: { fields: { a: int } }\n", "types:\n  p: { fields: { a: int, title: string } }\n", "~types/p: fields.title 추가: string"},
		{"키 삭제", "modules:\n  web: { tier: b, path: w }\n", "modules:\n  web: { tier: b }\n", "~modules/web: path 삭제: w"},
		{"맵 값 추가", "types:\n  p: { in: c }\n", "types:\n  p: { in: c, go: { name: P } }\n", "~types/p: go 추가: { name: P }"},
		{"목록", "modules:\n  web: { depends_on: [auth, core] }\n", "modules:\n  web: { depends_on: [core, edge] }\n", "~modules/web: depends_on: +edge -auth"},
		{"순서", "modules:\n  web: { depends_on: [a, b] }\n", "modules:\n  web: { depends_on: [b, a] }\n", "~modules/web: depends_on: 순서 바뀜"},
		{"단계 끼워 넣기", "flows:\n  f:\n    steps:\n      - { id: a }\n      - { id: c, undo: u }\n", "flows:\n  f:\n    steps:\n      - { id: a }\n      - { id: b }\n      - { id: c, undo: v }\n", "~flows/f: steps[b] 추가: { id: b }; steps[c].undo: u → v"},
		{"단계 순서", "flows:\n  f: { steps: [{ id: a }, { id: b }] }\n", "flows:\n  f: { steps: [{ id: b }, { id: a }] }\n", "~flows/f: steps: 순서 바뀜"},
		{"케이스", "funcs:\n  f: { cases: [{ name: x, error: E }] }\n", "funcs:\n  f: { cases: [{ name: x, error: F }] }\n", "~funcs/f: cases[x].error: E → F"},
		{"짝 키 없는 목록", "contracts:\n  c: { methods: { m: { params: [{ a: int }] } } }\n", "contracts:\n  c: { methods: { m: { params: [{ a: string }] } } }\n", "~contracts/c: methods.m.params: 값 바뀜"},
		{"추가 삭제", "errors:\n  ErrA: { message: a }\n", "errors:\n  ErrB: { message: b }\n", "+errors/ErrB\n-errors/ErrA"},
		{"설정", "version: 0.5\ntiers: [a]\n", "version: 0.6\ntiers: [a, b]\n", "설정 version: 0.5 → 0.6\n설정 tiers: +b"},
		{"멤버", "groups:\n  g:\n    tier: a\n    members:\n      m: { handles: [x] }\n", "groups:\n  g:\n    tier: b\n    members:\n      m: { handles: [x, y] }\n", "~groups/g: tier: a → b\n~members/m: handles: +y"},
		{"멤버 그룹 이동", "groups:\n  g: { members: { m: { handles: [x] } } }\n  h: { members: {} }\n", "groups:\n  g: { members: {} }\n  h: { members: { m: { handles: [x] } } }\n", "~members/m: 그룹: g → h"},
		{"system", "system:\n  id: s\n  actors:\n    a: { summary: x }\n", "system:\n  id: s\n  summary: y\n  actors:\n    a: { summary: z }\n", "~system/: summary 추가: y\n~actors/a: summary: x → z"},
		{"이름 바뀜 모듈", "types:\n  post: { in: core }\nmodules:\n  core: { tier: a }\n", "types:\n  post: { in: base }\nmodules:\n  base: { tier: a, path: core }\n", "=modules/core>base\n~types/post: in: core → base"},
		{"이름 바뀜 고정 키", "types:\n  post: { in: core, fields: { a: int } }\n", "types:\n  article: { in: core, fields: { a: int }, go: { name: Post } }\n", "=types/post>article"},
		{"이름 바뀜 go 맵 안", "enums:\n  k: { go: { prefix: K }, values: [a] }\n", "enums:\n  k2: { go: { prefix: K, type: K }, values: [a] }\n", "=enums/k>k2"},
		{"이름 바뀜 둘", "errors:\n  ErrA: { message: m }\n", "errors:\n  ErrB: { message: m }\n  ErrC: { message: m }\n", "+errors/ErrB\n+errors/ErrC\n-errors/ErrA"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := one(t, c.base, c.work); got != c.want {
				t.Fatalf("got\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

func TestCompareParseError(t *testing.T) {
	if _, err := Compare(map[string][]byte{}, map[string][]byte{"a.yaml": []byte("a: [\n")}); err == nil {
		t.Fatal("읽을 수 없는 파일을 받았다")
	}
}
