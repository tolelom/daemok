package edit

import (
	"strings"
	"testing"
)

const withGroups = `version: 0.5
tiers: [a, b]
modules:
  core: { tier: a }
  vm:   { tier: b, depends_on: [core] }
  last:
    tier: b
    depends_on: [core]
groups:
  tx:
    tier: b
    kind: h
    depends_on: [core]
    members:
      econ: { handles: [x] }
`

func TestGroupTierAndDeps(t *testing.T) {
	got, err := SetTier([]byte(withGroups), "f", "tx", "a")
	must(t, err)
	same(t, got, strings.Replace(withGroups, "  tx:\n    tier: b", "  tx:\n    tier: a", 1))
	got, err = AddDep([]byte(withGroups), "f", "tx", "vm")
	must(t, err)
	same(t, got, strings.Replace(withGroups, "    kind: h\n    depends_on: [core]", "    kind: h\n    depends_on: [core, vm]", 1))
	got, err = RemoveDep([]byte(withGroups), "f", "tx", "core")
	must(t, err)
	same(t, got, strings.Replace(withGroups, "    kind: h\n    depends_on: [core]", "    kind: h\n    depends_on: []", 1))
}

func TestRemoveModule(t *testing.T) {
	cases := []struct{ module, from string }{
		{"vm", "  vm:   { tier: b, depends_on: [core] }\n"},
		{"last", "  last:\n    tier: b\n    depends_on: [core]\n"},
	}
	for _, c := range cases {
		got, err := RemoveModule([]byte(withGroups), "f", c.module)
		must(t, err)
		same(t, got, strings.Replace(withGroups, c.from, "", 1))
	}
	if _, err := RemoveModule([]byte(withGroups), "f", "tx"); err == nil {
		t.Fatal("그룹을 모듈 삭제로 지웠다")
	}
	if _, err := RemoveModule([]byte(withGroups), "f", "nope"); err == nil {
		t.Fatal("없는 모듈")
	}
}
