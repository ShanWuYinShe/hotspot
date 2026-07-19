# 主题重命名 + 删除 sand + 全扁平重构 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把现有 3 主题(`indigo`/`forest`/`sand`)收敛为 2 主题(`mist` 岚 / `grove` 林),删除 sand 主题,把所有 neumorphic 双边框设计改为扁平 border-only + 单方向 elevation shadow。

**Architecture:** 6 批独立可回滚。批次 1-2 是主题系统重塑(TypeScript + i18n + CSS 选择器 + sand 块删除);批次 3-4 是 token + class 层扁平化(`--shadow-*` 重定义、Type A class 改 border-only、死类删除);批次 5 是 250 处内联 neumorphic 清理(按文件聚集度分子批);批次 6 是 `<style>` 块 + 主题预览卡 + 残余 grep 验证。

**Tech Stack:** SvelteKit 5 + Tailwind v4(config 在 `app.css` 的 `@theme {}`)+ CSS variables 设计 token 系统 + mode-watcher(明暗模式)。

## Global Constraints

- **契约权威**:`spec/` 是 API 契约唯一权威——本计划不改任何接口/字段/错误码。
- **token-only**:`frontend/AGENTS.md` 设计规范要求"禁止硬编码十六进制色值"。例外:`PREVIEW_ACCENT` 这类"展示用色卡"必须直接对齐真实 token 值。
- **主题命名映射**(verbatim):
  - 老 `indigo` → 新 `mist`,中文 label "岚",英文 "Mist"
  - 老 `forest` → 新 `grove`,中文 label "林",英文 "Grove"
  - 老 `sand` → 完全删除
  - 默认主题:`mist`
- **localStorage 重置**:`THEME_MIGRATION = {}`,老值(`indigo`/`forest`/`sand`/`social_blue`/`default_clean`/`figma_neu`/`vibrant`/`neu`)落回默认 `mist`,写回新值。
- **新 token 值**(全 4 主题块:mist/grove × light/dark):
  - `--theme-shadow-light: rgba(0, 0, 0, 0);`(置 0)
  - `--theme-shadow-dark: rgba(0, 0, 0, 0);`(置 0)
  - `--shadow-elevation`:
    - mist light `rgba(99, 102, 241, 0.08)` / mist dark `rgba(0, 0, 0, 0.32)`
    - grove light `rgba(27, 31, 27, 0.06)` / grove dark `rgba(0, 0, 0, 0.36)`
- **新 `--shadow-*` 公式**:
  - `--shadow-sm: 0 1px 2px var(--shadow-elevation);`
  - `--shadow-md: 0 2px 6px var(--shadow-elevation);`
  - `--shadow-lg: 0 4px 12px var(--shadow-elevation);`
  - `--shadow-xl: 0 8px 24px var(--shadow-elevation);`
  - `--shadow-inset: inset 0 1px 2px var(--shadow-elevation);`
- **构建校验**:每批结束必须运行 `bun check`(svelte-check,基线 7 errors)+ `bun run build` + `bun lint`(基线 49 warnings)。`package.json` 无 `test` 脚本。
- **触控 hover 守卫**:`app.css:34` 把 Tailwind `hover:` variant 重定义为 `(&:hover, @media (hover: hover))`——内联 `hover:*` 在触控设备自动失效。
- **commit 粒度**:每 task 一次 commit,中文 conventional commits 风格(`修复(UI):` / `重构(CSS):` / `重构(主题):`)。

## 当前工作树状态

- HEAD:`959ae474 文档(spec): 主题重命名(岚/林) + 删除 sand + 全扁平重构设计`
- 工作树干净(无 untracked / modified)
- 上一轮工作(hover 统一 + sand 方块修复)已全部 commit
- 当前分支:`main`

## File Structure

修改分三层:

- **`src/lib/stores/theme.svelte.ts`**:TypeScript 主题状态(`THEME_FAMILIES` / `THEME_FAMILY_OPTIONS` / `DEFAULT_THEME_FAMILY` / `THEME_MIGRATION` / `THEME_FAMILY_INIT_SCRIPT`)
- **`src/app.html`**:首屏阻塞脚本(硬编码副本,必须与 theme.svelte.ts 同步)
- **`src/lib/i18n/locales/zh-CN.ts` + `en.ts`**:主题 label + description key 重命名
- **`src/app.css`**:CSS 主题块重命名(`indigo→mist` / `forest→grove` / 删 `sand`)+ `--shadow-*` 重定义 + Type A class 扁平化 + 死类删除
- **`src/lib/components/**/*.svelte` + `src/routes/**/*.svelte`**:250 处内联 `box-shadow: var(--shadow-*)` 清理 + 5 个 `<style>` 块 + settings 主题预览卡重写

---

## Task 1: TypeScript 主题状态重命名(indigo→mist, forest→grove, 删 sand)

**Files:**
- Modify: `frontend/src/lib/stores/theme.svelte.ts:9-58`(整个常量定义段)

**Interfaces:**
- Consumes: 无
- Produces:`THEME_FAMILIES = ['mist', 'grove'] as const`、`ThemeFamily = 'mist' | 'grove'`、`DEFAULT_THEME_FAMILY = 'mist'`、`THEME_FAMILY_OPTIONS` 用新 labelKey/descriptionKey(`sidebar.mist`/`sidebar.mistDesc`/`sidebar.grove`/`sidebar.groveDesc`)、`THEME_MIGRATION = {}`、`THEME_FAMILY_INIT_SCRIPT` 自动跟随常量

注意:`app.html` 的硬编码脚本副本**本 task 不动**(Task 2 处理)。本 task 完成后,dev/build 仍可运行,但首屏可能用老值(因 `app.html` 仍写 `indigo`)——这是预期,Task 2 修。

- [ ] **Step 1: 读出当前 theme.svelte.ts 全文**

```bash
cat /Users/liyou/work/hotspot/frontend/src/lib/stores/theme.svelte.ts
```

- [ ] **Step 2: 用 Edit 替换 L9-58 整段常量定义**

把:

```typescript
export const THEME_FAMILIES = ['indigo', 'forest', 'sand'] as const;

export type ThemeFamily = (typeof THEME_FAMILIES)[number];

export const THEME_FAMILY_OPTIONS: {
  family: ThemeFamily;
  labelKey: TranslationKey;
  descriptionKey: TranslationKey;
}[] = [
  {
    family: 'indigo',
    labelKey: 'sidebar.indigo',
    descriptionKey: 'sidebar.indigoDesc'
  },
  {
    family: 'forest',
    labelKey: 'sidebar.forest',
    descriptionKey: 'sidebar.forestDesc'
  },
  {
    family: 'sand',
    labelKey: 'sidebar.sand',
    descriptionKey: 'sidebar.sandDesc'
  }
];

const THEME_FAMILY_STORAGE_KEY = 'hotspot_theme_family';
const DEFAULT_THEME_FAMILY: ThemeFamily = 'indigo';

/** `<html>` 上用于选择当前主题族的属性名 */
export const THEME_FAMILY_ATTRIBUTE = 'data-theme-family';

/** Map old theme names to current names */
const THEME_MIGRATION: Record<string, ThemeFamily> = {
  social_blue: 'indigo',
  default_clean: 'forest',
  figma_neu: 'sand',
  vibrant: 'indigo',
  neu: 'sand'
};
```

改为:

```typescript
export const THEME_FAMILIES = ['mist', 'grove'] as const;

export type ThemeFamily = (typeof THEME_FAMILIES)[number];

export const THEME_FAMILY_OPTIONS: {
  family: ThemeFamily;
  labelKey: TranslationKey;
  descriptionKey: TranslationKey;
}[] = [
  {
    family: 'mist',
    labelKey: 'sidebar.mist',
    descriptionKey: 'sidebar.mistDesc'
  },
  {
    family: 'grove',
    labelKey: 'sidebar.grove',
    descriptionKey: 'sidebar.groveDesc'
  }
];

const THEME_FAMILY_STORAGE_KEY = 'hotspot_theme_family';
const DEFAULT_THEME_FAMILY: ThemeFamily = 'mist';

/** `<html>` 上用于选择当前主题族的属性名 */
export const THEME_FAMILY_ATTRIBUTE = 'data-theme-family';

// 主题重命名(indigo→mist, forest→grove)+ 删除 sand 后,老 localStorage 值
// 一律落回默认(mist),不做语义迁移 —— 见 docs/specs/2026-07-19-theme-rename-and-flatten-design.md §1.A
const THEME_MIGRATION: Record<string, ThemeFamily> = {};
```

- [ ] **Step 3: 修复 `theme-toggle.svelte:24` 的 fallback**

`frontend/src/lib/components/common/theme-toggle.svelte:24` 把:

```typescript
    const nextFamily = themeFamilyCycle[(currentIndex + 1) % themeFamilyCycle.length] ?? 'forest';
```

改为:

