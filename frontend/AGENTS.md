# HotSpot Frontend (SvelteKit)

> **注意**：本项目为 **主前端**（SvelteKit），当前所有前端开发在此进行，**同时是 UI/UX 与设计系统的唯一事实源**。改前端样式前先阅读本文档底部的「UI/UX 设计规范」三节。

## 功能边界

SvelteKit **主前端**,UI/UX 与设计系统唯一事实源。覆盖认证/发现与搜索/信息流/生活瞬间与评论/媒体上传/聊天(私聊+群聊)/群组/通知与公告/举报/分享/地图/个人中心/管理后台。支付与交易、端侧离线同步不在范围内。

API 契约见 `spec/api_spec_*.md`（仓库根 `spec/` 目录，只读权威）。

## 构建 & 测试

- `bun run build` — 构建
- `bun check` — 类型检查（svelte-check）
- `bun lint` — 格式检查（prettier --check）
- `bun run format` — 格式化（prettier --write）

> **测试**：package.json 无 `test` 脚本，暂无自动化测试。

## 技术栈

- SvelteKit（Svelte 5）
- TypeScript（严格模式）
- Vite

## 目录与分层（必须）

- `src/routes/`：页面与路由（`+page.svelte` / `+page.ts` / `+layout.svelte`）
- `src/routes/(public)/(protected)/(admin)/(auth)`：按访问级别分组
- `src/lib/api/`：HTTP 客户端与服务层
  - `client.ts`：统一请求封装
  - `endpoints.ts`：API 路径常量
  - `services/*.ts`：业务 API 调用
- `src/lib/stores/`：全局状态（Svelte stores）
- `src/lib/components/`：可复用组件
- `src/lib/hooks/`：可复用逻辑封装
- `src/lib/attachments/`：Svelte actions

## API 调用规范

- **禁止硬编码路径**：必须通过 `src/lib/api/endpoints.ts`
- **统一请求封装**：所有请求走 `client.ts` 和 `services/*`
- **HTTP 方法**：统一 POST（RPC + JSON，不实现 REST 风格）
- **成功判定**：仅使用 `ApiResponse.code === 0`
- **HTTP 层错误**：通过 try/catch 处理网络/超时/非 200（见通用契约）
- **仅在 HTTP 200 时解析**：非 200 直接视为请求失败，不解析 `ApiResponse`
- **Cursor 不透明**：请求时透传 `cursor`，收到响应后将 `next_cursor` 作为下一页的 `cursor` 透传，不解析、不构造

## Token 与安全

- Web 端仅使用 HttpOnly Cookie，禁止在 localStorage/sessionStorage 存储 Refresh Token
- App 端使用 `Authorization: Bearer`（如有跨端场景）
- 前端不解析 JWT，只按业务接口返回值处理登录态

## WebSocket 规范

- 事件名与 payload 结构以 `spec/api_spec_websocket.md` 为准
- 可在前端定义**镜像类型**（仅映射契约，不得扩展）
- 消息最终态以 WebSocket 推送为准

## 位置与距离

- 位置请求使用 `Location`，响应使用 `RelativeLocation`
- 禁止前端自行计算距离/角度（由后端 PostGIS 完成）

## 数据格式（契约级）

- 所有时间字段为 UTC ISO 8601，前端负责时区转换
- 所有 `xxx_id` 字段按 `string` 处理，不做数值运算

## TypeScript 约束

- 禁止 `any`
- 访问可选字段必须使用 `?.`/`??`

## UI/UX 设计规范

以下三节为内联设计规范（原 `.claude/rules/` 已并入本文件），改 UI 前必读。

**响应式适配原则**：组件层用容器查询原语（`.density-grid` 等，组件按自身容器宽度自适应），视口层判断用语义化派生量（`useMasterDetail`/`useSidebarChrome`/`hideRedundantEntry`）。业务页禁止裸用 `isMobile/isTablet/isDesktop/isWide` 切换纯布局密度。

### UI 布局规范

**核心原则**：内容居中收束在一条合理的测量栏内，宽屏不铺满全宽、不留大片空白；同时始终保留舒适的响应式留白，内容不紧贴容器边缘。

**必用原语（定义在 `src/app.css`）**：
- **`.page-shell`**：内容栏容器，`margin-inline:auto` 居中 + 限定 `max-width`。页面的滚动内容**应**包在某一档 shell 内，不允许内容直接铺满 `desktop-main-card`。**例外**：附近的人 / 附近群组在桌面端无 shell 约束（`class:page-shell={!isDesktop}`），内容铺满全宽以最大化空间利用。
  - 默认（feed / 卡片列表 / 网格）：`min(1180px, 100vw - 32px)`
  - `--wide`（数据表格、管理后台、仪表盘）：`min(1320px, …)`
  - `--narrow`（详情、阅读、长文）：`min(720px, …)`
  - `--form`（表单、设置、个人资料）：`min(560px, …)`
