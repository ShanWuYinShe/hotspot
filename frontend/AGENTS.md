# HotSpot 前端（SvelteKit / Svelte 5 / TypeScript）

本文件是前端专属约束。根 `AGENTS.md` 中的通用原则（Git 规范、Commit 格式、Docker 等）仍然适用。

## 功能边界

Svelte Web 主前端。认证/注册/发现与搜索/信息流/生活瞬间与评论/群组/聊天/附近/用户页面/个人中心/地图/管理后台。用户管理、支付与交易不在范围。

API 契约见 `spec/api_spec_*.md`（`spec/` 目录只读权威）。

## 技术栈

- 框架：SvelteKit 2 + Svelte 5（runes）+ `adapter-static` SPA 模式
  （`ssr=false` + `fallback: 'index.html'`，nginx `try_files` 回退；全部页面客户端渲染。
  本项目不需要 SSR，禁止引入服务端渲染相关代码、依赖或 `hooks.server.ts`/`+server.ts`）
- 样式：Tailwind CSS v4 + CSS Token 三层架构（`hover-*` CSS 类承载交互反馈）
- UI 组件：bits-ui 生态组件（`components/ui/` 封装层：dialog、dropdown-menu、popover）+ 页面级自研视觉组件
- 包管理：bun（`bun install` / `bun run`）
- 测试：vitest + @testing-library/svelte（宿主机 `bun run test`）
- Mock：MSW（仅 dev 且 `PUBLIC_API_MOCK !== 'false'` 时启用；`hooks.client.ts` 的
  `init()` 中注册——service worker 首次注册后需刷新一次才接管，带 sessionStorage
  防死循环标记。`mocks/browser.ts` 的 `setupWorker(...handlers)` 未配显式 scope
  （默认同源）；`mocks/toggle.ts` 的 `onUnhandledRequest` 对非 `/api/v1/` 请求
  直接 `bypass`——只拦截 API 流量，页面/静态资源不受 mock 影响，`/api/v1/`
  未匹配的请求才 `warn`。mock 开关经 localStorage `hotspot:mock-enabled`
  持久化（`toggleMock()` 切换后 reload））

## 开发模式：必须通过 Docker Compose + Nginx 网关

前端是纯 UI 层，所有 API、WebSocket、S3 请求都依赖后端服务。禁止直接在宿主机启动 dev server。

### 正确流程

```bash
# 1. 启动全部服务（含前端 Vite dev server + 后端 + DB + nginx）
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 2. 唯一访问入口
open http://localhost:8888
```

worktree 内启动与收尾清理见根 `AGENTS.md`「worktree 调试与清理」。

Nginx 网关（`docker/nginx.dev.conf`）统一反代：
| 路径 | 上游 |
|------|------|
| `/` | Vite dev server（容器内 `frontend:5173`，不暴露到宿主机） |
| `/api/` | Go 后端（容器内 `backend:8081`） |
| `/ws/` | Go 后端 WebSocket |
| `/s3/` | SeaweedFS S3 |

### 宿主机上允许的调试操作

- 类型检查：`bun run check`（svelte-check）
- 测试：`bun run test`（vitest，不依赖 Docker）
- 构建验证：`bun run build`

`bun run build` 是 `vite build`（不含 `svelte-kit sync`；重建 `.svelte-kit/` 的是
`dev`/`check` 脚本里的 `svelte-kit sync`），但会读取并写入 `.svelte-kit/` 产物，
与正在运行的 dev server 共用该目录，可能相互干扰。需要跑 build 前先停掉 frontend
容器（`docker compose stop frontend`），跑完再 `docker compose start frontend`。

## 构建

```bash
bun install --cwd frontend
bun run --cwd frontend build   # adapter-static 产物输出到 build/
```

## 硬约束

