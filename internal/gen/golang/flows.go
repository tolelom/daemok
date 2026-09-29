package golang

import (
	"fmt"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

const flowHelpers = `// FlowError는 흐름의 한 단계가 실패했음을 알린다. Unwrap은 단계가 돌려준 오류다.
type FlowError struct {
	Flow string
	Step string
	Err  error
}

func (e *FlowError) Error() string { return e.Flow + ": " + e.Step + ": " + e.Err.Error() }

func (e *FlowError) Unwrap() error { return e.Err }

// daemokUndo는 되돌리기 하나다. 실패하면 이름으로 Fatal에 알린다.
type daemokUndo struct {
	name string
	fn   func() error
}`

func (g *gen) flows() error {
	helpers := map[string]bool{}
	for _, fl := range g.m.Flows {
		if !fl.Gen {
			continue
		}
		dir := g.ownerPath(fl.In.Name)
		if !helpers[dir] {
			helpers[dir] = true
			g.file(dir+"/daemok_flow.gen.go").add("%s", flowHelpers)
		}
		f := g.file(dir + "/" + fl.ID + ".gen.go")
		name := model.GoName(fl.ID)
		f.add("%s", flowInterface(fl, name))
		f.add("%s", flowRun(fl, name))
	}
	return nil
}

// flowInterface는 단계와 되돌리기 메서드를 처음 나온 순서로 중복 없이 담는다.
func flowInterface(fl *model.Flow, name string) string {
	var lines []string
	seen := map[string]bool{}
	add := func(id, doc string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		line := "\t" + model.GoName(id) + "() error"
		if doc != "" {
			line = "\t// " + doc + "\n" + line
		}
		lines = append(lines, line)
	}
	for _, s := range fl.Steps {
		add(s.ID, s.Summary)
		add(s.Undo, "")
	}
	lines = append(lines, "\t// Fatal은 되돌릴 수 없는 실패에서 불린다. 돌아오지 않는 것을 전제로 한다.\n\tFatal(step string, err error)")
	doc := fmt.Sprintf("// %sSteps는 %s 흐름의 단계다. 본문은 손으로 쓴다.", name, fl.ID)
	if fl.Summary != "" {
		doc += "\n// " + fl.Summary
	}
	return fmt.Sprintf("%s\ntype %sSteps interface {\n%s\n}", doc, name, strings.Join(lines, "\n"))
}

// flowRun은 단계를 순서대로 부르는 함수다.
// 커밋 전 단계가 실패하면 쌓인 되돌리기를 역순으로 부르고, fatal 단계는 되돌리지 않고 Fatal을 부른다.
func flowRun(fl *model.Flow, name string) string {
	hasUndo, needFail := false, false
	committed := false
	for _, s := range fl.Steps {
		if !committed && s.Undo != "" {
			hasUndo = true
		}
		if !committed && !s.Fatal {
			needFail = true
		}
		if s.Commit {
			committed = true
		}
	}
	flowErr := func(step string) string {
		return fmt.Sprintf("&FlowError{Flow: %q, Step: %q, Err: err}", fl.ID, step)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "// Run%s는 %s 흐름의 단계를 설계 순서대로 부른다.\nfunc Run%s(s %sSteps) error {\n", name, fl.ID, name, name)
	if hasUndo {
		b.WriteString("\tvar undo []daemokUndo\n")
	}
	if needFail {
		b.WriteString("\tfail := func(step string, err error) error {\n")
		if hasUndo {
			b.WriteString("\t\tfor i := len(undo) - 1; i >= 0; i-- {\n\t\t\tif uerr := undo[i].fn(); uerr != nil {\n\t\t\t\ts.Fatal(undo[i].name, uerr)\n\t\t\t}\n\t\t}\n")
		}
		fmt.Fprintf(&b, "\t\treturn &FlowError{Flow: %q, Step: step, Err: err}\n\t}\n", fl.ID)
	}
	committed = false
	for _, s := range fl.Steps {
		call := "s." + model.GoName(s.ID) + "()"
		switch {
		case s.Fatal:
			fmt.Fprintf(&b, "\tif err := %s; err != nil {\n\t\ts.Fatal(%q, err)\n\t\treturn %s\n\t}\n", call, s.ID, flowErr(s.ID))
		case committed:
			fmt.Fprintf(&b, "\tif err := %s; err != nil {\n\t\treturn %s\n\t}\n", call, flowErr(s.ID))
		default:
			fmt.Fprintf(&b, "\tif err := %s; err != nil {\n\t\treturn fail(%q, err)\n\t}\n", call, s.ID)
		}
		if !committed && s.Undo != "" && !s.Commit {
			fmt.Fprintf(&b, "\tundo = append(undo, daemokUndo{%q, s.%s})\n", s.Undo, model.GoName(s.Undo))
		}
		if s.Commit {
			committed = true
			if hasUndo {
				b.WriteString("\tundo = nil\n")
			}
		}
	}
	b.WriteString("\treturn nil\n}")
	return b.String()
}
