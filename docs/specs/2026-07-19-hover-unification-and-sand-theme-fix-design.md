# Hover 按类型统一 + Sand 主题色彩重塑 — 设计文档

## 背景

两个相互关联的问题:

**问题 1 — Hover 效果割裂**:同一个主题下,"列表行" 这一类元素当前混用 4 种 hover 机制:
- `.surface-row`(背景 tint,主页列表/nearby 用)
- `.row-hover`(inset `0 0 0 9999px var(--border-theme)` 巨型色块叠加,chat/notifications/profile 用)
- `.list-row-hover`(accent-soft tint,chat group 行用)
- 内联 `hover:bg-[var(--surface-secondary)]`(search modal 用)

"圆/pill icon 按钮" 一类元素则完全绕开已定义的 `.circle-hover` / `.pill-hover`(0 引用),散乱使用内联 `hover:scale-105/110` / `hover:opacity-70/80/90`(数值还不统一)。

**问题 2 — Sand 主题色彩失真**:用户反馈 sand(暖纱陶瓷)主题下"很多元素周围都有莫名其妙的方形色块"。根因审计发现:

- `--theme-shadow-light: rgba(255,252,245,0.96)`(96% 近不透明奶白)在 `#efe7da` 米色画布上不是阴影,而是实心奶白方块。所有 neumorphic class(`.surface-hero` / `.surface-raised` / `.surface-convex` / `.button-glow` / `.input-*` / `.icon-badge` / `.control-small`)以及 `--shadow-sm/md/lg/xl` preset 都用它,在元素周围画出 4–12px 偏移的奶白方块。
- 对比之下:forest 把两个 shadow 都置 0(扁平),indigo 的 light 是纯白叠在近白画布上(融合不可见)。**只有 sand 的 light shadow 既不透明又色相对比明显**——结构性根因。
- 放大因素:① 组件内联 `box-shadow: var(--shadow-sm/md/lg/xl)` 绕过 class 体系(search-modal、create-group-modal);② skeleton 占位用 `--theme-border`(9% 棕褐)作填充,在方形元素上读起来是 pigmented tan 方块;③ `.ambient-glow::before` 没有 sand 专属 opacity,terracotta glow 溢出。

**问题 3 — Sand dark 与 forest dark 同质化**:用户反馈 sand dark "完全和清爽森林一模一样"。数据对比证实:

- sand dark 的 `--theme-shadow-light` 仅 0.055(几乎不可见)→ 暖光高光丧失,陶瓷凸起感消失
- light:dark 比例 = 1:10.5,暗部压平,失去陶瓷温度
- `--theme-bg #211b17` 与 `--page-bg #15110e` 亮度差仅 ~7%(理想 8–18%),卡片几乎不凸出画布
- forest dark 是纯扁平(shadow 全 0),sand dark 当前的"几乎扁平"在视觉上确实与 forest dark 难以区分
- "暖纱陶瓷"理应有 ceramic tactility(可见的凸起/凹陷体积感),而非与"清爽森林"共享扁平暗色语言

## 目标

1. **同主题内按类型统一 hover**:同一类元素用同一种 hover 机制,消除 4 套并存。
2. **跨主题按主题族分策略**:forest(扁平)与 sand/indigo(neumorphic)对独立交互元素用各自最合适的 hover 语言;**列表行作为密集元素全主题统一用 background-tint**(避免 neumorphic 在 sand 下方块化)。
3. **根治 sand light 方块**:从 token 层降低 light shadow 不透明度,清理绕过 class 体系的内联阴影,skeleton 改用专门 token。
4. **重塑 sand 陶瓷质感**:light + dark 都要呈现可见的 ceramic tactility,让 sand 与 forest 在第一眼可分辨("立体暖调 vs 扁平冷调"),不再同质化。
5. **不改变功能行为**:本轮纯视觉/样式重构,不动任何业务逻辑、API、路由。

## 非目标