- 调试/开发/预览时，默认入口是 `http://localhost:8888`（nginx 网关），不是 dev server 地址
- 修改 nginx 配置后必须重启对应容器
- Svelte 组件互引必须写显式 `.svelte` 扩展名（默认导出）；Svelte 5 单文件单组件
- 新增依赖后容器内需重装（entrypoint 有自动检测，或 `docker compose up -d --force-recreate frontend`）

## 路由系统

基于 SvelteKit 文件路由（SPA fallback 服务一切路径，动态路由无需占位枚举）：

```
src/routes/
  +layout.ts               # export const ssr = false（SPA 模式开关）
  +layout.svelte           # 全局壳（styles + stores + AuthGate；MSW 注册在 hooks.client.ts）
  +error.svelte            # 404
  (auth)/login/+page.svelte    # 登录页（无 MainLayout 外壳）
  (app)/
    +layout.svelte         # MainLayout 外壳（桌面/平板/移动/横屏四端分派）
    +page.svelte           # 首页（附近动态；三端聚合视图——hero/动态/群组/用户）
    nearby/people/         # 附近的人
    nearby/groups/         # 附近的组
    nearby/moments/        # 附近时刻（页面样板结构权威基准）
    moments/[momentId]/    # 时刻详情
    messages/              # 消息列表
    messages/[peerId]/     # 私聊详情（peerId 即对端 user_id）
    messages/group/[groupId]/  # 群聊详情
    me/                    # 个人主页总览（mobile→ProfileMobile，tablet/desktop→ProfilePage）
    me/moments/            # 个人主页固定分区（moments / groups / friends / following / fans 五个
    me/groups/             # 独立目录均为固定路由，任意设备走 ProfilePage 分区模式，不走动态段）
    me/friends/
    me/following/
    me/fans/
    me/relationships/      # 关系聚合页（好友/关注/粉丝三 tab，colocate RelationshipsPage）
    me/edit/               # 编辑资料
    users/[userId]/        # 用户详情
    groups/[groupId]/      # 群组详情
    settings/              # 设置
    map/                   # 地图（+page.svelte 设备分派；desktop/ mobile/ shared/ 子目录）
    notifications/         # 通知
    search/                # 搜索
```

> 分栏页（`notifications/`、`moments/[momentId]/`、`users/[userId]/`）
> 使用 `SplitPageLayout`；其余 `(app)` 页面（含首页、群组详情、个人主页）使用 `StandardPageLayout`。

> 注：管理后台（spec `api_spec_admin_core.md` / `api_spec_admin_moderation.md`）已有后端契约，前端 `src/api/endpoints.ts` 也登记了 `Admin` 端点（如 `admin/rbac/get_me`），但**页面尚未实现，后续再补**——功能边界中的「管理后台」为待建项，勿误判为遗漏。

路由常量见 `@/lib/routes`（个人主页分区路径用 `ME_SECTION_PATHS` 静态映射，不走动态段）。

页面私有组件与路由同目录，**按设备分目录**：每个路由目录下，`+page.svelte` 做设备分派（`platform.platform`），设备专属组件放 `desktop/` / `mobile/` 子目录，跨设备共享的组件与逻辑放 `shared/` 子目录（如 `map/desktop/LocationDesktop.svelte` + `map/mobile/LocationMobile.svelte` + `map/shared/location-data.ts`）。**多组件路由的私有组件一律拆 `shared/`**（如 `nearby/groups/shared/`、`notifications/shared/`、`search/shared/`）；单组件路由（如 `nearby/people/UserCard.svelte`）保持平铺。跨路由复用的页面组件（如 `components/profile/SectionFriends.svelte` 同时被 `me/moments` 分区页与 `me/relationships` 聚合页复用）放在 `components/` 对应域目录；共享组件统一在 `components/`，设备专属共享组件按 `components/<域>/{desktop,tablet,mobile}/` 分目录（如 `layout/mobile/MobileLandscape.svelte`、`layout/desktop/DesktopLayout.svelte`）。**设备分派规则（页面内布局分支）**：
- **默认两分支**：`isDesktop = platform.platform !== 'mobile'`（tablet 与 desktop 同分支），适用于大多数页面（tablet 在字号/边距等细节差异由组件内 `platform` 派生，不需要独立 if 分支）。
- **按需三分支**：当 tablet 在**列数、栏宽、整段布局**上确实与 desktop 表现不同时（如 `nearby/moments` 桌面 3 列 / 平板 2 列、`nearby/people`、`nearby/groups`），显式写 `isDesktop` + `isTablet` + mobile 三分支是**合规**的，不算"逐页发明"。判断标准是"tablet 是否有独立的**布局结构**"，而非"字号/间距不同"。
- **反模式**：仅为字号/颜色/间距等可在组件内用 `platform` 单值派生的差异而开三分支——这种情况一律用两分支 + 组件内派生。

