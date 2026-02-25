---
globs: harmonyapp/**
alwaysApply: false
---

# 鸿蒙客户端开发规则路由

编辑鸿蒙代码时，须同时遵守以下规范（按需阅读）：

- **实现规范**：`spec/harmonyapp_rule.md` -- ArkTS 编码约束、组件规范
- **功能边界**：`spec/harmonyapp_doc.md` -- 鸿蒙端功能范围与非目标
- **Agent 操作指南**：`harmonyapp/AGENTS.md` -- 目录结构、图标规范、编译命令

新增 API 端点时的文件修改清单：

- `entry/src/main/ets/core/config/ApiEndpoints.ets` -- 添加路径常量
- `entry/src/main/ets/core/api/ApiClient.ets` -- 添加 API 调用方法
- `entry/src/main/ets/core/types/api.ets` -- 添加请求/响应类型定义
- 如为幂等接口：确认路径已加入 HttpUtil 的 `MUST_IDEMPOTENT_ENDPOINTS` 列表
