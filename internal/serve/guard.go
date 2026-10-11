package serve

import "net/http"

func (s *Server) guard(next http.Handler) http.Handler { return next }
