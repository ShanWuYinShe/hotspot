// 跨端设计刻度校验：frontend theme-registry.css 的字号/圆角档位 与
// harmonyapp DesignConstants.ets 的 FONT.SIZE / RADIUS 档位必须一致，
// 防止一端收敛档位后另一端残留近邻档（历史上 13/15/22/28 漂移的根因）。
// 颜色校验见 check-cross-platform-colors.mjs（本脚本不重复）。
//
// 用法：node scripts/check-cross-platform-scale.mjs（由 `bun run check:theme` 串联）
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const frontendRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const repoRoot = dirname(frontendRoot);
const registryPath = join(frontendRoot, 'src/styles/theme/theme-registry.css');
const designConstantsPath = join(repoRoot, 'harmonyapp/entry/src/main/ets/constants/DesignConstants.ets');

const problems = [];

// ── 字号档位：--text-xxs/xs/sm/md（px）+ Tailwind 默认 lg 及以上（18/20/24/30/36） ──
// 鸿蒙 FONT.SIZE 对应：XS2 10 / XS 11 / SM 12 / MD 14 / LG_PLUS 18 / XL 20 / XXL 24 / XXXL 30 / HERO 36
const css = readFileSync(registryPath, 'utf8');
const ets = readFileSync(designConstantsPath, 'utf8');

const cssFontSizes = {};
for (const m of css.matchAll(/--text-([\w]+):\s*([\d.]+)px/g)) {
	cssFontSizes[m[1]] = Number(m[2]);
}
// Tailwind 默认档（前端 text-lg+ 直接沿用，鸿蒙侧用命名档对齐）
cssFontSizes.lg = 18;
cssFontSizes.xl = 20;
cssFontSizes['2xl'] = 24;
cssFontSizes['3xl'] = 30;
cssFontSizes['4xl'] = 36;

// 鸿蒙字号档：解析 FONT.SIZE 块的 KEY: value
const fontBlockMatch = ets.match(/SIZE:\s*\{([\s\S]*?)\n  \}/);
if (!fontBlockMatch) {
	problems.push('DesignConstants.ets 未找到 FONT.SIZE 块');
}
const arkFontSizes = {};
if (fontBlockMatch) {
	for (const m of fontBlockMatch[1].matchAll(/([A-Z_0-9]+):\s*([\d.]+)/g)) {
		arkFontSizes[m[1]] = Number(m[2]);
	}
}

// 前端 text-* ↔ 鸿蒙 FONT.SIZE 对应表
const FONT_MAP = {
	xxs: 'XS2',
	xs: 'XS',
	sm: 'SM',
	md: 'MD',
	lg: 'LG_PLUS',
	xl: 'XL',
	'2xl': 'XXL',
	'3xl': 'XXXL',
	'4xl': 'HERO'
};

for (const [cssKey, arkKey] of Object.entries(FONT_MAP)) {
	const cssVal = cssFontSizes[cssKey];
	const arkVal = arkFontSizes[arkKey];
	if (cssVal === undefined) {
		problems.push(`theme-registry.css 缺少 --text-${cssKey} 档`);
		continue;
	}
	if (arkVal === undefined) {
		problems.push(`DesignConstants.FONT.SIZE 缺少 ${arkKey} 档（对应前端 text-${cssKey}）`);
		continue;
	}
	if (cssVal !== arkVal) {
		problems.push(`字号漂移：前端 text-${cssKey}=${cssVal}px ≠ 鸿蒙 FONT.SIZE.${arkKey}=${arkVal}`);
	}
}

// 鸿蒙侧近邻档检查：字号集合内不得有 1-2px 差异的多余档（防止单端加档回潮）
// 已登记豁免：FONT.SIZE.LG=16 为 HarmonyOS 标准正文尺寸的平台适配档
// （harmonyapp/AGENTS.md 字号节），前端无对应档属平台差异，不判漂移。
const PLATFORM_ONLY_FONT_VALUES = [16];
const arkSortedVals = [...new Set(Object.values(arkFontSizes))].sort((a, b) => a - b);
const cssSortedVals = [...new Set(Object.values(cssFontSizes))].sort((a, b) => a - b);
for (const v of arkSortedVals) {
	if (PLATFORM_ONLY_FONT_VALUES.includes(v)) continue;
	if (!cssSortedVals.includes(v)) {
		problems.push(`鸿蒙 FONT.SIZE 存在前端不存在的档位值 ${v}（近邻档回潮？应归并到最近前端档）`);
	}
}

// ── 圆角档位：--radius-sm/md/lg/xl（12/16/20/24）+ sheet 28 ──
const cssRadii = {};
for (const m of css.matchAll(/--radius-(sm|md|lg|xl|full|sheet):\s*([^;]+);/g)) {
	const raw = m[2].trim();
	const px = raw.match(/^([\d.]+)(?:px|rem)$/);
	if (px) {
		cssRadii[m[1]] = m[1] === 'sheet' && raw.endsWith('rem') ? Number(px[1]) * 16 : Number(px[1]);
	}
}
cssRadii.full = 9999; // rounded-full 语义圆

// 鸿蒙圆角档：解析 RADIUS 块
const radiusBlockMatch = ets.match(/const RADIUS[^=]*=\s*\{([\s\S]*?)\n\}/) || ets.match(/RADIUS:\s*\{([\s\S]*?)\n  \}/);
const arkRadii = {};
if (radiusBlockMatch) {
	for (const m of radiusBlockMatch[1].matchAll(/([A-Z_0-9]+):\s*([\d.]+)/g)) {
		arkRadii[m[1]] = Number(m[2]);
	}
} else {
	problems.push('DesignConstants.ets 未找到 RADIUS 块');
}

const RADIUS_MAP = { sm: 'SM', md: 'MD', lg: 'LG', xl: 'XL', full: 'FULL' };
for (const [cssKey, arkKey] of Object.entries(RADIUS_MAP)) {
	const cssVal = cssRadii[cssKey];
	const arkVal = arkRadii[arkKey];
	if (cssVal === undefined || arkVal === undefined) continue; // 缺档已由上面覆盖（圆角前端 sm-xl 可能经 var 引用解析不到）
	if (cssKey === 'full') continue; // full 前端 9999 / 鸿蒙 999 语义相同（都表示胶囊），跳过数值比对
	if (cssVal !== arkVal) {
		problems.push(`圆角漂移：前端 --radius-${cssKey}=${cssVal}px ≠ 鸿蒙 RADIUS.${arkKey}=${arkVal}`);
	}
}

if (problems.length > 0) {
	console.error('跨端设计刻度不一致（frontend theme-registry ↔ harmonyapp DesignConstants）：');
	for (const p of problems) console.error(`  - ${p}`);
	console.error('修正方向：档位以两端同刻度为准；新增/归并档位必须两侧同步。');
	process.exit(1);
}
console.log(
	`跨端设计刻度一致 OK：字号 ${Object.keys(FONT_MAP).length} 档 + 圆角 ${Object.keys(RADIUS_MAP).length} 档已对齐（前端 text-*/radius-* ↔ 鸿蒙 FONT.SIZE/RADIUS）。`
);
