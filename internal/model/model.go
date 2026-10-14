// Package model은 설계 파일을 노드 타입으로 옮기고 여러 파일을 하나로 합친다.
package model

import "github.com/tolelom/daemok/internal/yaml"

// Ref는 다른 노드를 가리키는 이름과 그 위치다. 해석은 check가 한다.
type Ref struct {
	Name string
	Pos  yaml.Pos
}

type Meta struct {
	ID        string
	Pos       yaml.Pos // ID 키 위치
	Summary   string
	Satisfies []Ref
}

type Requirement struct {
	Meta
	Text   string
	Status string // active, held, dropped
}

type Actor struct{ Meta }

type Interface struct {
	Meta
	Protocol string
	Port     int
	Path     string
	Spec     string
	UsedBy   []Ref
}

type System struct {
	Meta
	Actors     []*Actor
	Interfaces []*Interface
}

type Executable struct {
	Meta
	Exposes  []Ref
	Composes []Ref
}

type Module struct {
	Meta
	Tier       Ref
	Path       string
	Role       string // data, infra, adapter. 비면 도메인 모듈
	DependsOn  []Ref
	Implements []Ref
	Needs      []Need
}

// Need는 받는 쪽이 적은 계약 하나다.
type Need struct {
	Contract Ref
	Uses     []Ref // 비면 계약의 메서드 전부
	Why      string
	Pos      yaml.Pos
}

type Group struct {
	Meta
	Tier      Ref
	Kind      Ref
	Path      string
	DependsOn []Ref
	Needs     []Need
	Rule      string
	Members   []*Member
}

type Member struct {
	Meta
	Group      *Group
	Handles    []Ref
	Exceptions []Exception
	Needs      []Need
}

type Exception struct {
	DependsOn Ref
	Reason    string
}

type GoOpts struct {
	Type     string
	Name     string
	Prefix   string
	Register string
}

// Owned는 모듈이나 멤버에 속하는 L3·L4 노드의 공통 부분이다.
type Owned struct {
	Meta
	In  Ref
	Gen bool // 기본 true. false면 생성에서 뺀다
}

type Enum struct {
	Owned
	Values []Ref
	Go     GoOpts
	Export []string
}

type Const struct {
	Owned
	Type     string // int, string, duration
	Value    string
	ValuePos yaml.Pos
	Go       GoOpts
	Export   []string
}

type ErrorDef struct {
	Owned
	Message string
	HTTP    int // 0이면 없음
	Go      GoOpts
}

type Field struct {
	Name     string
	Pos      yaml.Pos
	Type     Ref // type 또는 ref 중 하나만 채워진다
	RefType  Ref
	Many     bool
	Optional bool
	Max      Ref // 상수 ID 또는 숫자
	Summary  string
}

type Type struct {
	Owned
	Fields []*Field
	Go     GoOpts
	Export []string
}

type Param struct {
	Name string
	Pos  yaml.Pos
	Type Ref
}

type Method struct {
	Name      string
	Pos       yaml.Pos
	Params    []*Param
	Returns   []Ref
	Errors    []Ref
	HasErrors bool // errors 키가 있었는지
}

type Contract struct {
	Owned
	Kind    string // call, query, port, external
	Go      GoOpts
	Methods []*Method
}

type HandlerKind struct {
	Owned
	Params  []*Param
	Returns []Ref
	Key     Ref
	Go      GoOpts
}

type Case struct {
	Name   string
	Pos    yaml.Pos
	Given  *yaml.Node // 맵
	Expect *yaml.Node // 맵 또는 nil
	Error  Ref        // 에러 ID, "invalid", 또는 비어 있음
}

type Func struct {
	Owned
	Kind    Ref
	Handles Ref
	Of      Ref // "계약.메서드"
	Errors  []Ref
	Cases   []*Case
	Go      GoOpts // 일반 함수의 go.name
}

type EventKind struct {
	Owned
	Key         Ref
	Encoding    string // map, json
	Subscribers []Ref  // 묶음의 모든 이벤트를 받는 모듈
}

type Event struct {
	Meta
	Kind        Ref
	Publishers  []Ref
	Subscribers []Ref
	Data        []*Field
}

type Step struct {
	ID      string
	Pos     yaml.Pos
	Summary string
	Undo    string
	UndoPos yaml.Pos
	Commit  bool
	Fatal   bool
	Calls   []Ref // 계약.메서드
	Emits   []Ref
}

type Flow struct {
	Owned
	On    Ref
	Steps []*Step
}

// CPP는 C++ include 검사 설정이다. 모듈 path는 각 include 디렉토리 아래 폴더다.
type CPP struct {
	IncludeDirs []Ref // 코드 루트 기준 상대 경로
	Pos         yaml.Pos
}

type Model struct {
	Version  string
	GoModule string
	TSOut    string
	TSOutPos yaml.Pos // ts_out 값 위치
	CPP      *CPP

	Requirements []*Requirement
	System       *System
	Executables  []*Executable
	Tiers        []Ref
	Modules      []*Module
	Groups       []*Group
	Enums        []*Enum
	Consts       []*Const
	Errors       []*ErrorDef
	Types        []*Type
	Contracts    []*Contract
	HandlerKinds []*HandlerKind
	EventKinds   []*EventKind
	Events       []*Event
	Funcs        []*Func
	Flows        []*Flow
}
