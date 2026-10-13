// 분석 모드 지도 e2e: 관계선이 많으면 처음에는 숨는지, 올리면 보이는지, 긴 이름을 가운데 줄여 상자 안에 두는지,
// 모듈 탭 왼쪽 목록이 같은 영역으로 묶이고 하위 패키지가 부모 아래 들어가며 이름이 잘리지 않는지, 머리글과 사실 칸을 본다.
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
  ['area_bands', `(() => { const l = [...document.querySelectorAll('.band text')].map(t => t.textContent); return JSON.stringify(l) === JSON.stringify(['루트', 'box/core', 'library/thirdparty', '공용']) ? 'ok' : JSON.stringify(l); })()`],
  ['shared', `(() => { const g = ${unit('base')}; const bands = [...document.querySelectorAll('.band')]; const last = bands[bands.length - 1].querySelector('rect').getBBox(); const y = +g.querySelector('rect.box').getAttribute('y'); return g.classList.contains('shared') && y > last.y && y < last.y + last.height ? 'ok' : 'shared ' + g.classList.contains('shared') + ', y ' + y + ', 띠 ' + last.y; })()`],
  ['shared_edges', `(async () => { const op = sel => [...document.querySelectorAll(sel)].map(e => +getComputedStyle(e).opacity); const wait = () => new Promise(r => setTimeout(r, 250)); ${unit('leafa')}.dispatchEvent(new MouseEvent('mouseenter')); await wait(); const fromLeaf = op('path.edge[data-from="leafa"][data-to="base"]'), leafIn = op('path.edge[data-to="leafa"]'); ${unit('leafa')}.dispatchEvent(new MouseEvent('mouseleave')); ${unit('base')}.dispatchEvent(new MouseEvent('mouseenter')); await wait(); const toBase = op('path.edge[data-to="base"]'); ${unit('base')}.dispatchEvent(new MouseEvent('mouseleave')); return fromLeaf.length === 1 && fromLeaf[0] === 0 && leafIn.every(x => x > .5) && toBase.length === 13 && toBase.every(x => x > .5) ? 'ok' : 'leafa→base ' + fromLeaf + ', leafa로 ' + leafIn + ', base로 ' + toBase.length + ' ' + toBase; })()`],
  ['band_label_fits', `(() => { const left = Math.min(...[...document.querySelectorAll('g.unit rect.box')].map(r => +r.getAttribute('x'))); const over = [...document.querySelectorAll('.band text')].filter(t => { const b = t.getBBox(); return b.x + b.width > left - 6; }).map(t => t.textContent); return over.length ? '상자에 닿는 띠 이름 ' + over : 'ok'; })()`],
  ['folded', `(() => { const kids = ['reg_query', 'reg_service', 'shop_query', 'rulecheckevaluation_service'].filter(id => document.querySelector('g.unit[data-id="' + id + '"]')); const sub = ${unit('reg')}.querySelector('text.sb').textContent, stack = !!${unit('reg')}.querySelector('rect.stack'); return !kids.length && sub === '하위 패키지 2' && stack ? 'ok' : '그려진 하위 ' + kids + ', 부제 ' + sub + ', 겹 ' + stack; })()`],
  ['folded_edges', `(() => { const from = [...document.querySelectorAll('path.edge')].map(e => e.dataset.from + '>' + e.dataset.to); const reg = from.filter(x => x.startsWith('reg>')).length, stray = from.filter(x => /_query|_service/.test(x)); return reg === 13 && !stray.length ? 'ok' : 'reg 선 ' + reg + ', 하위 선 ' + stray; })()`],
  ['search_child', `(async () => { const q = document.querySelector('#mapq'); q.value = 'query'; q.dispatchEvent(new Event('input')); await new Promise(r => setTimeout(r, 100)); const hits = [...document.querySelectorAll('g.unit.hit')].map(g => g.dataset.id).sort(); q.value = ''; q.dispatchEvent(new Event('input')); return JSON.stringify(hits) === JSON.stringify(['reg', 'shop']) ? 'ok' : JSON.stringify(hits); })()`],
  ['side_areas', `(() => { go('module', 'reg'); const l = [...document.querySelectorAll('#side h2')].map(h => h.textContent); return JSON.stringify(l) === JSON.stringify(['루트', 'box/core', 'library/thirdparty', '공용']) ? 'ok' : JSON.stringify(l); })()`],
  ['side_nested', `(() => { const li = document.querySelector('#side button[data-go="reg"]').closest('li'); const kids = [...li.querySelectorAll(':scope > ul button')].map(x => [x.dataset.go, x.textContent, x.title]); return JSON.stringify(kids) === JSON.stringify([['reg_query', 'query', 'reg_query'], ['reg_service', 'service', 'reg_service']]) ? 'ok' : JSON.stringify(kids); })()`],
  ['side_no_hscroll', `(() => { const s = document.querySelector('#side'); const cut = [...s.querySelectorAll('button')].filter(x => x.scrollWidth > x.clientWidth + 1).map(x => x.dataset.go); return s.scrollWidth <= s.clientWidth && !cut.length ? 'ok' : 'side ' + s.scrollWidth + '/' + s.clientWidth + ' 잘린 이름 ' + cut; })()`],
  ['module_header', `(() => { const t = document.querySelector('#main .sub').textContent; return t.includes('영역 box/core') && !t.includes('tier') ? 'ok' : t; })()`],
  ['module_facts', `(() => { const f = {}; document.querySelectorAll('#main dl.facts dt').forEach(dt => { f[dt.textContent] = dt.nextElementSibling.textContent; }); const ok = (f['하위 패키지'] || '').includes('reg_query') && (f['하위 패키지'] || '').includes('reg_service') && (f['외부 의존'] || '').includes('example.org/ext') && (f['L3 후보'] || '').includes('ErrGone'); return ok ? 'ok' : JSON.stringify(f); })()`],
];
await runSteps(b, steps, 200);
