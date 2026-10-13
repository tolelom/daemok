// 서랍. 여러 classic script가 전역을 함께 쓰며 index.html의 순서대로 읽힌다.
/* ---------- 서랍 ---------- */
function filesFor(key) { return S.files.filter(f => f.sources.includes(key)); }
function codeBlocks(key) { return filesFor(key).map(f => `<details><summary>${esc(f.path)}</summary><pre>${esc(f.code)}</pre></details>`).join(''); }
function fieldsTable(fields) {
  if (!fields.length) return '<p class="muted">필드 없음</p>';
  return `<table><thead><tr><th>필드</th><th>타입</th><th>제약</th></tr></thead><tbody>${fields.map(f => `<tr><td class="id">${esc(f.name)}</td><td class="id">${esc(f.ref ? (f.many ? '[]' : '*') + f.ref : (f.many ? '[]' : '') + f.type)}</td><td>${[f.optional ? 'optional' : '', f.max ? 'max ' + f.max : ''].filter(Boolean).join(', ') || '-'}</td></tr>`).join('')}</tbody></table>`;
}
const factRows = rows => `<dl class="facts" style="margin:0">${rows.map(([t, d]) => `<dt>${t}</dt><dd>${d}</dd>`).join('')}</dl>`;
function drawerBody(key) {
  const [k, ...rest] = key.split(':'), id = rest.join(':');
  const col = Object.keys(KEYKIND).find(c => KEYKIND[c] === k), n = col ? M[col][id] : null;
  if (!n) return null;
  let metaLine = '', body = '';
  if (k === 'contract') {
    metaLine = `${esc(n.kind)} 계약, ${unitLink(n.in)} 모듈`;
    const users = [...new Set(S.relations.filter(r => r.label === id && r.kind === n.kind).map(r => r.from))];
    body = `<table><thead><tr><th>메서드</th><th>파라미터</th><th>반환</th><th>에러</th></tr></thead><tbody>${n.methods.map(me => `<tr><td class="id">${esc(me.name)}</td><td class="id">${me.params.map(p => esc(p.name + ' ' + p.type)).join('<br>') || '-'}</td><td class="id">${esc(me.returns.join(', ') || '-')}</td><td class="id">${esc(me.errors.join(', ') || '-')}</td></tr>`).join('')}</tbody></table>
      ${factRows([['받는 쪽', users.length ? inlineList(users.map(unitLink)) : '<span class="muted">없음</span>']])}`;
  } else if (k === 'ev') {
    const ek = M.eks[n.kind] || {};
    metaLine = `이벤트, 종류 <span class="id">${esc(n.kind)}</span>, ${unitLink(ek.in)} 모듈`;
    const subs = [...new Set([...n.subscribers, ...(ek.subscribers || [])])];
    body = `${factRows([['발행', inlineList(n.publishers.map(unitLink))], ['구독', subs.length ? inlineList(subs.map(unitLink)) : '<span class="muted">없음</span>']])}${fieldsTable(n.data)}`;
  } else if (k === 'ek') {
    metaLine = `이벤트 종류, ${unitLink(n.in)} 모듈, 키 <span class="id">${esc(n.key)}</span>, ${esc(n.encoding)}`;
    body = factRows([['이벤트', inlineList(Object.values(M.events).filter(e => e.kind === id).map(e => nodeLink('ev', e.id)))]]);
  } else if (k === 'type') {
    metaLine = `타입, ${unitLink(n.in)} 모듈`; body = fieldsTable(n.fields);
  } else if (k === 'enum') {
    metaLine = `enum, ${unitLink(n.in)} 모듈`; body = `<p>${inlineList(n.values.map(v => `<span class="id">${esc(v)}</span>`))}</p>`;
  } else if (k === 'err') {
    metaLine = `에러, ${unitLink(n.in)} 모듈`; body = `<p>${esc(n.message)}${n.http ? `  (HTTP ${n.http})` : ''}</p>`;
  } else if (k === 'const') {
    metaLine = `상수, ${unitLink(n.in)} 모듈`; body = `<p class="id">${esc(n.value)} (${esc(n.type)})</p>`;
  } else if (k === 'hk') {
    metaLine = `핸들러 종류, ${unitLink(n.in)} 모듈, 키 <span class="id">${esc(n.key)}</span>`; body = `<p class="id">func(${esc(n.params.map(p => p.name + ' ' + p.type).join(', '))}) ${esc(n.returns.join(', '))}</p>`;
  } else if (k === 'func') {
    metaLine = `함수, ${unitLink(n.in)} 모듈`;
    body = `<table><thead><tr><th>케이스</th><th>주어진 상태</th><th>기대</th></tr></thead><tbody>${n.cases.map(c => `<tr><td>${esc(c.name)}</td><td class="id">${esc(JSON.stringify(c.given))}</td><td class="id">${esc(c.error || JSON.stringify(c.expect))}</td></tr>`).join('')}</tbody></table>`;
  } else metaLine = esc(k);
  const iss = issuesFor([key]);
  const files = codeBlocks(key);
  return `<div class="head"><div><h2>${esc(id)}</h2><div class="meta">${metaLine}</div></div><button type="button" class="close" data-close="1">닫기</button></div>${n.summary ? `<p>${esc(n.summary)}</p>` : ''}${body}${iss.length ? diagList(iss) : ''}${files ? `<div><h3>생성 코드</h3>${files}</div>` : ''}`;
}
function openDrawer(key) {
  const body = drawerBody(key);
  if (!body) return;
  drawerOpen = key;
  $('#drawer').innerHTML = `<div class="scrim" data-close="1"></div><aside class="drawer" role="dialog" aria-label="${esc(key)}">${body}</aside>`;
  $('#drawer .close').focus();
}
function issueRow(i) {
  const target = i.key && (['mod', 'grp', 'mem'].includes(i.key.split(':')[0]) ? `data-go="${esc(i.key.split(':').slice(1).join(':'))}"` : `data-open="${esc(i.key)}"`);
  const where = i.file ? `${i.file}${i.line ? ':' + i.line : ''}` : '';
  const loc = i.abs ? i.abs + (i.line ? ':' + i.line : '') : '';
  // VS Code는 URL 스킴으로 연다. 다른 에디터는 복사한 "경로:줄"을 파일 찾기에 붙여넣는다.
  const acts = loc ? `<span class="iact"><a href="vscode://file/${encodeURI(loc.replace(/\\/g, '/'))}">VS Code</a><button type="button" data-copy="${esc(loc)}">경로 복사</button></span>` : '';
  return `<div class="irow"><button type="button" ${target || 'disabled'}><span class="sev ${i.lv}">${SEV[i.lv]}</span><span>${esc(i.msg)}${where ? `<br><span class="where">${esc(where)}</span>` : ''}</span></button>${acts}</div>`;
}
// 같은 수준·같은 원인(메시지의 설명 부분)이 3개 이상이면 접히는 한 줄로 묶는다.
function issueRows(l) {
  const groups = new Map();
  l.forEach(i => { const k = i.lv + '\u0000' + (i.group || i.msg); if (!groups.has(k)) groups.set(k, []); groups.get(k).push(i); });
  return [...groups.values()].map(g => g.length < 3 ? g.map(issueRow).join('')
    : `<details class="igroup"><summary><span class="sev ${g[0].lv}">${SEV[g[0].lv]}</span><span>${esc(g[0].group)} <span class="muted">${g.length}개</span></span></summary>${g.map(issueRow).join('')}</details>`).join('');
}
function openIssues() {
  drawerOpen = '#issues';
  const sec = (o, title) => { const l = S.issues.filter(i => i.origin === o); return `<div><h3>${title} ${l.length}</h3><div class="issuelist">${issueRows(l) || '<span class="muted">없음</span>'}</div></div>`; };
  $('#drawer').innerHTML = `<div class="scrim" data-close="1"></div><aside class="drawer" role="dialog" aria-label="진단"><div class="head"><h2>진단</h2><button type="button" class="close" data-close="1">닫기</button></div>${sec('design', '설계 검사')}${sec('code', '코드 대조')}</aside>`;
  $('#drawer .close').focus();
}
// openChanges는 기준(HEAD) 대비 추가·수정·삭제된 노드를 상태별로 보인다.
function openChanges() {
  drawerOpen = '#changes';
  const by = { add: [], chg: [], rem: [] };
  for (const c of COLS) for (const id of new Set([...Object.keys(M[c]), ...Object.keys(B[c])])) { const st = stOf(c, id); if (st) by[st].push([c, id]); }
  const label = { reqs: '요구사항', actors: '액터', ifaces: '인터페이스', exes: '실행 단위', mods: '모듈', groups: '그룹', members: '그룹 멤버', enums: 'enum', consts: '상수', errors: '에러', types: '타입', contracts: '계약', hks: '핸들러 종류', funcs: '함수', eks: '이벤트 종류', events: '이벤트', flows: '흐름' };
  const item = (st, [c, id]) => {
    const k = KEYKIND[c];
    const link = st === 'rem' ? `<span class="id">${esc(id)}</span>` : ['mod', 'grp', 'mem'].includes(k) ? unitLink(id) : nodeLink(k, id);
    return `<li><span class="muted">${label[c] || c}</span> ${link}</li>`;
  };
  const sec = st => by[st].length ? `<div><h3><span class="st ${st}">${ST[st]}</span> ${by[st].length}</h3><ul class="chglist">${by[st].map(x => item(st, x)).join('')}</ul></div>` : '';
  $('#drawer').innerHTML = `<div class="scrim" data-close="1"></div><aside class="drawer" role="dialog" aria-label="HEAD 대비 변경"><div class="head"><h2>HEAD 대비 변경</h2><button type="button" class="close" data-close="1">닫기</button></div><p class="muted">작업 중인 설계 파일과 마지막 커밋의 차이다. 커밋하면 사라진다.</p>${sec('add')}${sec('chg')}${sec('rem')}</aside>`;
  $('#drawer .close').focus();
}
function closeDrawer() { drawerOpen = null; $('#drawer').innerHTML = ''; }
