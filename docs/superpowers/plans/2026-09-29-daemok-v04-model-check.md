# daemok v0.4 model·check 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 설계 파일 형식 v0.4(관계 종류, 이벤트, 흐름)를 읽고 설계 검사한다.

**Architecture:** `internal/model`에 새 노드 타입과 디코더를 더하고, `internal/check`에 참조·관계·이벤트·흐름 검사를 더한다. 관계는 따로 저장하지 않고 `needs`, `contracts.kind`, `events`에서 계산한다.

**Tech Stack:** Go 표준 라이브러리

**Spec:** `docs/superpowers/specs/2026-09-29-daemok-format-v04-design.md`

## Global Constraints

- 외부 의존성 없음. Go 표준 라이브러리만 쓴다.
- 진단 메시지는 "설명: 대상" 형식이다. ID 바로 뒤에 조사를 붙이지 않는다.
- `version`은 `0.4`만 받는다.
- 형식에 없는 키는 위치와 함께 오류다.

## Review Focus

- `needs`만 있는 관계가 순환을 만들 때: 순환 오류가 나야 한다. Task 6에서 테스트한다.
- 그룹 멤버가 발행자일 때: 그룹의 `depends_on`으로 event_kind 모듈에 의존하면 통과해야 한다. Task 7에서 테스트한다.
- 흐름의 commit 단계 자체에 `undo`가 있을 때: 오류다. Task 8에서 테스트한다.
- `needs` 항목이 null일 때: 조용히 넘기지 않고 오류다. Task 2에서 테스트한다.
- 이벤트 ID가 key enum에 없을 때: 오류다. Task 7에서 테스트한다.

---

### Task 1: version 0.4

**Files:**
- Modify: `internal/model/load.go`
- Modify: 모든 테스트와 `testdata`의 `version: 0.3`

- [ ] **Step 1: 테스트 기대값을 0.4로 바꾼다**

`command grep -rln "0\.3" internal cmd`로 찾은 테스트·testdata의 `version: 0.3`과 `"0.3"` 기대값을 `0.4`로 바꾼다. `TestVersionMismatch`의 메시지 기대값은 `지원하는 형식 버전은 0.4다`로 바꾼다.

- [ ] **Step 2: 실패 확인**

Run: `go test ./...`
Expected: 버전 불일치 오류로 FAIL

- [ ] **Step 3: 구현**

`load.go`:

```go
	if v := root.Get("version"); v != nil && v.Kind == yaml.Scalar && v.Value != "0.4" {
		d.diags.Errorf(v.Pos, "지원하는 형식 버전은 0.4다")
	}
```

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 형식 버전 0.4`

### Task 2: 관계 필드 읽기

**Files:**
- Modify: `internal/model/model.go`, `decode.go`, `decode_layers.go`, `decode_l3.go`, `index.go`
- Test: `internal/model/load_test.go`

**Interfaces:**
- Produces: `model.Need{Contract Ref; Uses []Ref; Why string; Pos yaml.Pos}`, `Module.Role string`, `Module.Needs`, `Group.Needs`, `Member.Needs []Need`, `Contract.Kind string`(기본 `call`), `model.GoName(id string) string`

- [ ] **Step 1: 실패 테스트**

```go
func TestDecodeRelations(t *testing.T) {
	m, ds := loadOne(t, `
tiers: [base]
modules:
  post:  { tier: base }
  cache: { tier: base, role: infra }
  feed:
    tier: base
    needs:
      - post_reader
      - { contract: post_cache, uses: [get], why: "캐시" }
groups:
  g:
    tier: base
    kind: k
    needs: [post_reader]
    members:
      a: { needs: [post_cache] }
contracts:
  post_reader: { in: post, kind: query, methods: { list: {} } }
  post_cache:  { in: cache, kind: port, methods: { get: {} } }
  plain:       { in: post, methods: { x: {} } }
`)
	wantClean(t, ds)
	if m.Module("cache").Role != "infra" || m.Module("post").Role != "" {
		t.Fatalf("roles")
	}
	n := m.Module("feed").Needs
	if len(n) != 2 || n[0].Contract.Name != "post_reader" || n[1].Uses[0].Name != "get" || n[1].Why != "캐시" {
		t.Fatalf("needs %+v", n)
	}
	if m.Group("g").Needs[0].Contract.Name != "post_reader" || m.Member("a").Needs[0].Contract.Name != "post_cache" {
		t.Fatalf("group needs")
	}
	if m.Contract("post_reader").Kind != "query" || m.Contract("plain").Kind != "call" {
		t.Fatalf("kinds")
	}
}

func TestDecodeRelationErrors(t *testing.T) {
	_, ds := loadOne(t, `
tiers: [base]
modules:
  a: { tier: base, role: service, needs: [~, [x], { uses: [m] }] }
contracts:
  c: { in: a, kind: rpc, methods: { m: {} } }
`)
	wantDiag(t, ds, Error, `허용하지 않는 값: role "service"`)
	wantDiag(t, ds, Error, `허용하지 않는 값: kind "rpc"`)
	wantDiag(t, ds, Error, "계약 ID나 맵이어야 한다: needs 항목")
	wantDiag(t, ds, Error, `필요한 키가 없다: "contract"`)
	if n := ds.Count(Error); n != 5 {
		t.Fatalf("want 5 errors, got %d:\n%s", n, join(ds))
	}
}

