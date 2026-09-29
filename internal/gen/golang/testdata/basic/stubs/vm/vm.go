package vm

import "example.com/basic/core"

type Context struct{}

var handlers = map[core.TxType]Handler{}

func Register(key core.TxType, h Handler) { handlers[key] = h }
