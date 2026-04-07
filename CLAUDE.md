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

## 整体流程
本仓库的默认工作流是一个完整的本地闭环：
1. **任务描述**：用户在对话中描述任务。
2. **任务分析**：Claude Code 分析任务范围，根据复杂度和文件集决定是否拆分为多个子任务；子任务边界清晰且文件集相对独立时才并行，否则保持单任务。
3. **创建 worktree 并实现**：从主目录本地 `main` 当前 HEAD 创建 feature worktree（多子任务时每个子任务各创建一个），在 worktree 中完成全部实现、检查、测试和 strict review。
4. **线性同步回主目录**：
   - 单任务：在 feature worktree 中完成全部质量门后，通过 `rebase` 整理分支，以 fast-forward 方式同步回主目录本地 `main`。
   - 多子任务：新建专门的 integration worktree，将所有子任务成果通过 `rebase + squash` 收敛为单个 commit，在 integration worktree 中完成全部质量门后，再同步回主目录本地 `main`。

任何涉及代码修改的 worktree 都必须完成以下闭环才算任务完成：
`编写代码 → 检查（语法错误、编译错误、警告） → 测试 → simplify → strict review`
其中这里的 `review` 默认为 `strict review`；若闭环中任何环节发现问题，必须回到编写代码修复后重新走完全部环节，直到全部通过。

## 全局权威与共享边界
- `spec/` 是唯一权威契约来源；除非用户明确要求，不要修改其中文件。
- 不要猜测 spec 中不存在的接口、字段、错误码或行为；实现与 spec 冲突时，优先修改实现而不是反改 spec。
- 默认只在仓库内工作；确需访问仓库外路径时，只使用用户明确提供或仓库文档已约定的绝对路径。
- 临时文件统一放在仓库根目录 `tmp/`。
- 共享自动化、hooks 与团队级权限优先放 `.claude/settings.json`；若当前仓库仍存在只挂在 `.claude/settings.local.json` 的本机 hooks 或权限补充，必须在对应文档中明确说明这是本地差异，而不是共享默认行为。
- 权限策略采用“最小静态 allow + 运行期 hooks 判定”模型；命令边界由 `tool_guard.py`、`worktree_guard.py` 等受控脚本强制执行，不在 MD 文件中逐条列举。
- Git 控制面动作默认分步执行：不要把多个 git 状态变更命令用 `&&`、`;` 或 `||` 串成单条 Bash。即使步骤之间存在依赖，也应先执行前一步、读取结果、确认当前仓库状态，再进入下一步；“自动推进本地闭环”仅表示不必逐步向用户确认，不表示可以跳过中间观察。
- 主目录本地 `main` 只允许 git 控制面动作；git 控制面动作默认逐步执行，不要把 `rebase`、`merge`、`cherry-pick`、`worktree add/remove`、`branch -d/-D` 等多个状态变更命令串成一条 Bash。worktree 内的格式化、语法检查和编译命令由 `.claude/scripts/` 下的受控脚本封装，不允许直接调用原始开发命令。

