package serve

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

var updateView = flag.Bool("update", false, "화면 모델 골든 파일을 다시 쓴다")

// 웹 화면이 받는 설계 모델(/api/state의 model)을 노드 종류가 모두 있는 두 설계로 고정한다.
// 깨지면 지도·노드 화면에서 액터, 인터페이스, 케이스, 흐름의 실패 시 되돌리기 같은 정보가 빠지거나 틀린다.
func TestViewGolden(t *testing.T) {
	samples := map[string]string{
		"tolchain": filepath.Join("..", "model", "testdata", "tolchain", "design"),
		"basic":    filepath.Join("..", "gen", "golang", "testdata", "basic", "design"),
	}
	for name, dir := range samples {
		t.Run(name, func(t *testing.T) {
			m, ds := model.Load(dir)
			if ds.HasErrors() {
				t.Fatalf("%v", ds)
			}
			got, err := json.MarshalIndent(toView(m), "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')
			golden := filepath.Join("testdata", "view", name+".json")
			if *updateView {
				os.MkdirAll(filepath.Dir(golden), 0o755)
				if err := os.WriteFile(golden, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("%s가 다르다. go test ./internal/serve -run TestViewGolden -update로 다시 쓴다", golden)
			}
		})
	}
}
