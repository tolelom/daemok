// 프로젝트, 편집, 이동, 서버 상태. 최상위에서 바로 실행하는 코드는 이 파일에만 둔다. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 프로젝트 ---------- */
let autoOpened = false;
async function post(url, body) {
  try {
    const r = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) });
    const j = await r.json().catch(() => ({}));
    return r.ok ? { ok: true, ...j } : { ok: false, error: j.error || `요청이 거부되었다 (${r.status})` };
  } catch (e) { return { ok: false, error: '서버에 연결할 수 없다' }; }
}
function toast(msg) {
  const t = document.createElement('div');
  t.className = 'toast'; t.setAttribute('role', 'status'); t.textContent = msg;
  document.body.appendChild(t);
  setTimeout(() => t.remove(), 4000);
}
const MODE_LABEL = { check: '설계 있음', analyze: '분석' };
// projectItem은 프로젝트 목록의 한 줄이다. 경로, 브랜치, 모드, 지금 연 것인지를 적는다.
function projectItem(x, ok) {
  const tags = [x.branch ? `<span class="id">${esc(x.branch)}</span>` : '', MODE_LABEL[x.mode] || '', x.current ? '지금' : ''].filter(Boolean);
  return `<button type="button" data-openpath="${esc(x.path)}" ${ok ? '' : 'disabled'}><span class="id">${esc(x.path)}</span>${tags.length ? `<span class="muted projtags">${tags.join(' · ')}</span>` : ''}</button>`;
}
// worktreeHint는 분석 모드에서 같은 저장소의 다른 worktree에 설계가 있으면 머리에 바로 여는 링크를 둔다.
let hintRoot = null;
async function worktreeHint() {
  const el = $('#wthint');
  if (S.mode !== 'analyze') { el.hidden = true; hintRoot = null; return; }
  if (hintRoot === S.root) return;
  hintRoot = S.root;
  let p = { worktrees: [], canOpen: false };
  try { p = await (await fetch('/api/projects', { cache: 'no-store' })).json(); } catch (e) {}
  const hits = p.canOpen ? p.worktrees.filter(x => x.mode === 'check' && !x.current) : [];
  el.innerHTML = hits.map(x => `<button type="button" class="link" data-openpath="${esc(x.path)}" title="${esc(x.path)}">설계 있음: ${esc(x.branch || x.path.split(/[\\/]/).pop())}</button>`).join(' ');
  el.hidden = !hits.length;
}
async function openProjects() {
  drawerOpen = '#projects';
  let p = { worktrees: [], recent: [], workspace: [], canOpen: false };
  try { p = await (await fetch('/api/projects', { cache: 'no-store' })).json(); } catch (e) {}
  const form = p.canOpen
    ? `<form id="openform"><input id="openpath" type="text" placeholder="${esc(p.current || '~/src/project')}" autocomplete="off" spellcheck="false" aria-label="프로젝트 경로"><button type="submit">열기</button></form><p id="openerr"></p>`
    : `<p class="muted">루프백이 아닌 주소로 띄워서 프로젝트를 열 수 없다</p>`;
  $('#drawer').innerHTML = `<div class="scrim" data-close="1"></div><aside class="drawer" role="dialog" aria-label="프로젝트 열기"><div class="head"><h2>프로젝트 열기</h2><button type="button" class="close" data-close="1">닫기</button></div>
${form}
${p.worktrees.length ? `<div><h3>같은 저장소</h3><div class="issuelist" id="worktrees">${p.worktrees.map(x => projectItem(x, x.exists && !x.current && p.canOpen)).join('')}</div></div>` : ''}
<div><h3>최근</h3><div class="issuelist">${p.recent.map(x => projectItem(x, x.exists && p.canOpen)).join('') || '<span class="muted">없음</span>'}</div></div>
${p.workspace.length ? `<div><h3>작업 폴더</h3><div class="issuelist">${p.workspace.map(x => projectItem({ path: x }, p.canOpen)).join('')}</div></div>` : ''}
</aside>`;
  ($('#openpath') || $('#drawer .close')).focus();
}
async function openProject(path) {
  const res = await post('/api/open', { path });
  if (!res.ok) { const e = $('#openerr'); if (e) e.textContent = res.error; else toast(res.error); return; }
  closeDrawer();
  view = 'map'; current = ''; flowSel = ''; flowCompare = ''; failAt = ''; mapFocus = '';
  history.replaceState(null, '', '#map');
  await load();
}
async function saveWorktree() {
  const btn = $('#savewt'); if (btn) btn.disabled = true;
  const res = await post('/api/init-worktree', { branch: $('#wtbranch').value });
  if (!res.ok) { $('#savemsg').textContent = res.error; if (btn) btn.disabled = false; return; }
  toast(`새 worktree에 ${res.path}을 만들었다`);
  view = 'map'; current = ''; mapFocus = '';
  history.replaceState(null, '', '#map');
  await load();
}
async function saveDraft() {
  const btn = $('#savedraft'); if (btn) btn.disabled = true;
  const res = await post('/api/init', {});
  if (!res.ok) { $('#savemsg').textContent = res.error; if (btn) btn.disabled = false; return; }
  toast(`${res.path}을 만들었다`);
  await load();
}

