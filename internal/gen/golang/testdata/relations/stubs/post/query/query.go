package query

import "example.com/relations/model"

type PostReader struct{}

func (PostReader) ListPublished(page int) ([]model.Post, error) { return nil, nil }
func (PostReader) Count() (int, error)                          { return 0, nil }
