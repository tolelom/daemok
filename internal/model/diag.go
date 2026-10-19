package model

import (
	"fmt"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

type Level int

const (
	Info Level = iota
	Warning
	Error
)

func (l Level) String() string {
	switch l {
	case Error:
		return "오류"
	case Warning:
		return "경고"
	}
	return "정보"
}

type Diagnostic struct {
	Pos     yaml.Pos
	Level   Level
	Message string
}

func (d Diagnostic) String() string {
	if d.Pos.Line == 0 {
		return fmt.Sprintf("%s: %s: %s", d.Pos.File, d.Level, d.Message)
	}
	return fmt.Sprintf("%s: %s: %s", d.Pos, d.Level, d.Message)
}

type Diagnostics []Diagnostic

func (ds *Diagnostics) Add(pos yaml.Pos, level Level, format string, args ...any) {
	*ds = append(*ds, Diagnostic{Pos: pos, Level: level, Message: fmt.Sprintf(format, args...)})
}

func (ds *Diagnostics) Errorf(pos yaml.Pos, format string, args ...any) {
	ds.Add(pos, Error, format, args...)
}

func (ds *Diagnostics) Warnf(pos yaml.Pos, format string, args ...any) {
	ds.Add(pos, Warning, format, args...)
}

func (ds *Diagnostics) Infof(pos yaml.Pos, format string, args ...any) {
	ds.Add(pos, Info, format, args...)
}

func (ds Diagnostics) HasErrors() bool { return ds.Count(Error) > 0 }

func (ds Diagnostics) Count(level Level) int {
	n := 0
	for _, d := range ds {
		if d.Level == level {
			n++
		}
	}
	return n
}

func (ds Diagnostics) Sort() {
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].Pos.Less(ds[j].Pos) })
}
