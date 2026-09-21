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

### OrbStack 本地域名（`.orb.local`）访问

同一个实例除 `http://localhost:8888` 外，还有 OrbStack 发布的本地域名，可直接走
HTTPS（OrbStack 自带受信任的开发证书）：

- **同一个 nginx 容器有多个域名**，都指向当前容器：
  `nginx.<compose 项目名>.orb.local`（服务名.项目名，主工作树即
  `nginx.hotspot.orb.local`）、`<容器名>.orb.local`（如
  `hotspot-nginx-1.orb.local`）；worktree 起的独立项目是
  `nginx.<worktree 目录名>.orb.local`。
- **只在本机有效**：解析走 **mDNS 多播**（`/etc/resolver/orb.local` 并不存在，
  路由器 DNS 对 `.orb.local` 返回 NXDOMAIN，只有 `dns-sd` 能查到）。因此手机、
  另一台电脑都解析不了——那类场景用 Mac 的局域网 IP。开了 VPN 或代理软件 TUN
  模式也可能让多播失效，重启 OrbStack 恢复。
- **`server.allowedHosts` 必须用前导点通配**：`vite.config.ts` 里写
  `['.orb.local']`，Vite 的语义是「该域及其所有子域」。**禁止写死单个主机名**——
  实测写死 `nginx.hotspot.orb.local` 时，`hotspot-nginx-1.orb.local`（容器名形式）
  与 worktree 的 `nginx.<目录名>.orb.local` 全部返回 403
  `Blocked request. This host ... is not allowed`。

### 宿主机上允许的调试操作

- 类型检查：`bun run check`（svelte-check）
- 测试：`bun run test`（vitest，不依赖 Docker）
- 构建验证：`bun run build`

### 浏览器巡检（可选，需 dev 环境已启动）

前置：dev 环境已启动（见上），宿主机已安装 `agent-browser`
（`bun install -g agent-browser && agent-browser install`）。

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

`bun run build` 是 `write-version-info`（注入版本信息）+ `vite build`（不含
`svelte-kit sync`；重建 `.svelte-kit/` 的是
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
5. **跨端色彩与刻度对齐**：harmonyapp `ThemeConstants.ets` 的 ColorConfig 与本项目
   `themes/*.css` 的 `--cst-palette-*` 是同一套色板的两端表达，改色必须两侧
   同步。校验脚本、覆盖范围与同步流程的**唯一权威描述**见根 `docs/design-system.md`
   第 14 节与 `harmonyapp/AGENTS.md`「跨端同步机制」——`bun run check:theme`
   已串联 parity 与色彩/刻度/utility 类名共四个检查脚本，CI 漂移即拦截；改字号/圆角 token
   必须两端同步。
6. **徽标与标签（Badge / Pill）居中与排版契约**：
   - 必须同时声明 `inline-flex items-center justify-center leading-none`，杜绝未设行高导致继承全局行高（24px）产生“文字下沉贴底”或在近正方形上施加 `rounded-full` 产生“怪异胖椭圆”；
   - 紧凑型徽标（未读数气泡、状态胶囊、卡片角标）：`h-5 px-2` 或 `h-5 px-2.5`，字号 `text-xxs`；
   - 标准型标签（封面分类胶囊、个人兴趣标签、大图页码）：`h-6 px-2.5`，字号 `text-xs font-medium`；
   - 表单筛选胶囊（`TagPill`）：`sm` 为 `h-6 px-2.5 text-xxs`，`md` 为 `h-7 px-3 text-xs`。

> 具体状态契约见 skill `hotspot-frontend-style`（样式自查清单 → 交互状态契约）。

> **UI 组件唯一目录**：`docs/ui-components.md`（原语清单、统一交互契约、
> 近重复收敛 backlog）。新增任何 UI 前先查该清单；一个交互意图只允许一种
> 反馈实现，新需求通过扩展原语 prop 表达，不新建近似组件。任何 UI 改动必须
> 截图对比审查（改前/改后、双主题、受影响断面），不允许只靠代码 diff 判断视觉结果。

## 路由与目录组织

**完整路由表与组件放置规则见 skill hotspot-frontend-routing**。要点速记：

- SvelteKit 文件路由 + adapter-static SPA（ssr=false），禁止引入 SSR 相关代码。
- 设备分目录：+page.svelte 分派，设备专属组件放 desktop/ mobile/，共享逻辑放 shared/。
- 路由常量见 @/lib/routes；每个 +page.svelte 必须有 svelte:head title。

## 页面布局

**完整契约见 skill hotspot-frontend-layout**（两种布局组件的选择与结构、PageHeader/TabBar/滚动、
三端分派、页面纵向间距与已登记例外）。要点速记：

- 普通页用 StandardPageLayout，分栏页用 SplitPageLayout；禁止自行拼装页头/滚动/横向 padding。
- 权威样板页：src/routes/(app)/nearby/moments/+page.svelte。
- Tab 一律 TabBar；列表一律 ListContainer + ListItem；空/加载/失败/到底四态各司其职。
- 页面纵向间距收敛到样板约定值，禁止散乱 py-* 值。

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

## 主题与 Token 架构

**完整契约见 skill hotspot-frontend-theme**（三层 token 架构、配色硬约束、主题文件、字体自托管）。要点速记：

- 配色一律来自 design token，禁止页内 hex/rgba/内置调色类；新增 token 需在主题文件补定义并注册 Tailwind 类。
- 跨端色彩与刻度（字号/圆角）必须与 harmonyapp 同步，bun run check:theme 漂移即拦截。
- 禁止在 CSS 里 @import 第三方样式表（阻塞渲染）；字体自托管，中文字形走系统字体栈。

## 样式与交互状态

**完整契约见 skill hotspot-frontend-style**（样式自查清单 19 条、交互状态契约、按钮 hover 两档、
禁止双重边框、token 类名有效性自查）。要点速记：

- 风格值一律落在 design token 体系，页面只消费语义类，禁止内联硬编码风格值（布局几何 px 除外）。
- 字号 9 档（4 档自定义 token + Tailwind 默认 5 档）/ 圆角 4 档 + full / 阴影 3+2+3 档，禁止近邻档位并存与 arbitrary 值。
- 交互反馈用既有工具类（hover-btn / scale-btn / tap-* / card-hover 等），不新发明交互类。
- 焦点必须可见（focus-ring 或等效），禁止无替代地 outline-none。
- **任何 UI 改动必须截图对比审查**（改前/改后、双主题、受影响断面），不允许只靠代码 diff 判断。
