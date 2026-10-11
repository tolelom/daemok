package yaml

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FuzzParse는 어떤 입력에도 Parse가 멈추거나 패닉하지 않고, 오류면 *Error를 주는지 본다.
func FuzzParse(f *testing.F) {
	seeds, _ := filepath.Glob(filepath.Join("..", "model", "testdata", "*", "design", "*.yaml"))
	for _, p := range seeds {
		if b, err := os.ReadFile(p); err == nil {
			f.Add(b)
		}
	}
	for _, s := range []string{"a: 1\n", "a: [b, {c: d}]\n", "a:\n  - b\n  - c: \"x\\\"y\"\n", "a: |\n", "\ufeffa: b\n", "a: { b: [c, d], e: f }\n"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, src []byte) {
		start := time.Now()
		n, err := Parse("f.yaml", src)
		if d := time.Since(start); d > time.Second {
			t.Fatalf("%d바이트 입력에 %v 걸렸다", len(src), d)
		}
		if err != nil {
			if _, ok := err.(*Error); !ok {
				t.Fatalf("오류 타입 %T", err)
			}
			return
		}
		if n == nil {
			t.Fatal("오류 없이 nil")
		}
	})
}
