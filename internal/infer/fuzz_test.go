package infer

import (
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

// FuzzRenderSummary는 어떤 doc 주석 첫 문장이 summary로 들어와도 초안이 다시 읽히고 값이 보존되는지 본다.
func FuzzRenderSummary(f *testing.F) {
	for _, s := range []string{"평범한 문장.", `"따옴표" \역슬래시`, "줄\n바꿈\t탭", "# 샵: 콜론 {} []", "", "\r\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, summary string) {
		d := &Draft{GoModule: "example.com/x", System: "x", Tiers: 1, Modules: []Module{{ID: "a", Summary: summary}}}
		m, diags := model.LoadSources(map[string][]byte{"x.yaml": Render(d)})
		if diags.HasErrors() {
			t.Fatalf("%q: %v\n%s", summary, diags, Render(d))
		}
		if want := strings.Join(strings.Fields(summary), " "); m.Modules[0].Summary != want {
			t.Fatalf("%q → %q, want %q", summary, m.Modules[0].Summary, want)
		}
	})
}
