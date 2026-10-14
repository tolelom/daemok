package codescan

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/model"
)

func TestIncludes(t *testing.T) {
	src := "#include \"a.h\"\n" +
		"  #  include <b/c.h> // 끝\n" +
		"// #include \"no1.h\"\n" +
		"/* #include \"no2.h\"\n#include \"no3.h\" */ #include \"no4.h\"\n" +
		"const char* s = \"/*\"; char c = '\"';\n" +
		"#include \"d.h\"\n" +
		"#define X \"#include \\\"no5.h\\\"\"\n" +
		"#import \"no6.h\"\n"
	got := includes([]byte(src))
	want := []include{
		{line: 1, col: 1, path: "a.h"},
		{line: 2, col: 3, path: "b/c.h", angle: true},
		{line: 5, col: 21, path: "no4.h"},
		{line: 7, col: 1, path: "d.h"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, src := range files {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const cppDesign = `version: 0.5
cpp: { include_dirs: [src/Public, src/Private] }
tiers: [base, mid, top]
modules:
  shared: { tier: base, path: Shared }
  extra: { tier: base, path: Extra }
  core: { tier: mid, path: Core, depends_on: [shared, extra] }
  zone: { tier: mid, path: Core/Zone }
  ui: { tier: top, path: UI, depends_on: [core] }
`

func cppScanOf(t *testing.T, design string, files map[string]string) (string, model.Diagnostics) {
	t.Helper()
	root := t.TempDir()
	writeTree(t, root, files)
	writeTree(t, root, map[string]string{"design/d.yaml": design})
	m, ds := model.Load(filepath.Join(root, "design"))
	if ds.HasErrors() {
		t.Fatalf("load: %v", ds)
	}
	if cds := check.Run(m); cds.HasErrors() {
		t.Fatalf("check: %v", cds)
	}
	return root, Scan(m, root)
}

func TestScanCPP(t *testing.T) {
	root, ds := cppScanOf(t, cppDesign, map[string]string{
		"src/Public/Shared/Types.h": "#pragma once\n",
		"src/Public/Misc/Thing.h":   "#pragma once\n",
		"src/Public/Helper.h":       "#pragma once\n",
		"src/Private/Extra/E.cpp":   "#pragma once\n",
		"src/Public/Core/Core.h": "#include \"Shared/Types.h\"\n" +
			"#include \"UI/Hud.h\"\n" +
			"// #include \"UI/Hud.h\"\n" +
			"/*\n#include \"UI/Hud.h\"\n*/\n" +
			"#include \"Components/Actor.h\"\n" +
			"#include \"Misc/Thing.h\"\n",
		"src/Private/Core/Core.cpp":   "#include \"Core/Core.h\"\n#include \"Helper.h\"\n#include \"Misc/Thing.h\"\n",
		"src/Private/Core/Helper.h":   "#pragma once\n",
		"src/Private/Core/notes.txt":  "#include \"UI/Hud.h\"\n",
		"src/Public/Core/Zone/Zone.h": "#include \"UI/Hud.h\"\n#include <Shared/Types.h>\n#include \"../Core.h\"\n",
		"src/Public/UI/Hud.h":         "#include \"Core/Core.h\"\nconst char* s = \"/*\";\n  #  include \"Shared/Types.h\"\n",
	})
	var got []string
	for _, d := range ds {
		got = append(got, d.Level.String()+": "+d.Msg)
	}
	sort.Strings(got)
	want := []string{
		"오류: 설계에 없는 의존을 include한다: core → ui",
		"오류: 설계에 없는 의존을 include한다: ui → shared",
		"오류: 설계에 없는 의존을 include한다: zone → core",
		"오류: 설계에 없는 의존을 include한다: zone → shared",
		"오류: 설계에 없는 의존을 include한다: zone → ui",
		"정보: 관리하지 않는 파일을 include한다: core → src/Public/Misc/Thing.h",
		"정보: 설계상 의존하지만 import하지 않는다: core → extra",
	}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("진단:\n%s", strings.Join(got, "\n"))
	}
	for _, d := range ds {
		if strings.HasSuffix(d.Msg, "core → ui") {
			if d.Pos.File != filepath.Join(root, "src", "Public", "Core", "Core.h") || d.Pos.Line != 2 || d.Pos.Col != 1 {
				t.Fatalf("위치: %v", d.Pos)
			}
		}
		if strings.HasSuffix(d.Msg, "ui → shared") && (d.Pos.Line != 3 || d.Pos.Col != 3) {
			t.Fatalf("위치: %v", d.Pos)
		}
	}
}

func TestScanCPPIncludeDirOrder(t *testing.T) {
	_, ds := cppScanOf(t, cppDesign, map[string]string{
		"src/Public/X.h":         "#pragma once\n",
		"src/Private/X.h":        "#pragma once\n",
		"src/Public/Core/Core.h": "#include \"X.h\"\n",
	})
	want(t, ds, model.Info, "관리하지 않는 파일을 include한다: core → src/Public/X.h")
}

func TestScanCPPMissingIncludeDir(t *testing.T) {
	design := strings.Replace(cppDesign, "src/Private", "nope", 1)
	_, ds := cppScanOf(t, design, map[string]string{"src/Public/Core/Core.h": "#include \"Shared/Types.h\"\n"})
	want(t, ds, model.Error, "include 디렉토리가 없다: nope")
}

func want(t *testing.T, ds model.Diagnostics, level model.Level, substr string) {
	t.Helper()
	for _, d := range ds {
		if d.Level == level && strings.Contains(d.Msg, substr) {
			return
		}
	}
	t.Fatalf("%q 진단이 없다: %v", substr, ds)
}
