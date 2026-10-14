// Package mcp는 표준 입출력으로 MCP(JSON-RPC 2.0, 한 줄에 메시지 하나)를 주고받아
// AI가 설계를 읽고 검사를 돌리게 한다. edit 외의 도구는 읽기 전용이고, edit는 --allow-write일 때만 있다.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/edit"
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
			Path   string `json:"path"`
			Kind   string `json:"kind"`
			Code   bool   `json:"code"`
			Base   string `json:"base"`
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
			if args.Op == "rename" {
				return renameTool(s.design, s.root, args.Kind, args.ID, args.Value, args.Code), nil
			}
			file, err := edit.ApplyConfined(s.design, args.Op, args.ID, args.Value)
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
		case "gen":
			text, err = genTool(design, root, args.Path)
		case "diff":
			text, err = diffTool(design, args.Base)
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