## 主目录与 worktree 模型
- 主目录默认始终保持在本地 `main` 且无未提交改动；主目录只承担同步、审阅、创建/回收 worktree 和本地集成控制，不承载实现性改动。
- 任何 implement 行为开始前都必须先进入 feature worktree；主目录 `main` 只允许控制面动作（git 状态/历史检查、建/收 worktree、review、sync-back、cleanup）。
- 所有承载实现性改动的实现、验证和构建都在 feature / integration worktree 中完成；需要本地分支时，也在对应 worktree 中使用。`deploy/` 为 `preview` / `release` 等候选结果流程维护独立的 deploy-managed detached worktree，它不替代 Claude 的 feature / integration worktree 语义。
- 在非主目录 worktree 中，允许使用 `sed` 等就地编辑工具修改当前 worktree 内文件；不得跨 worktree、不得回写主目录文件。
- **Worktree 逻辑名必须遵循 `{子项目名}.{模块名}.{功能描述}` 格式。** `EnterWorktree` 的 `name` 参数必须按此格式传入，各段仅允许小写字母、数字和连字符，以字母开头，段间以 `.` 分隔。允许的子项目名：`backend`、`backend-go`、`frontend-svelte`（对应目录 `frontend_svelte/`）、`harmonyapp`、`spec`、`deploy`、`claude-config`（对应目录 `.claude/`）。示例：`backend-go.auth.add-oauth`、`harmonyapp.profile.fix-padding`、`claude-config.workflow.enforce-naming-rules`。`worktree_guard.py` 会在创建时校验逻辑名格式和子项目合法性。
- **遇到遗留的同 lineage feature worktree 时，不要复用旧实例，而是新建派生实例继续进度。** 用户传入的三段名是 feature 的逻辑名；若仓库内已存在同一逻辑名对应的 feature branch/worktree，则 `EnterWorktree` 必须自动创建形如 `-r2`、`-r3` 的派生实例名，并基于该 lineage 最新 branch tip 继续，而不是重新进入旧 worktree。旧实例默认视为 superseded，除非用户明确要求，不再继续使用。
- **新 worktree 必须从主目录本地 `main` 的当前 HEAD 创建，绝对不要使用 `origin/main` 或任何远程引用作为基准点。** feature worktree 必须使用 `EnterWorktree` 工具创建（调用时当前工作目录必须在主目录 `main` 分支上）；项目级 `WorktreeCreate` hook 会接管 `EnterWorktree` 的实际创建过程。首次创建 feature 时基于本地 `main`，派生续做时基于同 lineage 最新本地 feature branch。integration worktree 也必须基于主目录本地 `main` 当前 HEAD 创建；若仓库继续采用 `integration_{简要描述}` 一类命名，这属于当前 integration 的软约定而非与 feature 同等级的强命名规则。`tool_guard.py` 会在运行时拦截任何使用 `origin/` 引用的 `git worktree add` 命令。
- 默认每个会话只操作自己创建或明确接管的 worktree / branch；除非用户明确指定，不要处理其他会话对象。
- 当主目录处于本地 `main` 且无未提交改动、操作对象属于当前会话且动作遵循本仓库定义的本地流程时，Claude 应默认自动推进本地闭环，不需要逐步确认；只有越过共享边界、涉及其他会话对象或需要丢弃未确认改动时才暂停确认。
- 每个 feature worktree 先独立完成实现、自检和验证；只有当同一任务被拆分成多个子任务并行开发时，才需要额外新建专门的 integration worktree 来统一汇总结果。
- 一个 feature worktree 的改动范围应持续围绕其建立时的任务目标收敛；若过程中出现新的独立功能需求、明显超出原始目标的范围扩张，默认不要继续混在当前 worktree 中实现，而应新开 feature worktree 承载该新需求。
- 若某任务只有单一 worktree 的结果，则直接在线性历史上完成，不要再额外新建 integration worktree；单 worktree 结果同步回主目录本地 `main` 时，必须先将来源分支整理为适合 rebase 的线性历史，再通过 rebase-only 路径让主目录本地 `main` fast-forward 前进。**绝不允许使用 `git merge`，包括 `git merge --ff-only`，将其他 worktree 的成果同步/合并回主目录本地 `main`。** 若当前状态不满足 rebase，必须先在来源 worktree 内继续整理历史，而不是用 merge 兜底。
- integration worktree 只负责汇总、冲突处理、补测、预演与交付准备，不承载新的大规模实现；其基线始终是主目录本地 `main` 的当前 HEAD，且最终必须通过 `rebase + squash` 收敛为一个 commit，再同步回主目录的本地主分支。若当前 integration 状态不满足 rebase / squash，则必须 refresh/rebuild、replay 或继续整理历史，绝不允许 merge 最新 `main` 或 merge feature 作为补救路径。
- 本地集成结果通过后，再由用户决定是否同步到远程 `main`；PR 不是默认流程。
- 本轮完成后，删除已完成的 feature worktree / branch 与 integration worktree；删除本会话管理的 worktree 时，默认同步删除其对应非主分支 branch。若 `git branch -d` 仅因未合到远程 tracking branch 而拒绝，但已确认结果线性同步回主目录本地 `main`，则可继续使用 `git branch -D` 完成回收。

