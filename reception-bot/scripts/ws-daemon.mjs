import { spawn } from 'node:child_process';
import { existsSync, mkdirSync, openSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const dir = dirname(fileURLToPath(import.meta.url));
const self = join(dir, 'ws-online.mjs');
const stateDir = join(dirname(dir), 'state');
const log = join(stateDir, 'ws-online.log');
// pid 文件由 ws-online.mjs 单实例守卫自己写入(此处不再覆写,避免指向已被守卫拒绝的短命进程)
if (!existsSync(stateDir)) mkdirSync(stateDir, { recursive: true });
const out = openSync(log, 'a');
const child = spawn(process.execPath, [self], {
  detached: true,
  stdio: ['ignore', out, out],
  env: process.env,
});
child.unref();
console.log('daemon started pid=' + child.pid + '(实际在线 pid 见 state/ws-online.pid,重复启动会被单实例守卫拒绝)');
