# CLAUDE.md

> **注意**：本项目为 **主前端**（SvelteKit），当前所有前端开发在此进行，**同时是 UI/UX 与设计系统的唯一事实源**。设计自洽于本项目——颜色 / 间距 / 圆角 / 阴影 / 图标全部由 `src/app.css` 的语义 token 与本文件约定，**不再参照任何其它前端实现**。新页面 / 组件以**站内既有页面**为对齐基准：「发现」是版式（shell + 右栏 + sticky 头部）基准，「设置 / 个人主页」是两栏基准，「管理后台」是表格 / 仪表盘基准。

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

- **`.page-shell`**：内容栏容器，`margin-inline:auto` 居中 + 限定 `max-width`。页面的滚动内容**应**包在某一档 shell 内，不允许内容直接铺满 `desktop-main-card`。**例外**：附近的人 / 附近群组在桌面端无 shell 约束（`class:page-shell={!isDesktop}`），内容铺满全宽以最大化空间利用。
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
| 发现 / 附近动态 / 好友 / 关注 / 粉丝 / 我的瞬间·群组 | 默认 `.page-shell`（网格随宽度自适应列数） |
| 附近的人 / 附近群组（桌面端） | **无 shell**（`class:page-shell={!isDesktop}`，桌面端铺满全宽） |
| 附近的人 / 附近群组（移动/平板） | 默认 `.page-shell` |
| 瞬间详情 / 用户主页 / 群组详情 / 活动详情 / 分享页                         | `.page-shell--narrow`                      |
| 设置全系 / 个人资料 / 编辑资料 / 搜索 / 登录                               | `.page-shell--form`                        |
| 管理后台所有页（表格密集）                                                 | `.page-shell--wide`                        |

### 三档设备一致性

- **桌面（≥1280px）**：内容在 `desktop-main-card` 内再经 shell 收束 → 宽屏两侧自然留白，不拉伸成超长行。
- **平板（600–1279px）**：单列内容用 shell（多取默认档量级）+ `page-pad`，底部为浮动玻璃导航留出空间。
- **移动（<600px）**：shell 的 `100vw - 32px` 兜底保证左右各约 16px 留白；不追求居中收束（视口已足够窄），重点是不贴边。

#### 宽屏（`platformStore.isWide`）：平板也要充分利用横向空间

> 平板不是"拉大的手机"。视口 **宽度 ≥ 960px**（横屏 / 大尺寸平板与桌面）时 `platformStore.isWide` 为真，用它决定**内容布局**是否升级为多列 / master-detail；**导航 chrome**（侧边栏 vs 底栏）仍由 `isDesktop` / `isTablet` 决定。

- **内容密度优先用响应式宽度**，而非 `isDesktop` 硬分支：自适应网格（`.feed-density-grid` 的 `auto-fit` / Tailwind `columns-*`）天然在平板出多列。把"多列 vs 单列"的把关写成 `!platformStore.isMobile`（含平板）或交给 `auto-fit`，**不要**用 `isDesktop` 把平板挤回单列（典型：附近的人 / 附近群组）。
- **多列 / 两栏页面用 `isWide` 而非 `isDesktop`**：home（两栏）、settings（左导航 + 右内容两栏）、notifications（feed + 概览 aside）、profile（左资料 + 右画廊两栏）均把布局 gate 从 `isDesktop` 改为 `isWide`，宽屏平板复用桌面两栏；窄屏 / 竖屏小平板（<960）回退单列。
- **master-detail 用 `isWide` 而非 `isDesktop`**：如 chat 的 split-pane（左列表 + 右会话）在横屏 / 大平板同样开启。详情子页（`/chat/users/[id]` 等）据 `isWide` 决定是"贴合右栏（无返回）"还是"全屏（带返回）"。
- **底栏不能在 split-pane 里丢失**：宽屏平板进入 chat 会话详情时，`app-shell` 通过 `isWideChatPane` 让该页**不**算全屏子页，保留底部主导航（桌面靠常驻侧边栏，平板靠底栏）。
- **复用桌面分支时，导航 chrome 要按平板补位**（桌面靠侧边栏，平板没有）：
  - **定位入口**：若桌面把 `AnchorEntry` 放侧边栏，平板复用桌面分支前必须在页面内保留定位入口。home 复用桌面两栏时由可点击的 `HeroStats`（打开 `LocationPickerModal`）承担；各 `nearby` 页则在概览行保留 `AnchorEntry` pill。
  - **返回按钮**：sub-page（`/settings`、`/notifications` 等，平板隐藏底栏）复用桌面两栏时，桌面分支默认无返回（靠侧边栏）→ 必须**为平板补回返回**，否则用户被困。settings 在左导航栏顶部加 `BackButton`；notifications 的返回按钮保持 `!isDesktop`（平板可见）。顶层目的地（home / profile）有底栏，无需返回。

