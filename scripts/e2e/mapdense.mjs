// 분석 모드 지도 e2e: 관계선이 많으면 처음에는 숨는지, 올리면 보이는지, 긴 이름을 가운데 줄여 상자 안에 두는지,
// 모듈 탭 왼쪽 목록이 같은 영역으로 묶이고 하위 패키지가 부모 아래 들어가며 이름이 잘리지 않는지, 머리글과 사실 칸을 본다.
// 띠가 디렉토리 영역이고 띠 이름이 상자에 닿지 않는지, 하위 패키지가 부모 바로 뒤에 오는지 본다.
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
  ['many_hidden', `(() => { const o = [...document.querySelectorAll('path.edge')].map(e => +getComputedStyle(e).opacity); const note = document.querySelector('#mappanel').textContent; return o.length > 60 && o.every(x => x === 0) && note.includes('올렸을 때만 보인다') ? 'ok' : '관계선 ' + o.length + ', opacity 최대 ' + Math.max(...o) + ', 안내 ' + note.includes('올렸을 때만 보인다'); })()`],
  ['hover_shows', `(async () => { ${unit('reg')}.dispatchEvent(new MouseEvent('mouseenter')); await new Promise(r => setTimeout(r, 300)); const o = [...document.querySelectorAll('path.edge[data-from="reg"]')].map(e => +getComputedStyle(e).opacity); ${unit('reg')}.dispatchEvent(new MouseEvent('mouseleave')); return o.length && o.every(x => x > .5) ? 'ok' : 'reg 관계선 opacity ' + o; })()`],
  ['long_name', `(() => { const g = ${unit('rulecheckevaluation_service')}; const t = g.querySelector('text.nm'), box = g.querySelector('rect.box'); const tb = t.getBBox(), bb = box.getBBox(); const title = g.querySelector('title')?.textContent; return t.textContent.includes('…') && t.textContent.endsWith('service') && tb.x + tb.width <= bb.x + bb.width - 8 && title === 'rulecheckevaluation_service' ? 'ok' : t.textContent + ' 끝 ' + (tb.x + tb.width) + ' 상자 끝 ' + (bb.x + bb.width) + ' title ' + title; })()`],
  ['area_bands', `(() => { const l = [...document.querySelectorAll('.band text')].map(t => t.textContent); return JSON.stringify(l) === JSON.stringify(['루트', 'box/core', 'library/thirdparty']) ? 'ok' : JSON.stringify(l); })()`],
  ['band_label_fits', `(() => { const left = Math.min(...[...document.querySelectorAll('g.unit rect.box')].map(r => +r.getAttribute('x'))); const over = [...document.querySelectorAll('.band text')].filter(t => { const b = t.getBBox(); return b.x + b.width > left - 6; }).map(t => t.textContent); return over.length ? '상자에 닿는 띠 이름 ' + over : 'ok'; })()`],
  ['children_follow', `(() => { const at = id => { const r = document.querySelector('g.unit[data-id="' + id + '"] rect.box'); return [+r.getAttribute('y'), +r.getAttribute('x')]; }; const ids = ['reg', 'reg_query', 'reg_service', 'shop', 'shop_query', 'rulecheckevaluation_service']; const ps = ids.map(at); const ok = ps.every((p, i) => !i || p[0] > ps[i - 1][0] || (p[0] === ps[i - 1][0] && p[1] > ps[i - 1][1])); return ok ? 'ok' : JSON.stringify(ps); })()`],
  ['sub_relative', `(() => { const s = ${unit('reg_query')}.querySelector('text.sb').textContent; return s === 'reg/query' ? 'ok' : s; })()`],
  ['side_areas', `(() => { go('module', 'reg'); const l = [...document.querySelectorAll('#side h2')].map(h => h.textContent); return JSON.stringify(l) === JSON.stringify(['루트', 'box/core', 'library/thirdparty']) ? 'ok' : JSON.stringify(l); })()`],
  ['side_nested', `(() => { const li = document.querySelector('#side button[data-go="reg"]').closest('li'); const kids = [...li.querySelectorAll(':scope > ul button')].map(x => [x.dataset.go, x.textContent, x.title]); return JSON.stringify(kids) === JSON.stringify([['reg_query', 'query', 'reg_query'], ['reg_service', 'service', 'reg_service']]) ? 'ok' : JSON.stringify(kids); })()`],
  ['side_no_hscroll', `(() => { const s = document.querySelector('#side'); const cut = [...s.querySelectorAll('button')].filter(x => x.scrollWidth > x.clientWidth + 1).map(x => x.dataset.go); return s.scrollWidth <= s.clientWidth && !cut.length ? 'ok' : 'side ' + s.scrollWidth + '/' + s.clientWidth + ' 잘린 이름 ' + cut; })()`],
  ['module_header', `(() => { const t = document.querySelector('#main .sub').textContent; return t.includes('영역 box/core') && !t.includes('tier') ? 'ok' : t; })()`],
  ['module_facts', `(() => { const f = {}; document.querySelectorAll('#main dl.facts dt').forEach(dt => { f[dt.textContent] = dt.nextElementSibling.textContent; }); const ok = (f['하위 패키지'] || '').includes('reg_query') && (f['하위 패키지'] || '').includes('reg_service') && (f['외부 의존'] || '').includes('example.org/ext') && (f['L3 후보'] || '').includes('ErrGone'); return ok ? 'ok' : JSON.stringify(f); })()`],
];
await runSteps(b, steps, 200);
