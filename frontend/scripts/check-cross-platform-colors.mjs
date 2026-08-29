// 跨端色彩 token 校验：frontend themes/*.css 的 --cst-palette-* 与
// harmonyapp ThemeConstants.ets 的 ColorConfig 必须逐值一致（归一化后比较），
// 防止两端各自改色后漂移。配套约定见 frontend/AGENTS.md「跨端色彩对齐」。
//
// 用法：node scripts/check-cross-platform-colors.mjs（由 `bun run check:theme` 串联）
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const frontendRoot = dirname(dirname(fileURLToPath(import.meta.url)));
const repoRoot = dirname(frontendRoot);
const harmonyTheme = join(repoRoot, 'harmonyapp/entry/src/main/ets/core/theme/ThemeConstants.ets');

// ── 权威映射表：CSS palette 变量 ↔ ColorConfig 字段（一对多：ets 派生字段共用一个色源）──
// 仅列两端都有明确对应关系的色；ets 侧独有装饰色（glow/tabBar 等）不在此列。
const MAPPING = [
	['--cst-palette-page-bg', ['pageBackground']],
	['--cst-palette-text', ['textPrimary']],
	['--cst-palette-text-muted', ['textSecondary']],
	['--cst-palette-accent-primary', ['accentBackground', 'info', 'primaryButtonBorderColor']],
	['--cst-palette-on-accent', ['accentText']],
	['--cst-palette-online', ['online', 'success']],
	['--cst-palette-hot', ['error']],
	['--cst-palette-warning', ['warning']],
	['--cst-palette-map-user', ['mapUser']],
	['--cst-palette-map-group', ['mapGroup']],
	['--cst-palette-map-moment', ['mapMoment']]
];

const THEMES = ['vibrant', 'forest'];
// ets 侧常量前缀映射：前端 forest(清野) ≡ harmonyapp default_clean（accent #3D7A55 同源）；
// harmonyapp 另有 figma_neu 开发用主题，前端无对应，不纳入校验。
const ETS_PREFIX = { vibrant: 'VIBRANT', forest: 'DEFAULT_CLEAN' };
const MODES = ['light', 'dark'];

// ── 归一化：rgba(r,g,b,a) / #RGB / #RRGGBB / #RRGGBBAA / 带空格 → 统一 #RRGGBBAA（小写）──
function normalizeColor(raw) {
	let s = raw.trim().toLowerCase();
	const rgba = s.match(/^rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*([\d.]+))?\)$/);
	if (rgba) {
		const [r, g, b] = [rgba[1], rgba[2], rgba[3]].map((v) => Number(v).toString(16).padStart(2, '0'));
		const alpha = rgba[4] === undefined ? 1 : Number(rgba[4]);
		return `#${r}${g}${b}${Math.round(alpha * 255).toString(16).padStart(2, '0')}`;
	}
	if (/^#[0-9a-f]{3}$/.test(s)) {
		s = `#${s[1]}${s[1]}${s[2]}${s[2]}${s[3]}${s[3]}`;
	}
	if (/^#[0-9a-f]{6}$/.test(s)) return `${s}ff`;
	if (/^#[0-9a-f]{8}$/.test(s)) return s;
	return null; // 无法解析（如 var() 引用）—— 由调用方跳过
}

// ── 解析 frontend themes css：{ theme: { mode: { var: normalized } } } ──
function parseThemeCSS(theme) {
	const css = readFileSync(join(frontendRoot, `src/styles/theme/themes/${theme}.css`), 'utf8');
	const result = { light: {}, dark: {} };
	// light 块：`.vibrant {` 到 `.vibrant.dark {`；dark 块：其后到配对结束
	const lightStart = css.indexOf(`.${theme} {`);
	const darkStart = css.indexOf(`.${theme}.dark {`);
	if (lightStart < 0 || darkStart < 0) throw new Error(`${theme}.css 缺少 .${theme}{} 或 .${theme}.dark{} 块`);
	const lightBlock = css.slice(lightStart, darkStart);
	const darkEnd = css.indexOf('\n}', darkStart);
	const darkBlock = css.slice(darkStart, darkEnd < 0 ? undefined : darkEnd);
	for (const [mode, block] of [
		['light', lightBlock],
		['dark', darkBlock]
	]) {
		for (const m of block.matchAll(/--cst-palette-([\w-]+):\s*([^;]+);/g)) {
			const normalized = normalizeColor(m[2]);
			if (normalized) result[mode][`--cst-palette-${m[1]}`] = normalized;
		}
	}
	return result;
}

// ── 解析 harmonyapp ThemeConstants.ets：{ THEME_MODE: { field: normalized } } ──
function parseArkPalettes() {
	const ets = readFileSync(harmonyTheme, 'utf8');
	const palettes = {};
	for (const m of ets.matchAll(
		/static readonly (VIBRANT_LIGHT|VIBRANT_DARK|DEFAULT_CLEAN_LIGHT|DEFAULT_CLEAN_DARK)_COLORS: ColorConfig = \{([\s\S]*?)\n  \};/g
	)) {
		const [, constName, body] = m;
		const fields = {};
		for (const f of body.matchAll(/(\w+):\s*'([^']+)'/g)) {
			const normalized = normalizeColor(f[2]);
			if (normalized) fields[f[1]] = normalized;
		}
		// 常量名（如 DEFAULT_CLEAN_LIGHT）→ 主题族/明暗 归一到前端命名（forest_light 等）
		const key = constName.replace('DEFAULT_CLEAN', 'forest').replace('VIBRANT', 'vibrant').toLowerCase();
		palettes[key] = fields;
	}
	return palettes;
}

