package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/edit"
	"github.com/tolelom/daemok/internal/infer"
	"github.com/tolelom/daemok/internal/lift"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/rename"
	"github.com/tolelom/daemok/internal/workspace"
)

func toolResult(text string, isErr bool) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}, "isError": isErr}
}

func toolList(allowWrite bool) []map[string]any {
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	paths := map[string]any{"design": str("설계 디렉토리. 기본은 서버의 --design"), "root": str("코드 루트. 기본은 서버의 --root")}
	with := func(extra map[string]any) map[string]any {
		props := map[string]any{}
		for k, v := range paths {
			props[k] = v
		}
		for k, v := range extra {
			props[k] = v
		}
		return map[string]any{"type": "object", "properties": props}
	}
	read := map[string]any{"readOnlyHint": true}
	tools := []map[string]any{
		{"name": "design", "description": "설계를 읽는다. module이 없으면 설계 파일 원문, 있으면 그 모듈·그룹·멤버의 tier, 의존, 쓰는 쪽, 소속 노드를 JSON으로 준다",
			"inputSchema": with(map[string]any{"module": str("모듈, 그룹, 그룹 멤버 ID")})},
		{"name": "check", "description": "설계 검사, import 대조, 생성 파일 최신 여부를 돌려 진단을 JSON으로 준다(daemok check --format json과 같다)",
			"inputSchema": with(nil)},
		{"name": "draft", "description": "Go 모듈의 코드에서 L2 설계 초안과 분석 메모를 만든다. 설계 파일이 있어도 코드만 본다(daemok init --dry-run과 같다). l3가 참이면 설계에 없는 손으로 쓴 에러·상수·enum·struct·interface와 설계 에러를 반환하는 함수를 gen: false 노드 초안으로 만든다. 설계에 합쳐도 검사를 통과하는 노드만 담는다(daemok init --l3와 같다). 파일은 쓰지 않는다",
			"inputSchema": with(map[string]any{"l3": map[string]any{"type": "boolean", "description": "L3 초안을 만든다. design은 이때만 쓴다"}})},
		{"name": "gen", "description": "daemok gen이 쓰고 지울 생성 파일을 상태(same, new, changed, delete, refuse)와 함께 JSON으로 준다. path가 있으면 그 생성 파일의 생성할 내용을 준다. 설계를 고친 뒤 생성 코드를 확인하는 데 쓴다. 파일은 쓰지 않는다",
			"inputSchema": with(map[string]any{"path": str("생성 파일 경로(코드 루트 기준 슬래시 경로)")})},
	}
	for _, t := range tools {
		t["annotations"] = read
	}
	if allowWrite {
		tools = append(tools, map[string]any{"name": "edit",
			"description": "설계 파일을 고친다. op는 tier(id의 tier를 value로), add_dep·remove_dep(id의 depends_on에 value를 더하거나 뺀다), add_module(id 모듈을 tier value로 더한다), remove_module(id 모듈을 지운다), rename(kind 종류의 id 노드 ID와 모든 참조를 value로 바꾼다. kind와 value가 필요하다. code가 거짓이면 코드 이름을 고정 키로 유지하고 생성 코드가 그대로일 때만 쓴다). rename 외의 id는 모듈이나 그룹. 서버의 설계 디렉토리만 고치고, 승인은 사용자의 커밋이다",
			"inputSchema": map[string]any{"type": "object", "required": []string{"op", "id"}, "properties": map[string]any{
				"op": map[string]any{"type": "string", "enum": append(slices.Clone(edit.Ops), "rename")}, "id": str("모듈이나 그룹 ID. rename이면 옛 ID"), "value": str("tier 이름, 의존 대상, rename이면 새 ID"),
				"kind": map[string]any{"type": "string", "enum": rename.Kinds(), "description": "rename할 노드 종류(설계 파일의 키)"},
				"code": map[string]any{"type": "boolean", "description": "rename에서 생성 코드 이름도 새 ID를 따르게 한다"}}},
			"annotations": map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false}})
	}
	return tools
}

func renameTool(design, root, kind, id, value string, code bool) map[string]any {
	res, err := rename.Run(design, root, kind, id, value, rename.Options{Code: code, Confined: true})
	if err != nil {
		return toolResult(err.Error(), true)
	}
	lines := []string{"고쳤다: " + strings.Join(res.Files, ", ")}
	if res.Pinned != "" {
		lines = append(lines, "고정: "+res.Pinned)
	}
	for _, mv := range res.Moves {
		lines = append(lines, "옮길 디렉토리: "+mv)
	}
	for _, n := range res.Notes {
		lines = append(lines, "안내: "+n)
	}
	lines = append(lines, "승인은 사용자의 커밋이다. check 도구로 진단을 확인한다")
	return toolResult(strings.Join(lines, "\n"), false)
}