/* ---------- 편집 ---------- */
let editing = false;  // 앞 편집이 끝나기 전의 클릭은 버린다
async function editOp(op, id, value) {
  if (editing) return false;
  editing = true;
  try {
    const res = await post('/api/edit', { op, id, value, version: meta.version });
    if (!res.ok) { toast(res.error); render(); return false; }
    await load();
    return true;
  } finally { editing = false; }
}

/* ---------- 이동 ---------- */
const VIEWS = { check: ['map', 'module', 'flow', 'reqs', 'l3', 'gen'], analyze: ['map', 'module', 'draft'], none: [] };
function render() {
  if (!(VIEWS[S.mode] || VIEWS.check).includes(view)) view = 'map';
  renderHeader();
  if (S.mode === 'none') { setSide(''); $('#main').innerHTML = `<p class="muted">열 프로젝트를 고른다. 설계 파일(design/*.yaml)이나 go.mod가 있는 폴더를 연다.</p>`; return; }
  ({ map: renderMap, module: renderModule, flow: renderFlow, reqs: renderReqs, draft: renderDraft, l3: renderL3, gen: renderGen })[view]();
}
function go(v, arg) {
  view = v;
  if (v === 'module' && arg) current = arg;
  if (v === 'flow' && arg) { flowSel = arg; failAt = ''; }
  closeDrawer();
  const hash = v === 'module' ? `#module/${current}` : v === 'flow' ? `#flow/${flowSel}` : '#' + v;
  if (location.hash !== hash) history.replaceState(null, '', hash);
  render();
  window.scrollTo(0, 0);
}
const toggleFail = id => { failAt = failAt === id ? '' : id; render(); };
document.addEventListener('click', e => {
  const t = e.target;
  if (t.closest('#projbtn')) return openProjects();
  if (t.closest('#changesbtn')) return openChanges();
  const rd = t.closest('[data-rmdep]'); if (rd) return editOp('remove_dep', current, rd.dataset.rmdep);
  // 모듈 지우기는 두 번 눌러야 한다. 다른 곳의 참조는 남고 설계 검사가 없는 참조로 알린다.
  const rm = t.closest('[data-rmmod]');
  if (rm) {
    if (rm.dataset.armed !== '1') { rm.dataset.armed = '1'; rm.textContent = '한 번 더 누르면 지운다(참조는 남는다)'; return; }
    return editOp('remove_module', rm.dataset.rmmod).then(ok => { if (ok) go('map'); });
  }
  const cp = t.closest('[data-copy]'); if (cp) { navigator.clipboard.writeText(cp.dataset.copy).then(() => toast('경로를 복사했다'), () => toast('복사할 수 없다')); return; }
  const rf = t.closest('[data-reqf]'); if (rf) { reqFilter = rf.dataset.reqf; return render(); }
  const op = t.closest('[data-openpath]'); if (op) return openProject(op.dataset.openpath);
  if (t.closest('#savedraft')) return saveDraft();
  if (t.closest('#savewt')) return saveWorktree();
  if (t.closest('#savel3')) return saveL3();
  const gf = t.closest('[data-genfile]'); if (gf) return showGenFile(gf.dataset.genfile);
  if (t.closest('#copyl3')) { navigator.clipboard.writeText(l3.yaml).then(() => toast('YAML을 복사했다'), () => toast('복사할 수 없다')); return; }
  if (t.closest('#copycmd')) { navigator.clipboard.writeText(S.draft.command).then(() => toast('명령을 복사했다'), () => toast('복사할 수 없다')); return; }
  const tab = t.closest('[data-view]'); if (tab) return go(tab.dataset.view);
  if (t.closest('#issuesbtn')) return openIssues();
  if (t.closest('[data-close]')) return closeDrawer();
  const g = t.closest('[data-go]'); if (g) return go('module', g.dataset.go);
  const o = t.closest('[data-open]'); if (o) return openDrawer(o.dataset.open);
  const fl = t.closest('[data-flow]'); if (fl) return go('flow', fl.dataset.flow);
  const fp = t.closest('[data-flowpick]'); if (fp) { flowSel = fp.dataset.flowpick; failAt = ''; return go('flow'); }
  if (t.closest('[data-clearfail]')) { failAt = ''; return render(); }
  const fa = t.closest('[data-fail]'); if (fa) return toggleFail(fa.dataset.fail);
});
document.addEventListener('submit', async e => {
  if (e.target.id === 'openform') { e.preventDefault(); openProject($('#openpath').value.trim()); }
  if (e.target.id === 'addmod') {
    e.preventDefault();
    const id = $('#addmodid').value.trim();
    if (id && await editOp('add_module', id, $('#addmodtier').value)) go('module', id);
  }
});
document.addEventListener('change', e => {
  const t = e.target;
  if (t.matches('select[data-edit="tier"]')) editOp('tier', t.dataset.id, t.value);
  if (t.matches('select[data-adddep]') && t.value) editOp('add_dep', t.dataset.adddep, t.value);
});
document.addEventListener('keydown', e => {
  const typing = /^(INPUT|TEXTAREA|SELECT)$/.test(document.activeElement.tagName);
  // 숫자 키는 보이는 탭을 왼쪽부터 고른다.
  if (!typing && !drawerOpen && !e.ctrlKey && !e.metaKey && !e.altKey && /^[1-9]$/.test(e.key)) {
    const tabs = [...document.querySelectorAll('nav.tabs button')].filter(b => !b.hidden);
    const b = tabs[+e.key - 1]; if (b) { e.preventDefault(); go(b.dataset.view); }
  }
  if (e.key === '/' && !drawerOpen && !typing) {
    if (view !== 'map') go('map');
    const q = $('#mapq'); if (q) { e.preventDefault(); q.focus(); }
  }
  if (e.key === 'Escape' && drawerOpen) closeDrawer();
});

