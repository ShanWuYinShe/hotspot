# CLAUDE.md

本文件只定义本子项目的技术约束与本地命令；仓库级 workflow 以根 `CLAUDE.md` 为准。

> **注意**：本项目为 **主前端**（SvelteKit），当前所有前端开发在此进行。设计需与 `frontend_react/`（React，Figma Make 生成）保持 UI/UX 一致——React 版作为设计参考保留，不再退役本项目。

## 构建 & 测试

- `bun run build` — 构建
- `bun check` — 类型检查（svelte-check）
- `bun lint` — 格式检查（prettier --check）
- `bun run format` — 格式化（prettier --write）

> **测试**：package.json 无 `test` 脚本，暂无自动化测试。

## 约束

- **API 路径**：统一从 `endpoints.ts` 引用。
- **请求封装**：所有请求走 `client.ts` 和 `services/*`。
- **HTTP 方法**：统一 POST。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **分页**：cursor 只透传，不解析、不构造。

## 页面布局规范（所有页面、所有设备的硬约束）

> 核心原则参考「发现」页对主内容区的处理：**内容居中收束在一条合理的测量栏内，宽屏不铺满全宽、不留大片空白；同时始终保留舒适的响应式留白，内容不紧贴容器边缘。** 借鉴 Linear / Notion / Stripe / Vercel 等优秀产品的版式：长内容有最佳阅读宽度，留白随视口平滑缩放，节奏统一——既克制又有设计感。

### 必用原语（定义在 `src/app.css`）

- **`.page-shell`**：内容栏容器，`margin-inline:auto` 居中 + 限定 `max-width`。任何页面的滚动内容**必须**包在某一档 shell 内，不允许内容直接铺满 `desktop-main-card`。
  - 默认（feed / 卡片列表 / 网格）：`min(1180px, 100vw - 32px)`
  - `--wide`（数据表格、管理后台、仪表盘）：`min(1320px, …)`
  - `--narrow`（详情、阅读、长文）：`min(720px, …)`
  - `--form`（表单、设置、个人资料）：`min(560px, …)`
- **`.page-pad`**：响应式留白（`clamp` 横向随视口缩放，底部更宽容纳「加载更多」/安全区）。`--tight` 为紧凑变体。
- 现有 `.feed-density-shell` / `.feed-density-pad`（discover、moments/nearby）是同一原则的 feed 专用实现，等价保留，**新页面优先用 `.page-shell` + `.page-pad`**。

### 用法

```svelte
<!-- 滚动区负责留白与居中，内容栏负责收束宽度 -->
<div class="flex-1 overflow-y-auto no-scrollbar page-pad">
  <div class="page-shell page-shell--narrow">…页面内容…</div>
</div>
```

带右栏的页面（如「发现」）：主栏用 shell 收束，右栏固定宽度、`border-left` 分隔，不与主栏争抢空白。

### Sticky 头部对齐（标题/搜索/筛选与正文必须共用同一测量栏）

> 症状：sticky 头部用裸 `px-5`/`px-6` 平铺全宽，正文却用 `.page-shell` 居中收束 → 头部内容比正文宽、左右边缘错位。**头部内容与正文必须左对齐、右对齐、同宽。**

规范结构：sticky 容器负责**全宽出血**（背景色 + `border-bottom` 通栏），内层用与正文**同档** `.page-shell` 收束内容，横向留白用 **`.page-pad-x`**（与 `.page-pad` 的横向 `clamp` 完全一致）：

```svelte
<!-- 全宽出血的 sticky 头部 -->
<div
  class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
  style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
>
  <div class="page-shell">…标题 / 搜索 / 筛选…</div>
</div>
<!-- 正文：同档 shell + page-pad -->
<div class="page-pad"><div class="page-shell">…列表 / 网格…</div></div>
```