func TestGoName(t *testing.T) {
	for in, want := range map[string]string{"asset_id": "AssetID", "list_published": "ListPublished", "id": "ID", "a__b": "AB"} {
		if got := GoName(in); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}
```

`needs: [~, [x], { uses: [m] }]`에서 `~`와 `[x]`는 각각 "계약 ID나 맵이어야 한다" 오류이므로 오류는 role, kind, needs 항목 둘, contract 부재로 다섯 개다.

- [ ] **Step 2: 실패 확인** `go test ./internal/model/ -run 'Relation|GoName'` FAIL(컴파일 오류)

- [ ] **Step 3: 구현**

`model.go`:

```go
// Need는 받는 쪽이 적은 계약 하나다.
type Need struct {
	Contract Ref
	Uses     []Ref
	Why      string
	Pos      yaml.Pos
}
```

`Module`에 `Role string`, `Needs []Need`, `Group`과 `Member`에 `Needs []Need`, `Contract`에 `Kind string`을 더한다.

`decode.go`:

```go
// oneOf는 key 값이 choices 중 하나인지 본다. 키가 없으면 def.
func (d *decoder) oneOf(n *yaml.Node, key, def string, choices ...string) string {
	v := d.scalar(n, key)
	if v == nil || v.IsNull() {
		return def
	}
	if !slices.Contains(choices, v.Value) {
		d.diags.Errorf(v.Pos, "허용하지 않는 값: %s %q (가능한 값: %s)", key, v.Value, strings.Join(choices, ", "))
		return def
	}
	return v.Value
}
```

`decode_layers.go`에 `needs`를 더하고 모듈·그룹·멤버의 `allow`에 `needs`(모듈은 `role`도)를 넣는다.

```go
func (mg *merger) needs(n *yaml.Node) []Need {
	if n == nil || n.IsNull() {
		return nil
	}
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "목록이어야 한다: %q", "needs")
		return nil
	}
	var out []Need
	for _, it := range n.Items {
		switch {
		case it.Kind == yaml.Scalar && !it.IsNull():
			out = append(out, Need{Contract: Ref{Name: it.Value, Pos: it.Pos}, Pos: it.Pos})
		case it.Kind == yaml.Map:
			d.allow(it, "contract", "uses", "why")
			if !d.need(it.Pos, it, "contract") {
				continue
			}
			out = append(out, Need{Contract: d.ref(it, "contract"), Uses: d.refs(it, "uses"), Why: d.str(it, "why"), Pos: it.Pos})
		default:
			d.diags.Errorf(it.Pos, "계약 ID나 맵이어야 한다: needs 항목")
		}
	}
	return out
}
```

모듈: `Role: d.oneOf(v, "role", "", "data", "infra", "adapter"), Needs: mg.needs(v.Get("needs"))`. 그룹·멤버: `Needs: mg.needs(...)`. 계약: `allow`에 `kind`, `Kind: d.oneOf(v, "kind", "call", "call", "query", "port", "external")`.

`index.go`:

```go
// GoName은 snake_case ID를 CamelCase로 바꾼다. 조각 id는 ID가 된다.
func GoName(id string) string {
	var b strings.Builder
	for _, part := range strings.Split(id, "_") {
		switch part {
		case "":
		case "id":
			b.WriteString("ID")
		default:
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}
```

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 관계 필드(role, needs, contracts.kind) 읽기`

### Task 3: 이벤트 읽기

**Files:**
- Modify: `internal/model/model.go`, `load.go`, `decode_l3.go`, `index.go`
- Test: `internal/model/load_test.go`

**Interfaces:**
- Produces: `model.EventKind{Owned; Key Ref; Encoding string; Subscribers []Ref}`, `model.Event{Meta; Kind Ref; Publishers, Subscribers []Ref; Data []*Field}`, `Model.EventKinds`, `Model.Events`, `(*Model).EventKind(id)`, `(*Model).Event(id)`

- [ ] **Step 1: 실패 테스트**

```go
func TestDecodeEvents(t *testing.T) {
	m, ds := loadOne(t, `
event_kinds:
  chain: { in: events, key: event_type, subscribers: [rpc] }
  web:   { in: events, key: web_type, encoding: json }
events:
  asset_minted:
    kind: chain
    publishers: [asset, reward]
    subscribers: [indexer]
    data: { asset_id: string, players: { type: string, many: true } }
  session_close: { kind: chain, publishers: [session] }
`)
	wantClean(t, ds)
	ek := m.EventKind("chain")
	if ek.In.Name != "events" || ek.Key.Name != "event_type" || ek.Encoding != "map" || ek.Subscribers[0].Name != "rpc" || m.EventKind("web").Encoding != "json" {
		t.Fatalf("kind %+v", ek)
	}
	e := m.Event("asset_minted")
	if e.Kind.Name != "chain" || len(e.Publishers) != 2 || e.Subscribers[0].Name != "indexer" || len(e.Data) != 2 || !e.Data[1].Many {
		t.Fatalf("event %+v", e)
	}
	if m.Event("session_close").Data != nil {
		t.Fatal("data")
	}
}

func TestDecodeEventErrors(t *testing.T) {
	_, ds := loadOne(t, `
event_kinds:
  k: { in: events, encoding: xml }
events:
  e: { data: {} }
`)
	wantDiag(t, ds, Error, `필요한 키가 없다: "key"`)
	wantDiag(t, ds, Error, `허용하지 않는 값: encoding "xml"`)
	wantDiag(t, ds, Error, `필요한 키가 없다: "kind"`)
	wantDiag(t, ds, Error, `필요한 키가 없다: "publishers"`)
}
```

- [ ] **Step 2: 실패 확인** FAIL(컴파일 오류)

- [ ] **Step 3: 구현**

`model.go`:

```go
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
```

`Model`에 `EventKinds []*EventKind`, `Events []*Event`. `load.go`의 `topKeys`에 `"event_kinds", "events"`, `file()`에서 `mg.eventKinds(root.Get("event_kinds"))`, `mg.events(root.Get("events"))`를 `handlerKinds` 뒤에 부른다.

`decode_l3.go`:

```go
func (mg *merger) eventKinds(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "event_kinds", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "key", "encoding", "subscribers")
		if !mg.unique("이벤트 종류", id, pos) {
			return
		}
		ek := &EventKind{Owned: d.owned(id, pos, v), Encoding: d.oneOf(v, "encoding", "map", "map", "json"), Subscribers: d.refs(v, "subscribers")}
		if d.need(pos, v, "key") {
			ek.Key = d.ref(v, "key")
		}
		mg.m.EventKinds = append(mg.m.EventKinds, ek)
	})
}

func (mg *merger) events(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "events", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "kind", "publishers", "subscribers", "data")
		if !mg.unique("이벤트", id, pos) {
			return
		}
		e := &Event{Meta: d.meta(id, pos, v), Subscribers: d.refs(v, "subscribers")}
		if d.need(pos, v, "kind") {
			e.Kind = d.ref(v, "kind")
		}
		if d.need(pos, v, "publishers") {
			e.Publishers = d.refs(v, "publishers")
		}
		if dv := v.Get("data"); dv != nil && !dv.IsNull() {
			e.Data = mg.fields(dv)
		}
		mg.m.Events = append(mg.m.Events, e)
	})
}
```

`index.go`에 `EventKind(id)`, `Event(id)`를 다른 조회 함수와 같은 모양으로 더한다.

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 이벤트 종류와 이벤트 읽기`

### Task 4: 흐름 읽기

**Files:**
- Modify: `internal/model/model.go`, `load.go`, `decode_l4.go`, `index.go`
- Test: `internal/model/load_test.go`

**Interfaces:**
- Produces: `model.Step{ID string; Pos yaml.Pos; Summary, Undo string; UndoPos yaml.Pos; Commit, Fatal bool; Calls, Emits []Ref}`, `model.Flow{Owned; On Ref; Steps []*Step}`, `Model.Flows`, `(*Model).Flow(id)`

- [ ] **Step 1: 실패 테스트**

```go
func TestDecodeFlows(t *testing.T) {
	m, ds := loadOne(t, `
flows:
  produce_block:
    in: consensus
    on: tick
    steps:
      - { id: lock, undo: unlock }
      - { id: add_block, calls: [chain.add] }
      - { id: commit_state, commit: true, fatal: true, summary: "상태 반영" }
      - { id: emit, emits: [block_commit] }
`)
	wantClean(t, ds)
	f := m.Flow("produce_block")
	if f.In.Name != "consensus" || f.On.Name != "tick" || len(f.Steps) != 4 {
		t.Fatalf("flow %+v", f)
	}
	s := f.Steps
	if s[0].ID != "lock" || s[0].Undo != "unlock" || s[0].UndoPos.Line == 0 || s[1].Calls[0].Name != "chain.add" ||
		!s[2].Commit || !s[2].Fatal || s[2].Summary != "상태 반영" || s[3].Emits[0].Name != "block_commit" || s[3].Commit {
		t.Fatalf("steps %+v %+v %+v %+v", s[0], s[1], s[2], s[3])
	}
}

func TestDecodeFlowErrors(t *testing.T) {
	_, ds := loadOne(t, `
flows:
  a: { in: x }
  b: { in: x, steps: { lock: {} } }
  c: { in: x, steps: [ lock, { undo: u }, { id: s, retry: 3 } ] }
`)
	wantDiag(t, ds, Error, `필요한 키가 없다: "steps"`)
	wantDiag(t, ds, Error, `목록이어야 한다: "steps"`)
	wantDiag(t, ds, Error, "맵이어야 한다: 단계")
	wantDiag(t, ds, Error, `필요한 키가 없다: "id"`)
	wantDiag(t, ds, Error, `알 수 없는 키 "retry"`)
}
```

- [ ] **Step 2: 실패 확인** FAIL(컴파일 오류)

- [ ] **Step 3: 구현**

`model.go`:

```go
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
```

`Model`에 `Flows []*Flow`. `topKeys`에 `"flows"`, `file()`에서 `mg.flows(root.Get("flows"))`를 `funcs` 뒤에 부른다.

`decode_l4.go`:

```go
func (mg *merger) flows(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "flows", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "on", "steps")
		if !mg.unique("흐름", id, pos) {
			return
		}
		f := &Flow{Owned: d.owned(id, pos, v), On: d.ref(v, "on")}
		if d.need(pos, v, "steps") {
			f.Steps = mg.steps(v.Get("steps"))
		}
		mg.m.Flows = append(mg.m.Flows, f)
	})
}

