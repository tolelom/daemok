package serve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// 같은 버전으로 동시에 온 편집은 하나만 받아들이고 나머지는 409로 돌려보낸다.
// 200을 받은 편집은 모두 파일에 남아야 한다.
func TestEditConcurrentSameVersion(t *testing.T) {
	src := `version: 0.5
go_module: example.com/x
tiers: [a, b]
modules:
  m1: { tier: a }
  m2: { tier: a }
  m3: { tier: a }
  m4: { tier: a }
  m5: { tier: a }
  m6: { tier: a }
  web: { tier: b, depends_on: [m1, m2, m3, m4, m5, m6] }
`
	s, dir := setup(t, src)
	local(t, s)
	v := s.Version()
	codes := make([]int, 6)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], _ = editReq(t, s, "remove_dep", "web", fmt.Sprintf("m%d", i+1), v)
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, c := range codes {
		switch c {
		case 200:
			ok++
		case 409:
		default:
			t.Fatalf("codes %v", codes)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "d.yaml"))
	line := ""
	for _, l := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "web:") {
			line = l
		}
	}
	if ok != 1 || strings.Count(line, "m") != 5 {
		t.Fatalf("codes %v, %q", codes, line)
	}
}
