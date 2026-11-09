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
- 开发数据：dev 环境直连真实后端，**不内置任何 API mock**。可复现的测试数据由
  后端种子命令注入（Postgres + SeaweedFS 双写），见根 `AGENTS.md`「开发种子数据」与
  `backend-go/AGENTS.md`：`./scripts/seed-dev.sh`（幂等；`--reset` 重建）。
  测试账号 `qa@hotspot.local`（普通用户）与 `admin@hotspot.local`（管理员，同时是完整普通用户），
  口令经 `.env` 的 `SEED_QA_PASSWORD` / `SEED_ADMIN_PASSWORD` 注入，仅本地开发使用。

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

前置：dev 环境已启动（见上），宿主机已装好 `agent-browser` 包装器。
**不要执行 `bun install -g agent-browser`**：本机 `~/.bun/bin/agent-browser` 与 `~/.local/bin/agent-browser`
都是指向 `~/.agents/bin/agent-browser`（自建 wrapper，含 chrome-guard 防弹窗与 session 隔离）的软链，
重装会把它覆盖成上游原版，直接失去浏览器自动化红线。缺失时按 `browser-automation-safety` skill 处置。

```bash
# 桌面 21 页（QA 18 + 详情 3）
bun run browser:qa
# 桌面 21 页 + 移动 5 页（390×844）
bun run browser:qa:mobile
# 指定网关地址（如 worktree 独立环境 8889）
bash scripts/browser-qa.sh --base http://localhost:8889 --mobile
```

以种子账号真实登录（默认 `qa@hotspot.local`，seed 未注入时降级访客/未登录）自动巡检：横向溢出 / 坏图 / h1 数量 / TabBar 数量 / 控制台错误，
逐页截图并生成报告。输出统一在仓库根 `.workwork/browser-check/qa-<时间戳>/`
（已 gitignore，不提交）。页面打开/评估失败时脚本以非零退出（报告仍会落盘），可直接作为发布前门禁。

沙盒/受限环境两个坑：
1. 脚本按 `command -v agent-browser` 找二进制，沙盒内可能解析到绕过 wrapper 的裸
   二进制（socket 目录不可写直接全 FAIL）——用 `AB=~/.local/bin/agent-browser`
   前缀显式指到 wrapper；被拦后按现行方案对该命令申请 `danger-full-access`
   （提权原生启动；插件有意不放行 `~/.agent-browser`）；
2. qa 会话是独立 `--session` 实例、自带独立临时 profile，与既有
   会话无 profile 冲突；`close --all` 属红线禁止（会关掉全机所有会话，wrapper 会直接
   拦截并给出正确指引）。浏览器起不来时先 `agent-browser session list` 复查残留，
   再用 `node ~/.agents/bin/ab-reap-orphans.mjs --min-age 0` 清孤儿（先确认无活任务），
   而不是关别人的会话。

`bun run build` 是 `write-version-info`（注入版本信息）+ `vite build`（不含
`svelte-kit sync`；重建 `.svelte-kit/` 的是
`dev`/`check` 脚本里的 `svelte-kit sync`），但会读取并写入 `.svelte-kit/` 产物，
与正在运行的 dev server 共用该目录，可能相互干扰。需要跑 build 前先停掉 frontend
容器（`docker compose stop frontend`），跑完再 `docker compose start frontend`。


### 质量门禁脚本（需 dev 环境 + agent-browser）

```bash
# axe 可访问性：全部桌面页（QA 18 + 审计扩展 17 + 详情页 3，详情取不到则 35）
bun run audit:a11y
# 追加 390×844 移动 5 页（可与 --reduced 组合）
bun run audit:a11y -- --mobile --reduced
# 390 触控热区：默认全量页面（QA 18 + 审计扩展 17 + 详情），对照 allowlist，未登记离群即失败
bun run audit:touch
# 主题矩阵专项（3 主题×2 明暗×桌面全量+移动 5 页；合并前专项门禁，不进 audit:ui）
bun run audit:a11y -- --matrix --parallel 2        # 约 15–20min（单 worker 25–30min）
# 矩阵快查：--only 路由子串、--limit 每页组前 n 页
bun run audit:a11y -- --matrix --only /admin/reports,/me/edit --limit 2
# 一键：browser-qa --mobile + a11y + touch（任一失败整体 exit 1，末尾打印各报告路径）
bun run audit:ui
# 指定网关（worktree 独立环境）与调试抽样
bun run audit:a11y -- --base http://localhost:8898
bun run audit:touch -- --base http://localhost:8898 --limit 5
```

