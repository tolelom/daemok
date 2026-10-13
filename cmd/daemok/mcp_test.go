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

func TestMCPAllowWriteFlag(t *testing.T) {
	list := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
	for _, c := range []struct {
		args []string
		edit bool
	}{{nil, false}, {[]string{"--allow-write"}, true}} {
		var out, errb bytes.Buffer
		if code := runMCP(c.args, strings.NewReader(list), &out, &errb); code != 0 {
			t.Fatalf("%d %s", code, errb.String())
		}
		if got := strings.Contains(out.String(), `"name":"edit"`); got != c.edit {
			t.Errorf("%v: edit %v\n%s", c.args, got, out.String())
		}
	}
}