```typescript
    const nextFamily = themeFamilyCycle[(currentIndex + 1) % themeFamilyCycle.length] ?? 'mist';
```

- [ ] **Step 4: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -15
```

预期:`bun check` 会报 i18n key 错误(`sidebar.mist`/`sidebar.grove` 还没定义,settings 里 `themeNameMist` 等也还没定义)——这是预期的,**下一个 Task 修**。只要 `bun check` 总错误数增加幅度合理(<15 个新错误,都是 i18n key 缺失)即可继续。如果出现 `ThemeFamily` 类型错误(比如某处仍 hardcode `'indigo'`),修这些类型错误。

```bash
bun run build 2>&1 | tail -5
```

预期:`bun run build` 成功(SvelteKit build 不检查 i18n key 存在性)。

- [ ] **Step 5: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/stores/theme.svelte.ts frontend/src/lib/components/common/theme-toggle.svelte
git commit -m "重构(主题): TypeScript 主题状态 indigo→mist / forest→grove, 删除 sand

- THEME_FAMILIES: ['indigo','forest','sand'] → ['mist','grove']
- DEFAULT_THEME_FAMILY: 'indigo' → 'mist'
- THEME_FAMILY_OPTIONS: labelKey/descriptionKey 改 sidebar.mist/grove
- THEME_MIGRATION: 清空(老值落回默认,不做语义迁移)
- theme-toggle.svelte fallback 'forest' → 'mist'

本 task 不动 app.html(硬编码副本,Task 2 处理);i18n key 也是 Task 2 处理。
本 task 完成后 bun check 会报 i18n key 缺失错误,属预期。"
```

---

## Task 2: i18n key 重命名 + 描述文案改写

**Files:**
- Modify: `frontend/src/lib/i18n/locales/zh-CN.ts:34-39, 220-225`
- Modify: `frontend/src/lib/i18n/locales/en.ts:34-39, 220-225`
- Modify: `frontend/src/routes/(protected)/settings/+page.svelte:28-44`(`PREVIEW_ACCENT` / `THEME_NAMES` / `THEME_DESC`)

**Interfaces:**
- Consumes: Task 1 的新 `ThemeFamily = 'mist' | 'grove'`
- Produces:`sidebar.mist`/`sidebar.mistDesc`/`sidebar.grove`/`sidebar.groveDesc`/`settings.themeNameMist`/`settings.themeDescMist`/`settings.themeNameGrove`/`settings.themeDescGrove` 在 i18n 中存在;`PREVIEW_ACCENT` / `THEME_NAMES` / `THEME_DESC` 用新 key

- [ ] **Step 1: 读 zh-CN.ts 的主题段(L34-39)和 themeName/Desc 段(L218-226)**

```bash
sed -n '33,40p;218,227p' /Users/liyou/work/hotspot/frontend/src/lib/i18n/locales/zh-CN.ts
```

- [ ] **Step 2: Edit zh-CN.ts 主题段(L34-39)**

把:

```typescript
    indigo: '活力靛蓝',
    forest: '清爽森林',
    sand: '暖砂陶瓷',
    indigoDesc: '靛蓝紫色调，现代活力，发光点缀',
    forestDesc: '清爽、中性、稳妥的绿色基调',
    sandDesc: '暖砂底色，柔和阴影，陶瓷质感'
```

改为:

```typescript
    mist: '岚',
    grove: '林',
    mistDesc: '山顶雾气，冷调清亮，如清晨远眺',
    groveDesc: '山林深处，沉静妥帖，如午后林间'
```

- [ ] **Step 3: Edit zh-CN.ts themeName/Desc 段(L220-225)**

把:

```typescript
    themeNameIndigo: '活力',
    themeNameForest: '自然',
    themeNameSand: '质感',
    themeDescIndigo: '霓虹活力',
    themeDescForest: '森林极简',
    themeDescSand: '柔和质感'
```

改为:

```typescript
    themeNameMist: '岚',
    themeNameGrove: '林',
    themeDescMist: '山顶雾气，冷调清亮',
    themeDescGrove: '山林深处，沉静妥帖'
```

- [ ] **Step 4: Edit en.ts 同步(L34-39, L220-225)**

把 en.ts 对应段改为:

```typescript
    mist: 'Mist',
    grove: 'Grove',
    mistDesc: 'Cool, luminous mist over mountain peaks',
    groveDesc: 'Quiet, settled shade in deep mountain woods'
```

和:

```typescript
    themeNameMist: 'Mist',
    themeNameGrove: 'Grove',
    themeDescMist: 'Cool mist over peaks',
    themeDescGrove: 'Deep mountain grove'
```

- [ ] **Step 5: Edit `settings/+page.svelte` PREVIEW_ACCENT(L28-32)**

把:

```typescript
  const PREVIEW_ACCENT: Record<ThemeFamily, string> = {
    indigo: '#6366f1',
    forest: '#3D7A55',
    sand: '#b86f4b'
  };
```

改为:

```typescript
  const PREVIEW_ACCENT: Record<ThemeFamily, string> = {
    mist: '#6366f1',
    grove: '#3d7a55'
  };
```

- [ ] **Step 6: Edit `settings/+page.svelte` THEME_NAMES(L34-38)**

把:

```typescript
  const THEME_NAMES = $derived<Record<ThemeFamily, string>>({
    indigo: t('settings.themeNameIndigo'),
    forest: t('settings.themeNameForest'),
    sand: t('settings.themeNameSand')
  });
```

改为:

```typescript
  const THEME_NAMES = $derived<Record<ThemeFamily, string>>({
    mist: t('settings.themeNameMist'),
    grove: t('settings.themeNameGrove')
  });
```

- [ ] **Step 7: Edit `settings/+page.svelte` THEME_DESC(L40-44)**

把:

```typescript
  const THEME_DESC = $derived<Record<ThemeFamily, string>>({
    indigo: t('settings.themeDescIndigo'),
    forest: t('settings.themeDescForest'),
    sand: t('settings.themeDescSand')
  });
```

改为:

```typescript
  const THEME_DESC = $derived<Record<ThemeFamily, string>>({
    mist: t('settings.themeDescMist'),
    grove: t('settings.themeDescGrove')
  });
```

- [ ] **Step 8: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -15
```

预期:`bun check` 错误数回到基线 7 个(都是预先存在的 i18n 类型问题,与本 task 无关)。如果还有 `themeNameIndigo`/`sidebar.indigo` 等老 key 被引用,说明有遗漏——grep 全局:

```bash
grep -rn "themeNameIndigo\|themeNameForest\|themeNameSand\|themeDescIndigo\|themeDescForest\|themeDescSand\|sidebar\.indigo\|sidebar\.forest\|sidebar\.sand" src
```

预期:0 匹配(全部已重命名)。

```bash
bun run build 2>&1 | tail -5
```

预期:成功。

- [ ] **Step 9: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/i18n/ frontend/src/routes/\(protected\)/settings/+page.svelte
git commit -m "重构(主题): i18n key indigo→mist / forest→grove, 描述改诗意意象

zh-CN:
- sidebar.indigo/forest/sand → sidebar.mist/grove(删 sand)
- 描述: '活力靛蓝/清爽森林/暖砂陶瓷' → '岚'/'林'
- 新描述(mistDesc/groveDesc): '山顶雾气,冷调清亮,如清晨远眺' / '山林深处,沉静妥帖,如午后林间'
- themeNameIndigo/Forest/Sand → themeNameMist/Grove(删 Sand)
- themeDesc 同步

en: 同步意译

settings/+page.svelte:
- PREVIEW_ACCENT indigo/forest/sand → mist/grove(删 sand)
- THEME_NAMES / THEME_DESC 用新 i18n key"
```

---

## Task 3: app.html 同步 + 主题切换器验证

**Files:**
- Modify: `frontend/src/app.html:4, 11-32`

**Interfaces:**
- Consumes: Task 1 的新 `ThemeFamily = 'mist' | 'grove'`、`DEFAULT_THEME_FAMILY = 'mist'`、`THEME_MIGRATION = {}`、`THEME_FAMILIES = ['mist', 'grove']`
- Produces:`app.html` 首屏阻塞脚本与新 TypeScript 常量一致;首屏不再闪老 `indigo` 主题

- [ ] **Step 1: Edit `app.html:4` html 标签默认主题**

把:

```html
<html lang="zh-CN" data-theme-family="indigo">
```

改为:

```html
<html lang="zh-CN" data-theme-family="mist">
```

- [ ] **Step 2: Edit `app.html:11-32` 阻塞脚本 config**

把:

```javascript
      })({
        storageKey: 'hotspot_theme_family',
        attribute: 'data-theme-family',
        families: ['indigo', 'forest', 'sand'],
        migration: {
          social_blue: 'indigo',
          default_clean: 'forest',
          figma_neu: 'sand',
          vibrant: 'indigo',
          neu: 'sand'
        },
        defaultFamily: 'indigo'
      });
```

