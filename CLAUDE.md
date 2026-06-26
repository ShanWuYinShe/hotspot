# CLAUDE.md

## 仓库地图
- `backend/`：Rust 后端（**已暂停**），Axum + PostGIS。仅保留代码，不做主动开发。
- `backend-go/`：Go **主后端**，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend/`：SvelteKit Web **主前端**，Vite + TypeScript + Tailwind CSS，当前所有前端开发在此进行.
- `frontend_react/`：React SPA 前端（**已暂停**），作为 UI/UX 设计参考保留，不作主前端推进。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。

## 项目特有约束
- **契约权威**：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- **跨端边界**：距离与位置计算统一由后端完成，前端和客户端只展示结果。
- **Docker Compose 统一管理**：所有服务（postgres、redis、seaweedfs、backend、frontend、nginx）统一通过 `docker compose` 编排启动/停止；不得混用宿主机进程与容器，也不允许单独启动某个服务绕过 compose。