### 红线

- 禁止页面直接用裸 `max-w-3xl` / `max-w-2xl` / 内联 `max-width: 640px` 等各自为政的写法——统一走 `.page-shell` 档位，保证全站节奏一致。
- 禁止内容直接 `w-full` 平铺到容器边缘（既无 max-width 又无 `page-pad`）。
- 禁止 sticky 头部宽度与正文不一致（头部 `px-5` 平铺、正文 shell 收束）——头部内容必须包进同档 `.page-shell` + `.page-pad-x`（feed 页用 `.feed-density-shell` + `.feed-density-pad-x`）。
- 禁止用 `isDesktop` 把关"多列 vs 单列"内容密度而把平板挤回单列——改用 `!isMobile` / `isWide` / `auto-fit`（见「宽屏」小节）。
- 禁止给"落在品牌色（`--brand` / 选中态 / 自己的气泡 / hover 高亮）之上的文字、图标、描边"硬编码 `white` / `rgba(255,255,255,…)`——必须用 `--text-on-accent`（或 `color-mix(in srgb, var(--text-on-accent) N%, transparent)` 表达透明度）。原因：`--text-on-accent` 在 forest-dark / indigo-dark 等"亮色强调"主题下解析为**深色**，硬编码白色会在这些主题下变成"白字压亮底"低对比。例外：iOS 风格开关的白色滑块（puck）、地图标记上带 `text-shadow` 的白字属于刻意设计，保留。
- 改动布局后用 `bun check` + `bun run build` 校验。

## 设备导航模型（四档基线，已用 Chrome DevTools 实测）

> 断点取自 `platform.svelte.ts`：桌面 `≥1280`；平板 `min(w,h)≥600 且 <1280`；其余移动。`isWide = 宽度≥960`（决定**内容**是否双栏 / master-detail；**导航 chrome** 仍由 `isDesktop`/`isTablet` 决定）。导航集合的唯一事实源是 `src/lib/config/navigation.ts`。

| 档位   | 视口示例               | 导航 chrome                                                                       | 内容布局                                    |
| ------ | ---------------------- | --------------------------------------------------------------------------------- | ------------------------------------------- |
| 桌面   | ≥1280                  | 顶栏(52px) + 常驻侧边栏(192px：发现组 + 个人中心组 + 管理后台 + 底部 AnchorEntry) | `desktop-main-card` 内经 shell 收束；可双栏 |
| 宽平板 | 960–1279（含横屏平板） | 浮动玻璃药丸底栏 `TabletBottomNav`（4 tab + 中央 FAB）                            | **复用桌面双栏**；chat 走 split-pane        |
| 窄平板 | 600–959（竖屏）        | 同上浮动玻璃底栏                                                                  | **单列** shell + `page-pad`                 |
| 移动   | <600                   | 通栏底栏 `MobileBottomNav`（4 tab + 中央 FAB）                                    | 单列；左右各约 16px 兜底留白                |