改为:

```javascript
      })({
        storageKey: 'hotspot_theme_family',
        attribute: 'data-theme-family',
        families: ['mist', 'grove'],
        migration: {},
        defaultFamily: 'mist'
      });
```

- [ ] **Step 3: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

预期:基线 7 errors,build 成功。

- [ ] **Step 4: 手动验证(本地启动 dev server)**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

打开 `http://localhost:5181/` 在 Chrome DevTools 里:
1. `document.documentElement.getAttribute('data-theme-family')` 应返回 `'mist'`
2. 切换主题按钮:cycle 应在 `mist` ↔ `grove` 之间(2 个值),不再有 `sand`
3. localStorage `hotspot_theme_family` 模拟老值 `'indigo'` → 刷新 → 应自动落回 `'mist'`(因 migration 为空)

杀 dev:

```bash
pkill -f "vite.*5181" || true
```

注意:此时 CSS 还没重命名(下个 Task),所以 `data-theme-family='mist'` 会落到 `:root,` 默认块(原 indigo 别名,L312),视觉是 indigo 紫色——这是预期。Task 4 会把 CSS 选择器改对。

- [ ] **Step 5: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.html
git commit -m "修复(主题): app.html 首屏脚本同步新主题常量(mist/grove)

- data-theme-family 默认 'indigo' → 'mist'
- families: ['indigo','forest','sand'] → ['mist','grove']
- migration 清空(老值落回默认)
- defaultFamily: 'indigo' → 'mist'

app.html 的脚本是与 theme.svelte.ts THEME_FAMILY_INIT_SCRIPT 同源的
硬编码副本(避免首屏 FOUC),Task 1 改了 ts 后必须同步 html。"
```

---

## Task 4: CSS 主题块重命名(indigo→mist, forest→grove, 删 sand)

**Files:**
- Modify: `frontend/src/app.css:130, 220, 312-313, 404, 496-676, 865-898, 1190-1230, 1928-1944`(及任何 `[data-theme-family='*']` 选择器)

**Interfaces:**
- Consumes: Task 1-3 的新主题值
- Produces:`app.css` 所有主题选择器用 `mist`/`grove`,sand 块整段删除;此时主题视觉正式生效

注意:本 task 只做"重命名选择器 + 删 sand 块",**不动 token 值**(shadow 等下批处理)。重命名后视觉与之前等价(只是 `data-theme-family` 值改了)。

- [ ] **Step 1: grep 所有主题选择器位置**

```bash
cd /Users/liyou/work/hotspot
grep -n "data-theme-family" frontend/src/app.css
```

记录所有出现的行号(预期 ~15-20 处)。

- [ ] **Step 2: 删除 sand light + dark 主题块(L496-586 + L588-676)**

读 L494-680 确认边界,然后用 Edit 删除整段。从注释:

```css
/* ═══════════════════════════════════════════════════════════════════════
   Theme: sand — 暖砂 warm-sand ceramic (default theme)
   ...
   ═══════════════════════════════════════════════════════════════════════ */
:root[data-theme-family='sand'] { ... }

