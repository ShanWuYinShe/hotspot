# 主题重命名 + 删除 sand + 全扁平重构 — 设计文档

## 背景

经过 sand 主题方块色块修复 + hover 统一一轮后,用户对设计系统做了三个重大决策:

1. **删除 sand 主题**:用户最初设计的"双边框 neumorphic 陶瓷质感"已被放弃,sand 主题与 forest 同质化、与 indigo 风格冲突,且大量 inline neumorphic 阴影绕过 token 体系造成维护负担。直接删除比修复更彻底。

2. **主题中文化重命名**:现有"活力靛蓝 / 清爽森林 / 暖砂陶瓷"的"形容词+颜色/材质"命名格式被认为信息量低、缺乏个性。改用**同一意象体系(山岳意象)**的单字命名,让两个主题代表产品调性而非颜色。

3. **全扁平重构**:既然放弃 neumorphic,所有 Type A 类(同时 border + neumorphic 阴影)都应去 neumorphic 化,统一为 border-only + 单方向 elevation shadow。175+ 处内联 `box-shadow: var(--shadow-*)` 全部清理。

## 目标

1. **主题系统重塑**:删除 sand;indigo→岚(mist),forest→林(grove);CSS 选择器同步用 `mist`/`grove`;localStorage 重置到默认。
2. **设计语言扁平化**:所有 Type A surface/input class 去 neumorphic 双向阴影,改为 border-only;hover 用单方向 elevation shadow + scale 反馈;输入框 focus 用 border + glow ring。
3. **代码清理**:删除 0 引用的死 class(`.surface-convex` / `.surface-raised` / `.button-glow` / `.control-small` / `.icon-badge`);清理 175+ 处内联 neumorphic。
4. **产品调性**:主题名"岚/林"出自同一山岳意象体系,呼应产品"附近/位置"主题;描述用诗意意象而非颜色说明。

## 非目标

- 不改变主题切换 UI 的交互流程(仍是 cycle 按钮 + 设置页卡片网格)。
- 不动 mode-watcher(明暗模式)逻辑。
- 不动后端契约、API、路由结构。
- 不动组件 DOM 结构(沿用 `.page-shell` / `.density-grid` 等既有原语)。
- 不重写 i18n 系统(只增删 key)。

## 设计

### §1. 主题系统重塑

#### 1.A 主题命名映射

| 老 code | 新 code | 中文 label | 英文 label | 描述(诗意意象) |
|---|---|---|---|---|
| `indigo` | `mist` | 岚 | Mist | 山顶雾气,冷调清亮,如清晨远眺 |
| `forest` | `grove` | 林 | Grove | 山林深处,沉静妥帖,如午后林间 |
| `sand` | (删除) | — | — | — |

**默认主题**:`mist`(对应原 `indigo` 默认)。

**localStorage 重置**:不做老→新迁移。`themeStore.init()` 检测到存储值不在新 `['mist', 'grove']` 中时,落回默认 `mist`,并写回新值。`THEME_MIGRATION` 表清空(避免老 `indigo`/`forest`/`sand` 被识别)。

理由:用户明确要求重置;且改名 + 删 sand 同时发生时,迁移规则易出错(老 `forest` 用户本期待墨绿,迁移后是 grove 仍墨绿——但老 `indigo` 用户的"紫色调"期待可能与 mist"岚"的意境冲突)。重置更干净。

#### 1.B `theme.svelte.ts` 改动

```typescript
export const THEME_FAMILIES = ['mist', 'grove'] as const;
// ThemeFamily = 'mist' | 'grove'

export const THEME_FAMILY_OPTIONS = [
  { family: 'mist', labelKey: 'sidebar.mist', descriptionKey: 'sidebar.mistDesc' },
  { family: 'grove', labelKey: 'sidebar.grove', descriptionKey: 'sidebar.groveDesc' }
];

const DEFAULT_THEME_FAMILY: ThemeFamily = 'mist';

// 清空迁移表 — 老值全部落回默认
const THEME_MIGRATION: Record<string, ThemeFamily> = {};
```

