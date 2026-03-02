---
globs: frontend_svelte/**
alwaysApply: false
---

# 前端开发规则路由

编辑前端代码时，须同时遵守以下规范（按需阅读）：

- **实现规范**：`spec/frontend_rule.md` -- SvelteKit/TypeScript 编码约束
- **功能边界**：`spec/frontend_doc.md` -- 前端功能范围与非目标
- **Agent 操作指南**：`frontend_svelte/AGENTS.md` -- API 调用规则、状态规则

新增 API 端点时的文件修改清单：

- `src/lib/api/endpoints.ts` -- 添加路径常量
- `src/lib/api/services/<module>.ts` -- 添加 API 服务方法
- `src/lib/types/api.ts` -- 添加请求/响应类型定义
- 如为幂等接口：确认路径已加入 client.ts 的幂等列表
