# Hover 按类型统一 + Sand 主题色彩重塑 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 sand 主题方块色块 + 重建 sand dark 陶瓷质感 + 把割裂的 4 套 hover 机制收敛到按类型/主题族统一的矩阵。

**Architecture:** 三批独立可回滚。批次 A(token 层)是根因,改完后 sand light 方块消失、sand dark 陶瓷凸起回归,并同步修复与 bg 联动的 glass/popover/preview 等"漂移 token"。批次 B(hover class 重构)定义新的 `.list-row` 并把现有 4 套机制收敛。批次 C(组件迁移)把 12 处 hover class、210+ 处内联 box-shadow、8 处缺 focus-visible 的 scoped hover 全部迁移到位。

**Tech Stack:** SvelteKit 5 + Tailwind v4(配置在 `app.css` 的 `@theme {}`,无 `tailwind.config.*`)+ CSS variables 设计 token 系统。

## Global Constraints

- **契约权威**:`spec/` 是 API 契约唯一权威——本计划不改任何接口/字段/错误码。
- **token-only**:`frontend/AGENTS.md` 设计规范要求"禁止硬编码十六进制色值"。本计划只改 `src/app.css` 的 token 定义和组件内联 style,不引入硬编码色值到组件层。例外:`PREVIEW_ACCENT` 这类"展示用色卡",必须直接对齐真实 token 值。
- **构建校验**:每批结束必须运行 `bun check`(svelte-check 类型检查)+ `bun run build` + `bun lint`。`package.json` 无 `test` 脚本,无自动化测试。
- **不破坏 dark toggle**:dark mode 由 `:root.dark` 触发(`app.css:3` `@custom-variant dark`),不要在组件层写 `.dark` 选择器。
- **触控 hover 守卫**:`app.css:34` 把 Tailwind `hover:` variant 重定义为 `(&:hover, @media (hover: hover))`——所有内联 `hover:*` 在触控设备自动失效。手写 CSS `.xxx:hover` 不受此影响,需要单独用 `@media (hover: hover)` 包装或加 `:focus-visible` 兜底。
- **复用现有原语**:不新增布局/容器组件,沿用 `.page-shell` / `.density-grid` / `.surface-*` 等。
- **commit 粒度**:每个 task 一次 commit,commit message 用中文 conventional commits 风格(`修复(UI): ...` / `重构(CSS): ...`),与项目历史一致。

## 当前工作树状态

- **stash@{0}**:`WIP: hover unification + sand theme fixes (in-progress from prior session)` — 上一会话的部分 hover 统一改动(11 文件)。Task 0 会恢复其中合理的部分。
- **未提交 HEAD**:`8780bea9 文档(spec): hover 按类型统一 + sand 主题色彩重塑` — spec 文档已 commit。
- **当前分支**:`main`。本计划全部在 `main` 上推进(项目约定,无 PR 流程)。

## File Structure

修改文件分三层:

- **`src/app.css`**(根因层,所有 token 与 hover class 在此):
  - L496–586 sand light 主题块
  - L588–676 sand dark 主题块
  - L130–218 forest light / L220–305 forest dark(参照,不动)
  - L857–1175 `.surface-*` / `.card-hover` / `.row-hover` / `.surface-row` / `.list-row-hover` / `.circle-hover` / `.pill-hover` 定义
  - L1184–1230 `.ambient-glow::before` opacity 覆盖
- **`src/lib/components/**/*.svelte` + `src/routes/**/*.svelte`**(组件层,迁移类改动):
  - 12 处 `.row-hover` / `.surface-row` / `.list-row-hover` → `.list-row`
  - 210+ 处内联 `box-shadow: var(--shadow-sm/md/lg/xl)` → `.surface-*` class
  - 8 处 skeleton 填充 → `var(--skeleton-bg)`
  - 8 处 scoped `:hover` 加 focus-visible + hover 媒体包装
- **`src/lib/components/ui/skeleton/skeleton.svelte`**(shimmer 核心):`:global(.shimmer)` 渐变接 `--skeleton-bg`

---

## Task 0: 恢复上一会话的合理改动

**Files:**
- Read: `git stash show -p stash@{0}` 内容(11 文件)

**Interfaces:**
- Consumes: stash@{0} 的 11 个改动文件
- Produces: 一个干净的"已应用部分合理改动"的工作树,作为后续 task 的起点

stash 里有价值的改动:
1. `app.css` 新增 `.surface-row` class(行 1116–1131)+ `.row-hover` 改成 `inset 9999px`(行 1093–1109)
2. `anchor-entry.svelte:32` 用 `surface-hero card-hover` class 替代内联 box-shadow
3. `user-card.svelte:212` 加 `class:surface-row` 条件类(nearby-row/member-row variants)
4. `avatar.svelte:29` 默认 `border = false`(去掉所有 avatar 的 1.5px border 环)
5. `activity-list.svelte` / `nearby-users-flat-list.svelte` 改用 `.surface-row` + 真 `border-b` 分割线
6. `people-radar-strip.svelte` skeleton 占位移除 dashed border
7. `moment-card.svelte` 移除 hover 时 border 变色(改用 `card-hover` 统一)
8. `notifications/+page.svelte` markAllAsRead/markAsRead 用 `$effect` 同步(逻辑简化,合理保留)

stash 里需要**丢弃**的改动:
- `app.css` 里 `.row-hover:hover { box-shadow: inset 0 0 0 9999px var(--border-theme) }` — Task 5 会重新定义为 background-color 方案(inset 9999px 仍是色块叠加,本质问题没解决)
- `chat/+page.svelte:276` 把 `.list-row-hover` 改成 `.row-hover` — Task 7 会统一到 `.list-row`

- [ ] **Step 1: 检查 stash 内容确认**

```bash
cd /Users/liyou/work/hotspot
git stash show -p stash@{0} | head -200
```

预期:看到上述 8 类改动。如果 stash 已不在或内容不符,跳到 Step 3 手动应用。

- [ ] **Step 2: pop stash**

```bash
git stash pop stash@{0}
git status
```

预期:工作树变 dirty,11 个文件 modified。

- [ ] **Step 3: 回滚 `.row-hover` 的 inset 9999px 改动**

用 Edit 把 `frontend/src/app.css` 里:

```css
  .row-hover,
  .neu-row-hover {
    transition: box-shadow 0.15s ease;
  }
  .row-hover:hover,
  .neu-row-hover:hover {
    box-shadow: inset 0 0 0 9999px var(--border-theme);
  }
  .row-hover:focus-visible,
  .neu-row-hover:focus-visible {
    box-shadow: inset 0 0 0 9999px var(--border-theme);
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

恢复成 stash 前的原始版本(从 git log 查):

```bash
git log --all --oneline -- frontend/src/app.css | head -5
git show <上一个 commit>:frontend/src/app.css | grep -A 12 "  \.row-hover," | head -20
```

然后用 Edit 把 `.row-hover` 块恢复成 HEAD 的原版本(只 transition + 用 `var(--hover-shadow-row)`)。Task 5 会重新定义。

- [ ] **Step 4: 暂时提交这一步的"中间态"**

```bash
git add frontend/src/app.css frontend/src/lib/components/ frontend/src/routes/
git commit -m "重构(UI): 恢复上一会话的 hover 统一中间态(待 Task 5 重新定义 .row-hover)"
```

预期:11 个文件 modified 全部 commit。后续 Task 在此基础上推进。

---

## Task 1: Sand 主题色彩重塑 — light mode shadow token + 联动 glass

**Files:**
- Modify: `frontend/src/app.css:499` sand light `--theme-shadow-light`
- Modify: `frontend/src/app.css:500` sand light `--theme-shadow-dark`
- Modify: `frontend/src/app.css:520` sand light `--theme-glass-border`(同步色系)

**Interfaces:**
- Consumes: 无
- Produces: sand light 的 neumorphic 阴影从"奶白方块"变成"陶瓷暖光阴影";所有用 `var(--shadow-light/dark)`、`--hover-shadow-*` 的 class 自动跟随

这一步只动 light mode,先验证方块消除。

- [ ] **Step 1: 改 sand light `--theme-shadow-light` 和 `--theme-shadow-dark`**

在 `frontend/src/app.css` 找到 L499–500(sand 主题块内):

```css
  --theme-shadow-light: rgba(255, 252, 245, 0.96);
  --theme-shadow-dark: rgba(167, 145, 116, 0.38);