- 不重构组件 DOM 结构、不改布局(沿用 `.page-shell` / `.density-grid` 等既有原语)。
- 不动 forest / indigo 主题(它们当前表现正常,仅在 hover class 层做forest 专属降级)。
- 不引入新依赖。

## 设计

### §1. Hover 类型分类矩阵

**核心原则**:列表行作为密集交互元素,全主题统一用 background-tint(不画 neumorphic 浮起阴影,避免在 sand 下密集列表变成方块墙);独立交互元素按主题族分策略。

| 元素类型 | forest(扁平) | sand / indigo(neumorphic) | 统一入口 |
|---|---|---|---|
| **卡片** (`.surface-hero`) | `border-color` 加深到 `--border-active` + `scale-1.02` | neumorphic 浮起 `--hover-shadow-card` + `scale-1.02` | `.card-hover` + `hover:scale-[1.02]` |
| **密集列表行** | background-tint `--border-theme` / `--border-active` | **background-tint `--border-theme` / `--border-active`**(全主题统一) | `.list-row`(新) |
| **圆形 icon 按钮** | `.surface-flat` border 加深 | neumorphic 浮起 `--hover-shadow-btn` | `.icon-btn-hover`(新,替代散乱内联 `hover:scale-*`) |
| **Pill 按钮 / 筛选 chip** | `.surface-flat` border 加深 | neumorphic 浮起 `--hover-shadow-pill` | `.pill-hover`(激活,已定义但 0 引用) |
| **Tag chip(hashtag)** | accent-tint(color-mix brand) | **accent-tint**(全主题统一) | `.tag-chip`(已存在,保持) |
| **品牌 CTA 按钮** | `hover:brightness-110` | `hover:brightness-110`(sand/indigo dark 加 glow) | 内联 `hover:brightness-110` 统一 |

**消除清单**:
- `.row-hover` / `.neu-row-hover`(inset 9999px 巨型叠加) → 并入 `.list-row`。
- `.surface-row`(未带 active 态、focus 态) → 并入 `.list-row`。
- `.list-row-hover`(accent-soft tint,只 1 引用) → 并入 `.list-row`(改用 border 系 tint)。
- 散乱内联 `hover:opacity-70/80/90`(无规则数值) → 按 CTA / 二级按钮分类收敛到 `hover:brightness-110` 或 `.surface-flat`。
- 散乱内联 `hover:scale-105/110` 在 icon/pill 按钮上 → `.icon-btn-hover` / `.pill-hover`。
- `0 引用` 的 `.circle-hover` → 重命名为 `.icon-btn-hover` 并实际激活使用(语义更清晰)。

### §2. Sand 主题色彩重塑(token 根治 + 陶瓷质感重建)

这一节合并两个目标:① 消除 light mode 方块色块;② 重建 light + dark 的陶瓷质感,让 sand 不再与 forest 同质化。

#### 设计原则

真正的 ceramic tactility 来自两个色层在 neumorphic 阴影下的相互作用:

1. **画布比卡片暗**(`page-bg` 比 `surface-bg` 暗 8–18% 亮度):卡片才能"凸"出画布
2. **暖光高光 + 暖阴阴影**(两个 shadow 都带色相):纯黑/纯白阴影看起来像塑料,带暖色相才像被阳光晒过的石头

当前 sand 的问题根因:
- **Light**:`shadow-light 0.96` 太实成奶白方块,但**色相方向是对的**(暖象牙)
- **Dark**:`shadow-light 0.055` 太弱(高光不可见)+ `shadow-dark 0.58` 偏重,light:dark = 1:10.5,陶瓷凸起感消失;bg/page 亮度差仅 ~7%,卡片几乎不凸出画布 → 看起来与 forest 扁平暗色难以区分

#### A. Light mode token

| Token | 当前 | 目标 | 理由 |
|---|---|---|---|
| `--theme-shadow-light` | `rgba(255,252,245,0.96)` | `rgba(251,232,216,0.5)` | **暖桃白 #fbe8d8**:与画布的黄褐形成色温对比,有"暖光打在陶瓷上"的感觉;0.5 回归阴影语义,消除方块 |
| `--theme-shadow-dark` | `rgba(167,145,116,0.38)` | `rgba(167,145,116,0.22)` | 同步降低,保持明暗对称(0.5:0.22 ≈ 既有 0.96:0.38 比例) |

