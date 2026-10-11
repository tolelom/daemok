package mcp

import (
	"strings"
	"testing"
)

// 너무 긴 줄은 그 줄만 거부하고 다음 요청을 계속 받는다.
func TestLongLineDoesNotStopServer(t *testing.T) {
	old := maxLine
	maxLine = 64
	defer func() { maxLine = old }()
	root := project(t)
	res := exchange(t, root,
		`{"jsonrpc":"2.0","id":1,"method":"ping","params":{"pad":"`+strings.Repeat("x", 200)+`"}}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if len(res) != 2 || errCode(res[0]) != -32700 || res[1]["id"] != float64(2) || res[1]["error"] != nil {
		t.Fatalf("%v", res)
	}
}

// 배치(JSON 배열)는 받지 않는다는 뜻으로 -32600을 준다.
func TestBatchRejectedAsInvalidRequest(t *testing.T) {
	res := exchange(t, project(t), `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`)
	if len(res) != 1 || errCode(res[0]) != -32600 {
		t.Fatalf("%v", res)
	}
}

// 아는 버전이면 그대로, 모르는 버전이면 지원하는 최신 버전으로 답한다.
func TestInitializeNegotiatesVersion(t *testing.T) {
	for _, c := range []struct{ asked, want string }{{"2025-03-26", "2025-03-26"}, {"2099-01-01", defaultProtocol}, {"", defaultProtocol}} {
		res := exchange(t, project(t), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+c.asked+`"}}`)
		got := res[0]["result"].(map[string]any)["protocolVersion"]
		if got != c.want {
			t.Fatalf("%q → %v", c.asked, got)
		}
	}
}
