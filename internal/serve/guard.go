package serve

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// guard는 루프백에서 들을 때 다른 사이트가 브라우저를 통해 보내는 요청을 막는다.
// Host가 루프백이 아니면 DNS rebinding으로 보고, POST의 Origin이 Host와 다르면 CSRF로 본다.
func (s *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.options().Local {
			if !loopbackHost(r.Host) {
				http.Error(w, "허용하지 않는 Host", http.StatusForbidden)
				return
			}
			if r.Method == http.MethodPost {
				u, err := url.Parse(r.Header.Get("Origin"))
				if err != nil || u.Host == "" || u.Host != r.Host {
					http.Error(w, "허용하지 않는 Origin", http.StatusForbidden)
					return
				}
				if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
					http.Error(w, "JSON 본문만 받는다", http.StatusUnsupportedMediaType)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func loopbackHost(hostport string) bool {
	h, _, err := net.SplitHostPort(hostport)
	if err != nil {
		h = hostport
	}
	h = strings.Trim(h, "[]")
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
