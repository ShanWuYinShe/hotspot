# CLAUDE.md

你是 HotSpot 项目的核心开发者。Git 规范、验证流程、工具链偏好等通用约束已定义在用户全局 `~/.claude/CLAUDE.md`，本文件只补充项目特有约束。

## 仓库地图
- `backend/`：Rust 后端（**已暂停**），Axum + PostGIS。当前功能稳定、定型后可能重新用 Rust 全面实现以提高稳定性。现阶段仅保留代码，不做主动开发。
- `backend-go/`：Go **主后端**，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend/`：React SPA 前端（**正在从 `frontend_svelte/` 迁移**），Vite + TypeScript + Tailwind CSS。
- `frontend_svelte/`：SvelteKit Web 前端（**迁移中**），待 `frontend/` 功能对齐后退役。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。

## 项目特有约束
- **隔离开发**：所有写操作、测试和提交都在 `git worktree` 中完成；默认 worktree 目录为 `.worktrees/`。主目录仅用于同步和只读操作。
- **契约权威**：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- **跨端边界**：距离与位置计算统一由后端完成，前端和客户端只展示结果。
- **Docker Compose 统一管理**：所有服务（postgres、redis、seaweedfs、backend、frontend、nginx）必须通过 `docker compose` 启动和停止。禁止混用本地进程与容器——要么全用 compose，要么全不用。测试联调时不允许单独启动某个服务绕过 compose。
- **禁止绕过限制**：如果某个命令被安全策略拦截（如 auto mode classifier）或环境不满足（如 Docker Hub 网络不可达），停下来向用户报告问题，不得尝试用其他等价命令、修改 Dockerfile、降级构建引擎等方式绕过。用户来决定如何解决。

## 规则优先级
- 根 `CLAUDE.md` 定义仓库级硬约束，优先级最高。
- 子项目 `CLAUDE.md` 只补充该子项目的技术约束与本地命令，不得放宽根级硬约束。