- **底栏只放 4 个高频顶层目的地**：首页 / 发现 / 消息 / 我的 + 中央发布 FAB。其余顶层项（附近的人 / 附近群组 / 通知 / 我的瞬间 / 我的群组 / 关系 / 管理后台）在移动·平板经「我的」(个人中心) 或首页「查看全部」**下钻进入**，均为 sub-page（`sub-page.ts`）：全屏、自带返回、隐藏底栏。
- **返回按钮**：规则统一见下文「导航与页面框架规范 › 返回按钮」（顶层目的地桌面不显示、移动·平板保留；详情/钻取页三档都带返回）。
- **宽平板复用桌面分支时补齐导航 chrome**：sub-page 在左导航顶部补 `BackButton`（settings 已如此）；split-pane（chat）通过 `isWideChatPane` 保留底栏。
- **实测基线（符合预期，作为回归参照保持）**：四档底栏 / 侧边栏切换、sub-page 返回、chat split-pane 在宽平板保留底栏、宽平板双栏在窄平板回退单列——均正确。
- **已修正**：`sub-page.ts` 头部注释此前把「nearby 浏览页」描述为"保留底栏的顶层页"，与代码将 `/users/nearby`、`/groups/nearby` 放入 `SUB_PAGE_EXACT`（隐藏底栏 + 带返回）自相矛盾——已改写注释以代码行为为准。附近的人 / 群组在移动·窄平板此前仅靠首页「查看全部」入口、**数据为空时整段隐藏导致入口消失**：现在首页「附近的人 / 附近群组」**区块标题与「查看全部 / 更多」入口常驻**（不随数据条件隐藏），数据为空时仅区块正文降级为一行轻量提示；宽分支「附近的人」空态补 `查看全部` 跳转、群组卡片轮播空态也保留标题入口——四档下进入 `/users/nearby`、`/groups/nearby` 的入口均不依赖数据。

## 导航与页面框架规范（消除重复入口与冗余 chrome）

> 原则：**一个动作只保留一个入口**；页面框架（侧边栏 / 头部 / 底部导航）已提供的能力，主内容区不再重复。桌面以侧边栏为导航中枢，主内容区保持克制。

### 返回按钮（仅在「钻取页」且非桌面时出现）

- **顶层目的地**（侧边栏可直达：首页、附近的人 / 群组 / 动态、消息、通知、我的瞬间 / 群组、关系、管理后台）：
  - **桌面**：**不显示**返回——侧边栏即导航，返回是冗余 chrome。用 `{#if !platformStore.isDesktop}` 包裹返回按钮。
  - **移动 / 平板**：这些页可能从搜索等处进入且不在底部导航中，**保留**返回。
- **钻取页 / 详情页**（用户主页 `/users/[id]`、群组详情、瞬间详情、活动详情、设置子页、会话详情、编辑资料等）：三档设备都**保留**返回——它们不在框架导航中，是真正的下钻。

### 不重复入口

- **地图 / 浏览锚点**：位置与半径调整的唯一入口是**侧边栏底部的 `AnchorEntry`**（桌面）/ 头部或概览行的 `AnchorEntry` pill（移动·平板）。**禁止**在主内容区再放一个「地图」按钮重复打开 `LocationPickerModal`。
- 通用：发布、搜索、通知、设置等已在桌面顶栏 / 侧边栏存在的全局动作，不在页面内重复放置同义入口。

### 路由收敛（合并重复 / 同类页面，避免侧边栏与心智模型碎片化）

> 原则：**同一类内容只保留一个顶层路由**；近重复或强相关的子列表用页内 `filter-chip` 分页（tabs）承载，而非各占一个侧边栏项。删除旧路由时**保留 `+page.ts` 重定向**以兼容旧链接 / 书签。

- **关系**：`/relations` 是「好友 / 关注 / 粉丝」的唯一入口，三者为页内 tabs（`?tab=friends|following|followers`，URL 即单一事实源，支持深链与前进/后退）。三档列表**统一用 `UserCard variant="square"` + `usePagination`**，视觉与交互完全一致。旧路由 `/friends`、`/follows`、`/followers` 仅保留 `+page.ts` 的 `redirect(307, …)`。
- **首页**：`/dashboard` 与 `/` 内容重复，已 `redirect(307, '/')` 收敛到 `/`。
- 新增同类页面时优先评估能否并入既有顶层路由的 tab，而非新增侧边栏项。

### 标题

- 保留**简洁页面标题**（H1）作为页面身份与搜索/筛选控件的容器——与「发现」「好友」等全站一致；但标题行**只放标题与该页特有的真实动作**（如「创建群组」「发布」「刷新」「全部已读」），不放返回 / 地图等框架已提供的入口。

