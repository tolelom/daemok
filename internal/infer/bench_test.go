package infer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// bigModule은 패키지 n개짜리 모듈을 만든다. pNNNN은 앞 패키지 두 개를 import한다.
func bigModule(b *testing.B, n int) string {
	b.Helper()
	root := b.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/big\n"), 0o644)
	for i := 0; i < n; i++ {
		dir := filepath.Join(root, "pkg", fmt.Sprintf("p%04d", i))
		os.MkdirAll(dir, 0o755)
		src := fmt.Sprintf("// Package p%04d는 벤치마크용이다.\npackage p%04d\n\nimport (\n", i, i)
		for _, j := range []int{i - 1, i - 7} {
			if j >= 0 {
				src += fmt.Sprintf("\t_ \"example.com/big/pkg/p%04d\"\n", j)
			}
		}
		src += ")\n\nvar ErrX = fmt.Errorf(\"x\")\n\nfunc F() int { return 1 }\n"
		for k := 0; k < 3; k++ {
			os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", k)), []byte(src), 0o644)
		}
	}
	return root
}

func BenchmarkAnalyze1000(b *testing.B) {
	root := bigModule(b, 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Analyze(root); err != nil {
			b.Fatal(err)
		}
	}
}