`app.html` 的阻塞脚本(`THEME_FAMILY_INIT_SCRIPT`)通过 `THEME_FAMILY_INIT_SCRIPT` 常量自动同步(families 数组 / default / migration 都从常量注入)。

#### 1.C i18n key 重命名

`src/lib/i18n/locales/zh-CN.ts` 和 `en.ts` 中:

| 老 key | 新 key | zh-CN | en |
|---|---|---|---|
| `sidebar.indigo` | `sidebar.mist` | 岚 | Mist |
| `sidebar.indigoDesc` | `sidebar.mistDesc` | 山顶雾气,冷调清亮,如清晨远眺 | Cool, luminous mist over mountain peaks |
| `sidebar.forest` | `sidebar.grove` | 林 | Grove |
| `sidebar.forestDesc` | `sidebar.groveDesc` | 山林深处,沉静妥帖,如午后林间 | Quiet, settled shade in deep mountain woods |
| `sidebar.sand` / `sidebar.sandDesc` | (删除) | — | — |
| `settings.themeNameIndigo` | `settings.themeNameMist` | 岚 | Mist |
| `settings.themeDescIndigo` | `settings.themeDescMist` | 山顶雾气,冷调清亮 | Cool mist over peaks |
| `settings.themeNameForest` | `settings.themeNameGrove` | 林 | Grove |
| `settings.themeDescForest` | `settings.themeDescGrove` | 山林深处,沉静妥帖 | Deep mountain grove |
| `settings.themeNameSand` / `themeDescSand` | (删除) | — | — |

`settings/+page.svelte` L34-44 的 `THEME_NAMES` / `THEME_DESC` 映射同步更新。

### §2. CSS token 块重命名 + sand 删除

#### 2.A `app.css` 主题块

- **删除**:`:root[data-theme-family='sand']`(L496-586)和 `:root[data-theme-family='sand'].dark`(L588-676)整块
- **重命名**:
  - `:root[data-theme-family='indigo']`(L313)→ `:root[data-theme-family='mist']`(L312 `:root,` 也指向 mist)
  - `:root[data-theme-family='indigo'].dark`(L404)→ `:root[data-theme-family='mist'].dark`
  - `:root[data-theme-family='forest']`(L130)→ `:root[data-theme-family='grove']`
  - `:root[data-theme-family='forest'].dark`(L220)→ `:root[data-theme-family='grove'].dark`
- **所有 `[data-theme-family='forest']` / `'indigo'` / `'sand'` 选择器**(L865, L890-893, L1190-1205, L1213-1230, L1928-1944 等)同步重命名为 `'grove'` / `'mist'`,删除所有 `'sand'` 选择器

#### 2.B `--shadow-*` token 重定义为单方向 elevation

当前(neumorphic 双向):
```css
--shadow-sm: 4px 4px 10px var(--shadow-dark), -4px -4px 10px var(--shadow-light);
--shadow-md: 5px 5px 10px var(--shadow-dark), -5px -5px 10px var(--shadow-light);
--shadow-lg: 8px 8px 20px var(--shadow-dark), -8px -8px 20px var(--shadow-light);
--shadow-xl: 10px 10px 20px var(--shadow-dark), -10px -10px 20px var(--shadow-light);
--shadow-inset: inset 6px 6px 12px var(--shadow-dark), inset -6px -6px 12px var(--shadow-light);
```

改为(单方向 elevation,统一柔和):
```css
--shadow-sm: 0 1px 2px var(--shadow-elevation);
--shadow-md: 0 2px 6px var(--shadow-elevation);
--shadow-lg: 0 4px 12px var(--shadow-elevation);
--shadow-xl: 0 8px 24px var(--shadow-elevation);
--shadow-inset: inset 0 1px 2px var(--shadow-elevation);
```

新 token `--shadow-elevation`(每个主题族 + light/dark 定义):
- mist light: `rgba(99, 102, 241, 0.08)` (带品牌色调的柔和暗影)
- mist dark: `rgba(0, 0, 0, 0.32)`
- grove light: `rgba(27, 31, 27, 0.06)`
- grove dark: `rgba(0, 0, 0, 0.36)`

