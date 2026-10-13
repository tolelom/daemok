// 분석 모드 지도 e2e: 관계선이 많으면 처음에는 숨는지, 올리면 보이는지, 긴 이름을 가운데 줄여 상자 안에 두는지,
// 모듈 탭 왼쪽 목록이 같은 영역으로 묶이고 하위 패키지가 부모 아래 들어가며 이름이 잘리지 않는지, 머리글과 사실 칸을 본다.
// 어떤 관계선도 끝점이 아닌 상자를 지나지 않는지, 같은 줄 관계선이 이웃이면 직선, 건너면 줄 위로 가고, 같은 두 상자 사이 선은 차선을 나누는지,
// 띠가 디렉토리 영역이고 띠 이름이 상자에 닿지 않는지, 하위 패키지가 부모 상자로 접히고 관계선과 찾기도 부모로 모이는지 본다.
// 사용: node scripts/e2e/mapdense.mjs <serve URL>. 코드는 TestUIMapDenseE2E의 것을 전제로 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Emulation.setDeviceMetricsOverride', { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false });
await b.send('Page.navigate', { url });
await sleep(1500);

const unit = (id) => `document.querySelector('g.unit[data-id="${id}"]')`;
const steps = [
  ['many_hidden', `(() => { const o = [...document.querySelectorAll('path.edge')].map(e => +getComputedStyle(e).opacity); const note = document.querySelector('#mappanel').textContent; return o.length > 0 && o.every(x => x === 0) && note.includes('올렸을 때만 보인다') ? 'ok' : '관계선 ' + o.length + ', opacity 최대 ' + Math.max(...o) + ', 안내 ' + note.includes('올렸을 때만 보인다'); })()`],
  ['hover_shows', `(async () => { ${unit('reg')}.dispatchEvent(new MouseEvent('mouseenter')); await new Promise(r => setTimeout(r, 300)); const o = [...document.querySelectorAll('path.edge[data-from="reg"]')].map(e => +getComputedStyle(e).opacity); ${unit('reg')}.dispatchEvent(new MouseEvent('mouseleave')); return o.length && o.every(x => x > .5) ? 'ok' : 'reg 관계선 opacity ' + o; })()`],
  ['long_name', `(() => { const g = ${unit('inventory_reconciliation_worker')}; const t = g.querySelector('text.nm'), box = g.querySelector('rect.box'); const tb = t.getBBox(), bb = box.getBBox(); const title = g.querySelector('title')?.textContent; return t.textContent.includes('…') && t.textContent.endsWith('_worker') && tb.x + tb.width <= bb.x + bb.width - 8 && title === 'inventory_reconciliation_worker' ? 'ok' : t.textContent + ' 끝 ' + (tb.x + tb.width) + ' 상자 끝 ' + (bb.x + bb.width) + ' title ' + title; })()`],
  ['sub_fits', `(() => { const over = [...document.querySelectorAll('g.unit')].filter(g => { const t = g.querySelector('text.sb'), b = g.querySelector('rect.box').getBBox(); return t.getBBox().x + t.getBBox().width > b.x + b.width - 8; }).map(g => g.dataset.id + ':' + g.querySelector('text.sb').textContent); const shop = ${unit('shop')}.querySelector('text.sb').textContent; return !over.length && shop.startsWith('Package shop은') && shop.endsWith('…') ? 'ok' : '넘침 ' + over + ' / shop ' + shop; })()`],
  ['area_bands', `(() => { const l = [...document.querySelectorAll('.band text')].map(t => t.textContent); return JSON.stringify(l) === JSON.stringify(['루트', 'box/core', 'library/thirdparty', '공용']) ? 'ok' : JSON.stringify(l); })()`],
  ['shared', `(() => { const g = ${unit('base')}; const bands = [...document.querySelectorAll('.band')]; const last = bands[bands.length - 1].querySelector('rect').getBBox(); const y = +g.querySelector('rect.box').getAttribute('y'); return g.classList.contains('shared') && y > last.y && y < last.y + last.height ? 'ok' : 'shared ' + g.classList.contains('shared') + ', y ' + y + ', 띠 ' + last.y; })()`],
  ['shared_edges', `(async () => { const op = sel => [...document.querySelectorAll(sel)].map(e => +getComputedStyle(e).opacity); const wait = () => new Promise(r => setTimeout(r, 250)); ${unit('leafa')}.dispatchEvent(new MouseEvent('mouseenter')); await wait(); const fromLeaf = op('path.edge[data-from="leafa"][data-to="base"]'), leafIn = op('path.edge[data-to="leafa"]'); ${unit('leafa')}.dispatchEvent(new MouseEvent('mouseleave')); ${unit('base')}.dispatchEvent(new MouseEvent('mouseenter')); await wait(); const toBase = op('path.edge[data-to="base"]'); ${unit('base')}.dispatchEvent(new MouseEvent('mouseleave')); return fromLeaf.length === 1 && fromLeaf[0] === 0 && leafIn.every(x => x > .5) && toBase.length === 13 && toBase.every(x => x > .5) ? 'ok' : 'leafa→base ' + fromLeaf + ', leafa로 ' + leafIn + ', base로 ' + toBase.length + ' ' + toBase; })()`],
  ['band_label_fits', `(() => { const left = Math.min(...[...document.querySelectorAll('g.unit rect.box')].map(r => +r.getAttribute('x'))); const over = [...document.querySelectorAll('.band text')].filter(t => { const b = t.getBBox(); return b.x + b.width > left - 6; }).map(t => t.textContent); return over.length ? '상자에 닿는 띠 이름 ' + over : 'ok'; })()`],
  ['folded', `(() => { const kids = ['reg_query', 'reg_service', 'shop_query', 'rulecheckevaluation_service'].filter(id => document.querySelector('g.unit[data-id="' + id + '"]')); const sub = ${unit('reg')}.querySelector('text.sb').textContent, stack = !!${unit('reg')}.querySelector('rect.stack'); return !kids.length && sub === '하위 패키지 2' && stack ? 'ok' : '그려진 하위 ' + kids + ', 부제 ' + sub + ', 겹 ' + stack; })()`],
  ['stack_see_through', `(() => { const g = ${unit('reg')}; const b = g.querySelector('rect.box').getBoundingClientRect(); const top = document.elementFromPoint(b.left + b.width / 2, b.top - 2.5), side = document.elementFromPoint(b.right + 2.5, b.top + b.height / 2); const bad = [top, side].filter(el => el && el.classList.contains('stack')).length; return bad ? '겹 상자가 띠를 덮음 ' + bad : 'ok'; })()`],
  ['folded_edges', `(() => { const from = [...document.querySelectorAll('path.edge')].map(e => e.dataset.from + '>' + e.dataset.to); const reg = from.filter(x => x.startsWith('reg>leaf')).length, stray = from.filter(x => /_query|_service/.test(x)); return reg === 13 && !stray.length ? 'ok' : 'reg 선 ' + reg + ', 하위 선 ' + stray; })()`],
  ['search_child', `(async () => { const q = document.querySelector('#mapq'); q.value = 'query'; q.dispatchEvent(new Event('input')); await new Promise(r => setTimeout(r, 100)); const hits = [...document.querySelectorAll('g.unit.hit')].map(g => g.dataset.id).sort(); q.value = ''; q.dispatchEvent(new Event('input')); return JSON.stringify(hits) === JSON.stringify(['reg', 'shop']) ? 'ok' : JSON.stringify(hits); })()`],
  ['row_edges', `(() => {
    const path = (f, t) => [...document.querySelectorAll('path.edge')].filter(e => e.dataset.from === f && e.dataset.to === t);
    const box = id => document.querySelector('g.unit[data-id="' + id + '"] rect.box').getBBox();
    const inside = (p, b) => p.x > b.x + 1 && p.x < b.x + b.width - 1 && p.y > b.y + 1 && p.y < b.y + b.height - 1;
    const pts = el => { const n = el.getTotalLength(); return Array.from({ length: 41 }, (_, i) => el.getPointAtLength(n * i / 40)); };
    const out = [];
    const [sr] = path('shop', 'reg'), [rs] = path('reg', 'shop');
    if (!sr || !rs) return '같은 줄 선 없음 ' + !!sr + ' ' + !!rs;
    const a = pts(sr), b = pts(rs);
    if (a.some(p => Math.abs(p.y - a[0].y) > .5)) out.push('이웃 칸 선이 직선이 아님');
    if (Math.abs(a[0].y - b[0].y) < 4) out.push('양방향 선이 겹침');
    const [ws] = path('inventory_reconciliation_worker', 'shop');
    const w = pts(ws), reg = box('reg'), top = Math.min(...w.map(p => p.y));
    if (w.some(p => inside(p, reg))) out.push('건너는 선이 reg 상자를 지남');
    if (top >= reg.y) out.push('건너는 선이 줄 위로 가지 않음');
    if (top < reg.y - 22) out.push('건너는 선이 윗줄 여백을 넘음 ' + top + ' ' + reg.y);
    const ys = edgeLanes([{ from: 'a', to: 'b', kind: 'depends' }, { from: 'a', to: 'b', kind: 'register' }, { from: 'b', to: 'a', kind: 'depends' }]);
    if (new Set(ys).size !== 3) out.push('같은 두 상자 사이 선 차선 ' + ys);
    return out.length ? out.join(', ') : 'ok';
  })()`],
  ['no_edge_through_box', `(() => {
    const boxes = [...document.querySelectorAll('g.unit')].map(g => [g.dataset.id, g.querySelector('rect.box').getBBox()]);
    const inside = (p, b) => p.x > b.x + 1 && p.x < b.x + b.width - 1 && p.y > b.y + 1 && p.y < b.y + b.height - 1;
    const bad = [];
    for (const el of document.querySelectorAll('path.edge')) {
      const n = el.getTotalLength();
      for (let i = 0; i <= 80; i++) {
        const p = el.getPointAtLength(n * i / 80);
        const hit = boxes.find(([id, b]) => id !== el.dataset.from && id !== el.dataset.to && inside(p, b));
        if (hit) { bad.push(el.dataset.from + '>' + el.dataset.to + ' 지남 ' + hit[0]); break; }
      }
    }
    return bad.length ? bad.length + '개: ' + bad.slice(0, 4).join(', ') : 'ok';
  })()`],
  ['side_areas', `(() => { go('module', 'reg'); const l = [...document.querySelectorAll('#side h2')].map(h => h.textContent); return JSON.stringify(l) === JSON.stringify(['루트', 'box/core', 'library/thirdparty', '공용']) ? 'ok' : JSON.stringify(l); })()`],
  ['side_nested', `(() => { const li = document.querySelector('#side button[data-go="reg"]').closest('li'); const kids = [...li.querySelectorAll(':scope > ul button')].map(x => [x.dataset.go, x.textContent, x.title]); return JSON.stringify(kids) === JSON.stringify([['reg_query', 'query', 'reg_query'], ['reg_service', 'service', 'reg_service']]) ? 'ok' : JSON.stringify(kids); })()`],
  ['side_no_hscroll', `(() => { const s = document.querySelector('#side'); const cut = [...s.querySelectorAll('button')].filter(x => x.scrollWidth > x.clientWidth + 1).map(x => x.dataset.go); return s.scrollWidth <= s.clientWidth && !cut.length ? 'ok' : 'side ' + s.scrollWidth + '/' + s.clientWidth + ' 잘린 이름 ' + cut; })()`],
  ['module_header', `(() => { const t = document.querySelector('#main .sub').textContent; return t.includes('영역 box/core') && !t.includes('tier') ? 'ok' : t; })()`],
  ['module_facts', `(() => { const f = {}; document.querySelectorAll('#main dl.facts dt').forEach(dt => { f[dt.textContent] = dt.nextElementSibling.textContent; }); const ok = (f['하위 패키지'] || '').includes('reg_query') && (f['하위 패키지'] || '').includes('reg_service') && (f['외부 의존'] || '').includes('example.org/ext') && !f['L3 후보']; return ok ? 'ok' : JSON.stringify(f); })()`],
  ['module_l3', `(() => { const t = document.querySelector('#main table.hints'); return t && t.textContent.includes('ErrGone') && t.textContent.includes('ErrBusy') ? 'ok' : 'L3 표 ' + (t && t.textContent); })()`],
  ['conn_sections', `(() => { const l = [...document.querySelectorAll('#main table.conn tr.sec')].map(r => r.textContent.trim()); return JSON.stringify(l) === JSON.stringify(['쓰는 것', '쓰는 쪽']) ? 'ok' : JSON.stringify(l); })()`],
  ['conn_names', `(() => {
    const rows = [...document.querySelectorAll('#main table.conn tr[data-other]')].map(r => [r.dataset.other, r.querySelector('td.side').textContent, r.querySelector('td.what').textContent]);
    const has = (o, side, ...w) => rows.some(([a, b, c]) => a === o && b === side && w.every(x => c.includes(x)));
    const out = [];
    if (!has('shop', '', 'ErrGone', '에러')) out.push('shop → reg ErrGone');
    if (!has('shop_query', 'service', 'Find', '함수', 'Service', '타입')) out.push('shop_query → reg_service Find, Service');
    if (rows.some(([a]) => a === 'reg_query' || a === 'reg_service')) out.push('자기 하위 패키지가 상대로 나옴');
    return out.length ? out.join(', ') + ' ' + JSON.stringify(rows) : 'ok';
  })()`],
  ['conn_bare', `(() => { const r = document.querySelector('#main table.conn tr.bare'); const t = r ? r.textContent : ''; return t.includes('이름 없이 import') && t.includes('leafa') && t.includes('leafm') ? 'ok' : '이름 없는 줄 ' + t; })()`],
  ['draft_ext', `(() => { go('draft'); const r = [...document.querySelectorAll('#main table.ext tbody tr')].map(t => [...t.cells].map(c => c.textContent.trim())); return JSON.stringify(r) === JSON.stringify([['example.org/ext', 'reg']]) ? 'ok' : JSON.stringify(r); })()`],
  ['draft_hints', `(() => { const r = [...document.querySelectorAll('#main table.hints tbody tr')].map(t => t.textContent); return r.length === 1 && r[0].includes('reg') && r[0].includes('ErrGone, ErrBusy') && r[0].includes('box/core/reg/errs.go') ? 'ok' : JSON.stringify(r); })()`],
  ['draft_yaml_folded', `(() => { const d = document.querySelector('#main details.yaml'); return d && !d.open && d.querySelector('pre.yaml') && document.querySelector('#savedraft') ? 'ok' : 'YAML ' + (d ? 'open ' + d.open : '없음'); })()`],
];
await runSteps(b, steps, 200);