## 页面布局（普通页 StandardPageLayout / 分栏页 SplitPageLayout）

所有路由页面必须使用统一布局组件，禁止自行拼装 PageHeader/ContentColumn/页面级
滚动/页面级横向 padding：

- **普通页面**：`StandardPageLayout`（`components/layout/page/StandardPageLayout.svelte`），
  正文单列，滚动由 `RoutedContentViewport` 承担；正文内可含多列 grid（如群组详情
  main + aside、个人主页双栏），不属于分栏需求。
- **分栏页面**（桌面主区 + 右 rail）：`SplitPageLayout`
  （`components/layout/page/SplitPageLayout.svelte`），主区与 rail 滚动由组件统一管理，
  footer 固定于主区底部。当前分栏页：通知、动态详情、用户详情。
- **临时豁免**：聊天页面（桌面端与移动端）因分栏/全屏会话布局特殊，可暂不套用；
  但页头、列表与正文的 gutter/padding 仍必须与样板一致（沿用 `PageHeader` +
  `ContentColumn` 的 `px-4` 布局，不得另起页面级横向 padding）。
  地图（全画布）与登录页（无 MainLayout 外壳）不适用。
- 强制统一布局是为了保证切换页面时右侧内容区不发生水平偏移；存量页面需逐步对齐，
  不得新增其他例外。

**权威样板页：`src/routes/(app)/nearby/moments/+page.svelte`。** 普通页面以它为基准，
结构固定为：

```
StandardPageLayout                     # 全站唯一外层布局（components/layout/page/）
├── header snippet
│   └── PageHeader                     # 三端统一页头
│       ├── title / titleSize          # 桌面 20 / 平板 24 / 移动 22（样板页参考值）
│       ├── trailing snippet           # 计数、精选等右上信息
│       └── below snippet              # 搜索框 + 分类 Tab 栏（随 header 吸顶）
└── children                           # 正文，直接放 ContentColumn 内
```

约束：

1. **布局**：普通页面必须用 `StandardPageLayout`，分栏页面必须用 `SplitPageLayout`；
   header 以 snippet 传入、正文放 children；不得自行组合
   PageHeader/ContentColumn/页面级滚动/页面级横向 padding（契约见对应组件注释，
   聊天页豁免见上）。gutter 一律由 `ContentColumn` 的 `px-[var(--cst-page-gutter)]` 管理。
2. **页头**：一律用 `PageHeader`（`sticky` 默认开），`onBack` 在非桌面自动显示返回
   按钮；`below` 只放搜索框与 Tab 栏等 header 附属区。
3. **分类 Tab 栏**：下划线样式（容器 `border-bottom` + 激活项底部 2px 指示条 +
   `aria-pressed`），放 `below` snippet 内随 header 吸顶；tab 状态可存 URL query
   （如 `?tab=`）以支持深链，切换用 `goto` 更新 URL。
4. **正文与滚动**：正文直接放 children；`StandardPageLayout` 页面滚动由
   `RoutedContentViewport`（main）唯一承担，`SplitPageLayout` 分栏页的主区/rail 滚动由
   组件统一管理；页面不得自建滚动容器（消息分栏、地图等全画布页面除外）。内嵌
   自带 gutter 的组件时用负 margin（`-mx-[var(--cst-page-gutter)]`）抵消 ContentColumn
   gutter，避免双重内边距（参照 `me/relationships/RelationshipsPage.svelte`）。
