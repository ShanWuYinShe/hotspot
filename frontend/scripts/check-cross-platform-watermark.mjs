// 跨端水印透明度校验：web `--cst-opacity-watermark` ↔ 鸿蒙 `DesignConstants.OPACITY.WATERMARK`。
//
// 两端都是单品值（当前 0.08），用于无图卡片 / 装饰水印槽位；场景少但一旦单端调整
// 就会静默漂移（同一视觉语义两端带宽不同），故纳入 check:theme 逐值校验。
// 无图 fallback 渐变不在此脚本内：鸿蒙 cardFallback* 是 App 端专属三段渐变
// （accent 与 page/surface 预混，各主题比例不同），web 侧仅纯色 surface-primary，
// 无共享公式，登记为人工视觉核对（见 docs/design-system.md §15）。
//
// 用法：node scripts/check-cross-platform-watermark.mjs（由 `bun run check:theme` 串联）
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const frontendRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const repoRoot = dirname(frontendRoot);

const SHARED_CSS = join(frontendRoot, 'src/styles/theme/shared.css');
const REGISTRY_CSS = join(frontendRoot, 'src/styles/theme/theme-registry.css');
const DESIGN_CONSTANTS = join(repoRoot, 'harmonyapp/entry/src/main/ets/constants/DesignConstants.ets');

const errors = [];

function fail(message) {
	errors.push(message);
}

function readToken(file, pattern, label) {
	const text = readFileSync(file, 'utf8');
	const match = text.match(pattern);
	if (!match) {
		fail(label + '：未找到定义（pattern ' + String(pattern) + '）');
		return null;
	}
	return Number(match[1]);
}

const webWatermark = readToken(SHARED_CSS, /--cst-opacity-watermark:\s*([0-9.]+);/, 'web --cst-opacity-watermark');
const harmonyWatermark = readToken(DESIGN_CONSTANTS, /WATERMARK:\s*([0-9.]+),/, '鸿蒙 OPACITY.WATERMARK');

// theme-registry 必须把原始 token 暴漏为 Tailwind opacity 名（否则组件里 opacity-watermark-* 类静默失效）
const registry = readFileSync(REGISTRY_CSS, 'utf8');
if (!/--opacity-watermark:\s*var\(--cst-opacity-watermark\);/.test(registry)) {
	fail('theme-registry.css 缺少 --opacity-watermark: var(--cst-opacity-watermark) 注册');
}

if (webWatermark !== null && harmonyWatermark !== null) {
	if (Math.abs(webWatermark - harmonyWatermark) > 1e-9) {
		fail(
			'水印透明度跨端不一致：web --cst-opacity-watermark=' +
				webWatermark +
				'，鸿蒙 OPACITY.WATERMARK=' +
				harmonyWatermark +
				'（改单端后必须同步另一端）'
		);
	} else {
		console.log(
			'✓ 水印透明度：web --cst-opacity-watermark = ' +
				webWatermark +
				' ↔ 鸿蒙 OPACITY.WATERMARK = ' +
				harmonyWatermark
		);
	}
}

if (errors.length > 0) {
	for (const message of errors) console.error('✗ ' + message);
	process.exit(1);
}
console.log('✓ 跨端水印透明度校验通过');