// ── 前端主题选择器 swatch 校验：THEME_OPTIONS.swatch 必须等于该主题 light accent ──
function checkSwatches(cssThemes) {
	const store = readFileSync(join(frontendRoot, 'src/stores/theme.svelte.ts'), 'utf8');
	const problems = [];
	for (const m of store.matchAll(/\{ id: '(\w+)',[^}]*swatch: '(#[0-9a-fA-F]{6})'/g)) {
		const [, id, swatch] = m;
		const accent = cssThemes[id]?.light['--cst-palette-accent-primary'];
		if (!accent) {
			problems.push(`主题 ${id} 缺少 light accent-primary 定义`);
			continue;
		}
		if (normalizeColor(swatch) !== accent) {
			problems.push(`THEME_OPTIONS[${id}].swatch ${swatch} 与 themes/${id}.css accent-primary ${accent.slice(0, 7)} 不一致`);
		}
	}
	return problems;
}

const cssThemes = {};
for (const theme of THEMES) cssThemes[theme] = parseThemeCSS(theme);
const arkPalettes = parseArkPalettes();

const problems = [];
for (const theme of THEMES) {
	for (const mode of MODES) {
		const ark = arkPalettes[`${theme}_${mode}`];
		if (!ark) {
			problems.push(`ThemeConstants.ets 缺少 ${ETS_PREFIX[theme]}_${mode.toUpperCase()}_COLORS`);
			continue;
		}
		for (const [cssVar, arkFields] of MAPPING) {
			const cssValue = cssThemes[theme][mode][cssVar];
			if (!cssValue) continue; // 该主题未定义此 palette 变量（如个别主题无 map-*）
			for (const field of arkFields) {
				const arkValue = ark[field];
				if (arkValue === undefined) {
					problems.push(`${theme}_${mode}: ColorConfig 缺少字段 ${field}（对应 ${cssVar}）`);
				} else if (arkValue !== cssValue) {
					problems.push(
						`${theme}_${mode}: ${cssVar}(${cssValue.slice(0, 7)}) ≠ ColorConfig.${field}(${arkValue.slice(0, 7)})`
					);
				}
			}
		}
	}
}
problems.push(...checkSwatches(cssThemes));

if (problems.length > 0) {
	console.error('跨端色彩 token 不一致（前端 themes/*.css ↔ harmonyapp ThemeConstants.ets）：');
	for (const p of problems) console.error(`  - ${p}`);
	console.error('修正方向：以 spec/设计定稿为准，两侧同步改，勿单端改色。');
	process.exit(1);
}
console.log(`跨端色彩一致 OK：${THEMES.length} 主题 × ${MODES.length} 明暗 × ${MAPPING.length} 组映射已对齐（vibrant/forest light+dark）。`);
