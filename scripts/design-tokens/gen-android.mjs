#!/usr/bin/env node
/**
 * Android / Web 设计 token 生成器（零第三方依赖，Node >= 18）
 *
 * 数据源（唯一权威）：
 *   frontend/src/styles/theme/themes/{vibrant,forest,inkwash}.css   原始色板 + 主题语义覆写（light/dark）
 *   frontend/src/styles/theme/semantic.css                          语义别名 + 输入族 + 导航/交互派生
 *   frontend/src/styles/theme/shared.css                            圆角 / 动效 / 缓动 / 边框宽度
 *   frontend/src/styles/theme/theme-registry.css                    字号档（--text-*）/ --radius-sheet
 *   frontend/src/styles/theme/utilities.css                         页面进场 / 内容淡入时长
 *
 * 产物：
 *   androidapp/app/src/main/java/com/hotspot/app/ui/theme/Tokens.kt   Compose 常量
 *   androidapp/app/src/main/assets/design-tokens.json               生成清单（供跨端校验脚本消费）
 *
 * 颜色归一化：一律输出小写 #rrggbbaa（alpha 在末位，与 CSS rgba(r,g,b,a) 等价）；
 * color-mix() 按 CSS Color 5 的 premultiplied-alpha 规则求值（round 到 8bit），
 * 与浏览器渲染值逐通道一致。
 *
 * 用法：
 *   node scripts/design-tokens/gen-android.mjs          重新生成产物
 *   node scripts/design-tokens/gen-android.mjs --check  只校验产物是否最新（CI 用，不写盘）
 */
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { dirname, join } from 'node:path';

const HERE = dirname(fileURLToPath(import.meta.url));
export const REPO_ROOT = join(HERE, '..', '..');

const THEME_DIR = join(REPO_ROOT, 'frontend/src/styles/theme');
const TOKENS_KT = join(REPO_ROOT, 'androidapp/app/src/main/java/com/hotspot/app/ui/theme/Tokens.kt');
const TOKENS_JSON = join(REPO_ROOT, 'androidapp/app/src/main/assets/design-tokens.json');

export const THEMES = ['vibrant', 'forest', 'inkwash'];
export const MODES = ['light', 'dark'];
const MODE_OBJECT = { light: 'Light', dark: 'Dark' };
const THEME_OBJECT = { vibrant: 'Vibrant', forest: 'Forest', inkwash: 'Inkwash' };

/** 语义色产出清单：全部是组件会消费的颜色 token；名字即 --cst-<name> 的后缀。 */
export const SEMANTIC_TOKENS = [
  // 文字
  'text-primary', 'text-secondary', 'text-on-accent', 'text-on-status',
  'text-accent-on-soft', 'text-status-online-on-soft', 'text-status-warning-on-soft', 'text-status-danger-on-soft',
  // 表面
  'surface-primary', 'surface-page', 'surface-chat', 'surface-island', 'surface-subtle', 'surface-panel',
  'surface-raised', 'surface-overlay', 'surface-overlay-strong', 'surface-scrim', 'surface-nav-glass',
  'surface-card-menu', 'surface-toggle-off', 'surface-toggle-thumb', 'surface-glass-badge',
  'surface-media-scrim-strong',
  // 品牌
  'accent-default', 'accent-soft', 'accent-border', 'accent-soft-bg', 'accent-border-soft',
  // 描边
  'border-default', 'border-strong', 'border-subtle',
  // 状态
  'status-online', 'status-danger', 'status-warning',
  'status-online-bg', 'status-online-border', 'status-danger-bg', 'status-danger-border',
  'status-warning-bg', 'status-warning-border',
  // 输入族
  'field-bg', 'field-bg-disabled', 'field-border', 'field-border-hover', 'field-border-focus',
  'field-ring', 'field-ring-danger',
  // 交互
  'interactive-hover', 'interactive-active', 'interactive-hover-fill', 'interactive-active-fill',
  // 压图/反色
  'content-inverse', 'content-inverse-muted', 'content-inverse-secondary', 'content-on-marker',
  // 地图标记
  'marker-user', 'marker-group', 'marker-moment',
  'marker-user-bg', 'marker-user-border', 'marker-user-glow',
  'marker-group-bg', 'marker-group-border', 'marker-group-glow',
  'marker-moment-bg', 'marker-moment-border', 'marker-moment-glow',
  // 导航
  'navigation-content', 'navigation-content-active', 'navigation-active-indicator',
  'surface-navigation-hover', 'surface-navigation-active', 'surface-navigation-active-hover',
  // 光晕原料
  'glow-accent', 'glow-danger',
];

