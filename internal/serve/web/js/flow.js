// 흐름. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 흐름 ---------- */
// 두 흐름의 단계를 같은 이름끼리 같은 행에 놓는다(LCS). 한쪽에만 있는 단계는 다른 쪽이 빈칸이다.
function alignSteps(flows) {
  if (flows.length === 1) return flows[0].steps.map((_, i) => [i]);
  const a = flows[0].steps.map(x => x.id), b = flows[1].steps.map(x => x.id);
  const L = Array.from({ length: a.length + 1 }, () => new Array(b.length + 1).fill(0));
  for (let i = a.length - 1; i >= 0; i--) for (let j = b.length - 1; j >= 0; j--) L[i][j] = a[i] === b[j] ? L[i + 1][j + 1] + 1 : Math.max(L[i + 1][j], L[i][j + 1]);
  const out = [];
  let i = 0, j = 0;
  while (i < a.length || j < b.length) {
    if (i < a.length && j < b.length && a[i] === b[j]) out.push([i++, j++]);
    else if (j >= b.length || (i < a.length && L[i + 1][j] >= L[i][j + 1])) out.push([i++, null]);
    else out.push([null, j++]);
  }
  return out;
}
// 되돌리기 괄호: 되돌리기를 등록한 단계에서 커밋 단계(커밋이 없으면 마지막 단계)까지. 안쪽 괄호가 단계 이름에 가깝다.
function sideInfo(f, s, rows) {
  const rowOf = {};
  rows.forEach((r, ri) => { if (r.kind === 'step' && r.idx[s] != null) rowOf[r.idx[s]] = ri; });
  const ci = f.steps.findIndex(x => x.commit);
  const last = ci >= 0 ? ci : f.steps.length - 1;
  const fi = f.steps.findIndex(x => x.id === failAt);
  const failed = fi >= 0 ? f.steps[fi] : null;
  const order = failed ? failed.onFailUndo : [];
  const lanes = f.steps.map((st, i) => ({ st, i })).filter(x => x.st.undo && x.i < (ci >= 0 ? ci : f.steps.length))
    .map(x => {
      const hot = !!failed && x.i < fi && order.includes(x.st.undo);
      return { name: x.st.undo, i: x.i, start: rowOf[x.i], end: rowOf[last], hot, cold: !!failed && !hot, ord: order.indexOf(x.st.undo) + 1 };
    })
    .sort((p, q) => q.start - p.start);
  return { f, lanes, fi, cols: 5 + lanes.length };
}
const heat = l => (l.hot ? ' hot' : l.cold ? ' cold' : '');
function laneCells(info, ri) {
  const starter = info.lanes.find(l => l.start === ri);
  let h = `<div class="ln${starter ? ' cross' + heat(starter) : ''}"></div>`;
  info.lanes.forEach((l, k) => {
    let st = '', owner = l;
    if (ri === l.start && ri === l.end) st = 'end';
    else if (ri === l.start) st = 'start';
    else if (ri > l.start && ri < l.end) st = 'mid';
    else if (ri === l.end) st = 'end';
    if (!st) { const o = info.lanes.find((l2, k2) => k2 > k && l2.start === ri); if (o) { st = 'cross'; owner = o; } }
    h += `<div class="ln${st ? ' ' + st + heat(owner) : ''}"></div>`;
  });
  return h;
}
function stepCells(info, ri, i) {
  if (i == null) return `<div class="r"><div class="c"></div><div class="c"></div>${laneCells(info, ri)}<div class="c"></div><div class="c"></div></div>`;
  const s = info.f.steps[i];
  const st = info.fi < 0 ? '' : i === info.fi ? ' failed' : i > info.fi ? ' skipped' : '';
  const lane = info.lanes.find(l => l.i === i);
  const undo = lane ? `<span class="undo${lane.cold ? ' cold' : ''}">${esc(lane.name)}</span>${lane.hot ? `<span class="ord">${lane.ord}번째로 부름</span>` : ''}` : '';
  const notes = [
    s.fatal ? '<span class="fatal">실패하면 Fatal</span>' : '',
    ...s.calls.map(c => `<span>호출 <button type="button" class="link" data-open="contract:${esc(c.split('.')[0])}">${esc(c)}</button></span>`),
    ...s.emits.map(e => `<span class="emit">발행 <button type="button" class="link" data-open="ev:${esc(e)}">${esc(e)}</button></span>`),
  ].filter(Boolean).join('');
  return `<div class="r" data-fail="${esc(s.id)}"><div class="c n${st}">${i + 1}</div><div class="c${st}"><button type="button" class="stepbtn">${esc(s.id)}</button>${s.summary ? `<div class="note">${esc(s.summary)}</div>` : ''}</div>${laneCells(info, ri)}<div class="c${st}">${undo}</div><div class="c${st}"><div class="notes">${notes}</div></div></div>`;
}
function flowGrid(shown) {
  const rows = [];
  for (const idx of alignSteps(shown)) {
    rows.push({ kind: 'step', idx });
    const sides = shown.map((f, s) => idx[s] != null && f.steps[idx[s]].commit);
    if (sides.some(Boolean)) rows.push({ kind: 'commit', sides });
  }
  const infos = shown.map((f, s) => sideInfo(f, s, rows));
  const two = shown.length > 1;
  const gap = two ? '<div></div>' : '';
  const tmpl = infos.map(x => `2.4em minmax(9em, max-content) 10px ${'14px '.repeat(x.lanes.length)}minmax(7em, max-content) minmax(8em, max-content)`).join(' 28px ');
  const span = x => `style="grid-column: span ${x.cols}"`;
  let h = '';
  if (two) h += infos.map(x => `<div class="sidehead" ${span(x)}><h2>${esc(x.f.id)}</h2><p>${esc(x.f.in)} 모듈, 단계 ${x.f.steps.length}${x.f.summary ? '. ' + esc(x.f.summary) : ''}</p></div>`).join(gap);
  h += infos.map(x => `<div class="th">#</div><div class="th">단계</div><div class="th" style="grid-column: span ${1 + x.lanes.length}"></div><div class="th">되돌리기</div><div class="th">호출·발행</div>`).join(gap);
  h += '<div class="sect">커밋 전. 실패하면 괄호가 열려 있는 되돌리기를 안쪽부터 부른다</div>';
  rows.forEach((r, ri) => {
    h += infos.map((x, s) => {
      if (r.kind === 'step') return stepCells(x, ri, r.idx[s]);
      if (r.sides[s]) return `<div class="c commitmark" ${span(x)}>커밋. 이후 단계는 실패해도 되돌리지 않는다</div>`;
      return `<div class="c"></div><div class="c"></div>${laneCells(x, ri)}<div class="c"></div><div class="c"></div>`;
    }).join(gap);
  });
  return `<div class="fgwrap"><div class="fg" style="grid-template-columns: ${tmpl}">${h}</div></div>`;
}
function failNote(f) {
  const s = f && f.steps.find(x => x.id === failAt);
  if (!s) return '<p class="hint">단계를 누르면 그 단계가 실패했을 때 생성된 Run 함수가 무엇을 어떤 순서로 되돌리는지 표시한다.</p>';
  const committedBefore = f.steps.slice(0, f.steps.indexOf(s)).some(x => x.commit);
  const at = `<span class="id">${esc(s.id)}</span>에서 실패하면`;
  let body;
  if (s.onFailFatal) body = `${at} 되돌리지 않고 Fatal을 부른다.`;
  else if (committedBefore) body = `${at} 커밋 뒤라 되돌리지 않고 오류를 돌려준다.`;
  else if (!s.onFailUndo.length) body = `${at} 되돌릴 것이 없어 오류만 돌려준다.`;
  else body = `${at} ${s.onFailUndo.map(u => `<span class="id">${esc(u)}</span>`).join(', ')} 순서로 되돌린 뒤 오류를 돌려준다. 되돌리기가 실패하면 Fatal이다.`;
  return `<div class="failnote"><p>${body}<button type="button" class="clear" data-clearfail="1">지우기</button></p></div>`;
}
function renderFlow() {
  const flows = Object.values(M.flows);
  if (!flows.length) { setSide(''); $('#main').innerHTML = '<h1 class="plain">흐름</h1><p class="muted">설계에 흐름이 없다.</p>'; return; }
  if (!M.flows[flowSel]) flowSel = flows[0].id;
  if (flowCompare && (!M.flows[flowCompare] || flowCompare === flowSel)) flowCompare = '';
  setSide(`<h2>흐름</h2><ul>${flows.map(f => `<li><button type="button" data-flowpick="${esc(f.id)}" aria-current="${f.id === flowSel}">${esc(f.id)}<small>${esc(f.in)}</small></button></li>`).join('')}</ul>`);
  const f = M.flows[flowSel], shown = [f, ...(flowCompare ? [M.flows[flowCompare]] : [])];
  const failFlow = shown.find(x => x.steps.some(s => s.id === failAt));
  const cmp = flows.length > 1 ? `<label>나란히 비교 <select id="cmp"><option value="">비교 안 함</option>${flows.filter(x => x.id !== flowSel).map(x => `<option value="${esc(x.id)}" ${x.id === flowCompare ? 'selected' : ''}>${esc(x.id)}</option>`).join('')}</select></label>` : '';
  const head = x => `<p class="sub">${unitLink(x.in)} 모듈, 단계 ${x.steps.length}${x.on ? `, 시작 이벤트 <span class="id">${esc(x.on)}</span>` : ''}</p>${x.summary ? `<p class="summary">${esc(x.summary)}</p>` : ''}`;
  const body = flowGrid(shown);
  $('#main').innerHTML = `<div class="flowhead"><div>${flowCompare ? '<h1 class="plain">흐름 비교</h1>' : `<h1>${esc(flowSel)}</h1>${head(f)}`}</div>${cmp}</div>${failNote(failFlow)}${body}`;
  const sel = $('#cmp');
  if (sel) sel.addEventListener('change', e => { flowCompare = e.target.value; render(); });
}