```

改成:

```css
  /* 陶瓷暖光阴影:从 0.96 实心奶白(画方块)降到 0.5 暖桃白,回归阴影语义。
     暖桃白 #fbe8d8 与画布 #efe7da 形成色温对比,呈现"暖光打在陶瓷上"的质感。 */
  --theme-shadow-light: rgba(251, 232, 216, 0.5);
  --theme-shadow-dark: rgba(167, 145, 116, 0.22);
```

- [ ] **Step 2: 同步 sand light `--theme-glass-border` 色系**

L520:

```css
  --theme-glass-border: rgba(255, 252, 245, 0.65);
```

改成:

```css
  /* glass-border 与 shadow-light 同色系,保持陶瓷温度一致 */
  --theme-glass-border: rgba(251, 232, 216, 0.65);
```

- [ ] **Step 3: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
```

预期:`bun check` 0 errors;`bun run build` 成功。CSS token 改动不会引发类型/构建错误,但必须确认没有引入语法错误。

- [ ] **Step 4: 手动视觉验证**

启动 dev server:

```bash
bun run dev --cwd frontend &
sleep 4
```

打开浏览器 → 切换到 sand light 主题 → 检查:
1. 首页 surface-hero 卡片不再有奶白方块环绕
2. user-card、anchor-entry、moment-card 周围的"光晕方块"消失,变成柔和的暖色阴影
3. 触发 hover(`.card-hover:hover` → `--hover-shadow-card`)仍是可见的浮起,但不再扎眼

如果方块没完全消失或对比太弱,微调 `--theme-shadow-light` 在 0.4–0.6 之间。

杀掉 dev server:

```bash
pkill -f "bun.*dev" || true
```

- [ ] **Step 5: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "修复(UI): sand light shadow token 从 0.96 实心奶白降到 0.5 暖桃白,消除方块色块

- --theme-shadow-light: rgba(255,252,245,0.96) → rgba(251,232,216,0.5)
- --theme-shadow-dark: 0.38 → 0.22(同步降低,保持明暗对称)
- --theme-glass-border: 同步到暖桃白色系

所有用 var(--shadow-light/dark) 和 --hover-shadow-* 的 class 自动跟随,
.surface-hero / .surface-raised / .surface-convex / .card-hover / .row-hover
等不再在元素周围画出奶白方块。"
```

---

## Task 2: Sand dark 主题陶瓷质感重塑 + 联动 token 漂移修复

**Files:**
- Modify: `frontend/src/app.css:589` sand dark `--theme-bg`
- Modify: `frontend/src/app.css:590` sand dark `--page-bg`
- Modify: `frontend/src/app.css:591` sand dark `--theme-shadow-light`
- Modify: `frontend/src/app.css:592` sand dark `--theme-shadow-dark`
- Modify: `frontend/src/app.css:593` sand dark `--theme-text`
- Modify: `frontend/src/app.css:594` sand dark `--theme-text-muted`
- Modify: `frontend/src/app.css:611` sand dark `--theme-glass-bg`(漂移修复)
- Modify: `frontend/src/app.css:641` sand dark `--popover`(漂移修复)
- Modify: `frontend/src/app.css:621` sand dark `--overlay-soft`(漂移修复)

**Interfaces:**
- Consumes: Task 1 的 light mode 改动(已完成)
- Produces: sand dark 的陶瓷凸起感回归,与 forest dark 在第一眼可分辨("立体暖调 vs 扁平冷调")

- [ ] **Step 1: 改 sand dark 核心 token**

在 `frontend/src/app.css` 的 sand dark 块(L588–676)里,L589–594:

```css
  --theme-bg: #211b17;
  --page-bg: #15110e;
  --theme-shadow-light: rgba(255, 218, 179, 0.055);
  --theme-shadow-dark: rgba(0, 0, 0, 0.58);
  --theme-text: #f1e4d2;
  --theme-text-muted: #a99682;
```

改成:

```css
  /* 陶瓷深棕:bg/page 亮度差从 7% 提到 15%,卡片才凸出画布。
     shadow-light 从 0.055(几乎不可见)提到 0.18(陶瓷凸起感回归),
     light:dark 比例 1:10.5 → 1:3,立体而不被压平。
     text/muted 略提亮,避免与 forest dark 灰绿混淆。 */
  --theme-bg: #2a221c;
  --page-bg: #18120e;
  --theme-shadow-light: rgba(255, 200, 150, 0.18);
  --theme-shadow-dark: rgba(0, 0, 0, 0.55);
  --theme-text: #f5ead8;
  --theme-text-muted: #b8a48c;
```

- [ ] **Step 2: 修复 `--theme-glass-bg` 漂移**

L611:

```css
  --theme-glass-bg: rgba(33, 27, 23, 0.9);
```

(33,27,23) = 旧 `#211b17`。改成新 `#2a221c` 的 RGB(42, 34, 28):

```css
  --theme-glass-bg: rgba(42, 34, 28, 0.9);
```

- [ ] **Step 3: 修复 `--popover` 漂移**

L641:

```css
  --popover: #28211c;
```

`#28211c` 是介于旧 bg 和 page 之间的硬编码值。改成跟随新 bg 的 color-mix(更接近 bg 但略亮):

```css
  --popover: color-mix(in srgb, var(--theme-bg) 92%, var(--page-bg));
```

- [ ] **Step 4: 修复 `--overlay-soft` 漂移**

L621:

```css
  --overlay-soft: rgba(241, 228, 210, 0.08);
```

(241,228,210) = 旧 `--theme-text #f1e4d2`。改成新 text `#f5ead8` = (245, 234, 216):

```css
  --overlay-soft: rgba(245, 234, 216, 0.08);
```

