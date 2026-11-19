#!/usr/bin/env node
/**
 * 跨端设计 token 校验（Web CSS ↔ Android）
 *
 * 三重防线（任何一层漂移即 exit 1）：
 *   1. frontend/src/styles/theme/**.css（唯一权威）↔ androidapp/app/src/main/assets/design-tokens.json
 *   2. design-tokens.json ↔ androidapp/.../ui/theme/Tokens.kt
 *   3. 产物完整性：3 主题 × 2 明暗、token 集合齐全、颜色统一小写 #rrggbbaa
 *
 * 由 bun run check:theme 末尾串联（见 frontend/package.json），CI 与本地同一入口。
 *
 * 用法：node frontend/scripts/check-cross-platform-android.mjs
 */
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { buildTokenModel, REPO_ROOT, THEMES, MODES } from '../../scripts/design-tokens/gen-android.mjs';

const TOKENS_JSON = join(REPO_ROOT, 'androidapp/app/src/main/assets/design-tokens.json');
const TOKENS_KT = join(REPO_ROOT, 'androidapp/app/src/main/java/com/hotspot/app/ui/theme/Tokens.kt');
const THEME_OBJECT = { vibrant: 'Vibrant', forest: 'Forest', inkwash: 'Inkwash' };
const MODE_OBJECT = { light: 'Light', dark: 'Dark' };

const problems = [];
function fail(message) {
  problems.push(message);
}

function readIfExists(path, label) {
  try {
    return readFileSync(path, 'utf8');
  } catch {
    fail('缺少 Android 产物：' + label + '（先运行 node scripts/design-tokens/gen-android.mjs）');
    return null;
  }
}

function report() {
  if (problems.length > 0) {
    console.error('Android 设计 token 与 Web CSS 不一致（漂移即 CI 红）：');
    for (const p of problems) console.error('  - ' + p);
    console.error(
      '修正方向：以 frontend/src/styles/theme/*.css 为唯一真值，运行 node scripts/design-tokens/gen-android.mjs 重新生成，勿手改 Tokens.kt / design-tokens.json。',
    );
    process.exit(1);
  }
  const paletteCount = Object.keys(json.themes.vibrant.light.palette).length;
  const semanticCount = Object.keys(json.themes.vibrant.light.semantic).length;
  console.log(
    'Android 设计 token 与 Web CSS 逐值一致 OK：' +
      THEMES.length + ' 主题 × ' + MODES.length + ' 明暗，palette ' + paletteCount + ' 项、语义色 ' + semanticCount +
      ' 项、字号 ' + Object.keys(json.typography).length + ' 档、圆角 ' + Object.keys(json.radius).length +
      ' 档、动效 ' + Object.keys(json.motion).length + ' 档（Tokens.kt ↔ design-tokens.json ↔ CSS）。',
  );
}

// ────────────────────────────────────────────────────────────────────────────
// 1. CSS → 期望模型
// ────────────────────────────────────────────────────────────────────────────
const model = buildTokenModel();
const jsonText = readIfExists(TOKENS_JSON, 'androidapp/app/src/main/assets/design-tokens.json');
const ktText = readIfExists(TOKENS_KT, 'androidapp/app/src/main/java/com/hotspot/app/ui/theme/Tokens.kt');
let json = null;
if (jsonText !== null) {
  try {
    json = JSON.parse(jsonText);
  } catch (error) {
    fail('design-tokens.json 不是合法 JSON：' + error.message);
  }
}
if (json === null || ktText === null) report();

// ────────────────────────────────────────────────────────────────────────────
// 2. CSS ↔ JSON
// ────────────────────────────────────────────────────────────────────────────
function assertObjectKeysEqual(label, expected, actual) {
  const expectedKeys = Object.keys(expected).sort();
  const actualKeys = Object.keys(actual ?? {}).sort();
  const missing = expectedKeys.filter((k) => !actualKeys.includes(k));
  const extra = actualKeys.filter((k) => !expectedKeys.includes(k));
  if (missing.length) fail(label + ' 缺少 token：' + missing.join(', '));
  if (extra.length) fail(label + ' 多出未知 token：' + extra.join(', '));
}

function assertValuesEqual(label, expected, actual) {
  for (const [key, value] of Object.entries(expected)) {
    if (actual === undefined || actual[key] === undefined) continue; // 缺键已由集合校验报告
    if (actual[key] !== value) fail(label + '.' + key + '：JSON ' + actual[key] + ' ≠ CSS 求值 ' + value);
  }
}

const COLOR_RE = /^#[0-9a-f]{8}$/;
function assertColorsNormalized(label, palette) {
  for (const [key, value] of Object.entries(palette ?? {})) {
    if (typeof value !== 'string' || !COLOR_RE.test(value)) {
      fail(label + '.' + key + ' 颜色格式非法（应为小写 #rrggbbaa）：' + value);
    }
  }
}

