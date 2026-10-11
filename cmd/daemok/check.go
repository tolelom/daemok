package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
	"github.com/tolelom/daemok/internal/yaml"
)

func runCheck(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	design := fs.String("design", "design", "설계 디렉토리")
	root := fs.String("root", ".", "코드 루트")
	strict := fs.Bool("strict", false, "경고도 실패로 본다")
	format := fs.String("format", "text", "출력 형식: text, json")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "알 수 없는 --format %q (text, json)\n", *format)
		return 2
	}
	r := workspace.Run(*design, *root)
	if *format == "json" {
		printJSON(stdout, r.Diags)
	} else {
		printDiags(stdout, r.Diags)
	}
	switch {
	case r.Fatal:
		return 2
	case r.Diags.HasErrors() || (*strict && r.Diags.Count(model.Warning) > 0):
		return 1
	}
	return 0
}

// printDiags는 진단을 한 줄씩 쓴다. 같은 수준·같은 설명("설명: 대상"의 설명)이 3개 이상이면
// 머리 한 줄 아래에 위치와 대상만 들여 쓴다.
func printDiags(w io.Writer, diags model.Diagnostics) {
	type key struct {
		lv    model.Level
		group string
	}
	keyOf := func(d model.Diagnostic) key {
		g, _, _ := strings.Cut(d.Msg, ": ")
		return key{d.Level, g}
	}
	count := map[key]int{}
	for _, d := range diags {
		count[keyOf(d)]++
	}
	done := map[key]bool{}
	for _, d := range diags {
		k := keyOf(d)
		if count[k] < 3 {
			fmt.Fprintln(w, d.String())
			continue
		}
		if done[k] {
			continue
		}
		done[k] = true
		fmt.Fprintf(w, "%s: %s %d개\n", k.lv, k.group, count[k])
		for _, x := range diags {
			if keyOf(x) == k {
				_, target, _ := strings.Cut(x.Msg, ": ")
				fmt.Fprintf(w, "  %s: %s\n", where(x.Pos), target)
			}
		}
	}
	fmt.Fprintf(w, "오류 %d, 경고 %d, 정보 %d\n", diags.Count(model.Error), diags.Count(model.Warning), diags.Count(model.Info))
}

func where(p yaml.Pos) string {
	if p.Line == 0 {
		return p.File
	}
	return p.String()
}

// printJSON은 진단을 CI와 도구가 읽는 JSON으로 쓴다. group은 메시지의 "설명: 대상"에서 설명 부분이다.
func printJSON(w io.Writer, diags model.Diagnostics) {
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
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
