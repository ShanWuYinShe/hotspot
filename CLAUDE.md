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
- 再读对应子项目 `CLAUDE.md`，了解局部架构、实现护栏和验证提示。
- 任何接口、字段、错误码、分页、幂等或 WebSocket 变更前，先读 `spec/api_spec_index.md`、对应模块 spec，以及 `spec/api_spec_base.md`、`spec/api_spec_types.md`、`spec/api_spec_errors.md`。

## 根文档是唯一 workflow authority
- 仓库级 workflow、quality gate、sync-back、cleanup、ownership、自动模式停止条件，**统一以根 `CLAUDE.md` 为准**。
- `.claude/commands/*.md` 只说明某个入口什么时候用、会推进哪些状态、不会做什么；不要在里面重复整套仓库级规则。
- 子项目 `CLAUDE.md` 只讲局部架构、局部实现边界和局部验证提示；不要复制 worktree / integration / sync-back / cleanup 正文。
- memory 只记录 durable 的协作偏好或 repo delta；不要再重复根文档里已经权威表达的 workflow 规则。

## 默认协作模型
本仓库优先通过三种机制协同，而不是把 `PreToolUse` 当成主控制器：

1. **文档引导**：根 `CLAUDE.md` 说明 workflow、边界和推荐路径；子项目 `CLAUDE.md` 只补局部上下文。
2. **skill 暴露能力**：优先通过 `.claude/commands/*.md` 暴露成熟入口，例如 `start_feature`、`start_integrate`、`verify_integrate`、`finish_integrate`、`sync_back_main`、`cleanup_worktrees`、`status_worktrees`、`controlled_tools`。
3. **受控脚本执行严格步骤**：只有隔离运行/测试、集成控制面、候选预演、发布输入物等高确定性或高副作用步骤，才需要强制走 `.claude/scripts/*.py` 或 `deploy/scripts/*.sh`。

`PreToolUse` 只保留底线保护：
- 主目录本地 `main` 禁止直接改文件。
- 高风险 git / 破坏性动作继续阻断。
- 明确需要隔离环境的运行态动作，继续要求走仓库入口。
- 发生拦截时，hook 必须把拦截原因显式打印出来，不能只在内部 JSON 字段里静默返回。

不要把“没有对应受控脚本”自动理解成“这个动作不能做”。优先判断它是：
- 普通只读/低副作用动作，可直接做；
- 仓库已提供 skill 的成熟入口，应优先用 skill；
- 隔离运行或高确定性控制面动作，应走受控脚本。

## 状态术语
### worktree / integration ownership
- `owned`：当前自动流程创建或已明确接管，并且 metadata 中的 `owner_token` 与当前流程一致。
- `foreign`：对象存在，但归属于其他流程 / 其他会话，不应自动继续推进。
- `takeover_required`：对象存在但缺少可验证 owner，必须先明确接管，不能默认自动继续。

### 派生状态
- `active`：基线仍与主目录本地 `main HEAD` 一致，可继续当前流程。
- `stale`：基线已过期；feature / integration 若要继续，必须 refresh/rebuild，而不是 merge 最新 `main`。
- `superseded`：同 lineage 中已有更新的派生实例，旧实例默认不再继续承载实现，只等 cleanup 统一判断。
- `ready_for_integration`：feature 已完成本地实现、quality gate 和 review，但当前目标是进入 integration 汇总，而不是立即 sync-back。
- `ready_to_sync`：当前 feature 或 integration 已满足 rebase-only sync-back 的前提。
- `safe_to_cleanup`：对象已 sync-back 到本地 `main`、无未提交改动、分支 tip 已等于 `main`，且 ownership 可验证，允许自动 cleanup。

## 整体流程
本仓库的默认本地 workflow 是：
1. **任务描述**：用户在对话中描述任务。
2. **任务分析**：Claude Code 分析任务范围，判断是否拆成多个子任务；子任务边界清晰且文件集相对独立时才并行，否则保持单任务。
3. **创建 feature worktree 并实现**：从主目录本地 `main` 当前 HEAD 创建 feature worktree，在 worktree 中完成实现、自检、测试与 review。
4. **线性同步回主目录**：
   - 单任务：在 feature worktree 中整理为适合同步的线性历史，通过 rebase-only 路径同步回主目录本地 `main`。
   - 多子任务：新建专门的 integration worktree，汇总多个 feature 结果，并在 integration worktree 中完成最终 verify / simplify / review / squash，再同步回主目录本地 `main`。
