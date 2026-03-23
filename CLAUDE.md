# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 仓库概览
HotSpot 是一个多端 monorepo：
- `backend/`：Rust + Axum 主后端，负责大部分业务 API、WebSocket、事件总线和 PostGIS 地理查询。
- `backend-go/`：Go + chi 辅助后端，采用 modular-monolith 结构，地理能力统一走 S2。
- `frontend_svelte/`：SvelteKit Web 前端。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约和行为规范的权威来源。
- `deploy/`：本地验证、候选预演与交付输入物的统一入口。

## 阅读顺序
- 先读根 `CLAUDE.md`，了解跨项目流程、共享边界和风险边界。
- 再读对应子项目 `CLAUDE.md`，了解局部架构、实现护栏和本地调试提示。
- 任何接口、字段、错误码、分页、幂等或 WebSocket 变更前，先读 `spec/api_spec_index.md`、对应模块 spec，以及 `spec/api_spec_base.md`、`spec/api_spec_types.md`、`spec/api_spec_errors.md`。

## 全局权威与共享边界
- `spec/` 是唯一权威契约来源；除非用户明确要求，不要修改其中文件。
- 不要猜测 spec 中不存在的接口、字段、错误码或行为；实现与 spec 冲突时，优先修改实现而不是反改 spec。
- 默认只在仓库内工作；确需访问仓库外路径时，只使用用户明确提供或仓库文档已约定的绝对路径。
- 临时文件统一放在仓库根目录 `tmp/`。
- 共享自动化、hooks 与团队级权限放 `.claude/settings.json`；个人机器差异放 `.claude/settings.local.json`。
- 所有测试、check、build 的最终结论默认都以 `deploy/` 下的 Docker 流程为准；宿主机直接运行的命令只作为临时调试参考。

## 主目录与 worktree 模型
- 主目录默认始终保持在本地 `main` 且 clean；主目录只承担同步、审阅、创建/回收 worktree 和本地集成控制，不承载实现性改动。
- 所有实现、验证和构建都在 worktree 中完成；需要本地分支时，也在对应 worktree 中使用。
- 复杂任务先规划；只有当子任务边界清晰且文件集相对独立时才并行。
- 每个 feature worktree 先独立完成实现、自检和验证；多分支结果统一汇总到专门的 integration worktree。
- integration worktree 只负责汇总、冲突处理、补测、预演与交付准备，不承载新的大规模实现。
- 本地集成结果通过后，再由用户决定是否同步到远程 `main`；PR 不是默认流程。
- 本轮完成后，删除已完成的 feature worktree / branch 与 integration worktree，让主目录继续保持 `main + clean`。

## 质量门顺序
- 默认顺序：`plan -> task list -> implement -> verify -> simplify -> strict review -> commit -> integrate -> verify -> preview/release -> cleanup`
- `simplify` 负责实现后的代码收敛与重复清理；严格 reviewer 负责合并前门槛，不固定绑定某一个第三方 skill。
- integration 后必须再做一轮 `verify`，不能把各 feature worktree 的局部通过当成集成通过。

## 本地验证与交付入口
### 短版工作流
- `verify`：面向并行开发验证，支持多 agent + 多 worktree；默认在当前 worktree 上下文验证，并且在 Docker 隔离环境里完成 backend-go/frontend 的测试与构建检查。用法：`./deploy/scripts/verify.sh --env-name <name>`。
- `preview`：面向最终候选结果预演，只有一个实例。先运行 `./deploy/scripts/preview-build.sh`，再运行 `./deploy/scripts/preview-up.sh`，最后用 `./deploy/scripts/preview-inspect.sh` 巡检。
- `release`：面向交付，只产部署输入物，不运行环境。用法：`./deploy/scripts/release-build.sh --ref <git-ref>`。

### 原则
- 只有 `verify` 支持多 env / 多 worktree 并行；`preview` 只有一个实例。
- `verify` 默认验证的是当前工作目录 / worktree，但执行发生在 Docker 隔离环境中，不应把宿主机运行结果视为最终验证结论。
- 测试过程中产生的缓存、store、构建产物必须落在 `tmp/` 或容器临时目录，不能把 `.pnpm-store` 之类的临时目录写回仓库根。
- `preview` / `release` 面向候选结果与确定性版本，不再承担多 env 开发隔离语义。
- 运行与测试继续遵循：同一套配置 key，不同 datasource value。不要引入额外环境模式抽象或 Redis key-prefix 机制，除非用户明确要求。

## 风险边界
### 可直接执行
- 读取文件、搜索代码、编辑仓库内已有文件。
- 在专用 worktree 中完成实现、验证、代码收敛与 review。
- 运行本地非破坏性的标准 `deploy/` 验证流程。

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

## 跨端共享业务原则
- 距离、方位、附近排序始终由后端完成；前端和客户端只消费结果，不自行计算距离或角度。
- 是否需要坐标转换，取决于坐标真实来源与地图 SDK 使用的坐标系是否一致，而不是后端存储格式。
- 当确实存在坐标系差异时，只在业务坐标与地图坐标的单一边界层做一次转换，避免重复正反转换造成漂移。

## 子项目入口
- Go 后端规则：`backend-go/CLAUDE.md`
- Rust 后端规则：`backend/CLAUDE.md`
- Web 前端规则：`frontend_svelte/CLAUDE.md`
- Harmony 客户端规则：`harmonyapp/CLAUDE.md`
- 契约规则：`spec/CLAUDE.md`