/** 交互数值 token（随主题）：--cst-<name>。 */
const INTERACTION_TOKENS = ['card-tap-scale', 'hover-lift', 'hover-image-zoom'];

/** 氛围画布 token（design-system §16：三主题 × 明暗，装饰层专用）。 */
export const AMBIENT_TOKENS = [
  'glow-1', 'glow-2', 'sky', 'line', 'dot', 'sun',
  'ink-1', 'ink-2', 'ink-3', 'emblem', 'horizon', 'vignette',
];

/** 字号档：前四档是 Web 覆写（theme-registry.css），lg 及以上沿用 Tailwind 默认。 */
const FONT_SIZE_SPEC = [
  ['xxs', '--text-xxs', null],
  ['xs', '--text-xs', null],
  ['sm', '--text-sm', null],
  ['md', '--text-md', null],
  ['lg', '--text-lg', 1.125], // Tailwind text-lg = 1.125rem
  ['xl', '--text-xl', 1.25],
  ['xxl', '--text-2xl', 1.5],
  ['xxxl', '--text-3xl', 1.875],
  ['hero', '--text-4xl', 2.25],
];

/** 圆角档：--cst-shape-* 在 shared.css，--radius-sheet 在 theme-registry.css。 */
const RADIUS_SPEC = [
  ['sm', '--cst-shape-sm'],
  ['md', '--cst-shape-md'],
  ['lg', '--cst-shape-lg'],
  ['xl', '--cst-shape-xl'],
  ['full', '--cst-shape-full'],
  ['sheet', '--radius-sheet'],
];

/** 动效时长 token：--cst-motion-* / 页面进场 / 内容淡入。 */
const MOTION_SPEC = [
  ['instant', '--cst-motion-instant'],
  ['rapid', '--cst-motion-rapid'],
  ['snappy', '--cst-motion-snappy'],
  ['standard', '--cst-motion-standard'],
  ['relaxed', '--cst-motion-relaxed'],
  ['spring', '--cst-motion-spring'],
  ['slow', '--cst-motion-slow'],
  ['gentle', '--cst-motion-gentle'],
  ['spinner', '--cst-motion-spinner'],
  ['pulse', '--cst-motion-pulse'],
  ['reduced', '--cst-motion-reduced'],
  ['skeleton', '--cst-motion-skeleton'],
  ['ambient', '--cst-motion-ambient'],
  ['ambientSlow', '--cst-motion-ambient-slow'],
  ['pageEnter', '--cst-page-enter-duration'],
  ['contentFade', '--cst-content-fade-duration'],
];

const EASING_SPEC = [
  ['standard', '--cst-ease-standard'],
  ['bounce', '--cst-ease-bounce'],
];

// ────────────────────────────────────────────────────────────────────────────
// CSS 解析
// ────────────────────────────────────────────────────────────────────────────

function readCss(file) {
  return readFileSync(join(THEME_DIR, file), 'utf8');
}