func (mg *merger) steps(n *yaml.Node) []*Step {
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "목록이어야 한다: %q", "steps")
		return nil
	}
	var out []*Step
	for _, it := range n.Items {
		if !d.mapping(it, "단계") {
			continue
		}
		d.allow(it, "id", "summary", "undo", "commit", "fatal", "calls", "emits")
		if !d.need(it.Pos, it, "id") {
			continue
		}
		idv := d.scalar(it, "id")
		if idv == nil {
			continue
		}
		s := &Step{
			ID: idv.Value, Pos: idv.Pos, Summary: d.str(it, "summary"),
			Commit: d.boolv(it, "commit", false), Fatal: d.boolv(it, "fatal", false),
			Calls: d.refs(it, "calls"), Emits: d.refs(it, "emits"),
		}
		if u := d.scalar(it, "undo"); u != nil && !u.IsNull() {
			s.Undo, s.UndoPos = u.Value, u.Pos
		}
		out = append(out, s)
	}
	return out
}
```

`index.go`에 `Flow(id)`.

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 흐름 읽기`

### Task 5: 새 참조 검사

**Files:**
- Modify: `internal/check/check.go`(eachMeta), `internal/check/refs.go`
- Test: `internal/check/check_test.go`

- [ ] **Step 1: 실패 테스트**