## 设计风格 / 视觉语言（全站统一的语义 token 设计系统）

> 设计系统由本项目自有：所有颜色、阴影、留白**只用语义 token**（定义在 `src/app.css`，随 `[data-theme-family]` 主题族解析），**禁止**硬编码十六进制色值。以站内既有页面为对齐基准，保持全站统一。

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
- **同一导航面内图标唯一表义**：一个图标不得在同一侧边栏 / 底栏里指代两个不同目的地。
- **图标默认单色**走 `--text-secondary` / `--text-primary`，**包括设置项**——不再用 per-item 彩色图标（已收敛）。一个图标语义不要被过度复用稀释（如 `Shield`）。
- **图标实现一律用 lucide 组件**，禁止再手写内联 `<svg>` 复刻 lucide 已有图标（同一图标两套实现 = 维护与风格双重隐患）。

## 一致性审查清单（多设备 UI 实测归纳，硬约束）

> 用 Chrome DevTools 跨「移动 / 窄平板 / 宽平板 / 桌面」四档巡检所有顶层页 + 代表详情页归纳。下列与上文各「红线」并列生效。本轮审查发现的偏差**均已收敛**，标注 **已修正 / 已解决** 的条目记录了正确范式与落地方案，应作为后续开发的参照基线、避免回归。

### 图标唯一性

- **已修正**：`附近群组`（discoverNav）保留 `UsersRound`、`我的群组`（personalNav）改用 `Users` 区分——**同一侧边栏内同列图标必须各不相同**，禁止仅靠文字 / 分组区分。
- **已修正**：设置左导航「通知」保留 `Bell`（跳 `/settings/notifications` 子页），「通知偏好」（页内开关组）改用 `SlidersHorizontal`——图标 + 语义双重消歧；同名近义入口须用不同图标。
- **已修正**：发布「+」桌面顶栏改用 `lucide Plus`（原内联 `<svg>`）；app-shell 登录提示改用 `lucide Lock`（原内联锁 `<svg>`）——**装饰性图标一律走 `lucide-svelte`，禁止内联 `<svg>` path**（底栏 FAB 已用 `Plus`，全站统一）。
- **已修正**：设置项图标由 per-item 多彩（brand/warning/success/tertiary）统一为单色 `--text-secondary`；并按语义拆分被稀释的 `Shield`——会话管理→`MonitorSmartphone`（管理登录设备）、隐私设置→`EyeOff`（可见性）、我的举报记录→`Flag`（与管理后台「举报审核」一致）；`Shield` 在设置内仅保留「隐私」分区标题。**线性图标默认单色 + 同语义专图标**。
- **已修正**：chat 空状态（选择会话 / 没有找到会话·群聊）由语义偏弱的 `MessageSquareDashed` 改为 `MessageCircle`，与「消息」导航图标统一。

### 一个动作一个入口（承「### 不重复入口」）

- **一页同一动作只保留一个入口**。**已修正**：`我的群组` 此前同屏 3 个「刷新」，已收敛为头部单一 `refreshAll()`（一并刷新 已加入群组 / 收到的邀请 / 我的申请），两张卡内的独立刷新按钮已移除。
- **已修正**：通知此前桌面双入口（顶栏 `Bell` + 侧边栏「通知」）、双角标（顶栏头像 + 侧边栏项）。已**移除顶栏 Bell** 与**顶栏头像上语义错位的未读角标**，桌面通知统一由侧边栏「通知」单一入口 + 单一角标承载（与「消息」角标同款）。顶栏只保留不与侧边栏导航目的地重复的工具动作（搜索 / 主题 / 配色 / 发布 / 设置 / 头像）。

### 派生展示字段单一事实源

- **凡从原始数据"算出来"的展示值（`@handle`、距离、相对时间、计数缩写等）必须走唯一共享 helper**，禁止各页内联各算各的。**已修正**：用户 `@handle` 统一走 `formatHandle({ nickname, user_id })`（`$lib/utils/format`）——优先昵称小写、连续空白折叠为点号，昵称为空才回退 `user_id` 前 8 位；profile（×3）/ profile/edit / users/[id] / settings（×2）已全部改用该 helper，同一用户跨页显示一致（如 `@ui.audit`）。新增「@handle / 距离 / 时间」类展示一律复用此约定。

