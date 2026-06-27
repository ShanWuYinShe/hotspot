---
paths:
  - 'src/**/*.svelte'
---

# UI 布局规范

## 核心原则

> **内容居中收束在一条合理的测量栏内，宽屏不铺满全宽、不留大片空白；同时始终保留舒适的响应式留白，内容不紧贴容器边缘。**

## 必用原语（定义在 `src/app.css`）

- **`.page-shell`**：内容栏容器，`margin-inline:auto` 居中 + 限定 `max-width`。页面的滚动内容**应**包在某一档 shell 内，不允许内容直接铺满 `desktop-main-card`。**例外**：附近的人 / 附近群组在桌面端无 shell 约束（`class:page-shell={!isDesktop}`），内容铺满全宽以最大化空间利用。
  - 默认（feed / 卡片列表 / 网格）：`min(1180px, 100vw - 32px)`
  - `--wide`（数据表格、管理后台、仪表盘）：`min(1320px, …)`
  - `--narrow`（详情、阅读、长文）：`min(720px, …)`
  - `--form`（表单、设置、个人资料）：`min(560px, …)`
- **`.page-pad`**：响应式留白（`clamp` 横向随视口缩放，底部更宽容纳「加载更多」/安全区）。`--tight` 为紧凑变体。
- 现有 `.feed-density-shell` / `.feed-density-pad`（discover、moments/nearby）是同一原则的 feed 专用实现，等价保留，**新页面优先用 `.page-shell` + `.page-pad`**。

## 用法

```svelte
<div class="flex-1 overflow-y-auto no-scrollbar page-pad">
  <div class="page-shell page-shell--narrow">…页面内容…</div>
</div>
```

带右栏的页面（如「发现」）：主栏用 shell 收束，右栏固定宽度、`border-left` 分隔。

## Sticky 头部对齐

sticky 容器负责**全宽出血**（背景色 + `border-bottom` 通栏），内层用与正文**同档** `.page-shell` 收束内容，横向留白用 **`.page-pad-x`**：

```svelte
<div
  class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
  style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
>
  <div class="page-shell">…标题 / 搜索 / 筛选…</div>
</div>
<div class="page-pad"><div class="page-shell">…列表 / 网格…</div></div>
```

- 头部与正文 `.page-shell` 档位必须一致。
- 分隔线/背景全宽通栏，内容收束。
- **禁止**头部 `px-5` 正文 `page-shell` 的宽度不一致写法。
- feed 变体用 `.feed-density-shell` + `.feed-density-pad-x`，规则等价。
- 例外：桌面 feed 页「全宽顶栏 + 主列/右栏分屏」不要求对齐；app-bar 式头部横跨全宽。

## 档位选择

| 页面类型                                             | 档位                                            |
| ---------------------------------------------------- | ----------------------------------------------- |
| 发现 / 附近动态 / 好友 / 关注 / 粉丝 / 我的瞬间·群组 | 默认 `.page-shell`                              |
| 附近的人 / 附近群组（桌面端）                        | **无 shell**（`class:page-shell={!isDesktop}`） |
| 附近的人 / 附近群组（移动/平板）                     | 默认 `.page-shell`                              |
| 瞬间详情 / 用户主页 / 群组详情 / 活动详情 / 分享页   | `.page-shell--narrow`                           |
| 设置全系 / 个人资料 / 编辑资料 / 搜索 / 登录         | `.page-shell--form`                             |
| 管理后台所有页（表格密集）                           | `.page-shell--wide`                             |

## 三档设备一致性

- **桌面（≥1280px）**：`desktop-main-card` 内经 shell 收束。
- **平板（600–1279px）**：单列 shell + `page-pad`，底部为浮动玻璃导航留空间。
- **移动（<600px）**：shell `100vw - 32px` 兜底左右 16px 留白。

**宽屏（`platformStore.isWide`，宽度 ≥960px）**：平板也要充分利用横向空间。内容密度优先用响应式宽度（`auto-fit` / `columns-*`），不用 `isDesktop` 把关多列。多列/两栏页面用 `isWide` 而非 `isDesktop`。master-detail 用 `isWide`。底栏不在 split-pane 里丢失。宽平板复用桌面分支时补齐导航 chrome。

