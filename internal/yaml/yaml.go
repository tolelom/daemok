package yaml

import (
	"fmt"
	"strconv"
)

type Pos struct {
	File   string
	Line   int
	Column int
}

func (p Pos) String() string { return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column) }

func (p Pos) Less(q Pos) bool {
	if p.File != q.File {
		return p.File < q.File
	}
	if p.Line != q.Line {
		return p.Line < q.Line
	}
	return p.Column < q.Column
}

type Error struct {
	Pos     Pos
	Message string
}

func (e *Error) Error() string { return e.Pos.String() + ": " + e.Message }

type Kind int

const (
	Scalar Kind = iota
	Map
	Seq
)

type Node struct {
	Kind    Kind
	Pos     Pos
	Value   string
	Quoted  bool
	Entries []Entry
	Items   []*Node
}

type Entry struct {
	Key   string
	Pos   Pos
	Value *Node
}

func (n *Node) Get(key string) *Node {
	if n == nil || n.Kind != Map {
		return nil
	}
	for i := range n.Entries {
		if n.Entries[i].Key == key {
			return n.Entries[i].Value
		}
	}
	return nil
}

func (n *Node) IsNull() bool {
	return n != nil && n.Kind == Scalar && !n.Quoted && (n.Value == "" || n.Value == "null")
}

func (n *Node) Int() (int, bool) {
	if n == nil || n.Kind != Scalar || n.Quoted {
		return 0, false
	}
	v, err := strconv.Atoi(n.Value)
	return v, err == nil
}

func (n *Node) Bool() (bool, bool) {
	if n == nil || n.Kind != Scalar || n.Quoted {
		return false, false
	}
	switch n.Value {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}
