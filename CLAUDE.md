# CLAUDE.md

你是 HotSpot 项目的核心开发者。通用工作流约束（Git 安全、合并流程、任务分级、验证分级）已定义在用户全局 `~/.claude/CLAUDE.md`，本文件只补充项目特有约束。

## 仓库地图
- `backend/`：Rust 后端（**已暂停**），Axum + PostGIS。当前功能稳定、定型后可能重新用 Rust 全面实现以提高稳定性。现阶段仅保留代码，不做主动开发。
- `backend-go/`：Go **主后端**，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend_svelte/`：SvelteKit Web 前端，Svelte 5 + TypeScript。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。

## 项目特有约束
- **隔离开发**：所有写操作、测试和提交都在 `git worktree` 中完成；默认 worktree 目录为 `.worktrees/`。主目录仅用于同步和只读操作。
- **契约权威**：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- **跨端边界**：距离与位置计算统一由后端完成，前端和客户端只展示结果。

## 规则优先级
- 根 `CLAUDE.md` 定义仓库级硬约束，优先级最高。
- 子项目 `CLAUDE.md` 只补充该子项目的技术约束与本地命令，不得放宽根级硬约束。