light mode 的 `bg/page`(`#efe7da`/`#e2d6c5`,亮度差 ~10%)已经合理,卡片能凸出画布,不动。

#### B. Dark mode token —— 重建陶瓷凸起

| Token | 当前 | 目标 | 理由 |
|---|---|---|---|
| `--theme-bg` | `#211b17` | `#2a221c` | 略提亮,给 page-bg 留出向下亮度空间 |
| `--page-bg` | `#15110e` | `#18120e` | 同步调整,保持与 bg 的亮度差 ~15%(原来仅 ~7%,卡片几乎不凸出) |
| `--theme-shadow-light` | `rgba(255,218,179,0.055)` | `rgba(255,200,150,0.18)` | **暖桃光高光**:0.18 让陶瓷凸起可见(原 0.055 几乎不可见);色相偏桃色与 light mode 呼应 |
| `--theme-shadow-dark` | `rgba(0,0,0,0.58)` | `rgba(0,0,0,0.55)` | 微降,把 light:dark 从 1:10.5 拉到约 1:3,陶瓷感立体而不被压平 |
| `--theme-text` | `#f1e4d2` | `#f5ead8` | 略提亮,与新 bg 配合提高对比度 |
| `--theme-text-muted` | `#a99682` | `#b8a48c` | 提亮,避免与 forest dark 的灰绿 muted 混淆 |

light:dark 阴影比例 = `0.18 : 0.55` ≈ 1:3,达到 indigo dark 的同档质感(`0.07 : 0.72` ≈ 1:10 偏扁,我们更暖更立体)。

#### C. Brand 强调色

保持现状不变——这是 sand 与 forest 的关键区分点:

- Light `--theme-accent-primary: #b86f4b`(terracotta 赤陶)
- Dark `--theme-accent-primary: #d18a5d`(ember 余烬橙)

不动 `--theme-accent`(secondary warm gray `#8b806f` / `#b7a28a`),它们承担次级强调。

#### D. 新增 `--skeleton-bg` token(全主题族)

skeleton 占位符不再借 `--theme-border` 做填充(后者在 sand 下是 9% 棕褐,pigmented tan,看起来像方块)。

```css
/* 每个主题族块内追加 */
--skeleton-bg: color-mix(in srgb, var(--surface-bg) 65%, var(--page-bg));
```

`--surface-bg` 与 `--page-bg` 混合得到一个比 surface 略深的同色调,作为占位填充时是"凹陷感"而非"色块"。sand light 下这个混合色是 ~`#e8ddc9`,sand dark 下是 ~`#221a14`。

#### E. `.ambient-glow::before` sand 专属 opacity

当前 sand 沿用默认 0.55,terracotta glow 在奶白底上过亮。新增:

```css
[data-theme-family='sand'] .ambient-glow::before,
[data-theme-family='sand'].dark .ambient-glow::before {
  opacity: 0.4;
}
```

dark 下 terracotta glow 反而需要保持可见,但 0.4 配合新的暖光阴影正好够亮。

#### F. 与 forest 的区分总结

| 维度 | forest | sand(改后) |
|---|---|---|
| 整体结构 | **扁平**(shadow 全 0) | **neumorphic**(shadow 双向) |
| Light 色温 | 偏冷绿白 `#f9f9f5` | 暖象牙 `#efe7da` + 桃光高光 |
| Dark 色温 | 偏冷墨绿 `#161a16` | 暖深棕 `#2a221c` + 桃光高光 |
| 卡片凸起 | border-only | **可见凸起**(dark 下 bg/page 差 15% + 阴影 1:3) |
| Brand | forest green `#3d7a55` | terracotta→ember `#b86f4b`/`#d18a5d` |

