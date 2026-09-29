package golang

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/model"
)

var update = flag.Bool("update", false, "골든 파일을 다시 쓴다")

var samples = []string{"basic", "relations"}

func generateSample(t *testing.T, sample string) (*model.Model, []File) {
	t.Helper()
	m, ds := model.Load(filepath.Join("testdata", sample, "design"))
	if ds.HasErrors() {
		t.Fatalf("load %s:\n%v", sample, ds)
	}
	if cds := check.Run(m); cds.HasErrors() {
		t.Fatalf("check %s:\n%v", sample, cds)
	}
	files, err := Generate(m)
	if err != nil {
		t.Fatal(err)
	}
	return m, files
}

func TestGolden(t *testing.T) {
	for _, s := range samples {
		t.Run(s, func(t *testing.T) {
			_, files := generateSample(t, s)
			dir := filepath.Join("testdata", s, "golden")
			if *update {
				if err := os.RemoveAll(dir); err != nil {
					t.Fatal(err)
				}
				for _, f := range files {
					writeFile(t, filepath.Join(dir, filepath.FromSlash(f.Path)), f.Content)
				}
				return
			}
			want := map[string][]byte{}
			filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					rel, _ := filepath.Rel(dir, p)
					want[filepath.ToSlash(rel)], _ = os.ReadFile(p)
				}
				return nil
			})
			for _, f := range files {
				w, ok := want[f.Path]
				if !ok {
					t.Errorf("골든에 없는 파일: %s", f.Path)
					continue
				}
				if !bytes.Equal(w, f.Content) {
					t.Errorf("골든과 다르다: %s\n--- got\n%s", f.Path, f.Content)
				}
				delete(want, f.Path)
			}
			for p := range want {
				t.Errorf("생성되지 않은 골든 파일: %s", p)
			}
		})
	}
}

func TestDeterministic(t *testing.T) {
	for _, s := range samples {
		_, a := generateSample(t, s)
		_, b := generateSample(t, s)
		if len(a) != len(b) {
			t.Fatalf("%s: 파일 수가 다르다", s)
		}
		for i := range a {
			if a[i].Path != b[i].Path || !bytes.Equal(a[i].Content, b[i].Content) {
				t.Fatalf("%s: 두 번 생성한 결과가 다르다: %s", s, a[i].Path)
			}
		}
	}
}

func TestHeaderFirstLine(t *testing.T) {
	for _, s := range samples {
		_, files := generateSample(t, s)
		for _, f := range files {
			if !bytes.HasPrefix(f.Content, []byte(Header+"\n")) {
				t.Errorf("%s: 첫 줄이 생성 헤더가 아니다", f.Path)
			}
		}
	}
}

// TestGeneratedBuilds는 생성 파일과 손으로 쓴 스텁을 임시 모듈에 넣고 build, vet, test를 돌린다.
func TestGeneratedBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go가 PATH에 없다")
	}
	for _, s := range samples {
		t.Run(s, func(t *testing.T) {
			m, files := generateSample(t, s)
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "go.mod"), []byte("module "+m.GoModule+"\n\ngo 1.22\n"))
			for _, f := range files {
				writeFile(t, filepath.Join(dir, filepath.FromSlash(f.Path)), f.Content)
			}
			stubs := filepath.Join("testdata", s, "stubs")
			filepath.WalkDir(stubs, func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, _ := filepath.Rel(stubs, p)
				src, _ := os.ReadFile(p)
				writeFile(t, filepath.Join(dir, strings.TrimSuffix(rel, ".txt")), src)
				return nil
			})
			for _, args := range [][]string{{"build", "./..."}, {"vet", "./..."}, {"test", "./..."}} {
				cmd := exec.Command("go", args...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, out)
				}
			}
		})
	}
}

func writeFile(t *testing.T, p string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
}
