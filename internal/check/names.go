package check

import (
	"math"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

var (
	goIdent       = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	exportedIdent = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	valueChars    = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	durationValue = regexp.MustCompile(`^\d+(ms|s|m|h)$`)
)

// names는 생성 코드가 컴파일되도록 Go 이름과 경로의 규칙을 본다.
func (c *checker) names() {
	c.goModule()
	c.paths()
	c.cppDirs()
	c.nodeIDs()
	c.enumValues()
	c.errorIDs()
	c.funcNames()
	c.fieldNames()
	c.durations()
	c.packageNames()
}

// goModule은 생성 코드가 다른 패키지를 import할 수 있는 설계에 go_module이 있는지 본다.
func (c *checker) goModule() {
	m := c.m
	if m.GoModule != "" {
		return
	}
	var first *yaml.Pos
	consider := func(p yaml.Pos) {
		if first == nil || p.Less(*first) {
			q := p
			first = &q
		}
	}
	for _, x := range m.Groups {
		consider(x.Pos)
	}
	for _, x := range m.Enums {
		consider(x.Pos)
	}
	for _, x := range m.Consts {
		consider(x.Pos)
	}
	for _, x := range m.Errors {
		consider(x.Pos)
	}
	for _, x := range m.Types {
		consider(x.Pos)
	}
	for _, x := range m.Contracts {
		consider(x.Pos)
	}
	for _, x := range m.HandlerKinds {
		consider(x.Pos)
	}
	for _, x := range m.EventKinds {
		consider(x.Pos)
	}
	for _, x := range m.Funcs {
		consider(x.Pos)
	}
	for _, x := range m.Flows {
		consider(x.Pos)
	}
	if first != nil {
		c.errorf(*first, "필요한 키가 없다: %q", "go_module")
	}
}

// cppDirs는 include 디렉토리가 있고 코드 루트 안의 상대 경로인지 본다.
func (c *checker) cppDirs() {
	if c.m.CPP == nil {
		return
	}
	if len(c.m.CPP.IncludeDirs) == 0 {
		c.errorf(c.m.CPP.Pos, "include_dirs가 비어 있다")
	}
	for _, d := range c.m.CPP.IncludeDirs {
		p := path.Clean(d.Name)
		if d.Name == "" || path.IsAbs(d.Name) || strings.Contains(d.Name, `\`) || p == ".." || strings.HasPrefix(p, "../") {
			c.errorf(d.Pos, "코드 루트 기준 상대 경로가 아닌 include 디렉토리: %s", d.Name)
		}
	}
}

// paths는 모듈·그룹 path와 멤버 ID가 Go 패키지 경로로 쓸 수 있고 생성 파일 범위 안인지 본다.
func (c *checker) paths() {
	check := func(p string, pos yaml.Pos) {
		for _, el := range strings.Split(p, "/") {
			if el == "testdata" || el == "vendor" || strings.HasPrefix(el, ".") || strings.HasPrefix(el, "_") {
				c.errorf(pos, "생성 파일 범위에서 빠지는 path: %s", p)
				return
			}
			if !goIdent.MatchString(el) {
				c.errorf(pos, "Go 패키지 경로로 쓸 수 없는 path: %s", p)
				return
			}
		}
	}
	for _, mod := range c.m.Modules {
		check(mod.Path, mod.Pos)
	}
	for _, g := range c.m.Groups {
		check(g.Path, g.Pos)
		for _, mem := range g.Members {
			check(mem.ID, mem.Pos)
		}
	}
}

// nodeIDs는 Go 이름과 파일 이름이 되는 ID가 소문자 snake_case인지, 타입·enum ID가 내장 타입 이름과 겹치지 않는지 본다.
func (c *checker) nodeIDs() {
	check := func(kind string, meta model.Meta) {
		if !snakeName.MatchString(meta.ID) {
			c.errorf(meta.Pos, "Go 이름으로 바꿀 수 없는 ID: %s %s", kind, meta.ID)
		}
	}
	shadow := func(kind string, meta model.Meta) {
		if model.Builtins[meta.ID] != "" {
			c.errorf(meta.Pos, "내장 타입 이름과 같은 ID: %s %s (타입 참조는 내장 타입을 먼저 찾는다)", kind, meta.ID)
		}
	}
	m := c.m
	for _, x := range m.Enums {
		check("enum", x.Meta)
		shadow("enum", x.Meta)
	}
	for _, x := range m.Consts {
		check("상수", x.Meta)
	}
	for _, x := range m.Types {
		check("타입", x.Meta)
		shadow("타입", x.Meta)
	}
	for _, x := range m.Contracts {
		check("계약", x.Meta)
	}
	for _, x := range m.HandlerKinds {
		check("핸들러 종류", x.Meta)
	}
	for _, x := range m.EventKinds {
		check("이벤트 종류", x.Meta)
	}
	for _, x := range m.Events {
		check("이벤트", x.Meta)
	}
	for _, x := range m.Funcs {
		check("함수", x.Meta)
	}
	for _, x := range m.Flows {
		check("흐름", x.Meta)
	}
}

// enumValues는 enum 값이 Go 상수 이름이 되는지, enum 안에서 상수 이름이 겹치지 않는지 본다.
func (c *checker) enumValues() {
	for _, e := range c.m.Enums {
		seen := map[string]string{}
		for _, v := range e.Values {
			if !valueChars.MatchString(v.Name) || !goIdent.MatchString(e.ConstName(v.Name)) {
				c.errorf(v.Pos, "Go 이름으로 바꿀 수 없는 enum 값: %s.%s", e.ID, v.Name)
				continue
			}
			name := e.ConstName(v.Name)
			if prev, ok := seen[name]; ok {
				c.errorf(v.Pos, "Go 상수 이름이 겹치는 enum 값: %s.%s (%s)", e.ID, v.Name, prev)
				continue
			}
			seen[name] = v.Name
		}
	}
}

// errorIDs는 에러의 Go 이름이 export되는 이름인지 본다. go.name이 없으면 ID가 Go 이름이다.
func (c *checker) errorIDs() {
	for _, e := range c.m.Errors {
		switch {
		case e.Go.Name != "" && !exportedIdent.MatchString(e.Go.Name):
			c.errorf(e.Pos, "Go 이름으로 쓸 수 없는 에러 이름: %s %s", e.ID, e.Go.Name)
		case e.Go.Name == "" && !exportedIdent.MatchString(e.ID):
			c.errorf(e.Pos, "Go 이름으로 쓸 수 없는 에러 ID: %s", e.ID)
		}
	}
}

// funcNames는 일반 함수의 go.name이 export되는 이름인지 본다.
func (c *checker) funcNames() {
	for _, f := range c.m.Funcs {
		if f.Go.Name != "" && !exportedIdent.MatchString(f.Go.Name) {
			c.errorf(f.Pos, "Go 이름으로 쓸 수 없는 함수 이름: %s %s", f.ID, f.Go.Name)
		}
	}
}

// fieldNames는 struct와 이벤트 data의 필드가 Go 필드 이름이 되고 겹치지 않는지 본다.
func (c *checker) fieldNames() {
	check := func(owner string, fields []*model.Field, event bool) {
		seen := map[string]string{}
		for _, f := range fields {
			name := model.GoName(f.Name)
			if !valueChars.MatchString(f.Name) || !goIdent.MatchString(name) {
				c.errorf(f.Pos, "Go 이름으로 바꿀 수 없는 필드: %s.%s", owner, f.Name)
				continue
			}
			if prev, ok := seen[name]; ok {
				c.errorf(f.Pos, "Go 필드 이름이 겹친다: %s.%s (%s)", owner, f.Name, prev)
				continue
			}
			seen[name] = f.Name
			if event && name == "Map" {
				c.errorf(f.Pos, "Map 메서드와 겹치는 필드: %s.%s", owner, f.Name)
			}
		}
	}
	for _, t := range c.m.Types {
		check(t.ID, t.Fields, false)
	}
	for _, e := range c.m.Events {
		check(e.ID, e.Data, true)
	}
}

func (c *checker) durations() {
	unit := map[string]int64{"ms": int64(time.Millisecond), "s": int64(time.Second), "m": int64(time.Minute), "h": int64(time.Hour)}
	for _, k := range c.m.Consts {
		if k.Type != "duration" {
			continue
		}
		if !durationValue.MatchString(k.Value) {
			c.errorf(k.ValuePos, "duration 형식이 아니다: 상수 %s %q", k.ID, k.Value)
			continue
		}
		digits := strings.TrimRight(k.Value, "smh")
		n, err := strconv.ParseInt(digits, 10, 64)
		if err != nil || n > math.MaxInt64/unit[k.Value[len(digits):]] {
			c.errorf(k.ValuePos, "duration이 time.Duration 범위를 넘는다: 상수 %s %q", k.ID, k.Value)
		}
	}
}

// packageNames는 같은 패키지(모듈·멤버)에 생성되는 최상위 이름이 겹치지 않는지 본다.
func (c *checker) packageNames() {
	m := c.m
	type owner struct {
		first map[string]string // 이름 → 처음 만든 노드
	}
	byOwner := map[string]*owner{}
	add := func(ownerID, name, source string, pos yaml.Pos) {
		o := byOwner[ownerID]
		if o == nil {
			o = &owner{first: map[string]string{}}
			byOwner[ownerID] = o
		}
		if prev, ok := o.first[name]; ok && prev != source {
			c.errorf(pos, "같은 패키지에서 생성 이름이 겹친다: %s 패키지의 %s (%s, %s)", ownerID, name, prev, source)
			return
		}
		o.first[name] = source
	}
	for _, e := range m.Enums {
		if !e.Gen {
			continue
		}
		src := "enum " + e.ID
		add(e.In.Name, e.GoType(), src, e.Pos)
		add(e.In.Name, e.GoType()+"Values", src, e.Pos)
		for _, v := range e.Values {
			add(e.In.Name, e.ConstName(v.Name), src, v.Pos)
		}
	}
	for _, k := range m.Consts {
		if k.Gen {
			add(k.In.Name, k.GoName(), "상수 "+k.ID, k.Pos)
		}
	}
	for _, e := range m.Errors {
		if e.Gen {
			add(e.In.Name, e.GoName(), "에러 "+e.ID, e.Pos)
		}
	}
	for _, t := range m.Types {
		if t.Gen {
			add(t.In.Name, t.GoName(), "타입 "+t.ID, t.Pos)
		}
	}
	for _, ct := range m.Contracts {
		if ct.Gen && ct.Kind == "call" {
			add(ct.In.Name, ct.GoName(), "계약 "+ct.ID, ct.Pos)
		}
	}
	for _, id := range c.unitIDs() {
		if m.Group(id) != nil {
			continue // 그룹의 needs는 멤버 패키지에 생긴다
		}
		for _, n := range c.needsOf(id) {
			if ct := m.Contract(n.Contract.Name); ct != nil && ct.Gen {
				add(id, ct.GoName(), "받는 계약 "+ct.ID, n.Contract.Pos)
			}
		}
	}
	for _, hk := range m.HandlerKinds {
		if hk.Gen {
			add(hk.In.Name, hk.GoName(), "핸들러 종류 "+hk.ID, hk.Pos)
		}
	}
	for _, e := range m.Events {
		ek := m.EventKind(e.Kind.Name)
		if ek == nil || !ek.Gen {
			continue
		}
		src := "이벤트 " + e.ID
		add(ek.In.Name, model.GoName(e.ID)+"Data", src, e.Pos)
		if ek.Encoding == "map" {
			add(ek.In.Name, model.GoName(e.ID)+"DataFrom", src, e.Pos)
		}
	}
	for _, f := range m.Flows {
		if f.Gen {
			src := "흐름 " + f.ID
			add(f.In.Name, model.GoName(f.ID)+"Steps", src, f.Pos)
			add(f.In.Name, "Run"+model.GoName(f.ID), src, f.Pos)
		}
	}
}