## 容器查询原语（component-driven responsiveness）

组件按**自身容器宽度**自适应，零视口判断、零 `platformStore` 引用。设计参照 GitHub Primer「两层抽象」+ Josh Comeau 混合模式：视口层只管导航 chrome（见 `app-shell.svelte`），组件层全部走本原语。

### 决策树：用哪个原语？

| 场景 | 原语 | 说明 |
| --- | --- | --- |
| 卡片网格（列表/feed） | `.density-grid` | 容器窄单列，够宽按卡片最小宽自动铺列。替代 `!isMobile ? grid : flex`。`--compact` 更密，`--list` 强制单列 |
| 主栏 + 固定宽侧栏（settings/profile/chat） | `.sidebar-layout` | 容器 ≥720px 并排，否则堆叠。第一子元素是侧栏，第二是主内容 |
| N 项水平排列，不够就堆叠 | `.switcher` | 纯 flex+wrap，无需 `@container`。`--switcher-threshold` 控制换行阈值 |
| 一组内联元素换行（标签/工具栏） | `.cluster` | 子项按内容尺寸不强制等分 |
| 返回按钮宽屏隐藏 | `.back-on-wide` | 配合 `.sidebar-layout`：宽容器 `display:none` |

### 用法

```svelte
<!-- 卡片网格：自动列数，无需任何 JS 判断 -->
<div class="density-grid">
  {#each items as item (item.id)}<Card {item} />{/each}
</div>

<!-- 主栏 + 侧栏：单一 DOM，容器查询切列 -->
<div class="sidebar-layout" style="--sidebar-width: 260px;">
  <aside><!-- 侧栏内容 --></aside>
  <main><!-- 主内容（自动 overflow-y-auto）--></main>
</div>
```

### 布局 token（app.css `:root`）

测量栏宽度：`--measure-default`(1180) / `--measure-wide`(1320) / `--measure-narrow`(720) / `--measure-form`(560) / `--measure-medium`(960)。容器断点：`--cq-narrow`(520) / `--cq-sidebar`(720) / `--cq-compact`(360)。**禁止再硬编码这些数值**。

## platformStore 语义化派生量

业务页需要视口级判断时，**优先用语义化派生量**（表达「布局意图」而非「设备类型」），而非直接读 `isMobile/isTablet/isDesktop/isWide`：

| 派生量 | 含义 | 典型场景 |
| --- | --- | --- |
| `useMasterDetail` | 宽屏采用 master-detail（左导航+右内容 / split-pane） | settings/profile/chat 的双栏 vs 长滚动 |
| `useSidebarChrome` | 桌面常驻侧栏档（全宽内容区、rightSidebar） | nearby 全宽出血、discover 桌面双栏 |
| `hideRedundantEntry` | 桌面侧栏已直达，页面内隐藏冗余入口（返回/创建按钮） | nearby/profile/notifications 的 BackButton |

直接读 `isWide/isDesktop` 仅限：① `app-shell` 导航 chrome 装配 ② 上述派生量无法覆盖的特定视口行为。**纯布局密度差异必须走容器查询原语，不得用 JS 判断。**

## 红线

- 禁止用 `max-w-3xl` / 内联 `max-width` 等各自为政——统一走 `.page-shell` 档位。
- 禁止内容 `w-full` 平铺到容器边缘（无 max-width 又无 `page-pad`）。
- 禁止 sticky 头部宽度与正文不一致。
- **禁止业务页用 `platformStore.isMobile/isTablet/isDesktop/isWide` 切换纯布局密度**（多列/单列、grid/flex）——用 `.density-grid` 等容器查询原语。
- 视口级判断（master-detail vs 长滚动、chrome 装配、入口去重）用语义化派生量（`useMasterDetail` / `useSidebarChrome` / `hideRedundantEntry`），禁止裸用 `isWide`。
- 禁止硬编码测量栏宽度（1180/720/560…）与断点（600/960/1280…）——走布局 token。
- 禁止在品牌色之上硬编码白字——用 `--text-on-accent`。例外：iOS 开关滑块、地图标记。
- 改动布局后 `bun check` + `bun run build` 校验。
