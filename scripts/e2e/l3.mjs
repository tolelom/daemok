// L3 초안 e2e: 탭이 초안과 옮기지 않은 선언을 보여 주고, 저장하면 빈 초안이 되는지 본다.
// 사용: node scripts/e2e/l3.mjs <serve URL>. 설계와 코드는 TestUIL3E2E의 것을 전제로 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Page.navigate', { url: url + '#l3' });
await sleep(1500);

const main = `document.querySelector('#main').textContent`;
const steps = [
  ['tab_visible', `(() => { const t = document.querySelector('nav.tabs button[data-view="l3"]'); return t && !t.hidden ? 'ok' : '탭이 안 보인다'; })()`],
  ['draft_shown', `(() => { const y = document.querySelector('#main pre.yaml'); return y && y.textContent.includes('limit: { in: core, gen: false, summary: "Limit is the cap."') ? 'ok' : ${main}.slice(0, 300); })()`],
  ['skipped_listed', `(() => ${main}.includes('옮기지 않은 선언 1개') && ${main}.includes('Twice') ? 'ok' : ${main}.slice(0, 300))()`],
  ['save', `(async () => { document.querySelector('#savel3').click(); for (let i = 0; i < 40; i++) { await new Promise(r => setTimeout(r, 100)); if (${main}.includes('설계에 올릴 선언이 없다')) return 'ok'; } return '저장 뒤: ' + ${main}.slice(0, 300); })()`],
];
await runSteps(b, steps, 300);