- [ ] **Step 5: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
```

预期:0 errors。

- [ ] **Step 6: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

打开浏览器 → 切换到 sand **dark** 主题 → 与 forest dark 并排对比(可截图后对比):
1. sand dark:卡片有可见凸起(暖光高光 + 暖阴阴影),整体偏暖棕
2. forest dark:扁平无 neumorphic 阴影,整体偏冷绿
3. 第一眼能明显分辨两者(用户原诉求)
4. `.surface-hero` / `.surface-pressed` 卡片有"凸起/凹陷"体积感

杀掉 dev server。

- [ ] **Step 7: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "修复(UI): sand dark 陶瓷质感重塑 — 与 forest dark 区分

- --theme-bg #211b17 → #2a221c,--page-bg #15110e → #18120e(亮度差 7%→15%)
- --theme-shadow-light 0.055 → 0.18(暖桃光,陶瓷凸起可见)
- --theme-shadow-dark 0.58 → 0.55,light:dark 比例 1:10.5 → 1:3
- --theme-text #f1e4d2 → #f5ead8,--theme-text-muted #a99682 → #b8a48c
- 同步修复漂移 token: --theme-glass-bg / --popover / --overlay-soft

用户反馈 sand dark 与 forest dark 同质化;根因是 shadow-light 太弱
(0.055 几乎不可见)+ bg/page 亮度差仅 7%,陶瓷凸起感消失。"
```

---

## Task 3: 新增 `--skeleton-bg` token + 接入 skeleton 组件

**Files:**
- Modify: `frontend/src/app.css:217` forest light 末尾(追加 token)
- Modify: `frontend/src/app.css:305` forest dark 末尾
- Modify: `frontend/src/app.css:585` indigo light 末尾(注:indigo 块结束位置)
- Modify: `frontend/src/app.css:404` 起 indigo dark 末尾
- Modify: `frontend/src/app.css:586` sand light 末尾
- Modify: `frontend/src/app.css:676` sand dark 末尾
- Modify: `frontend/src/lib/components/ui/skeleton/skeleton.svelte:18,20`(shimmer 接 token)

**Interfaces:**
- Consumes: 无
- Produces: 全主题族统一 `--skeleton-bg` token;skeleton 组件不再借 `--border-theme` 做填充

- [ ] **Step 1: 在每个主题族块的末尾追加 `--skeleton-bg`**

对 6 个主题族块(forest light L130–218、forest dark L220–305、indigo light L312–404、indigo dark L404–491、sand light L496–586、sand dark L588–676)分别追加:

```css
  /* Skeleton 占位填充:比 surface 略深的同色调,
     作占位时是"凹陷感"而非 pigmented 色块。 */
  --skeleton-bg: color-mix(in srgb, var(--surface-bg) 65%, var(--page-bg));
```

具体插入位置:每个主题族块的 `--topbar-height-mobile: 3.5rem;` 行之前(若该块没有 topbar,则插到 `--ring:` 行之前)。forest dark 块没有 topbar,插到 `--sidebar-ring: ...` 行之后。

- [ ] **Step 2: 把 skeleton.svelte 的 shimmer 接到 `--skeleton-bg`**

读 `frontend/src/lib/components/ui/skeleton/skeleton.svelte` 全文,找到 L18、L20:

```css
    :global(.shimmer) {
      ...
      background: linear-gradient(
        90deg,
        var(--border-theme) 25%,  /* L18 */
        ...
        var(--border-theme) 75%   /* L20 */
      );
    }
```

把两处 `var(--border-theme)` 改成 `var(--skeleton-bg)`:

```css
        var(--skeleton-bg) 25%,
        ...
        var(--skeleton-bg) 75%
```

- [ ] **Step 3: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
```

预期:0 errors。

- [ ] **Step 4: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

打开浏览器 → 切换到 sand 主题 → 触发加载态(可在 chat 列表首次加载、附近的人加载时观察):
1. skeleton 占位条不再显示为 pigmented tan 方块,显示为更柔和的"凹陷感"米色
2. 切换到 forest/indigo,skeleton 占位色仍合理

杀掉 dev server。

- [ ] **Step 5: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css frontend/src/lib/components/ui/skeleton/skeleton.svelte
git commit -m "重构(CSS): 新增 --skeleton-bg token,skeleton 占位不再借 --border-theme 做填充

skeleton 之前用 --border-theme(在 sand 下是 9% 棕褐)做填充色,
在方形元素上读起来是 pigmented tan 方块。改为专门的 --skeleton-bg
(color-mix(surface-bg 65%, page-bg))后,占位是"凹陷感"而非色块。

全 6 个主题族块(forest/indigo/sand × light/dark)统一接入。"
```

---

## Task 4: `.ambient-glow::before` sand 专属 opacity

**Files:**
- Modify: `frontend/src/app.css:1226`(在 `.dark .ambient-glow::before` 后追加 sand 覆盖)

**Interfaces:**
- Consumes: 无
- Produces: sand 主题下 `.ambient-glow::before` terracotta glow opacity 从默认 0.55 降到 0.4

- [ ] **Step 1: 在 ambient-glow 覆盖块末尾追加 sand 规则**

读 `frontend/src/app.css` L1213–1230:

```css
  [data-theme-family='forest'] .ambient-glow::before,
  [data-theme-family='forest'] .neu-ambient-glow::before,
  [data-theme-family='forest'].dark .ambient-glow::before,
  [data-theme-family='forest'].dark .neu-ambient-glow::before {
    opacity: 0.95;
  }
  [data-theme-family='indigo'] .ambient-glow::before,
  [data-theme-family='indigo'] .neu-ambient-glow::before {
    opacity: 0.7;
  }
  [data-theme-family='indigo'].dark .ambient-glow::before,
  [data-theme-family='indigo'].dark .neu-ambient-glow::before {
    opacity: 0.8;
  }
  .dark .ambient-glow::before,
  .dark .neu-ambient-glow::before {
    opacity: 0.85;
  }
```

在 `.dark .ambient-glow::before` 块**之后**追加(specificity 必须 > `.dark`,所以 family-qualified 选择器即可;源序在后更稳妥):

```css
  /* sand terracotta glow 在奶白/暖棕底上过亮,降到 0.4。
     必须放在 .dark .ambient-glow::before 之后才能覆盖 sand dark 的 0.85。 */
  [data-theme-family='sand'] .ambient-glow::before,
  [data-theme-family='sand'] .neu-ambient-glow::before,
  [data-theme-family='sand'].dark .ambient-glow::before,
  [data-theme-family='sand'].dark .neu-ambient-glow::before {
    opacity: 0.4;
  }
```

- [ ] **Step 2: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

- [ ] **Step 3: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

切换 sand 主题 → 查看用 `.ambient-glow` 的卡片(如 hero-stats、anchor-summary-card):
1. sand 下顶部 terracotta 光晕不再刺眼,柔和

杀掉 dev server。

- [ ] **Step 4: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "修复(UI): sand 主题 .ambient-glow terracotta glow opacity 从 0.55 降到 0.4

sand 的 terracotta glow 在奶白/暖棕底上过亮,在默认 0.55 下显眼;
与新的暖光阴影配合 0.4 正好够亮且不抢戏。

