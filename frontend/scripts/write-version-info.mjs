// 构建期生成 static/version.json（nginx 静态直出，线上排障用：
// curl https://<域名>/version.json 即可核对前端产物版本）。
//
// 信息来源优先级：
//   1. 环境变量 GIT_VERSION / GIT_COMMIT / GIT_BUILD_TIME
//      —— Docker 构建上下文没有 .git，部署链路（frontend/Dockerfile ←
//         docker-compose.yml build args ← scripts/deploy-production.sh）必须
//         以环境变量注入；
//   2. 本地 git（git describe --tags --always --dirty）—— 本机直跑构建用；
//   3. 兜底 dev，保证 version.json 始终存在、永不 404。
//
// 该文件由构建生成、不入库（.gitignore），随 adapter-static 从 static/ 拷入产物。
import { execFileSync } from 'node:child_process';
import { writeFileSync, mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

function fromEnv() {
	if (!process.env.GIT_VERSION && !process.env.GIT_COMMIT) return null;
	return {
		version: process.env.GIT_VERSION || 'dev',
		commit: process.env.GIT_COMMIT || '',
		build_time: process.env.GIT_BUILD_TIME || ''
	};
}

function fromGit() {
	const run = (args) => execFileSync('git', args, { cwd: root, encoding: 'utf8' }).trim();
	try {
		return {
			version: run(['describe', '--tags', '--always', '--dirty']),
			commit: run(['rev-parse', 'HEAD']),
			build_time: new Date().toISOString()
		};
	} catch {
		return null;
	}
}

const info =
	fromEnv() ??
	fromGit() ?? {
		version: 'dev',
		commit: '',
		build_time: new Date().toISOString()
	};

mkdirSync(join(root, 'static'), { recursive: true });
writeFileSync(
	join(root, 'static', 'version.json'),
	`${JSON.stringify({ project: 'hotspot-frontend', ...info }, null, 2)}\n`
);
console.log(`version.json: ${info.version} (commit ${info.commit || '-'})`);