- 头部与正文的 `.page-shell` 档位必须一致（同为默认档或同为 `--narrow` 等）。
- 分隔线/背景全宽通栏，内容收束——参考「发现」桌面头部的处理。
- **禁止**头部用 `px-5` 而正文用 `.page-shell` 这种宽度不一致的写法。
- **feed 专用变体**：discover、moments/nearby 用的是 `.feed-density-shell[--narrow|--medium]` + `.feed-density-pad`；其 sticky 头部对应用 **`.feed-density-pad-x`**（横向 `clamp` 与 `.feed-density-pad` 一致）+ 内层同档 `.feed-density-shell`，规则与上面完全等价。
- **例外（无需收束对齐）**：① 桌面端 feed 页（discover / moments/nearby 的 `isDesktop` 分支）采用「全宽顶栏 + 主列/右栏分屏」范式，顶栏本就横跨整个面板，不要求与居中主列对齐；② 设置 / 资料 / 编辑资料等 app-bar 式头部（返回在左、操作在右、`justify-between` 横跨全宽）按惯例保持全宽，不要把返回按钮收进 `--form` 档居中。

### 档位选择

| 页面类型                                                                   | 档位                                       |
| -------------------------------------------------------------------------- | ------------------------------------------ |
| 发现 / 附近动态 / 附近的人 / 附近群组 / 好友 / 关注 / 粉丝 / 我的瞬间·群组 | 默认 `.page-shell`（网格随宽度自适应列数） |
| 瞬间详情 / 用户主页 / 群组详情 / 活动详情 / 分享页                         | `.page-shell--narrow`                      |
| 设置全系 / 个人资料 / 编辑资料 / 搜索 / 登录                               | `.page-shell--form`                        |
| 管理后台所有页（表格密集）                                                 | `.page-shell--wide`                        |

### 三档设备一致性

- **桌面（≥1280px）**：内容在 `desktop-main-card` 内再经 shell 收束 → 宽屏两侧自然留白，不拉伸成超长行。
- **平板（600–1279px）**：单列内容用 shell（多取默认档量级）+ `page-pad`，底部为浮动玻璃导航留出空间。
- **移动（<600px）**：shell 的 `100vw - 32px` 兜底保证左右各约 16px 留白；不追求居中收束（视口已足够窄），重点是不贴边。

### 红线

- 禁止页面直接用裸 `max-w-3xl` / `max-w-2xl` / 内联 `max-width: 640px` 等各自为政的写法——统一走 `.page-shell` 档位，保证全站节奏一致。
- 禁止内容直接 `w-full` 平铺到容器边缘（既无 max-width 又无 `page-pad`）。
- 禁止 sticky 头部宽度与正文不一致（头部 `px-5` 平铺、正文 shell 收束）——头部内容必须包进同档 `.page-shell` + `.page-pad-x`（feed 页用 `.feed-density-shell` + `.feed-density-pad-x`）。
- 改动布局后用 `bun check` + `bun run build` 校验。

## 导航与页面框架规范（消除重复入口与冗余 chrome）

> 原则：**一个动作只保留一个入口**；页面框架（侧边栏 / 头部 / 底部导航）已提供的能力，主内容区不再重复。桌面以侧边栏为导航中枢，主内容区保持克制。

### 返回按钮（仅在「钻取页」且非桌面时出现）

- **顶层目的地**（侧边栏可直达：首页、附近的人 / 群组 / 动态、消息、通知、我的瞬间 / 群组、好友、关注、粉丝、管理后台）：
  - **桌面**：**不显示**返回——侧边栏即导航，返回是冗余 chrome。用 `{#if !platformStore.isDesktop}` 包裹返回按钮。
  - **移动 / 平板**：这些页可能从搜索等处进入且不在底部导航中，**保留**返回。
- **钻取页 / 详情页**（用户主页 `/users/[id]`、群组详情、瞬间详情、活动详情、设置子页、会话详情、编辑资料等）：三档设备都**保留**返回——它们不在框架导航中，是真正的下钻。

### 不重复入口