### 版式密度与空间利用（承「### 三档设备一致性」）

- **宽档（桌面 / 宽平板）横跨主栏的"概览卡"必须利用横向空间**：内容不得全堆最左、右侧大片留白。**已修正**：附近的人 / 附近群组「附近概览」卡（桌面）左侧为图标 + 标题/说明，右侧用共享 **`StatSegments`**（`$lib/components/common/stat-segments.svelte`）横向分段统计填充——附近的人 = `在线 / 总数 / 半径`，附近群组 = `群组 / 半径`。`StatSegments` 同时是 `hero-stats` 三栏统计的底层实现（`layout="grid"`），**全站"数值 + 标签"统计条只此一个组件**，禁止再内联手搭。
- **同页只用一套网格机制，但按内容高度选型**：**已修正**：`我的群组`已加入群组列表由 `columns-*` 瀑布流改为 `grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3`，与下方"邀请 / 申请"的 `grid lg:grid-cols-2` 同为 CSS Grid、行对齐。**选型规则**：① 卡片**等高**（群组卡：头像 + 名称 + 成员数）→ 用 CSS **Grid**（行对齐、列规整）；② 卡片**不等高**（我的瞬间：文本 + 可选图墙）→ 保留 `columns-*` **瀑布流**（Pinterest 式，避免行内大留白）。两者均 `gap-4`，同页不得混用 `auto-fit` 卡网格 + 50/50 两栏两套机制。
- **空状态自适应、垂直居中收束**：避免固定高度的空状态小卡在大屏下方留大片空白。**已修正**：`EmptyState` 新增 `fill` 模式（外层 `min-height: clamp(360px, 56vh, 620px)` + 垂直居中）。**整页空态**（附近的人 / 附近群组 / 我的瞬间——其下无并列内容）一律传 `fill`，让空态在 shell / 主列内垂直居中；**下方仍有并列内容的空态**（如 `我的群组`空列表下方还有邀请/申请区）**不传** `fill`，避免把后续区块推到屏外。首页右栏空态用 `flex flex-1 items-center justify-center` 自行居中（同款效果）。
- **顶层 feed 页右栏（aside）策略按页面性质区分（成文约定）**：**已解决**——并非所有 feed 页都该有右栏，而是**有真实补充模块才挂右栏**：① **发现 / 通知**＝"浏览发现"型，桌面挂 `220px` `border-left` 右栏承载补充模块（热门话题·活跃用户 / 概览·类型分布），用「全宽顶栏 + 主列/右栏分屏」范式；② **附近的人 / 附近群组**＝"筛选结果列表"型，无独立补充模块，**桌面端去掉 shell 约束铺满全宽**（`class:page-shell={!isDesktop}`），最大化空间利用 + `feed-density-grid` 自适应列数；移动/平板端仍用居中 `.page-shell` 单列 + 全宽统计头（`StatSegments`）。新增 feed 页据此二分归类，勿再两可。

### 标题与交互节奏

- **H1 副标题策略统一**：**门面页（home）/ 管理·配置页（settings·admin）** 可带一行说明性副标题；**纯内容列表页一律不带**。**已修正**：`我的瞬间` 去掉副标题"记录和管理…"，与 发现 / 附近的人 / 附近群组 / 通知 / 关系 统一为单行 H1。
- **搜索交互全站即时**（输入即搜，防抖）。**已修正**：管理后台「用户管理」由独立「搜索」按钮改为 `oninput` 防抖即时搜索（保留 Enter 立即触发），与 发现 / 附近 一致；全站不再有按钮式精确检索的例外。
- **profile / 设置页内的快捷入口在桌面与侧边栏重复**。**已修正**：profile 左栏个人中心快捷入口（我的瞬间 / 我的群组 / 关系 / 通知 / 设置 / 管理后台）在**桌面（`isDesktop`）隐藏**（侧边栏已直达）；移动 / 宽平板（无侧边栏）仍保留。其价值仅在无侧边栏场景。
