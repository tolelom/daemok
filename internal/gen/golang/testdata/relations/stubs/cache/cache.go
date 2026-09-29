package cache

type PostCache struct{}

func (*PostCache) Get(key string) ([]byte, error) { return nil, nil }