5. **cleanup**：清理本轮完成使命且已安全同步的 worktree / branch / runtime metadata / 中间产物。

任何涉及代码修改的 worktree 都必须完成以下闭环：
`编写代码 → 检查（语法错误、编译错误、警告） → 测试 → simplify → strict review`
若任一环节发现问题，必须修复后重新循环，直到全部通过。

## 默认自动模式
当任务边界清晰、对象 ownership 可验证、metadata 状态完整且改动范围仍在安全边界内时，Claude 可以默认自动推进：
`任务分析 → feature worktree → 实现 → quality gates → commit → sync-back → ExitWorktree → cleanup`

但自动模式只适用于**本地仓库内、当前流程拥有的对象**。以下情况必须停止自动推进，回到澄清或确认：
- 需求边界不清，或新增需求已明显偏离当前 feature 原始目标。
- 当前对象是 `foreign` / `takeover_required`。
- feature / integration 基线已 `stale`。
- verify / simplify / review / replay completeness 任一未完成。
- 触达 `spec/`、`deploy/`、`.claude/settings*.json`、数据库 migration、大量删除/批量重命名。
- cleanup 涉及 foreign object、状态不明对象或仍有未合并改动的对象。

## 全局权威与共享边界
- `spec/` 是唯一权威契约来源；除非用户明确要求，不要修改其中文件。
- 不要猜测 spec 中不存在的接口、字段、错误码或行为；实现与 spec 冲突时，优先修改实现而不是反改 spec。
- 默认只在仓库内工作；确需访问仓库外路径时，只使用用户明确提供或仓库文档已约定的绝对路径。
- 临时文件、中间文件、验证截图、调试输出统一放在仓库根目录 `tmp/`；`tmp/` 已被 git ignore，不要把这类文件落到 worktree 根或子项目目录中。
- 共享自动化、hooks 与团队级权限优先放 `.claude/settings.json`；本机额外差异若必须存在，也只能作为共享默认行为的补充，而不能承载另一套 workflow。
- Git 控制面动作默认分步执行：不要把多个 git 状态变更命令用 `&&`、`;` 或 `||` 串成单条 Bash。即使步骤之间存在依赖，也应先执行前一步、观察结果，再进入下一步。

## 什么时候优先用 skill
遇到以下场景，优先考虑 skill，而不是临时拼装流程：
- 启动标准 feature 开发流：`start_feature`
- 查看当前 worktree / integration 总览：`status_worktrees`
- 启动 integration 汇总：`start_integrate`
- 运行 integration 标准验证：`verify_integrate`
- 进行 integration 最终 gate 判断：`finish_integrate`
- 执行 sync-back：`sync_back_main`
- 清理 worktree / branch：`cleanup_worktrees`
- 想知道仓库已经提供了哪些严格入口：`controlled_tools`

skill 是推荐入口和能力发现层，不等于普通分析、只读检查、局部开发动作都必须通过 skill 完成。

## 主目录与 worktree 模型
- 主目录默认保持在本地 `main` 且无未提交改动；主目录不承载实现性改动。
- 主目录允许只读分析、安全的 git 控制面动作，以及通过 skill/脚本执行标准 workflow 控制面。
- 任何实现性改动都在 feature / integration worktree 中完成；需要本地分支时，也在对应 worktree 中使用。
- 在非主目录 worktree 中，允许做普通开发、局部 git、只读/低副作用检查；不要跨 worktree 写文件。
- **Worktree 逻辑名必须遵循 `{子项目名}.{模块名}.{功能描述}` 格式。** `EnterWorktree` 的 `name` 参数必须按此格式传入，各段仅允许小写字母、数字和连字符，以字母开头，段间以 `.` 分隔。允许的子项目名：`backend`、`backend-go`、`frontend-svelte`（对应目录 `frontend_svelte/`）、`harmonyapp`、`spec`、`deploy`、`claude-config`（对应目录 `.claude/`）。示例：`backend-go.auth.add-oauth`、`harmonyapp.profile.fix-padding`、`claude-config.workflow.enforce-naming-rules`。
- **遇到遗留的同 lineage feature worktree 时，不要复用旧实例，而是新建派生实例继续进度。** 若仓库内已存在同一逻辑名对应的 feature branch/worktree，则 `EnterWorktree` 应创建 `-rN` 派生实例，并基于该 lineage 最新 branch tip 继续。
- **新 worktree 必须从主目录本地 `main` 的当前 HEAD 创建，绝对不要使用 `origin/main` 或任何远程引用作为基准点。**
- 若某任务只有单一 feature worktree 的结果，则不额外创建 integration worktree；在 sync-back 前把来源分支整理为适合同步的线性历史即可。
- integration worktree 只负责汇总、冲突处理、补测、预演与交付准备，不承载新的大规模实现；其基线始终是主目录本地 `main` 的当前 HEAD，且最终必须通过 `rebase + squash` 收敛为一个 commit，再同步回主目录。
- **绝不允许使用 `git merge`（包括 `git merge --ff-only`）将其他 worktree 的成果同步/合并回主目录本地 `main`。** 若当前状态不满足 rebase，应先在来源 worktree 或 integration worktree 内整理历史。

