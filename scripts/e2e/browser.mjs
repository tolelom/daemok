// e2e 공용: headless 브라우저를 띄워 페이지 하나를 DevTools 프로토콜로 다룬다.
// 브라우저는 DAEMOK_BROWSER, 없으면 흔한 설치 경로를 찾는다.
import { spawn } from 'node:child_process';
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

export const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

export async function openBrowser() {
  const candidates = [
    process.env.DAEMOK_BROWSER,
    'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',
    'C:/Program Files/Google/Chrome/Application/chrome.exe',
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/usr/bin/google-chrome',
    '/usr/bin/chromium',
  ].filter(Boolean);
  const browser = candidates.find((p) => existsSync(p));
  if (!browser) {
    console.error('브라우저를 찾지 못했다');
    process.exit(2);
  }
  const prof = mkdtempSync(join(tmpdir(), 'daemok-e2e-'));
  const proc = spawn(browser, ['--headless=new', '--disable-gpu', '--no-first-run', '--remote-debugging-port=0', `--user-data-dir=${prof}`, 'about:blank']);
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
  const finish = async (code) => {
    ws.close();
    proc.kill();
    await sleep(500);
    try { rmSync(prof, { recursive: true, force: true }); } catch {}
    process.exit(code);
  };
  return { send, run, exceptions, finish };
}

// runSteps는 'ok'를 돌려주지 않는 단계와 페이지 예외를 실패로 보고 끝낸다.
export async function runSteps(b, steps, wait = 0) {
  let failed = false;
  for (const [name, js] of steps) {
    const r = await b.run(js);
    console.log(name, r);
    if (r !== 'ok') failed = true;
    await sleep(wait);
  }
  if (b.exceptions.length) { console.log('페이지 예외:', b.exceptions.join(' | ')); failed = true; }
  await b.finish(failed ? 1 : 0);
}
