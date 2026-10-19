package workspace

import (
	"encoding/json"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

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
		group, _, _ := strings.Cut(d.Message, ": ")
		out.Diagnostics = append(out.Diagnostics, diag{level[d.Level], d.Message, d.Pos.File, d.Pos.Line, d.Pos.Col, group})
	}
	for lv, name := range level {
		out.Counts[name] = diags.Count(lv)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	return append(b, '\n')
}
