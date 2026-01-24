# HotSpot React Frontend

基于 Next.js 15 + React 19 + shadcn/ui 的前端项目。

## 技术栈

| 类别 | 选型 | 说明 |
|------|------|------|
| 框架 | Next.js 15 | App Router 模式 |
| UI 库 | shadcn/ui + Tailwind CSS | 基于 Radix UI |
| 状态管理 | Zustand | 客户端状态 |
| 数据获取 | TanStack Query v5 | 服务端状态管理 |
| 国际化 | next-intl | 中英文支持 |
| 表单 | React Hook Form + Zod | 类型安全的表单验证 |
| 图标 | Lucide React | 与 shadcn/ui 配套 |

## 目录结构

```
frontend_react/
├── src/
│   ├── app/                    # Next.js App Router 路由
│   │   ├── (auth)/             # 认证相关页面组
│   │   ├── (main)/             # 主应用页面组
│   │   └── (admin)/            # 管理后台页面组
│   ├── components/             # UI 组件
│   │   ├── ui/                 # shadcn/ui 基础组件
│   │   ├── common/             # 通用业务组件
│   │   ├── cards/              # 卡片组件
│   │   └── forms/              # 表单组件
│   ├── hooks/                  # 自定义 Hooks
│   ├── lib/                    # 工具库
│   │   ├── api/                # API 客户端和服务层
│   │   ├── websocket/          # WebSocket 管理
│   │   └── utils/              # 工具函数
│   ├── stores/                 # Zustand stores
│   ├── types/                  # TypeScript 类型定义
│   └── i18n/                   # 国际化配置
├── messages/                   # i18n 翻译文件
│   ├── en.json
│   └── zh.json
└── public/                     # 静态资源
```

## 开发指南

### 安装依赖

```bash
pnpm install
```

### 启动开发服务器

```bash
pnpm dev
```

### 构建生产版本

```bash
pnpm build
```

### 代码检查

```bash
pnpm lint
```

## 环境变量

创建 `.env.local` 文件：

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
```

## 规范文档

- 功能说明：`spec/frontend_doc.mdc`
- 工程规范：`spec/frontend_rule.mdc`
- API 契约：`spec/api_spec.mdc`
