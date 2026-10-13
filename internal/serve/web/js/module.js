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
function relGroups(list, side) {
  if (!list.length) return '<p class="muted">없음</p>';
  const rows = [];
  for (const r of list) {
    const other = side === 'out' ? r.to : r.from;
    let row = rows.find(x => x.kind === r.kind && x.other === other);
    if (!row) rows.push(row = { kind: r.kind, other, labels: [], whys: [] });
    if (r.label && r.kind !== 'depends' && r.kind !== 'exception' && !row.labels.includes(r.label)) row.labels.push(r.label);
    if (r.why && !row.whys.includes(r.why)) row.whys.push(r.why);
  }
  return Object.keys(KINDS).filter(k => rows.some(r => r.kind === k)).map(k => {
    const rs = rows.filter(r => r.kind === k).sort((a, b) => a.other.localeCompare(b.other));
    const bare = rs.every(r => !r.labels.length && !r.whys.length);
    const body = bare
      ? `<p>${inlineList(rs.map(r => unitLink(r.other)))}</p>`
      : `<ul>${rs.map(r => `<li><div>${unitLink(r.other)}${r.labels.length ? ` <span class="labels">${inlineList(r.labels.map(l => labelLink(k, l)))}</span>` : ''}</div>${r.whys.length ? `<div class="why">${esc(r.whys.join('. '))}</div>` : ''}</li>`).join('')}</ul>`;
    return `<div class="kgroup k-${k}"><span class="kname">${KINDS[k].name}</span>${body}</div>`;
  }).join('');
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
    const hints = (S.draft.report.hints || []).filter(x => x.module === current);
    if (hints.length) facts.push(['L3 후보', `<ul class="hints">${hints.map(x => `<li>${esc(x.kind)}${x.type ? ` <span class="id">${esc(x.type)}</span>` : ''}: ${x.names.map(n => `<span class="id">${esc(n)}</span>`).join(', ')} <span class="muted">${esc(x.where)}</span></li>`).join('')}</ul>`]);
  }
  const iss = issuesFor(ownedKeys(current));
  if (iss.length) facts.push(['진단', `<div class="issuelist">${issueRows(iss)}</div>`]);
  const area = S.mode === 'analyze' ? (areaRows(all).find(x => x.ids.includes(current)) || {}).label : '';
  const where = [kind === 'grp' ? '그룹' : kind === 'mem' ? '그룹 멤버' : '모듈', area ? `영역 ${area}` : tierOf(current) ? `tier ${tierOf(current)}` : '', u.path && u.path !== current ? u.path : '', u.role].filter(Boolean).map(esc).join(', ');
  $('#main').innerHTML = `<h1>${esc(current)}${stMark(unitSt(current))}</h1><p class="sub">${where}</p>${u.summary ? `<p class="summary">${esc(u.summary)}</p>` : ''}
    ${facts.length ? `<dl class="facts">${facts.map(([t, d]) => `<dt>${t}</dt><dd>${d}</dd>`).join('')}</dl>` : ''}
    <div class="cols"><div><h2>이 모듈이 쓰는 것</h2>${relGroups(S.relations.filter(r => r.from === current), 'out')}</div><div><h2>이 모듈을 쓰는 쪽</h2>${relGroups(S.relations.filter(r => r.to === current), 'in')}</div></div>`;
}
