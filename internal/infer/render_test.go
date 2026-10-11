package infer

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

const layersYAML = `# daemok init이 만든 초안이다. 코드의 패키지와 import에서 추정했다.
# tier 이름, 그룹, summary는 사람이 정한다.
version: 0.4
go_module: example.com/layers

system:
  id: layers

executables:
  app: { summary: "app은 예제 실행 파일이다.", composes: [api] }

tiers: [tier0, tier1, tier2]

modules:
  internal_util: { tier: tier0, path: internal/util }
  util: { tier: tier0 }
  cert_gen: { tier: tier1, path: certGen, depends_on: [internal_util] }
  core: { tier: tier1, summary: "Package core는 도메인 타입이다.", depends_on: [util] }
  api: { tier: tier2, depends_on: [core, util] }
`

func TestRenderLayers(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "layers"))
	if err != nil {
		t.Fatal(err)
	}
	if string(a.YAML) != layersYAML {
		t.Fatalf("got\n%s", a.YAML)
	}
	b, _ := Analyze(filepath.Join("testdata", "layers"))
	if !bytes.Equal(a.YAML, b.YAML) {
		t.Fatal("같은 코드에서 다른 초안")
	}
}

func TestRenderRoundTrip(t *testing.T) {
	d := &Draft{
		GoModule: "example.com/x", System: "x", Tiers: 2,
		Exes: []Exe{{ID: "x", Dir: ".", Summary: "줄\n바꿈과\t탭"}},
		Mods: []Mod{
			{ID: "a", Tier: 0, Summary: `"따옴표"와 \역슬래시, # 샵: 콜론 {중괄호} [대괄호]`},
			{ID: "b", Path: "pkg/b", Tier: 1, DependsOn: []string{"a"}},
		},
	}
	m, diags := model.LoadSources(map[string][]byte{"x.yaml": Render(d)})
	if diags.HasErrors() {
		t.Fatalf("%v\n%s", diags, Render(d))
	}
	if m.GoModule != d.GoModule || m.System.ID != "x" || len(m.Tiers) != 2 || m.Tiers[1].Name != "tier1" {
		t.Fatalf("%+v", m)
	}
	if len(m.Executables) != 1 || m.Executables[0].Summary != "줄 바꿈과 탭" || len(m.Executables[0].Composes) != 0 {
		t.Fatalf("%+v", m.Executables[0])
	}
	a, b := m.Modules[0], m.Modules[1]
	if a.ID != "a" || a.Summary != d.Mods[0].Summary || a.Tier.Name != "tier0" {
		t.Fatalf("%+v", a)
	}
	if b.ID != "b" || b.Path != "pkg/b" || len(b.DependsOn) != 1 || b.DependsOn[0].Name != "a" {
		t.Fatalf("%+v", b)
	}
}

func TestRenderNoModules(t *testing.T) {
	d := &Draft{GoModule: "example.com/x", System: "x", Exes: []Exe{{ID: "x", Dir: "."}}}
	if _, diags := model.LoadSources(map[string][]byte{"x.yaml": Render(d)}); diags.HasErrors() {
		t.Fatalf("%v\n%s", diags, Render(d))
	}
}
