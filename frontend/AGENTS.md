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
- UI 组件：bits-ui 生态组件（`components/ui/` 封装层：dialog、dropdown-menu、popover、button）+ 页面级自研视觉组件
- 包管理：bun（`bun install` / `bun run`）
- 测试：vitest + @testing-library/svelte（宿主机 `bun run test`）
- 组件测试约定：与组件同目录的 `__tests__/Xxx.test.ts`（如
  `components/shared/__tests__/Avatar.test.ts`）；snippet 类 props
  在 jsdom 下传参不稳定，契约用例以渲染/交互/布局差异为准
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

### 浏览器巡检（可选，需 dev 环境已启动）

前置：dev 环境已启动（见上），宿主机已安装 `agent-browser`
（`npm i -g agent-browser && agent-browser install`）。

```bash
# 桌面 20 页
bun run browser:qa
# 桌面 20 页 + 移动 5 页（390×844）
bun run browser:qa:mobile
# 指定网关地址（如 worktree 独立环境 8889）
bash scripts/browser-qa.sh --base http://localhost:8889 --mobile
```

以 MSW mock + 访客身份自动巡检：横向溢出 / 坏图 / h1 数量 / TabBar 数量 / 控制台错误，
逐页截图并生成报告。输出统一在仓库根 `.workwork/browser-check/qa-<时间戳>/`
（已 gitignore，不提交）。

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
- **改动生效闭环**：代码改动合并进主分支后，若主实例开发容器在运行（`http://localhost:8888`），必须主动帮开发者执行 `docker compose -f docker-compose.yml -f docker-compose.dev.yml restart frontend nginx` 并确认 healthy，确保刷新浏览器即可立即看到最新改动效果
- Svelte 组件互引必须写显式 `.svelte` 扩展名（默认导出）；Svelte 5 单文件单组件
- 新增依赖后容器内需重装（entrypoint 有自动检测，或 `docker compose up -d --force-recreate frontend`）
- 每个路由 `+page.svelte` 必须设置 `<svelte:head><title>`（格式 `页面名 · HotSpot`）；
  `(app)` 组外层已提供 skip link（`#main-content`），路由内容滚动区
  `RoutedContentViewport` 的 `<main id="main-content">` 不得移除

## UI 设计原则（用户硬性要求）

1. **好看**：效果克制、有节奏，不堆叠效果，不用生硬或突兀的视觉。
2. **交互状态一眼可辨**：hover / focus / active / disabled 四态都必须有明确可感知的
   差异，且差异长在元素本体上（变色、描边、内凹、缩放、投影），禁止「元素外再画一圈」。
3. **风格统一**：复用既有原语（SearchInput、input-surface、card-hover、
   image-hover-frame、focus-ring、hover-btn/scale-btn/tap-*、TabBar、ListItem…）
   与 design token，禁止每页自造交互样式。
4. **风格走 design token**：所有风格类变化（颜色、阴影、圆角、字号、字距、行高、
   透明度、边框粗细、动效时长等）必须落在 token 体系（primitive → semantic →
   shared/registry），页面只消费语义类（text-*、bg-*、border-*、shadow-*、
   rounded-*、tracking-*、opacity-*、duration-* 等），禁止内联硬编码风格值
   （如 `color:#xxx`、`opacity:0.45`、`border-radius:Npx`、`border:1px solid`、
   `transition:... 0.2s ...`）。改主题/节奏只需改 token 文件，全站一致生效。
   登记例外：布局几何（width/height/position/gap 的 px）与 StatusBar 模拟系统
   chrome 的微圆角属于非主题风格，不受此条约束。
5. **跨端色彩对齐**：harmonyapp `ThemeConstants.ets` 的 ColorConfig 与本项目
   `themes/*.css` 的 `--cst-palette-*` 是同一套色板的两端表达，改色必须两侧
   同步。`bun run check:theme` 已串联
   `scripts/check-cross-platform-colors.mjs`（映射表内嵌于脚本：前端 forest ≡
   harmonyapp default_clean、success/error/info 随主题族色源），CI 会在漂移时
   拦截；主题选择器 swatch 也已纳入同一校验。
   同链还有 `scripts/check-utility-classes.mjs`：扫描 class 属性中消费的 token 类名，
   校验每个类名都能在 `theme-registry.css` 注册键 / `themes/*.css` 定义 /
   Tailwind 自带命名空间 / 项目自定义 `.class` 中找到来源——Tailwind v4 对未注册
   utility 静默失效（写了不报错也不生成 CSS），这是唯一的防线。
   还有 `scripts/check-cross-platform-scale.mjs`：校验字号档
   （`--text-*` ↔ 鸿蒙 `FONT.SIZE`，9 档）与圆角档（`--radius-*` ↔ 鸿蒙
   `RADIUS`，5 档）逐值一致，单端新增近邻档/改档即 CI 拦截。已登记豁免：
   鸿蒙 `FONT.SIZE.LG=16` 为 HarmonyOS 平台适配档。改字号/圆角 token 必须
   两端同步（详见 harmonyapp/AGENTS.md「跨端同步机制」）。