## 质量门顺序
- 默认顺序：`analyze -> plan -> task list -> implement -> check(语法/编译/警告) -> test -> simplify -> strict review -> commit -> (integrate -> verify) -> sync-back -> cleanup`
- 任何涉及代码修改的 worktree 都必须完成 `check → test → review` 闭环；若任一环节失败，必须修复后重新从 check 开始循环。
- `simplify` 负责实现后的代码收敛与重复清理；严格 reviewer 负责合并前门槛，不固定绑定某一个第三方 skill。
- integration 后必须再做一轮 `verify`，不能把各 feature worktree 的局部通过当成集成通过。
- 只有当 runtime metadata 已明确表达 `ready_to_sync` 时，才允许自动继续 `/sync_back_main`。

## Commit 规范
- feature worktree 开发期间允许多个 commit；在 sync-back 前必须整理为适合同步的线性历史。
- integration worktree 在进入最终 sync-back 前必须 squash 为 **1 个 commit**。
- commit 消息必须包含子项目名、模块名和更改的功能点，格式：`<type>(<子项目名>/<模块名>): <功能描述>`。scope 必须是 `子项目名/模块名` 双段格式，子项目名须与 worktree 命名中的允许列表一致。
- 不要在 commit 消息中添加 `Co-Authored-By` 或任何自动生成的署名行。
- 本文提到的 `main` 始终指主目录的**本地** `main` 分支，不涉及远程分支状态。

## 本地验证与交付入口
### 推荐入口
- `verify`：并行开发验证入口，支持多 agent + 多 worktree；默认在当前 worktree 上下文验证，并在 Docker 隔离环境里完成 backend-go/frontend 的测试与构建检查。
- `verify-inspect`：开发内环的隔离调试入口；在 verify 之后提供稳定页面入口，供 Claude Code 主动访问页面、查看 console、核对 UI 与交互。
- `preview`：最终候选结果预演，只有一个实例。
- `release`：正式交付输入物入口，只产物，不运行环境。

### 原则
- 只有 `verify` / `verify-inspect` 支持多 env / 多 worktree 并行；`preview` 只有一个实例。
- `verify` 默认验证的是当前工作目录 / worktree，但执行发生在 Docker 隔离环境中，不应把宿主机运行结果视为最终验证结论。
- 测试、起服务、preview、release、会写大量缓存/产物或会在多 agent 下相互冲突的动作，优先通过 `deploy/` 隔离执行。
- 静态分析、普通只读检查、局部低副作用动作，不必因为没有脚本就一律阻断。
- 测试过程中产生的缓存、store、构建产物必须落在 `tmp/` 或容器临时目录，不能把 `.pnpm-store` 之类的临时目录写回仓库根。

## 风险边界
### 可直接执行
- 主目录中的只读分析、安全 git 控制面动作。
- worktree 中的普通实现、局部 git、只读/低副作用检查。
- 通过 skill 使用仓库已经提供的标准 workflow 入口。

### 需要先确认
- 大量文件删除、批量重命名、跨目录大改。
- 修改 `deploy/` 脚本、CI/CD、数据库 migration、共享 `.claude/settings.json`。
- 其他会话对象、状态不明对象，或需要丢弃未确认改动的 branch / worktree 清理与重置操作。

### 高风险
- `git reset --hard`、`git clean`、任何 `--force` 推送或历史改写。
- 使用 `git merge` 将其他 worktree 的成果同步/合并回主目录本地 `main`，或在 integration 中 merge 最新 `main` / feature 作为补救。
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