/** 去掉 /* 注释 *\/，避免注释里的分号 / 花括号污染解析。 */
function stripComments(css) {
  return css.replace(/\/\*[\s\S]*?\*\//g, '');
}

/** 从 selector 处开始截取配对花括号内的声明块文本。 */
function extractBlock(css, selector) {
  const idx = css.indexOf(selector);
  if (idx < 0) throw new Error(' 找不到选择器 ' + selector);
  const open = css.indexOf('{', idx);
  let depth = 0;
  for (let i = open; i < css.length; i++) {
    if (css[i] === '{') depth++;
    else if (css[i] === '}') {
      depth--;
      if (depth === 0) return css.slice(open + 1, i);
    }
  }
  throw new Error('花括号不配对：' + selector);
}

/** 解析一组 --name: value; 声明，返回 { 'name': 'value' }（name 保留 -- 前缀）。 */
function parseDecls(block) {
  const out = {};
  for (const m of block.matchAll(/(--[\w-]+)\s*:\s*([^;{}]+);/g)) {
    out[m[1]] = m[2].trim();
  }
  return out;
}

function parseSimpleCss(file) {
  return parseDecls(stripComments(readCss(file)));
}

/** themes/{theme}.css → { light: {...}, dark: {...} }；dark 未定义时按 CSS 级联回落到 light。 */
function parseTheme(theme) {
  const css = stripComments(readCss('themes/' + theme + '.css'));
  const light = parseDecls(extractBlock(css, '.' + theme + ' {'));
  const dark = parseDecls(extractBlock(css, '.' + theme + '.dark {'));
  return { light, dark };
}

// ────────────────────────────────────────────────────────────────────────────
// 颜色求值：字面量 / var() / color-mix()
// ────────────────────────────────────────────────────────────────────────────

function hexToColor(hex) {
  let h = hex.trim().toLowerCase();
  if (h.length === 4) h = '#' + h[1] + h[1] + h[2] + h[2] + h[3] + h[3];
  if (h.length === 7) h += 'ff';
  if (!/^#[0-9a-f]{8}$/.test(h)) throw new Error('无法解析颜色字面量：' + hex);
  return {
    r: parseInt(h.slice(1, 3), 16),
    g: parseInt(h.slice(3, 5), 16),
    b: parseInt(h.slice(5, 7), 16),
    a: parseInt(h.slice(7, 9), 16) / 255,
  };
}

function parseRgba(raw) {
  const m = /^rgba?\(\s*([\d.]+)\s*,\s*([\d.]+)\s*,\s*([\d.]+)\s*(?:,\s*([\d.]+)\s*)?\)$/.exec(raw.trim());
  if (!m) return null;
  return {
    r: Number(m[1]), g: Number(m[2]), b: Number(m[3]),
    a: m[4] === undefined ? 1 : Number(m[4]),
  };
}

function roundChannel(v) {
  return Math.max(0, Math.min(255, Math.round(v)));
}

function colorToHex(c) {
  const to2 = (v) => roundChannel(v).toString(16).padStart(2, '0');
  return '#' + to2(c.r) + to2(c.g) + to2(c.b) + to2(c.a * 255);
}

/** 按括号深度切分逗号参数（var() 兜底里可能带逗号）。 */
function splitArgs(raw) {
  const out = [];
  let depth = 0;
  let cur = '';
  for (const ch of raw) {
    if (ch === '(') depth++;
    if (ch === ')') depth--;
    if (ch === ',' && depth === 0) {
      out.push(cur.trim());
      cur = '';
    } else {
      cur += ch;
    }
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

/** CSS Color 5：srgb 空间 premultiplied alpha 插值。 */
function mixColors(c1, p1, c2, p2) {
  const w1 = p1 / 100;
  const w2 = p2 / 100;
  const sum = w1 + w2;
  if (sum <= 0) throw new Error('color-mix 权重非法');
  const n1 = w1 / sum;
  const n2 = w2 / sum;
  const a = c1.a * n1 + c2.a * n2;
  if (a <= 0) return { r: 0, g: 0, b: 0, a: 0 };
  return {
    r: (c1.r * c1.a * n1 + c2.r * c2.a * n2) / a,
    g: (c1.g * c1.a * n1 + c2.g * c2.a * n2) / a,
    b: (c1.b * c1.a * n1 + c2.b * c2.a * n2) / a,
    a,
  };
}

function createResolver(themeVars, semanticVars, sharedVars) {
  const cache = new Map();

  function lookupRaw(name, theme, mode) {
    if (themeVars[theme][mode][name] !== undefined) return { raw: themeVars[theme][mode][name] };
    // CSS 级联：.theme.dark 未覆写时沿用 .theme（light 块）的值
    if (themeVars[theme].light[name] !== undefined) return { raw: themeVars[theme].light[name] };
    if (semanticVars[name] !== undefined) return { raw: semanticVars[name] };
    if (sharedVars[name] !== undefined) return { raw: sharedVars[name] };
    return null;
  }

  function evalColor(raw, theme, mode, depth) {
    if (depth > 24) throw new Error('var()/color-mix() 递归过深：' + raw);
    const value = raw.trim();
    if (value === 'transparent') return { r: 0, g: 0, b: 0, a: 0 };
    if (value.startsWith('#')) return hexToColor(value);
    const rgba = parseRgba(value);
    if (rgba) return rgba;

    const varMatch = /^var\(\s*(--[\w-]+)\s*(?:,(.+))?\)$/.exec(value);
    if (varMatch) {
      const found = lookupRaw(varMatch[1], theme, mode);
      if (found) return evalColor(found.raw, theme, mode, depth + 1);
      if (varMatch[2] !== undefined) return evalColor(varMatch[2], theme, mode, depth + 1);
      throw new Error('未定义变量 ' + varMatch[1] + '（' + theme + '/' + mode + '）');
    }

    const cm = /^color-mix\(\s*in\s+srgb\s*,([\s\S]+)\)$/.exec(value);
    if (cm) {
      const parts = splitArgs(cm[1]);
      if (parts.length !== 2) throw new Error('color-mix 参数数量异常：' + value);
      const parsed = parts.map((part) => {
        const pm = /([\d.]+)%\s*$/.exec(part);
        const pct = pm ? Number(pm[1]) : null;
        const colorRaw = pm ? part.slice(0, pm.index).trim() : part;
        return { pct, colorRaw };
      });
      let [p1, p2] = [parsed[0].pct, parsed[1].pct];
      if (p1 === null && p2 === null) throw new Error('color-mix 缺少百分比：' + value);
      if (p1 === null) p1 = 100 - p2;
      if (p2 === null) p2 = 100 - p1;
      const c1 = evalColor(parsed[0].colorRaw, theme, mode, depth + 1);
      const c2 = evalColor(parsed[1].colorRaw, theme, mode, depth + 1);
      return mixColors(c1, p1, c2, p2);
    }
    throw new Error('无法求值的颜色表达式：' + value);
  }

  return function resolveColor(name, theme, mode) {
    const key = theme + '/' + mode + '/' + name;
    if (cache.has(key)) return cache.get(key);
    const found = lookupRaw(name, theme, mode);
    if (!found) throw new Error('未定义 token ' + name + '（' + theme + '/' + mode + '）');
    const color = evalColor(found.raw, theme, mode, 0);
    cache.set(key, color);
    return color;
  };
}

/** 解析长度/数值：12px / 0.75rem / 1 / 100ms / 2px。 */
function parseDimension(raw, kind) {
  const value = raw.trim();
  const num = /^([\d.]+)\s*(px|rem|ms|s)?$/.exec(value);
  if (!num) throw new Error('无法解析数值：' + raw);
  const n = Number(num[1]);
  const unit = num[2] || '';
  if (unit === 'rem') return n * 16;
  if (unit === 's') return n * 1000;
  if (kind === 'duration' && !unit) throw new Error('动效时长缺少单位：' + raw);
  return n;
}

function camelCase(name) {
  const parts = name.split('-');
  return parts[0] + parts.slice(1).map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join('');
}

// ────────────────────────────────────────────────────────────────────────────
// 构建 token 模型
// ────────────────────────────────────────────────────────────────────────────

export function buildTokenModel() {
  const themeVars = {};
  for (const theme of THEMES) themeVars[theme] = parseTheme(theme);
  const semanticVars = parseSimpleCss('semantic.css');
  const sharedVars = parseSimpleCss('shared.css');
  const registryVars = parseSimpleCss('theme-registry.css');
  const utilitiesVars = parseSimpleCss('utilities.css');
  // 页面进场 / 内容淡入 token 定义在 utilities.css 的组件规则里，不在 :root
  for (const key of ['--cst-page-enter-duration', '--cst-content-fade-duration']) {
    if (utilitiesVars[key] === undefined) throw new Error('utilities.css 缺少 ' + key);
    sharedVars[key] = utilitiesVars[key];
  }
  for (const key of Object.keys(registryVars)) {
    if (sharedVars[key] === undefined) sharedVars[key] = registryVars[key];
  }
  // theme-registry 的 --text-* / --radius-sheet 也放进 shared 查找面
  const resolveColor = createResolver(themeVars, semanticVars, sharedVars);

  const themes = {};
  for (const theme of THEMES) {
    themes[theme] = {};
    for (const mode of MODES) {
      const palette = {};
      for (const name of Object.keys(themeVars[theme][mode])) {
        if (!name.startsWith('--cst-palette-')) continue;
        if (name.startsWith('--cst-palette-opacity-')) continue; // 数值档，不是颜色
        const raw = themeVars[theme][mode][name];
        if (!/^(#|rgba?\(|color-mix\(|var\()/.test(raw)) continue; // 跳过 shadow-card/btn 等字符串
        palette[name.slice('--cst-palette-'.length)] = colorToHex(resolveColor(name, theme, mode));
      }
      // palette 补全：light 独有的颜色在 dark 里可能未覆写（例如 border-* 已覆写，但保守补全）
      for (const name of Object.keys(themeVars[theme].light)) {
        if (!name.startsWith('--cst-palette-') || name.startsWith('--cst-palette-opacity-')) continue;
        const key = name.slice('--cst-palette-'.length);
        if (palette[key] !== undefined) continue;
        const raw = themeVars[theme].light[name];
        if (!/^(#|rgba?\(|color-mix\(|var\()/.test(raw)) continue;
        palette[key] = colorToHex(resolveColor(name, theme, mode));
      }

      const semantic = {};
      for (const name of SEMANTIC_TOKENS) {
        semantic[name] = colorToHex(resolveColor('--cst-' + name, theme, mode));
      }

      const interaction = {};
      for (const name of INTERACTION_TOKENS) {
        const full = '--cst-' + name;
        const found = themeVars[theme][mode][full] ?? themeVars[theme].light[full];
        if (found === undefined) continue;
        interaction[name] = parseDimension(found, 'number');
      }

      // 氛围画布：dark 未覆写时按 CSS 级联回落到 light（如 inkwash 的 sun 恒为 transparent）
      const ambient = {};
      for (const name of AMBIENT_TOKENS) {
        ambient[name] = colorToHex(resolveColor('--cst-ambient-' + name, theme, mode));
      }

      themes[theme][mode] = {
        palette: sortObject(palette),
        semantic: sortObject(semantic),
        interaction: sortObject(interaction),
        ambient: sortObject(ambient),
      };
    }
  }

  const fontSize = {};
  const fontSizeTokens = {};
  for (const [name, token, fallbackRem] of FONT_SIZE_SPEC) {
    const raw = sharedVars[token] ?? registryVars[token];
    let px;
    if (raw !== undefined) px = parseDimension(raw, 'length');
    else if (fallbackRem !== null) px = fallbackRem * 16;
    else throw new Error('缺少字号 token：' + token);
    fontSize[name] = roundTiny(px);
    fontSizeTokens[name] = token;
  }

  const radius = {};
  const radiusTokens = {};
  for (const [name, token] of RADIUS_SPEC) {
    const raw = sharedVars[token] ?? registryVars[token];
    if (raw === undefined) throw new Error('缺少圆角 token：' + token);
    radius[name] = roundTiny(parseDimension(raw, 'length'));
    radiusTokens[name] = token;
  }

  const durations = {};
  const durationTokens = {};
  for (const [name, token] of MOTION_SPEC) {
    const raw = sharedVars[token] ?? registryVars[token];
    if (raw === undefined) throw new Error('缺少动效 token：' + token);
    durations[name] = Math.round(parseDimension(raw, 'duration'));
    durationTokens[name] = token;
  }

  const easings = {};
  for (const [name, token] of EASING_SPEC) {
    const raw = sharedVars[token];
    if (raw === undefined) throw new Error('缺少缓动 token：' + token);
    const m = /^cubic-bezier\(\s*([\d.+-]+)\s*,\s*([\d.+-]+)\s*,\s*([\d.+-]+)\s*,\s*([\d.+-]+)\s*\)$/.exec(raw.trim());
    if (!m) throw new Error('无法解析缓动曲线：' + raw);
    easings[name] = [Number(m[1]), Number(m[2]), Number(m[3]), Number(m[4])];
  }

  return {
    themes,
    typography: { fontSize: sortObject(fontSize), tokens: fontSizeTokens },
    radius: { values: sortObject(radius), tokens: radiusTokens },
    motion: { durations: sortObject(durations), tokens: durationTokens, easings },
  };
}

function sortObject(obj) {
  return Object.fromEntries(Object.entries(obj).sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)));
}

function roundTiny(n) {
  return Math.round(n * 1000) / 1000;
}

// ────────────────────────────────────────────────────────────────────────────
// 产物序列化
// ────────────────────────────────────────────────────────────────────────────

const HEADER = '自动生成，勿手改；重新生成：node scripts/design-tokens/gen-android.mjs';

export function buildTokensJson(model) {
  return JSON.stringify(
    {
      $comment: HEADER,
      $source: [
        'frontend/src/styles/theme/themes/{vibrant,forest,inkwash}.css',
        'frontend/src/styles/theme/semantic.css',
        'frontend/src/styles/theme/shared.css',
        'frontend/src/styles/theme/theme-registry.css',
        'frontend/src/styles/theme/utilities.css',
      ],
      $colorFormat: '#rrggbbaa（小写，alpha 在末位；color-mix 按 premultiplied-alpha 求值）',
      themes: model.themes,
      typography: model.typography.fontSize,
      typographyTokens: model.typography.tokens,
      radius: model.radius.values,
      radiusTokens: model.radius.tokens,
      motion: model.motion.durations,
      motionTokens: model.motion.tokens,
      easings: model.motion.easings,
    },
    null,
    2,
  ) + '\n';
}

function argbHex(hexRgba) {
  // #rrggbbaa → 0xAARRGGBB（Compose Color(long) 的入参形态）
  const rr = hexRgba.slice(1, 3).toUpperCase();
  const gg = hexRgba.slice(3, 5).toUpperCase();
  const bb = hexRgba.slice(5, 7).toUpperCase();
  const aa = hexRgba.slice(7, 9).toUpperCase();
  return '0x' + aa + rr + gg + bb;
}

function objectName(theme, mode) {
  return THEME_OBJECT[theme] + MODE_OBJECT[mode];
}

function buildColorGroup(lines, indent, group, model) {
  const pad = ' '.repeat(indent);
  const isSemantic = group === 'Semantic';
  const isAmbient = group === 'Ambient';
  // Semantic 组实现同包内的 SemanticPalette 接口（Theme.kt 用接口统一 6 组主题 × 明暗的取色）
  lines.push(pad + 'object ' + group + ' {');
  for (const theme of THEMES) {
    for (const mode of MODES) {
      const entries = model.themes[theme][mode][isAmbient ? 'ambient' : isSemantic ? 'semantic' : 'palette'];
      const interfaceName = isAmbient ? 'AmbientPalette' : isSemantic ? 'SemanticPalette' : 'PaletteColors';
      lines.push(pad + '    /** ' + THEME_OBJECT[theme] + ' · ' + mode + ' */');
      lines.push(pad + '    object ' + objectName(theme, mode) + ' : ' + interfaceName + ' {');
      for (const [name, value] of Object.entries(entries)) {
        const token = '--cst-' + (isSemantic ? '' : isAmbient ? 'ambient-' : 'palette-') + name;
        lines.push(
          pad + '        override val ' + camelCase(name) + ': Color = Color(' +
            argbHex(value) + ') // ' + token,
        );
      }
      lines.push(pad + '    }');
    }
  }
  lines.push(pad + '}');
}

export function buildTokensKotlin(model) {
  const lines = [];
  lines.push('// ' + HEADER);
  lines.push('// 数据源：frontend/src/styles/theme/（CSS token 是唯一权威，校验脚本见');
  lines.push('// frontend/scripts/check-cross-platform-android.mjs，漂移即 CI 红）。');
  lines.push('// 颜色注释里的 token 名与 Web 端一一对应；颜色值为 #AARRGGBB 的 Compose Color 入参。');
  lines.push('package com.hotspot.app.ui.theme');
  lines.push('');
  lines.push('import androidx.compose.animation.core.CubicBezierEasing');
  lines.push('import androidx.compose.ui.graphics.Color');
  lines.push('');
  lines.push('/** Web 端设计 token 的 Compose 镜像；由 scripts/design-tokens/gen-android.mjs 生成。 */');
  lines.push('object Tokens {');
  buildColorGroup(lines, 4, 'Palette', model);
  lines.push('');
  buildColorGroup(lines, 4, 'Semantic', model);
  lines.push('');
  buildColorGroup(lines, 4, 'Ambient', model);
  lines.push('');
  lines.push('    /** 字号档（sp）：10/11/12/14/18/20/24/30/36，对齐 design-system §4.1 */');
  lines.push('    object FontSize {');
  for (const [name, value] of Object.entries(model.typography.fontSize)) {
    lines.push('        const val ' + name + ': Float = ' + value + 'f // ' + model.typography.tokens[name]);
  }
  lines.push('    }');
  lines.push('');
  lines.push('    /** 圆角档：12/16/20/24/9999(胶囊)/28(sheet) */');
  lines.push('    object Radius {');
  for (const [name, value] of Object.entries(model.radius.values)) {
    lines.push('        const val ' + name + ': Float = ' + value + 'f // ' + model.radius.tokens[name]);
  }
  lines.push('    }');
  lines.push('');
  lines.push('    /** 动效常量（ms）与曲线：对齐 design-system §10.1 */');
  lines.push('    object Motion {');
  for (const [name, value] of Object.entries(model.motion.durations)) {
    lines.push('        const val ' + name + ': Int = ' + value + ' // ' + model.motion.tokens[name]);
  }
  for (const [name, points] of Object.entries(model.motion.easings)) {
    const token = EASING_SPEC.find(([n]) => n === name)[1];
    lines.push('        val ease' + name.charAt(0).toUpperCase() + name.slice(1) + ': CubicBezierEasing = CubicBezierEasing(' +
      points.map((p) => p + 'f').join(', ') + ') // ' + token);
  }
  lines.push('    }');
  lines.push('}');
  lines.push('');
  return lines.join('\n');
}

// ────────────────────────────────────────────────────────────────────────────
// CLI
// ────────────────────────────────────────────────────────────────────────────

function main() {
  const checkOnly = process.argv.includes('--check');
  const model = buildTokenModel();
  const tokensKotlin = buildTokensKotlin(model);
  const tokensJson = buildTokensJson(model);

  if (checkOnly) {
    const problems = [];
    for (const [path, expected] of [
      [TOKENS_KT, tokensKotlin],
      [TOKENS_JSON, tokensJson],
    ]) {
      let actual;
      try {
        actual = readFileSync(path, 'utf8');
      } catch {
        problems.push('缺少生成产物：' + path);
        continue;
      }
      if (actual !== expected) problems.push('产物已过期（重新运行生成器）：' + path);
    }
    if (problems.length > 0) {
      for (const p of problems) console.error('  - ' + p);
      console.error('Android 设计 token 产物与 CSS 源不一致（node scripts/design-tokens/gen-android.mjs 重新生成）。');
      process.exit(1);
    }
    console.log('Android 设计 token 产物最新 OK（Tokens.kt + design-tokens.json 与 CSS 源逐值一致）。');
    return;
  }

  mkdirSync(dirname(TOKENS_KT), { recursive: true });
  mkdirSync(dirname(TOKENS_JSON), { recursive: true });
  writeFileSync(TOKENS_KT, tokensKotlin);
  writeFileSync(TOKENS_JSON, tokensJson);
  const colorCount = THEMES.length * MODES.length;
  const paletteCount = Object.keys(model.themes.vibrant.light.palette).length;
  const semanticCount = Object.keys(model.themes.vibrant.light.semantic).length;
  console.log(
    '已生成 Android 设计 token：' + colorCount + ' 组主题 × 明暗，palette ' + paletteCount + ' 项、语义色 ' +
      semanticCount + ' 项、字号 ' + Object.keys(model.typography.fontSize).length + ' 档、圆角 ' +
      Object.keys(model.radius.values).length + ' 档、动效 ' + Object.keys(model.motion.durations).length + ' 档。',
  );
  console.log('  → ' + TOKENS_KT);
  console.log('  → ' + TOKENS_JSON);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (error) {
    console.error('生成 Android 设计 token 失败：' + error.message);
    process.exit(1);
  }
}