func genTool(design, root, path string) (string, error) {
	r := workspace.Run(design, root)
	if r.Plan == nil {
		return "", fmt.Errorf("설계 검사 오류가 있어 생성할 수 없다:\n%s", strings.Join(diagLines(r.Diags), "\n"))
	}
	if path != "" {
		content, ok := workspace.Content(r, path)
		if !ok {
			return "", fmt.Errorf("생성 파일이 아니다: %s", path)
		}
		return string(content), nil
	}
	b, _ := json.MarshalIndent(map[string]any{"files": workspace.Preview(r, root)}, "", "  ")
	return string(b), nil
}

func designTool(design, module string) (string, error) {
	if module == "" {
		paths, _ := filepath.Glob(filepath.Join(design, "*.yaml"))
		if len(paths) == 0 {
			return "", fmt.Errorf("설계 파일(*.yaml)이 없다: %s", design)
		}
		sort.Strings(paths)
		var b strings.Builder
		for _, p := range paths {
			src, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "# %s\n%s\n", filepath.Base(p), src)
		}
		return b.String(), nil
	}
	m, diags := model.Load(design)
	if diags.HasErrors() {
		return "", fmt.Errorf("설계를 읽을 수 없다:\n%s", strings.Join(diagLines(diags), "\n"))
	}
	info, ok := unitInfo(m, module)
	if !ok {
		return "", fmt.Errorf("모듈, 그룹, 그룹 멤버가 아니다: %s", module)
	}
	b, _ := json.MarshalIndent(info, "", "  ")
	return string(b), nil
}

func diagLines(ds model.Diagnostics) []string {
	var out []string
	for _, d := range ds {
		out = append(out, d.String())
	}
	return out
}

type unit struct {
	ID        string              `json:"id"`
	Kind      string              `json:"kind"` // module, group, member
	Tier      string              `json:"tier"`
	Path      string              `json:"path"`
	Summary   string              `json:"summary"`
	DependsOn []string            `json:"depends_on"`
	UsedBy    []string            `json:"used_by"`
	Owned     map[string][]string `json:"owned"`
}

func names(rs []model.Ref) []string {
	out := []string{}
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}

func unitInfo(m *model.Model, id string) (unit, bool) {
	var u unit
	switch {
	case m.Module(id) != nil:
		x := m.Module(id)
		u = unit{ID: id, Kind: "module", Tier: x.Tier.Name, Path: x.Path, Summary: x.Summary, DependsOn: names(x.DependsOn)}
	case m.Group(id) != nil:
		x := m.Group(id)
		u = unit{ID: id, Kind: "group", Tier: x.Tier.Name, Path: x.Path, Summary: x.Summary, DependsOn: names(x.DependsOn)}
	case m.Member(id) != nil:
		x := m.Member(id)
		u = unit{ID: id, Kind: "member", Tier: x.Group.Tier.Name, Path: x.Path(), Summary: x.Summary, DependsOn: names(x.Group.DependsOn)}
	default:
		return u, false
	}
	u.UsedBy = []string{}
	add := func(from string, deps []model.Ref) {
		for _, d := range deps {
			if d.Name == id {
				u.UsedBy = append(u.UsedBy, from)
			}
		}
	}
	for _, x := range m.Modules {
		add(x.ID, x.DependsOn)
	}
	for _, x := range m.Groups {
		add(x.ID, x.DependsOn)
	}
	sort.Strings(u.UsedBy)
	u.Owned = map[string][]string{}
	own := func(kind, nodeID, in string) {
		if in == id {
			u.Owned[kind] = append(u.Owned[kind], nodeID)
		}
	}
	for _, x := range m.Enums {
		own("enums", x.ID, x.In.Name)
	}
	for _, x := range m.Consts {
		own("consts", x.ID, x.In.Name)
	}
	for _, x := range m.Errors {
		own("errors", x.ID, x.In.Name)
	}
	for _, x := range m.Types {
		own("types", x.ID, x.In.Name)
	}
	for _, x := range m.Contracts {
		own("contracts", x.ID, x.In.Name)
	}
	for _, x := range m.HandlerKinds {
		own("handler_kinds", x.ID, x.In.Name)
	}
	for _, x := range m.EventKinds {
		own("event_kinds", x.ID, x.In.Name)
	}
	for _, x := range m.Funcs {
		own("funcs", x.ID, x.In.Name)
	}
	for _, x := range m.Flows {
		own("flows", x.ID, x.In.Name)
	}
	return u, true
}

// draftL3Tool은 L3 초안 YAML과, 옮기지 않은 선언·설계 에러를 반환하지 않는 함수 수의 JSON을 ---로 나눠 준다.
func draftL3Tool(design, root string) (string, error) {
	r, err := lift.Draft(design, root)
	if err != nil {
		return "", err
	}
	rep, _ := json.MarshalIndent(map[string]any{"skipped": r.Skipped, "funcs": r.Funcs, "coverage": r.Coverage}, "", "  ")
	return string(r.YAML) + "\n---\n" + string(rep), nil
}

func draftTool(root string) (string, error) {
	a, err := infer.Analyze(root)
	if err != nil {
		return "", err
	}
	rep, _ := json.MarshalIndent(a.Report, "", "  ")
	return string(a.YAML) + "\n---\n" + string(rep), nil
}
