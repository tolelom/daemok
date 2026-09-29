package check

import (
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

func (c *checker) l3() {
	c.typeDeps()
	c.errorHTTP()
	c.contractErrors()
}

// typeDeps는 노드가 쓰는 enum·타입의 모듈에 그 노드의 모듈이 의존하는지 본다.
func (c *checker) typeDeps() {
	m := c.m
	use := func(owner string, r model.Ref) {
		if r.Name == "" || owner == "" {
			return
		}
		kind, base := m.ResolveType(r.Name)
		var typeOwner string
		switch kind {
		case model.EnumType:
			typeOwner = m.Enum(base).In.Name
		case model.StructType:
			typeOwner = m.Type(base).In.Name
		default:
			return
		}
		if typeOwner != "" && !m.CanDepend(owner, typeOwner) {
			c.errorf(r.Pos, "의존하지 않는 모듈의 타입을 쓴다: %s → %s.%s", owner, typeOwner, base)
		}
	}
	for _, t := range m.Types {
		for _, f := range t.Fields {
			use(t.In.Name, f.Type)
			use(t.In.Name, f.RefType)
		}
	}
	for _, ct := range m.Contracts {
		for _, me := range ct.Methods {
			for _, p := range me.Params {
				use(ct.In.Name, p.Type)
			}
			for _, r := range me.Returns {
				use(ct.In.Name, r)
			}
		}
	}
	for _, hk := range m.HandlerKinds {
		for _, p := range hk.Params {
			use(hk.In.Name, p.Type)
		}
		for _, r := range hk.Returns {
			use(hk.In.Name, r)
		}
		use(hk.In.Name, hk.Key)
	}
}

var httpBySuffix = []struct {
	suffix string
	code   int
}{
	{"NotFound", 404},
	{"Unauthorized", 401},
	{"Forbidden", 403},
	{"AlreadyExists", 409},
	{"Conflict", 409},
}

// errorHTTP는 에러 이름의 접미사가 뜻하는 HTTP 상태와 http 값이 다르면 경고한다.
func (c *checker) errorHTTP() {
	for _, e := range c.m.Errors {
		if e.HTTP == 0 {
			continue
		}
		for _, s := range httpBySuffix {
			if strings.HasSuffix(e.ID, s.suffix) && e.HTTP != s.code {
				c.warnf(e.Pos, "이름이 뜻하는 상태와 http가 다르다: %s (이름 %d, http %d)", e.ID, s.code, e.HTTP)
			}
		}
	}
}

func (c *checker) contractErrors() {
	for _, ct := range c.m.Contracts {
		for _, me := range ct.Methods {
			if !me.HasErrors {
				c.warnf(me.Pos, "계약 %s의 메서드 %s에 errors가 없다", ct.ID, me.Name)
			}
		}
	}
}