for (const theme of THEMES) {
  for (const mode of MODES) {
    const expected = model.themes[theme][mode];
    const actual = json.themes && json.themes[theme] ? json.themes[theme][mode] : undefined;
    if (!actual) {
      fail('design-tokens.json 缺少主题组合 ' + theme + '/' + mode);
      continue;
    }
    assertObjectKeysEqual(theme + '/' + mode + ' palette', expected.palette, actual.palette);
    assertValuesEqual(theme + '/' + mode + ' palette', expected.palette, actual.palette);
    assertObjectKeysEqual(theme + '/' + mode + ' semantic', expected.semantic, actual.semantic);
    assertValuesEqual(theme + '/' + mode + ' semantic', expected.semantic, actual.semantic);
    assertObjectKeysEqual(theme + '/' + mode + ' ambient', expected.ambient, actual.ambient);
    assertValuesEqual(theme + '/' + mode + ' ambient', expected.ambient, actual.ambient);
    assertObjectKeysEqual(theme + '/' + mode + ' interaction', expected.interaction, actual.interaction);
    for (const [key, value] of Object.entries(expected.interaction)) {
      if (!actual.interaction || actual.interaction[key] !== value) {
        fail(theme + '/' + mode + ' interaction.' + key + ' 与 CSS 不一致');
      }
    }
    assertColorsNormalized(theme + '/' + mode + ' palette', actual.palette);
    assertColorsNormalized(theme + '/' + mode + ' semantic', actual.semantic);
  }
}

// 字号 / 圆角 / 动效：JSON ↔ CSS
for (const [key, value] of Object.entries(model.typography.fontSize)) {
  if (json.typography[key] !== value) fail('typography.' + key + '：JSON ' + json.typography[key] + ' ≠ CSS ' + value + 'px');
  if (json.typographyTokens[key] !== model.typography.tokens[key]) fail('typographyTokens.' + key + ' 来源 token 不一致');
}
for (const [key, value] of Object.entries(model.radius.values)) {
  if (json.radius[key] !== value) fail('radius.' + key + '：JSON ' + json.radius[key] + ' ≠ CSS ' + value + 'px');
  if (json.radiusTokens[key] !== model.radius.tokens[key]) fail('radiusTokens.' + key + ' 来源 token 不一致');
}
for (const [key, value] of Object.entries(model.motion.durations)) {
  if (json.motion[key] !== value) fail('motion.' + key + '：JSON ' + json.motion[key] + ' ≠ CSS ' + value + 'ms');
  if (json.motionTokens[key] !== model.motion.tokens[key]) fail('motionTokens.' + key + ' 来源 token 不一致');
}
for (const [key, points] of Object.entries(model.motion.easings)) {
  const actual = json.easings[key];
  if (!Array.isArray(actual) || actual.length !== 4 || actual.some((v, i) => v !== points[i])) {
    fail('easings.' + key + '：JSON ' + JSON.stringify(actual) + ' ≠ CSS ' + JSON.stringify(points));
  }
}

// ────────────────────────────────────────────────────────────────────────────
// 3. JSON ↔ Tokens.kt
// ────────────────────────────────────────────────────────────────────────────
if (!ktText.startsWith('// 自动生成，勿手改')) {
  fail('Tokens.kt 缺少「自动生成，勿手改」头注释（禁止手改生成物）');
}

function camelCase(name) {
  const parts = name.split('-');
  return parts[0] + parts.slice(1).map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join('');
}

const kt = {
  palette: {},
  semantic: {},
  ambient: {},
  fontSize: {},
  radius: {},
  motion: {},
  easings: {},
};