`base`에 아래를 더해 `TestBaseIsClean`이 통과하는 상태를 유지한다(Task 5~8의 테스트가 이것을 바꿔 위반을 만든다).

```yaml
enums:
  event_type: { in: core, values: [minted] }
event_kinds:
  chain_event: { in: core, key: event_type, subscribers: [rpc_mod] }
events:
  minted: { kind: chain_event, publishers: [asset], subscribers: [vm], data: { to: address } }
flows:
  run_block:
    in: vm
    steps:
      - { id: lock, undo: unlock }
      - { id: load, calls: [store.get] }
      - { id: commit, commit: true, fatal: true }
      - { id: unlock }
```

`base`의 `enums:`, `contracts:` 절에 맞춰 끼워 넣는다. `vm`은 `store`(in: core)를 부르므로 `vm`에 `needs: [store]`를 더한다.

```go
func TestRefErrorsV04(t *testing.T) {
	cases := []struct{ old, new, msg string }{
		{"needs: [store]", "needs: [shop]", `없는 계약 "shop"`},
		{"needs: [store]", "needs: [{ contract: store, uses: [put] }]", `계약 store에 없는 메서드 "put"`},
		{"key: event_type", "key: evt", `없는 enum "evt"`},
		{"subscribers: [rpc_mod]", "subscribers: [nobody]", `없는 모듈 "nobody"`},
		{"kind: chain_event", "kind: chain", `없는 이벤트 종류 "chain"`},
		{"publishers: [asset]", "publishers: [ghost]", `없는 모듈 "ghost"`},
		{"data: { to: address }", "data: { to: adress }", `알 수 없는 타입 "adress"`},
		{"calls: [store.get]", "calls: [store.put]", `계약 store에 없는 메서드 "put"`},
		{"calls: [store.get]", "calls: [storeget]", "계약.메서드 형식"},
		{"in: vm\n    steps", "in: nowhere\n    steps", `in이 가리키는 모듈이나 멤버가 없다: "nowhere"`},
		{"{ id: unlock }", "{ id: unlock, emits: [burned] }", `없는 이벤트 "burned"`},
	}
	for _, c := range cases {
		ds := run(t, replace(t, c.old, c.new))
		want(t, ds, model.Error, c.msg)
	}
	ds := run(t, replace(t, "in: vm\n    steps", "in: vm\n    on: burned\n    steps"))
	want(t, ds, model.Error, `없는 이벤트 "burned"`)
}
```

- [ ] **Step 2: 실패 확인** FAIL

- [ ] **Step 3: 구현**

`check.go`의 `eachMeta` 끝에 `EventKinds`, `Events`, `Flows`의 `Meta`를 더한다.

`refs.go`:

```go
	for _, mod := range m.Modules {
		// 기존 검사 뒤에
		c.needRefs(mod.Needs)
	}
	for _, g := range m.Groups {
		c.needRefs(g.Needs)
		for _, mem := range g.Members {
			c.needRefs(mem.Needs)
		}
	}
	for _, t := range m.Types {
		c.in(t.In)
		c.fieldRefs(t.Fields)
	}
	for _, ek := range m.EventKinds {
		c.in(ek.In)
		if ek.Key.Name != "" && m.Enum(ek.Key.Name) == nil {
			c.errorf(ek.Key.Pos, "없는 enum %q", ek.Key.Name)
		}
		for _, r := range ek.Subscribers {
			c.dep(r)
		}
	}
	for _, e := range m.Events {
		if e.Kind.Name != "" && m.EventKind(e.Kind.Name) == nil {
			c.errorf(e.Kind.Pos, "없는 이벤트 종류 %q", e.Kind.Name)
		}
		for _, r := range e.Publishers {
			c.dep(r)
		}
		for _, r := range e.Subscribers {
			c.dep(r)
		}
		c.fieldRefs(e.Data)
	}
	for _, f := range m.Flows {
		c.in(f.In)
		if f.On.Name != "" {
			c.eventRef(f.On)
		}
		for _, s := range f.Steps {
			for _, r := range s.Calls {
				c.methodRef(r, "calls")
			}
			for _, r := range s.Emits {
				c.eventRef(r)
			}
		}
	}
```

