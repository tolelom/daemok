package serve

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuard(t *testing.T) {
	s, _ := setup(t, design)
	s.SetOptions(Options{Local: true})
	cases := []struct {
		name, method, path, host, origin, ctype string
		want                                    int
	}{
		{"다른 Host의 GET", "GET", "/api/state", "evil.example:7412", "", "", 403},
		{"localhost GET", "GET", "/api/state", "localhost:7412", "", "", 200},
		{"IPv6 루프백 GET", "GET", "/api/state", "[::1]:7412", "", "", 200},
		{"Origin 없는 POST", "POST", "/api/init", "127.0.0.1:7412", "", "application/json", 403},
		{"다른 Origin POST", "POST", "/api/init", "127.0.0.1:7412", "http://evil.example", "application/json", 403},
		{"text/plain POST", "POST", "/api/init", "127.0.0.1:7412", "http://127.0.0.1:7412", "text/plain", 415},
		{"같은 Origin POST", "POST", "/api/init", "127.0.0.1:7412", "http://127.0.0.1:7412", "application/json", 409},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, strings.NewReader("{}"))
		req.Host = c.host
		if c.origin != "" {
			req.Header.Set("Origin", c.origin)
		}
		if c.ctype != "" {
			req.Header.Set("Content-Type", c.ctype)
		}
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: %d, want %d", c.name, rec.Code, c.want)
		}
	}
}

func TestGuardOffWhenNotLocal(t *testing.T) {
	s, _ := setup(t, design)
	req := httptest.NewRequest("GET", "/api/state", nil)
	req.Host = "192.168.0.5:7412"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d", rec.Code)
	}
}
