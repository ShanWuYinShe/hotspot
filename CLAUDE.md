# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 仓库概览
HotSpot 是一个多端 monorepo：
- `backend/`：Rust + Axum 主后端，负责大部分业务 API、WebSocket、事件总线和 PostGIS 地理查询。
- `backend-go/`：Go + chi 辅助后端，采用 modular-monolith 结构，地理能力统一走 S2。
- `frontend_svelte/`：SvelteKit Web 前端。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约和行为规范的权威来源。

## 先读什么
- 根 `CLAUDE.md` 只放跨项目流程、共享边界和通用规则。
- 子项目命令、局部架构和实现护栏，以各目录下的 `CLAUDE.md` 为准。
- 任何接口、字段、错误码或 WebSocket 变更前，先读 `spec/api_spec_index.md`、对应模块 spec，以及 `spec/api_spec_base.md`、`spec/api_spec_types.md`、`spec/api_spec_errors.md`。

## Claude Code 工作方式
- 小任务直接处理；多步骤任务先用 task list。
- 跨模块、跨子项目、存在多种实现路径或需要先摸清边界的任务，先进入 plan mode。
- 需求不清、实现分歧或碰到风险边界时，先用 AskUserQuestion 澄清。
- 只有在子任务彼此独立时才使用 subagents，避免多个 agent 改同一片核心代码。
- 长期稳定偏好放 memory；共享权限与自动化放 `.claude/settings.json`；个人机器差异放 `.claude/settings.local.json`。
- 不再使用旧式会话落盘、工具能力探测或强制暂存流程。

## 全局约束
- `spec/` 是唯一权威契约来源；除非用户明确要求，不要修改其中文件。
- 不要猜测 spec 中不存在的接口、字段、错误码或行为；实现与 spec 冲突时，优先修改实现而不是反改 spec。
- 临时文件统一放在仓库根目录 `tmp/`。
- 默认只在仓库内工作；确需访问仓库外路径时，只使用用户明确提供或仓库文档已约定的绝对路径。
- 标准本地 Docker 测试与 runtime 验证属于默认开发流程；清理数据、重置环境、正式部署、数据库重置和远程同步仍需先确认。

## 本地优先开发流程
1. 开始新一轮开发前，先确认上一轮成果已经同步回本地 `main`，并确保当前工作区 clean。
2. 默认在本地完成开发、测试、review 和集成；远程仓库只作为最终同步出口。
3. 在本地分支上工作；worktree 只在并行任务、长任务或需要隔离上下文时使用。
4. 每个本地分支先完成本地验证；多分支结果统一在本地集成分支汇总、review、补测。
5. 只有本地集成分支通过验证后，才由用户决定是否同步到远程 `main`；PR 不是默认流程。
6. 全部完成后，清理本轮使用到的本地分支和不再需要的 worktree。

## 风险边界
### 可直接执行
- 读取文件、搜索代码、编辑仓库内已有文件。
- 使用 task list、plan mode、AskUserQuestion、subagents 组织工作。
- 运行本地非破坏性的构建、类型检查、lint、单测和标准 Docker 验证。

### 需要先确认
- 大量文件删除、批量重命名、跨目录大改。
- 修改 `deploy/` 脚本、CI/CD、数据库 migration、共享 `.claude/settings.json`。
- `git fetch`、`git pull`、`git merge`、`git cherry-pick`、`git push`。
- 清理或重置 runtime / 数据库 / branch / worktree 的操作。

### 高风险
- `git reset --hard`、`git clean`、`git branch -D`、任何 `--force` 推送或历史改写。
- 删除仍有未合并改动的 branch / worktree。
- 数据库重置、批量清理 volume、破坏性 deploy / 运维动作。

### 绝对不要
- 未经用户明确要求修改 `spec/`。
- 在仓库里写入密钥、凭据、`.env` 内容或提交本地私有配置。
- 为了省事绕过安全边界、跳过确认、或编造未定义契约。

## 本地验证与交付入口
### 短版工作流
- `verify`：面向并行开发验证，支持多 agent + 多 worktree；默认验证当前工作目录 / worktree。用法：`./deploy/scripts/verify.sh --env-name <name>`。
- `preview`：面向最终候选结果预演，只有一个实例。先运行 `./deploy/scripts/preview-build.sh`，再运行 `./deploy/scripts/preview-up.sh`，最后用 `./deploy/scripts/preview-inspect.sh` 巡检。
- `release`：面向交付，只产部署输入物，不运行环境。用法：`./deploy/scripts/release-build.sh --ref <git-ref>`。

### 原则
- 只有 `verify` 支持多 env / 多 worktree 并行；`preview` 只有一个实例。
- `verify` 默认验证的是**当前工作目录 / worktree**，不是直接把当前目录作为 Docker build context。
- `preview` / `release` 面向候选结果与确定性版本，不再承担多 env 开发隔离语义。
- 运行与测试继续遵循：**同一套配置 key，不同 datasource value**。不要引入额外环境模式抽象或 Redis key-prefix 机制，除非用户明确要求。

## 跨端地图与坐标规则
- 距离、方位、附近排序始终由后端完成；前端和客户端只消费结果，不自行计算距离或角度。
- 是否需要坐标转换，取决于“坐标真实来源”和“地图 SDK 使用的坐标系”是否一致，而不是后端存储格式。
- marker、overlay、center pin 都必须使用地图 SDK 原生能力；不要叠加绝对定位 DOM 或自定义伪 marker。
- center pin 必须绑定地图相机中心；写回业务状态时直接使用地图 SDK 返回的中心坐标。
- 当确实存在坐标系差异时，只在业务坐标与地图坐标的单一边界层做一次转换，避免重复正反转换造成漂移。
