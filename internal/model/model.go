package model

import "github.com/tolelom/daemok/internal/yaml"

type Ref struct {
	Name string
	Pos  yaml.Pos
}

type Meta struct {
	ID        string
	Pos       yaml.Pos
	Summary   string
	Satisfies []Ref
}

type Requirement struct {
	Meta
	Text   string
	Status string
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
	Role       string
	DependsOn  []Ref
	Implements []Ref
	Needs      []Need
}

type Need struct {
	Contract Ref
	Uses     []Ref
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

type Owned struct {
	Meta
	In  Ref
	Gen bool
}

type Enum struct {
	Owned
	Values []Ref
	Go     GoOpts
	Export []string
}

type Const struct {
	Owned
	Type     string
	Value    string
	ValuePos yaml.Pos
	Go       GoOpts
	Export   []string
}

type ErrorDef struct {
	Owned
	Message string
	HTTP    int
	Go      GoOpts
}

type Field struct {
	Name     string
	Pos      yaml.Pos
	Type     Ref
	RefType  Ref
	Many     bool
	Optional bool
	Max      Ref
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
	HasErrors bool
}

type Contract struct {
	Owned
	Kind    string
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
	Given  *yaml.Node
	Expect *yaml.Node
	Error  Ref
}

type Func struct {
	Owned
	Kind    Ref
	Handles Ref
	Of      Ref
	Errors  []Ref
	Cases   []*Case
	Go      GoOpts
}

type EventKind struct {
	Owned
	Key         Ref
	Encoding    string
	Subscribers []Ref
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
	Calls   []Ref
	Emits   []Ref
}

type Flow struct {
	Owned
	On    Ref
	Steps []*Step
}

type CPP struct {
	IncludeDirs []Ref
	Pos         yaml.Pos
}

type Model struct {
	Version  string
	GoModule string
	TSOut    string
	TSOutPos yaml.Pos
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
