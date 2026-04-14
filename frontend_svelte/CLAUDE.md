# CLAUDE.md

- 本文件只定义本子项目的技术约束与本地命令；仓库级 workflow、任务分级与验证边界以根 `CLAUDE.md` 为准。

## 构建 & 测试 & Lint
- `pnpm check` — 类型检查
- `pnpm build` — 构建
- `pnpm test` — 单元测试
- `pnpm lint` — lint
- `pnpm format` — 格式化

## 约束
- **API 路径**：统一从 `endpoints.ts` 引用。
- **请求封装**：所有请求走 `client.ts` 和 `services/*`。
- **HTTP 方法**：统一 POST。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **分页**：cursor 只透传，不解析、不构造。
- **认证**：使用 HttpOnly Cookie，不在前端存储或解析 Token。
- **TypeScript**：类型明确，避免 `any`。
- **地图交互**：距离和角度由后端计算，前端仅展示；坐标转换仅在"业务坐标 ↔ 地图坐标"单一边界层执行一次；marker、overlay、中心点使用地图 SDK 原生能力。
