# HotSpot 前端（React / Next.js / TypeScript）

> 本文件是前端专属约束。根 `AGENTS.md` 中的通用原则（Git 规范、Commit 格式、Docker 等）仍然适用。

## 功能边界

React Web **主前端**。认证/注册/发现与搜索/信息流/生活瞬间与评论/群组/聊天/附近/用户页面/个人中心/地图/管理后台。用户管理、支付与交易不在范围。

API 契约见 `spec/api_spec_*.md`（`spec/` 目录只读权威）。

## 技术栈

- **框架**：Next.js 16（App Router + Turbopack + `output: 'export'` 静态导出）
- **样式**：Tailwind CSS v4 + CSS Token 三层架构
- **包管理**：bun（`bun install` / `bun run`）
- **Mock**：MSW（仅 dev 模式）

## 开发模式：必须通过 Docker Compose + Nginx 网关

前端是纯 UI 层，所有 API、WebSocket、S3 请求都依赖后端服务。**禁止**直接在宿主机启动 `next dev`。

### 正确流程

```bash
# 1. 启动全部服务（含前端 Next.js dev server + 后端 + DB + nginx）
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d

# 2. 唯一访问入口
open http://localhost:8888
```

Nginx 网关（`docker/nginx.dev.conf`）统一反代：
| 路径 | 上游 |
|------|------|
| `/` | Next.js dev server（容器内 `frontend:5173`，不暴露到宿主机） |
| `/api/` | Go 后端（容器内 `backend:8081`） |
| `/ws/` | Go 后端 WebSocket |
| `/s3/` | SeaweedFS S3 |

### 宿主机上允许的调试操作

- 类型检查：`bun run check`
- 构建验证：`bun run build`

> ⚠️ **`bun run build` 会清空共享的 `out/` 目录（distDir），打断正在运行的 Next.js dev server**（dev 产物在 `out/dev/`）。需要跑 build 前先停掉 frontend 容器（`docker compose stop frontend`），跑完再 `docker compose start frontend`。

## 构建

```bash
bun install --cwd frontend
bun run --cwd frontend build
```

## ⚠️ 硬约束

- **调试/开发/预览时，默认入口是 `http://localhost:8888`（nginx 网关），不是 Next.js dev server 地址**
- 修改 nginx 配置后必须重启对应容器

## 路由系统

基于 Next.js App Router 文件路由：

> ⚠️ **静态导出约束仅构建期生效**：`next.config.mjs` 中 `output: 'export'` 仅在 `NODE_ENV=production`（即 `next build`）时启用。若 dev 也启用，Next.js dev server 只允许访问 `generateStaticParams()` 枚举的路径，所有动态详情页（`/moments/:id`、`/users/:id` 等）都会 500 "missing param"。动态路由的 `generateStaticParams()`（如返回 `[{ postId: '_' }]` 占位）仅服务于静态导出壳页，详情内容由客户端渲染。

```
src/app/
  layout.tsx               # 根布局（Providers + HTML shell）
  (auth)/login/page.tsx    # 登录页（无 MainLayout 外壳）
  (app)/
    layout.tsx             # MainLayout 外壳
    page.tsx               # 首页（附近动态）
    nearby/people/         # 附近的人
    nearby/groups/         # 附近的组
    nearby/moments/        # 附近时刻
    moments/[postId]/      # 时刻详情
    messages/              # 消息列表
    messages/[chatId]/     # 聊天详情
    me/                    # 个人主页
    me/[section]/          # 个人主页分段
    me/edit/               # 编辑资料
    users/[userId]/        # 用户详情
    groups/[groupId]/      # 群组详情
    settings/              # 设置
    map/                   # 地图
    notify/                # 通知
    search/                # 搜索
```

路由常量见 `@/lib/routes`。

页面实现与路由同目录 colocate：每个路由目录下，除 `page.tsx`（路由入口，仅做引用）外，同目录存放页面组件及其私有文件（如 `map/LocationPage.tsx` + `LocationDesktop.tsx` + `location.data.ts`）。跨路由复用的页面组件（如 `moments/MomentsFeedPage` 同时被首页和 `nearby/moments` 使用）放在所属域路由目录，由其他路由用相对路径引用；跨路由复用的共享组件（如 `CreateGroupSheet`）提升到 `components/`。

## 目录结构

```
src/
  app/            # Next.js App Router（路由 + 布局 + 页面实现）
    (auth)/       # 认证路由组（无外壳）
    (app)/        # 主路由组（MainLayout 外壳）
  components/     # 共享组件
    layout/       # 布局组件
    shared/       # 通用组件（Avatar, ActionIconButton 等）
    auth/         # 认证组件
    groups/       # 群组共享组件（CreateGroupSheet 等）
  hooks/          # 共享 hooks
  contexts/       # React Context（Theme, Auth, Platform, Composer）
  lib/            # 工具函数 + 路由常量
  api/            # API 客户端层
  styles/         # CSS（主题、Tailwind）
  assets/         # 静态资源
  mocks/          # MSW mock（仅 dev）
```

## 主题 & CSS Token 架构

`src/styles/theme/` 目录按三层架构组织：

```
theme/
  index.css              # 入口，按依赖顺序导入所有分片
  shadcn.css             # Layer 0: shadcn/ui 基础变量
  defaults.css           # Layer 0: 自定义 token 默认值（星际回退）
  primitives.css         # Layer 1: --pr-* 原始色值（每主题定义）
  semantic.css           # Layer 2: 语义 token（--text-primary 等 → --pr-*）
  compatibility.css      # Layer 2.5: 旧名别名（--neu-* → 语义名）
  theme-registry.css     # @theme inline Tailwind 注册
  themes/vibrant.css     # 星际主题覆盖
  themes/forest.css      # 清野主题覆盖
  layout.css             # 布局变量（context-rail、workspace）
  base.css               # @layer base（排版 + 按钮基线）
  utilities.css          # @layer utilities（组件工具类）
```

- Token 三层链：`--neu-*` → 兼容别名 → `--text-primary` 等语义名 → `--pr-*` 原始值
- 新组件优先使用语义名（`var(--text-primary)`、`var(--surface-page)` 等）+ Tailwind className
- 主题切换通过 `<html>` 上的 `.vibrant` / `.forest` + `.dark` 类实现
- 禁止使用内联 `style={{}}`，统一用 Tailwind className + CSS Token 变量