规则必须放在 .dark .ambient-glow::before 之后才能覆盖 sand dark 的 0.85。"
```

---

## Task 5: 定义 `.list-row` + 重构 `.row-hover` / `.surface-row` / `.list-row-hover` 为别名

**Files:**
- Modify: `frontend/src/app.css:1093–1131`(替换 `.row-hover` 和 `.surface-row` 块)
- Modify: `frontend/src/app.css:1133–1145`(`.list-row-hover` 块)

**Interfaces:**
- Consumes: 无
- Produces: 新的 `.list-row` class(全主题统一 background-tint hover);旧名 `.row-hover` / `.surface-row` / `.list-row-hover` 成为别名,行为统一

- [ ] **Step 1: 找到 `.row-hover` 当前定义位置**

```bash
cd /Users/liyou/work/hotspot
grep -n "^  \.row-hover\|^  \.surface-row\|^  \.list-row-hover" frontend/src/app.css
```

预期:看到 3 处定义,大致在 L1093 / L1116 / L1133 附近。

- [ ] **Step 2: 用 Edit 把 `.row-hover` 块替换成 `.list-row` + 别名**

读出 L1090–1145 的完整内容(从注释 `/* ── Row hover overlay */` 到 `.list-row-hover:focus-visible` 结束)。用 Edit 把整段替换为:

```css
  /* ── Unified list-row hover ─────────────────────────────────────
     全主题统一:密集列表行用 background-tint,无阴影、无 transform。
     避免 neumorphic 浮起在 sand 下方块化,避免 inset overlay 在 neumorphic
     主题下与既有阴影冲突。
     - hover: --border-theme tint(轻度,9-18% 描边色作 tint)
     - active: --border-active(按下加重)
     - focus-visible: --glow outline 环(键盘可达性)
     @media (hover: hover) 包装避免触控设备误触发。 */
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

  /* @deprecated 别名 — 已统一到 .list-row 行为。
     审计确认所有 6 处 .row-hover 元素都没有内联 background,
     可以安全从 box-shadow inset overlay 改为 background-color。
     短期保留以兼容既有引用,等组件迁移完成后(Task 7)考虑废弃。 */
  .row-hover,
  .neu-row-hover {
    background-color: var(--surface-bg);
    transition: background-color 0.15s ease;
  }
  @media (hover: hover) {
    .row-hover:hover,
    .neu-row-hover:hover {
      background-color: var(--border-theme);
    }
  }
  .row-hover:active,
  .neu-row-hover:active {
    background-color: var(--border-active);
  }
  .row-hover:focus-visible,
  .neu-row-hover:focus-visible {
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }

  /* @deprecated 别名 — 已统一到 .list-row 行为。
     1 处引用(chat/+page.svelte:361 group chat 行),Task 7 迁移后废弃。 */
  .surface-row {
    background-color: var(--surface-bg);
    transition: background-color 0.15s ease;
  }
  @media (hover: hover) {
    .surface-row:hover {
      background-color: var(--border-theme);
    }
  }
  .surface-row:active {
    background-color: var(--border-active);
  }
  .surface-row:focus-visible {
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }

  /* @deprecated 别名 — 已统一到 .list-row 行为。 */
  .list-row-hover {
    background-color: var(--surface-bg);
    transition: background-color 0.15s ease;
  }
  @media (hover: hover) {
    .list-row-hover:hover {
      background-color: var(--border-theme);
    }
  }
  .list-row-hover:active {
    background-color: var(--border-active);
  }
  .list-row-hover:focus-visible {
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

- [ ] **Step 3: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

- [ ] **Step 4: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

切换 sand 主题 → 查看用 `.row-hover` / `.surface-row` / `.list-row-hover` 的位置(chat 列表、notifications、profile、activity-list、nearby-users-flat-list、user-card nearby-row):
1. hover 时背景色变成柔和 tint,无方块色块
2. 触控设备(用 Chrome DevTools 模拟移动端)hover 不触发
3. Tab 键聚焦显示 outline 环

杀掉 dev server。

- [ ] **Step 5: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "重构(CSS): 新增 .list-row 统一 hover, .row-hover/.surface-row/.list-row-hover 统一为别名

新增 .list-row class(全主题统一 background-tint, 无阴影/无 transform):
- hover: --border-theme tint
- active: --border-active
- focus-visible: --glow outline
- @media (hover: hover) 包装避免触控误触发

.row-hover / .surface-row / .list-row-hover 行为统一到 .list-row,
消除之前 4 套机制(.row-hover box-shadow inset 9999px / .surface-row /
.list-row-hover accent-soft / 内联 hover:bg-*)的视觉割裂。

旧 class 暂保留为别名,Task 7 把组件迁移到 .list-row 后再考虑废弃。"
```

---

## Task 6: 把 `.circle-hover` 重命名为 `.icon-btn-hover`(0 引用,可安全改名)

**Files:**
- Modify: `frontend/src/app.css:1147–1160`(重命名 + 保留旧名别名)

**Interfaces:**
- Consumes: 无
- Produces: 新名 `.icon-btn-hover` 用于圆形 icon 按钮 hover(替代散乱的内联 `hover:scale-*`);旧名 `.circle-hover` 保留别名

- [ ] **Step 1: 用 Edit 把 `.circle-hover` 块替换**

读 L1147–1160(`.circle-hover` 定义),用 Edit 替换为:

```css
  /* ── Icon button hover (圆形 icon 按钮) ──────────────────────────
     neumorphic 浮起阴影 + 按下缩放反馈。用于圆形 icon 按钮,
     替代散乱的内联 hover:scale-105/110(Task 8 迁移)。
     forest 主题下 shadow 全 0,自动降级为只有 active 缩放。 */
  .icon-btn-hover,
  .circle-hover,  /* @deprecated alias */
  .neu-circle-hover {
    transition: box-shadow 0.2s cubic-bezier(0.22, 1, 0.36, 1),
                transform 0.15s ease;
  }
  .icon-btn-hover:hover,
  .circle-hover:hover,
  .neu-circle-hover:hover {
    box-shadow: var(--hover-shadow-btn);
  }
  .icon-btn-hover:active,
  .circle-hover:active,
  .neu-circle-hover:active {
    transform: scale(0.92);
  }
  .icon-btn-hover:focus-visible,
  .circle-hover:focus-visible,
  .neu-circle-hover:focus-visible {
    box-shadow: var(--hover-shadow-btn);
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

- [ ] **Step 2: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -5
bun run build 2>&1 | tail -5
```

- [ ] **Step 3: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/app.css
git commit -m "重构(CSS): .circle-hover 重命名为 .icon-btn-hover(语义更清晰),保留旧名别名

旧名 .circle-hover 当前 0 引用,可安全改名。新名 .icon-btn-hover
配合 Task 8 把散乱的内联 hover:scale-105/110 收敛到 class。"
```

---

## Task 7: 组件迁移 — 列表行统一到 `.list-row`(12 处)

**Files:**
- Modify: `frontend/src/lib/components/home/activity-list.svelte:57`
- Modify: `frontend/src/lib/components/home/nearby-users-flat-list.svelte:44`
- Modify: `frontend/src/lib/components/common/user-card.svelte:212`
- Modify: `frontend/src/lib/components/chat/chat-list-panel.svelte:225, 312`
- Modify: `frontend/src/routes/(protected)/chat/+page.svelte:276, 361`
- Modify: `frontend/src/routes/(protected)/profile/+page.svelte:360`
- Modify: `frontend/src/routes/(protected)/notifications/+page.svelte:478, 583`

**Interfaces:**
- Consumes: Task 5 定义的 `.list-row` class
- Produces: 12 处列表行 hover 全部统一到 `.list-row`

- [ ] **Step 1: `activity-list.svelte:57` — `.surface-row` → `.list-row`**

读文件确认 L57 附近,把 class 字符串里的 `surface-row` 改成 `list-row`。原:

```svelte
        class="surface-row flex items-center gap-3 px-4 py-3 text-left w-full border-b border-[var(--border-theme)] last:border-b-0"
```

改为:

```svelte
        class="list-row flex items-center gap-3 px-4 py-3 text-left w-full border-b border-[var(--border-theme)] last:border-b-0"
```

- [ ] **Step 2: `nearby-users-flat-list.svelte:44` — `.surface-row` → `.list-row`**

同样替换。原 `class="surface-row flex ..."` → `class="list-row flex ..."`。

- [ ] **Step 3: `user-card.svelte:212` — `.surface-row` 条件类 → `.list-row`**

读 L212 附近:

```svelte
  class:surface-row={variant === 'nearby-row' || variant === 'member-row'}
```

改为:

```svelte
  class:list-row={variant === 'nearby-row' || variant === 'member-row'}
```

- [ ] **Step 4: `chat-list-panel.svelte:225, 312` — `.row-hover` → `.list-row`**

读 L225 附近,把 `transition-all row-hover` 改成 `list-row`(去掉 `transition-all`,因 `.list-row` 自带 transition):

```svelte
          class="flex items-center gap-3.5 cursor-pointer transition-all row-hover"
```

改为:

```svelte
          class="list-row flex items-center gap-3.5 cursor-pointer"
```

L312 同样处理。

- [ ] **Step 5: `chat/+page.svelte:276` — `.row-hover` → `.list-row`**

读 L276 附近:

```svelte
                    class="flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3 row-hover active:scale-[0.98]"
```

改为(去掉 `active:scale-[0.98]`,`.list-row:active` 已用 background-color 表达按下):

```svelte
                    class="list-row flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3"
```

- [ ] **Step 6: `chat/+page.svelte:361` — `.list-row-hover` → `.list-row`**

读 L361 附近:

```svelte
                  class="flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3 transition-all list-row-hover active:scale-[0.98]"
```

改为:

```svelte
                  class="list-row flex items-center gap-3.5 rounded-[var(--radius-md)] px-4 py-3"
```

- [ ] **Step 7: `profile/+page.svelte:360` — `.row-hover` → `.list-row`**

读 L360 附近,把 `class="flex items-center gap-3 px-4 py-3 row-hover"` 改为 `class="list-row flex items-center gap-3 px-4 py-3"`。

- [ ] **Step 8: `notifications/+page.svelte:478, 583` — `.row-hover` → `.list-row`**

读 L478 附近:

```svelte
                  class="row-hover flex items-start gap-3.5 rounded-[var(--radius-md)] px-3 py-3.5 transition-transform duration-150 active:scale-[0.98]"
```

改为:

```svelte
                  class="list-row flex items-start gap-3.5 rounded-[var(--radius-md)] px-3 py-3.5"
```

L583 同样处理。

- [ ] **Step 9: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
bun lint 2>&1 | tail -5
```

预期:0 errors。

- [ ] **Step 10: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

依次打开主页 activity list / 附近的人 / chat 列表 / chat 着陆页 / profile / notifications:
1. 列表行 hover 在 sand / forest / indigo 三个主题下表现一致(背景 tint)
2. 触控设备 hover 不触发
3. Tab 键聚焦显示 outline 环

杀掉 dev server。

- [ ] **Step 11: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/ frontend/src/routes/
git commit -m "重构(UI): 列表行 hover 统一到 .list-row(12 处,跨 7 文件)

涉及:
- activity-list.svelte:57 (.surface-row → .list-row)
- nearby-users-flat-list.svelte:44 (.surface-row → .list-row)
- user-card.svelte:212 (.surface-row → .list-row,nearby-row/member-row 变体)
- chat-list-panel.svelte:225,312 (.row-hover → .list-row)
- chat/+page.svelte:276,361 (.row-hover + .list-row-hover → .list-row)
- profile/+page.svelte:360 (.row-hover → .list-row)
- notifications/+page.svelte:478,583 (.row-hover → .list-row)

同时移除冗余的 active:scale-[0.98] / transition-all / transition-transform,
.list-row :active 已用 --border-active 背景表达按下态。

消除之前 4 套 hover 机制在列表行上的视觉割裂。"
```

---

## Task 8: 清理 hand-rolled neumorphic 阴影(8 处核心)

**Files:**
- Modify: `frontend/src/lib/components/common/anchor-entry.svelte`(已在 stash 中迁移到 `.surface-hero`)
- Modify: `frontend/src/lib/components/common/user-card.svelte`(square variant 内层 inset)
- Modify: `frontend/src/lib/components/common/page-loading.svelte:12`
- Modify: `frontend/src/lib/components/common/online-dot.svelte:35`
- Modify: `frontend/src/lib/components/common/map-card.svelte:82`
- Modify: `frontend/src/routes/(protected)/profile/+page.svelte:652`
- Modify: `frontend/src/routes/(public)/groups/[id]/+page.svelte:1751, 1883, 1922`
- Modify: `frontend/src/routes/(protected)/settings/+page.svelte:296, 512`(主题预览)

**Interfaces:**
- Consumes: Task 1/2 后的 sand token + 既有 `.surface-*` class
- Produces: 移除绕过 token 体系的内联 neumorphic 阴影,避免它们继续画方块/与 token 改动脱节

注意:`.surface-flat` **没有 box-shadow**,不能直接用它替代内联 box-shadow。替代策略:
- 内联 neumorphic 阴影 → 对应 `.surface-*` class(`.surface-hero` / `.surface-raised` / `.surface-pressed` / `.surface-convex`)
- 内联 glow shadow(`0 0 20px var(--glow)`)→ 保留(语义正确,只是手写)
- 真正不需要的装饰阴影 → 移除

- [ ] **Step 1: 验证 anchor-entry.svelte 已用 `.surface-hero`**

```bash
grep -n "surface-hero\|box-shadow" frontend/src/lib/components/common/anchor-entry.svelte
```

预期:看到 `.surface-hero`,看不到内联 `box-shadow`。如果 stash 已应用,应该 OK。否则手动改。

- [ ] **Step 2: user-card.svelte square variant 内层 inset → `.surface-pressed`**

读 L296 附近(Task 0 stash 后的状态):

```svelte
      <div
        class="surface-pressed rounded-[var(--radius-md)] p-3"
      >
```

如果仍是内联 box-shadow,改成 `.surface-pressed` class(去掉 `style="background: var(--surface-bg); box-shadow: inset 3px 3px 6px var(--shadow-dark), inset -3px -3px 6px var(--shadow-light); border: 1px solid var(--border-theme);"` 全部)。

- [ ] **Step 3: page-loading.svelte:12 — `0 0 20px var(--glow)` 保留**

读 L12:`box-shadow: 0 0 20px var(--color-glow-primary);`。这是合理的 glow 阴影,**保留不动**。

- [ ] **Step 4: online-dot.svelte:35 — 保留**

读 L35:`box-shadow: 0 0 0 3px color-mix(in srgb, var(--semantic-online) 22%, transparent);`。这是 ring 不是 neumorphic,**保留**。

- [ ] **Step 5: map-card.svelte:82 — `0 0 0 4px var(--glow)` 保留**

ring shadow,**保留**。

- [ ] **Step 6: profile/+page.svelte:652 — `0 4px 14px color-mix(theme-success 35%)` 保留**

success glow,**保留**(语义正确)。

- [ ] **Step 7: groups/[id]/+page.svelte:1751, 1883, 1922 — error glow 保留**

读这三处:`box-shadow: 0 4px 14px var(--theme-error-border);`。destructive glow,**保留**。

- [ ] **Step 8: settings/+page.svelte:296, 512 — 主题预览卡片硬编码 neumorphic**

读 settings/+page.svelte L290–320 附近(主题选择器)。找到 `5px 5px 12px rgba(0,0,0,0.10); border: 1.5px solid` 模板字面量。这里硬编码了 neumorphic,**不会跟随任何 token 改动**——是 sand 主题"门面"的 bug。

把 `rgba(0,0,0,0.10)` 改成动态从 token 取值。具体改动需要读完整模板字符串后,把硬编码部分替换为 `var(--shadow-md)` 或基于 `accent` 的 color-mix:

```typescript
// 找到 PREVIEW_ACCENT 附近构造 inline style 的地方
// 把 ', 5px 5px 12px rgba(0,0,0,0.10); border: 1.5px solid ' +
// 改成更轻量的 ', box-shadow: 4px 4px 10px rgba(0,0,0,0.08); border: 1.5px solid ' +
```

读具体上下文后决定。最小改动:把 `rgba(0,0,0,0.10)` 改成 `rgba(0,0,0,0.06)`(更轻),让预览不再误导。L296 和 L512 两处都改。

- [ ] **Step 9: 同步 `PREVIEW_ACCENT.sand` 与真实 token**

读 settings/+page.svelte L29–31:

```typescript
const PREVIEW_ACCENT = {
  indigo: '#6366f1',
  forest: '#3D7A55',
  sand: '#D4845A',
};
```

把 `sand: '#D4845A'` 改成 sand light 的实际 accent `'#b86f4b'`(对齐 `--theme-accent-primary`):

```typescript
const PREVIEW_ACCENT = {
  indigo: '#6366f1',
  forest: '#3D7A55',
  sand: '#b86f4b',
};
```

- [ ] **Step 10: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
bun lint 2>&1 | tail -5
```

- [ ] **Step 11: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

打开 settings 页(主题选择器):
1. sand 主题预览卡片的 accent 显示为正确的赤陶色,不再是偏粉的 `#D4845A`
2. 预览卡片的阴影柔和,不误导用户以为 sand 是深色重的主题

杀掉 dev server。

- [ ] **Step 12: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/ frontend/src/routes/
git commit -m "修复(UI): 清理绕过 token 的内联 neumorphic + 修正 sand 主题预览

- settings/+page.svelte 主题预览卡片硬编码 rgba(0,0,0,0.10) → 0.06(更轻)
- PREVIEW_ACCENT.sand '#D4845A' → '#b86f4b'(对齐 --theme-accent-primary)
  之前预览色与真实 token 不一致,会误导用户选主题
- user-card.svelte square variant 内层 inset box-shadow → .surface-pressed class

保留语义正确的 glow ring(online-dot / map-card / profile success / 
groups destructive error),它们不是 neumorphic 方块问题来源。"
```

---

## Task 9: Skeleton 占位填充迁移到 `--skeleton-bg`(8 文件)

**Files:**
- Modify: `frontend/src/lib/components/home/people-radar-strip.svelte:44`
- Modify: `frontend/src/lib/components/home/group-carousel.svelte:41, 47`
- Modify: `frontend/src/lib/components/home/nearby-users-flat-list.svelte:31, 35, 38`
- Modify: `frontend/src/lib/components/chat/chat-list-panel.svelte:204, 209, 213, 293, 298, 302`
- Modify: `frontend/src/lib/components/common/stat-segments.svelte:22, 33`
- Modify: `frontend/src/lib/components/common/search-modal.svelte:381, 564`

**Interfaces:**
- Consumes: Task 3 定义的 `--skeleton-bg` token
- Produces: skeleton 占位填充不再借 `--border-theme`,在 sand 下不再读起来像 tan 方块

注意:只替换**占位/skeleton 语义**的 `background: var(--border-theme)`。真正用作分割线(border-like)的 `--border-theme` 不动(如 separator.svelte、scroll-area-scrollbar.svelte、admin-sidebar.svelte 等)。

- [ ] **Step 1: people-radar-strip.svelte:44 — 56px 占位 avatar**

读 L41–47 附近(loading 态)。找到 `background: var(--border-theme)` 用作 avatar 占位的:

```svelte
          <div
            style="width: 56px; height: 56px; border-radius: var(--radius-pill); background: var(--border-theme);"
          ></div>
```

改成:

```svelte
          <div
            style="width: 56px; height: 56px; border-radius: var(--radius-pill); background: var(--skeleton-bg);"
          ></div>
```

- [ ] **Step 2: group-carousel.svelte:41, 47 — 72px image 占位 + text bar**

读 L41、L47 附近。把用作占位填充的 `background: var(--border-theme)` → `var(--skeleton-bg)`。

- [ ] **Step 3: nearby-users-flat-list.svelte:31, 35, 38 — avatar + text bars**

读 L31、L35、L38 附近。同样替换占位填充。注意区分:这些是占位(skeleton 语义),不是 divider。

- [ ] **Step 4: chat-list-panel.svelte:204, 209, 213, 293, 298, 302 — skeleton 加载态**

读这 6 行附近,把 skeleton 占位 `background: var(--border-theme)` → `var(--skeleton-bg)`。

- [ ] **Step 5: stat-segments.svelte:22, 33 — segment container + value bar**

读 L22、L33 附近,替换占位填充。

- [ ] **Step 6: search-modal.svelte:381, 564 — search 占位**

读 L381、L564 附近,替换。

- [ ] **Step 7: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
bun lint 2>&1 | tail -5
```

- [ ] **Step 8: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

切换 sand 主题 → 触发各组件的 loading 态(强制刷新 chat/首页/search):
1. 占位条/avatar 不再显示为 pigmented tan 方块
2. 显示为更柔和的"凹陷感"米色

杀掉 dev server。

- [ ] **Step 9: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/
git commit -m "重构(UI): skeleton 占位填充迁移到 --skeleton-bg(8 文件)

涉及 people-radar-strip / group-carousel / nearby-users-flat-list /
chat-list-panel / stat-segments / search-modal 共 8 文件的 skeleton 占位。

之前用 --border-theme(在 sand 下是 9% 棕褐)做填充,在方形元素上读起来
是 pigmented tan 方块。改为 --skeleton-bg 后,占位是"凹陷感"而非色块。

只替换占位/skeleton 语义,真正用作分割线的 --border-theme 不动。"
```

---

## Task 10: icon/pill 按钮内联 hover 收敛到 `.icon-btn-hover` / `.pill-hover`(核心 5 处)

**Files:**
- Modify: `frontend/src/lib/components/common/theme-toggle.svelte:31, 43`(2 个圆形按钮)
- Modify: `frontend/src/lib/components/common/load-more-button.svelte:48`(pill)
- Modify: `frontend/src/lib/components/common/image-preview.svelte:100, 115, 124`(3 个圆形 overlay 按钮)

**Interfaces:**
- Consumes: Task 6 重命名的 `.icon-btn-hover` + 既有 `.pill-hover`
- Produces: 6 处 icon/pill 按钮的 `hover:scale-*` 收敛到对应 class

scope 说明:`.icon-btn-hover` / `.pill-hover` 已含 `:active` 缩放,迁移时去掉冗余的 `active:scale-*`(若元素已有)。

- [ ] **Step 1: theme-toggle.svelte:31, 43 — 圆形按钮**

读 L31、L43 附近(两个 theme-toggle 按钮,desktop + mobile)。原(示例):

```svelte
    class="... rounded-full ... transition-transform hover:scale-105 active:scale-95"
```

改为:

```svelte
    class="... rounded-full ... icon-btn-hover"
```

去掉 `transition-transform`、`hover:scale-105`、`active:scale-95`(`.icon-btn-hover` 已包含这些行为)。

- [ ] **Step 2: load-more-button.svelte:48 — pill 按钮**

读 L48 附近。原:

```svelte
    class="... rounded-full ... hover:scale-105 active:scale-95"
```

改为:

```svelte
    class="... rounded-full ... pill-hover"
```

- [ ] **Step 3: image-preview.svelte:100, 115, 124 — 3 个圆形 overlay 按钮**

读 L100、L115、L124 附近(close / prev / next)。每个都是圆形 `<button>` with `hover:scale-110`。改为 `.icon-btn-hover`。注意这些按钮本身有 `opacity-*` 表示 hover 反白,需保留 opacity 行为,只替换 scale:

读具体 class 字符串后判断。如果 class 是 `... hover:scale-110 ...`,改成 `.icon-btn-hover`(并保留其他 opacity 等行为)。最小改法:在 class 字符串里把 `hover:scale-110` 删掉,加上 `icon-btn-hover`。

- [ ] **Step 4: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
bun lint 2>&1 | tail -5
```

- [ ] **Step 5: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

切换 sand 主题 → 测试 theme-toggle(主题切换按钮)、load-more-button(列表底部加载更多)、image-preview(动态图片预览的 close/prev/next):
1. hover 时圆形按钮浮起 neumorphic 阴影,按下时 0.92 缩放
2. forest 主题下阴影自动消失,只剩缩放反馈
3. Tab 键聚焦显示 outline 环

杀掉 dev server。

- [ ] **Step 6: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/
git commit -m "重构(UI): icon/pill 按钮内联 hover:scale-* 收敛到 .icon-btn-hover / .pill-hover

涉及 theme-toggle (2 处圆形) / load-more-button (1 处 pill) /
image-preview (3 处圆形 overlay 按钮)。

之前这些按钮各自用 hover:scale-105/110 + active:scale-95 等内联 utility,
数值不统一。收敛到 .icon-btn-hover (圆形, neumorphic 浮起 + 0.92 按下) /
.pill-hover (pill, neumorphic 浮起) 后:
- sand/indigo: neumorphic 阴影 + 缩放双重反馈
- forest: shadow 全 0 自动降级,只剩缩放
- 全主题: focus-visible outline 环(键盘可达性)"
```

---

## Task 11: 修复 8 处缺 focus-visible 的 scoped `<style>` `:hover` 规则

**Files:**
- Modify: `frontend/src/lib/components/home/people-radar-strip.svelte:95`(`.radar-user:hover` 缺 focus-visible + 缺 hover 媒体包装)
- Modify: `frontend/src/lib/components/moments/moment-card.svelte:351`(`.moment-card:hover` border 变色,缺 focus-visible)
- Modify: `frontend/src/lib/components/common/create-moment-modal.svelte:570, 732, 747`(3 处缺 focus-visible)
- Modify: `frontend/src/lib/components/common/auth-prompt.svelte:128, 146`(2 处缺 focus-visible)

**Interfaces:**
- Consumes: 无
- Produces: 8 处 scoped hover 规则补充 `:focus-visible` 与 `@media (hover: hover)` 包装,修复键盘可达性 + 触控误触发

scope 说明:`location-picker-modal.svelte:2932` 的 `:global(.map-cluster-bubble:hover)` 是 map 聚类气泡,涉及地图交互改造,放后续批次(不在本轮)。

- [ ] **Step 1: people-radar-strip.svelte:95 — `.radar-user:hover`**

读 L90–100 附近。原:

```css
  .radar-user {
    ...
    transition: transform 0.2s cubic-bezier(0.22, 1, 0.36, 1);
  }
  .radar-user:hover {
    transform: scale(1.02);
  }
```

改为(用 hover 媒体包装 + 加 focus-visible):

```css
  .radar-user {
    ...
    transition: transform 0.2s cubic-bezier(0.22, 1, 0.36, 1);
  }
  @media (hover: hover) {
    .radar-user:hover {
      transform: scale(1.02);
    }
  }
  .radar-user:focus-visible {
    transform: scale(1.02);
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

- [ ] **Step 2: moment-card.svelte:351 — `.moment-card:hover` border 变色**

读 L345–360 附近。原:

```css
  .moment-card {
    ...
    transition: border-color 0.22s ease;
  }
  .moment-card:hover {
    border-color: color-mix(in srgb, var(--brand) 30%, var(--border-theme));
  }
```

如果 Task 0 stash 已应用,这条 hover 规则可能已被移除。先 grep 确认:

```bash
grep -n "moment-card:hover" frontend/src/lib/components/moments/moment-card.svelte
```

- 若已移除:跳过此步
- 若仍存在:把整个 `:hover` 块用 `@media (hover: hover)` 包装,并加 `:focus-visible` 同样的 border 变化:

```css
  @media (hover: hover) {
    .moment-card:hover {
      border-color: color-mix(in srgb, var(--brand) 30%, var(--border-theme));
    }
  }
  .moment-card:focus-visible {
    border-color: color-mix(in srgb, var(--brand) 30%, var(--border-theme));
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

- [ ] **Step 3: create-moment-modal.svelte:570 — `.compose__close:hover`**

读 L565–575 附近。原 `.compose__close:hover` 改为:

```css
  @media (hover: hover) {
    .compose__close:hover {
      background: var(--border-theme);
      border-color: var(--border-theme);
    }
  }
  .compose__close:focus-visible {
    background: var(--border-theme);
    border-color: var(--border-theme);
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

- [ ] **Step 4: create-moment-modal.svelte:732, 747 — `.compose__image-add/__image-empty:hover`**

读 L730–750 附近。两处 `:hover { border-color: var(--brand) !important }` 改为:

```css
  @media (hover: hover) {
    .compose__image-add:hover {
      border-color: var(--brand) !important;
    }
  }
  .compose__image-add:focus-visible {
    border-color: var(--brand) !important;
    outline: 2px solid var(--glow);
    outline-offset: 2px;
  }
```

`.compose__image-empty` 同样。

- [ ] **Step 5: auth-prompt.svelte:128, 146 — dismiss + login 按钮**

读 L125–150 附近。两处 `:hover` 改为带 `@media (hover: hover)` 包装 + `:focus-visible`:

`.auth-prompt-dismiss:hover` → 加包装 + focus-visible 让 color 变 `--text-primary` + outline 环
`.auth-prompt-login:hover` → 加包装 + focus-visible 让 opacity 0.9 + outline 环

- [ ] **Step 6: 构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -10
bun run build 2>&1 | tail -10
bun lint 2>&1 | tail -5
```

- [ ] **Step 7: 手动视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

测试:
1. people-radar 圆形头像:Tab 聚焦显示 outline + 微缩放
2. create-moment-modal:关闭按钮、添加图片按钮 Tab 聚焦显示 outline
3. auth-prompt:dismiss / login 按钮 Tab 聚焦显示 outline

杀掉 dev server。

- [ ] **Step 8: commit**

```bash
cd /Users/liyou/work/hotspot
git add frontend/src/lib/components/
git commit -m "修复(a11y): 8 处 scoped :hover 补齐 :focus-visible + @media (hover: hover)

涉及:
- people-radar-strip.svelte .radar-user:hover
- moment-card.svelte .moment-card:hover (若仍存在)
- create-moment-modal.svelte .compose__close / __image-add / __image-empty :hover
- auth-prompt.svelte .auth-prompt-dismiss / __login :hover

之前这些 :hover 规则:
1. 没有 @media (hover: hover) 包装 → 触控设备点按后 :hover 卡住不消失
2. 没有 :focus-visible 对应 → 键盘用户看不到焦点反馈

location-picker-modal 的 :global(.map-cluster-bubble:hover) 涉及地图交互
改造,放后续批次。"
```

---

## Task 12: 最终全主题交叉验证 + 文档更新

**Files:**
- Modify: `frontend/AGENTS.md`(若需要记录新的 hover class 矩阵)
- Verify: 所有前述 Task 的累积效果

**Interfaces:**
- Consumes: 所有前 Task
- Produces: 确认 spec 所有目标达成;更新 AGENTS.md(若需要)

- [ ] **Step 1: 全量构建校验**

```bash
cd /Users/liyou/work/hotspot/frontend
bun check 2>&1 | tail -15
bun run build 2>&1 | tail -15
bun lint 2>&1 | tail -10
```

预期:`bun check` 0 errors,`bun run build` 成功,`bun lint` 0 violations。

- [ ] **Step 2: 全主题交叉视觉验证**

```bash
bun run dev --cwd frontend &
sleep 4
```

依次切换 6 个主题组合(forest/indigo/sand × light/dark),每个主题下访问:
- 首页(surface-hero 卡片、activity list、people-radar、nearby-users)
- 附近的人(user-card nearby-row)
- 动态(moment-card)
- 聊天(chat-list-panel、chat +page)
- 通知(notifications list-row)
- 个人中心(profile hub row)
- 设置(主题预览)

每个主题验证:
1. sand light:无方块色块,卡片有暖光陶瓷质感
2. sand dark:与 forest dark 第一眼可分辨(立体暖调 vs 扁平冷调),陶瓷凸起可见
3. forest:扁平无 neumorphic 阴影
4. indigo:neumorphic 现代感
5. 列表行 hover 全主题统一(背景 tint)
6. icon/pill 按钮 hover:neumorphic 主题浮起,forest 缩放
7. Tab 键聚焦都能看到 outline 环

发现任何问题回到对应 Task 修复。

杀掉 dev server。

- [ ] **Step 3: 更新 AGENTS.md(若有可记录的设计决策)**

读 `frontend/AGENTS.md`。如果有"hover 类型矩阵"等需要文档化的内容,在「UI/UX 设计规范」对应小节追加。

如果 spec 已足够,跳过此步。

- [ ] **Step 4: 总结 commit(若有 AGENTS.md 改动)**

```bash
git status
# 如果只有 AGENTS.md 改动:
git add frontend/AGENTS.md
git commit -m "文档(AGENTS): 记录 hover 按类型统一矩阵 + sand 主题色彩设计"
# 如果无改动:跳过
```

- [ ] **Step 5: 回顾 spec 目标达成**

读 `docs/specs/2026-07-19-hover-unification-and-sand-theme-fix-design.md` 的「目标」章节,逐项确认:

| 目标 | 达成 | 验证位置 |
|---|---|---|
| 1. 同主题内按类型统一 hover | ✓ | Task 5/7/10/11 |
| 2. 跨主题按主题族分策略 | ✓ | Task 5/6 |
| 3. 根治 sand light 方块 | ✓ | Task 1/8/9 |
| 4. 重塑 sand 陶瓷质感 | ✓ | Task 1/2 |
| 5. 不改变功能行为 | ✓ | 所有 Task 只动样式 |

---

## 风险与回滚

每个 Task 是独立 commit,失败可 `git revert <hash>` 单 Task 回滚。

关键风险:
- **Task 1/2 token 改动**:若 sand 陶瓷手感不满意,可在 0.4–0.6(light)/0.14–0.22(dark)范围微调
- **Task 5 hover class 重构**:旧 class 保留为别名,不破坏既有引用
- **Task 7 组件迁移**:12 处一次性迁移,若发现遗漏可单文件 patch
- **Task 9 skeleton 迁移**:只替换占位语义,误伤分割线的风险低

## 后续(非本轮)

按 spec「后续(非本轮)」章节,这些是下一轮工作:
- 废弃 `.row-hover` / `.surface-row` / `.list-row-hover` / `.circle-hover` 别名(等 1–2 发布周期)
- 把剩余组件 scoped `<style>` `:hover` 规则统一按矩阵审计
- **`.surface-flat` 在 40+ 个按钮上的 focus-visible 缺失**(最大 a11y 漏洞,审计 agent 2 发现)
- 统一散乱的 `active:scale-*` 数值(0.88/0.9/0.93/0.94/0.95/0.97/0.98)
- `:global(.map-cluster-bubble:hover)` 的地图交互改造
- location-picker-modal 的 24 处内联 box-shadow + 26 处 raw surface-bg(单独批次)
- **CTA `hover:opacity-*` → `hover:brightness-110`**(spec 批次 C 第 5 项):18+ 处散落 in settings/* + admin/*,低风险高扰动,放后续单独批次
- **剩余 ~240 处内联 `box-shadow: var(--shadow-sm/md/lg/xl)` → `.surface-*` class 迁移**:spec §4 B 显式列出的 `create-group-modal:237` 和 `search-modal:316, 451` 也在内。**重要**:Task 1 的 token 修复已经通过 cascade 自动修复了这些元素的用户可见"方块色块"问题(它们读的是同一组 token),所以这不是用户可见 bug,只是代码一致性 / 未来防回归的清理工作。单独批次处理。
- **Task 11 的 dead CSS**:`.radar-user:focus-visible` / `.compose__image-add:focus-visible` / `.compose__image-empty:focus-visible` 加在了非聚焦宿主元素(`<div aria-hidden>` / `<label>`)上,规则不触发。修法:把 `:focus-visible` 改为 `:focus-visible-within`,或重构 markup 让宿主可聚焦。

## 自审笔记(不修改计划,仅记录决策)

- **CTA `hover:opacity-*` 收敛**(spec 批次 C 第 5 项)移到后续:`hover:opacity-*` 在触控设备 hover 失效时元素回到 opacity 1,本身视觉安全,不属于 a11y 漏洞;改动 18+ 处会扰动 settings/admin 所有按钮,与"消除方块色块 / 重建陶瓷质感"的主目标关系不大
- **`.surface-flat` 的 box-shadow 缺失**(审计 agent 1 发现):Task 8 已修正策略,不再用 `.surface-flat` 替代有 box-shadow 的内联样式,改用对应的 `.surface-hero/raised/pressed/convex`
- **PREVIEW_ACCENT + glass-bg + popover 漂移**:这三个审计新发现的问题已纳入 Task 2(glass/popover/overlay-soft)和 Task 8(PREVIEW_ACCENT),与 token 改动联动
- **scoped hover 缺 focus-visible**(审计 agent 2 发现 8 处):Task 11 覆盖核心 8 处中的 7 处(map-cluster-bubble 因涉及地图改造放后续)
