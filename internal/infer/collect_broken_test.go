package infer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// 함수 본문에 문법 오류가 있는 파일(작업 중인 파일)도 import는 의존에 넣고, 건너뛴 이유를 남긴다.
func TestCollectKeepsImportsOfBrokenFile(t *testing.T) {
	root := t.TempDir()
	write := func(p, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		os.WriteFile(filepath.Join(root, p), []byte(s), 0o644)
	}
	write("go.mod", "module example.com/w\n\ngo 1.22\n")
	write("core/core.go", "package core\n")
	write("api/api.go", "package api\n\nimport \"example.com/w/core\"\n\nfunc F() { core.X( }\n")
	pkgs, skips, err := collect(root, "example.com/w")
	if err != nil {
		t.Fatal(err)
	}
	var api *pkg
	for _, p := range pkgs {
		if p.dir == "api" {
			api = p
		}
	}
	if api == nil || !reflect.DeepEqual(api.imports, []string{"core"}) {
		t.Fatalf("%+v", api)
	}
	if len(skips) != 1 || !strings.Contains(skips[0].Reason, "import만") {
		t.Fatalf("%+v", skips)
	}
}