`--theme-shadow-light` / `--theme-shadow-dark` 在所有 mist + grove 块内**置 0**(对齐 grove 现状,拆除 neumorphic 系统)。这同时让所有现存未清理的内联 `var(--shadow-*)` 引用也自动扁平化(虽然数值变化,但不再画方块)。

### §3. Type A surface/input class 扁平化

| 类 | 当前 | 之后 |
|---|---|---|
| `.surface-hero` / `.neu-surface-hero` | border + 8px 双向 shadow | **border-only**(`border: 1px solid var(--border-theme)` + hover `border-active`),等同 `.surface-flat`。保留类名作为 `.surface-flat` 的语义别名(8 处组件无需立即迁移)。forest 的 `border-active` 强边框 override(L890-893)删除——既然全主题统一 border-only,forest 不再特殊。 |
| `.surface-flat` / `.neu-surface-flat` | border-only + hover border-active | **不变**(已是目标状态) |
| `.surface-card` | border-only 静态容器 | **不变** |
| `.surface-ghost` | transparent + hover bg | **不变** |
| `.surface-raised` / `.neu-surface-raised` / `.neu-flat` | border + 10px 双向 | **删除类定义**(0 引用);`.neu-flat` 3 处引用迁移到 `.surface-flat` |
| `.surface-pressed` / `.neu-pressed` | border-active + inset 双向 | **改 border-only**:更深 border-color + 更暗 background 表达"凹陷"。inset shadow 删除。2 处引用(user-card square variant、chat/users/[id])。 |
| `.surface-convex` / `.neu-convex` | border + 12px 双向 | **删除**(0 引用) |
| `.input-surface` / `.neu-input-surface` | border + 5px 双向, focus 翻 inset | **border-only**:rest 用 `--border-theme`,focus-within 用 `border-color: var(--brand)` + `box-shadow: 0 0 0 2px var(--glow)` ring。20 处引用无需迁移。 |
| `.input-base` / `.neu-input-base` | border + 8px 双向, focus 翻 inset | 同 `.input-surface`。16 处引用无需迁移。 |
| `.field-inset` | border + inset 双向 | **删除 inset shadow**,改 border-only + 略深 background 表达凹陷。13 处引用无需迁移。 |
| `.button-glow` / `.neu-button-glow` | shadow-only | **删除**(0 引用) |
| `.control-small` / `.neu-small` | border + 5px 双向 | **删除**(0 引用) |
| `.icon-badge` / `.neu-icon-badge` | shadow-only | **删除**(0 引用) |

**`.card-hover:hover`**:
- 当前用 `var(--hover-shadow-card)`(mist 是 neumorphic `-9px -9px`,grove 已置 0)
- 改为统一的单方向 elevation:`box-shadow: var(--shadow-md) !important`(`0 2px 6px var(--shadow-elevation)`)
- 配合组件既有的 `hover:scale-[1.02]` 表达浮起
- 全主题一致(不再主题特殊化)

### §4. 组件内联 neumorphic 清理

**175+ 处** `style="...border + box-shadow: var(--shadow-*)..."` 全部清理。

**策略**:
1. **卡片/面板级**(audit 第 2 节列出的 modal/dialog/section card,例如 `confirm-dialog.svelte:50`、`create-group-modal.svelte:231`、`search-modal.svelte:310` 等):
   - 移除 `box-shadow: var(--shadow-lg/md/sm)` 整段
   - 保留 `border: 1px solid var(--border-theme)` + `background: var(--surface-bg)`
   - 视觉等价 `.surface-flat`(有些本来就是 `.surface-flat` + 多余 inline shadow)

2. **图标按钮级**(`box-shadow: var(--shadow-sm)` 在 36-48px 图标徽章上,如 `admin/*/+page.svelte` 多处):
   - 移除 box-shadow
   - 保留 border + background

3. **chip / 筛选按钮级**(`box-shadow: var(--shadow-md)` + border):
   - 移除 box-shadow
   - 保留 border + color

