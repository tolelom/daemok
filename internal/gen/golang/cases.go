package golang

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

const caseHelpers = `type caseInput = map[string]any

// errAny는 sentinel 없이 "에러가 나야 한다"만 적은 케이스에 쓴다.
var errAny = errors.New("any error")

func checkErr(t *testing.T, got, want error) {
	t.Helper()
	switch {
	case want == nil && got != nil:
		t.Fatalf("unexpected error: %v", got)
	case want == errAny && got == nil:
		t.Fatal("expected an error")
	case want != nil && want != errAny && !errors.Is(got, want):
		t.Fatalf("want %v, got %v", want, got)
	}
}

func checkExpect(t *testing.T, got, want caseInput) {
	t.Helper()
	for k, v := range want {
		if fmt.Sprint(got[k]) != fmt.Sprint(v) {
			t.Fatalf("%s: want %v, got %v", k, v, got[k])
		}
	}
}`

func (g *gen) cases() error {
	helpers := map[string]bool{}
	for _, fn := range g.m.Funcs {
		if !fn.Gen || len(fn.Cases) == 0 {
			continue
		}
		dir := g.ownerPath(fn.In.Name)
		if !helpers[dir] {
			helpers[dir] = true
			h := g.file(dir + "/daemok_case_gen_test.go")
			h.use("errors")
			h.use("fmt")
			h.use("testing")
			h.add("%s", caseHelpers)
		}
		f := g.file(dir + "/" + fn.ID + "_gen_test.go")
		f.use("testing")
		name := fn.ID
		if fn.Of.Name != "" {
			name = strings.ReplaceAll(fn.Of.Name, ".", "_")
		}
		var rows []string
		for _, c := range fn.Cases {
			row := fmt.Sprintf("\t\t{name: %q, given: %s", c.Name, literal(c.Given, "caseInput"))
			if c.Expect != nil {
				row += ", expect: " + literal(c.Expect, "caseInput")
			}
			if c.Error.Name != "" {
				row += ", err: " + g.caseErr(c.Error.Name, dir, f)
			}
			rows = append(rows, row+"},")
		}
		f.add(`func Test%s(t *testing.T) {
	cases := []struct {
		name   string
		given  caseInput
		expect caseInput
		err    error
	}{
%s
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := run%sCase(t, c.given) // 손으로 쓰는 함수: 상태를 만들고 호출한다
			checkErr(t, err, c.err)
			checkExpect(t, got, c.expect)
		})
	}
}`, model.GoName(name), strings.Join(rows, "\n"), model.GoName(fn.ID))
	}
	return nil
}

func (g *gen) caseErr(id, dir string, f *goFile) string {
	if id == "invalid" {
		return "errAny"
	}
	e := g.m.Error(id)
	if e == nil {
		return id
	}
	return g.qualify(e.In.Name, e.GoName(), dir, f)
}

// literal은 given·expect 노드를 Go 리터럴로 바꾼다. 맨 위 맵은 typ, 안쪽 맵은 map[string]any다.
func literal(n *yaml.Node, typ string) string {
	if n == nil {
		return "nil"
	}
	if n.Kind == yaml.Map {
		var parts []string
		for _, e := range n.Entries {
			parts = append(parts, strconv.Quote(e.Key)+": "+literal(e.Value, "map[string]any"))
		}
		return typ + "{" + strings.Join(parts, ", ") + "}"
	}
	if n.IsNull() {
		return "nil"
	}
	if i, ok := n.Int(); ok {
		return strconv.Itoa(i)
	}
	if b, ok := n.Bool(); ok {
		return strconv.FormatBool(b)
	}
	return strconv.Quote(n.Value)
}