5. **三端差异**：共用组件只写一次，设备差异用 `platform.platform` 派生（列数、字号、
   布局分支），不得复制三份页面。
6. **数据获取与状态**：`$effect` + `cancelled` 竞态清理模式（参照样板页）；加载中用
   spinner、空列表用 `EmptyState`（参照样板页 feedState snippet）。
7. **列表骨架契约**：列表一律用 `ListContainer` + `ListItem`（`components/list/`）。
   joined 行（默认 tone）**禁止叠加 rounded/border/shadow/ring 卡片类**——行间分隔
   由容器 divide 线承担，圆角外框由容器级（`ListContainer bordered` 唯一外框）或
   `tone="card"`（独立卡片容器 + gap 分隔）承担；调用方 className/style 只允许布局
   类。**高亮形状由组件统一应用**：joined 行 hover/focus/pressed 时背景自带
   `rounded-small`（12px，`--cst-shape-small`）圆角，非高亮保持直角连接行视觉；
   圆角随背景填充出现并纳入 transition——调用方不得重复设置。三重防线：
   ESLint 自定义规则 `hotspot/no-list-item-card-classes`
   （`eslint-rules/`，Svelte AST 级，IDE 即时报错）挂入 `bun run lint`/`bun run check`；
   仓库根 `lefthook.yml` pre-commit 提交前强制拦截；ListItem dev 模式运行时警告。
   后续需扩展同类组件级契约时，在 `eslint-rules/` 新增规则并在
   `eslint.config.mjs` 注册（最小化接入：不启用默认规则集）。

## 目录结构

```
src/
  routes/         # SvelteKit 路由（+page.svelte / +layout.svelte + 页面私有组件按设备分子目录）
  components/     # 共享组件（按域分子目录：auth/ data-display/ forms/ home/ layout/ list/
                  #   messages/ navigation/ overlays/ profile/ rail/ shared/ ui/；
                  #   设备专属共享组件再按 {desktop,tablet,mobile}/ 分目录，如 layout/mobile/、
                  #   layout/desktop/；layout/ 域含 page/ 通用页面原语子目录——
                  #   StandardPageLayout/SplitPageLayout/PageHeader/ContentColumn/
                  #   PageViewport/RoutedContentViewport）
                  #   单路由私有组件不下沉到 components/（如 moments/ 域已清空——MomentCard
                  #   等随路由走 shared/）
  stores/         # runes 模块单例（auth/theme/platform/presence/ws/composer，.svelte.ts）
  lib/            # 工具函数 + 路由常量 + 数据工厂（.svelte.ts 响应式；ws/ 为 WebSocket
                  #   客户端封装 client.ts + types.ts）
  api/            # API 客户端层（client.ts 零框架依赖）
  styles/         # CSS（主题、Tailwind）
  assets/         # 静态资源（含地图图钉 SVG）
  mocks/          # MSW mock（仅 dev；toggle.ts 提供 localStorage 持久化开关）
```
> `static/` 位于**项目根**（SvelteKit 约定，不在 `src/` 下）：`static/mockServiceWorker.js`（MSW service worker）。

## 主题 & CSS Token 架构

`src/styles/theme/` 目录按三层架构组织：

```
theme/
  index.css              # 入口，按依赖顺序导入所有分片
  shadcn.css             # Layer 0: shadcn/ui 基础变量
  shared.css             # Layer 0: z-index/shape/motion/ease（与主题无关）
  semantic.css           # Layer 1: 语义 token（--cst-text-primary 等 → --cst-pr-*）
  themes/vibrant.css     # Layer 2: 星际主题（全部 --cst-pr-* 值）
  themes/forest.css      # Layer 2: 清野主题（全部 --cst-pr-* 值）
  layout.css             # 布局变量（context-rail、workspace）
  base.css               # @layer base（排版 + 按钮基线）
  theme-registry.css     # @theme inline Tailwind 注册（含字号刻度 text-xxs/xs/sm/base/md）
  utilities.css          # @layer utilities（组件工具类 + hover-* 交互类）
```

