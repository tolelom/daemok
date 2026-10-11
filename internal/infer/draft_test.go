package infer

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSnake(t *testing.T) {
	for in, want := range map[string]string{
		"core": "core", "certGen": "cert_gen", "HTTPServer": "http_server", "v2": "v2",
		"a1B": "a1_b", "tool-kit": "tool_kit", "API": "api", "x__y_": "x_y",
	} {
		if got := snake(in); got != want {
			t.Errorf("snake(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnalyzeLayers(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "layers"))
	if err != nil {
		t.Fatal(err)
	}
	want := &Draft{
		GoModule: "example.com/layers",
		System:   "layers",
		Exes:     []Exe{{ID: "app", Dir: "cmd/app", Summary: "app은 예제 실행 파일이다.", Composes: []string{"api"}}},
		Tiers:    3,
		Mods: []Mod{
			{ID: "internal_util", Path: "internal/util", Tier: 0},
			{ID: "util", Tier: 0},
			{ID: "cert_gen", Path: "certGen", Tier: 1, DependsOn: []string{"internal_util"}},
			{ID: "core", Tier: 1, Summary: "Package core는 도메인 타입이다.", DependsOn: []string{"util"}},
			{ID: "api", Tier: 2, DependsOn: []string{"core", "util"}},
		},
	}
	if !reflect.DeepEqual(a.Draft, want) {
		t.Fatalf("got  %+v\nwant %+v", a.Draft, want)
	}
	wantRep := Report{
		Packages: 7, Modules: 5, Executables: 1,
		Skipped: []Skip{{"bad-name", "Go 패키지 경로로 쓸 수 없는 디렉토리 이름"}, {"nested", "중첩 Go 모듈"}},
		FanIn:   []FanIn{{"util", 2}, {"api", 1}, {"core", 1}, {"internal_util", 1}},
	}
	if !reflect.DeepEqual(a.Report, wantRep) {
		t.Fatalf("got  %+v\nwant %+v", a.Report, wantRep)
	}
}

func TestAnalyzeRootMain(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "rootmain"))
	if err != nil {
		t.Fatal(err)
	}
	d := a.Draft
	if d.System != "tool_kit" || len(d.Exes) != 1 || d.Exes[0].ID != "tool_kit" || d.Exes[0].Dir != "." {
		t.Fatalf("%+v", d)
	}
	if len(d.Mods) != 1 || d.Mods[0].ID != "lib" || d.Mods[0].Summary != `Package lib는 "따옴표"와 \역슬래시, #, : 를 쓴다.` {
		t.Fatalf("%+v", d.Mods)
	}
}

func TestAnalyzeNotModule(t *testing.T) {
	if _, err := Analyze(filepath.Join("testdata", "nomod")); !errors.Is(err, ErrNotModule) {
		t.Fatalf("%v", err)
	}
}

func TestAssignIDs(t *testing.T) {
	got := assignIDs([]string{"a/util", "b/util", "util", "foo/z", "Foo/z"}, "sys")
	want := map[string]string{"a/util": "a_util", "b/util": "b_util", "util": "util", "Foo/z": "foo_z", "foo/z": "foo_z_2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
}

func TestBuildCycleSharesTier(t *testing.T) {
	pkgs := []*pkg{
		{dir: "a", imports: []string{"b"}},
		{dir: "b", imports: []string{"a", "base"}},
		{dir: "base"},
		{dir: "c", imports: []string{"a"}},
	}
	d, rep := build("example.com/x", pkgs, nil)
	tiers := map[string]int{}
	for _, m := range d.Mods {
		tiers[m.ID] = m.Tier
	}
	if tiers["base"] != 0 || tiers["a"] != 1 || tiers["b"] != 1 || tiers["c"] != 2 {
		t.Fatalf("%v", tiers)
	}
	if !reflect.DeepEqual(rep.Cycles, [][]string{{"a", "b"}}) {
		t.Fatalf("%v", rep.Cycles)
	}
}

func TestGroupCandidates(t *testing.T) {
	pkgs := []*pkg{
		{dir: "core"}, {dir: "vm", imports: []string{"core"}},
		{dir: "vm/modules/a", imports: []string{"core", "vm"}},
		{dir: "vm/modules/b", imports: []string{"core", "vm"}},
		{dir: "vm/modules/c", imports: []string{"core", "vm", "vm/modules/a"}},
		{dir: "internal/x"}, {dir: "internal/y", imports: []string{"core"}}, {dir: "internal/z", imports: []string{"vm"}},
		{dir: "pair/p", imports: []string{"core"}}, {dir: "pair/q", imports: []string{"core"}},
	}
	_, rep := build("example.com/g", pkgs, nil)
	want := []GroupCandidate{{Path: "vm/modules", Members: []string{"a", "b", "c"}, Common: []string{"core", "vm"}, Exceptions: []string{"c → a"}}}
	if !reflect.DeepEqual(rep.GroupCandidates, want) {
		t.Fatalf("%+v", rep.GroupCandidates)
	}
}

func TestExternalRequires(t *testing.T) {
	a, err := Analyze(filepath.Join("testdata", "ext"))
	if err != nil {
		t.Fatal(err)
	}
	want := []External{
		{Module: "github.com/gofiber/fiber/v2", Users: []string{"api"}},
		{Module: "github.com/google/uuid", Users: []string{"api", "db"}},
		{Module: "github.com/lib/pq", Users: []string{"db"}},
	}
	if !reflect.DeepEqual(a.Report.External, want) {
		t.Fatalf("%+v", a.Report.External)
	}
}