/* ---------- 서버 상태 ---------- */
async function load() {
  let st;
  try { st = await (await fetch('/api/state', { cache: 'no-store' })).json(); }
  catch (e) { $('#status').textContent = '서버에 연결할 수 없다. daemok serve가 떠 있는지 확인한다'; $('#status').className = 'bad'; return; }
  S = { ...st, issues: st.issues || [], files: st.files || [], relations: st.relations || [] };
  meta = { version: st.version, hasBase: !!st.hasBase };
  M = toMaps(st.model);
  B = st.hasBase ? toMaps(st.base) : M;
  render();
  worktreeHint();
  if (drawerOpen === '#issues') openIssues(); else if (drawerOpen === '#changes') openChanges(); else if (drawerOpen && drawerOpen[0] !== '#') openDrawer(drawerOpen);
  if (S.mode === 'none' && !autoOpened) { autoOpened = true; openProjects(); }
}
{
  const [v, arg] = location.hash.slice(1).split('/');
  if (['map', 'module', 'flow', 'reqs', 'draft', 'l3', 'gen'].includes(v)) { view = v; if (v === 'module') current = decodeURIComponent(arg || ''); if (v === 'flow') flowSel = decodeURIComponent(arg || ''); }
}
const es = new EventSource('/api/events');
es.onmessage = e => { if (+e.data !== meta.version) load(); };
load();
