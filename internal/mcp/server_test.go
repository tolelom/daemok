package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const design = `version: 0.5
go_module: example.com/m
tiers: [a, b]
modules:
  core: { tier: a, summary: "도메인" }
  api:  { tier: b, depends_on: [core] }
enums:
  kind: { in: core, values: [x, y] }
errors:
  ErrNotFound: { in: core, message: "not found" }
`

func project(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "design"), 0o755)
	os.WriteFile(filepath.Join(root, "design", "d.yaml"), []byte(design), 0o644)
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/m\n"), 0o644)
	return root
}

// exchange는 요청 줄들을 보내고 응답을 id 순서대로 준다.
func exchange(t *testing.T, root string, reqs ...string) []map[string]any {
	t.Helper()
	var out bytes.Buffer
	s := New(filepath.Join(root, "design"), root)
	if err := s.Serve(strings.NewReader(strings.Join(reqs, "\n")+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var res []map[string]any
	sc := bufio.NewScanner(&out)
	sc.Buffer(nil, 1<<24)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("응답이 JSON이 아니다: %v %s", err, sc.Text())
		}
		res = append(res, m)
	}
	return res
}

func errCode(m map[string]any) float64 {
	e, _ := m["error"].(map[string]any)
	c, _ := e["code"].(float64)
	return c
}

func TestProtocol(t *testing.T) {
	root := project(t)
	res := exchange(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":4,"method":"nope"}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nope","arguments":{}}}`,
		`{not json`,
	)
	if len(res) != 6 {
		t.Fatalf("알림에는 응답하지 않아 6개여야 한다: %d %v", len(res), res)
	}
	init := res[0]["result"].(map[string]any)
	if init["protocolVersion"] != "2025-03-26" || init["serverInfo"].(map[string]any)["name"] != "daemok" {
		t.Fatalf("%v", init)
	}
	if _, ok := init["capabilities"].(map[string]any)["tools"]; !ok {
		t.Fatalf("%v", init)
	}
	if res[1]["id"].(float64) != 2 || res[1]["result"] == nil {
		t.Fatalf("ping %v", res[1])
	}
	var names []string
	for _, tl := range res[2]["result"].(map[string]any)["tools"].([]any) {
		tool := tl.(map[string]any)
		names = append(names, tool["name"].(string))
		if tool["inputSchema"].(map[string]any)["type"] != "object" {
			t.Fatalf("%v", tool)
		}
	}
	if strings.Join(names, ",") != "design,check,draft" {
		t.Fatal(names)
	}
	if errCode(res[3]) != -32601 || errCode(res[4]) != -32602 || errCode(res[5]) != -32700 {
		t.Fatalf("%v %v %v", res[3], res[4], res[5])
	}
}

func call(t *testing.T, root, name, args string) (string, bool) {
	t.Helper()
	res := exchange(t, root, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"`+name+`","arguments":`+args+`}}`)
	r := res[0]["result"].(map[string]any)
	text := r["content"].([]any)[0].(map[string]any)["text"].(string)
	isErr, _ := r["isError"].(bool)
	return text, isErr
}

func TestDesignTool(t *testing.T) {
	root := project(t)
	text, isErr := call(t, root, "design", `{}`)
	if isErr || !strings.Contains(text, "d.yaml") || !strings.Contains(text, "api:  { tier: b, depends_on: [core] }") {
		t.Fatal(text)
	}
	text, isErr = call(t, root, "design", `{"module":"core"}`)
	var m struct {
		ID      string              `json:"id"`
		Kind    string              `json:"kind"`
		Tier    string              `json:"tier"`
		Summary string              `json:"summary"`
		UsedBy  []string            `json:"used_by"`
		Owned   map[string][]string `json:"owned"`
	}
	if err := json.Unmarshal([]byte(text), &m); isErr || err != nil {
		t.Fatal(text)
	}
	if m.Kind != "module" || m.Tier != "a" || m.Summary != "도메인" || strings.Join(m.UsedBy, ",") != "api" || strings.Join(m.Owned["enums"], ",") != "kind" || strings.Join(m.Owned["errors"], ",") != "ErrNotFound" {
		t.Fatalf("%+v", m)
	}
	if text, isErr := call(t, root, "design", `{"module":"zzz"}`); !isErr || !strings.Contains(text, "zzz") {
		t.Fatal(text)
	}
}

func TestCheckAndDraftTools(t *testing.T) {
	root := project(t)
	text, isErr := call(t, root, "check", `{}`)
	var c struct {
		Counts map[string]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(text), &c); isErr || err != nil || c.Counts["error"] == 0 {
		t.Fatalf("생성 파일이 없어 오류가 있어야 한다: %s", text)
	}
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n"), 0o644)
	text, isErr = call(t, root, "draft", `{}`)
	if isErr || !strings.Contains(text, "core: { tier: tier0 }") || !strings.Contains(text, `"modules": 1`) {
		t.Fatal(text)
	}
	if text, isErr := call(t, root, "draft", `{"root":"`+filepath.ToSlash(t.TempDir())+`"}`); !isErr || !strings.Contains(text, "Go 모듈이 아니다") {
		t.Fatal(text)
	}
}

func TestCheckToolMissingRoot(t *testing.T) {
	root := project(t)
	missing := filepath.ToSlash(filepath.Join(root, "nope"))
	text, _ := call(t, root, "check", `{"design":"`+filepath.ToSlash(filepath.Join(root, "design"))+`","root":"`+missing+`"}`)
	if !strings.Contains(text, "코드 루트가 없다") {
		t.Fatal(text)
	}
}
