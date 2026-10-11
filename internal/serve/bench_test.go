package serve

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkSignature1000은 감시 루프가 1초마다 계산하는 서명을 파일 3000개에서 잰다.
func BenchmarkSignature1000(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 1000; i++ {
		dir := filepath.Join(root, "pkg", fmt.Sprintf("p%04d", i))
		os.MkdirAll(dir, 0o755)
		for k := 0; k < 3; k++ {
			os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.go", k)), []byte("package p\n"), 0o644)
		}
	}
	design := filepath.Join(root, "design")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signature(design, root)
	}
}
