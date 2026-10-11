package serve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 설계와 상관없는 코드 저장으로 버전이 올라도 화면이 본 설계가 그대로면 편집을 받는다.
func TestEditAcceptsAfterCodeOnlyChange(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	v := s.Version()
	os.WriteFile(filepath.Join(filepath.Dir(dir), "x.go"), []byte("package x\n"), 0o644)
	if !s.Poll() || s.Version() == v {
		t.Fatal("코드가 바뀌면 상태가 새로 나와야 한다")
	}
	if code, out := editReq(t, s, "tier", "api", "a", v); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
}

// 화면이 본 뒤에 설계 파일이 바뀌었으면 409다.
func TestEditRejectsAfterDesignChange(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	v := s.Version()
	os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(strings.Replace(design, "core: { tier: a }", "core: { tier: a, summary: s }", 1)), 0o644)
	s.Poll()
	if code, _ := editReq(t, s, "tier", "api", "a", v); code != 409 {
		t.Fatalf("%d", code)
	}
	if code, out := editReq(t, s, "tier", "api", "a", s.Version()); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
}

// 편집 전부터 있던 읽기 오류는 422 판정에 넣지 않는다.
func TestEditIgnoresExistingLoadErrors(t *testing.T) {
	s, dir := setup(t, design)
	local(t, s)
	os.WriteFile(filepath.Join(dir, "e.yaml"), []byte("bogus: 1\n"), 0o644)
	s.Poll()
	if code, out := editReq(t, s, "tier", "api", "a", s.Version()); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	if !strings.Contains(string(b), "api:  { tier: a,") {
		t.Fatal(string(b))
	}
}

// 설계 파일이 심볼릭 링크면 링크는 두고 가리키는 파일을 고친다.
func TestEditThroughSymlink(t *testing.T) {
	s, dir := setup(t, design)
	real := filepath.Join(t.TempDir(), "real.yaml")
	os.WriteFile(real, []byte(design), 0o644)
	link := filepath.Join(dir, "d.yaml")
	os.Remove(link)
	if err := os.Symlink(real, link); err != nil {
		t.Skip("심볼릭 링크를 만들 수 없다:", err)
	}
	s.Poll()
	local(t, s)
	if code, out := editReq(t, s, "tier", "api", "a", s.Version()); code != 200 {
		t.Fatalf("%d %v", code, out)
	}
	if fi, _ := os.Lstat(link); fi.Mode()&os.ModeSymlink == 0 {
		t.Fatal("링크가 일반 파일로 바뀌었다")
	}
	if b, _ := os.ReadFile(real); !strings.Contains(string(b), "api:  { tier: a,") {
		t.Fatal(string(b))
	}
}
