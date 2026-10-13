package store

import "errors"

type Store struct{}

type Reader interface{ Read() }

var ErrGone = errors.New("gone")

var Default = 1

const Limit = 3

func New() *Store { return &Store{} }

func helper() {}

func (s *Store) Get() {}
