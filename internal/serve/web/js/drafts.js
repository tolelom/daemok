// 초안, L3 초안, 생성 미리보기. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 초안 ---------- */
function renderDraft() {
  setSide('');
  const d = S.draft, r = d.report;
  const skipped = (r.skipped || []).map(x => `<li><span class="id">${esc(x.path)}</span> <span class="muted">${esc(x.reason)}</span></li>`).join('');
  // 외부 의존을 쓰는 곳은 지도처럼 뿌리 모듈로 접는다. 하위 패키지까지 다 적으면 한 줄이 수십 개가 된다.
  const ext = (r.external || []).map(e => {
    const users = [...new Set((e.users || []).map(u => mapUnit(u) || u))].sort();
    return `<tr><td class="id">${esc(e.module)}</td><td>${users.length ? inlineList(users.map(unitLink)) : '<span class="muted">쓰는 모듈 없음</span>'}</td></tr>`;
  });
  // L3 후보는 지도의 영역 순서로 놓는다.
  const order = areaRows(topUnits()).flatMap(x => x.ids);
  const hints = [...(r.hints || [])].sort((a, b) => order.indexOf(a.module) - order.indexOf(b.module)).map(h => `<tr><td>${unitLink(h.module)}</td><td>${esc(h.kind)}${h.type ? ` <span class="id">${esc(h.type)}</span>` : ''}</td><td class="id">${h.names.map(esc).join(', ')}</td><td class="muted id">${esc(h.where)}</td></tr>`);
  const fan = (r.fanIn || []).map(x => `${unitLink(x.id)} ${x.count}`).join(', ') || '<span class="muted">없음</span>';
  const cycles = (r.cycles || []).map(c => `<li>${c.map(unitLink).join(', ')}</li>`).join('');
  const groups = (r.groupCandidates || []).map(g => `<li><span class="id">${esc(g.path)}</span>: ${g.members.map(unitLink).join(', ')} <span class="muted">공통 의존</span> ${g.common.map(unitLink).join(', ')}${(g.exceptions || []).length ? ` <span class="muted">예외 후보</span> ${g.exceptions.map(esc).join(', ')}` : ''}</li>`).join('');
  $('#main').innerHTML = `<h1 class="plain">초안</h1>
<p class="sub">설계 파일이 없어 코드의 패키지와 import에서 추정했다. tier 이름과 묶음은 사람이 정한다.</p>
<dl class="facts"><dt>패키지</dt><dd>${r.packages}</dd><dt>모듈</dt><dd>${r.modules}</dd><dt>실행 단위</dt><dd>${r.executables}</dd><dt>많이 쓰이는 패키지</dt><dd>${fan}</dd><dt>건너뛴 것</dt><dd>${skipped ? `<details><summary>${r.skipped.length}개</summary><ul>${skipped}</ul></details>` : '없음'}</dd></dl>
${cycles ? `<h2 class="sect">서로 import하는 패키지</h2><ul>${cycles}</ul>` : ''}
${groups ? `<h2 class="sect">그룹 후보</h2><p class="muted">그룹은 핸들러 종류가 있어야 해서 초안에 넣지 않았다. 묶을지는 사람이 정한다.</p><ul>${groups}</ul>` : ''}
${ext.length ? `<h2 class="sect">외부 의존</h2><table class="ext"><thead><tr><th>모듈</th><th>쓰는 곳</th></tr></thead><tbody>${ext.join('')}</tbody></table>` : ''}
${hints.length ? `<h2 class="sect">L3 노드 후보</h2><p class="muted">이미 손으로 쓴 선언이다. 설계에 올릴 때는 gen: false로 둔다.</p><table class="hints"><thead><tr><th>패키지</th><th>종류</th><th>이름</th><th>위치</th></tr></thead><tbody>${hints.join('')}</tbody></table>` : ''}
<h2 class="sect">초안 YAML</h2>
<div class="draftbar"><button type="button" class="primary" id="savedraft">design/에 저장</button><code>${esc(d.command)}</code><button type="button" id="copycmd">명령 복사</button></div>
<p id="savemsg" class="muted"></p>
<details class="yaml"><summary>${d.yaml.split('\n').length}줄 보기</summary><pre class="yaml">${esc(d.yaml)}</pre></details>`;
}