改完后用户能在第一眼分辨:"扁平冷调 vs 立体暖调",不会再说"和 forest 一模一样"。

### §3. Hover class 重构细节

**新增 `.list-row`**(合并 row-hover / surface-row / list-row-hover):

```css
/* 统一密集列表行 hover: background-tint, 无阴影, 全主题一致。
   - hover: --border-theme tint(轻度)
   - active: --border-active(按下加重)
   - focus-visible: 保持 outline 环
   无 transform、无 box-shadow,保证 sand 下不会画方块。 */
.list-row {
  background-color: var(--surface-bg);
  transition: background-color 0.15s ease;
}
@media (hover: hover) {
  .list-row:hover {
    background-color: var(--border-theme);
  }
}
.list-row:active {
  background-color: var(--border-active);
}
.list-row:focus-visible {
  outline: 2px solid var(--glow);
  outline-offset: 2px;
}
```

**保留 `.row-hover` 作为向后兼容别名** = `.list-row`(因为它是 composable 用法,可能叠加在内联背景上)。短期保留语义,内部改为 background-color:

```css
/* @deprecated alias — use .list-row. Kept for backward compat;
   behavior unified with .list-row (background tint, no inset overlay). */
.row-hover,
.neu-row-hover {
  transition: background-color 0.15s ease;
}
.row-hover:hover,
.neu-row-hover:hover {
  background-color: var(--border-theme);
}
```

> 注意:旧 `.row-hover` 用的是 `box-shadow: inset 0 0 0 9999px var(--border-theme)`(因为某些场景元素本身已有内联 `background`,需要叠加而非替换)。审计后未发现 `.row-hover` 在"已有内联 background 的元素"上使用——所有 6 处用 `.row-hover` 的元素都没有内联 background。因此可以安全地改成 background-color 方案。

**重命名 `.circle-hover` → `.icon-btn-hover`**(0 引用,可安全改名):

```css
.icon-btn-hover {
  transition: box-shadow 0.2s cubic-bezier(0.22, 1, 0.36, 1),
              transform 0.15s ease;
}
.icon-btn-hover:hover {
  box-shadow: var(--hover-shadow-btn);
}
.icon-btn-hover:active {
  transform: scale(0.92);
}
.icon-btn-hover:focus-visible {
  box-shadow: var(--hover-shadow-btn);
  outline: 2px solid var(--glow);
  outline-offset: 2px;
}
```

**forest 主题下 `.card-hover` 行为**:

forest 已把 `--theme-shadow-dark/light` 置 0(app.css:133-134 / 223-224),因此 forest 的 `--hover-shadow-card`(派生自这两个 token)自然为零,`.card-hover:hover` 在 forest 下不会画任何阴影——**保持现状,无需特别处理**。forest 的卡片 hover 反馈仅靠 `hover:scale-[1.02]` + `.surface-flat` 的 border-color 加深(矩阵中已列)。批次 B 不动 forest 的 `--hover-shadow-card`。

**激活 `.pill-hover`**:把散落在 search-modal、load-more-button、theme-toggle 的内联 `hover:scale-*` / `hover:opacity-*` 收敛到 `.pill-hover`(pill 元素)或 `.icon-btn-hover`(圆形)。

### §4. 组件迁移清单

按文件组织(基于审计):

**A. 列表行(7 文件)**:
- `src/lib/components/home/activity-list.svelte:57` — `.surface-row` → `.list-row`
- `src/lib/components/home/nearby-users-flat-list.svelte:44` — `.surface-row` → `.list-row`
- `src/lib/components/common/user-card.svelte:212` — `.surface-row`(nearby-row/member-row variants)→ `.list-row`
- `src/lib/components/chat/chat-list-panel.svelte:225, 312` — `.row-hover` → `.list-row`
- `src/routes/(protected)/chat/+page.svelte:276, 361` — `.row-hover` 和 `.list-row-hover` → `.list-row`
- `src/routes/(protected)/profile/+page.svelte:360` — `.row-hover` → `.list-row`
- `src/routes/(protected)/notifications/+page.svelte:478, 583` — `.row-hover` → `.list-row`

