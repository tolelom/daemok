// 생성 미리보기 e2e: 탭이 파일마다 상태를 보이고, 파일을 누르면 생성할 내용을 보이는지 본다.
// 사용: node scripts/e2e/gen.mjs <serve URL>. 설계와 코드는 TestUIGenE2E의 것을 전제로 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Page.navigate', { url: url + '#gen' });
await sleep(1500);

const main = `document.querySelector('#main').textContent`;
const steps = [
  ['tab_visible', `(() => { const t = document.querySelector('nav.tabs button[data-view="gen"]'); return t && !t.hidden ? 'ok' : '탭이 안 보인다'; })()`],
  ['listed', `(() => ${main}.includes('core/consts.gen.go') && ${main}.includes('새 파일') && ${main}.includes('core/stale.gen.go') && ${main}.includes('지울 파일') ? 'ok' : ${main}.slice(0, 400))()`],
  ['summary', `(() => ${main}.includes('새 파일 1') && ${main}.includes('지울 파일 1') ? 'ok' : ${main}.slice(0, 400))()`],
  ['content', `(async () => { document.querySelector('[data-genfile="core/consts.gen.go"]').click(); for (let i = 0; i < 40; i++) { await new Promise(r => setTimeout(r, 100)); const p = document.querySelector('#main pre.yaml'); if (p && p.textContent.includes('Limit = 3')) return 'ok'; } return ${main}.slice(0, 400); })()`],
  // 파일을 연 채 상태가 갱신되면 그 파일 내용을 다시 받아 보인다.
  ['content_after_refresh', `(async () => { const r = await fetch('/api/edit', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ op: 'add_module', id: 'zz', value: 'a', version: meta.version }) }); if (!r.ok) return 'edit ' + r.status; const v = (await r.json()).version; for (let i = 0; i < 60; i++) { await new Promise(r => setTimeout(r, 100)); const p = document.querySelector('#main pre.yaml'); if (meta.version >= v && genVersion === meta.version && p && p.textContent.includes('Limit = 3')) return 'ok'; } return ${main}.slice(-300); })()`],
  // 다른 탭에 있는 동안 갱신돼도 돌아오면 내용을 보인다.
  ['content_after_refresh_elsewhere', `(async () => { go('map'); if (view !== 'map') return '탭이 바뀌지 않았다: ' + view; const r = await fetch('/api/edit', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ op: 'add_module', id: 'zy', value: 'a', version: meta.version }) }); if (!r.ok) return 'edit ' + r.status; const v = (await r.json()).version; for (let i = 0; i < 40 && meta.version < v; i++) await new Promise(r => setTimeout(r, 100)); go('gen'); for (let i = 0; i < 60; i++) { await new Promise(r => setTimeout(r, 100)); const p = document.querySelector('#main pre.yaml'); if (p && p.textContent.includes('Limit = 3')) return 'ok'; } return ${main}.slice(-300); })()`],
];
await runSteps(b, steps, 300);