- **`.page-pad`**：响应式留白（`clamp` 横向随视口缩放，底部更宽容纳「加载更多」/安全区）。`--tight` 为紧凑变体。
- 现有 `.feed-density-shell` / `.feed-density-pad`（discover、moments/nearby）是同一原则的 feed 专用实现，等价保留，**新页面优先用 `.page-shell` + `.page-pad`**。

**用法**：
```svelte
<div class="flex-1 overflow-y-auto no-scrollbar page-pad">
  <div class="page-shell page-shell--narrow">…页面内容…</div>
</div>
```
带右栏的页面（如「发现」）：主栏用 shell 收束，右栏固定宽度、`border-left` 分隔。

**Sticky 头部对齐**：sticky 容器负责**全宽出血**（背景色 + `border-bottom` 通栏），内层用与正文**同档** `.page-shell` 收束内容，横向留白用 **`.page-pad-x`**：
```svelte
<div class="sticky top-0 z-10 page-pad-x pt-3 pb-3" style="background: var(--surface-bg); border-bottom: 1px solid var(--border-theme);">
  <div class="page-shell">…标题 / 搜索 / 筛选…</div>
</div>
<div class="page-pad"><div class="page-shell">…列表 / 网格…</div></div>
```
- 头部与正文 `.page-shell` 档位必须一致。
- 分隔线/背景全宽通栏，内容收束。
- **禁止**头部 `px-5` 正文 `page-shell` 的宽度不一致写法。

**档位选择**：

| 页面类型 | 档位 |
| --- | --- |
| 发现 / 附近动态 / 好友 / 关注 / 粉丝 / 我的瞬间·群组 | 默认 `.page-shell` |
| 附近的人 / 附近群组（桌面端） | **无 shell**（`class:page-shell={!isDesktop}`） |
| 附近的人 / 附近群组（移动/平板） | 默认 `.page-shell` |
| 瞬间详情 / 用户主页 / 群组详情 / 活动详情 / 分享页 | `.page-shell--narrow` |
| 设置全系 / 个人资料 / 编辑资料 / 搜索 / 登录 | `.page-shell--form` |
| 管理后台所有页（表格密集） | `.page-shell--wide` |

**三档设备一致性**：
- 桌面（≥1280px）：`desktop-main-card` 内经 shell 收束。
- 平板（600–1279px）：单列 shell + `page-pad`，底部为浮动玻璃导航留空间。
- 移动（<600px）：shell `100vw - 32px` 兜底左右 16px 留白。

**宽屏（`platformStore.isWide`，宽度 ≥960px）**：平板也要充分利用横向空间。内容密度优先用响应式宽度（`auto-fit` / `columns-*`），不用 `isDesktop` 把关多列。多列/两栏页面用 `isWide` 而非 `isDesktop`。master-detail 用 `isWide`。底栏不在 split-pane 里丢失。宽平板复用桌面分支时补齐导航 chrome。

**容器查询原语（component-driven responsiveness）**：组件按**自身容器宽度**自适应，零视口判断、零 `platformStore` 引用。
- 卡片网格（列表/feed）：`.density-grid`（容器窄单列，够宽按卡片最小宽自动铺列；`--compact` 更密，`--list` 强制单列）。
- 主栏 + 固定宽侧栏（settings/profile/chat）：`.sidebar-layout`（容器 ≥720px 并排，否则堆叠）。
- N 项水平排列，不够就堆叠：`.switcher`（纯 flex+wrap；`--switcher-threshold` 控制换行阈值）。
- 一组内联元素换行（标签/工具栏）：`.cluster`。
- 返回按钮宽屏隐藏：`.back-on-wide`（配合 `.sidebar-layout`）。

**布局 token（app.css `:root`）**：测量栏宽度 `--measure-default`(1180) / `--measure-wide`(1320) / `--measure-narrow`(720) / `--measure-form`(560) / `--measure-medium`(960)。容器断点 `--cq-narrow`(520) / `--cq-sidebar`(720) / `--cq-compact`(360)。**禁止再硬编码这些数值**。

**platformStore 语义化派生量**：业务页需要视口级判断时，**优先用语义化派生量**（表达「布局意图」而非「设备类型」）：
- `useMasterDetail`：宽屏采用 master-detail（左导航+右内容 / split-pane）。
- `useSidebarChrome`：桌面常驻侧栏档（全宽内容区、rightSidebar）。
- `hideRedundantEntry`：桌面侧栏已直达，页面内隐藏冗余入口（返回/创建按钮）。

