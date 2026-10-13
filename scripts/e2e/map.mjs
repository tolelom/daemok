// 지도 e2e: 관계선이 처음부터 흐리게 보이는지, 감춘 종류는 안 보이는지, tier 안 배치가 선 교차를 줄이는지,
// 올린 모듈과 이어진 상자 부제가 관계(사용, 제공, 발행, 구독, 등록)로 바뀌었다 돌아오는지,
// 모듈 탭의 같은 진단이 묶이는지, 시스템 설정이 라이트면 밝은 바탕인지 본다.
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

// 바탕 밝기(0~1)를 시스템 색 설정마다 잰다.
const brightness = `(() => { const c = getComputedStyle(document.body).backgroundColor.match(/\\d+/g).map(Number); return (c[0] * .299 + c[1] * .587 + c[2] * .114) / 255; })()`;
const scheme = (value) => b.send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value }] });
await scheme('light');
const light = await b.run(brightness);
await scheme('dark');
const dark = await b.run(brightness);
const theme = dark < .3 && light > .8 ? 'ok' : `바탕 밝기 다크 ${dark}, 라이트 ${light}`;

const opacity = (sel) => `getComputedStyle(document.querySelector('${sel}')).opacity`;
const steps = [
  ['theme', `'${theme}'`],
  ['updated_label', `(() => { const t = document.querySelector('#status').textContent; return t.includes('갱신 ') ? 'ok' : '머리 상태: ' + t; })()`],
  ['faint_edges', `(() => { const o = +${opacity('path.edge')}; return o > 0 && o < .5 ? 'ok' : '처음 관계선 opacity ' + o; })()`],
  ['no_labels', `(() => { const l = [...document.querySelectorAll('text.edge-label')].filter(e => +getComputedStyle(e).opacity > 0); return l.length === 0 ? 'ok' : '처음 보이는 라벨 ' + l.length; })()`],
  ['hidden_kind', `(async () => { const cb = document.querySelector('#mappanel input[data-kind="depends"]'); if (!cb) return '종류 칸 없음'; cb.click(); await new Promise(r => setTimeout(r, 300)); const o = +${opacity('path.edge.k-depends')}; cb.click(); return o === 0 ? 'ok' : '감춘 종류 opacity ' + o; })()`],
  ['no_crossing', `(() => { const x = id => +document.querySelector('g.unit[data-id="' + id + '"] rect.box').getAttribute('x'); return Math.sign(x('x') - x('y')) === Math.sign(x('q') - x('p')) ? 'ok' : 'x ' + x('x') + ', y ' + x('y') + ', q ' + x('q') + ', p ' + x('p'); })()`],
  ['hover_rel', `(async () => { const sb = id => document.querySelector('g.unit[data-id="' + id + '"] text.sb').textContent; const before = sb('x'), beforeP = sb('p'); const g = id => document.querySelector('g.unit[data-id="' + id + '"]'); g('q').dispatchEvent(new MouseEvent('mouseenter')); await new Promise(r => setTimeout(r, 200)); const onQ = [sb('x'), sb('p')]; g('q').dispatchEvent(new MouseEvent('mouseleave')); g('x').dispatchEvent(new MouseEvent('mouseenter')); await new Promise(r => setTimeout(r, 200)); const onX = sb('q'); g('x').dispatchEvent(new MouseEvent('mouseleave')); const after = sb('x'); return onQ[0] === '사용' && onQ[1] === beforeP && onX === '제공' && after === before ? 'ok' : JSON.stringify({ before, onQ, onX, after }); })()`],
  ['rel_text', `(() => { const L = (...x) => new Set(x); const got = [
    relSub([{ kind: 'event', labels: L('A', 'B', 'C') }], [{ kind: 'depends', labels: L() }]),
    relSub([{ kind: 'register', labels: L('hk') }], [{ kind: 'event', labels: L('E') }, { kind: 'call', labels: L('C') }]),
    relSub([], [{ kind: 'register', labels: L('hk') }]),
  ]; const want = ['제공 / 발행 · A, B 외 1', '제공 · call 계약 C / 구독 · E / 등록 · hk', '등록받음 · hk']; return JSON.stringify(got) === JSON.stringify(want) ? 'ok' : JSON.stringify(got); })()`],
  ['hover_no_labels', `(async () => {
    const w = () => new Promise(r => setTimeout(r, 250)), y = document.querySelector('g.unit[data-id="y"]');
    const vis = () => [...document.querySelectorAll('text.edge-label')].filter(e => +getComputedStyle(e).opacity > 0).length;
    y.dispatchEvent(new MouseEvent('mouseenter')); await w();
    const edge = +getComputedStyle(document.querySelector('path.edge[data-from="y"][data-to="x"][data-kind="event"]')).opacity, onHover = vis();
    y.dispatchEvent(new MouseEvent('mouseleave'));
    document.querySelector('#showall').click(); await w(); const onAll = vis(); document.querySelector('#showall').click(); await w();
    return edge > .5 && onHover === 0 && onAll === 1 ? 'ok' : '이벤트 선 ' + edge + ', 올렸을 때 라벨 ' + onHover + ', 모두 보기 라벨 ' + onAll;
  })()`],
  ['module_diag_group', `(() => { go('module', 'p'); const g = document.querySelectorAll('#main details.igroup'); return g.length === 1 && g[0].textContent.includes('3개') ? 'ok' : '묶음 ' + g.length + ': ' + document.querySelector('#main').textContent.slice(0, 200); })()`],
  ['side_breaks_at_underscore', `(() => {
    go('module', 'x');
    const btn = document.querySelector('#side button[data-go="inventory_reconciliation_query"]');
    if (!btn) return '목록에 없음';
    const r = document.createRange(), chars = [];
    const walk = n => { if (n.nodeType === 3) { for (let i = 0; i < n.length; i++) { r.setStart(n, i); r.setEnd(n, i + 1); const b = r.getBoundingClientRect(); if (b.width) chars.push([n.data[i], b.top]); } } else n.childNodes.forEach(walk); };
    walk(btn.firstChild.nodeType === 3 ? btn : btn);
    const bad = chars.filter((c, i) => i && c[1] > chars[i - 1][1] + 2 && chars[i - 1][0] !== '_').map((c, i) => c[0]);
    return bad.length ? '밑줄이 아닌 곳에서 줄바꿈 ' + JSON.stringify(chars.map(c => c[0]).join('')) : 'ok';
  })()`],
  ['conn_design', `(() => { go('module', 'x'); const sec = [...document.querySelectorAll('#main table.conn tr.sec')].map(r => r.textContent.trim()); const bare = document.querySelector('#main table.conn tr.bare')?.textContent.trim(); return JSON.stringify(sec) === '["쓰는 것","구독"]' && bare === 'q' ? 'ok' : JSON.stringify([sec, bare]); })()`],
  ['conn_words', `(() => { const t = connSections('m', [{ from: 'm', to: 'a', kind: 'event', label: 'E' }, { from: 'b', to: 'm', kind: 'event', label: 'F' }, { from: 'm', to: 'c', kind: 'depends' }, { from: 'd', to: 'm', kind: 'register', label: 'hk' }]).map(s => s.title + ':' + s.rows.map(r => r.other + '=' + r.what.join('|')).join(',')); const want = ['쓰는 것:c=', '발행:a=E', '구독:b=F', '등록받음:d=hk']; return JSON.stringify(t) === JSON.stringify(want) ? 'ok' : JSON.stringify(t); })()`],
];
await runSteps(b, steps, 300);