6. **徽标与标签（Badge / Pill）居中与排版契约**：
   - 必须同时声明 `inline-flex items-center justify-center leading-none`，杜绝未设行高导致继承全局行高（24px）产生“文字下沉贴底”或在近正方形上施加 `rounded-full` 产生“怪异胖椭圆”；
   - 紧凑型徽标（未读数气泡、状态胶囊、卡片角标）：`h-5 px-2` 或 `h-5 px-2.5`，字号 `text-xxs`；
   - 标准型标签（封面分类胶囊、个人兴趣标签、大图页码）：`h-6 px-2.5`，字号 `text-xs font-medium`；
   - 表单筛选胶囊（`TagPill`）：`sm` 为 `h-6 px-2.5 text-xxs`，`md` 为 `h-7 px-3 text-xs`。

> 具体状态契约见「样式自查清单 → 交互状态契约」。

> **UI 组件唯一目录**：`docs/ui-components.md`（原语清单、统一交互契约、
> 近重复收敛 backlog）。新增任何 UI 前先查该清单；一个交互意图只允许一种
> 反馈实现，新需求通过扩展原语 prop 表达，不新建近似组件。任何 UI 改动必须
> 截图对比审查（改前/改后、双主题、受影响断面），不允许只靠代码 diff 判断视觉结果。

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
    +page.svelte           # 首页（hero/动态/群组/用户；三端统一标题）
    nearby/+page.svelte    # 附近聚合页（平板/移动端顶级导航 tab——人/群组/动态 + 搜索）
    nearby/people/         # 附近的人
    nearby/groups/         # 附近的组
    nearby/moments/        # 附近时刻（页面样板结构权威基准）
    moments/[momentId]/    # 时刻详情
    messages/              # 消息列表
    messages/[peerId]/     # 私聊详情（peerId 即对端 user_id）
    messages/group/[groupId]/  # 群聊详情
    me/                    # 个人主页总览（mobile→ProfileMobile，tablet/desktop→ProfilePage）
    me/moments/            # 个人主页固定分区（moments / groups 两个独立固定路由，
    me/groups/             # 任意设备走 ProfilePage 分区模式，不走动态段）
    me/friends/            # ⚠️ 旧路径：仅 +page.ts 做 308 重定向到
    me/following/          #    /me/relationships?tab=…（单一事实源 lib/relationships.ts），
    me/fans/               #    已无 +page.svelte，不得再在此挂任何 UI
    me/relationships/      # 关系聚合页（好友/关注/粉丝三 tab，colocate RelationshipsPage）
    me/edit/               # 编辑资料
    users/[userId]/        # 用户详情
    groups/[groupId]/      # 群组详情
    settings/              # 设置
    map/                   # 地图（+page.svelte 设备分派；desktop/ mobile/ shared/ 子目录）
    notify/               # 通知
    search/                # 搜索
```

> 分栏页（`notify/`、`moments/[momentId]/`、`users/[userId]/`）
> 使用 `SplitPageLayout`；其余 `(app)` 页面（含首页、群组详情、个人主页）使用 `StandardPageLayout`。

> 注：管理后台（spec `api_spec_admin_core.md` / `api_spec_admin_moderation.md`）已有后端契约，前端 `src/api/endpoints.ts` 也登记了 `Admin` 端点（如 `admin/rbac/me`），但**页面尚未实现，后续再补**——功能边界中的「管理后台」为待建项，勿误判为遗漏。

路由常量见 `@/lib/routes`（个人主页分区路径用 `ME_SECTION_PATHS` 静态映射，不走动态段）。

页面私有组件与路由同目录，**按设备分目录**：每个路由目录下，`+page.svelte` 做设备分派（`platform.platform`），设备专属组件放 `desktop/` / `mobile/` 子目录，跨设备共享的组件与逻辑放 `shared/` 子目录（如 `map/desktop/LocationDesktop.svelte` + `map/mobile/LocationMobile.svelte` + `map/shared/location-data.ts`）。**多组件路由的私有组件一律拆 `shared/`**（如 `nearby/groups/shared/`、`notify/shared/`、`search/shared/`）；单组件路由（如 `nearby/people/UserCard.svelte`）保持平铺。跨路由复用的页面组件（如 `components/profile/SectionFriends.svelte` 同时被 `me/moments` 分区页与 `me/relationships` 聚合页复用）放在 `components/` 对应域目录；共享组件统一在 `components/`，设备专属共享组件按 `components/<域>/{desktop,tablet,mobile}/` 分目录（如 `layout/mobile/MobileLandscape.svelte`、`layout/desktop/DesktopLayout.svelte`）。**设备分派规则（页面内布局分支）**：
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
- **临时豁免/已登记例外**：
  - 聊天页面（桌面端与移动端）因分栏/全屏会话布局特殊，可暂不套用
    `StandardPageLayout`/`SplitPageLayout`；但页头、列表与正文的 gutter/padding
    仍必须与样板一致（沿用 `PageHeader` + `ContentColumn`）。桌面消息分栏左栏
    （`w-75`=300px 窄列）不进 `--cst-page-gutter`，用 `px-4` 压缩 gutter（见样式
    自查清单第 8 条窄列特例）。
  - **封面/紧凑型资料头**（`groups/[groupId]`、`users/[userId]`）：标题与封面/头像
    绑定，不在 header 操作行内，故不套 `PageHeader`，自建返回/分享/更多操作行 +
    封面下方 `<h1>`。与 `moments/[momentId]`（有独立标题"动态详情"，套 PageHeader）
    区分。
  - 移动个人主页（`ProfileMobile`）：自建 eyebrow 页眉，非标准列表页 header；
    设置/外观/编辑入口统一收在身份卡的拇指操作区，页眉不放重复入口。
  - 地图（全画布）与登录页（无 MainLayout 外壳）不适用。
- 强制统一布局是为了保证切换页面时右侧内容区不发生水平偏移；存量页面需逐步对齐，
  不得新增其他例外（新增例外必须在上方登记）。

**权威样板页：`src/routes/(app)/nearby/moments/+page.svelte`。** 普通页面以它为基准，
结构固定为：

```
StandardPageLayout                     # 全站唯一外层布局（components/layout/page/）
├── header snippet
│   └── PageHeader                     # 三端统一页头
│       ├── title / titleSize          # 统一档：桌面 xl(20) / 平板·移动 2xl(24)
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
   按钮；`below` 只放搜索框与 Tab 栏等 header 附属区。**below 首元素必须带
   上间距**（`mt-3`，与样板 nearby/moments 一致）——TabBar/搜索框直接贴 title
   属于间距错误；below 内多行元素之间用 `mt-3` 分隔。
