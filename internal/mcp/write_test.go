package mcp

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// send는 AllowWrite를 정한 서버에 요청 하나를 보내고 응답을 준다.
func send(t *testing.T, root string, allow bool, req string) map[string]any {
	t.Helper()
	var out bytes.Buffer
	s := New(filepath.Join(root, "design"), root)
	s.AllowWrite = allow
	if err := s.Serve(strings.NewReader(req+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatalf("%v %s", err, out.String())
	}
	return m
}

func toolNames(t *testing.T, root string, allow bool) map[string]map[string]any {
	t.Helper()
	res := send(t, root, allow, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	out := map[string]map[string]any{}
	for _, x := range res["result"].(map[string]any)["tools"].([]any) {
		tool := x.(map[string]any)
		out[tool["name"].(string)] = tool
	}
	return out
}

func TestEditToolOnlyWhenAllowed(t *testing.T) {
	root := project(t)
	if _, ok := toolNames(t, root, false)["edit"]; ok {
		t.Fatal("--allow-write 없이 edit 도구가 보인다")
	}
	tools := toolNames(t, root, true)
	ann, _ := tools["edit"]["annotations"].(map[string]any)
	if ann["readOnlyHint"] != false || ann["destructiveHint"] != true {
		t.Fatalf("%v", tools["edit"])
	}
	if ann, _ := tools["check"]["annotations"].(map[string]any); ann["readOnlyHint"] != true {
		t.Fatalf("%v", tools["check"])
	}
	res := send(t, root, false, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"tier","id":"api","value":"a"}}}`)
	if res["error"] == nil {
		t.Fatalf("허락 없이 edit을 받았다: %v", res)
	}
}

func TestEditTool(t *testing.T) {
	root := project(t)
	res := send(t, root, true, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"tier","id":"api","value":"a"}}}`)
	r := res["result"].(map[string]any)
	text := r["content"].([]any)[0].(map[string]any)["text"].(string)
	if r["isError"] == true || !strings.Contains(text, "d.yaml") {
		t.Fatal(text)
	}
	b, _ := os.ReadFile(filepath.Join(root, "design", "d.yaml"))
	if !strings.Contains(string(b), "api:  { tier: a, depends_on: [core] }") {
		t.Fatal(string(b))
	}
}

// edit은 도구 인자의 design·root를 받지 않고 서버의 설계 디렉토리만 고친다.
func TestEditToolIgnoresPaths(t *testing.T) {
	root, other := project(t), project(t)
	res := send(t, root, true, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"tier","id":"api","value":"a","design":"`+filepath.ToSlash(filepath.Join(other, "design"))+`"}}}`)
	if r := res["result"].(map[string]any); r["isError"] != true {
		t.Fatalf("%v", res)
	}
	if b, _ := os.ReadFile(filepath.Join(other, "design", "d.yaml")); !strings.Contains(string(b), "api:  { tier: b") {
		t.Fatal("다른 설계를 고쳤다")
	}
}

// 설계 파일이 설계 디렉토리 밖을 가리키는 심볼릭 링크면 edit은 고치지 않는다.
func TestEditToolRefusesSymlinkOutside(t *testing.T) {
	root := project(t)
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	link := filepath.Join(root, "design", "d.yaml")
	b, _ := os.ReadFile(link)
	os.WriteFile(outside, b, 0o644)
	os.Remove(link)
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("심볼릭 링크를 만들 수 없다: ", err)
	}
	res := send(t, root, true, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"tier","id":"api","value":"a"}}}`)
	if r := res["result"].(map[string]any); r["isError"] != true {
		t.Fatalf("설계 디렉토리 밖 파일을 고쳤다: %v", res)
	}
	if after, _ := os.ReadFile(outside); string(after) != string(b) {
		t.Fatal("링크 대상이 바뀌었다")
	}
}

// 설계 디렉토리 안을 가리키는 링크는 고친다.
func TestEditToolFollowsSymlinkInside(t *testing.T) {
	root := project(t)
	design := filepath.Join(root, "design")
	os.Rename(filepath.Join(design, "d.yaml"), filepath.Join(design, "real.yml"))
	if err := os.Symlink("real.yml", filepath.Join(design, "d.yaml")); err != nil {
		t.Skip("심볼릭 링크를 만들 수 없다: ", err)
	}
	res := send(t, root, true, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"tier","id":"api","value":"a"}}}`)
	if r := res["result"].(map[string]any); r["isError"] == true {
		t.Fatalf("%v", res)
	}
	if b, _ := os.ReadFile(filepath.Join(design, "real.yml")); !strings.Contains(string(b), "api:  { tier: a") {
		t.Fatal(string(b))
	}
}

func callText(t *testing.T, res map[string]any) (string, bool) {
	t.Helper()
	r := res["result"].(map[string]any)
	return r["content"].([]any)[0].(map[string]any)["text"].(string), r["isError"] == true
}

func TestEditToolRename(t *testing.T) {
	root := project(t)
	text, isErr := callText(t, send(t, root, true, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"rename","kind":"modules","id":"core","value":"base"}}}`))
	if isErr || !strings.Contains(text, "path: core") {
		t.Fatal(text)
	}
	b, _ := os.ReadFile(filepath.Join(root, "design", "d.yaml"))
	if !strings.Contains(string(b), `base: { tier: a, summary: "도메인", path: core }`) || !strings.Contains(string(b), "depends_on: [base]") {
		t.Fatal(string(b))
	}
}

func TestEditToolRenameRefusesSymlinkOutside(t *testing.T) {
	root := project(t)
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	link := filepath.Join(root, "design", "d.yaml")
	b, _ := os.ReadFile(link)
	os.WriteFile(outside, b, 0o644)
	os.Remove(link)
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("심볼릭 링크를 만들 수 없다: ", err)
	}
	if _, isErr := callText(t, send(t, root, true, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"edit","arguments":{"op":"rename","kind":"modules","id":"core","value":"base"}}}`)); !isErr {
		t.Fatal("설계 디렉토리 밖 파일을 고쳤다")
	}
	if after, _ := os.ReadFile(outside); string(after) != string(b) {
		t.Fatal("링크 대상이 바뀌었다")
	}
}