let section = null;
let colorBlock = null;
for (const rawLine of ktText.split('\n')) {
  const line = rawLine.trim();
  const indent = rawLine.length - rawLine.trimStart().length;
  const sectionMatch = /^object (Palette|Semantic|Ambient|FontSize|Radius|Motion) \{/.exec(line);
  if (sectionMatch) {
    section = sectionMatch[1];
    colorBlock = null;
    continue;
  }
  const blockMatch = /^object (VibrantLight|VibrantDark|ForestLight|ForestDark|InkwashLight|InkwashDark)(?: : \w+)? \{/.exec(line);
  if (blockMatch) {
    colorBlock = blockMatch[1];
    if (section === 'Palette') kt.palette[colorBlock] = {};
    if (section === 'Semantic') kt.semantic[colorBlock] = {};
    if (section === 'Ambient') kt.ambient[colorBlock] = {};
    continue;
  }
  if (line === '}' && indent === 8 && colorBlock) {
    colorBlock = null;
    continue;
  }

  const colorMatch = /^(?:override )?val (\w+): Color = Color\((0x[0-9A-F]{8})\)(?:\s*\/\/\s*(--cst-[\w-]+))?$/.exec(line);
  if (colorMatch) {
    const name = colorMatch[1];
    const argb = colorMatch[2];
    const tokenComment = colorMatch[3];
    if (!colorBlock || !section) {
      fail('Tokens.kt 颜色行缺少所属分组：' + line);
      continue;
    }
    const rgba = '#' + argb.slice(4, 10).toLowerCase() + argb.slice(2, 4).toLowerCase();
    const bucket = section === 'Palette' ? kt.palette : section === 'Semantic' ? kt.semantic : kt.ambient;
    bucket[colorBlock][name] = { rgba, token: tokenComment || null };
    continue;
  }

  const numberMatch = /^const val (\w+): (Float|Int) = (-?[\d.]+)f?(?:\s*\/\/\s*(--[\w-]+))?$/.exec(line);
  if (numberMatch) {
    const name = numberMatch[1];
    if (section === 'FontSize') kt.fontSize[name] = Number(numberMatch[3]);
    else if (section === 'Radius') kt.radius[name] = Number(numberMatch[3]);
    else if (section === 'Motion') kt.motion[name] = Number(numberMatch[3]);
    else fail('Tokens.kt 数值行位置非法：' + line);
    continue;
  }

  const easingMatch = /^val (ease\w+): CubicBezierEasing = CubicBezierEasing\(([^)]+)\)(?:\s*\/\/\s*(--cst-[\w-]+))?$/.exec(line);
  if (easingMatch) {
    const name = easingMatch[1];
    const points = easingMatch[2].split(',').map((s) => Number(s.trim().replace(/f$/, '')));
    kt.easings[name] = { points, token: easingMatch[3] || null };
  }
}

function compareColorGroup(groupName, expectedThemes, ktGroup, prefix) {
  for (const theme of THEMES) {
    for (const mode of MODES) {
      const blockName = THEME_OBJECT[theme] + MODE_OBJECT[mode];
      const expected =
        groupName === 'Palette'
          ? expectedThemes[theme][mode].palette
          : groupName === 'Semantic'
            ? expectedThemes[theme][mode].semantic
            : expectedThemes[theme][mode].ambient;
      const actual = ktGroup[blockName];
      if (!actual) {
        fail('Tokens.kt 缺少 ' + groupName + ' 块 ' + blockName);
        continue;
      }
      for (const [jsonKey, rgba] of Object.entries(expected)) {
        const name = camelCase(jsonKey);
        const entry = actual[name];
        if (!entry) {
          fail('Tokens.kt ' + groupName + '.' + blockName + ' 缺少 ' + name + '（' + prefix + jsonKey + '）');
          continue;
        }
        if (entry.rgba !== rgba) {
          fail('Tokens.kt ' + groupName + '.' + blockName + '.' + name + ' = ' + entry.rgba + ' ≠ design-tokens.json ' + rgba);
        }
        const expectedToken = prefix + jsonKey;
        if (entry.token !== expectedToken) {
          fail('Tokens.kt ' + groupName + '.' + blockName + '.' + name + ' 注释 token ' + entry.token + ' ≠ ' + expectedToken);
        }
      }
    }
  }
}

compareColorGroup('Palette', json.themes, kt.palette, '--cst-palette-');
compareColorGroup('Semantic', json.themes, kt.semantic, '--cst-');
compareColorGroup('Ambient', json.themes, kt.ambient, '--cst-ambient-');

for (const [key, expected] of Object.entries(json.typography)) {
  if (!kt.fontSize[key]) fail('Tokens.kt FontSize 缺少 ' + key);
  else if (kt.fontSize[key] !== expected) fail('Tokens.kt FontSize.' + key + ' = ' + kt.fontSize[key] + ' ≠ JSON ' + expected);
}
for (const [key, expected] of Object.entries(json.radius)) {
  if (!kt.radius[key]) fail('Tokens.kt Radius 缺少 ' + key);
  else if (kt.radius[key] !== expected) fail('Tokens.kt Radius.' + key + ' = ' + kt.radius[key] + ' ≠ JSON ' + expected);
}
for (const [key, expected] of Object.entries(json.motion)) {
  if (!kt.motion[key]) fail('Tokens.kt Motion 缺少 ' + key);
  else if (kt.motion[key] !== expected) fail('Tokens.kt Motion.' + key + ' = ' + kt.motion[key] + ' ≠ JSON ' + expected);
}
for (const [key, expected] of Object.entries(json.easings)) {
  const ktName = 'ease' + key.charAt(0).toUpperCase() + key.slice(1);
  const entry = kt.easings[ktName];
  if (!entry) {
    fail('Tokens.kt Motion 缺少缓动 ' + ktName);
    continue;
  }
  if (entry.points.length !== 4 || entry.points.some((v, i) => v !== expected[i])) {
    fail('Tokens.kt 缓动 ' + key + ' = ' + JSON.stringify(entry.points) + ' ≠ JSON ' + JSON.stringify(expected));
  }
}

// 完整性：6 组主题 × 明暗全部具备
for (const theme of THEMES) {
  for (const mode of MODES) {
    if (!json.themes[theme] || !json.themes[theme][mode]) fail('design-tokens.json 缺少 ' + theme + '/' + mode);
    const blockName = THEME_OBJECT[theme] + MODE_OBJECT[mode];
    if (!kt.palette[blockName]) fail('Tokens.kt Palette 缺少 ' + blockName);
    if (!kt.semantic[blockName]) fail('Tokens.kt Semantic 缺少 ' + blockName);
    if (!kt.ambient[blockName]) fail('Tokens.kt Ambient 缺少 ' + blockName);
  }
}

report();
