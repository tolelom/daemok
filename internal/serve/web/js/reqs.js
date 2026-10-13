// 요구사항. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 요구사항 ---------- */
function renderReqs() {
  setSide('');
  const all = Object.values(M.reqs);
  const by = {};
  for (const [col, k] of Object.entries(KEYKIND)) for (const n of Object.values(M[col] || {})) for (const r of n.satisfies || []) (by[r] = by[r] || []).push([k, n.id]);
  // active인데 충족 요소가 없는 요구사항은 설계에서 빠진 것이라 따로 고를 수 있게 한다.
  const unmet = r => r.status === 'active' && !by[r.id];
  const filters = [['all', '전체', all.length], ...['active', 'held', 'dropped'].map(st => [st, st, all.filter(r => r.status === st).length]), ['unmet', '충족 요소 없음', all.filter(unmet).length]].filter(([k, , n]) => k === 'all' || n);
  if (!filters.some(([k]) => k === reqFilter)) reqFilter = 'all';
  const reqs = all.filter(r => reqFilter === 'all' || (reqFilter === 'unmet' ? unmet(r) : r.status === reqFilter));
  const bar = all.length ? `<div class="reqbar" role="group" aria-label="요구사항 고르기">${filters.map(([k, label, n]) => `<button type="button" data-reqf="${k}" aria-pressed="${k === reqFilter}">${esc(label)} <span class="muted">${n}</span></button>`).join('')}</div>` : '';
  $('#main').innerHTML = `<h1 class="plain">요구사항</h1>${bar}<div class="reqs">${reqs.length ? reqs.map(r => {
    const iss = issuesFor(['req:' + r.id]);
    return `<section class="req${unmet(r) ? ' unmet' : ''}"><h2>${esc(r.id)}<small class="${r.status === 'active' ? 'active' : ''}">${esc(r.status)}</small></h2><p>${esc(r.text)}</p><p class="by">${by[r.id] ? `충족: ${inlineList(by[r.id].map(([k, id]) => nodeLink(k, id)))}` : '연결된 설계 요소 없음'}</p>${iss.length ? diagList(iss) : ''}</section>`;
  }).join('') : '<p class="muted">설계에 요구사항이 없다.</p>'}</div>`;
}