타입 필드 검사 본문은 `fieldRefs(fields []*model.Field)`로 옮긴다. `ofRef`는 `methodRef(r model.Ref, field string)`로 이름을 바꾸고 형식 오류 메시지를 `"계약.메서드 형식이어야 한다: %s %q", field, r.Name`으로 바꾼다. 함수의 `of`는 `c.methodRef(f.Of, "of")`로 부른다.

```go
func (c *checker) needRefs(ns []model.Need) {
	for _, n := range ns {
		ct := c.m.Contract(n.Contract.Name)
		if ct == nil {
			c.errorf(n.Contract.Pos, "없는 계약 %q", n.Contract.Name)
			continue
		}
		for _, u := range n.Uses {
			if ct.Method(u.Name) == nil {
				c.errorf(u.Pos, "계약 %s에 없는 메서드 %q", ct.ID, u.Name)
			}
		}
	}
}

func (c *checker) eventRef(r model.Ref) {
	if c.m.Event(r.Name) == nil {
		c.errorf(r.Pos, "없는 이벤트 %q", r.Name)
	}
}
```

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 관계·이벤트·흐름 참조 검사`

### Task 6: 관계 검사

**Files:**
- Modify: `internal/check/check.go`(depsOf, needsOf), `l2.go`, `l3.go`
- Test: `internal/check/check_test.go`

**Interfaces:**
- Produces: `(*checker).needsOf(id string) []model.Need`, `(*checker).useType(owner string, r model.Ref)`

- [ ] **Step 1: 실패 테스트**

```go
func TestNeedsTierAndCycle(t *testing.T) {
	// crypto(foundation)가 core(domain)의 계약을 받으면 상위 tier 의존
	ds := run(t, replace(t, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, needs: [store] }"))
	want(t, ds, model.Error, "상위 tier 의존: crypto(foundation) → core(domain)")
	// core가 vm의 계약을 받고 vm이 core에 의존하면 순환
	src := replace(t, "core:    { tier: domain, depends_on: [crypto]", "core:    { tier: domain, depends_on: [crypto], needs: [runner]")
	src = strings.Replace(src, "contracts:\n", "contracts:\n  runner: { in: vm, methods: { run: {} } }\n", 1)
	ds = run(t, src)
	want(t, ds, model.Error, "의존 순환: core → vm → core")
}

func TestContractRole(t *testing.T) {
	src := strings.Replace(base, "contracts:\n", "contracts:\n  cache_port: { in: crypto, kind: port, methods: { get: {} } }\n  mailer: { in: crypto, kind: external, methods: { send: {} } }\n", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "port 계약의 모듈은 role이 infra여야 한다: 계약 cache_port, 모듈 crypto")
	want(t, ds, model.Error, "external 계약의 모듈은 role이 adapter여야 한다: 계약 mailer, 모듈 crypto")
	ds = run(t, strings.Replace(src, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, role: infra }", 1))
	wantNone := func(msg string) {
		for _, d := range ds {
			if strings.Contains(d.Msg, msg) {
				t.Fatalf("unexpected %s", d)
			}
		}
	}
	wantNone("cache_port")
}

func TestNeedsTypeDependency(t *testing.T) {
	// rpc_mod가 store를 받는데 store의 반환 타입 payload(core)에 rpc_mod는 core로 의존하므로 통과
	ds := run(t, replace(t, "rpc_mod: { tier: service, depends_on: [core, vm]", "rpc_mod: { tier: service, depends_on: [core, vm], needs: [store]"))
	wantNone(t, ds, model.Error)
	// crypto가 store를 받으면 payload(core)에 의존하지 않으므로 오류(상위 tier 오류와 함께)
	ds = run(t, replace(t, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, needs: [store] }"))
	want(t, ds, model.Error, "의존하지 않는 모듈의 타입을 쓴다: crypto → core.payload")
}
```

- [ ] **Step 2: 실패 확인** FAIL

- [ ] **Step 3: 구현**

`check.go`:

```go
// needsOf는 id가 받는 계약이다. 멤버는 그룹의 needs와 자기 needs를 받는다.
func (c *checker) needsOf(id string) []model.Need {
	m := c.m
	switch {
	case m.Module(id) != nil:
		return m.Module(id).Needs
	case m.Group(id) != nil:
		return m.Group(id).Needs
	case m.Member(id) != nil:
		mem := m.Member(id)
		return append(slices.Clone(mem.Group.Needs), mem.Needs...)
	}
	return nil
}

// needTargets는 needs가 가리키는 계약의 모듈을 참조 위치와 함께 준다.
func (c *checker) needTargets(ns []model.Need) []model.Ref {
	var out []model.Ref
	for _, n := range ns {
		if ct := c.m.Contract(n.Contract.Name); ct != nil && ct.In.Name != "" {
			out = append(out, model.Ref{Name: ct.In.Name, Pos: n.Contract.Pos})
		}
	}
	return out
}
```

`depsOf` 끝에서 `for _, r := range c.needTargets(c.needsOf(id)) { out = append(out, r.Name) }`.

`l2.go`의 `tierDirection`: 모듈은 `check(mod.ID, mod.Tier.Name, append(slices.Clone(mod.DependsOn), c.needTargets(mod.Needs)...))`, 그룹은 `g.Needs`, 멤버는 `mem.Needs`의 대상을 그룹 tier로 검사한다. `l2()`에 `c.contractRoles()`를 더한다.

```go
// contractRoles는 port·external 계약이 알맞은 role의 모듈에 있는지 본다.
func (c *checker) contractRoles() {
	want := map[string]string{"port": "infra", "external": "adapter"}
	for _, ct := range c.m.Contracts {
		role, ok := want[ct.Kind]
		if !ok {
			continue
		}
		mod := c.m.Module(ct.In.Name)
		if mod == nil || mod.Role != role {
			c.errorf(ct.Pos, "%s 계약의 모듈은 role이 %s여야 한다: 계약 %s, 모듈 %s", ct.Kind, role, ct.ID, ct.In.Name)
		}
	}
}
```

`l3.go`: `typeDeps`의 `use` 클로저를 메서드 `useType(owner string, r model.Ref)`로 옮기고, 끝에 다음을 더한다.

```go
	for _, id := range c.unitIDs() {
		for _, n := range c.needsOf(id) {
			ct := m.Contract(n.Contract.Name)
			if ct == nil {
				continue
			}
			for _, me := range ct.Methods {
				if len(n.Uses) > 0 && !has(n.Uses, me.Name) {
					continue
				}
				for _, p := range me.Params {
					c.useType(id, p.Type)
				}
				for _, r := range me.Returns {
					c.useType(id, r)
				}
			}
		}
	}
```

```go
// unitIDs는 모듈, 그룹, 멤버 ID 전부다.
func (c *checker) unitIDs() []string {
	var ids []string
	for _, mod := range c.m.Modules {
		ids = append(ids, mod.ID)
	}
	for _, g := range c.m.Groups {
		ids = append(ids, g.ID)
		for _, mem := range g.Members {
			ids = append(ids, mem.ID)
		}
	}
	return ids
}
```

멤버는 그룹 needs를 이미 포함하므로 그룹 ID 검사에서 한 번, 멤버에서 한 번 나올 수 있다. 같은 위치 같은 메시지 중복을 피하려고 멤버는 자기 `Needs`만 본다: 루프를 `c.ownNeeds(id)`(모듈·그룹·멤버 각자의 `Needs` 필드)로 돈다.

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: needs 관계의 tier·순환·타입 의존과 계약 role 검사`

### Task 7: 이벤트 검사

**Files:**
- Create: `internal/check/events.go`
- Modify: `internal/check/check.go`(Run에서 `c.events()`)
- Test: `internal/check/check_test.go`

- [ ] **Step 1: 실패 테스트**

```go
func TestEvents(t *testing.T) {
	ds := run(t, replace(t, "values: [minted]", "values: [minted, burned]"))
	want(t, ds, model.Error, "이벤트가 없는 값: event_type.burned")

	ds = run(t, strings.Replace(base, "events:\n", "events:\n  ghost: { kind: chain_event, publishers: [asset] }\n", 1))
	want(t, ds, model.Error, `event_type에 없는 값 "ghost"`)

	// crypto는 core에 의존하지 않는다
	ds = run(t, replace(t, "publishers: [asset]", "publishers: [asset, crypto]"))
	want(t, ds, model.Error, "이벤트 종류의 모듈에 의존하지 않는다: crypto → core")
	ds = run(t, replace(t, "subscribers: [rpc_mod]", "subscribers: [rpc_mod, crypto]"))
	want(t, ds, model.Error, "이벤트 종류의 모듈에 의존하지 않는다: crypto → core")

	// 멤버 asset은 그룹 의존으로 core에 닿는다(base가 이미 통과)
	ds = run(t, replace(t, "publishers: [asset]", "publishers: []"))
	want(t, ds, model.Error, "발행자가 없는 이벤트: minted")
}
```

- [ ] **Step 2: 실패 확인** FAIL

- [ ] **Step 3: 구현** `events.go`:

```go
package check

// events는 key enum 값마다 이벤트가 있는지, 발행자·구독자가 이벤트 종류의 모듈에 의존하는지 본다.
func (c *checker) events() {
	m := c.m
	for _, ek := range m.EventKinds {
		en := m.Enum(ek.Key.Name)
		if en != nil {
			for _, v := range en.Values {
				if e := m.Event(v.Name); e == nil || e.Kind.Name != ek.ID {
					c.errorf(v.Pos, "이벤트가 없는 값: %s.%s", en.ID, v.Name)
				}
			}
		}
		for _, r := range ek.Subscribers {
			c.dependsOnKind(r, ek.In.Name)
		}
	}
	for _, e := range m.Events {
		ek := m.EventKind(e.Kind.Name)
		if ek == nil {
			continue
		}
		if en := m.Enum(ek.Key.Name); en != nil && !en.HasValue(e.ID) {
			c.errorf(e.Pos, "%s에 없는 값 %q", en.ID, e.ID)
		}
		if len(e.Publishers) == 0 {
			c.errorf(e.Pos, "발행자가 없는 이벤트: %s", e.ID)
		}
		for _, r := range e.Publishers {
			c.dependsOnKind(r, ek.In.Name)
		}
		for _, r := range e.Subscribers {
			c.dependsOnKind(r, ek.In.Name)
		}
		for _, f := range e.Data {
			c.useType(ek.In.Name, f.Type)
			c.useType(ek.In.Name, f.RefType)
		}
	}
}

func (c *checker) dependsOnKind(r model.Ref, kindModule string) {
	if kindModule == "" || !c.isUnit(r.Name) {
		return
	}
	if !c.m.CanDepend(r.Name, kindModule) {
		c.errorf(r.Pos, "이벤트 종류의 모듈에 의존하지 않는다: %s → %s", r.Name, kindModule)
	}
}

func (c *checker) isUnit(id string) bool {
	return c.m.Module(id) != nil || c.m.Group(id) != nil || c.m.Member(id) != nil
}
```

`publishers: []`는 `refs`가 빈 목록을 돌려주므로 `need`를 통과하고 여기서 걸린다.

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 이벤트 검사`

### Task 8: 흐름 검사

**Files:**
- Create: `internal/check/flows.go`
- Modify: `internal/check/check.go`(Run에서 `c.flows()`)
- Test: `internal/check/check_test.go`

- [ ] **Step 1: 실패 테스트**

```go
func TestFlows(t *testing.T) {
	cases := []struct{ old, new, msg string }{
		{"{ id: load, calls", "{ id: Load, calls", `단계 ID는 소문자 snake_case여야 한다: "Load"`},
		{"{ id: unlock }", "{ id: lock }", "흐름 안에서 중복된 단계: lock"},
		{"{ id: unlock }", "{ id: fatal }", "Fatal과 겹치는 단계 이름: fatal"},
		{"{ id: unlock }", "{ id: unlock, commit: true }", "commit 단계가 둘 이상이다: unlock"},
		{"{ id: unlock }", "{ id: unlock, undo: relock }", "커밋 뒤 단계에 undo가 있다: unlock"},
		{"{ id: commit, commit: true, fatal: true }", "{ id: commit, commit: true, fatal: true, undo: rollback }", "커밋 뒤 단계에 undo가 있다: commit"},
		{"{ id: lock, undo: unlock }", "{ id: lock, undo: Unlock }", `undo는 소문자 snake_case여야 한다: "Unlock"`},
	}
	for _, c := range cases {
		ds := run(t, replace(t, c.old, c.new))
		want(t, ds, model.Error, c.msg)
	}
	// vm이 받지 않는 계약
	ds := run(t, strings.Replace(replace(t, "needs: [store]", "needs: []"), "", "", 1))
	want(t, ds, model.Error, "흐름의 모듈이 받지 않는 계약: vm → store")
	// vm은 minted의 발행자가 아니다
	ds = run(t, replace(t, "{ id: unlock }", "{ id: unlock, emits: [minted] }"))
	want(t, ds, model.Error, "흐름의 모듈이 발행자가 아닌 이벤트: vm → minted")
	// vm은 minted의 구독자다
	ds = run(t, replace(t, "in: vm\n    steps", "in: vm\n    on: minted\n    steps"))
	wantNone(t, ds, model.Error)
	// crypto는 구독자가 아니다
	src := replace(t, "in: vm\n    steps", "in: crypto\n    on: minted\n    steps")
	ds = run(t, strings.Replace(src, "calls: [store.get]", "calls: []", 1))
	want(t, ds, model.Error, "흐름의 모듈이 구독자가 아닌 이벤트: crypto → minted")
}
```

- [ ] **Step 2: 실패 확인** FAIL

- [ ] **Step 3: 구현** `flows.go`:

```go
package check

import (
	"regexp"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

var snakeName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// flows는 단계 이름, commit 위치, 흐름의 모듈과 calls·emits·on의 관계를 본다.
func (c *checker) flows() {
	m := c.m
	for _, f := range m.Flows {
		owner := f.In.Name
		seen := map[string]bool{}
		committed := false
		for _, s := range f.Steps {
			if !snakeName.MatchString(s.ID) {
				c.errorf(s.Pos, "단계 ID는 소문자 snake_case여야 한다: %q", s.ID)
			}
			if seen[s.ID] {
				c.errorf(s.Pos, "흐름 안에서 중복된 단계: %s", s.ID)
			}
			seen[s.ID] = true
			if s.Undo != "" && !snakeName.MatchString(s.Undo) {
				c.errorf(s.UndoPos, "undo는 소문자 snake_case여야 한다: %q", s.Undo)
			}
			for _, name := range []string{s.ID, s.Undo} {
				if model.GoName(name) == "Fatal" {
					c.errorf(s.Pos, "Fatal과 겹치는 단계 이름: %s", name)
				}
			}
			if s.Commit {
				if committed {
					c.errorf(s.Pos, "commit 단계가 둘 이상이다: %s", s.ID)
				}
				committed = true
			}
			if committed && s.Undo != "" {
				c.errorf(s.UndoPos, "커밋 뒤 단계에 undo가 있다: %s", s.ID)
			}
			for _, r := range s.Calls {
				ctID, _, _ := strings.Cut(r.Name, ".")
				ct := m.Contract(ctID)
				if ct != nil && ct.In.Name != owner && !c.receives(owner, ct.ID) {
					c.errorf(r.Pos, "흐름의 모듈이 받지 않는 계약: %s → %s", owner, ct.ID)
				}
			}
			for _, r := range s.Emits {
				if e := m.Event(r.Name); e != nil && !has(e.Publishers, owner) {
					c.errorf(r.Pos, "흐름의 모듈이 발행자가 아닌 이벤트: %s → %s", owner, e.ID)
				}
			}
		}
		if e := m.Event(f.On.Name); e != nil {
			ek := m.EventKind(e.Kind.Name)
			if !has(e.Subscribers, owner) && (ek == nil || !has(ek.Subscribers, owner)) {
				c.errorf(f.On.Pos, "흐름의 모듈이 구독자가 아닌 이벤트: %s → %s", owner, e.ID)
			}
		}
	}
}

// receives는 id가 needs로 계약을 받는지다.
func (c *checker) receives(id, contract string) bool {
	for _, n := range c.needsOf(id) {
		if n.Contract.Name == contract {
			return true
		}
	}
	return false
}
```

`commit` 단계 자신에 `undo`가 있어도 `committed`가 먼저 true가 되므로 걸린다.

- [ ] **Step 4: 통과 확인** `go test ./...` PASS

- [ ] **Step 5: 커밋** `feat: 흐름 검사`

### Task 9: 샘플과 모델 문서

**Files:**
- Modify: `internal/model/testdata/tolchain/design/tolchain.yaml`
- Create: `internal/model/testdata/relations/design/relations.yaml`
- Modify: `internal/model/load_test.go`(TestSamplesLoadClean 목록), `internal/check/check_test.go`(TestSamples)
- Modify: `docs/design/01-model.md`

- [ ] **Step 1: 샘플**

tolchain 샘플에 더한다.

```yaml
enums:
  event_type:
    in: events
    go: { type: EventType, prefix: Event }
    values: [block_commit, asset_minted, session_open]

event_kinds:
  chain_event: { in: events, key: event_type, encoding: map, subscribers: [rpc] }

events:
  block_commit:
    kind: chain_event
    publishers: [consensus]
    data: { hash: string, txs: int, block: "*core.Block" }
  asset_minted:
    kind: chain_event
    publishers: [asset, reward]
    subscribers: [indexer]
    data: { asset_id: string, template_id: string, owner: string }
  session_open:
    kind: chain_event
    publishers: [session]
    subscribers: [indexer]
    data: { session_id: string, game_id: string, players: { type: string, many: true } }

flows:
  produce_block:
    in: consensus
    steps:
      - { id: check_proposer }
      - { id: select_txs }
      - { id: lock, undo: unlock }
      - { id: snapshot, undo: revert }
      - { id: buffer_events, undo: discard_events }
      - { id: execute_txs }
      - { id: restore_emitter }
      - { id: compute_root }
      - { id: sign }
      - { id: validate }
      - { id: add_block }
      - { id: commit_state, commit: true, fatal: true }
      - { id: unlock }
      - { id: flush_events }
      - { id: emit_block_commit, emits: [block_commit] }
      - { id: remove_from_mempool }
```

`relations.yaml`(query·port·external):

```yaml
version: 0.4
go_module: example.com/relations

executables:
  server: { composes: [feed, post, cache, mailer_adapter, model] }

tiers: [base, infra, domain, edge]

modules:
  model:  { tier: base, role: data }
  cache:  { tier: infra, role: infra }
  mailer_adapter: { tier: infra, role: adapter, path: mailer }
  post:   { tier: domain, depends_on: [model], needs: [post_cache] }
  feed:
    tier: edge
    depends_on: [model]
    needs:
      - { contract: post_reader, uses: [list_published], why: "피드용 공개 글 목록" }
      - mailer

types:
  post: { in: model, fields: { id: uint, title: string } }

contracts:
  post_reader:
    in: post
    kind: query
    methods:
      list_published: { params: { page: int }, returns: ["[]post"], errors: [] }
      count: { returns: [int], errors: [] }
  post_cache:
    in: cache
    kind: port
    methods:
      get: { params: { key: string }, returns: [bytes], errors: [] }
  mailer:
    in: mailer_adapter
    kind: external
    methods:
      send: { params: { to: string, body: string }, errors: [] }
```

- [ ] **Step 2: 테스트 목록** `TestSamplesLoadClean`에 `relations`를, check `TestSamples`에 `{name: "relations"}`(진단 없음)를 더한다. tolchain의 기대 진단은 그대로다.

- [ ] **Step 3: 통과 확인** `go test ./...` PASS. 실패하면 샘플이 아니라 검사 구현과 스펙을 대조한다.

- [ ] **Step 4: 모델 문서** `docs/design/01-model.md`를 v0.4로 고친다: 머리의 버전, `version` 규칙, 참조 필드 표(`needs`, `uses`, `kind`(이벤트), `publishers`·`subscribers`, `on`, `calls`, `emits`), 노드 종류 표(L2 `role`·`needs`, L3 `event_kinds`·`events`, L4 `flows`), "실행 흐름과 시퀀스는 모델에 넣지 않는다" 문장 삭제와 흐름 절, 설계 검사 표. 생성 파일 표는 계획 3에서 고친다. `tools/doc-lint.sh`를 돌린다.

- [ ] **Step 5: 커밋** `feat: v0.4 샘플과 모델 문서`