4. **大型聚集文件**单独批次处理:
   - `location-picker-modal.svelte`(22 处)
   - `routes/(public)/groups/[id]/+page.svelte`(48 处)
   - `routes/(admin)/admin/*/+page.svelte`(累计 ~25 处)
   - `routes/(protected)/settings/*/+page.svelte`(累计 ~20 处)

### §5. `<style>` 块清理

5 处组件 `<style>` 块含 neumorphic shadow:
- `auth-prompt.svelte` L60-88(`.auth-prompt-card` + `.auth-prompt-icon` 用 raw + `var(--shadow-lg)`):移除 box-shadow,保留 border
- `create-moment-modal.svelte` L524-528(`.compose` 用 `var(--shadow-lg)`):移除
- `location-picker-modal.svelte` L2880(`.dialog-responsive-position.location-dialog`):移除
- `moment-card.svelte` L365-368(`.moment-card__avatar` 用 `var(--shadow-sm)`):移除
- `anchor-summary-card.svelte` L29 + L56:移除

### §6. 设置页主题预览卡扁平化

`routes/(protected)/settings/+page.svelte` L296 + L512 主题预览卡:
- 当前:`border: 1px solid {accent}22; box-shadow: var(--shadow-md)` + 复杂内联
- 重写:仅 `border: 1px solid var(--border-theme)` + 内部用 `accent` 色块表达主题色调,不再画 neumorphic 阴影预览

`PREVIEW_ACCENT` 同步重命名:
```typescript
const PREVIEW_ACCENT: Record<ThemeFamily, string> = {
  mist: '#6366f1',  // indigo 紫
  grove: '#3d7a55'  // forest 绿
};
```

## 实施分批

每批可独立验证 + 回滚:

### 批次 1 — 主题系统重塑(删 sand + 重命名)
1. `theme.svelte.ts`:`THEME_FAMILIES` = `['mist', 'grove']`,`DEFAULT_THEME_FAMILY = 'mist'`,`THEME_MIGRATION = {}`,`THEME_FAMILY_OPTIONS` 改名
2. i18n key 重命名 + 描述文案改写 + 删除 sand 相关 key
3. `theme-toggle.svelte` fallback `'forest'` → `'grove'`
4. `settings/+page.svelte` `THEME_NAMES` / `THEME_DESC` / `PREVIEW_ACCENT` 重命名
5. `app.html` 检查(若硬编码主题相关常量需同步)
6. 构建校验

### 批次 2 — CSS 重命名 + sand 块删除
1. `app.css`:`[data-theme-family='indigo']` → `'mist'`,`'forest'` → `'grove'`,删除所有 `'sand'` 选择器
2. 删除 sand light + dark 整块 token
3. 全 grep 验证无 `'indigo'` / `'forest'` / `'sand'` 残留
4. 构建校验

### 批次 3 — `--shadow-*` 重定义为单方向 + token 置 0
1. 引入 `--shadow-elevation` token(4 主题块:mist/grove × light/dark)
2. `--shadow-sm/md/lg/xl/inset` 改为单方向公式
3. `--theme-shadow-light/dark` 在 mist + grove 全部块内置 0
4. 构建校验 + 视觉验证(此时所有现存内联 `var(--shadow-*)` 自动扁平化,虽然显示效果改变但不会破坏布局)

### 批次 4 — Type A class 扁平化 + 死类删除
1. `.surface-hero` 改为 `.surface-flat` 别名(border-only);删除 forest 特殊 border-active override
2. `.surface-pressed` 改 border + 略深 background 表达凹陷(去 inset shadow)
3. `.input-surface` / `.input-base` / `.field-inset` 去 neumorphic,改 border-only + focus glow ring
4. `.card-hover:hover` 改用 `var(--shadow-md)` 单方向
5. 删除死类:`.surface-raised` / `.neu-flat`(组件层 3 处 `.neu-flat` 先迁移到 `.surface-flat`)、`.surface-convex`、`.button-glow`、`.control-small`、`.icon-badge`
6. 构建校验