- 依赖：axe-core 为 devDependency（`bun install` 后从 `node_modules/axe-core` 注入），不进 build/check；
- 限流：dev 后端默认 `RATE_LIMIT_BYPASS=1`（见 `docker-compose.dev.yml`），QA/axe/触控审计与并发浏览器会话不会被 429 打断；要验证限流行为见 `backend-go/AGENTS.md`「dev 环境限流默认关闭」；
- 登录：QA 脚本用种子账号真实登录（`QA_LOGIN_ID` / `QA_PASSWORD`，默认 `qa@hotspot.local`，见 `.env.example`）；seed 未注入或登录失败时降级为「未登录态扫描」并提示先跑 `./scripts/seed-dev.sh`；
- 输出：仓库根 `.workwork/browser-check/{a11y,touch,qa}-<时间戳>/`（report.json + summary.txt + raw/），可重复运行；
- 会话：脚本使用独立 `hotspot-audit-*` session 并在结束时关闭，不干扰其它浏览器会话；
- allowlist 维护：`scripts/ui-quality-touch-allowlist.json`（稳定 selector + 理由，禁止 nth-child）；新增例外必须写明理由并在 `docs/frontend-ui-audit.md` 记录，详见该文「质量门禁」节；
- `--limit <n>` 仅用于调试抽样，正式门禁不要带；`audit:touch -- --baseline` 可只跑 r5 基线 22 页做快速对照。

## 硬约束

- 调试/开发/预览时，默认入口是 `http://localhost:8888`（nginx 网关），不是 dev server 地址
- 修改 nginx 配置后必须重启对应容器
- Svelte 组件互引必须写显式 `.svelte` 扩展名（默认导出）；Svelte 5 单文件单组件
- 新增依赖后**必须重建容器**：frontend 的 `entrypoint` 只在**启动时**跑 `bun install --frozen-lockfile`，运行中不会检测依赖变化——`docker compose up -d --force-recreate frontend`（验证方式：核对 `docker-compose.dev.yml` 第 166-169 行）
- 每个路由 `+page.svelte` 必须设置 `<svelte:head><title>`（格式 `页面名 · HotSpot`）；
  `(app)` 组外层已提供 skip link（`#main-content`），路由内容滚动区
  `RoutedContentViewport` 的 `<main id="main-content">` 不得移除

## UI 设计原则（硬性要求）

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
   登记例外：布局几何（width/height/position/gap 的 px）属于非主题风格，
   不受此条约束。
5. **跨端色彩与刻度对齐**：harmonyapp `ThemeConstants.ets` 的 ColorConfig 与本项目
   `themes/*.css` 的 `--cst-palette-*` 是同一套色板的两端表达，改色必须两侧
   同步。校验脚本、覆盖范围与同步流程的**唯一权威描述**见根 `docs/design-system.md`
   第 14 节与 skill `hotspot-harmony-ui`「跨端同步机制」——`bun run check:theme`
   已串联 parity 与色彩/刻度/utility 类名共四个检查脚本，CI 漂移即拦截；改字号/圆角 token
   必须两端同步。
