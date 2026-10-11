package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMCPCommand(t *testing.T) {
	var out, errb bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n")
	if code := runMCP(nil, in, &out, &errb); code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `"name":"daemok"`) || !strings.Contains(out.String(), `"protocolVersion":"2025-06-18"`) {
		t.Fatal(out.String())
	}
}
