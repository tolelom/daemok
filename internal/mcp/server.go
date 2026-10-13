// Package mcp는 표준 입출력으로 MCP(JSON-RPC 2.0, 한 줄에 메시지 하나)를 주고받아
// AI가 설계를 읽고 검사를 돌리게 한다. 도구는 모두 읽기 전용이다.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/edit"
	"github.com/tolelom/daemok/internal/infer"
	"github.com/tolelom/daemok/internal/lift"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// defaultProtocol은 지원하는 MCP 프로토콜 중 최신이다. 클라이언트가 모르는 버전을 요청하면 이것으로 답한다.
const defaultProtocol = "2025-06-18"

var protocols = map[string]bool{"2025-06-18": true, "2025-03-26": true, "2024-11-05": true}

// maxLine은 메시지 한 줄의 최대 바이트다. 넘는 줄은 그 줄만 거부한다.
var maxLine = 16 << 20

// Server는 기본 설계 디렉토리와 코드 루트를 가진다. 도구 인자로 바꿀 수 있다.
type Server struct {
	design, root string
	Version      string
	// AllowWrite면 설계 파일을 고치는 edit 도구를 낸다. 사용자가 서버를 등록할 때 --allow-write로 켠다.
	AllowWrite bool
}

func New(design, root string) *Server { return &Server{design: design, root: root, Version: "dev"} }

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Serve는 r가 끝날 때까지 한 줄에 하나씩 요청을 읽고 w에 응답을 쓴다. 알림(id 없음)에는 응답하지 않는다.
func (s *Server) Serve(r io.Reader, w io.Writer) error {
	br := bufio.NewReader(r)
	enc := json.NewEncoder(w)
	for {
		raw, tooLong, err := readLine(br)
		if err != nil && err != io.EOF {
			return err
		}
		if tooLong {
			enc.Encode(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "메시지가 너무 길다"}})
		}
		line := strings.TrimSpace(string(raw))
		if line == "" || tooLong {
			if err == io.EOF {
				return nil
			}
			continue
		}
		if line[0] == '[' {
			enc.Encode(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32600, "배치 요청은 받지 않는다"}})
			continue
		}
		var req request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			enc.Encode(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "JSON을 읽을 수 없다"}})
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		result, rerr := s.handle(req)
		resp := response{JSONRPC: "2.0", ID: req.ID, Result: result, Error: rerr}
		if rerr == nil && result == nil {
			resp.Result = struct{}{}
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
		if err == io.EOF {
			return nil
		}
	}
}

// readLine은 줄바꿈까지 한 줄을 읽는다. maxLine을 넘으면 나머지를 버리고 tooLong을 준다.
func readLine(br *bufio.Reader) (line []byte, tooLong bool, err error) {
	for {
		chunk, err := br.ReadSlice('\n')
		if !tooLong {
			if len(line)+len(chunk) > maxLine {
				tooLong, line = true, nil
			} else {
				line = append(line, chunk...)
			}
		}
		if err != bufio.ErrBufferFull {
			return line, tooLong, err
		}
	}
}

func (s *Server) handle(req request) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(req.Params, &p)
		if !protocols[p.ProtocolVersion] {
			p.ProtocolVersion = defaultProtocol
		}
		return map[string]any{
			"protocolVersion": p.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "daemok", "version": s.Version},
		}, nil
	case "ping":
		return struct{}{}, nil
	case "tools/list":
		return map[string]any{"tools": toolList(s.AllowWrite)}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return nil, &rpcError{-32602, "params를 읽을 수 없다"}
		}
		var args struct {
			Module string `json:"module"`
			Design string `json:"design"`
			Root   string `json:"root"`
			Op     string `json:"op"`
			ID     string `json:"id"`
			Value  string `json:"value"`
			L3     bool   `json:"l3"`
		}
		if len(p.Arguments) > 0 && json.Unmarshal(p.Arguments, &args) != nil {
			return nil, &rpcError{-32602, "arguments를 읽을 수 없다"}
		}
		if p.Name == "edit" {
			if !s.AllowWrite {
				return nil, &rpcError{-32602, fmt.Sprintf("알 수 없는 도구 %q", p.Name)}
			}
			// 쓰기는 사용자가 등록한 서버의 설계 디렉토리에만 한다. 도구 인자로 경로를 바꾸지 못한다.
			if args.Design != "" || args.Root != "" {
				return toolResult("edit은 design·root 인자를 받지 않는다. 서버의 --design만 고친다", true), nil
			}
			file, err := edit.Apply(s.design, args.Op, args.ID, args.Value)
			if err != nil {
				return toolResult(err.Error(), true), nil
			}
			return toolResult("고쳤다: "+file+". 승인은 사용자의 커밋이다. check 도구로 진단을 확인한다", false), nil
		}
		design, root := s.design, s.root
		if args.Root != "" {
			root = args.Root
			if args.Design == "" {
				design = filepath.Join(root, "design")
			}
		}
		if args.Design != "" {
			design = args.Design
		}
		var text string
		var err error
		switch p.Name {
		case "design":
			text, err = designTool(design, args.Module)
		case "check":
			text = string(workspace.ReportJSON(workspace.Check(design, root).Diags))
		case "draft":
			if args.L3 {
				text, err = draftL3Tool(design, root)
			} else {
				text, err = draftTool(root)
			}
		default:
			return nil, &rpcError{-32602, fmt.Sprintf("알 수 없는 도구 %q", p.Name)}
		}
		if err != nil {
			return toolResult(err.Error(), true), nil
		}
		return toolResult(text, false), nil
	}
	return nil, &rpcError{-32601, fmt.Sprintf("지원하지 않는 메서드 %q", req.Method)}
}

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
	}
	for _, t := range tools {
		t["annotations"] = read
	}
	if allowWrite {
		tools = append(tools, map[string]any{"name": "edit",
			"description": "설계 파일의 L2를 한 곳 고친다. op는 tier(id의 tier를 value로), add_dep·remove_dep(id의 depends_on에 value를 더하거나 뺀다), add_module(id 모듈을 tier value로 더한다), remove_module(id 모듈을 지운다). id는 모듈이나 그룹. 서버의 설계 디렉토리만 고치고, 승인은 사용자의 커밋이다",
			"inputSchema": map[string]any{"type": "object", "required": []string{"op", "id"}, "properties": map[string]any{
				"op": map[string]any{"type": "string", "enum": edit.Ops}, "id": str("모듈이나 그룹 ID"), "value": str("tier 이름이나 의존 대상")}},
			"annotations": map[string]any{"readOnlyHint": false, "destructiveHint": true, "idempotentHint": false}})
	}
	return tools
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
