// Package yaml은 daemok 설계 파일이 쓰는 YAML 부분집합을 파싱한다.
// 모든 노드가 파일·줄·열 위치를 갖는다. 열은 바이트 기준 1-base다.
package yaml

import (
	"fmt"
	"strconv"
)

type Pos struct {
	File string
	Line int
	Col  int
}

func (p Pos) String() string { return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Col) }

type Error struct {
	Pos Pos
	Msg string
}

func (e *Error) Error() string { return e.Pos.String() + ": " + e.Msg }

type Kind int

const (
	Scalar Kind = iota
	Map
	Seq
)

type Node struct {
	Kind    Kind
	Pos     Pos
	Value   string // Scalar. 따옴표 문자열은 이스케이프를 풀어 담는다.
	Quoted  bool   // Scalar가 "..."였는지
	Entries []Entry
	Items   []*Node
}

type Entry struct {
	Key   string
	Pos   Pos // 키 위치
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