6. **徽标与标签（Badge / Pill）居中与排版契约**：
   - 必须同时声明 `inline-flex items-center justify-center leading-none`，杜绝未设行高导致继承全局行高（24px）产生“文字下沉贴底”或在近正方形上施加 `rounded-full` 产生“怪异胖椭圆”；
   - 紧凑型徽标（未读数气泡、状态胶囊、卡片角标）：`h-5 px-2` 或 `h-5 px-2.5`，字号 `text-xxs`；
   - 标准型标签（封面分类胶囊、个人兴趣标签、大图页码）：`h-6 px-2.5`，字号 `text-xs font-medium`；
   - 表单筛选胶囊（`TagPill`）：`sm` 为 `h-6 px-2.5 text-xxs`，`md` 为 `h-7 px-3 text-xs`。
   - **实现必须用原语**：非压图徽章/标签一律 `data-display/Badge`（`size="sm"`=20px / `"md"`=24px，`tone` 枚举）；压图角标用 `.media-badge`；数字未读用 `shared/UnreadBadge`（`variant="accent|inverse"`）。禁止手写 `inline-flex h-5/h-6 rounded-full px-*` 胶囊或旧 `.badge` 类（检查器 `raw-badge` 拦截）。

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
- **hub/列表页页头统一 `PageHeader variant="integrated"`**：桌面/平板无背景/边框/吸顶（移动端例外吸顶，85% 画布底 + blur），标题+副标题/计数+工具同一行，筛选/操作 below 紧随（组件统一间距），正文直接开始；**禁止正文再重复页面标题/分区头**。适用 /nearby/people、/nearby/groups、/nearby/moments、/me/relationships、/me/reports、/notify；搜索页桌面端不渲染页头（输入框/筛选中枢由灵动岛承担），移动端 integrated 例外吸顶保住输入入口；设置/表单、聊天分栏、品牌首页不套用。**/nearby 枢纽页是登记例外：使用默认 sticky 变体（与首页对齐，顶部保留常驻搜索/地图工具；桌面端整条隐藏，地图入口由左侧栏 `LocationCard` 承担，不重复进岛）**。其余页面用默认变体（sticky）。
- **页面锚点岛（Page Island，仅桌面端）**：桌面端标准页头整条隐藏（`PageHeader` 默认 `desktopVisible=false` → `lg:hidden`，`/messages` 左栏例外），标题/身份/meta 与 ≤2 页面行动由 52px 顶栏中央的**常驻胶囊**承担；右侧 18px 圆环显示阅读进度（滚动 owner 上报 0–100，每帧写 `stroke-dashoffset`、无 CSS 过渡，同一读数只此一处）。胶囊形态：常驻 34px、圆角 20px（渲染即半高胶囊），点击右侧箭头向下展开控制中枢（440/460px、圆角 24px；宽度/高度/圆角 220ms 连续 morph，禁止用 `rounded-full` 驱动展开）；hover/focus 只由外壳承载（实底 + accent 细边 + 抬升 + 轻缩放；标题区不画独立灰底，除展开 morph 外不改宽度）；`Esc`/点岛外空白收起。点击标题回锚点、点击身份进作者主页；页面工具（搜索/编辑/全部已读等 ≤2 个）放进 `islandTrailing` 或 `islandExpanded`；首页由 `HomeDesktop`、搜索页由 `search/+page.svelte` 自定义注册。唯一形态实现 `components/layout/page/IslandCapsule.svelte`（`PageIsland` 组合内容）；完整契约见 skill `hotspot-frontend-layout`「页面锚点岛」与 design-system §22。移动/平板不渲染岛、页头照常。
- Tab 一律 TabBar；列表一律 ListContainer + ListItem；空/加载/失败/到底四态各司其职。
- **内容型列表默认无限滚动**：尾部挂 `components/list/InfiniteSentinel`（首屏不足一屏自动补载；加载中 `InlineLoading`、到底 `ListEnd`），不再放手动「加载更多」按钮；只有管理后台等显式分页场景用 `LoadMoreButton`。
- 页面纵向间距收敛到样板约定值，禁止散乱 py-* 值。

## 目录结构

```
src/
  routes/         # SvelteKit 路由（+page.svelte / +layout.svelte + 页面私有组件按设备分子目录）
  components/     # 共享组件（按域分子目录：auth/ data-display/ forms/ home/ layout/ list/
                  #   messages/ moments/ navigation/ overlays/ profile/ rail/ shared/ ui/；
                  #   设备专属共享组件再按 {desktop,tablet,mobile}/ 分目录，如 layout/mobile/、
                  #   layout/desktop/；layout/ 域含 page/ 通用页面原语子目录——
                  #   StandardPageLayout/SplitPageLayout/PageHeader/ContentColumn/
                  #   PageViewport/RoutedContentViewport）
                  #   单路由私有组件留在该路由目录，跨路由复用的才下沉到 components/
                  #   （moments/ 是仍在使用的域目录，MomentCard 位于
                  #   components/moments/，被 /nearby/moments 与 ProfileMomentTile 引用）
  stores/         # runes 模块单例（auth/theme/platform/presence/ws/composer/toast/admin/mapSource/unread，.svelte.ts）
  lib/            # 工具函数 + 路由常量 + 数据工厂（.svelte.ts 响应式；ws/ 为 WebSocket
                  #   客户端封装 client.ts + types.ts）
  api/            # API 客户端层（零框架依赖：client.ts 为 facade，传输核心在
                  #   core.ts，按业务域的端点包装在 domains/*.ts——新增端点放对应域文件）
  styles/         # CSS（主题、Tailwind）
  assets/         # 静态资源（含地图图钉 SVG）
```
> `static/` 位于**项目根**（SvelteKit 约定，不在 `src/` 下）：静态资源原样拷贝进构建产物。

