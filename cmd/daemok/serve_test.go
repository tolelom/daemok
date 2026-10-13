package main

import (
	"net"
	"testing"
)

func TestIsLoopback(t *testing.T) {
	for _, c := range []struct {
		addr string
		want bool
	}{{"127.0.0.1:7412", true}, {"[::1]:7412", true}, {"0.0.0.0:7412", false}, {"[::]:7412", false}, {"192.168.0.10:7412", false}} {
		a, err := net.ResolveTCPAddr("tcp", c.addr)
		if err != nil {
			t.Fatal(err)
		}
		if got := isLoopback(a); got != c.want {
			t.Errorf("%s: %v", c.addr, got)
		}
	}
	if isLoopback(&net.UnixAddr{Name: "x"}) {
		t.Error("TCP가 아닌 주소")
	}
}