### 配色硬约束（强制）

**所有涉及配色的取值——字体颜色、背景、边框、hover 颜色、active 颜色、阴影、
遮罩、状态色、图钉/标记色等——必须来自 design token，严格禁止在页面/组件中
单独定义色值**（禁止 `text-white`、`color:#fff`、`bg-black/50`、内联 `rgba(...)`
等字面量）。所有 token 必须能随主题（vibrant/forest）与明暗模式正确变化。

- **前缀约定**：全部 design token 使用 `cst-` 前缀（`--cst-text-primary`、
  `--cst-surface-primary`、`--cst-elevation-card`…），一眼可识别 token 来源。
  定义层（shared/semantic/themes/layout/shadcn）禁止出现无前缀变量。
- **Tailwind 注册层例外**：`theme-registry.css` 的 `@theme inline` 注册键
  （`--color-*`、`--text-xxs` 等字号、`--radius-*`、`--shadow-*`、
  `--ease-standard`）是 Tailwind 工具类生成机制，**必须保持原名**；
  其值引用 `var(--cst-*)`。
- **Token 两层链**：`--cst-text-primary` 等语义名 → `--cst-pr-*` 原始值
  （无旧名兼容层）
- **配色映射速查**（页面中如何选 token）：
  - 正文/辅助文字：`--cst-text-primary` / `--cst-text-secondary`
  - 主题色背景上的文字：`--cst-text-on-accent`（按钮/选中态；**随明暗模式切换
    深浅**——light 白配深色 accent、dark 深配浅色 accent）
  - 状态色背景上的文字：`--cst-text-on-status`（online/warning 底）
  - 图片/遮罩上的反色文字：`--cst-content-inverse`（**恒白**，仅用于图片/遮罩/
    深色 overlay 上，禁止用于 accent/status 实色背景——那些背景在 dark 下是浅色，
    恒白文字会低对比隐形；正确做法见下条）
  - 图钉色背景上的文字：`--cst-content-on-marker`（地图标记）
  - **accent 背景上的文字/图标一律用 `--cst-text-on-accent` 系列**（含次级文字
    用 `opacity-70/80` 派生，参照 ListItem/ChatListItem selected 态）；图钉等
    marker 色与 accent 同色（`--cst-pr-map-user` = `--cst-pr-accent`），在
    selected 行内需换实色底衬托（参照 `PinMedia` 的 `selected` prop）
  - 遮罩/玻璃：`--cst-surface-scrim`（弹层遮罩）、
    `--cst-surface-media-scrim-strong`（照片遮罩）、
    `--cst-surface-cover-scrim`（封面渐变）、`--cst-surface-glass-badge`（玻璃徽章）
  - hover/active：优先 `hover:bg-accent-soft` 等 token 类；背景填充走
    `--cst-interactive-hover-fill` / `--cst-interactive-active`
- **数据层禁止携带颜色**：mock 数据/API 响应只返回语义标识（tag/type/category），
  不得包含 `color`/`bg` 等展示字段；视觉色由前端按语义键映射到 token
  （参照 `notifications/shared/notification-data.ts` 的 `TYPE_CONFIG` 语义键
  → token 映射模式）
