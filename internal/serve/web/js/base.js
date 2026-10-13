// 공통 상태와 도우미, 머리. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
const $ = s => document.querySelector(s);
const esc = s => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

const KINDS = {
  depends: { name: '의존', style: '' }, exception: { name: '그룹 예외', style: 'dash' },
  call: { name: 'call 계약', style: '' }, query: { name: 'query', style: '' }, port: { name: 'port', style: '' },
  external: { name: 'external', style: '' }, event: { name: '이벤트', style: 'dash' }, register: { name: '핸들러 등록', style: 'dot' },
};
const SEV = { err: '오류', warn: '경고', info: '정보' };
const ST = { add: '추가', chg: '수정', rem: '삭제' };
const COLS = ['reqs', 'actors', 'ifaces', 'exes', 'mods', 'groups', 'members', 'enums', 'consts', 'errors', 'types', 'contracts', 'hks', 'funcs', 'eks', 'events', 'flows'];
const KEYKIND = { reqs: 'req', mods: 'mod', groups: 'grp', members: 'mem', enums: 'enum', consts: 'const', errors: 'err', types: 'type', contracts: 'contract', hks: 'hk', funcs: 'func', eks: 'ek', events: 'ev', flows: 'flow', actors: 'actor', ifaces: 'iface', exes: 'exe' };

let S = null, M = null, B = null, meta = { version: 0 };
const hiddenKinds = new Set();
let reqFilter = 'all';
let view = 'map', current = '', flowSel = '', flowCompare = '', failAt = '', showAll = false, drawerOpen = null, mapFocus = '';

function toMaps(v) {
  const m = { ...(v || {}) };
  for (const c of COLS) m[c] = Object.fromEntries(((v || {})[c] || []).map(x => [x.id, x]));
  m.tiers = (v || {}).tiers || [];
  return m;
}
const unitOf = id => (M.members[id] ? M.members[id].group : id);
const unitKind = id => M.mods[id] ? 'mod' : M.groups[id] ? 'grp' : M.members[id] ? 'mem' : '';
const unitNode = id => M.mods[id] || M.groups[id] || M.members[id];
const tierOf = id => { const u = unitNode(id); if (!u) return ''; return u.type === 'mem' ? (M.groups[u.group] || {}).tier : u.tier; };
const topUnits = () => [...Object.values(M.mods), ...Object.values(M.groups)];
const tiersTopDown = () => [...M.tiers].reverse().concat(['']);

// 기준(git HEAD) 대비 상태: add, chg, rem, ''
// changeCount는 기준(HEAD) 대비 추가·수정·삭제된 노드 수다.
function changeCount() {
  if (!meta.hasBase) return 0;
  let n = 0;
  for (const c of COLS) for (const id of new Set([...Object.keys(M[c]), ...Object.keys(B[c])])) if (stOf(c, id)) n++;
  return n;
}
function stOf(col, id) {
  if (!meta.hasBase) return '';
  const a = B[col][id], b = M[col][id];
  if (b && !a) return 'add';
  if (a && !b) return 'rem';
  if (a && b && JSON.stringify(a) !== JSON.stringify(b)) return 'chg';
  return '';
}
const unitSt = id => stOf(M.mods[id] || B.mods[id] ? 'mods' : M.groups[id] || B.groups[id] ? 'groups' : 'members', id);
const stMark = st => st ? ` <span class="st ${st}">${ST[st]}</span>` : '';

function issuesFor(keys) { return S.issues.filter(i => keys.includes(i.key)); }
function worst(list) { return list.some(i => i.lv === 'err') ? 'err' : list.some(i => i.lv === 'warn') ? 'warn' : ''; }
function ownedKeys(id) {
  const keys = [unitKind(id) + ':' + id];
  for (const [col, k] of Object.entries(KEYKIND)) for (const n of Object.values(M[col] || {})) if (n.in === id) keys.push(k + ':' + n.id);
  if (M.groups[id]) for (const mid of M.groups[id].members) keys.push(...ownedKeys(mid));
  return keys;
}
const diagList = iss => `<ul class="diag">${iss.map(i => `<li><span class="sev ${i.lv}">${SEV[i.lv]}</span><span>${esc(i.msg)}</span></li>`).join('')}</ul>`;
const inlineList = items => `<span class="list">${items.join('')}</span>`;

function unitLink(id) { return unitNode(id) ? `<button type="button" class="link" data-go="${esc(id)}">${esc(id)}</button>` : `<span class="id">${esc(id)}</span>`; }
function nodeLink(k, id) {
  if (['mod', 'grp', 'mem'].includes(k)) return unitLink(id);
  if (k === 'flow') return `<button type="button" class="link" data-flow="${esc(id)}">${esc(id)}</button>`;
  return `<button type="button" class="link" data-open="${k}:${esc(id)}">${esc(id)}</button>`;
}
function labelLink(kind, label) {
  if (kind === 'event' && M.events[label]) return nodeLink('ev', label);
  if (M.contracts[label]) return nodeLink('contract', label);
  if (kind === 'register' && M.hks[label]) return nodeLink('hk', label);
  return `<span class="id">${esc(label)}</span>`;
}
function legend(kinds) {
  return `<div class="legend">${kinds.map(k => `<span class="k-${k}"><span class="sw ${KINDS[k].style}"></span>${KINDS[k].name}</span>`).join('')}</div>`;
}

/* ---------- 머리 ---------- */
function renderHeader() {
  const t = new Date(S.updated);
  $('#projbtn').textContent = (S.root || '').replace(/[\\/]+$/, '').split(/[\\/]/).pop() || '프로젝트 열기';
  $('#projbtn').title = S.root || '';
  $('#modebadge').hidden = S.mode !== 'analyze';
  const changed = S.mode === 'check' && meta.hasBase ? changeCount() : 0;
  const base = S.mode !== 'check' ? '' : !meta.hasBase ? '기준 없음' : changed ? `<button type="button" class="link chgbtn" id="changesbtn">HEAD 대비 변경 ${changed}개</button>` : 'HEAD와 같음';
  const status = S.mode === 'none' ? [esc(S.problem)] : ['갱신 ' + esc(t.toLocaleTimeString()), S.fatal ? '설계 파일을 읽지 못했다' : '', base];
  $('#status').innerHTML = status.filter(Boolean).join('  ');
  $('#status').className = S.fatal ? 'bad' : '';
  const views = VIEWS[S.mode] || VIEWS.check;
  document.querySelectorAll('nav.tabs button').forEach(b => {
    b.hidden = !views.includes(b.dataset.view);
    b.title = b.hidden ? '' : `단축키 ${views.indexOf(b.dataset.view) + 1}`;
    b.setAttribute('aria-current', String(b.dataset.view === view));
  });
  const c = { err: 0, warn: 0, info: 0 };
  S.issues.forEach(i => c[i.lv]++);
  $('#issuesbtn').innerHTML = [['e', 'err'], ['w', 'warn'], ['n', 'info']].map(([cl, lv]) => `<span class="${cl} ${c[lv] ? 'on' : ''}">${SEV[lv]} ${c[lv]}</span>`).join('');
}
function setSide(html) {
  $('#side').innerHTML = html;
  $('#page').classList.toggle('with-side', !!html);
}