- **地图 / 浏览锚点**：位置与半径调整的唯一入口是**侧边栏底部的 `AnchorEntry`**（桌面）/ 头部或概览行的 `AnchorEntry` pill（移动·平板）。**禁止**在主内容区再放一个「地图」按钮重复打开 `LocationPickerModal`。
- 通用：发布、搜索、通知、设置等已在桌面顶栏 / 侧边栏存在的全局动作，不在页面内重复放置同义入口。

### 标题

- 保留**简洁页面标题**（H1）作为页面身份与搜索/筛选控件的容器——与「发现」「好友」等全站一致；但标题行**只放标题与该页特有的真实动作**（如「创建群组」「发布」「刷新」「全部已读」），不放返回 / 地图等框架已提供的入口。

## 设计风格 / 视觉语言（Figma Make 设计系统，全站统一）

> 与 `frontend_react/` 的 Figma 设计保持一致。所有颜色、阴影、留白**只用语义 token**（定义在 `src/app.css`，随 `[data-theme-family]` 主题族解析），**禁止**硬编码十六进制色值。

### 颜色 / 表面 token（节选，统一引用）

- 画布与表面：`--page-bg`（最外层画布）、`--surface-bg`（卡片 / 头部 / 输入面）、`--border-theme`（分隔线 / 描边）。
- 文字：`--text-primary`（标题 / 正文）、`--text-secondary`（次要 / 说明 / 占位）、`--text-on-accent`（落在品牌色上的文字）。
- 品牌与强调：`--brand`（主操作 / 选中态 / FAB）、`--glow`（品牌发光阴影）、`--theme-accent-soft` / `--theme-accent-soft-border`（弱化的品牌底）。
- 语义：`--color-semantic-online`（在线）、`--semantic-hot`（热门）、`--semantic-warning`（提醒）；地图：`--map-user` / `--map-group` / `--map-post`。

### 表面与质感

- 主题族目前为 **forest（扁平）**：阴影归零，组件呈干净的扁平形状；其它主题族可有 neumorphic 阴影（`--shadow-light` / `--shadow-dark` / `--shadow-md` / `--shadow-lg`）。
- 复用类：`.surface-flat`（扁平交互面，如图标按钮 / chip）、`.surface-hero` / `.ambient-glow`（概览 / 主张卡）、`.input-surface`（输入容器）。圆形图标按钮统一 `36px`（移动可 `38px`）。

### 圆角

- 基础 `--radius: 0.625rem`。常用：卡片 `22px`、概览/主张卡 `24px`、输入/搜索 `18px`、小面 `16px`、chip / pill / 头像 `999px`（全圆）。

### 字号 / 字重

- 页面标题 H1 `18–24px`（列表/钻取页约 `18–20px`，发现等门面页 `22–24px`），`font-weight 600`。
- 正文 `13–14px`；次要/说明 `11–13px`；eyebrow / 分组标签 `10px` + `letter-spacing ~0.12em` + 大写。
- 颜色一律走 `--text-primary` / `--text-secondary`。

### 间距 / 节奏

- 横向留白随视口 `clamp`（见 `.page-pad` / `.page-pad-x`）；移动兜底左右各约 `16px`。
- 头部内元素纵向间距 `mb-2 ~ mb-3`；网格/列表 gap 走 `.feed-density-grid` / `gap-2.5 ~ gap-5`。
- 桌面侧边栏固定 `192px`，顶栏 `52px`，发现等右栏 `220px` + `border-left`。

### 动效

- 点击反馈统一缩放：按钮 `active:scale-[0.88~0.97]`，`transition-transform duration-150`。
- 主题切换有 `transition-colors duration-500`。克制为主，不堆砌动画。

### 图标

- 统一 `lucide-svelte`，按需单文件引入（`lucide-svelte/icons/<name>`）。线宽：默认 `1.5`，选中/强调 `2`；尺寸随容器（图标按钮内约 `15–17px`）。
