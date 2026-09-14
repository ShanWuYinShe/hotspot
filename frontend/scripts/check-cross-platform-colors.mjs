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
	['--cst-palette-chat-bg', ['chatBackground']],
	['--cst-palette-text', ['textPrimary']],
	['--cst-palette-text-muted', ['textSecondary']],
	['--cst-palette-accent-primary', ['accentBackground', 'info', 'primaryButtonBorderColor']],
	['--cst-palette-on-accent', ['accentText']],
	['--cst-palette-online', ['online', 'success']],
	['--cst-palette-hot', ['error']],
	['--cst-palette-warning', ['warning']],
	['--cst-palette-map-user', ['mapUser']],
	['--cst-palette-map-group', ['mapGroup']],
	['--cst-palette-map-moment', ['mapMoment']],
	['--cst-palette-overlay-strong', ['overlayStrong']],
	['--cst-palette-border', ['borderColor']]
];

// ── 合成色规则：ets 侧为不透明 hex = CSS 半透明色按 alpha 叠加底色的结果 ──
// ArkTS 无法表达「引用式透明叠加」，只能预合成不透明色；alpha 或底色一变即漂移，
// 故在此按同一公式复算校验（容差 ±1，四舍五入边界）。
//   blend(fg, bg, a) = round(fg*a + bg*(1-a)) 逐通道
const COMPOSED = [
	// accent-soft（10%~14% accent）叠 page-bg
	['accentMuted', '--cst-palette-accent-soft', '--cst-palette-page-bg'],
	// 底部导航 active 指示器：accent @ --cst-palette-opacity-active(0.13) 叠 page-bg
	// （nav 玻璃底 = surface-panel 90% over page，预合成近似 page-bg）
	['tabBarActiveBackground', '--cst-palette-accent-primary@opacity-active', '--cst-palette-page-bg']
];

function blendColor(fg, bg) {
	// fg/bg: #rrggbbaa；返回不透明 #rrggbbaa（a=ff）
	const a = parseInt(fg.slice(7, 9), 16) / 255;
	const mix = (i) =>
		Math.round(parseInt(fg.slice(1 + i * 2, 3 + i * 2), 16) * a + parseInt(bg.slice(1 + i * 2, 3 + i * 2), 16) * (1 - a));
	const near = (v, t) => Math.abs(v - t) <= 1;
	return { hex: `#${[0, 1, 2].map((i) => mix(i).toString(16).padStart(2, '0')).join('')}ff`, near };
}

function nearColor(a, b) {
	if (a.length !== 9 || b.length !== 9) return false;
	for (let i = 1; i <= 6; i += 2) {
		if (Math.abs(parseInt(a.slice(i, i + 2), 16) - parseInt(b.slice(i, i + 2), 16)) > 1) return false;
	}
	return a.slice(7, 9) === b.slice(7, 9);
}

function checkComposed(theme, mode, ark) {
	const problems = [];
	const vars = cssThemes[theme][mode];
	for (const [field, fgVar, bgVar] of COMPOSED) {
		let fg = vars[fgVar];
		let fgDesc = fgVar;
		if (fgVar.includes('@')) {
			// accent-primary @ opacity-active：从 CSS 原文抓透明度档
			const [baseVar, opacityName] = fgVar.split('@');
			const css = readFileSync(join(frontendRoot, `src/styles/theme/themes/${theme}.css`), 'utf8');
			const blockStart = css.indexOf(mode === 'light' ? `.${theme} {` : `.${theme}.dark {`);
			const blockEnd = css.indexOf('\n}', blockStart);
			const block = css.slice(blockStart, blockEnd < 0 ? undefined : blockEnd);
			const om = block.match(new RegExp(`--cst-palette-${opacityName}:\\s*([\\d.]+)`));
			if (!om) { problems.push(`${theme}_${mode}: CSS 缺少 --cst-palette-${opacityName}`); continue; }
			const base = vars[baseVar];
			if (!base) continue;
			const alpha = Math.round(Number(om[1]) * 255).toString(16).padStart(2, '0');
			fg = base.slice(0, 7) + alpha;
			fgDesc = `${baseVar}@${om[1]}`;
		}
		const bg = vars[bgVar];
		if (!fg || !bg) continue;
		const expected = blendColor(fg, bg).hex;
		const actual = ark[field];
		if (actual === undefined) {
			problems.push(`${theme}_${mode}: ColorConfig 缺少字段 ${field}（合成自 ${fgDesc} over ${bgVar}）`);
		} else if (!nearColor(actual, expected)) {
			problems.push(
				`${theme}_${mode}: ColorConfig.${field}(${actual.slice(0, 7)}) ≠ 预合成 ${fgDesc} over ${bgVar}(${expected.slice(0, 7)})`
			);
		}
	}
	return problems;
}

const THEMES = ['vibrant', 'forest', 'inkwash'];
// ets 侧常量前缀映射：前端 forest(清野) ≡ harmonyapp default_clean（accent #3D7A55 同源）；
// harmonyapp 另有 figma_neu 开发用主题，前端无对应，不纳入校验。
const ETS_PREFIX = { vibrant: 'VIBRANT', forest: 'DEFAULT_CLEAN', inkwash: 'INKWASH' };
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
		/static readonly (VIBRANT_LIGHT|VIBRANT_DARK|DEFAULT_CLEAN_LIGHT|DEFAULT_CLEAN_DARK|INKWASH_LIGHT|INKWASH_DARK)_COLORS: ColorConfig = \{([\s\S]*?)\n  \};/g
	)) {
		const [, constName, body] = m;
		const fields = {};
		for (const f of body.matchAll(/(\w+):\s*'([^']+)'/g)) {
			const raw = f[2].toLowerCase();
			let normalized = normalizeColor(raw);
			if (normalized && /^#[0-9a-f]{8}$/.test(raw)) {
				// ArkTS 8 位 hex 为 #AARRGGBB（alpha 前置），与本脚本标准 #RRGGBBAA 相反
				normalized = `#${raw.slice(3, 9)}${raw.slice(1, 3)}`;
			}
			if (normalized) fields[f[1]] = normalized;
		}
		// 常量名（如 DEFAULT_CLEAN_LIGHT）→ 主题族/明暗 归一到前端命名（forest_light 等）
		const key = constName
			.replace('DEFAULT_CLEAN', 'forest')
			.replace('VIBRANT', 'vibrant')
			.replace('INKWASH', 'inkwash')
			.toLowerCase();
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
	for (const mode of MODES) problems.push(...checkComposed(theme, mode, arkPalettes[`${theme}_${mode}`]));
}
problems.push(...checkSwatches(cssThemes));

if (problems.length > 0) {
	console.error('跨端色彩 token 不一致（前端 themes/*.css ↔ harmonyapp ThemeConstants.ets）：');
	for (const p of problems) console.error(`  - ${p}`);
	console.error('修正方向：以 spec/设计定稿为准，两侧同步改，勿单端改色。');
	process.exit(1);
}
console.log(
	`跨端色彩一致 OK：${THEMES.length} 主题 × ${MODES.length} 明暗 × ${MAPPING.length} 组映射已对齐（${THEMES.join('/')} light+dark）。`
);
