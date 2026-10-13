// worktree e2e: 분석 모드에서 같은 저장소의 설계 있는 worktree를 머리 링크로 열고, 프로젝트 서랍의 같은 저장소 목록을 본다.
// 사용: node scripts/e2e/worktree.mjs <serve URL>. 저장소는 TestUIWorktreeE2E의 것을 전제로 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Page.navigate', { url });
await sleep(1500);

const until = (cond, what) => `(async () => { for (let i = 0; i < 40; i++) { if (${cond}) return 'ok'; await new Promise(r => setTimeout(r, 100)); } return ${what}; })()`;
const steps = [
  ['hint_shown', until(`S.mode === 'analyze' && !document.querySelector('#wthint').hidden && document.querySelector('#wthint').textContent === '설계 있음: feat/daemok'`, `S.mode + ' ' + document.querySelector('#wthint').outerHTML`)],
  ['hint_opens', `(async () => { document.querySelector('#wthint button').click(); for (let i = 0; i < 40; i++) { await new Promise(r => setTimeout(r, 100)); if (S.mode === 'check' && document.querySelector('#wthint').hidden) return 'ok'; } return S.mode + ' ' + document.querySelector('#wthint').hidden; })()`],
  ['header_names_repo', `(() => { const t = document.querySelector('#projbtn').textContent, repo = S.root.split('/').slice(-3)[0]; return t === repo + ' · feat/daemok' ? 'ok' : t; })()`],
  ['drawer_list', `(async () => {
    openProjects();
    for (let i = 0; i < 40; i++) {
      await new Promise(r => setTimeout(r, 100));
      const items = [...document.querySelectorAll('#worktrees button')];
      if (items.length === 2) {
        const t = items.map(x => [x.textContent.includes('main · 분석'), x.textContent.includes('feat/daemok · 설계 있음 · 지금'), x.disabled]);
        return JSON.stringify(t) === JSON.stringify([[true, false, false], [false, true, true]]) ? 'ok' : JSON.stringify(t);
      }
    }
    return document.querySelector('#drawer').textContent.slice(0, 300);
  })()`],
  ['placeholder', `(() => { const p = document.querySelector('#openpath').placeholder; return p === S.root ? 'ok' : p; })()`],
];
await runSteps(b, steps, 200);