3. **分类/筛选/排序 Tab（强制统一）**：一律使用 `TabBar`
   （`components/navigation/TabBar.svelte`，胶囊样式 + 激活项 accent 实底胶囊 +
   `aria-pressed`，对齐鸿蒙 TabsPillGroup 分段式视觉），放 `below` snippet 内
   随 header 吸顶；tab 状态可存 URL query（如 `?tab=`）以支持深链，切换用
   `goto` 更新 URL。
   **禁止再写手写 tab/chip 变体**（通知/搜索/个人主页/地图 sheet 的手写胶囊已统一为
   TabBar）。表单内的标签/多选 chips 一律用 `TagPill`；网格选择器
   （主题 swatch、发布可见范围）属例外，不套 TabBar。
4. **正文与滚动**：正文直接放 children；`StandardPageLayout` 页面滚动由
   `RoutedContentViewport`（main）唯一承担，`SplitPageLayout` 分栏页的主区/rail 滚动由
   组件统一管理；页面不得自建滚动容器（消息分栏、地图等全画布页面除外）。内嵌
   自带 gutter 的组件时用负 margin（`-mx-[var(--cst-page-gutter)]`）抵消 ContentColumn
   gutter，避免双重内边距（参照 `me/relationships/RelationshipsPage.svelte`）。
5. **三端差异**：共用组件只写一次，设备差异用 `platform.platform` 派生（列数、字号、
   布局分支），不得复制三份页面。
6. **数据获取与状态**：`$effect` + `cancelled` 竞态清理模式（参照样板页）；加载中用
   spinner、空列表用 `EmptyState`（参照样板页 feedState snippet）；数据加载失败
   必须展示 `LoadError`（`components/data-display/LoadError.svelte`）并提供重试
   （`reloadKey` 递增触发 effect 重跑），禁止静默转空态。
   分页/无限滚动列表**内容耗尽时必须渲染 `ListEnd`**（`components/data-display/ListEnd.svelte`，
   文案「没有更多了」）——「加载更多」按钮消失而无收尾提示会让用户无法区分
   「已到底」与「加载卡住」。四种状态各司其职：空列表 `EmptyState`、加载中
   `InlineLoading`、失败 `LoadError`、到底 `ListEnd`，不得互相替代。
7. **列表骨架契约**：列表一律用 `ListContainer` + `ListItem`（`components/list/`）。
   joined 行（默认 tone）**禁止叠加 rounded/border/shadow/ring 卡片类**——行间分隔
   由容器 divide 线承担，圆角外框由容器级（`ListContainer bordered` 唯一外框）或
   `tone="card"`（独立卡片容器 + gap 分隔）承担；调用方 className/style 只允许布局
   类。**高亮形状由组件统一应用**：joined 行 hover/focus/pressed 时背景自带
   `rounded-small`（12px，`--cst-shape-sm`）圆角，非高亮保持直角连接行视觉；
   圆角随背景填充出现并纳入 transition——调用方不得重复设置。双重防线：
   ESLint 自定义规则 `hotspot/no-list-item-card-classes`
   （`eslint-rules/`，Svelte AST 级，IDE 即时报错）挂入 `bun run lint`/`bun run check`；
   ListItem dev 模式运行时警告。
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
  stores/         # runes 模块单例（auth/theme/platform/presence/ws/composer/toast，.svelte.ts）
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
  semantic.css           # Layer 1: 语义 token（--cst-text-primary 等 → --cst-palette-*）
  themes/vibrant.css     # Layer 2: 星际主题（全部 --cst-palette-* 值）
  themes/forest.css      # Layer 2: 清野主题（全部 --cst-palette-* 值）
  layout.css             # 布局变量（context-rail、workspace）
  base.css               # @layer base（排版 + 按钮基线）
  theme-registry.css     # @theme inline Tailwind 注册（字号 text-*/字距 tracking-*/行高
                         #   leading-*/圆角 radius-*/阴影 shadow-raised-*+inset-*+glow-*/动效 ease-*）
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
- **Token 两层链**：`--cst-text-primary` 等语义名 → `--cst-palette-*` 原始值
  （无旧名兼容层）
