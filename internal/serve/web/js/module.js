// 모듈. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 모듈 ---------- */
function moduleSide() {
  const item = (id, extra = '', label = '') => `<button type="button" data-go="${esc(id)}" aria-current="${id === current}"${label ? ` title="${esc(id)}"` : ''}>${esc(label || id)}${unitSt(id) ? `<small class="st ${unitSt(id)}">${ST[unitSt(id)]}</small>` : ''}</button>${extra}`;
  let h = '';
  // 분석 모드는 지도와 같은 영역으로 묶고, 하위 패키지는 뿌리 모듈 아래에 뿌리 기준 상대 경로로 넣는다. ID는 툴팁이다.
  if (S.mode === 'analyze') {
    for (const r of areaRows(topUnits())) {
      const roots = r.ids.filter(id => r.rootOf[id] === id);
      h += `<h2>${esc(r.label)}</h2><ul>${roots.map(id => {
        const kids = r.ids.filter(k => k !== id && r.rootOf[k] === id);
        const rel = k => unitPath(unitNode(k)).slice(unitPath(unitNode(id)).length + 1);
        return `<li>${item(id, kids.length ? `<ul>${kids.map(k => `<li>${item(k, '', rel(k))}</li>`).join('')}</ul>` : '')}</li>`;
      }).join('')}</ul>`;
    }
    return h;
  }
  for (const t of tiersTopDown()) {
    const us = topUnits().filter(u => (u.tier || '') === t);
    if (!us.length) continue;
    h += `<h2>${esc(t || 'tier 없음')}</h2><ul>${us.map(u => {
      const open = u.type === 'grp' && (u.id === current || u.members.includes(current));
      return `<li>${item(u.id, open ? `<ul>${u.members.map(m => `<li>${item(m)}</li>`).join('')}</ul>` : '')}</li>`;
    }).join('')}</ul>`;
  }
  return h;
}
// connSections는 모듈 ids와 바깥 모듈 사이 관계를 이 모듈 입장의 말로 묶는다. 쓰는 쪽 → 쓰이는 쪽 관계는 "쓰는 것"·"쓰는 쪽",
// 이벤트는 "발행"·"구독", 핸들러 등록은 "등록"·"등록받음"이다. 같은 말과 같은 상대·같은 이 모듈 쪽 패키지는 한 줄로 모은다.
// 분석 모드는 import한 패키지에서 실제로 쓴 export 이름을 무엇에 넣는다.
const CONN_TITLE = { 사용: '쓰는 것', 제공: '쓰는 쪽' };
function connSections(current, rels, ids = [current], uses = {}) {
  const own = new Set(ids), secs = {};
  for (const r of rels) {
    const a = own.has(unitOf(r.from)) || own.has(r.from), b = own.has(unitOf(r.to)) || own.has(r.to);
    if (a === b) continue;
    const dir = a ? 0 : 1, other = a ? r.to : r.from, side = a ? r.from : r.to;
    const title = (w => CONN_TITLE[w] || w)((REL_WORDS[r.kind] || ['사용', '제공'])[dir]);
    const sec = secs[title] ||= { title, rows: [] };
    let row = sec.rows.find(x => x.other === other && x.side === side);
    if (!row) sec.rows.push(row = { other, side, kinds: [], what: [], names: [], why: [] });
    if (!row.kinds.includes(r.kind)) row.kinds.push(r.kind);
    if (r.label && r.kind !== 'depends' && r.kind !== 'exception' && !row.what.includes(r.label)) row.what.push(r.label);
    if (r.why && !row.why.includes(r.why)) row.why.push(r.why);
    for (const n of uses[`${r.from}>${r.to}`] || []) if (!row.names.some(x => x.name === n.name)) row.names.push(n);
  }
  const order = ['쓰는 것', '쓰는 쪽', '발행', '구독', '등록', '등록받음'];
  // 분석 모드는 같은 뿌리의 패키지를 붙여 놓는다.
  const root = id => (S.mode === 'analyze' && typeof mapUnit === 'function' && mapUnit(id)) || id;
  return order.filter(t => secs[t]).map(t => {
    secs[t].rows.sort((x, y) => root(x.other).localeCompare(root(y.other)) || x.other.localeCompare(y.other) || x.side.localeCompare(y.side));
    return secs[t];
  });
}
const NAME_KIND_ORDER = ['함수', '인터페이스', '타입', '에러', '상수', '변수'];
const NAMES_SHOWN = 8;
function nameChips(names) {
  const ns = [...names].sort((a, b) => NAME_KIND_ORDER.indexOf(a.kind) - NAME_KIND_ORDER.indexOf(b.kind) || a.name.localeCompare(b.name));
  const chip = n => `<span class="nm"><span class="id">${esc(n.name)}</span><small>${esc(n.kind)}</small></span>`;
  const head = ns.slice(0, NAMES_SHOWN).map(chip).join('');
  return ns.length > NAMES_SHOWN ? `${head}<details class="more"><summary>외 ${ns.length - NAMES_SHOWN}</summary>${ns.slice(NAMES_SHOWN).map(chip).join('')}</details>` : head;
}
// connTable은 connSections를 상대·이 모듈 쪽·무엇·설명 표로 그린다. 분석 모드에서 이름 없이 import만 한 상대는 절 끝 한 줄로 모은다.
function connTable(secs, ids) {
  if (!secs.length) return '<p class="muted">다른 모듈과 관계가 없다</p>';
  const sideCol = ids.length > 1, whyCol = secs.some(s => s.rows.some(r => r.why.length));
  const n = 2 + sideCol + whyCol, base = unitPath(unitNode(ids[0]));
  const rel = id => id === ids[0] ? '' : unitPath(unitNode(id)).slice(base.length + 1);
  let h = `<table class="conn"><thead><tr><th>상대</th>${sideCol ? '<th>이 모듈 쪽</th>' : ''}<th>무엇</th>${whyCol ? '<th>설명</th>' : ''}</tr></thead><tbody>`;
  for (const s of secs) {
    // 무엇·설명이 없는 상대는 절 끝 한 줄로 모은다. 분석 모드에서는 이름 없이 import만 한 상대다.
    const empty = r => !r.names.length && !r.what.length && !r.why.length && r.kinds.every(k => k === 'depends');
    const rows = s.rows.filter(r => !empty(r));
    const bare = [...new Set(s.rows.filter(empty).map(r => r.other))].filter(o => !rows.some(r => r.other === o));
    h += `<tr class="sec"><th colspan="${n}">${esc(s.title)}</th></tr>`;
    let prev = '';
    for (const r of rows) {
      const what = r.names.length ? nameChips(r.names)
        : r.what.length ? r.what.map(l => `<span class="nm">${labelLink(r.kinds.find(k => k !== 'depends') || 'depends', l)}${r.kinds.some(k => KINDS[k] && !REL_WORDS[k] && k !== 'depends') ? `<small>${esc(r.kinds.filter(k => k !== 'depends').map(k => KINDS[k].name).join(', '))}</small>` : ''}</span>`).join('')
        : r.kinds.filter(k => k !== 'depends').map(k => `<span class="muted">${esc(KINDS[k].name)}</span>`).join(', ');
      h += `<tr data-other="${esc(r.other)}"${r.other !== prev ? ' class="first"' : ''}><td class="who">${unitLink(r.other)}</td>${sideCol ? `<td class="side id">${esc(rel(r.side))}</td>` : ''}<td class="what">${what}</td>${whyCol ? `<td class="why">${esc(r.why.join('. '))}</td>` : ''}</tr>`;
      prev = r.other;
    }
    if (bare.length) h += `<tr class="bare first"><td colspan="${n}">${S.mode === 'analyze' ? '<span class="muted">이름 없이 import</span> ' : ''}${inlineList(bare.sort().map(unitLink))}</td></tr>`;
  }
  return h + '</tbody></table>';
}
function renderModule() {
  const all = topUnits();
  // 처음 들어오면 지도에서 마지막으로 본 모듈, 없으면 가장 많이 쓰이는 모듈을 고른다.
  if (!current || !unitNode(current)) {
    const fanIn = id => new Set(S.relations.filter(r => unitOf(r.to) === id && unitOf(r.from) !== id).map(r => unitOf(r.from))).size;
    current = unitNode(mapFocus) ? mapFocus : all.length ? [...all].sort((a, b) => fanIn(b.id) - fanIn(a.id) || a.id.localeCompare(b.id))[0].id : '';
  }
  setSide(moduleSide());
  if (!current) { $('#main').innerHTML = '<p class="muted">설계에 모듈이 없다.</p>'; return; }
  const u = unitNode(current), kind = u.type;
  const ids = kind === 'grp' ? [current, ...u.members] : [current];
  const facts = [];
  if (kind === 'grp') facts.push(['멤버', inlineList(u.members.map(unitLink))]);
  if (kind === 'mem') facts.push(['그룹', unitLink(u.group)]);
  if (kind === 'mem' && u.handles.length) facts.push(['처리하는 값', inlineList(u.handles.map(h => `<span class="id">${esc(h)}</span>`))]);
  for (const [col, k, label] of [['contracts', 'contract', '계약'], ['eks', 'ek', '이벤트 종류'], ['events', 'ev', '이벤트'], ['flows', 'flow', '흐름'], ['hks', 'hk', '핸들러 종류'], ['types', 'type', '타입'], ['enums', 'enum', 'enum'], ['errors', 'err', '에러'], ['consts', 'const', '상수'], ['funcs', 'func', '함수']]) {
    const list = Object.values(M[col]).filter(n => ids.includes(n.in));
    if (list.length) facts.push([label, inlineList(list.map(n => nodeLink(k, n.id)))]);
  }
  // 이 모듈(그룹이면 멤버까지) 디렉토리 바로 아래의 생성 파일
  const dirs = ids.map(id => (unitNode(id) || {}).path).filter(Boolean);
  const gen = S.files.filter(f => dirs.includes(f.path.slice(0, f.path.lastIndexOf('/'))));
  if (gen.length) facts.push(['생성 파일', `<span class="list">${gen.map(f => `<span class="id">${esc(f.path)}</span>`).join('')}</span>`]);
  if (S.canEdit && (kind === 'mod' || kind === 'grp')) {
    const opts = M.tiers.map(t => `<option ${t === u.tier ? 'selected' : ''}>${esc(t)}</option>`).join('');
    facts.unshift(['tier', `<select data-edit="tier" data-id="${esc(current)}" aria-label="tier 바꾸기">${opts}</select>`]);
    const deps = u.depends_on || [];
    const cands = topUnits().map(x => x.id).filter(x => x !== current && !deps.includes(x)).sort();
    facts.push(['depends_on', `<span class="deps">${deps.map(d => `<span class="dep">${unitLink(d)}<button type="button" class="x" data-rmdep="${esc(d)}" aria-label="${esc(d)} 의존 지우기">×</button></span>`).join('')}<select data-adddep="${esc(current)}" aria-label="의존 추가"><option value="">의존 추가</option>${cands.map(x => `<option>${esc(x)}</option>`).join('')}</select></span>`]);
    if (kind === 'mod') facts.push(['', `<button type="button" class="rmmod" data-rmmod="${esc(current)}">모듈 지우기</button>`]);
  }
  if (S.mode === 'analyze' && S.draft) {
    const r = areaRows(all).find(x => x.ids.includes(current));
    const kids = r.ids.filter(k => k !== current && r.rootOf[k] === current);
    if (kids.length) facts.push(['하위 패키지', inlineList(kids.map(unitLink))]);
    const ext = (S.draft.report.external || []).filter(e => e.users.includes(current)).map(e => `<span class="id">${esc(e.module)}</span>`);
    if (ext.length) facts.push(['외부 의존', inlineList(ext)]);
  }
  const iss = issuesFor(ownedKeys(current));
  if (iss.length) facts.push(['진단', `<div class="issuelist">${issueRows(iss)}</div>`]);
  const area = S.mode === 'analyze' ? (areaRows(all).find(x => x.ids.includes(current)) || {}).label : '';
  const where = [kind === 'grp' ? '그룹' : kind === 'mem' ? '그룹 멤버' : '모듈', area ? `영역 ${area}` : tierOf(current) ? `tier ${tierOf(current)}` : '', u.path && u.path !== current ? u.path : '', u.role].filter(Boolean).map(esc).join(', ');
  // 분석 모드의 뿌리 모듈은 지도처럼 하위 패키지까지 한 모듈로 본다.
  let own = [current], uses = {}, l3 = '';
  if (S.mode === 'analyze' && S.draft) {
    const r = areaRows(all).find(x => x.ids.includes(current));
    own = [current, ...r.ids.filter(k => k !== current && r.rootOf[k] === current)];
    for (const x of S.draft.report.uses || []) uses[`${x.from}>${x.to}`] = x.names;
    const hints = (S.draft.report.hints || []).filter(x => own.includes(x.module));
    if (hints.length) l3 = `<h2 class="sect">L3 후보</h2><p class="muted">손으로 쓴 선언이다. 설계에 올릴 때는 gen: false로 둔다.</p><table class="hints"><thead><tr>${own.length > 1 ? '<th>패키지</th>' : ''}<th>종류</th><th>이름</th><th>위치</th></tr></thead><tbody>${hints.map(x => `<tr>${own.length > 1 ? `<td>${unitLink(x.module)}</td>` : ''}<td>${esc(x.kind)}${x.type ? ` <span class="id">${esc(x.type)}</span>` : ''}</td><td class="id">${x.names.map(esc).join(', ')}</td><td class="muted id">${esc(x.where)}</td></tr>`).join('')}</tbody></table>`;
  }
  $('#main').innerHTML = `<h1>${esc(current)}${stMark(unitSt(current))}</h1><p class="sub">${where}</p>${u.summary ? `<p class="summary">${esc(u.summary)}</p>` : ''}
    ${facts.length ? `<dl class="facts">${facts.map(([t, d]) => `<dt>${t}</dt><dd>${d}</dd>`).join('')}</dl>` : ''}
    <h2 class="sect">연결</h2>${connTable(connSections(current, S.relations, own, uses), own)}${l3}`;
}