直接读 `isWide/isDesktop` 仅限：① `app-shell` 导航 chrome 装配 ② 上述派生量无法覆盖的特定视口行为。**纯布局密度差异必须走容器查询原语，不得用 JS 判断**。

**红线**：
- 禁止用 `max-w-3xl` / 内联 `max-width` 等各自为政——统一走 `.page-shell` 档位。
- 禁止内容 `w-full` 平铺到容器边缘（无 max-width 又无 `page-pad`）。
- 禁止 sticky 头部宽度与正文不一致。
- **禁止业务页用 `platformStore.isMobile/isTablet/isDesktop/isWide` 切换纯布局密度**——用 `.density-grid` 等容器查询原语。
- 视口级判断用语义化派生量，禁止裸用 `isWide`。
- 禁止硬编码测量栏宽度（1180/720/560…）与断点（600/960/1280…）——走布局 token。
- 禁止在品牌色之上硬编码白字——用 `--text-on-accent`。例外：iOS 开关滑块、地图标记。
- 改动布局后 `bun check` + `bun run build` 校验。

### 导航与页面框架

**设备导航模型（四档基线）**：断点：桌面 `≥1280`；平板 `min(w,h)≥600 且 <1280`；其余移动。`isWide = 宽度≥960`。

| 档位 | 导航 chrome | 内容布局 |
| --- | --- | --- |
| 桌面 | 顶栏(52px) + 常驻侧边栏(192px) | `desktop-main-card` 内经 shell 收束 |
| 宽平板 | 浮动玻璃药丸底栏 `TabletBottomNav`（4 tab + 中央 FAB） | **复用桌面双栏**；chat 走 split-pane |
| 窄平板 | 同上浮动玻璃底栏 | **单列** shell + `page-pad` |
| 移动 | 通栏底栏 `MobileBottomNav`（4 tab + 中央 FAB） | 单列；左右各约 16px 兜底留白 |

- 底栏只放 4 个高频顶层目的地 + 中央发布 FAB。其余项经「我的」或首页「查看全部」下钻进入。
- `/users/nearby`、`/groups/nearby` 属于 `SUB_PAGE_EXACT`（隐藏底栏 + 带返回）。
- 首页附近人/群组区块标题与入口常驻，不随数据条件隐藏。

**返回按钮**：顶层目的地（侧边栏可直达）桌面不显示返回；移动/平板保留。钻取页/详情页三档设备都保留返回。

**不重复入口**：位置与半径调整唯一入口：侧边栏底部 `AnchorEntry`（桌面）/ 头部 pill（移动·平板）。发布、搜索、通知、设置等已在顶栏/侧边栏存在的全局动作，不在页面内重复放置。

**路由收敛**：同一类内容只保留一个顶层路由；近重复用页内 `filter-chip` 分页。已废弃路由保留 `+page.ts` 重定向。`/relations` 是唯一入口，三者为页内 tabs；`/dashboard` 重定向到 `/`；`formatHandle({ nickname, user_id })` 是 @handle 的唯一共享 helper。

**一致性审查清单（硬约束）**：
- **图标**：同一侧边栏内同列图标必须各不相同；同名近义入口须用不同图标（图标 + 语义双重消歧）；设置项图标单色 `--text-secondary`；chat 空态用 `MessageCircle`。
- **一个动作一个入口**：一页同一动作只保留一个入口；桌面通知统一由侧边栏「通知」单一入口 + 单一角标。
- **派生展示字段**：`@handle`、距离、相对时间、计数缩写等展示值必须走唯一共享 helper，禁止各页内联。
- **版式密度**：宽档概览卡用 `StatSegments` 横向填充；等高卡片用 CSS Grid，不等高用 `columns-*` 瀑布流；`EmptyState` 整页空态传 `fill`（垂直居中）。
- **标题与交互**：门面页/管理·配置页可带副标题，纯内容列表页一律不带；搜索全站即时（输入即搜，防抖）；profile 左栏个人中心快捷入口桌面隐藏。

### 设计系统（design-tokens）

所有颜色、阴影、留白**只用语义 token**（定义在 `src/app.css`，随 `[data-theme-family]` 主题族解析），**禁止**硬编码十六进制色值。

**颜色 / 表面 token**：
- 画布：`--page-bg`；表面：`--surface-bg`（卡片/头部/输入面）；描边：`--border-theme`
- 文字：`--text-primary`（标题/正文）/ `--text-secondary`（次要/说明/占位）/ `--text-on-accent`（品牌色上文字）
- 品牌：`--brand` / `--glow` / `--theme-accent-soft` / `--theme-accent-soft-border`
- 语义：`--color-semantic-online` / `--semantic-hot` / `--semantic-warning`
- 地图：`--map-user` / `--map-group` / `--map-post` / `--map-mixed`（多类型聚合，深饱和色不随明暗翻转）

