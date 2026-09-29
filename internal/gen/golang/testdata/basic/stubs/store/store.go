package store

type Kv struct{ data map[string][]byte }

func (k *Kv) Get(key string) ([]byte, error)     { return k.data[key], nil }
func (k *Kv) Set(key string, value []byte) error { k.data[key] = value; return nil }
