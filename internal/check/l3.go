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
	for _, t := range m.Types {
		for _, f := range t.Fields {
			c.useType(t.In.Name, f.Type)
			c.useType(t.In.Name, f.RefType)
		}
	}
	for _, ct := range m.Contracts {
		for _, me := range ct.Methods {
			for _, p := range me.Params {
				c.useType(ct.In.Name, p.Type)
			}
			for _, r := range me.Returns {
				c.useType(ct.In.Name, r)
			}
		}
	}
	for _, hk := range m.HandlerKinds {
		for _, p := range hk.Params {
			c.useType(hk.In.Name, p.Type)
		}
		for _, r := range hk.Returns {
			c.useType(hk.In.Name, r)
		}
		c.useType(hk.In.Name, hk.Key)
	}
	// needs로 받은 계약의 시그니처 타입에도 받는 쪽이 의존해야 한다.
	for _, id := range c.unitIDs() {
		for _, n := range c.ownNeeds(id) {
			ct := m.Contract(n.Contract.Name)
			if ct == nil {
				continue
			}
			for _, me := range ct.Methods {
				if len(n.Uses) > 0 && !has(n.Uses, me.Name) {
					continue
				}
				for _, p := range me.Params {
					c.useNeededType(id, p.Type, n, ct.ID)
				}
				for _, r := range me.Returns {
					c.useNeededType(id, r, n, ct.ID)
				}
			}
		}
	}
}

// useNeededType은 받은 계약의 시그니처 타입에 받는 쪽이 의존하는지 본다. 위치는 받는 쪽의 needs 항목이다.
func (c *checker) useNeededType(owner string, r model.Ref, n model.Need, contract string) {
	if typeOwner, base := c.typeOwner(r); typeOwner != "" && !c.m.CanDepend(owner, typeOwner) {
		c.errorf(n.Contract.Pos, "받는 계약의 타입이 속한 모듈에 의존하지 않는다: %s → %s.%s (계약 %s)", owner, typeOwner, base, contract)
	}
}

// typeOwner는 r이 설계 enum·struct거나 "모듈.이름" 외부 타입이면 그 모듈과 이름을 준다.
func (c *checker) typeOwner(r model.Ref) (string, string) {
	if r.Name == "" {
		return "", ""
	}
	kind, base := c.m.ResolveType(r.Name)
	switch kind {
	case model.EnumType:
		return c.m.Enum(base).In.Name, base
	case model.StructType:
		return c.m.Type(base).In.Name, base
	case model.ExternalType:
		if pkg, name, ok := strings.Cut(base, "."); ok {
			if _, isOwner := c.m.Owner(pkg); isOwner {
				return pkg, name
			}
		}
	}
	return "", base
}

// useType은 owner가 r이 가리키는 enum·타입의 모듈에 의존하는지 본다.
func (c *checker) useType(owner string, r model.Ref) {
	if owner == "" {
		return
	}
	if typeOwner, base := c.typeOwner(r); typeOwner != "" && !c.m.CanDepend(owner, typeOwner) {
		c.errorf(r.Pos, "의존하지 않는 모듈의 타입을 쓴다: %s → %s.%s", owner, typeOwner, base)
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