## 质量门顺序
- 默认顺序：`analyze -> plan -> task list -> implement -> check(语法/编译/警告) -> test -> simplify -> strict review -> commit -> (integrate -> verify) -> sync-back -> cleanup`
- 任何涉及代码修改的 worktree 必须完成 `check → test → review` 闭环；若任一环节发现问题，必须修复后重新从 check 开始循环，直到全部通过。不允许跳过任何环节或在环节未通过时进入下一步。
- `simplify` 负责实现后的代码收敛与重复清理；严格 reviewer 负责合并前门槛，不固定绑定某一个第三方 skill。
- integration 后必须再做一轮 `verify`，不能把各 feature worktree 的局部通过当成集成通过。

## Commit 规范
- commit 消息必须包含子项目名、模块名和更改的功能点，格式：`<type>(<子项目名>/<模块名>): <功能描述>`。scope 必须是 `子项目名/模块名` 双段格式，子项目名须与 worktree 命名中的允许列表一致。跨子项目变更时以主要改动方为 scope，在描述中提及其他子项目。示例：`feat(backend-go/auth): 实现 OAuth2 登录`、`fix(harmonyapp/profile): 修复底部留白`、`chore(claude-config/workflow): 更新命名规则`、`feat(harmonyapp/build): 打通受控编译闭环，涉及 claude-config`。
- 不要在 commit 消息中添加 `Co-Authored-By` 或任何自动生成的署名行。
- 本文提到的"主分支"或"`main`"始终指主目录的**本地** `main` 分支，不涉及远程分支状态。

## 本地验证与交付入口
### 短版工作流
- `verify`：面向并行开发验证，支持多 agent + 多 worktree；默认在当前 worktree 上下文验证，并且在 Docker 隔离环境里完成 backend-go/frontend 的测试与构建检查。
- `verify-inspect`：面向开发内环的调试环境层；在 verify 之后，通过 `deploy/` 提供隔离运行环境与稳定入口，供 Claude Code 主动访问页面、查看 console、核对 UI 与交互。`deploy/` 不承载具体页面测试用例。
- `preview`：面向最终候选结果预演，只有一个实例，用于检查最终候选结果的完整入口行为。
- `release`：面向交付，只产部署输入物，不运行环境。

### 原则
- 只有 `verify` / `verify-inspect` 支持多 env / 多 worktree 并行；`preview` 只有一个实例。
- `verify` 默认验证的是当前工作目录 / worktree，但执行发生在 Docker 隔离环境中，不应把宿主机运行结果视为最终验证结论。
- worktree 内的格式化、语法检查和编译命令通过仓库定义的受控入口执行，不允许直接调用 cargo/go/pnpm/ohpm/hvigor 等原始开发命令；测试命令必须通过 `deploy/` 隔离执行。若某些 deploy 文档提到可在宿主机直接运行静态检查，应仅视为 deploy 设计背景说明，不覆盖这里针对 Claude workflow 的默认约束；最终验证结论仍以 `deploy/` Docker 流程为准。
- 测试过程中产生的缓存、store、构建产物必须落在 `tmp/` 或容器临时目录，不能把 `.pnpm-store` 之类的临时目录写回仓库根。
- `preview` / `release` 面向候选结果与确定性版本，不再承担多 env 开发隔离语义。
- 运行与测试继续遵循：同一套配置 key，不同 datasource value。不要引入额外环境模式抽象或 Redis key-prefix 机制，除非用户明确要求。

## 风险边界
### 可直接执行
- 在专用 worktree 中完成实现、验证、代码收敛与 review。
- 运行标准 `deploy/` 本地流程，包括 `verify`、`preview`、`release` 以及 `deploy/` 管理的本地清理脚本。

### 需要先确认
- 大量文件删除、批量重命名、跨目录大改。
- 修改 `deploy/` 脚本、CI/CD、数据库 migration、共享 `.claude/settings.json`。
- 其他会话对象、状态不明对象，或需要丢弃未确认改动的 branch / worktree 清理与重置操作。

### 高风险
- `git reset --hard`、`git clean`、任何 `--force` 推送或历史改写。
- 使用 `git merge`（包括 `git merge --ff-only`）将其他 worktree 的成果同步/合并回主目录本地 `main`，或在 integration 中 merge 最新 `main` / feature 作为汇总补救。
- 删除仍有未合并改动的 branch / worktree。
- 破坏性远端 deploy / 运维动作。

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
