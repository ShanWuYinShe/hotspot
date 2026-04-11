# CLAUDE.md

你是 HotSpot 项目的核心开发者。这份文件定义全局边界和工作流偏好，具体执行细节由你自主规划。

## 仓库地图
- `backend/`：Rust 主后端，Axum + PostGIS。
- `backend-go/`：Go 辅助后端，模块化单体 + S2 地理索引。
- `frontend_svelte/`：SvelteKit Web 前端，Svelte 5 + TypeScript。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。
- `deploy/`：部署与隔离验证脚本。

## 工作流偏好
我们追求 `main` 分支的线性历史和单功能单 Commit。

- **隔离开发**：所有修改在 `git worktree` 中完成，主目录仅用于同步和只读操作。
- **历史整理**：开发期间的多个临时 commit 在同步前通过 `git rebase -i` 压缩为**单独一个 commit**。
- **线性同步**：压缩后的单个 commit 通过 `git rebase` 或 `git merge --ff-only` 同步到主目录 `main` 分支。
- **Commit 规范**：最终 commit message 格式为 `<type>(<scope>): <subject>`，例如 `feat(backend-go/auth): add oauth login flow`。

## 质量标准
功能完成时，你已确保：
- 代码通过编译和 lint。
- **单元测试**（不依赖数据库/外部服务）在 worktree 中直接通过。
- **集成测试与数据库写入测试**已通过 `./deploy/scripts/verify.sh --env-name <name>` 在 Docker 隔离环境中验证通过。
- 无遗留调试代码或临时注释。

## 跨端铁律
- **距离与位置计算**：统一由后端完成，前端和客户端仅展示结果。

## 临时文件
- 所有临时文件、截图、中间输出写入 `tmp/` 目录。
