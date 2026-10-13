// 화면 편집 e2e: headless 브라우저로 모듈 탭에서 tier 바꾸기, 의존 추가·삭제, 지도에서 모듈 추가를 한다.
// 사용: node scripts/e2e/edit.mjs <serve URL>.
// 페이지 예외가 있으면 1로 끝난다. 파일 내용 확인은 이 스크립트를 부르는 Go 테스트가 한다.
import { openBrowser, runSteps, sleep } from './browser.mjs';

const url = process.argv[2];
if (!url) {
  console.error('serve URL이 필요하다');
  process.exit(2);
}
const b = await openBrowser();
await b.send('Page.navigate', { url: url + '#module/api' });
await sleep(1500);

const steps = [
  ['tier', `(() => { const s = document.querySelector('select[data-edit="tier"]'); if (!s) return 'tier 칸 없음'; s.value = 'a'; s.dispatchEvent(new Event('change', { bubbles: true })); return 'ok'; })()`],
  ['add_dep', `(() => { const s = document.querySelector('select[data-adddep]'); if (!s) return '의존 추가 칸 없음'; s.value = 'util'; s.dispatchEvent(new Event('change', { bubbles: true })); return 'ok'; })()`],
  ['remove_dep', `(() => { const b = document.querySelector('[data-rmdep="core"]'); if (!b) return '지우기 버튼 없음'; b.click(); return 'ok'; })()`],
  ['add_module', `(() => { go('map'); const f = document.querySelector('#addmod'); if (!f) return '모듈 추가 칸 없음'; f.querySelector('#addmodid').value = 'web'; f.querySelector('#addmodtier').value = 'b'; f.requestSubmit(); return 'ok'; })()`],
  ['group_tier', `(() => { go('module', 'grp'); const s = document.querySelector('select[data-edit="tier"]'); if (!s) return '그룹 tier 칸 없음'; s.value = 'b'; s.dispatchEvent(new Event('change', { bubbles: true })); return 'ok'; })()`],
  ['remove_once', `(() => { go('module', 'gone'); const x = document.querySelector('[data-rmmod]'); if (!x) return '지우기 버튼 없음'; x.click(); return x.dataset.armed === '1' ? 'ok' : '한 번에 지웠다'; })()`],
  ['remove_twice', `(() => { const x = document.querySelector('[data-rmmod]'); if (!x) return '지우기 버튼 없음'; x.click(); return 'ok'; })()`],
];
await runSteps(b, steps, 1200);