/* ---------- L3 초안 ---------- */
// 서버가 설계 대조를 다시 돌려 만들므로 탭을 열 때만 받고, 상태 버전이 바뀌면 다시 받는다.
let l3 = null, l3Version = -1, l3Loading = false;
function renderL3() {
  setSide('');
  if (l3 && l3Version === meta.version) return paintL3();
  $('#main').innerHTML = '<h1 class="plain">L3 초안</h1><p class="muted">코드를 읽는 중이다</p>';
  if (l3Loading) return;
  l3Loading = true;
  const version = meta.version;
  fetch('/api/l3', { cache: 'no-store' })
    .then(r => r.json().then(j => (r.ok ? j : { error: j.error || `요청이 거부되었다 (${r.status})` })))
    .catch(() => ({ error: '서버에 연결할 수 없다' }))
    .then(j => { l3 = j; l3Version = version; l3Loading = false; if (view === 'l3') renderL3(); });
}
const pct = (a, b) => b ? Math.floor(a * 100 / b) + '%' : '-';
function paintL3() {
  const d = l3;
  if (d.error) { $('#main').innerHTML = `<h1 class="plain">L3 초안</h1><p class="sub">${esc(d.error)}</p>`; return; }
  const kinds = {};
  for (const n of d.nodes || []) kinds[n.kind] = (kinds[n.kind] || 0) + 1;
  const count = k => kinds[k] || 0;
  const skipped = (d.skipped || []).map(x => `<li><span class="id">${esc(x.name)}</span> <span class="muted">${esc(x.where)}</span> ${esc(x.reason)}</li>`).join('');
  const has = (d.nodes || []).length > 0;
  $('#main').innerHTML = `<h1 class="plain">L3 초안</h1>
<p class="sub">설계 모듈 패키지에 손으로 쓴 선언 중 설계에 없는 것을 gen: false 노드로 옮겼다. 설계에 더해도 검사를 통과하는 노드만 담는다.</p>
<dl class="facts"><dt>에러</dt><dd>${count('errors')}</dd><dt>상수</dt><dd>${count('consts')}</dd><dt>enum</dt><dd>${count('enums')}</dd><dt>타입</dt><dd>${count('types')}</dd><dt>계약</dt><dd>${count('contracts')}</dd><dt>함수</dt><dd>${count('funcs')}</dd><dt>옮기지 않은 선언</dt><dd>${(d.skipped || []).length}</dd><dt>설계 에러를 반환하지 않는 함수</dt><dd>${d.funcs}</dd><dt>coverage</dt><dd>${pct(d.coverage.before, d.coverage.decls)} → ${pct(d.coverage.after, d.coverage.decls)} <span class="muted">선언 ${d.coverage.decls}개 중 ${d.coverage.before} → ${d.coverage.after}</span></dd></dl>
${has ? `<div class="draftbar">${d.canSave ? '<button type="button" class="primary" id="savel3">design/에 새 파일로 저장</button>' : '<span class="muted">루프백 주소로 띄웠을 때만 저장할 수 있다</span>'}<button type="button" id="copyl3">YAML 복사</button></div>
<p id="savemsg" class="muted"></p>
<pre class="yaml">${esc(d.yaml)}</pre>` : '<p class="muted">설계에 올릴 선언이 없다.</p>'}
${skipped ? `<details><summary>옮기지 않은 선언 ${(d.skipped || []).length}개</summary><ul>${skipped}</ul></details>` : ''}`;
}
async function saveL3() {
  const btn = $('#savel3'); if (btn) btn.disabled = true;
  const res = await post('/api/l3', {});
  if (!res.ok) { $('#savemsg').textContent = res.error; if (btn) btn.disabled = false; return; }
  toast(`${res.path}에 노드 ${res.nodes}개를 썼다`);
  l3 = null;
  await load();
}

/* ---------- 생성 미리보기 ---------- */
// daemok gen이 할 일을 보인다. 쓰지 않는다. 서버가 설계 대조를 다시 돌리므로 탭을 열 때만 받는다.
let gen = null, genVersion = -1, genLoading = false, genSel = '', genText = null;
const GEN_LABEL = { new: '새 파일', changed: '바뀜', delete: '지울 파일', refuse: '쓰지 않음', same: '그대로' };
function renderGen() {
  setSide('');
  if (gen && genVersion === meta.version) return paintGen();
  $('#main').innerHTML = '<h1 class="plain">생성</h1><p class="muted">생성하는 중이다</p>';
  if (genLoading) return;
  genLoading = true;
  const version = meta.version;
  fetch('/api/gen', { cache: 'no-store' })
    .then(r => r.json().then(j => (r.ok ? j : { error: j.error || `요청이 거부되었다 (${r.status})` })))
    .catch(() => ({ error: '서버에 연결할 수 없다' }))
    .then(j => { gen = j; genVersion = version; genLoading = false; genText = null; if (view === 'gen') renderGen(); });
}
function paintGen() {
  const d = gen;
  if (d.error) { $('#main').innerHTML = `<h1 class="plain">생성</h1><p class="sub">${esc(d.error)}</p>`; return; }
  const files = d.files || [];
  const n = st => files.filter(f => f.status === st).length;
  const row = f => {
    const name = f.status === 'delete' || f.status === 'refuse'
      ? `<code>${esc(f.path)}</code>`
      : `<button type="button" class="linklike${f.path === genSel ? ' on' : ''}" data-genfile="${esc(f.path)}">${esc(f.path)}</button>`;
    return `<li>${name} <span class="muted">${GEN_LABEL[f.status]}${f.reason ? ': ' + esc(f.reason) : ''}</span></li>`;
  };
  const todo = files.filter(f => f.status !== 'same'), same = files.filter(f => f.status === 'same');
  $('#main').innerHTML = `<h1 class="plain">생성</h1>
<p class="sub">daemok gen이 쓰고 지울 파일이다. 이 화면은 쓰지 않는다.</p>
<p>${['new', 'changed', 'delete', 'refuse', 'same'].filter(st => n(st)).map(st => `${GEN_LABEL[st]} ${n(st)}`).join(' · ') || '생성 파일이 없다'}</p>
${todo.length ? `<ul class="genlist">${todo.map(row).join('')}</ul>` : '<p class="muted">생성 파일이 설계와 같다.</p>'}
${same.length ? `<details${todo.length ? '' : ' open'}><summary>그대로인 파일 ${same.length}개</summary><ul class="genlist">${same.map(row).join('')}</ul></details>` : ''}
${genSel ? `<h3 style="margin-top:22px"><code>${esc(genSel)}</code></h3>${genText === null ? '<p class="muted">읽는 중이다</p>' : `<pre class="yaml">${esc(genText)}</pre>`}` : ''}`;
}
async function showGenFile(path) {
  genSel = path; genText = null; paintGen();
  const r = await fetch('/api/gen/file?path=' + encodeURIComponent(path), { cache: 'no-store' }).catch(() => null);
  genText = r && r.ok ? await r.text() : '내용을 받을 수 없다';
  if (view === 'gen' && genSel === path) paintGen();
}