**B. 内联 neumorphic 阴影清理(token 根治后剩余检查)**:
- `src/lib/components/common/create-group-modal.svelte:237` — `box-shadow: var(--shadow-sm)` → `.surface-flat` 或对应 class
- `src/lib/components/common/search-modal.svelte:316, 451` — `box-shadow: var(--shadow-sm)` → `.surface-flat`
- (其余 .svelte 内联 `box-shadow: var(--shadow-*)` 一并清理)

**C. Skeleton 填充换 token(8+ 文件)**:
- `src/lib/components/home/people-radar-strip.svelte:44`
- `src/lib/components/home/group-carousel.svelte:41, 47`
- `src/lib/components/home/nearby-users-flat-list.svelte:28, 32, 35`
- `src/lib/components/chat/chat-list-panel.svelte:204, 209, 213, 293, 298, 302`
- `src/lib/components/common/search-modal.svelte:381, 564`
- `src/lib/components/common/stat-segments.svelte:22, 33`
- 其他用 `var(--border-theme)` 或 `var(--theme-border)` 作填充的位置(全局 grep 确认)

`background: var(--border-theme)`(作填充用) → `background: var(--skeleton-bg)`。
**只替换"占位/skeleton 语义"的填充**,不动真正用作"描边色 tint"的位置。

