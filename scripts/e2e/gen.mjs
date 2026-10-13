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
];
await runSteps(b, steps, 300);