- **组件/数据层禁止 color-mix 表达式**：需要淡底/淡边框/光晕时引用主题预定义
  档位 token——每个语义色族在 themes/{vibrant,forest}.css 定义三档：
  `-bg`（淡底）、`-border`（淡边框）、`-glow`（光晕），如
  `--cst-status-warning-bg/-border/-glow`、`--cst-marker-user-bg/-border/-glow`。
  color-mix 只允许出现在 themes/*.css 的 token 值内部（值即最终定义）
- 新增组件时若缺语义 token，先在 themes/{vibrant,forest}.css 补 `--cst-*`
  定义（light + dark），不得在组件里写死色值绕过
- **新增 token 必须同步注册 Tailwind 类**：需要以 `bg-*/text-*/border-*` 类名
  消费的 token，必须在 `theme-registry.css` 的 `@theme inline` 补 `--color-*`
  注册（如 `--color-text-on-status`）；只经 `var()` 内联引用的才可跳过。
  已踩坑：`--cst-text-on-status` 未注册导致 `text-cst-text-on-status` 死类名
  静默失效（Tailwind 无此前缀，类名不生成任何样式，svelte-check 不报错）
- **图钉 SVG 资源（assets/markers/*.svg）**：主题色 fill/stroke 一律用
  `currentColor`，由 `location-shared.ts` 的 `markerSvg()` 以
  `style="color:var(--cst-marker-*)"` 染色，随主题/明暗变化；资源内仅保留
  图标固有固定对比色（深色底 `#102A43`、白色高光 `#FFFFFF`）不随主题，
  属合理例外。新增 marker 类 SVG 必须遵循：主题色走 currentColor，
  禁止内嵌主题色 hex

- 新组件优先使用语义名（`var(--cst-text-primary)`、`var(--cst-surface-page)` 等）+ Tailwind className
- 主题切换通过 `<html>` 上的 `.vibrant` / `.forest` + `.dark` 类实现
- 悬停/按压反馈统一用 `hover-btn/action/bg`、`scale-btn`、`tap-*`、`header-action` CSS
  类（主题差异由 themes/*.css 的 `@layer utilities` 覆盖承载），避免 JS 状态动画
- 主题专属组件覆盖（`.vibrant .foo` / `.forest .foo`）统一放对应 themes/*.css 的
  `@layer utilities` 块，utilities.css 只保留与主题无关的通用类

## 样式自查清单（新增/修改样式后逐条过）

1. **圆角**：控件/卡片圆角一律用 `rounded-*` 类（`--radius-*` 档位：
   compact 8 / tight 10 / small 12 / control 16 / relaxed 18 / panel 20 /
   feature 24 / sheet 28px）或 `var(--cst-shape-*)`；禁止 `rounded-[13px]` 等
   任意值、禁止 style 内硬编码 `border-radius:20px`。例外仅限物理形状细件
   （开关半高圆角、状态条 1-3px 指示条、拖拽把手）
2. **颜色**：组件内禁止 hex/rgba 字面量（数据层同样禁止）；arbitrary 类
   （`bg-[#...]`/`shadow-[...]`）禁止含色值
3. **字号（强制）**：全站字号一律用 `text-*` 刻度类（`--text-*` token 五档：
   xxs=10 / xs=11 / sm=12 / base=13 / md=14px；text-lg 及以上沿用 Tailwind 默认
   18/20/24/30/36）。**禁止任何 style 内联 `font-size:Npx`**（含动态 style
   ternary——把字号移到 class 上，style 只留颜色/布局）。已全量清理存量
   硬编码；新增代码违反即视为样式规范错误
4. **accent/status 实色背景上的内容**：文字/图标必须用对应 `on-*` token
   （`text-text-on-accent`/`text-text-on-status`/`text-content-on-marker`），
   禁止恒白 `content-inverse`；选中行内若有与 accent 同色的 marker 图标需
   实色底衬托（PinMedia `selected`）
5. **列表行**：交互行一律走 `ListItem`（高亮圆角/背景/反色由组件统一处理），
   不在页面里自写 hover 背景或叠行级卡片类
6. **token 注册**：新 token 若需类名消费，同步补 `theme-registry.css` 注册，
   并核对类名不带 `cst-` 前缀（`text-text-on-status` 而非 `text-cst-...`）
7. **hover/tap 类**：交互反馈用既有工具类（`hover-btn`/`hover-bg`/`scale-btn`/
   `tap-*`/`header-action`/`surface-ghost`/`card-hover`），不新发明交互类