**D. icon/pill 按钮内联 hover 收敛**:
- `src/lib/components/common/load-more-button.svelte:48` — `hover:scale-105` → `.pill-hover`
- `src/lib/components/common/theme-toggle.svelte:31, 43` — `hover:scale-105` → `.icon-btn-hover`(圆形)
- `src/lib/components/common/image-preview.svelte:100, 115, 124` — `hover:scale-110` → `.icon-btn-hover`
- CTA `hover:opacity-70/80/90`(settings/* 等)→ 统一为 `hover:brightness-110`(或保留 opacity 但统一数值,本批次取前者更轻)

**E. `.ambient-glow` 容器检查**(预防性):
- 审计所有用 `.ambient-glow` 的容器是否有 `overflow: hidden`,若没有且会与相邻 neumorphic 元素重叠,补齐。本批次仅在发现问题时处理。

### §5. 兼容与降级

- `.row-hover` / `.neu-row-hover` 保留为别名(行为改为 `.list-row`),不破坏既有引用,等组件迁移完后再考虑废弃。
- `.surface-row` 同样保留别名,等迁移完成。
- `.list-row-hover` 保留别名,等 chat/+page.svelte:361 迁移后再考虑废弃。
- `.circle-hover` 重命名为 `.icon-btn-hover`,但保留旧名作别名(`.circle-hover, .icon-btn-hover { ... }`),不破坏外部潜在引用。

## 实施分批

每批可独立验证,失败可单独回滚:

### 批次 A — Token 层(根因 + 陶瓷质感)
1. sand light `--theme-shadow-light`: `rgba(255,252,245,0.96)` → `rgba(251,232,216,0.5)`(暖桃白)
2. sand light `--theme-shadow-dark`: `rgba(167,145,116,0.38)` → `rgba(167,145,116,0.22)`
3. sand dark `--theme-bg`: `#211b17` → `#2a221c`
4. sand dark `--page-bg`: `#15110e` → `#18120e`
5. sand dark `--theme-shadow-light`: `rgba(255,218,179,0.055)` → `rgba(255,200,150,0.18)`(暖桃光)
6. sand dark `--theme-shadow-dark`: `rgba(0,0,0,0.58)` → `rgba(0,0,0,0.55)`
7. sand dark `--theme-text`: `#f1e4d2` → `#f5ead8`
8. sand dark `--theme-text-muted`: `#a99682` → `#b8a48c`
9. 全主题族新增 `--skeleton-bg` token(forest / indigo / sand,light + dark,共 6 处)
10. `.ambient-glow::before` sand 专属 opacity 0.4

**效果**:① 消除 light 下奶白方块;② 重建 dark 下陶瓷凸起感,与 forest 扁平暗色明显区分。

### 批次 B — Hover class 重构
1. 新增 `.list-row`(全主题 background-tint)
2. `.row-hover` / `.surface-row` / `.list-row-hover` 改为 `.list-row` 别名(行为统一)
3. `.circle-hover` 重命名 `.icon-btn-hover`(保留旧名别名)
4. 激活 `.pill-hover` 准备工作(class 已存在)

### 批次 C — 组件迁移
1. 列表行 7 文件:`.surface-row` / `.row-hover` / `.list-row-hover` → `.list-row`
2. 内联 `box-shadow: var(--shadow-*)` 清理 → `.surface-flat` 等 class
3. Skeleton 填充 → `--skeleton-bg`
4. icon/pill 按钮内联 hover → class
5. CTA `hover:opacity-*` → `hover:brightness-110`

## 验证

每批完成后:

```bash
bun check         # svelte-check 类型检查
bun run build     # 构建
bun lint          # prettier 格式
```

主题视觉验证(手动):
1. 在 sand light 主题下打开首页、附近的人、动态、聊天、通知、profile,确认无奶白方块。
2. 切换 forest、indigo 主题,确认 hover 行为按主题族正确表现。
3. 在 sand dark、indigo dark、forest dark 下重复 1–2。
4. 触摸设备(关闭 hover 媒体查询)确认 `.list-row` 在无 hover 时不变色。
5. **主题区分度验证(批次 A 后)**:并排截图 sand dark vs forest dark,确认能一眼分辨:
   - sand dark:卡片有可见凸起(暖光高光 + 暖阴阴影),整体偏暖棕
   - forest dark:扁平,无 neumorphic 阴影,整体偏冷绿
6. **陶瓷质感验证**:在 sand dark 下打开包含 `.surface-hero` / `.surface-pressed` 的卡片(moment-card、user-card square variant),确认能看到"凸起/凹陷"的体积感,而非纯扁平。

## 风险与回滚

| 风险 | 缓解 |
|---|---|
| sand light shadow 降到 0.5 后陶瓷手感变弱 | 0.5 仍保留明显温度 + 暖桃白色相加持;实测后若不满意可在 0.4–0.6 微调,单 token 改动 |
| sand dark 整体提亮后对比度不足 | bg 仅提亮 ~3%,page-bg 同步调整保持 15% 亮度差;text 同步提亮;若 WCAG 失败(对比度 < 4.5:1),回退 text 到 `#f1e4d2` |
| sand dark 阴影 1:3 比例过强,与 indigo dark(1:10)风格脱节 | 1:3 是 sand 专属的"立体陶瓷"语言,与 indigo "现代发光"刻意区分;若实测过冲,降 shadow-light 到 0.14 |
| `.row-hover` 别名行为变化破坏既有视觉 | 别名内部统一到 `.list-row` 行为,所有 6 处用 `.row-hover` 的元素都没有内联 background,迁移前后视觉差异是"box-shadow 叠加"→"background-color",更轻量更一致 |
| skeleton token 在 forest 下与现有表现不一致 | forest 的 `--page-bg` 与 `--surface-bg` 接近,color-mix 后差异极小,实测确认 |
| 内联 `box-shadow: var(--shadow-*)` 清理遗漏 | 全局 grep 三轮,组件迁移批次配合 |

回滚:每批是独立 commit,失败可 `git revert` 单批。

## 后续(非本轮)

- 废弃 `.row-hover` / `.surface-row` / `.list-row-hover` / `.circle-hover` 别名(等 1–2 个发布周期确认无外部依赖)。
- 把组件剩余的 scoped `<style>` 内 `:hover` 规则(auth-prompt、back-button、create-moment-modal、people-radar、discover tag-row)按本矩阵审计统一。
- 给所有命名 hover class 配 `:focus-visible` 键盘焦点态(部分组件 scoped style 缺失)。
