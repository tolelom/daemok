package svc

import s "example.com/uses/store"

func Run() error {
	st := s.New()
	_ = st
	_ = s.Limit
	_ = s.Missing
	return s.ErrGone
}
