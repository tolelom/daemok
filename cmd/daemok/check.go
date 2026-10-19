package main

import (
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
	design, root := pathFlags(fs)
	strict := fs.Bool("strict", false, "경고도 실패로 본다")
	format := fs.String("format", "text", "출력 형식: text, json")
	if !parse(fs, args) {
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "알 수 없는 --format %q (text, json)\n", *format)
		return 2
	}
	r := workspace.Check(designDir(*design, *root), *root)
	if *format == "json" {
		stdout.Write(workspace.ReportJSON(r.Diags))
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
