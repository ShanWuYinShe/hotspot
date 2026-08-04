# HotSpot 前端（SvelteKit / Svelte 5 / TypeScript）

本文件是前端专属约束。根 `AGENTS.md` 中的通用原则（Git 规范、Commit 格式、Docker 等）仍然适用。

## 功能边界

Svelte Web 主前端。认证/注册/发现与搜索/信息流/生活瞬间与评论/群组/聊天/附近/用户页面/个人中心/地图/管理后台。用户管理、支付与交易不在范围。

API 契约见 `spec/api_spec_*.md`（`spec/` 目录只读权威）。

## 技术栈

- 框架：SvelteKit 2 + Svelte 5（runes）+ `adapter-static` SPA 模式
  （`ssr=false` + `fallback: 'index.html'`，nginx `try_files` 回退；全部页面客户端渲染）
- 样式：Tailwind CSS v4 + CSS Token 三层架构（`neu-hover-*` CSS 类承载交互反馈）
- UI 组件：bits-ui 生态组件（`components/ui/` 封装层：dialog/sheet 定位变体、dropdown-menu、popover）+ 页面级自研视觉组件
- 包管理：bun（`bun install` / `bun run`）
- 测试：vitest + @testing-library/svelte（宿主机 `bun run test`）
- Mock：MSW（仅 dev 模式）

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

`bun run build` 会重建 `.svelte-kit/`，可能打断正在运行的 dev server。需要跑 build 前先停掉 frontend 容器（`docker compose stop frontend`），跑完再 `docker compose start frontend`。

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
  +layout.svelte           # 全局壳（styles + stores + AuthGate + MSW 挂载）
  +error.svelte            # 404
  (auth)/login/+page.svelte    # 登录页（无 MainLayout 外壳）
  (app)/
    +layout.svelte         # MainLayout 外壳（桌面/平板/移动/横屏四端分派）
    +page.svelte           # 首页（附近动态）
    discover/              # 发现聚合页（平板/移动端顶级导航）
    nearby/people/         # 附近的人
    nearby/groups/         # 附近的组
    nearby/moments/        # 附近时刻（页面样板结构权威基准）
    moments/[momentId]/    # 时刻详情
    messages/              # 消息列表
    messages/[chatId]/     # 聊天详情（chatId 实为对端 user_id）
    messages/group/[groupId]/  # 群聊详情
    me/                    # 个人主页
    me/[section]/          # 个人主页分段（5 独立分区 + relationships 聚合页）
    me/edit/               # 编辑资料
    users/[userId]/        # 用户详情
    groups/[groupId]/      # 群组详情
    events/[eventId]/      # 活动详情
    settings/              # 设置
    map/                   # 地图
    notifications/         # 通知
    notify/                # 旧路径 307 重定向到 /notifications
    search/                # 搜索
```

> 注：管理后台（spec `api_spec_admin_core.md` / `api_spec_admin_moderation.md`）仅有后端契约，**前端路由尚未实现**——功能边界中的「管理后台」为待建项，勿误判为遗漏。

路由常量见 `@/lib/routes`。

页面实现与路由同目录 colocate：每个路由目录下，除 `+page.svelte` 外，同目录存放页面私有组件（如 `map/LocationDesktop.svelte` + `location-data.ts`）。跨路由复用的页面组件（如 `components/profile/SectionFriends.svelte` 同时被 `me/[section]` 分区页与 `me/relationships` 聚合页复用）放在 `components/` 对应域目录；共享组件统一在 `components/`。

## 页面样板结构（所有页面必须遵守）

**权威样板页：`src/routes/(app)/nearby/moments/+page.svelte`。** 任何新增路由页面
（含三端差异）都以它为基准，结构固定为：

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

1. **布局**：页面必须用 `StandardPageLayout`，header 以 snippet 传入、正文放 children；
   不得自行组合 PageHeader/ContentColumn/页面级滚动/页面级横向 padding（契约见
   StandardPageLayout.svelte 注释）。gutter 一律由 `ContentColumn` 的
   `px-[var(--page-gutter)]` 管理。
2. **页头**：一律用 `PageHeader`（`sticky` 默认开），`onBack` 在非桌面自动显示返回
   按钮；`below` 只放搜索框与 Tab 栏等 header 附属区。
3. **分类 Tab 栏**：下划线样式（容器 `border-bottom` + 激活项底部 2px 指示条 +
   `aria-pressed`），放 `below` snippet 内随 header 吸顶；tab 状态可存 URL query
   （如 `?tab=`）以支持深链，切换用 `goto` 更新 URL。
4. **正文与滚动**：正文直接放 children；页面滚动由 `RoutedContentViewport`（main）
   唯一承担，页面不得自建滚动容器（消息分栏、地图等全画布页面除外）。内嵌自带
   gutter 的组件时用负 margin（`-mx-[var(--page-gutter)]`）抵消 ContentColumn
   gutter，避免双重内边距（参照 `SectionRelationships.svelte`）。
5. **三端差异**：共用组件只写一次，设备差异用 `platform.platform` 派生（列数、字号、
   布局分支），不得复制三份页面。
6. **数据获取与状态**：`$effect` + `cancelled` 竞态清理模式（参照样板页）；加载中用
   spinner、空列表用 `EmptyState`（参照样板页 feedState snippet）。

## 目录结构

```
src/
  routes/         # SvelteKit 路由（+page.svelte / +layout.svelte + 页面私有组件）
  components/     # 共享组件（按域分子目录：auth/ data-display/ forms/ layout/ messages/ moments/ navigation/ overlays/ profile/ shared/ ui/）
  stores/         # runes 模块单例（auth/theme/platform/composer，.svelte.ts）
  lib/            # 工具函数 + 路由常量 + 数据工厂（.svelte.ts 响应式）
  api/            # API 客户端层（client.ts 零框架依赖）
  styles/         # CSS（主题、Tailwind）
  assets/         # 静态资源（含地图图钉 SVG）
  mocks/          # MSW mock（仅 dev）
  static/         # SvelteKit 静态目录（mockServiceWorker.js 等）
```

## 主题 & CSS Token 架构

`src/styles/theme/` 目录按三层架构组织：

```
theme/
  index.css              # 入口，按依赖顺序导入所有分片
  shadcn.css             # Layer 0: shadcn/ui 基础变量
  shared.css             # Layer 0: z-index/blur/shape/space/motion/ease（与主题无关）
  semantic.css           # Layer 1: 语义 token（--text-primary 等 → --pr-*）
  themes/vibrant.css     # Layer 2: 星际主题（全部 --pr-* 值）
  themes/forest.css      # Layer 2: 清野主题（全部 --pr-* 值）
  compatibility.css      # Layer 2.5: 旧名别名（--neu-* → 语义名）
  layout.css             # 布局变量（context-rail、workspace）
  base.css               # @layer base（排版 + 按钮基线）
  theme-registry.css     # @theme inline Tailwind 注册
  utilities.css          # @layer utilities（组件工具类 + neu-hover-* 交互类）
```

- Token 三层链：`--neu-*` → 兼容别名 → `--text-primary` 等语义名 → `--pr-*` 原始值
- 新组件优先使用语义名（`var(--text-primary)`、`var(--surface-page)` 等）+ Tailwind className
- 主题切换通过 `<html>` 上的 `.vibrant` / `.forest` + `.dark` 类实现
- 悬停/按压反馈统一用 `neu-hover-btn/action/row/card/bg` CSS 类（主题差异由类选择器承载），避免 JS 状态动画