### 批次 5 — 组件内联 neumorphic 清理(175+ 处)
按文件聚集度分子批:
- 5a: `lib/components/common/*`(`search-modal`、`create-group-modal`、`create-moment-modal`、`create-location-share-modal`、`mobile-create-sheet`、`confirm-dialog`、`auth-form-content`、`anchor-summary-card`、`map-card`、`list-skeleton`、`page-header`、`theme-toggle`、`empty-state`)
- 5b: `lib/components/ui/*`(`dialog-content`、`dropdown-menu-content`、`dropdown-menu-sub-content`、`tabs-list`)
- 5c: `lib/components/layout/*`(`app-shell`、`page-header`)
- 5d: `routes/(admin)/admin/*/*`(累计 ~25 处)
- 5e: `routes/(protected)/settings/*/*`(累计 ~20 处)
- 5f: `routes/(protected)/chat/*`(少量)
- 5g: `routes/(public)/groups/[id]/+page.svelte`(48 处,单独批)
- 5h: `routes/(public)/moments/[id]/+page.svelte` + `share/[code]/+page.svelte`
- 5i: `lib/components/common/location-picker-modal.svelte`(22 处,单独批)
- 5j: 其他散落(`+error.svelte`、`banned/+page.svelte`、`(auth)/auth/*`、`(protected)/profile/*`、`(protected)/notifications/+page.svelte`、`(protected)/moments/+page.svelte`、`(protected)/groups/+page.svelte`、`(protected)/relations/+page.svelte`、`(public)/groups/nearby`、`(public)/users/nearby`、`(public)/discover`、`(public)/search`)

### 批次 6 — `<style>` 块 + 主题预览卡 + 残余
1. 5 个 `<style>` 块清理(auth-prompt、create-moment-modal、location-picker-modal、moment-card、anchor-summary-card)
2. settings 主题预览卡扁平化重写
3. 最终 grep 验证无 `var(--shadow-` 残留在 .svelte 内联 style(除合法的 `--shadow-elevation` 引用)

## 验证

每批完成后:
```bash
bun check         # 类型检查(基线 7 errors)
bun run build     # 构建
bun lint          # 格式
```

主题视觉验证(手动,跨 4 个组合):
1. mist light / mist dark / grove light / grove dark 依次切换
2. 每个主题下访问关键页面:首页、附近的人、动态、聊天、通知、profile、设置、admin
3. 确认:
   - 无任何"方形色块"halo(任何主题、任何位置)
   - 卡片 hover 表现一致(单方向浮起 + scale)
   - 输入框 focus 用 border + glow ring,不再 inset 凹陷
   - 主题切换器显示"岚/林"
   - 老用户主题被重置到默认 mist

WCAG 对比度验证:
- mist light `--theme-text #1e1b4b` on `--page-bg #eef2ff` ~ 12.5:1 ✓
- grove dark `--theme-text #dce8de` on `--page-bg #0e110e` ~ 14:1 ✓
- 所有 brand 色配 `--text-on-accent` 维持现状 ✓

## 风险与回滚

| 风险 | 缓解 |
|---|---|
| localStorage 重置让老用户困惑 | 重置是用户明确决策;若反馈强烈,可在批次 1 加一个 toast 提示"主题已更新" |
| 全扁平后 mist 与 grove 区分度下降 | 仍保持色相对比(mist 紫调 / grove 绿调);hover 反馈全主题一致是预期效果 |
| 175+ 处内联清理遗漏 | 每子批后 grep 验证;批次 6 最终全量 grep |
| `--shadow-*` token 重定义改变既有视觉 | 这是预期效果(扁平化);批次 3 后所有现存引用自动跟随 |
| 删除 `.neu-flat` 等死类时漏掉 3 处 `.neu-flat` 引用 | 批次 4 先迁移 `.neu-flat` → `.surface-flat`,再删除类定义 |

每批是独立 commit,失败可 `git revert` 单批。

## 后续(非本轮)

- 主题切换 toast 提示(可选)
- 进一步简化 `--color-shadow-light/dark` `@theme` token(零引用,可删)
- `PREVIEW_ACCENT` 改用 `var(--theme-accent-primary)` 而非硬编码 hex(需要预览卡支持运行时取 token)
