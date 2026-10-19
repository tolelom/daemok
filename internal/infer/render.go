package infer

import (
	"bytes"
	"fmt"
	"strings"
)

func Render(d *Draft) []byte {
	var b bytes.Buffer
	b.WriteString("# daemok init이 만든 초안이다. 코드의 패키지와 import에서 추정했다.\n")
	b.WriteString("# tier 이름, 그룹, summary는 사람이 정한다.\n")
	b.WriteString("version: 0.5\n")
	fmt.Fprintf(&b, "go_module: %s\n", d.GoModule)
	fmt.Fprintf(&b, "\nsystem:\n  id: %s\n", d.System)
	if len(d.Executables) > 0 {
		b.WriteString("\nexecutables:\n")
		for _, e := range d.Executables {
			var kv []string
			if e.Summary != "" {
				kv = append(kv, "summary: "+quote(e.Summary))
			}
			kv = append(kv, "composes: "+list(e.Composes))
			fmt.Fprintf(&b, "  %s: { %s }\n", e.ID, strings.Join(kv, ", "))
		}
	}
	if len(d.Modules) > 0 {
		tiers := make([]string, d.Tiers)
		for i := range tiers {
			tiers[i] = tierName(i)
		}
		fmt.Fprintf(&b, "\ntiers: %s\n\nmodules:\n", list(tiers))
		for _, m := range d.Modules {
			kv := []string{"tier: " + tierName(m.Tier)}
			if m.Path != "" {
				kv = append(kv, "path: "+m.Path)
			}
			if m.Summary != "" {
				kv = append(kv, "summary: "+quote(m.Summary))
			}
			if len(m.DependsOn) > 0 {
				kv = append(kv, "depends_on: "+list(m.DependsOn))
			}
			fmt.Fprintf(&b, "  %s: { %s }\n", m.ID, strings.Join(kv, ", "))
		}
	}
	return b.Bytes()
}

func tierName(i int) string { return fmt.Sprintf("tier%d", i) }

func list(xs []string) string { return "[" + strings.Join(xs, ", ") + "]" }

func quote(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
