// 지도. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 지도 ---------- */
const NW = 176, NH = 58, GAP = 22, LEFT = 120, TOP = 12, BAND_PAD = 18, PER = 5;
const EDGE_FAINT_MAX = 60;  // 관계선이 이보다 많으면 흐린 선이 지도를 덮어서 처음에는 숨긴다
// edgesHidden은 관계선을 올렸을 때만 보일지다. 분석 모드의 영역 띠는 의존 순서가 아니라서 흐린 선이 흐름을 보여 주지 못한다.
const edgesHidden = n => S.mode === 'analyze' || n > EDGE_FAINT_MAX;
const NAME_PX = 14.5, NAME_MIN_PX = 12, NAME_W = NW - 34, MONO_EM = .6;
function mapEdges() {
  const out = new Map();
  for (const r of S.relations) {
    const a = unitOf(r.from), b = unitOf(r.to);
    if (!a || !b || a === b) continue;
    const key = `${a}>${b}>${r.kind}`;
    if (!out.has(key)) out.set(key, { from: a, to: b, kind: r.kind, labels: new Set() });
    if (r.label) out.get(key).labels.add(r.label);
  }
  return [...out.values()];
}
// kindFilter는 이 설계에 있는 관계 종류만 개수와 함께 체크 상자로 보인다. 끈 종류는 지도에서 숨긴다.
function kindFilter() {
  const count = {};
  mapEdges().forEach(e => { count[e.kind] = (count[e.kind] || 0) + 1; });
  const kinds = Object.keys(KINDS).filter(k => count[k]);
  if (!kinds.length) return '<p class="muted">관계선이 없다</p>';
  return `<div class="legend kfilter">${kinds.map(k => `<label class="k-${k}"><input type="checkbox" data-kind="${k}" ${hiddenKinds.has(k) ? '' : 'checked'}><span class="sw ${KINDS[k].style}"></span>${KINDS[k].name} <span class="muted">${count[k]}</span></label>`).join('')}</div>`;
}
function mapPanel(id) {
  const n = mapEdges().length;
  const tail = `<hr>${kindFilter()}${edgesHidden(n) ? `<p class="muted">관계선은 모듈에 올렸을 때만 보인다</p>` : ''}<div class="legend"><span><svg width="24" height="10" aria-hidden="true"><circle class="issue-dot err" cx="12" cy="5"/></svg>오류가 있는 모듈</span><span><svg width="24" height="10" aria-hidden="true"><circle class="issue-dot warn" cx="12" cy="5"/></svg>경고만 있는 모듈</span></div><label><input type="checkbox" id="showall" ${showAll ? 'checked' : ''}> 고른 종류의 관계선 모두 보기</label>${S.canEdit ? `<hr><form id="addmod" class="addmod"><h3>모듈 추가</h3><input id="addmodid" placeholder="모듈 ID (snake_case)" autocomplete="off" spellcheck="false" aria-label="모듈 ID"><select id="addmodtier" aria-label="tier">${[...M.tiers].reverse().map(t => `<option>${esc(t)}</option>`).join('')}</select><button type="submit">추가</button></form>` : ''}`;
  const u = unitNode(id);
  if (!u) return `<p class="muted">모듈에 마우스를 올리면 연결이 여기 적힌다. 누르면 모듈 보기로 간다.</p>${tail}`;
  const es = mapEdges();
  const part = (title, list, side) => {
    if (!list.length) return `<div class="rel"><h3>${title}</h3><p class="muted">없음</p></div>`;
    const kinds = Object.keys(KINDS).filter(k => list.some(e => e.kind === k));
    return `<div class="rel"><h3>${title}</h3>${kinds.map(k => `<p class="k-${k}"><span class="kname">${KINDS[k].name}</span> ${list.filter(e => e.kind === k).map(e => `<span class="id">${esc(e[side])}</span>`).join(', ')}</p>`).join('')}</div>`;
  };
  const sub = u.type === 'grp' ? `그룹, 멤버 ${u.members.length}` : (u.summary || '');
  return `<div><h2>${esc(id)}</h2>${sub ? `<p class="muted">${esc(sub)}</p>` : ''}</div>${part('쓰는 것', es.filter(e => e.from === id), 'to')}${part('쓰는 쪽', es.filter(e => e.to === id), 'from')}${tail}`;
}
// tierOrder는 tier마다 모듈 순서를 정한다. 다른 tier 이웃의 평균 칸 위치로 위아래를 번갈아 정렬해
// 관계선 교차를 줄인다. 이웃이 없거나 값이 같으면 설계 순서를 유지한다.
function tierOrder(units, edges) {
  const tiers = tiersTopDown();
  const rows = tiers.map(t => units.filter(u => (u.tier || '') === t).map(u => u.id));
  const nb = {};
  for (const e of edges) { (nb[e.from] ||= []).push(e.to); (nb[e.to] ||= []).push(e.from); }
  const slot = {};
  const place = r => r.forEach((id, i) => { slot[id] = i % PER; });
  rows.forEach(place);
  const sweep = r => {
    const own = new Set(r);
    const key = id => { const ns = (nb[id] || []).filter(n => n in slot && !own.has(n)); return ns.length ? ns.reduce((a, n) => a + slot[n], 0) / ns.length : slot[id]; };
    const keyed = r.map((id, i) => [id, key(id), i]).sort((a, b) => a[1] - b[1] || a[2] - b[2]);
    r.splice(0, r.length, ...keyed.map(k => k[0]));
    place(r);
  };
  for (let k = 0; k < 4; k++) (k % 2 ? [...rows].reverse() : rows).forEach(sweep);
  return Object.fromEntries(tiers.map((t, i) => [t, rows[i]]));
}
const unitPath = u => u.path || u.id;
// areaRows는 분석 모드의 띠다. 분석 모드의 tier는 import 깊이로 센 것이라 띠를 디렉토리 영역으로 나눈다.
// 하위 패키지는 가장 가까운 조상 모듈로 접고, 영역은 접은 뿌리의 부모 디렉토리다. 사이 디렉토리에 모듈 갈래가 둘 이상이면
// 조상은 담는 그릇이라 접지 않는다. 사이 디렉토리가 internal이면 그 조상 전용이라 갈래를 세지 않는다.
// 하위 패키지는 부모와 같은 띠에서 경로 순으로 부모 뒤에 온다. 띠는 tier 평균이 높은 영역부터다.
// rootOf는 모듈 ID → 접은 뿌리 모듈 ID다. 모듈 탭이 하위 패키지를 뿌리 아래에 넣는 데 쓴다.
function areaRows(units) {
  const paths = units.map(unitPath), has = new Set(paths);
  const branches = d => new Set(paths.filter(q => q.startsWith(d + '/')).map(q => q.slice(d.length + 1).split('/')[0])).size;
  const rootOf = p => {
    const seg = p.split('/');
    for (let i = seg.length - 1; i > 0; i--) {
      const a = seg.slice(0, i).join('/');
      if (!has.has(a)) continue;
      const between = seg.slice(i, -1).map((_, k) => seg.slice(0, i + k + 1).join('/'));
      return between.some(d => !d.endsWith('/internal') && d !== 'internal' && branches(d) > 1) ? p : rootOf(a);
    }
    return p;
  };
  const areas = new Map(), idOf = Object.fromEntries(units.map(u => [unitPath(u), u.id])), root = {};
  for (const u of units) {
    const r = rootOf(unitPath(u)), a = r.includes('/') ? r.slice(0, r.lastIndexOf('/')) : '.';
    root[u.id] = idOf[r];
    if (!areas.has(a)) areas.set(a, []);
    areas.get(a).push(u);
  }
  const avg = us => us.reduce((n, u) => n + M.tiers.indexOf(u.tier), 0) / us.length;
  return [...areas].map(([a, us]) => ({ area: a, label: a === '.' ? '루트' : a, avg: avg(us), ids: us.map(u => u.id).sort((x, y) => (unitPath(M.mods[x] || M.groups[x]) < unitPath(M.mods[y] || M.groups[y]) ? -1 : 1)) }))
    .sort((x, y) => y.avg - x.avg || (x.area < y.area ? -1 : 1))
    .map(r => Object.assign(r, { rootOf: root }));
}
// fitName은 이름이 상자에 들어가도록 글자를 줄이고, 그래도 넘치면 가운데를 줄인다. 끝(_service, _query)이 구분에 쓰여서다.
function fitName(id) {
  const fits = px => Math.floor(NAME_W / (px * MONO_EM));
  if (id.length <= fits(NAME_PX)) return { text: id, px: NAME_PX };
  const px = Math.max(NAME_MIN_PX, NAME_W / (id.length * MONO_EM)), n = fits(px);
  if (id.length <= n) return { text: id, px };
  const us = id.lastIndexOf('_'), tail = us > 0 && id.length - us <= n - 4 ? id.length - us : Math.ceil((n - 1) / 2);
  return { text: id.slice(0, n - 1 - tail) + '…' + id.slice(-tail), px };
}
// bandLines는 띠 이름을 상자 왼쪽 여백에 들어가게 / 뒤에서 줄을 나눈다. 한 줄은 BAND_CH자까지고, 더 긴 마디는 줄인다.
const BAND_CH = 12;
function bandLines(t) {
  const out = [];
  for (const seg of t.split(/(?<=\/)/)) {
    if (out.length && (out[out.length - 1] + seg).length <= BAND_CH) out[out.length - 1] += seg;
    else out.push(fit(seg, BAND_CH));
  }
  return out;
}
function renderMap() {
  setSide('');
  const units = topUnits(), pos = {}, bands = [], areaOf = {};
  const rows = S.mode === 'analyze' ? areaRows(units) : (order => tiersTopDown().map(t => ({ label: t || 'tier 없음', ids: order[t] })))(tierOrder(units, mapEdges()));
  let y = TOP;
  for (const r of rows) {
    const us = r.ids.map(id => M.mods[id] || M.groups[id]);
    if (!us.length) continue;
    const top = y, lines = Math.ceil(us.length / PER);
    us.forEach((u, i) => { pos[u.id] = { x: LEFT + (i % PER) * (NW + GAP), y: y + BAND_PAD + Math.floor(i / PER) * (NH + GAP) }; areaOf[u.id] = r.area; });
    y += BAND_PAD * 2 + lines * NH + (lines - 1) * GAP;
    bands.push({ t: r.label, y0: top, y1: y });
    y += 12;
  }
  const W = LEFT + PER * (NW + GAP) + 6, H = Math.max(y, 80);
  const edges = mapEdges().filter(e => pos[e.from] && pos[e.to]);
  let s = `<svg viewBox="0 0 ${W} ${H}" width="${W}" id="mapsvg" class="${edgesHidden(edges.length) ? 'many' : ''}" role="img" aria-label="모듈 지도">`;
  s += `<defs>${Object.keys(KINDS).map(k => `<marker id="ar-${k}" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto"><path d="M0,0 L8,4 L0,8 z" style="fill: var(--k-${k})"/></marker>`).join('')}</defs>`;
  for (const b of bands) s += `<g class="band"><rect x="0" y="${b.y0}" width="${W}" height="${b.y1 - b.y0}" rx="6"/><text x="14" y="${b.y0 + 24}">${bandLines(b.t).map((l, i) => `<tspan x="14" dy="${i ? 17 : 0}">${esc(l)}</tspan>`).join('')}</text></g>`;
  const anchor = (p, q) => { const cx = p.x + NW / 2, cy = p.y + NH / 2, tx = q.x + NW / 2, ty = q.y + NH / 2; return Math.abs(ty - cy) < 4 ? [cx + (tx > cx ? NW / 2 : -NW / 2), cy] : [cx, ty > cy ? p.y + NH : p.y]; };
  for (const e of edges) {
    const [x1, y1] = anchor(pos[e.from], pos[e.to]), [x2, y2] = anchor(pos[e.to], pos[e.from]);
    const my = (y1 + y2) / 2;
    const d = Math.abs(y2 - y1) < 4 ? `M${x1},${y1} C${(x1 + x2) / 2},${y1 - 40} ${(x1 + x2) / 2},${y2 - 40} ${x2},${y2}` : `M${x1},${y1} C${x1},${my} ${x2},${my} ${x2},${y2}`;
    const lbl = [...e.labels].slice(0, 2).join(', ') + (e.labels.size > 2 ? ` 외 ${e.labels.size - 2}` : '');
    s += `<path class="edge k-${e.kind} ${KINDS[e.kind].style}" d="${d}" data-from="${esc(e.from)}" data-to="${esc(e.to)}" data-kind="${e.kind}" marker-end="url(#ar-${e.kind})"/>`;
    if (lbl) s += `<text class="edge-label k-${e.kind}" x="${(x1 + x2) / 2 + 6}" y="${my - 4}" data-from="${esc(e.from)}" data-to="${esc(e.to)}" data-kind="${e.kind}">${esc(lbl)}</text>`;
  }
  for (const u of units) {
    const p = pos[u.id], st = unitSt(u.id), iss = worst(issuesFor(ownedKeys(u.id)));
    const a = areaOf[u.id], rel = a && a !== '.' ? unitPath(u).slice(a.length + 1) : u.path;
    const sub = u.type === 'grp' ? `그룹, 멤버 ${u.members.length}` : (u.summary || (rel && rel !== u.id ? rel : '') || u.role || '');
    const nm = fitName(u.id);
    s += `<g class="unit ${st ? 'st-' + st : ''}" tabindex="0" role="button" data-id="${esc(u.id)}" aria-label="${esc(u.id)} 모듈 보기"><title>${esc(u.id)}</title>`;
    if (u.type === 'grp') s += `<rect class="stack" x="${p.x + 5}" y="${p.y - 5}" width="${NW}" height="${NH}" rx="6"/>`;
    s += `<rect class="box" x="${p.x}" y="${p.y}" width="${NW}" height="${NH}" rx="6"/>`;
    s += `<text class="nm" x="${p.x + 12}" y="${p.y + 25}"${nm.px !== NAME_PX ? ` style="font-size: ${nm.px.toFixed(1)}px"` : ''}>${esc(nm.text)}</text><text class="sb" x="${p.x + 12}" y="${p.y + 44}">${esc(fit(sub, 22))}</text>`;
    if (iss) s += `<circle class="issue-dot ${iss}" cx="${p.x + NW - 12}" cy="${p.y + 12}"><title>${iss === 'err' ? '오류가 있다' : '경고가 있다'}</title></circle>`;
    s += `</g>`;
  }
  s += '</svg>';
  $('#main').innerHTML = `<div class="mapsearch"><input id="mapq" type="search" placeholder="모듈 찾기 (/)" autocomplete="off" spellcheck="false" aria-label="모듈 찾기"><span id="mapqn" class="muted"></span></div><div class="mapwrap"><div class="map">${s}</div><aside class="panel" id="mappanel"></aside></div>`;
  const svg = $('#mapsvg');
  const hi = id => {
    svg.classList.toggle('dim', !!id || !!($('#mapq') && $('#mapq').value.trim()));
    const near = new Set();
    svg.querySelectorAll('.edge, .edge-label').forEach(el => {
      const on = !hiddenKinds.has(el.dataset.kind) && (showAll || (id && (el.dataset.from === id || el.dataset.to === id)));
      el.classList.toggle('show', !!on);
      el.classList.toggle('off', hiddenKinds.has(el.dataset.kind));
      if (id && on) { near.add(el.dataset.from); near.add(el.dataset.to); }
    });
    svg.querySelectorAll('.unit').forEach(g => { g.classList.toggle('near', near.has(g.dataset.id)); g.classList.toggle('on', g.dataset.id === id); });
  };
  const panel = id => {
    mapFocus = id || mapFocus;
    $('#mappanel').innerHTML = mapPanel(mapFocus);
    $('#showall').addEventListener('change', e => { showAll = e.target.checked; hi(null); });
    $('#mappanel').querySelectorAll('[data-kind]').forEach(cb => cb.addEventListener('change', e => {
      if (e.target.checked) hiddenKinds.delete(e.target.dataset.kind); else hiddenKinds.add(e.target.dataset.kind);
      hi(null);
    }));
  };
  svg.querySelectorAll('.unit').forEach(g => {
    const on = () => { hi(g.dataset.id); panel(g.dataset.id); };
    g.addEventListener('mouseenter', on);
    g.addEventListener('focus', on);
    g.addEventListener('mouseleave', () => hi(null));
    g.addEventListener('blur', () => hi(null));
    g.addEventListener('click', () => go('module', g.dataset.id));
    g.addEventListener('keydown', e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); go('module', g.dataset.id); } });
  });
  const q = $('#mapq');
  const find = () => {
    const t = q.value.trim().toLowerCase();
    if (!t) { svg.classList.remove('dim'); svg.querySelectorAll('.unit').forEach(g => g.classList.remove('hit')); $('#mapqn').textContent = ''; return []; }
    const match = n => [n.id, n.path, n.summary].some(x => String(x || '').toLowerCase().includes(t));
    // 그룹 멤버는 지도에 따로 그리지 않으므로 맞으면 그 그룹을 강조한다.
    const hits = [...new Set([...units.filter(match).map(u => u.id), ...Object.values(M.members).filter(match).map(m => m.group)])];
    svg.classList.add('dim');
    svg.querySelectorAll('.unit').forEach(g => g.classList.toggle('hit', hits.includes(g.dataset.id)));
    $('#mapqn').textContent = `${hits.length}개`;
    return hits;
  };
  q.addEventListener('input', find);
  q.addEventListener('keydown', e => {
    if (e.key === 'Enter') { const h = find(); if (h.length) go('module', h[0]); }
    if (e.key === 'Escape') { q.value = ''; find(); q.blur(); }
  });
  panel(null);
  hi(null);
}
function fit(s, n) { s = String(s || ''); return s.length > n ? s.slice(0, n - 1) + '…' : s; }
