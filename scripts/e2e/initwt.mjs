// 초안 worktree e2e: 분석 모드 초안 탭에서 커밋 안 된 변경 경고를 보고, 새 worktree에 저장하면 그 worktree가 대조 모드로 열리는지 본다.
// 사용: node scripts/e2e/initwt.mjs <serve URL>. 저장소는 TestUIInitWorktreeE2E의 것을 전제로 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Page.navigate', { url: url + '#draft' });
await sleep(1500);

const steps = [
  ['dirty_warning', `(() => { const m = document.querySelector('#main').textContent; return S.mode === 'analyze' && m.includes('커밋 안 된 변경 1개는 새 worktree에 들어가지 않는다') && document.querySelector('#wtbranch').value === 'daemok/design' ? 'ok' : m.slice(-400); })()`],
  ['bad_branch', `(async () => { document.querySelector('#wtbranch').value = 'a b'; document.querySelector('#savewt').click(); for (let i = 0; i < 40; i++) { await new Promise(r => setTimeout(r, 100)); const t = document.querySelector('#savemsg').textContent; if (t) return t.includes('브랜치 이름이 아니다') && S.mode === 'analyze' ? 'ok' : t; } return '메시지 없음'; })()`],
  ['save', `(async () => { document.querySelector('#wtbranch').value = 'try/one'; document.querySelector('#savewt').click(); for (let i = 0; i < 60; i++) { await new Promise(r => setTimeout(r, 100)); if (S.mode === 'check') { const t = document.querySelector('#projbtn').textContent; return t.endsWith(' · try/one') && S.root.endsWith('/.worktrees/try-one') && view === 'map' ? 'ok' : t + ' ' + S.root + ' ' + view; } } return S.mode; })()`],
];
await runSteps(b, steps, 200);