## 主题与 Token 架构

**完整契约见 skill hotspot-frontend-theme**（三层 token 架构、配色硬约束、主题文件、字体自托管）。要点速记：

- 配色一律来自 design token，禁止页内 hex/rgba/内置调色类；新增 token 需在主题文件补定义并注册 Tailwind 类。
- 跨端色彩与刻度（字号/圆角）必须与 harmonyapp 同步，bun run check:theme 漂移即拦截。
- 禁止在 CSS 里 @import 第三方样式表（阻塞渲染）；字体自托管，中文字形走系统字体栈。

## 样式与交互状态

**完整契约见 skill hotspot-frontend-style**（样式自查清单、交互状态契约、档位表；档位数值以根 `docs/design-system.md` 为准）。要点一句话：风格值落 token 体系（布局几何 px 除外）、档位禁止近邻并存、用既有交互工具类、焦点必须可见、UI 改动必须截图对比审查（细则见「UI 设计原则」）。

- **输入族**：搜索一律 `forms/SearchInput`（胶囊 / 39px / text-md / 清空钮 44 命中）；单行一律 `forms/Input`（`density="default"` 43px / `"compact"` 39px）；多行一律 `forms/Textarea`（统一 `px-3.5 py-3` 壳）；下拉一律 `forms/Select`（`field`/`ghost`）；复选一律 `forms/Checkbox`。页面不得手写 input/textarea/select/checkbox/`.input-surface`，不得传 padding/圆角/字号覆盖原语（例外只能登记 allowlist 并写理由）。
- **同类型一致性（强制）**：任何同类型 UI（搜索/输入/文本域/下拉/复选/Tab/按钮/卡片/列表行/空态…）在所有页面必须走同一原语或同一登记类契约，上下文差异只允许通过原语变体表达，禁止页内近似实现。输入族由 `bun run check:ui-consistency`（scripts/check-component-consistency.mjs，已挂入 `bun run check`）源码级拦截；契约与例外清单见 `docs/ui-components.md`「同类型一致性契约」，表面状态详见同文「输入族契约」与 `src/styles/__tests__/input-surface-contract.test.ts`。
- **卡片行内交互**：作者行内所有「看起来可点」的元素必须是真实 `button/a` + `pointer-events-auto` + `stopPropagation`；禁止不可点徽章伪装按钮（会穿透整卡覆盖层直接打开详情）。关注态统一 `ui/FollowButton size="chip"`（`aria-pressed` + 「取消/关注 + 名字」 + 乐观态 + toast），点击不得触发卡片 `onClick`；非交互徽章允许点击穿透。新增卡片必须过「行内控件逐一点击」审计：头像/昵称→主页、动作→只切换、评论→详情评论区、封面→详情、关注→取关不导航。
- **详情页（瞬间详情）**：顶部用 `DetailHeader`（`sticky top-0`，`scrollThreshold` 默认 48px（乘 uiScale）：起始为整行工具行，滚动过阈值后收成居中胶囊岛并常驻，不随内容滚走）——**不渲染 `PageHeader`/横栏**，这两点别混为一谈；图片是内容卡首屏：整幅出血、宽度撑满内容卡 + 高度按图片自身比例（`w-full h-auto`），**任何分辨率下左右不得出现留白/模糊铺底**（高图由页面滚动承载，禁止用 max-h 压成异形）；ambient matte 仅作加载失败兜底；禁止固定高度 + `object-cover` 裁切；**多图走画廊导航**（主图 + 底部 56px 横向缩略图条，点击内联切换主图，右下角 `N / M` 页码；点击主图进全屏查看器），禁止再堆 3 列大缩略图网格；**图片区只保留顶部圆角**（卡片 `overflow-hidden` 裁剪）、底部直角贴边（禁止悬浮圆角矩形式图片卡）；缩略图条容器需给选中 ring 留绘制空间（`-mx-1 px-1 py-1`），禁止被 `overflow-x-auto` 裁掉；评论输入器 **全端 `docked`（常驻贴底）**：由 `SplitPageLayout` 的 footer 渲染——桌面 `sticky bottom-0`（xl 与内容卡同宽、内部对齐内容轴）、移动端 `fixed` 并用 footerHeight 补内容内距；横向 padding + 顶部分割线 + 毛玻璃实底，消除悬浮漏光，随时可发评论；发送钮 `size-9`（36px）；移动端不叠加第二条底部动作条。契约见 `docs/ui-components.md`「详情页契约」。
