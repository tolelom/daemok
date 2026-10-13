// 지도. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 지도 ---------- */
const NW = 176, NH = 58, GAP = 22, LEFT = 120, TOP = 12, BAND_PAD = 18, PER = 5;
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
  const tail = `<hr>${kindFilter()}<div class="legend"><span><svg width="24" height="10" aria-hidden="true"><circle class="issue-dot err" cx="12" cy="5"/></svg>오류가 있는 모듈</span><span><svg width="24" height="10" aria-hidden="true"><circle class="issue-dot warn" cx="12" cy="5"/></svg>경고만 있는 모듈</span></div><label><input type="checkbox" id="showall" ${showAll ? 'checked' : ''}> 고른 종류의 관계선 모두 보기</label>${S.canEdit ? `<hr><form id="addmod" class="addmod"><h3>모듈 추가</h3><input id="addmodid" placeholder="모듈 ID (snake_case)" autocomplete="off" spellcheck="false" aria-label="모듈 ID"><select id="addmodtier" aria-label="tier">${[...M.tiers].reverse().map(t => `<option>${esc(t)}</option>`).join('')}</select><button type="submit">추가</button></form>` : ''}`;
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
function renderMap() {
  setSide('');
  const units = topUnits(), pos = {}, bands = [], order = tierOrder(units, mapEdges());
  let y = TOP;
  for (const t of tiersTopDown()) {
    const us = order[t].map(id => M.mods[id] || M.groups[id]);
    if (!us.length) continue;
    const top = y, rows = Math.ceil(us.length / PER);
    us.forEach((u, i) => { pos[u.id] = { x: LEFT + (i % PER) * (NW + GAP), y: y + BAND_PAD + Math.floor(i / PER) * (NH + GAP) }; });
    y += BAND_PAD * 2 + rows * NH + (rows - 1) * GAP;
    bands.push({ t: t || 'tier 없음', y0: top, y1: y });
    y += 12;
  }
  const W = LEFT + PER * (NW + GAP) + 6, H = Math.max(y, 80);
  const edges = mapEdges().filter(e => pos[e.from] && pos[e.to]);
  let s = `<svg viewBox="0 0 ${W} ${H}" width="${W}" id="mapsvg" role="img" aria-label="모듈 지도">`;
  s += `<defs>${Object.keys(KINDS).map(k => `<marker id="ar-${k}" viewBox="0 0 8 8" refX="7" refY="4" markerWidth="7" markerHeight="7" orient="auto"><path d="M0,0 L8,4 L0,8 z" style="fill: var(--k-${k})"/></marker>`).join('')}</defs>`;
  for (const b of bands) s += `<g class="band"><rect x="0" y="${b.y0}" width="${W}" height="${b.y1 - b.y0}" rx="6"/><text x="14" y="${b.y0 + 24}">${esc(b.t)}</text></g>`;
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
    const sub = u.type === 'grp' ? `그룹, 멤버 ${u.members.length}` : (u.summary || (u.path !== u.id ? u.path : '') || u.role || '');
    s += `<g class="unit ${st ? 'st-' + st : ''}" tabindex="0" role="button" data-id="${esc(u.id)}" aria-label="${esc(u.id)} 모듈 보기">`;
    if (u.type === 'grp') s += `<rect class="stack" x="${p.x + 5}" y="${p.y - 5}" width="${NW}" height="${NH}" rx="6"/>`;
    s += `<rect class="box" x="${p.x}" y="${p.y}" width="${NW}" height="${NH}" rx="6"/>`;
    s += `<text class="nm" x="${p.x + 12}" y="${p.y + 25}">${esc(u.id)}</text><text class="sb" x="${p.x + 12}" y="${p.y + 44}">${esc(fit(sub, 22))}</text>`;
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
