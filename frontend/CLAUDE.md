# CLAUDE.md

本文件只定义本子项目的技术约束与本地命令；仓库级 workflow 以根 `CLAUDE.md` 为准。

> **注意**：本项目正在从 `frontend_svelte/` 迁移至此，使用 React + Vite + TypeScript + Tailwind CSS。

## 技术栈

- **UI 组件**：shadcn/ui（Radix 原语）+ Tailwind CSS 4
- **状态管理**：Zustand（`src/lib/stores/`）
- **路由**：react-router 7（懒加载见 `routes.tsx`）
- **地图**：react-leaflet

## 构建 & 测试

- `bun run dev` — 开发服务器
- `bun run build` — 构建

> **测试 / Lint / Format**：package.json 仅有 `dev` 和 `build` 脚本，test、lint、format 均未配置。迁移完成后再补充。

## 约束

- **API 路径**：统一从 `endpoints.ts`（或等价文件）引用。
- **HTTP 方法**：统一 POST。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **分页**：cursor 只透传，不解析、不构造。
- **契约对齐**：修改接口前查阅 `spec/`。
