package model

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

var topKeys = []string{
	"version", "go_module", "ts_out",
	"requirements", "system", "executables", "tiers", "modules", "groups",
	"enums", "consts", "errors", "types", "contracts", "handler_kinds", "funcs",
}

// Load는 dir의 *.yaml을 이름순으로 읽어 한 모델로 합친다.
// 오류가 있어도 읽은 만큼의 모델을 돌려준다.
func Load(dir string) (*Model, Diagnostics) {
	var diags Diagnostics
	paths, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if len(paths) == 0 {
		diags.Errorf(yaml.Pos{File: dir}, "설계 파일(*.yaml)이 없다")
		return nil, diags
	}
	sort.Strings(paths)
	mg := &merger{
		m:    &Model{},
		d:    &decoder{diags: &diags},
		ids:  map[string]map[string]yaml.Pos{},
		once: map[string]yaml.Pos{},
	}
	for _, p := range paths {
		src, err := os.ReadFile(p)
		if err != nil {
			diags.Errorf(yaml.Pos{File: p}, "읽을 수 없다: %v", err)
			continue
		}
		root, perr := yaml.Parse(p, src)
		if perr != nil {
			e := perr.(*yaml.Error)
			diags.Add(e.Pos, Error, "%s", e.Msg)
			continue
		}
		mg.file(root)
	}
	mg.finish()
	return mg.m, diags
}

type merger struct {
	m    *Model
	d    *decoder
	ids  map[string]map[string]yaml.Pos // 종류 → ID → 처음 위치
	once map[string]yaml.Pos            // 한 번만 나올 수 있는 최상위 키
}

func (mg *merger) file(root *yaml.Node) {
	d := mg.d
	if !d.mapping(root, "설계 파일") {
		return
	}
	d.allow(root, topKeys...)
	mg.scalarOnce(root, "version", &mg.m.Version)
	if v := root.Get("version"); v != nil && v.Kind == yaml.Scalar && v.Value != "0.3" {
		d.diags.Errorf(v.Pos, "지원하는 형식 버전은 0.3이다")
	}
	mg.scalarOnce(root, "go_module", &mg.m.GoModule)
	mg.scalarOnce(root, "ts_out", &mg.m.TSOut)
	mg.requirements(root.Get("requirements"))
	mg.system(root.Get("system"))
	mg.executables(root.Get("executables"))
	mg.tiers(root.Get("tiers"))
	mg.modules(root.Get("modules"))
	mg.groups(root.Get("groups"))
	mg.enums(root.Get("enums"))
	mg.consts(root.Get("consts"))
	mg.errors(root.Get("errors"))
	mg.types(root.Get("types"))
	mg.contracts(root.Get("contracts"))
	mg.handlerKinds(root.Get("handler_kinds"))
	mg.funcs(root.Get("funcs"))
}

func (mg *merger) scalarOnce(root *yaml.Node, key string, dst *string) {
	v := mg.d.scalar(root, key)
	if v == nil {
		return
	}
	if prev, ok := mg.once[key]; ok && *dst != v.Value {
		mg.d.diags.Errorf(v.Pos, "%q가 다른 파일(%s)과 다르다", key, prev)
		return
	}
	mg.once[key] = v.Pos
	*dst = v.Value
}

// unique는 종류 안의 ID 중복을 잡는다. 처음 보는 ID면 true.
func (mg *merger) unique(kind, id string, pos yaml.Pos) bool {
	if mg.ids[kind] == nil {
		mg.ids[kind] = map[string]yaml.Pos{}
	}
	if prev, dup := mg.ids[kind][id]; dup {
		mg.d.diags.Errorf(pos, "중복 ID %s %q (처음: %s)", kind, id, prev)
		return false
	}
	mg.ids[kind][id] = pos
	return true
}

// finish는 한 참조 필드가 받는 종류끼리 ID가 겹치는지 본다.
func (mg *merger) finish() {
	mg.collide("모듈", "그룹")
	mg.collide("모듈", "멤버")
	mg.collide("그룹", "멤버")
	mg.collide("enum", "타입")
}

func (mg *merger) collide(a, b string) {
	for id, posB := range mg.ids[b] {
		if posA, ok := mg.ids[a][id]; ok {
			mg.d.diags.Errorf(posB, "%s %q가 %s(%s)와 같은 이름이라 참조가 모호하다", b, id, a, posA)
		}
	}
}
