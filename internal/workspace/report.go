package workspace

import (
	"encoding/json"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// ReportJSON은 진단을 CI와 도구가 읽는 JSON으로 만든다. group은 메시지의 "설명: 대상"에서 설명 부분이다.
// check --format json과 MCP check 도구가 쓴다.
func ReportJSON(diags model.Diagnostics) []byte {
	type diag struct {
		Level string `json:"level"`
		Msg   string `json:"msg"`
		File  string `json:"file"`
		Line  int    `json:"line"`
		Col   int    `json:"col"`
		Group string `json:"group"`
	}
	level := map[model.Level]string{model.Error: "error", model.Warning: "warning", model.Info: "info"}
	out := struct {
		Diagnostics []diag         `json:"diagnostics"`
		Counts      map[string]int `json:"counts"`
	}{Diagnostics: []diag{}, Counts: map[string]int{}}
	for _, d := range diags {
		group, _, _ := strings.Cut(d.Msg, ": ")
		out.Diagnostics = append(out.Diagnostics, diag{level[d.Level], d.Msg, d.Pos.File, d.Pos.Line, d.Pos.Col, group})
	}
	for lv, name := range level {
		out.Counts[name] = diags.Count(lv)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return append(b, '\n')
}