:root[data-theme-family='sand'].dark { ... }
```

完整删除(两个块 + 头部注释),保留前后的空行结构。

- [ ] **Step 3: 重命名 indigo → mist(4 处主题块 + 默认块)**

用 Edit replace_all(谨慎——只能用唯一字符串,不能用通用 `'indigo'`):

L312-313:
```css
:root,
:root[data-theme-family='indigo'] {
```
改为:
```css
:root,
:root[data-theme-family='mist'] {
```

L404:
```css
:root[data-theme-family='indigo'].dark {
```
改为:
```css
:root[data-theme-family='mist'].dark {
```

注释 L307-310 的 `Theme: indigo` 描述也改为 `Theme: mist`。

- [ ] **Step 4: 重命名 forest → grove(2 处主题块)**

L130:
```css
:root[data-theme-family='forest'] {
```
改为:
```css
:root[data-theme-family='grove'] {
```

L220:
```css
:root[data-theme-family='forest'].dark {
```
改为:
```css
:root[data-theme-family='grove'].dark {
```

注释 L126-128 的 `Theme: forest` 描述改为 `Theme: grove`。

- [ ] **Step 5: 重命名所有 `[data-theme-family='forest']` / `'indigo'` 选择器(L865+)**

L865-873(forest surface-hero override):
```css
  [data-theme-family='forest'] .surface-hero,
  [data-theme-family='forest'] .neu-surface-hero,
  [data-theme-family='forest'].dark .surface-hero,
  [data-theme-family='forest'].dark .neu-surface-hero {
```
改为:
```css
  [data-theme-family='grove'] .surface-hero,
  [data-theme-family='grove'] .neu-surface-hero,
  [data-theme-family='grove'].dark .surface-hero,
  [data-theme-family='grove'].dark .neu-surface-hero {
```

L1190-1230(`.ambient-glow::before` 各主题 opacity 覆盖):用 grep 定位每处,把 `'forest'` → `'grove'`、`'indigo'` → `'mist'`、删除所有 `'sand'` 的选择器组。具体看每段处理:

- forest 段(L1190-1193)的 4 个 `'forest'` selector → `'grove'`
- indigo light 段(L1196-1197)的 2 个 `'indigo'` selector → `'mist'`
- indigo dark 段(L1200-1201)的 2 个 `'indigo'` selector → `'mist'`
- **sand 段(L1213-1216 等,之前批次加的)** 整段删除

L1928-1944(tablet glass-bg override):`':root[data-theme-family='indigo']:not(.dark)'` → `'mist'`,`'[data-theme-family='forest']:not(.dark)'` → `'grove'`,`'[data-theme-family='sand']:not(.dark)'` → 删除整段。

- [ ] **Step 6: grep 验证无残留**

```bash
cd /Users/liyou/work/hotspot
grep -n "'indigo'\|'forest'\|'sand'" frontend/src/app.css
```

预期:0 匹配。

```bash
grep -n "data-theme-family" frontend/src/app.css
```

预期:所有 selector 都用 `'mist'` 或 `'grove'`。

- [ ] **Step 7: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

预期:基线 7 errors,build 成功。

- [ ] **Step 8: 手动视觉验证**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

打开浏览器 → 主题切换器循环 `mist ↔ grove`:
- mist:紫色 indigo 调(原 indigo 视觉)
- grove:绿色 forest 调(原 forest 视觉)
- sand 不再可选

杀 dev:

```bash
pkill -f "vite.*5181" || true
```

- [ ] **Step 9: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "重构(主题): CSS 主题块 indigo→mist / forest→grove, 删除 sand

- :root[data-theme-family='indigo'] → 'mist'(默认 :root 别名同步)
- :root[data-theme-family='forest'] → 'grove'
- 删除 :root[data-theme-family='sand'](light + dark 整块,含注释)
- 所有 .ambient-glow / .surface-hero override 选择器同步重命名
- 删除所有 'sand' 主题专属 CSS 规则

本 task 仅重命名选择器,不动 token 值;视觉等价于改名前的 indigo/forest。"
```

---

## Task 5: `--shadow-*` token 重定义为单方向 elevation + `--theme-shadow-*` 置 0

**Files:**
- Modify: `frontend/src/app.css:130-305`(grove light + dark 块)
- Modify: `frontend/src/app.css:312-491`(mist light + dark 块)
- Modify: `frontend/src/app.css:700-720`(`:root` 共享 token 块的 shadow preset)

**Interfaces:**
- Consumes: Task 4 的 CSS 主题块重命名
- Produces:`--theme-shadow-light/dark` 全置 0(拆除 neumorphic),`--shadow-sm/md/lg/xl/inset` 改单方向公式,新增 `--shadow-elevation` token(每主题族块定义)

**关键效果**:本 task 后,所有现存内联 `var(--shadow-*)` 引用(250+ 处)**自动扁平化**——虽然视觉变化大(neumorphic 双向阴影变单方向柔和阴影),但不会破坏布局或可用性。

- [ ] **Step 1: 在 4 个主题族块加 `--shadow-elevation` token**

每个主题块(grove light L130、grove dark L220、mist light L312、mist dark L404)的 `--theme-shadow-light:` 那行**之前**插入:

grove light(L133 附近,`--theme-shadow-light: rgba(255, 255, 255, 0);` 之前):
```css
  --shadow-elevation: rgba(27, 31, 27, 0.06);
```

grove dark(L223 附近,`--theme-shadow-light: rgba(255, 255, 255, 0);` 之前):
```css
  --shadow-elevation: rgba(0, 0, 0, 0.36);
```

mist light(L316 附近,`--theme-shadow-light: rgba(255, 255, 255, 1);` 之前):
```css
  --shadow-elevation: rgba(99, 102, 241, 0.08);
```

mist dark(L407 附近,`--theme-shadow-light: rgba(129, 140, 248, 0.07);` 之前):
```css
  --shadow-elevation: rgba(0, 0, 0, 0.32);
```

- [ ] **Step 2: 把所有 4 个主题块的 `--theme-shadow-light/dark` 置 0**

grove light L133-134(原已是 0):
```css
  --theme-shadow-light: rgba(255, 255, 255, 0);
  --theme-shadow-dark: rgba(0, 0, 0, 0);
```
**保持不变**(grove 已经是 0)。

grove dark L223-224(原已是 0):**保持不变**。

mist light L316-317(原是 `rgba(255, 255, 255, 1)` / `rgba(99, 91, 200, 0.12)`):
改为:
```css
  --theme-shadow-light: rgba(0, 0, 0, 0);
  --theme-shadow-dark: rgba(0, 0, 0, 0);
```

mist dark L407-408(原是 `rgba(129, 140, 248, 0.07)` / `rgba(0, 0, 0, 0.72)`):
改为:
```css
  --theme-shadow-light: rgba(0, 0, 0, 0);
  --theme-shadow-dark: rgba(0, 0, 0, 0);
```

- [ ] **Step 3: 重定义 `--shadow-sm/md/lg/xl/inset` preset(L733-737 附近)**

在 `:root` 共享 token 块内,把:

```css
  --shadow-inset: inset 6px 6px 12px var(--shadow-dark), inset -6px -6px 12px var(--shadow-light);
  --shadow-sm: 4px 4px 10px var(--shadow-dark), -4px -4px 10px var(--shadow-light);
  --shadow-md: 5px 5px 10px var(--shadow-dark), -5px -5px 10px var(--shadow-light);
  --shadow-lg: 8px 8px 20px var(--shadow-dark), -8px -8px 20px var(--shadow-light);
  --shadow-xl: 10px 10px 20px var(--shadow-dark), -10px -10px 20px var(--shadow-light);
```

改为:

```css
  /* 单方向 elevation shadow(向下,柔和)。
     用于卡片悬浮、模态、下拉菜单等需要"浮起"反馈的元素。
     取代之前的 neumorphic 双向 split shadow。 */
  --shadow-inset: inset 0 1px 2px var(--shadow-elevation);
  --shadow-sm: 0 1px 2px var(--shadow-elevation);
  --shadow-md: 0 2px 6px var(--shadow-elevation);
  --shadow-lg: 0 4px 12px var(--shadow-elevation);
  --shadow-xl: 0 8px 24px var(--shadow-elevation);
```

- [ ] **Step 4: mist 主题 `--hover-shadow-card/row/btn/pill` 也改单方向**

mist light(L360-365 附近)和 mist dark(L454-459 附近)的:

```css
  --hover-shadow-card:
    ...neumorphic split...;
```

替换为单方向柔和浮起:

mist light:
```css
  --hover-shadow-card: 0 4px 12px var(--shadow-elevation), 0 0 0 1px rgba(99, 102, 241, 0.07);
  --hover-shadow-row: 0 2px 6px var(--shadow-elevation);
  --hover-shadow-btn: 0 2px 6px rgba(99, 102, 241, 0.18), 0 1px 2px rgba(99, 102, 241, 0.12);
  --hover-shadow-pill: 0 2px 6px rgba(99, 102, 241, 0.16), 0 1px 2px rgba(99, 102, 241, 0.1);
```

mist dark:
```css
  --hover-shadow-card: 0 6px 16px var(--shadow-elevation), 0 0 0 1px rgba(129, 140, 248, 0.1);
  --hover-shadow-row: 0 2px 8px var(--shadow-elevation);
  --hover-shadow-btn: 0 2px 8px rgba(129, 140, 248, 0.22), 0 1px 2px rgba(0, 0, 0, 0.2);
  --hover-shadow-pill: 0 2px 6px rgba(129, 140, 248, 0.2), 0 1px 2px rgba(0, 0, 0, 0.18);
```

grove 的 hover-shadow 已经是单方向(L174-177 / L263-266),**保持不变**。

- [ ] **Step 5: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

预期:基线 7 errors,build 成功。

- [ ] **Step 6: 手动视觉验证(关键)**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

打开浏览器,mist + grove × light + dark 共 4 个组合,访问关键页面(首页、附近的人、动态、聊天、设置):
1. 卡片现在是单方向柔和浮起阴影,无 neumorphic 双向 halo
2. 输入框边框单薄,无 inset 凹陷
3. 列表 hover 用 `.list-row` 的 background-tint(上一轮已实现),不受 shadow 改动影响
4. **重点**:确认没有任何残留"方块色块"或"奶白光晕"

如有问题,微调 `--shadow-elevation` 透明度。

杀 dev:

```bash
pkill -f "vite.*5181" || true
```

- [ ] **Step 7: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "重构(CSS): --shadow-* 重定义为单方向 elevation, --theme-shadow-* 全置 0

拆除 neumorphic 双向 split shadow 系统:
- 4 主题块(grove/mist × light/dark)新增 --shadow-elevation token
- 所有 --theme-shadow-light/dark 置 rgba(0,0,0,0)(mist 也置 0,grove 本已 0)
- --shadow-sm/md/lg/xl/inset 改为 0 Npx Mpx var(--shadow-elevation) 单方向公式
- mist 的 --hover-shadow-card/row/btn/pill 改单方向柔和浮起

效果:250+ 处内联 var(--shadow-*) 引用自动扁平化(任务 6+ 逐个清理),
视觉从 neumorphic 双向阴影转为 Apple HIG 风格的单方向柔和浮起。"
```

---

## Task 6: Type A class 扁平化 + 死类删除

**Files:**
- Modify: `frontend/src/app.css:857-1100`(整个 surface/input utility 段)
- Modify: `frontend/src/lib/components/layout/app-shell.svelte:56`(.neu-flat → .surface-flat)
- Modify: `frontend/src/routes/(protected)/chat/+page.svelte:197`(.neu-flat → .surface-flat)
- Modify: `frontend/src/routes/(protected)/chat/users/[id]/+page.svelte:412`(.neu-flat → .surface-flat)

**Interfaces:**
- Consumes: Task 5 的 `--shadow-*` 单方向公式
- Produces:`.surface-hero` 改 border-only、`.surface-pressed` 改 border + 略深 bg、`.input-surface`/`.input-base`/`.field-inset` 改 border-only + focus glow ring、`.card-hover:hover` 改单方向;死类(`.surface-raised`/`.neu-flat`/`.surface-convex`/`.button-glow`/`.control-small`/`.icon-badge`)删除

注意:`.neu-flat` 在 3 处 .svelte 用了,先迁移到 `.surface-flat` 再删类定义。

- [ ] **Step 1: 把 3 处 `.neu-flat` 迁移到 `.surface-flat`**

读每处上下文确认 class 字符串:

```bash
grep -n "neu-flat" /Users/liyou/work/hotspot/frontend/src/lib/components/layout/app-shell.svelte /Users/liyou/work/hotspot/frontend/src/routes/\(protected\)/chat/+page.svelte /Users/liyou/work/hotspot/frontend/src/routes/\(protected\)/chat/users/\[id\]/+page.svelte
```

每处的 `class="... neu-flat ..."` 改成 `class="... surface-flat ..."`(把 `neu-flat` 替换为 `surface-flat`,其他不变)。

- [ ] **Step 2: app.css `.surface-hero` 改 border-only(L882-898)**

把:

```css
  .surface-hero,
  .neu-surface-hero {
    background-color: var(--surface-bg);
    border: 1px solid var(--border-theme);
    box-shadow:
      8px 8px 18px var(--shadow-dark),
      -8px -8px 18px var(--shadow-light);
  }
  [data-theme-family='grove'] .surface-hero,
  [data-theme-family='grove'] .neu-surface-hero,
  [data-theme-family='grove'].dark .surface-hero,
  [data-theme-family='grove'].dark .neu-surface-hero {
    /* forest 扁平无 neumorphic 阴影,border 是唯一边界,用 border-active
       (24% 对比度)而非默认的 border-theme(13%)确保清晰可见。
       保持 border 属性(非 box-shadow),避免圆角处渲染断裂。 */
    border-color: var(--border-active);
  }
```

改为(扁平化后两主题一致,删 grove override):

```css
  /* surface-hero: 概览/主张卡(.surface-flat 的语义别名)。
     全扁平重构后与 .surface-flat 等价;border + hover border-active。
     保留类名以兼容既有组件引用,逐步迁移到 .surface-flat。 */
  .surface-hero,
  .neu-surface-hero,
  .surface-flat,
  .neu-surface-flat {
    background-color: var(--surface-bg);
    border: 1px solid var(--border-theme);
    transition:
      border-color 0.18s ease,
      background-color 0.18s ease;
  }
  .surface-hero:hover,
  .neu-surface-hero:hover,
  .surface-flat:hover,
  .neu-surface-flat:hover {
    border-color: var(--border-active);
  }
```

然后**删除原 `.surface-flat, .neu-surface-flat { ... }` 单独定义**(L899-910,已合并进上面)。

- [ ] **Step 3: app.css `.surface-pressed` 改 border + 略深 bg(L946-955)**

把:

```css
  .surface-pressed,
  .neu-surface-pressed,
  .neu-pressed {
    background-color: var(--surface-bg);
    box-shadow:
      inset 8px 8px 16px var(--shadow-dark),
      inset -8px -8px 16px var(--shadow-light);
    color: var(--text-primary);
    border: 1px solid var(--border-active);
  }
```

改为:

```css
  /* surface-pressed: 凹陷面板(用更深 border + 略深 bg 表达"压下去",
     不再用 inset neumorphic shadow)。 */
  .surface-pressed,
  .neu-surface-pressed,
  .neu-pressed {
    background-color: color-mix(in srgb, var(--surface-bg) 88%, var(--page-bg));
    color: var(--text-primary);
    border: 1px solid var(--border-active);
  }
```

- [ ] **Step 4: app.css `.input-surface` 改 border-only + focus glow ring(L968-992)**

把整段:

```css
  .input-surface,
  .neu-input-surface {
    box-shadow:
      5px 5px 10px var(--shadow-dark),
      -5px -5px 10px var(--shadow-light);
    border: 1px solid var(--border-theme);
    transition:
      box-shadow 0.2s cubic-bezier(0.22, 1, 0.36, 1),
      border-color 0.2s ease;
  }
  .input-surface:focus-within,
  .neu-input-surface:focus-within {
    box-shadow:
      inset 5px 5px 10px var(--shadow-dark),
      inset -5px -5px 10px var(--shadow-light);
    border-color: var(--border-active);
  }
  .input-surface:focus-within:focus-visible,
  .neu-input-surface:focus-within:focus-visible {
    box-shadow:
      inset 5px 5px 10px var(--shadow-dark),
      inset -5px -5px 10px var(--shadow-light),
      0 0 0 2px var(--glow);
    border-color: var(--brand);
  }
```

改为:

```css
  /* input-surface: 输入容器(border-only,focus 时 brand border + glow ring)。 */
  .input-surface,
  .neu-input-surface {
    border: 1px solid var(--border-theme);
    transition:
      border-color 0.18s ease,
      box-shadow 0.18s ease;
  }
  .input-surface:focus-within,
  .neu-input-surface:focus-within {
    border-color: var(--brand);
    box-shadow: 0 0 0 2px var(--glow);
  }
```

- [ ] **Step 5: app.css `.input-base` 同样改(L994-1026)**

把整段 `.input-base, .neu-input-base { ... }` 及 `:focus` / `:focus-visible` 改为:

```css
  .input-base,
  .neu-input-base {
    border: 1px solid var(--border-theme);
    padding: 6px 12px;
    font-size: 14px;
    transition:
      border-color 0.18s ease,
      box-shadow 0.18s ease;
  }
  .input-base::placeholder,
  .neu-input-base::placeholder {
    color: var(--text-secondary);
    opacity: 0.6;
  }
  .input-base:focus,
  .neu-input-base:focus {
    border-color: var(--brand);
    box-shadow: 0 0 0 2px var(--glow);
    outline: none;
  }
  .input-base:focus-visible,
  .neu-input-base:focus-visible {
    border-color: var(--brand);
    box-shadow: 0 0 0 2px var(--glow);
    outline: none;
  }
```

- [ ] **Step 6: app.css `.field-inset` 改 border-only(L1031-1061)**

把整段(保留 `.field-error` 变体):

```css
  .field-inset {
    border: 1px solid var(--border-theme);
    padding: 6px 12px;
    font-size: 14px;
    background-color: color-mix(in srgb, var(--surface-bg) 92%, var(--page-bg));
    transition:
      border-color 0.18s ease,
      box-shadow 0.18s ease;
  }
  .field-inset::placeholder {
    color: var(--text-secondary);
    opacity: 0.6;
  }
  .field-inset:focus,
  .field-inset:focus-visible {
    border-color: var(--brand);
    box-shadow: 0 0 0 2px var(--glow);
    outline: none;
  }
  .field-inset.field-error {
    border-color: var(--error);
  }
  .field-inset.field-error:focus,
  .field-inset.field-error:focus-visible {
    border-color: var(--error);
    box-shadow: 0 0 0 2px var(--error-subtle);
  }
```

- [ ] **Step 7: app.css `.card-hover:hover` 改单方向(L1100-1116 附近)**

把:

```css
  .card-hover,
  .neu-card-hover {
    transition:
      transform 0.2s cubic-bezier(0.22, 1, 0.36, 1),
      box-shadow 0.26s cubic-bezier(0.22, 1, 0.36, 1),
      outline 0.2s cubic-bezier(0.22, 1, 0.36, 1);
  }
  .card-hover:hover,
  .neu-card-hover:hover {
    box-shadow: var(--hover-shadow-card) !important;
  }
  .card-hover:focus-visible,
  .neu-card-hover:focus-visible {
    box-shadow: var(--hover-shadow-card) !important;
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

改为(全主题统一单方向):

```css
  /* card-hover: 卡片悬浮反馈(全主题统一单方向 elevation + 缩放)。
     组件需自行加 hover:scale-[1.02] 表达浮起。 */
  .card-hover,
  .neu-card-hover {
    transition:
      transform 0.2s cubic-bezier(0.22, 1, 0.36, 1),
      box-shadow 0.26s cubic-bezier(0.22, 1, 0.36, 1),
      outline 0.2s cubic-bezier(0.22, 1, 0.36, 1);
  }
  .card-hover:hover,
  .neu-card-hover:hover {
    box-shadow: var(--shadow-md) !important;
  }
  .card-hover:focus-visible,
  .neu-card-hover:focus-visible {
    box-shadow: var(--shadow-md) !important;
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

- [ ] **Step 8: 删除死类定义**

- `.surface-raised, .neu-surface-raised, .neu-flat { ... }`(L935-944)— 已无引用(Step 1 迁移了 3 处 .neu-flat)
- `.surface-convex, .neu-surface-convex, .neu-convex { ... }`(L957-965)— 0 引用
- `.button-glow, .neu-button-glow { ... }` 和 `.dark .button-glow, .dark .neu-button-glow { ... }`(L1064-1076)— 0 引用
- `.control-small, .neu-small { ... }`(L1079-1087)— 0 引用
- `.icon-badge, .neu-icon-badge { ... }`(L1090-1095)— 0 引用

每段整段删除。

- [ ] **Step 9: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
bun lint 2>&1 | tail -5
```

预期:基线 7 errors,build 成功,lint 基线 49。

- [ ] **Step 10: 手动视觉验证**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

跨 4 主题组合验证:
1. 卡片 hover 用 `--shadow-md`(单方向柔和浮起)
2. 输入框 focus 用 brand border + glow ring(无 inset 凹陷)
3. `.surface-pressed`(user-card square variant、chat send 按钮)用更深 border + bg 表达凹陷
4. 没有任何 neumorphic 双向 halo

杀 dev。

- [ ] **Step 11: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css frontend/src/lib/components/ frontend/src/routes/
git commit -m "重构(CSS): Type A class 扁平化 + 死类删除

- .surface-hero 改 border-only(等同 .surface-flat),删 grove 特殊 override
- .surface-pressed 改更深 border + 略深 bg 表达凹陷(去 inset shadow)
- .input-surface / .input-base 改 border-only + focus brand border + glow ring
- .field-inset 改 border-only + focus glow
- .card-hover:hover 改 var(--shadow-md) 单方向(全主题统一)
- 删除死类: .surface-raised / .neu-flat / .surface-convex / .button-glow / 
  .control-small / .icon-badge(均 0 引用)
- 3 处 .neu-flat 引用迁移到 .surface-flat(app-shell, chat +page, chat users/[id])

全扁平重构:所有保留的 surface/input class 都是 border-only 或 border+focus glow,
不再有 neumorphic 双向 split shadow。"
```

---

## Task 7: 组件内联清理 — `lib/components/common/*` 第一批(modals + dialogs)

**Files:**
- Modify: `frontend/src/lib/components/common/confirm-dialog.svelte:50, 59`
- Modify: `frontend/src/lib/components/common/create-group-modal.svelte:231, 237, 254, 283, 311, 338, 379, 422`
- Modify: `frontend/src/lib/components/common/create-moment-modal.svelte:528, 806`
- Modify: `frontend/src/lib/components/common/create-location-share-modal.svelte:99, 107, 122, 174, 189, 204`
- Modify: `frontend/src/lib/components/common/mobile-create-sheet.svelte:43, 48, 64, 69`
- Modify: `frontend/src/lib/components/common/search-modal.svelte:310, 316, 374, 413, 420, 445, 451, 500, 529, 536, 551, 577`
- Modify: `frontend/src/lib/components/common/anchor-summary-card.svelte:29, 56`
- Modify: `frontend/src/lib/components/common/auth-form-content.svelte:226, 298`
- Modify: `frontend/src/lib/components/common/list-skeleton.svelte:22`
- Modify: `frontend/src/lib/components/common/map-card.svelte:101`
- Modify: `frontend/src/lib/components/common/page-header.svelte:27`
- Modify: `frontend/src/lib/components/common/empty-state.svelte:37`
- Modify: `frontend/src/lib/components/common/theme-toggle.svelte:32, 44`

**Interfaces:**
- Consumes: Task 6 的扁平 class 系统
- Produces:上述 14 个文件的内联 `box-shadow: var(--shadow-*)` 全部移除,保留 `border` + `background`

**通用规则**(每个 inline style 都遵循):
- 删除 `box-shadow: var(--shadow-sm|md|lg|xl|inset);` 整段子句(包括尾随的 `;` 和它前后的空格)
- 保留 `border: 1px solid var(--border-theme);`(或 `--theme-error-border` / `--glass-border` 等变体)
- 保留 `background: var(--surface-bg);`(或其他合法 background)
- 保留 `width / height / padding / color` 等其他声明
- 若同一 inline style 同时有 `box-shadow` 和 `border`,只删 `box-shadow`

- [ ] **Step 1: 读出每个文件的 inline style 上下文**

逐文件 cat 出 target line 附近 ±3 行,确认确切的 style 字符串。可以用一个命令批量预览:

```bash
for f in confirm-dialog create-group-modal create-moment-modal create-location-share-modal mobile-create-sheet search-modal anchor-summary-card auth-form-content list-skeleton map-card page-header empty-state theme-toggle; do
  echo "=== $f ==="
  grep -n "box-shadow: var(--shadow-" /Users/liyou/work/hotspot/frontend/src/lib/components/common/$f.svelte 2>/dev/null
done
```

- [ ] **Step 2: 用 Edit 逐个文件清理**

对每个文件,读完整 inline style 字符串,用 Edit 删除 `box-shadow: var(--shadow-*);` 子句。

举例 `confirm-dialog.svelte:50`:

之前:
```svelte
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); box-shadow: var(--shadow-lg); padding: 24px; ..."
```

之后:
```svelte
    style="background: var(--surface-bg); border: 1px solid var(--border-theme); padding: 24px; ..."
```

注意保留分号结构正确(不要留下 `;;` 或缺分号)。

- [ ] **Step 3: 单独处理 `map-card.svelte:101`(用 `--glass-border`)**

读 L99-103 上下文,删 `box-shadow: var(--shadow-md);` 子句,保留 `border: 1px solid var(--glass-border);`。

- [ ] **Step 4: 单独处理 `theme-toggle.svelte:32, 44`(刚在 Task 10 改过的)**

L32 + L44 都是 `style="...; box-shadow: var(--shadow-md); border: 1px solid var(--border-theme)"` 模式。删 box-shadow 子句。

- [ ] **Step 5: grep 验证本批文件清理干净**

```bash
cd /Users/liyou/work/hotspot/frontend
for f in confirm-dialog create-group-modal create-moment-modal create-location-share-modal mobile-create-sheet search-modal anchor-summary-card auth-form-content list-skeleton map-card page-header empty-state theme-toggle; do
  count=$(grep -c "box-shadow: var(--shadow-" src/lib/components/common/$f.svelte 2>/dev/null || echo 0)
  if [ "$count" != "0" ]; then
    echo "REMAINING in $f: $count"
    grep -n "box-shadow: var(--shadow-" src/lib/components/common/$f.svelte
  fi
done
```

预期:无输出(全部清理干净)。

- [ ] **Step 6: 构建校验**

```bash
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
bun lint 2>&1 | tail -5
```

- [ ] **Step 7: 手动视觉验证**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

打开模态/对话框(发布动态、创建群组、搜索、位置选择、主题切换、登录表单等),确认:
1. 所有面板/卡片仅 border + background,无 neumorphic 阴影
2. 输入框 focus 用 brand border + glow ring
3. 主题切换器按钮 hover 用 `.icon-btn-hover`/`.pill-hover`(单方向浮起)

杀 dev。

- [ ] **Step 8: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/common/
git commit -m "重构(UI): lib/components/common 内联 box-shadow 清理(14 文件)

移除 modals/dialogs/sheets/buttons 的内联 box-shadow: var(--shadow-*),
保留 border + background + 其他声明。涉及:
- confirm-dialog, create-group-modal, create-moment-modal, 
  create-location-share-modal, mobile-create-sheet (modals/sheets)
- search-modal (12 处)
- anchor-summary-card, auth-form-content, list-skeleton, map-card, 
  page-header, empty-state, theme-toggle (其他 common 组件)

视觉上从 neumorphic 双向阴影变为纯 border 扁平化。"
```

---

## Task 8: 组件内联清理 — `lib/components/ui/*` + `lib/components/layout/*` + `lib/components/chat/*`

**Files:**
- Modify: `frontend/src/lib/components/ui/dialog/dialog-content.svelte:76, 85`
- Modify: `frontend/src/lib/components/ui/dropdown-menu/dropdown-menu-content.svelte:26`
- Modify: `frontend/src/lib/components/ui/dropdown-menu/dropdown-menu-sub-content.svelte:20`
- Modify: `frontend/src/lib/components/ui/tabs/tabs-list.svelte:16`
- Modify: `frontend/src/lib/components/layout/page-header.svelte:27`(若 Task 7 未覆盖)
- Modify: `frontend/src/lib/components/chat/chat-list-panel.svelte`(若有 inline shadow)
- Modify: `frontend/src/lib/components/chat/chat-bubble.svelte`(若有 inline shadow)

**Interfaces:**
- Consumes: Task 6 的扁平 class 系统
- Produces:UI primitive 组件(dialog/dropdown/tabs)的内联 neumorphic 清理

- [ ] **Step 1: 扫描 ui/ + layout/ + chat/ 的所有 inline `var(--shadow-*)`**

```bash
cd /Users/liyou/work/hotspot/frontend
grep -rn "box-shadow: var(--shadow-" src/lib/components/ui src/lib/components/layout src/lib/components/chat --include="*.svelte"
```

记录所有 file:line。

- [ ] **Step 2: 逐文件 Edit 删除 box-shadow 子句**

每个文件按 Task 7 的规则处理:删 `box-shadow: var(--shadow-*);`,保留 `border` + 其他。

特别注意:
- `tabs-list.svelte:16` 用的是 `--shadow-inset`,同样删除
- `dialog-content.svelte` 的 box-shadow 可能与 `<style>` 块混用,只删 inline,不动 `<style>` 块(Task 11 处理 `<style>` 块)

- [ ] **Step 3: grep 验证**

```bash
grep -rn "box-shadow: var(--shadow-" src/lib/components/ui src/lib/components/layout src/lib/components/chat --include="*.svelte"
```

预期:0 匹配。

- [ ] **Step 4: 构建校验 + 视觉验证 + commit**

```bash
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/ui frontend/src/lib/components/layout frontend/src/lib/components/chat
git commit -m "重构(UI): ui/layout/chat 内联 box-shadow 清理

- dialog-content, dropdown-menu-content, dropdown-menu-sub-content, tabs-list
- layout/page-header, chat/chat-list-panel, chat/chat-bubble

删除所有 var(--shadow-*) 内联,保留 border。"
```

---

## Task 9: 组件内联清理 — `routes/(admin)/admin/*/*`

**Files:**
- Modify: `frontend/src/routes/(admin)/+layout.svelte:62`
- Modify: `frontend/src/routes/(admin)/admin/+page.svelte:115, 119, 141, 157, 175`
- Modify: `frontend/src/routes/(admin)/admin/announcements/+page.svelte:164, 228`
- Modify: `frontend/src/routes/(admin)/admin/audit/+page.svelte:45, 60, 83, 125, 129, 170`
- Modify: `frontend/src/routes/(admin)/admin/config/+page.svelte:114, 131, 159, 208, 258`
- Modify: `frontend/src/routes/(admin)/admin/groups/+page.svelte:81, 96, 119, 161, 246`
- Modify: `frontend/src/routes/(admin)/admin/rbac/+page.svelte:172, 196, 213, 241, 245, 293, 338, 368, 435`
- Modify: `frontend/src/routes/(admin)/admin/reports/+page.svelte:80, 106, 123, 153, 159, 259`
- Modify: `frontend/src/routes/(admin)/admin/users/+page.svelte:96, 110, 133, 175, 276`

**Interfaces:**
- Consumes: Task 6 的扁平 class 系统
- Produces:所有 admin 页面(stat cards / row cards / icon badges / action buttons)的 inline neumorphic 清理

- [ ] **Step 1: 扫描所有 admin 文件**

```bash
cd /Users/liyou/work/hotspot/frontend
grep -rn "box-shadow: var(--shadow-" "src/routes/(admin)" --include="*.svelte"
```

确认总数(预期 ~25 处)。

- [ ] **Step 2: 逐文件 Edit 清理**

按 Task 7 规则。注意 admin 页面 pattern 很重复(card icon / row button / row icon),可以用 `replace_all: true` 配合精确的子串(但要确保子串唯一,不影响 brand-glow `--shadow-brand-btn` 引用)。

- [ ] **Step 3: grep 验证**

```bash
grep -rn "box-shadow: var(--shadow-" "src/routes/(admin)" --include="*.svelte"
```

预期:0 匹配(但 `--shadow-brand-btn` 引用应保留,不要误删——它不在 `var(--shadow-` 范围内,因它是 `var(--shadow-brand-btn)` 也匹配这个 prefix!需要区分)。

**重要修正**:grep pattern `box-shadow: var(--shadow-` 同时匹配 `var(--shadow-sm/md/lg/xl/inset)` 和 `var(--shadow-brand-btn)`。但 brand-btn 不在本任务范围(它是单方向 glow,不是 neumorphic)。

更精确的 grep:

```bash
grep -rn "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" "src/routes/(admin)" --include="*.svelte"
```

- [ ] **Step 4: 构建校验 + commit**

```bash
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

```bash
cd /Users/liyou/work/hotspot
git add "frontend/src/routes/(admin)"
git commit -m "重构(UI): routes/(admin) 内联 box-shadow 清理(~25 处)

admin 所有页面: stat cards / row cards / icon badges / action buttons
移除 box-shadow: var(--shadow-sm|md|lg|xl),保留 border + background。
brand-glow shadow (--shadow-brand-btn) 保留不动。"
```

---

## Task 10: 组件内联清理 — `routes/(protected)/settings/*/*`

**Files:**
- Modify: `frontend/src/routes/(protected)/settings/+page.svelte:179, 367, 372, 399, 428, 437, 475, 581, 611, 658`
- Modify: `frontend/src/routes/(protected)/settings/notifications/+page.svelte:157, 162, 205, 222, 243`
- Modify: `frontend/src/routes/(protected)/settings/password/+page.svelte:105`
- Modify: `frontend/src/routes/(protected)/settings/privacy/+page.svelte:94, 109, 189, 242`
- Modify: `frontend/src/routes/(protected)/settings/reports/+page.svelte:78, 91, 156`
- Modify: `frontend/src/routes/(protected)/settings/sessions/+page.svelte:96, 105, 123, 149, 154, 181`
- Modify: `frontend/src/routes/(protected)/settings/shares/+page.svelte:100, 151, 193, 201`

**Interfaces:**
- Consumes: Task 6 的扁平 class 系统
- Produces:所有 settings 页面 row cards / cards / icons 的 inline neumorphic 清理

**重要**:Task 10 不动 `settings/+page.svelte:296, 512` 的主题预览卡 neumorphic——那是 Task 12 的工作(主题预览卡扁平化重写)。

- [ ] **Step 1: 扫描**

```bash
cd /Users/liyou/work/hotspot/frontend
grep -rn "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" "src/routes/(protected)/settings" --include="*.svelte"
```

- [ ] **Step 2: 逐文件 Edit 清理(同 Task 7 规则)**

- [ ] **Step 3: grep 验证**

```bash
grep -rn "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" "src/routes/(protected)/settings" --include="*.svelte"
```

预期:只剩 `settings/+page.svelte:296, 512`(主题预览卡,Task 12 处理)。

- [ ] **Step 4: 构建校验 + commit**

```bash
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

```bash
cd /Users/liyou/work/hotspot
git add "frontend/src/routes/(protected)/settings"
git commit -m "重构(UI): routes/(protected)/settings 内联 box-shadow 清理(~20 处)

settings 主页 row cards / 各子页面(password/privacy/sessions/shares/reports/notifications)
的 row cards / icons 移除 box-shadow: var(--shadow-*),保留 border。

主题预览卡(L296/512)的 neumorphic 由 Task 12 单独重写为扁平预览。"
```

---

## Task 11: 组件内联清理 — `routes/(public)/groups/[id]/+page.svelte`(48 处)

**Files:**
- Modify: `frontend/src/routes/(public)/groups/[id]/+page.svelte`(48 处 inline shadow,L901-1922)

**Interfaces:**
- Consumes: Task 6 的扁平 class 系统
- Produces:最大的单文件 inline neumorphic 清理

注意:这个文件有 48 处,且很多是同类 pattern。可以用更系统的方式(精确 sed 替换 或 多次 Edit)。

- [ ] **Step 1: 扫描确认**

```bash
cd /Users/liyou/work/hotspot/frontend
grep -c "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" "src/routes/(public)/groups/[id]/+page.svelte"
```

预期:48(或接近)。

- [ ] **Step 2: 用 Python/sed 批量替换**

用精确字符串替换(只删 box-shadow 子句,保留 border):

```bash
cd /Users/liyou/work/hotspot
python3 << 'EOF'
import re
path = 'frontend/src/routes/(public)/groups/[id]/+page.svelte'
with open(path, 'r') as f:
    content = f.read()

# 删除 ' box-shadow: var(--shadow-lg);' 等(注意前导空格 + 后续分号)
# 不影响 ' box-shadow: var(--shadow-brand-btn);'
patterns = [
    r' box-shadow: var\(--shadow-sm\);',
    r' box-shadow: var\(--shadow-md\);',
    r' box-shadow: var\(--shadow-lg\);',
    r' box-shadow: var\(--shadow-xl\);',
    r' box-shadow: var\(--shadow-inset\);',
]
total = 0
for p in patterns:
    new_content, n = re.subn(p, '', content)
    total += n
    content = new_content

with open(path, 'w') as f:
    f.write(content)

print(f"Removed {total} box-shadow clauses")
EOF
```

- [ ] **Step 3: grep 验证**

```bash
cd /Users/liyou/work/hotspot/frontend
grep -c "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" "src/routes/(public)/groups/[id]/+page.svelte"
```

预期:0(或只剩 brand-btn,但 brand-btn 不匹配这个 pattern)。

确认 brand-btn 引用仍在:

```bash
grep -c "var(--shadow-brand-btn)" "src/routes/(public)/groups/[id]/+page.svelte"
```

- [ ] **Step 4: 检查是否有损坏的 inline style(双分号、缺分号)**

```bash
grep -n ";;\| ;" "src/routes/(public)/groups/[id]/+page.svelte" | head -10
```

预期:0 或极少(若有手工修)。

- [ ] **Step 5: 构建校验 + commit**

```bash
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
bun lint 2>&1 | tail -5
```

```bash
cd /Users/liyou/work/hotspot
git add "frontend/src/routes/(public)/groups/[id]/+page.svelte"
git commit -m "重构(UI): groups/[id] 内联 box-shadow 批量清理(48 处)

Python 脚本批量删除 ' box-shadow: var(--shadow-{sm,md,lg,xl,inset});' 子句,
保留 border + background + 其他声明。brand-glow shadow 保留。"
```

---

## Task 12: 主题预览卡扁平化重写 + `<style>` 块清理

**Files:**
- Modify: `frontend/src/routes/(protected)/settings/+page.svelte:285-320, 500-525`(主题预览卡)
- Modify: `frontend/src/lib/components/common/auth-prompt.svelte:60-61, 85-88`(scoped style)
- Modify: `frontend/src/lib/components/common/create-moment-modal.svelte:524-528`(scoped style)
- Modify: `frontend/src/lib/components/common/location-picker-modal.svelte:2880-2881`(scoped style)
- Modify: `frontend/src/lib/components/moments/moment-card.svelte:365-368`(scoped style)
- Modify: `frontend/src/lib/components/common/anchor-summary-card.svelte`(scoped style,若有)

**Interfaces:**
- Consumes: Task 6 的扁平 class 系统 + Task 7-11 的内联清理
- Produces:主题预览卡扁平化(只显示 border + accent 色块,不再画 neumorphic 阴影预览);5 个 `<style>` 块的 neumorphic shadow 移除

- [ ] **Step 1: 主题预览卡扁平化(L285-320 + L500-525)**

读 settings/+page.svelte L285-320(desktop 主题预览卡)。找到内联 style:

```svelte
style="width: 60px; height: 80px; ...; border: 1px solid {accent}22; box-shadow: var(--shadow-md); position: relative; overflow: hidden;"
```

改为(扁平化,只用 border + accent,不再 box-shadow):

```svelte
style="width: 60px; height: 80px; ...; border: 1px solid var(--border-theme); background: var(--surface-bg); position: relative; overflow: hidden;"
```

预览卡内部应有 accent 色块(已有的 `{accent}` 引用),保留。L500-525 同样处理。

读出实际 style 字符串后用 Edit 修改——不要硬编码未确认的字符串。

- [ ] **Step 2: auth-prompt.svelte scoped style(L36-167 区域)**

读 `<style>` 块全文。找到 `.auth-prompt-card { ... box-shadow: var(--shadow-lg); ... }` 和 `.auth-prompt-icon { ... box-shadow: 4px 4px 9px ...; ... }`。

删除 box-shadow 声明,保留 border + background。

- [ ] **Step 3: create-moment-modal.svelte scoped style(L524-528)**

`.compose { ... box-shadow: var(--shadow-lg); ... }` 删除 box-shadow,保留 border。

- [ ] **Step 4: location-picker-modal.svelte scoped style(L2880-2881)**

`:global(.dialog-responsive-position.location-dialog) { ... box-shadow: var(--shadow-lg); ... }` 删除 box-shadow。

- [ ] **Step 5: moment-card.svelte scoped style(L365-368)**

`.moment-card__avatar { ... box-shadow: var(--shadow-sm); ... }` 删除 box-shadow,保留 border。

- [ ] **Step 6: anchor-summary-card.svelte scoped style**

```bash
grep -n "box-shadow" /Users/liyou/work/hotspot/frontend/src/lib/components/common/anchor-summary-card.svelte
```

若有 scoped box-shadow,同样清理。

- [ ] **Step 7: grep 最终验证(整个 src/)**

```bash
cd /Users/liyou/work/hotspot/frontend
grep -rn "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" src --include="*.svelte" --include="*.css"
```

预期:**只有 app.css 里的 preset 定义**(那些是 token 定义,必须保留),.svelte 文件 0 匹配。

确认 brand-glow 引用仍在:

```bash
grep -rn "var(--shadow-brand-btn)" src --include="*.svelte" | wc -l
```

- [ ] **Step 8: 全量构建校验**

```bash
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
bun lint 2>&1 | tail -10
```

预期:基线 7 errors,build 成功,lint 基线 49。

- [ ] **Step 9: 手动视觉验证(全主题 + 全页面)**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

跨 4 主题组合 × 关键页面验证(参考 spec「验证」章节):
1. 无任何"方形色块"或 neumorphic halo
2. 卡片 hover 单方向柔和浮起 + scale
3. 输入框 focus brand border + glow ring
4. 主题预览卡扁平(border + accent 色块)
5. 主题切换器循环 `mist ↔ grove`,显示"岚"/"林"

杀 dev。

- [ ] **Step 10: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/routes/\(protected\)/settings/+page.svelte frontend/src/lib/components/common/auth-prompt.svelte frontend/src/lib/components/common/create-moment-modal.svelte frontend/src/lib/components/common/location-picker-modal.svelte frontend/src/lib/components/moments/moment-card.svelte frontend/src/lib/components/common/anchor-summary-card.svelte
git commit -m "重构(UI): 主题预览卡扁平化 + 5 个 scoped style box-shadow 清理

settings 主题预览卡(L296/512):
- 删除 box-shadow: var(--shadow-md) neumorphic 预览
- 改为 border-only + accent 色块,真实反映新扁平设计

scoped style 清理(5 个 <style> 块):
- auth-prompt (.auth-prompt-card / .auth-prompt-icon)
- create-moment-modal (.compose)
- location-picker-modal (.location-dialog desktop)
- moment-card (.moment-card__avatar)
- anchor-summary-card (若有)

至此,所有 .svelte 内联 + scoped 的 neumorphic box-shadow 都已清理。
仅保留:① app.css 的 --shadow-* preset 定义 ② brand-glow 单方向 glow。"
```

---

## Task 13: 最终全 grep 验证 + AGENTS.md 文档同步

**Files:**
- Modify: `frontend/AGENTS.md`(若需更新主题/hover 矩阵描述)
- Verify: 所有前述 Task 的累积效果

**Interfaces:**
- Consumes: 所有前 Task
- Produces:整个分支交付清单 ready,文档与代码同步

- [ ] **Step 1: 全量 grep 终验**

```bash
cd /Users/liyou/work/hotspot/frontend

# 1. 无任何老主题名残留
grep -rn "'indigo'\|'forest'\|'sand'\|\"indigo\"\|\"forest\"\|\"sand\"" src --include="*.ts" --include="*.svelte" --include="*.css" --include="*.html"
# 预期: 0

# 2. 无任何 .svelte 内联 box-shadow: var(--shadow-{sm,md,lg,xl,inset})
grep -rn "box-shadow: var(--shadow-sm\|box-shadow: var(--shadow-md\|box-shadow: var(--shadow-lg\|box-shadow: var(--shadow-xl\|box-shadow: var(--shadow-inset" src --include="*.svelte"
# 预期: 0

# 3. 无任何死类引用(.surface-raised / .neu-flat / .surface-convex / .button-glow / .control-small / .icon-badge)
grep -rn "surface-raised\|neu-flat\|surface-convex\|button-glow\|control-small\|icon-badge" src --include="*.svelte"
# 预期: 0

# 4. brand-glow 仍正常引用
grep -rn "var(--shadow-brand-btn)" src --include="*.svelte" | wc -l
# 预期: 与之前一致(~50 处)

# 5. 新 token 已定义
grep -n "shadow-elevation" src/app.css
# 预期: 4 处(grove/mist × light/dark)
```

如有任何 grep 不符合预期,回到对应 Task 修。

- [ ] **Step 2: 全量构建校验**

```bash
bun check 2>&1 | tail -15
bun run build 2>&1 | tail -15
bun lint 2>&1 | tail -10
```

预期:基线 7 errors / build 成功 / 基线 49 warnings。

- [ ] **Step 3: 跨主题视觉验证(完整)**

```bash
bun run dev --port 5181 > /tmp/hotspot-dev.log 2>&1 &
sleep 4
```

依次切换 4 个主题组合(mist/grove × light/dark),每个主题访问:
- 首页、附近的人、动态、聊天、通知、profile、设置、admin、登录

验证清单:
1. ✅ 无任何方块色块 / neumorphic halo
2. ✅ 卡片 hover 单方向柔和浮起 + scale
3. ✅ 输入框 focus brand border + glow ring
4. ✅ 主题切换器显示"岚"/"林",循环 mist ↔ grove
5. ✅ 主题预览卡扁平(border + accent 色块)
6. ✅ 老用户 localStorage 自动重置到 mist

杀 dev。

- [ ] **Step 4: 更新 `frontend/AGENTS.md`**

读 `frontend/AGENTS.md`,在「UI/UX 设计规范」相关段更新:
- 主题族从 3 个(indigo/forest/sand)→ 2 个(mist 岚 / grove 林)
- 删除"forest 扁平 + indigo/sand neumorphic"的描述,改为"全主题扁平 border-only + 单方向 elevation hover"
- 删除"neumorphic / ceramic tactility"等过时概念

具体改动看现有文档结构,保持风格一致。

- [ ] **Step 5: 总结 commit(若 AGENTS.md 改)**

```bash
cd /Users/liyou/work/hotspot
git status
git add frontend/AGENTS.md
git commit -m "文档(AGENTS): 主题系统更新为 mist/grove + 全扁平设计语言"
```

- [ ] **Step 6: 总结整个工作分支**

读 spec「目标」章节,逐项确认:

| 目标 | 达成 | 验证位置 |
|---|---|---|
| 1. 主题系统重塑(删 sand,重命名) | ✓ | Task 1-4 |
| 2. 设计语言扁平化 | ✓ | Task 5-6 |
| 3. 代码清理(死类 + 内联 neumorphic) | ✓ | Task 6-12 |
| 4. 产品调性(岚/林 + 诗意描述) | ✓ | Task 2 |

---

## 风险与回滚

每个 Task 是独立 commit,失败可 `git revert <hash>` 单 Task 回滚。

关键风险:
- **Task 5 token 重定义**:全站视觉变化最大。若某处 elevation shadow 太弱/太强,可微调 `--shadow-elevation` 透明度。
- **Task 6 死类删除**:`.neu-flat` 在 3 处 .svelte 用了,Step 1 必须先迁移再删除类定义,否则组件失样式。
- **Task 11 批量 sed**:48 处替换可能误伤,Step 4 必须检查双分号 / 缺分号。
- **Task 1+3 app.html 不同步**:首屏会闪老主题——Task 3 必须同步 app.html 的硬编码脚本。

## 后续(非本轮)

- 进一步简化 `--color-shadow-light/dark` `@theme` token(零引用,可删)
- `PREVIEW_ACCENT` 改用 `var(--theme-accent-primary)` 而非硬编码 hex
- `:global(.map-cluster-bubble:hover)` 的地图交互 a11y 改造(之前一轮遗留)
- `.surface-flat` 在 40+ 按钮上的 focus-visible 缺失(之前一轮遗留)
- 主题切换 toast 提示(可选)
