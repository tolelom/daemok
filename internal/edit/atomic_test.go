package edit

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// 다른 쪽이 대상 파일을 읽고 있어도 쓰기는 성공한다(Windows는 열린 파일로 이름을 바꾸지 못한다).
func TestWriteAtomicWhileRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.yaml")
	os.WriteFile(p, []byte("x"), 0o644)
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				os.ReadFile(p)
			}
		}
	}()
	defer func() { close(stop); <-done }()
	for i := 0; i < 200; i++ {
		if err := WriteAtomic(p, []byte(fmt.Sprint(i))); err != nil {
			t.Fatalf("%d: %v", i, err)
		}
	}
}