**对比度 / 可访问性（WCAG）**：背景色与文字色**配对使用**，按背景类型选文字色。

| 背景场景 | 背景色 token | 配什么文字色 | 备注 |
| --- | --- | --- | --- |
| 卡片 / 头部 / 输入面 | `--surface-bg` | `--text-primary` / `--text-secondary` | 常规页面元素，跟随主题明暗 |
| 画布 | `--page-bg` | `--text-primary` | 比 surface 更深的底 |
| 彩色按钮 / 品牌块 | `--brand` | `--text-on-accent` | 品牌主色按钮 |
| 深色遮罩 / 弹层 | `--overlay-strong` / `--overlay-scene` | `--text-on-overlay`（白） | 图片 scrim、全屏遮罩、地图聚合气泡 |
| 饱和填充块（地图 marker / 徽章 / 标签 pill） | `--map-*` / `--brand` / `--semantic-*` | `--text-on-overlay`（白） | 不随明暗翻转，永远配纯白字 |

三条硬规则：
1. **彩色背景上的文字永远用 `--text-on-overlay`（纯白）**，不用 `--text-primary`（dark 下翻浅色，浅底浅字看不清）。
2. **背景色和文字色来自同一明暗体系**：翻转体系（`--surface-bg`/`--page-bg` + `--text-primary`/`--text-secondary`）；不翻转体系（`--map-*`/`--brand`/`--semantic-*`/`--overlay-*` + `--text-on-overlay`/`--text-on-accent`）。**禁止跨体系混用**。
3. **新增彩色 token 时同时定义各主题族的 dark 值**，dark 下取更亮/更饱和同色；「白字彩底」对比度 ≥ WCAG AA（正文 4.5:1，大号/粗体 ≥14px/700 时 3:1）。

**表面与质感**：当前主题族 **forest（扁平）**：阴影归零，组件扁平；其他主题族可有 neumorphic 阴影（`--shadow-light`/`--shadow-dark`/`--shadow-md`/`--shadow-lg`）。复用类：`.surface-flat`（扁平交互面）、`.surface-hero`/`.ambient-glow`（概览卡）、`.input-surface`（输入容器）。圆形图标按钮统一 `36px`（移动可 `38px`）。

**圆角**：基础 `--radius: 0.625rem`；卡片 `22px`、概览/主张卡 `24px`、输入/搜索 `18px`、小面 `16px`、chip/pill/头像 `999px`。**token 阶梯**（`--radius-*`，禁用魔法数）：`xs 8` / `sm 12` / `md 18` / `lg 22` / `xl 24` / `xxl 22`(对齐鸿蒙 `RADIUS.XXL`) / `pill 40` / `button 28`。原语内联 `border-radius:Npx` 必须改用 `var(--radius-*)` 或 `rounded-[var(--radius-*)]`。

**字号 / 字重**：**token 阶梯**（`--text-*` 命名空间，自动生成 `text-*` 工具类；禁用内联 `font-size:Npx`）：

| token | class | 用途 |
| --- | --- | --- |
| `--text-xs` 10px | `text-xs` | eyebrow/micro |
| `--text-caption` 11px | `text-caption` | 时间戳/badge/小注 |
| `--text-meta` 12px | `text-meta` | 次要/说明 |
| `--text-body` 13px | `text-body` | 正文（最高频） |
| `--text-body-lg` 14px | `text-body-lg` | 强调正文/主按钮 |
| `--text-title` 16px | `text-title` | 卡片/列表标题 |
| `--text-h2` 18px | `text-h2` | 页级标题 |
| `--text-h1` 20px | `text-h1` | 区块 H1/stat |
| `--text-display` 24px | `text-display` | 门面页大标题/群组名 |
| `--text-display-lg` 36px | — | 错误页 hero |

红线：`.svelte` 内禁止内联 `style="font-size:Npx"`；H1 字重 `600`。

**间距 / 节奏**：横向留白 `clamp`（`.page-pad`/`.page-pad-x`）；头部内元素纵向 `mb-2 ~ mb-3`；网格/列表 gap `.feed-density-grid` / `gap-2.5 ~ gap-5`；桌面侧边栏 `192px`，顶栏 `52px`，右栏 `220px` + `border-left`。

**动效**：点击反馈缩放 `active:scale-[0.88~0.97]`，`transition-transform duration-150`；主题切换 `transition-colors duration-500`。

**图标**：统一 `lucide-svelte`，按需单文件引入；线宽默认 `1.5`，选中/强调 `2`。**同一导航面内图标唯一表义**；图标默认单色走 `--text-secondary`/`--text-primary`；**图标实现一律用 lucide 组件**，禁止手写内联 `<svg>` 复刻。
