# CLAUDE.md

你是 HotSpot 项目的核心开发者。这份文件只定义全局边界、任务分级和协作入口；具体实现细节由你自主规划。

## 仓库地图
- `backend/`：Rust 主后端，Axum + PostGIS。
- `backend-go/`：Go 辅助后端，模块化单体 + S2 地理索引。
- `frontend_svelte/`：SvelteKit Web 前端，Svelte 5 + TypeScript。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。
- `deploy/`：部署与隔离验证脚本。

## 硬约束
- **隔离开发**：所有写操作、测试和提交都在 `git worktree` 中完成；默认 worktree 目录为 `.worktrees/`。主目录仅用于同步和只读操作。
- **契约权威**：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- **跨端边界**：距离与位置计算统一由后端完成，前端和客户端只展示结果。
- **临时文件**：所有临时文件、截图、中间输出写入 `tmp/` 目录。
- **Git 安全**：禁止破坏性 git 操作；同步到 `main` 前必须整理为单功能单 commit，并保持线性历史。
- **Commit 规范**：最终 commit message 格式为 `<type>(<scope>): <subject>`，例如 `feat(backend-go/auth): add oauth login flow`。

## 规则优先级
- 根 `CLAUDE.md` 定义仓库级硬约束、任务分级、验证边界与默认协作入口，优先级最高。
- 子项目 `CLAUDE.md` 只补充该子项目的技术约束、本地命令与实现边界，不得放宽根级硬约束。
- workflow skill 负责过程编排与阶段引导；若与根/子项目规则冲突，以更高层规则为准。

## 任务分级
- **L0**：问答、只读分析、查文档。
- **L1**：单子项目、影响面小、无契约或架构变化的局部改动；可在读完上下文后直接实现。
- **L2**：单模块内多点改动或多文件改动；先做简短计划，再实现。
- **L3**：新增功能、跨模块改动、契约变更、架构调整或跨端行为变化；先设计，再计划，再实现。

## 验证分级
- **开发完成**：完成与改动直接相关的格式化、lint、类型检查、编译和本地单元测试；不得遗留调试代码或临时注释。
- **准备合并**：在“开发完成”的基础上，如改动涉及数据库写入、迁移、跨服务联调或真实环境链路，再运行 `./deploy/scripts/verify.sh --env-name <name>` 进行隔离验证。
- 当前 `deploy/scripts/verify.sh` 的 Docker 隔离验证主要覆盖 `backend-go/` 与 `frontend_svelte/`。
- `backend/` 与 `harmonyapp/` 目前仍以各自子项目定义的本地校验为主；若后续新增对应隔离验证入口，再统一收敛到 deploy workflow。

## 协作入口
- 先阅读根 `CLAUDE.md`，进入子项目后再阅读对应目录下的 `CLAUDE.md`。
- 默认先理解上下文，再做最小必要改动；低风险任务直接实现，高风险任务再升级流程。
- `hotspot-workflow` 是仓库级 workflow 入口，用于 worktree 内实现、分级验证、必要时的整理收尾，以及独立的 cleanup 流程。
- 默认目标是达到“开发完成”或“准备合并”状态；只有在用户明确要求时，才进入合并回主线、删除临时 worktree / 分支等后续动作。
- 仅在任务类型命中时再使用相关 skill。以下既可能是仓库内提供的入口，也可能是平台内置能力；若仓库内未显式提供，以当前执行环境实际可用项为准：
  - `brainstorming`：新增功能、行为变化、范围不清、跨模块设计。
  - `writing-plans`：需要明确拆解步骤的中大型任务。
  - `systematic-debugging`：bug、失败日志、异常行为排查。
  - `verification-before-completion`：准备宣称完成、提交或开 PR 前。