- **配色映射速查**（页面中如何选 token）：
  - 正文/辅助文字：`--cst-text-primary` / `--cst-text-secondary`
  - 主题色背景上的文字：`--cst-text-on-accent`（按钮/选中态；**随明暗模式切换
    深浅**——light 白配深色 accent、dark 深配浅色 accent）
  - 状态色背景上的文字：`--cst-text-on-status`（online/warning 实底，如连接横幅）；
    danger 实底按钮一律走 `--cst-text-on-accent`（随明暗翻转：light 白 / dark 深；
    vibrant light danger `#C81E4E` 配白字 5.60，对齐鸿蒙 error 同值，与各确认按钮一致）
  - 图片/遮罩上的反色文字：`--cst-content-inverse`（**恒白**，仅用于图片/遮罩/
    深色 overlay 上，禁止用于 accent/status 实色背景——那些背景在 dark 下是浅色，
    恒白文字会低对比隐形；正确做法见下条）
  - 图钉色背景上的文字：`--cst-content-on-marker`（地图标记）
  - **accent 背景上的文字/图标一律用 `--cst-text-on-accent` 系列**（含次级文字
    用 `opacity-70/80` 派生，参照 ListItem/ChatListItem selected 态）；图钉等
    marker 色与 accent 同色（`--cst-palette-map-user` = `--cst-palette-accent-primary`），在
    selected 行内需换实色底衬托（参照 `PinMedia` 的 `selected` prop）
  - 遮罩/玻璃：`--cst-surface-scrim`（弹层遮罩）、
    `--cst-surface-media-scrim-strong`（照片遮罩）、
    `--cst-surface-cover-scrim`（封面渐变）、`--cst-surface-glass-badge`（玻璃徽章）
  - hover/active：优先 `hover:bg-accent-soft` 等 token 类；背景填充走
    `--cst-interactive-hover-fill` / `--cst-interactive-active`
  - 弱化嵌入面（分段控件底轨、置顶会话行等「比卡片低一档」的静态填充）：
    `--cst-surface-subtle`（`bg-surface-subtle`）。与 `bg-interactive-hover`
    的区别是语义：前者是「静态的凹陷/次表面」，后者是「悬停反馈」，不可互换。
    该表达式经 `--cst-text-primary` 自适配明暗（light 微沉 / dark 微浮），
    因此只在 themes/*.css 的 light 块定义一次，**不算缺失 dark 覆盖**
- **画廊连续画布契约（强制）**：`--cst-surface-page`（= page-bg）是应用内唯一的
  整层底色。外壳各层（外框/内框/视口/状态栏/安全区/导航底座透明区）一律只许用它；
  层级只靠白卡（`surface-primary`：`card-raised`/`surface-hero`）+ 发丝描边 +
  主题投影表达。禁止再引入第二个整层底色（`workspace` token 已删除，桌面舞台
  并入画布，侧栏/主面板/顶栏走白浮卡语言）。分壳规则见 `utilities.css`：
  移动/平板视口直接暴露画布，桌面视口落在白浮卡内（`desktop-main-scroll > *`
  强制白底）；`PageHeader` 实底走 `.page-header-solid`（外壳感知：移动/平板画布色、
  桌面白色），sticky glass 仍走 `surface-nav-glass`。
  唯一例外：聊天页独立画布 `--cst-surface-chat`（= chat-bg，微差于页面画布，
  对齐鸿蒙 `chatBackground`），仅聊天页根/头部/输入条使用。
- **数据层禁止携带颜色**：mock 数据/API 响应只返回语义标识（tag/type/category），
  不得包含 `color`/`bg` 等展示字段；视觉色由前端按语义键映射到 token
  （参照 `notify/shared/notify-data.ts` 的 `TYPE_CONFIG` 语义键
  → token 映射模式）
- **组件/数据层禁止 color-mix 表达式**：需要淡底/淡边框/光晕时引用主题预定义
  档位 token——每个语义色族在 themes/{vibrant,forest}.css 定义三档：
  `-bg`（淡底）、`-border`（淡边框）、`-glow`（光晕），如
  `--cst-status-warning-bg/-border/-glow`、`--cst-marker-user-bg/-border/-glow`。
  color-mix 只允许出现在 `styles/theme/` 定义体系内部（themes/*.css 的 token 值、
  semantic.css 的语义派生档位、utilities.css/base.css 的既有工具类——值即最终定义）；
  页面/组件/数据层仍然禁止，需要新档位时先在定义体系补 token
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
- **主题选择器 swatch（stores/theme.svelte.ts `THEME_OPTIONS`）**：色板需在
  主题激活前静态展示**目标主题**的 accent 代表色，而 CSS 变量只能解析“当前”
  主题，故保留字面 hex（vibrant `#2E66C0` / forest `#3D7A55`）作为已登记例外；
  必须与 themes/{vibrant,forest}.css 的 `--cst-palette-accent-primary` 保持同步

- 新组件优先使用语义名（`var(--cst-text-primary)`、`var(--cst-surface-page)` 等）+ Tailwind className
- 明暗三档（`stores/theme.svelte.ts`）：`colorScheme='auto'|'light'|'dark'`（持久化键
  `colorScheme`），`auto`=跟随系统（`matchMedia('(prefers-color-scheme: dark)')`
  初值 + change 监听刷新，监听由根布局持有/清理）；`light`/`dark`=手动档（忽略系统
  变化）。老 `isDark` 键只做一次性迁移（存在→对应手动档并删键，不存在→`auto`），
  首帧脚本（`app.html`）同步三态逻辑防闪。设置页与 `AppearanceMenu` 用 `TabBar`
  三段选择（跟随系统/浅色/深色），`toggleDark` 仅作手动档间切换的兼容入口
- 主题切换通过 `<html>` 上的 `.vibrant` / `.forest` + `.dark` 类实现
- 浏览器 chrome 配色（`<meta name="theme-color">`）随当前主题画布色同步：
  首帧由 app.html 内联脚本按持久化偏好设置四态映射，之后由
  theme store 的 `applyClasses` 读取 `--cst-surface-page` 动态更新
- 悬停/按压反馈统一用 `hover-btn/action/bg`、`scale-btn`、`tap-*`、`header-action` CSS
  类（主题差异由 themes/*.css 的 `@layer utilities` 覆盖承载），避免 JS 状态动画。
  **行内次要动作的显现**统一用 `desktop-reveal`（仅 fine pointer 默认隐藏，
  行 `.group` hover / 自身 `:focus-visible` / `aria-expanded="true"` 时显现；
  触屏无 hover 则常显）——禁止各页用 `opacity-0 group-hover:opacity-100` 自行拼装
- 主题专属组件覆盖（`.vibrant .foo` / `.forest .foo`）统一放对应 themes/*.css 的
  `@layer utilities` 块，utilities.css 只保留与主题无关的通用类

## 字体（自托管，禁止第三方阻塞式引入）

- **Inter 自托管**：`static/fonts/inter-latin-wght.woff2`（latin 子集，可变字重轴
  100–900，48 KB），`@font-face` 定义在 `src/styles/fonts.css`，首帧 `<link rel="preload">`
  在 `src/app.html`（字体请求必须带 `crossorigin`，否则同源也会下载两次）。
- **中文字形走系统字体栈**，不自托管 CJK 字体：`"Inter", "PingFang SC",
  "HarmonyOS Sans SC", "Microsoft YaHei", "Noto Sans CJK SC", ui-sans-serif, system-ui…`
  （定义在 `theme/base.css`）。与 harmonyapp 使用系统字体的策略一致，也避免为 CJK
  背上数 MB 分片。
- **禁止在 CSS 里 `@import` 任何第三方样式表**（字体/图标/CDN 主题等）。`@import`
  在样式表内部是串行且阻塞的：浏览器会把整张应用样式表挂起，直到该第三方资源返回。
  本项目实测代价——`@import url("https://fonts.googleapis.com/css2?...")` 122 KB /
  2.4–5.1s，样式生效时间 2999ms；删除后同一构建 159ms（首次绘制 2908ms → 364ms）。
  需要外部字体/图标时改为自托管，或用 `media="print"` + `onload` 切回的非阻塞 `<link>`
  并注明理由。
- **许可**：字体许可全文随资源存放（`static/fonts/OFL.txt`），并在
  `frontend/ATTRIBUTIONS.md` 登记来源与许可类型。

## 样式自查清单（新增/修改样式后逐条过）

> **核心原则：少档位 + 强制收敛**。设计系统的价值在约束而非自由——档位要少、
> 近邻值必须归并（1-2px 差异人眼分不清，并存只会让界面碎、无节奏）。「不够用就
> 归到最近档位」是默认动作，新增档位需充分论证。同一类东西禁止变体过多。

1. **圆角（4 档 + full）**：控件/卡片圆角一律用 `rounded-*` 类（`--radius-*` 档位：
   **sm 12 / md 16 / lg 20 / xl 24px** + `rounded-full` 圆形 + `rounded-sheet` 28px 弹层）
   或 `var(--cst-shape-sm/md/lg/xl)`；禁止 `rounded-[13px]` 等任意值、禁止 style 内
   硬编码 `border-radius:20px`、禁止近邻档位并存（原 compact/tight/small/control/
   relaxed/panel/feature 7 档已收敛到 4 档）。例外仅限物理形状细件
   （开关半高圆角、状态条 1-3px 指示条、拖拽把手）
2. **颜色**：组件内禁止 hex/rgba 字面量（数据层同样禁止）；arbitrary 类
   （`bg-[#...]`/`shadow-[...]`）禁止含色值。阴影色值统一走 `--shadow-*` token
   （见第 11 条），不得在 `shadow-[...]` 或内联 `box-shadow` 里手写色值
3. **字号（强制，6 档）**：全站字号一律用 `text-*` 刻度类（`--text-*` token 6 档：
   **xxs=10 / xs=11 / sm=12 / md=14**；text-lg 及以上沿用 Tailwind 默认 18/20/24/30/36）。
   - xxs=10：极小 uppercase 标签（eyebrow、分组标）
   - xs=11：辅助文字（副标题、计数、时间戳）
   - sm=12：正文（原 13px 已归并，消除 1px 噪声）
   - md=14：强调正文（昵称、按钮主标、侧栏导航）
   **禁止 1-2px 近邻档位并存**（原 xxxs/15/16/base 已砍/归并）；**禁止 `text-[Npx]`
   arbitrary 类、禁止 style 内联 `font-size:Npx`**（含动态 ternary——移到 class 三元）。
   **已登记例外**（与 `base.css` 的 h1-h4 元素级映射同等地位）：
   - `Avatar` fallback 首字母 `font-size:{size*0.42}px`：随头像尺寸等比缩放，物理比例
   > `PageHeader` 的 `titleSize`/`subtitleSize` 已重构为**字号档位 prop**
   > （`md/base/lg/xl/2xl/3xl` → 映射 `TITLE_SIZE_CLASS`），不再有内联 `font-size`。
   > 全站页面统一传 `device === 'desktop' ? 'xl' : '2xl'`（桌面 20 / 平板·移动 24），
   > 不得逐页自选档位——标题层级是页面锚点，跨页漂移会破坏节奏。
4. **accent/status 实色背景上的内容**：文字/图标必须用对应 `on-*` token
   （`text-text-on-accent`/`text-text-on-status`/`text-content-on-marker`），
   禁止恒白 `content-inverse`；选中行内若有与 accent 同色的 marker 图标需
   实色底衬托（PinMedia `selected`）
5. **列表行**：交互行一律走 `ListItem`（高亮圆角/背景/反色由组件统一处理），
   不在页面里自写 hover 背景或叠行级卡片类
6. **token 注册与类名有效性（强制）**：新 token 若需类名消费，同步补
   `theme-registry.css` 注册，并核对类名不带 `cst-` 前缀（`text-text-on-status`
   而非 `text-cst-...`）。Tailwind v4 对**未注册 utility 静默失效**（不报错、
   不生成 CSS、svelte-check 无感），因此：
   - 写任何带 token 语义的类名前，确认它在注册层/定义层/项目自定义 `.class` 中真实存在；
   - 项目自定义工具类拼写必须与 `utilities.css` 一致（如 `no-scrollbar`，
     曾误写 `scrollbar-none` 静默失效）；
   - `bun run check:theme` 已串联 `scripts/check-utility-classes.mjs` 自动拦截，
     本地 `bun run check` 与 CI 都会失败——不要绕过该检查。
7. **hover/tap 类**：交互反馈用既有工具类（`hover-btn`/`hover-bg`/`scale-btn`/
   `tap-*`/`header-action`/`surface-ghost`/`card-hover`），不新发明交互类；
   状态差异必须一眼可辨，且禁止双重边框（高亮与元素自带边框不得叠成两圈），
   具体契约见本节末尾「交互状态契约」
8. **字距（强制）**：字距一律用 `tracking-*` 刻度类，禁止 `tracking-[Nem]` arbitrary
   类与内联 `letter-spacing:Nem`：
   - `tracking-eyebrow`=0.14em：uppercase 分组标签（外观/通知/通用 section 标、
     「在线/未读/最近」小标、「新消息/较早」通知分组标）
   - `tracking-label`=0.12em：次级 uppercase 小标（DesktopLayout 侧栏分组）
   - `tracking-field`=0.08em：表单字段标签（AuthForm 昵称/密码/验证码）
   - `tracking-title`=-0.01em：大标题微收紧（用户名、昵称 h1/h2）
   - Tailwind 内置 `tracking-tight`(-0.025em)/`tracking-wide`(0.025em) 用于
     状态栏时钟收紧、连接横幅微扩等边界场景
   **动态字距**（如选中态切换）用 class 三元（`{active?'tracking-x':''}`），禁止
   `style="letter-spacing:..."` 内联
9. **行高**：禁止 `leading-[N]` arbitrary 类与内联 `line-height:N`（一律移到
   `leading-*` 类）。优先用语义档位 `leading-card`(1.55 摘要/副标题) /
   `leading-copy`(1.75 正文阅读) / `leading-none`(单行紧凑如数字、徽标)；
   多行正文段落允许用 Tailwind 默认 `leading-5/6/7`（固定 px 行高，对可读性
   敏感、随字号变化需要不同行距，不强制收敛）。
10. **阴影（强制，3+2+3 档）**：阴影一律用 `shadow-*` token 类或
    `var(--shadow-*)`/`var(--cst-elevation-*)` 引用，禁止 `shadow-[0_3px_10px_...]`
    arbitrary 类、禁止内联 `box-shadow:Npx...` 手抄值、禁止 utilities.css 自行组合
    双向投影表达式。两套投影语言，按语义选择：
    - **双向凸起/凹陷**（控件/卡片/头像立体感）：`shadow-raised-sm/md/lg`（3 档：
      sm=4/4/9 控件、md=5/5/12 卡片、lg=8/8/18 大卡片）+ `shadow-inset-sm/md`（2 档凹陷）。
      消费 `--cst-elevation-shadow-{dark,light}` 原料——vibrant 显影、forest 透明（改由 border 表达层级）
    - **accent 光晕**（accent 实色背景发光）：`shadow-glow-sm/md/lg`（3 档：
      sm=0/3/10、md=0/4/14、lg=0/6/18）
    - **Material 单向/弹层投影**：`var(--cst-elevation-card)`/`-button`/`-modal`/`-sheet`/`-menu`
      （语义 token，themes/{vibrant,forest}.css 各自定义真值）；弹层/下拉菜单用
      投影表达层级，不依赖描边环
    **地图标记光晕**：结构 token `var(--cst-marker-shadow)`（0 2px 10px 定位点）/
    `var(--cst-marker-halo)`（0 0 8px 小光点），色值由调用方按 marker tone 提供
    （`var(--cst-marker-*-glow)`），定义于 shared.css——禁止组件内联手抄偏移量
    **命名原则**：按投影方向/语义命名（raised/inset/glow/elevation-card），禁止设计风格前缀（neu-/skeuo-/glass-）。
    **已登记例外**：`me/edit` 保存按钮 `box-shadow:0 4px 14px {saved?'online-glow':'accent-glow'}`（固定尺寸+动态双色）
11. **font-weight（强制）**：字重一律用 `font-*` 类（`font-medium`=500/
    `font-semibold`=600/`font-bold`=700/`font-normal`=400），禁止内联
    `style="font-weight:N"`。动态字重（选中态切换）用 class 三元
    （`{active?'font-semibold':'font-normal'}`）
12. **页面纵向间距（强制）**：正文 `contentClassName` 收敛到样板 `/nearby/moments`
    的约定值，禁止 `py-5`/`py-6`/`py-7 lg:py-8` 等散乱值：
    - 标准 header 页：`contentClassName="pt-4 pb-6"`（顶 padding 由 header `pt-4`
      承接，正文上下 `pt-4 pb-6`）
    - 长表单页（settings、me/edit）：`pt-4 pb-10`（尾部加大留白）
    - 封面型页（无标准 header，顶由封面承担）：`pb-6`
    - 资料总览长内容页（ProfilePage 总览）：`pt-6 pb-8`（呼吸型变体）
    - 状态/空态区：`py-20`（样板 `feedState` + `EmptyState` 内置）
    **header 内 ContentColumn** 统一 `pt-4 pb-0`（顶留白、底不留白，below 区自带头部
    间距）；`below` snippet 首元素必须 `mt-3`（与样板一致，见页面布局契约第 2 条）。
    **窄列特例**（不进 `--cst-page-gutter`，用 `px-4`（16px）压缩 gutter，已登记例外，
    其他窄列场景需在 AGENTS.md 补登记）：
    - 桌面消息分栏左栏（`w-75`=300px）
    - 桌面地图左侧面板（`w-85`=340px，`LocationDesktopPanel`：PageHeader 传
      `contentClassName="px-4 pt-4 pb-0"`，搜索/Tab/列表/确认区统一 `px-4`）
    **布局尺寸 token**（layout.css 定义，禁止组件内裸数字）：
    `--cst-desktop-sidebar-width`（224px 展开）/ `--cst-desktop-sidebar-collapsed`（72px 折叠）/
    `--cst-tablet-dock-max`（680px 平板 Dock）/ `--cst-cover-height`（clamp 220→340px 封面）/
    `--cst-cover-height-lg`（clamp 220→380px 大封面）/ `--cst-cover-height-sm`
    （clamp 120→160px 编辑资料小封面）
13. **尺寸与间距**：UI 控件/图标尺寸优先用 Tailwind `size-*` 刻度类（`size-9`=36px
    等），减少内联 `style="width:Npx;height:Npx"`。**间距**（gap/px/py/mt/mb）用
    Tailwind 默认 4px 步进刻度（`gap-1/1.5/2/2.5/3...`），**不强制收敛、不建 spacing
    token**——间距的 2px 差异是布局设计的一部分（卡片内 8px、section 间 16px 等
    刻意微调），与字号/圆角的"1px 无感噪声"性质不同，强行归并会破坏视觉层次。
    横向页面 gutter 是唯一例外（`--cst-page-gutter`，由 ContentColumn 统一管理）。
    存量内联尺寸（地图 PIN、StatusBar 信号/电池等物理装饰细件）可保留为例外
    布局尺寸例外（非控件尺寸，可用类时仍优先用类）：对话框高度
    `78vh/max-height:700px`（桌面居中模态统一档）、移动 bottom-sheet 高度
    `85%`（新会话）/`92%`（建群，表单更高）、聊天空态 `min-height:240px/max-width:220px`、
    上传占位 `height:110px`、封禁页 `max-width:360px`、横幅 `h-7` 类化后
    仅剩物理例外——新增此类布局尺寸时优先 Tailwind 类，确需内联则登记
16. **毛玻璃模糊半径（强制，3 档）**：`backdrop-filter:blur()` 半径一律用
    `var(--cst-blur-sm/md/lg)` token（sm=8 轻徽章、md=16 sticky header/overlay、
    lg=24 底部导航/tablet dock），禁止内联 `blur(Npx)` 字面量（原 8/10/16/18/24/28px
    六种已收敛到 3 档）
17. **opacity**：优先用 Tailwind `opacity-*` 标准档类（10/20/30.../100），减少内联
    `style="opacity:0.N"`。装饰性图标/文字透明度应归标准档（0.5→opacity-50 等）；
    导航激活态背景等精确视觉调校允许保留内联值（0.12~0.15 区间属刻意微调）
18. **焦点可见性（强制）**：所有可交互元素必须有可见键盘焦点环。统一用
    `focus-ring` 类（`focus-visible` 时 accent outline）或等效的
    `focus-visible:ring-2 focus-visible:ring-accent-default`；**禁止无替代地使用
    `outline-none`**（Tailwind v4 的 `outline-none` 会压掉 base.css 的全局按钮轮廓）。
    图标按钮必须同时提供 `aria-label`；紧凑图标按钮（<44px）叠加 `hit-slop` 扩展触控热区
19. **减弱动效（强制）**：新增加载/装饰动画必须考虑 `prefers-reduced-motion`；
    utilities.css 已提供全局降级（动画瞬时化、`map-pulse-halo` 停用、
    spinner 放慢），不得绕过
14. **动效曲线（强制）**：过渡缓动曲线一律用 `var(--cst-ease-*)` token，禁止内联
    `cubic-bezier(...)` 字面量：
    - `--cst-ease-standard`=(0.22,1,0.36,1)：标准减速曲线（绝大多数 transition）
    - `--cst-ease-bounce`=(0.34,1.56,0.64,1)：弹簧 overshoot 曲线（FAB、toggle 滑块、
      地图标记放大、sheet 拖拽回弹等「有弹性」的入场/位移）
    - Tailwind 类消费经 `--ease-standard`（theme-registry 注册）
    **duration 不强制 token 化**：`0.10s`~`0.30s` 的精细时长差异是 Figma 动效契约的
   有意设计（btn 0.17s、action 0.14s、tap 0.10s 等），`--cst-motion-standard:220ms`
    仅作基准参考，不代表统一时长；组件按 Figma 契约保留各自动效时长
15. **z-index 层叠**：分两类，不混用：
    - **全站页面级层叠**（弹层/抽屉/Toast 等跨组件浮层）：用 `--cst-z-*` token
      （dropdown=10 / overlay=40 / modal=50 / toast=60），经 `z-[var(--cst-z-modal)]`
      等引用
    - **局部 stacking context 内层叠**（卡片角标盖图片、地图容器内部排序、sticky header）：
      用 Tailwind 默认 `z-10/20/30` 或 arbitrary `z-[1]/z-[2]`，表达「盖几层」的相对
      语义，**不套全站 token**——它们在各自 `position:relative` 容器内独立成层，
      与全站 z-index 刻度无语义对应。地图组件内部的 `z-index:0~50` 内联属此类
      （地图外层 `overflow:hidden` 形成独立 stacking context）

### 交互状态契约（现状即规范，改动前先对齐）

| 元素 | hover | focus / active |
|------|-------|----------------|
| 输入框（input-surface） | — | 边框加深（border-strong）+ 内缘 1px 加深线 + 内凹阴影（`--shadow-inset-sm`），底色保持中性 |
| 图片卡（image-hover-frame） | 内嵌 accent 2px 描边 + 图片缩放 1.045（仅 fine pointer） | — |
| 带边框卡片（card-hover） | 边框变 accent（`hover:border-accent-default`）或投影抬升，不画外环 | — |
| 可交互 surface-flat | 描边加深（border-strong） | — |
| 弹层 / 下拉菜单 | 无描边环，层级靠 elevation-menu 投影 | — |

#### 按钮 hover 底色：两档，按「控件是否有可见轮廓/底色」选

utilities.css 的全局规则会给**无 `bg-*` 且无 `hover:*` 类**的 `<button>` 自动填
`--cst-interactive-hover-fill`（仅 fine pointer）。因此有底色/描边的按钮被该规则
排除，必须**显式**给 hover；两档按控件是否"有可见轮廓"选择，不要混用：

| 控件形态 | hover 底色 | 例 |
|----------|-----------|-----|
| 透明底图标按钮 / 行式按钮 / 未选中 pill | `hover:bg-interactive-hover`（中性灰） | 图标按钮、ListItem、`CreateSheet` 心情/话题按钮 |
| 有可见轮廓或底色（border / ring / `bg-surface-primary`） | `hover:bg-accent-soft`（accent 淡底，更强信号） | `ProfilePage` 描边白底按钮、`AppearanceMenu` ring-1 控件、下拉菜单项 |

- **禁止透明度越档**：`hover:bg-accent-soft/50` 之类 `/N` 变体一律用整档
  `hover:bg-interactive-hover`（accent-soft 本身就是 10~14% 淡底，再砍半档等于
  用 opacity 造新档位，违反第 17 条）。
- 已选中态（`aria-pressed=true` / `bg-accent-soft` 打底）的 hover 走
  `hover:bg-interactive-active`，不用 interactive-hover。
- 导航 Tab 不参与本表：`.tab-item` 规则统一提供 `--cst-interactive-hover-fill`
  （无 layer，优先级高于 `@layer utilities`），**不要在 TabBar 上再写 `hover:bg-*`**
  （写了也不生效，是死类名）。

硬性规则：
- 输入框聚焦态由整框承载：内层 input/textarea 不画自身焦点框
  （utilities.css `.input-surface:focus-visible` 统一压制，与第 18 条焦点可见性配合）；
  底色保持中性，不引入 accent 色，避免与错误态等 accent 元素抢视觉。
- 禁止双重边框：card-hover 只抬投影不画外环；带边框的卡片 hover 用
  `hover:border-accent-default`；surface-flat 的 hover 描边仅对可交互元素生效。
- 图片缩放必须由外层 `overflow:hidden` + 圆角裁剪，图片任何像素不得溢出卡片。
- **禁止渲染空 src 图片**：`<img src="">` 会被浏览器解析为当前页面 URL 而显示破图
  （"我的瞬间"页曾因此对无图动态整体破图）。无图/空 URL 一律走占位：
  大图区用 `MediaPlaceholder`（data-display，ImageOff 图标 + accent-soft 底），
  头像用 `Avatar`（首字母兜底），列表小缩略图内联 `{#if url}` 分支。
- **计数变化反馈**：点赞/评论等瞬时数值变化用 `count-pop` 类 + `{#key 值}` 重建元素
  触发弹跳（utilities.css cst-count-pop，snappy + bounce，自动遵循 reduced-motion）。
- **大图查看统一入口**：任何"点击查看大图"交互一律复用 `components/overlays/ImageViewer`
  （bits-ui Dialog 底座：Esc/外点关闭、左右键/滑动切图、单击图片切换放大（以点击
  位置为锚）与滚轮连续缩放、放大后拖拽平移、缩略图条），禁止各页自建 lightbox。
  **入口仅限动态详情页**（单图封面/多图网格）；列表卡片与画廊一律整卡跳详情，
  不在卡片上直接开大图。
- **地图筛选菜单统一**：`map/shared/MapFilterMenu.svelte` 是桌面/移动唯一筛选实现
  （Popover 外点/Escape 关闭、徽章、至少一项约束、「全部显示」重置）；地图空白点击
  一律经 `MapCanvas onMapTap`（内部区分拖拽，拖动不误触），禁止父级用原生 onclick 判定。
