// 지도 e2e: 관계선이 처음부터 흐리게 보이는지, 감춘 종류는 안 보이는지, tier 안 배치가 선 교차를 줄이는지,
// 모듈 탭의 같은 진단이 묶이는지 본다.
// 사용: node scripts/e2e/map.mjs <serve URL>. 설계는 TestUIMapE2E의 것을 전제로 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Page.navigate', { url });
await sleep(1500);

const opacity = (sel) => `getComputedStyle(document.querySelector('${sel}')).opacity`;
const steps = [
  ['faint_edges', `(() => { const o = +${opacity('path.edge')}; return o > 0 && o < .5 ? 'ok' : '처음 관계선 opacity ' + o; })()`],
  ['no_labels', `(() => { const l = [...document.querySelectorAll('text.edge-label')].filter(e => +getComputedStyle(e).opacity > 0); return l.length === 0 ? 'ok' : '처음 보이는 라벨 ' + l.length; })()`],
  ['hidden_kind', `(async () => { const cb = document.querySelector('#mappanel input[data-kind="depends"]'); if (!cb) return '종류 칸 없음'; cb.click(); await new Promise(r => setTimeout(r, 300)); const o = +${opacity('path.edge.k-depends')}; cb.click(); return o === 0 ? 'ok' : '감춘 종류 opacity ' + o; })()`],
  ['no_crossing', `(() => { const x = id => +document.querySelector('g.unit[data-id="' + id + '"] rect.box').getAttribute('x'); return Math.sign(x('x') - x('y')) === Math.sign(x('q') - x('p')) ? 'ok' : 'x ' + x('x') + ', y ' + x('y') + ', q ' + x('q') + ', p ' + x('p'); })()`],
  ['module_diag_group', `(() => { go('module', 'p'); const g = document.querySelectorAll('#main details.igroup'); return g.length === 1 && g[0].textContent.includes('3개') ? 'ok' : '묶음 ' + g.length + ': ' + document.querySelector('#main').textContent.slice(0, 200); })()`],
];
await runSteps(b, steps, 300);
