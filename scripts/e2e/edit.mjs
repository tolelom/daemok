// 화면 편집 e2e: headless 브라우저로 모듈 탭에서 tier 바꾸기, 의존 추가·삭제, 지도에서 모듈 추가를 한다.
// 사용: node scripts/e2e/edit.mjs <serve URL>. 브라우저는 DAEMOK_BROWSER, 없으면 흔한 설치 경로를 찾는다.
// 페이지 예외가 있으면 1로 끝난다. 파일 내용 확인은 이 스크립트를 부르는 Go 테스트가 한다.
import { spawn } from 'node:child_process';
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const url = process.argv[2];
const candidates = [
  process.env.DAEMOK_BROWSER,
  'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
  'C:/Program Files/Google/Chrome/Application/chrome.exe',
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/usr/bin/google-chrome',
  '/usr/bin/chromium',
].filter(Boolean);
const browser = candidates.find((p) => existsSync(p));
if (!url || !browser) {
  console.error(!url ? 'serve URL이 필요하다' : '브라우저를 찾지 못했다');
  process.exit(2);
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const prof = mkdtempSync(join(tmpdir(), 'daemok-e2e-'));
const proc = spawn(browser, ['--headless=new', '--disable-gpu', '--no-first-run', '--remote-debugging-port=0', `--user-data-dir=${prof}`, 'about:blank']);
const finish = async (code) => {
  proc.kill();
  await sleep(500);
  try { rmSync(prof, { recursive: true, force: true }); } catch {}
  process.exit(code);
};

let port;
for (let i = 0; i < 100 && !port; i++) {
  try { port = readFileSync(join(prof, 'DevToolsActivePort'), 'utf8').split('\n')[0]; } catch { await sleep(100); }
}
const page = (await (await fetch(`http://127.0.0.1:${port}/json`)).json()).find((t) => t.type === 'page');
const ws = new WebSocket(page.webSocketDebuggerUrl);
await new Promise((r) => ws.addEventListener('open', r));
let id = 0;
const pending = new Map();
const exceptions = [];
ws.addEventListener('message', (e) => {
  const m = JSON.parse(e.data);
  if (m.id && pending.has(m.id)) { pending.get(m.id)(m); pending.delete(m.id); }
  if (m.method === 'Runtime.exceptionThrown') exceptions.push(m.params.exceptionDetails.exception?.description || m.params.exceptionDetails.text);
});
const send = (method, params = {}) => new Promise((r) => { const i = ++id; pending.set(i, r); ws.send(JSON.stringify({ id: i, method, params })); });
const run = async (js) => (await send('Runtime.evaluate', { expression: js, returnByValue: true, awaitPromise: true })).result?.result?.value;

await send('Runtime.enable');
await send('Page.navigate', { url: url + '#module/api' });
await sleep(1500);

const steps = [
  ['tier', `(() => { const s = document.querySelector('select[data-edit="tier"]'); if (!s) return 'tier 칸 없음'; s.value = 'a'; s.dispatchEvent(new Event('change', { bubbles: true })); return 'ok'; })()`],
  ['add_dep', `(() => { const s = document.querySelector('select[data-adddep]'); if (!s) return '의존 추가 칸 없음'; s.value = 'util'; s.dispatchEvent(new Event('change', { bubbles: true })); return 'ok'; })()`],
  ['remove_dep', `(() => { const b = document.querySelector('[data-rmdep="core"]'); if (!b) return '지우기 버튼 없음'; b.click(); return 'ok'; })()`],
  ['add_module', `(() => { go('map'); const f = document.querySelector('#addmod'); if (!f) return '모듈 추가 칸 없음'; f.querySelector('#addmodid').value = 'web'; f.querySelector('#addmodtier').value = 'b'; f.requestSubmit(); return 'ok'; })()`],
];
let failed = false;
for (const [name, js] of steps) {
  const r = await run(js);
  console.log(name, r);
  if (r !== 'ok') failed = true;
  await sleep(1200);
}
if (exceptions.length) { console.log('페이지 예외:', exceptions.join(' | ')); failed = true; }
ws.close();
await finish(failed ? 1 : 0);
