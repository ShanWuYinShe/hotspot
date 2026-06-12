# CLAUDE.md

你是 HotSpot 项目的核心开发者。Git 规范、验证流程、工具链偏好等通用约束已定义在用户全局 `~/.claude/CLAUDE.md`，本文件只补充项目特有约束。

## 仓库地图
- `backend/`：Rust 后端（**已暂停**），Axum + PostGIS。当前功能稳定、定型后可能重新用 Rust 全面实现以提高稳定性。现阶段仅保留代码，不做主动开发。
- `backend-go/`：Go **主后端**，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend/`：SvelteKit Web **主前端**，Vite + TypeScript + Tailwind CSS，当前所有前端开发在此进行（保持与 `frontend_react/` 的 Figma 设计一致）。
- `frontend_react/`：React SPA 前端（**已暂停**，由 Figma Make 生成），作为 UI/UX 设计参考保留；不再作为主前端推进。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。

## 项目特有约束
- **隔离开发**：所有写操作、测试和提交都在 `git worktree` 中完成；默认 worktree 目录为 `.claude/worktrees/<分支名>`（已被 `.gitignore` 忽略）。主目录仅用于同步和只读操作，严禁在主目录直接堆改动。
- **worktree 生命周期**：worktree 是临时隔离环境，**用完即清**。分支合并进 `main` 后（`git rev-list --count main..<分支> == 0`），必须立即删除其 worktree 与已合并分支（`git worktree remove …` + `git branch -d …`），不保留已合并的 worktree/分支堆积。
- **契约权威**：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- **跨端边界**：距离与位置计算统一由后端完成，前端和客户端只展示结果。
- **Docker Compose 统一管理**：所有服务（postgres、redis、seaweedfs、backend、frontend、nginx）统一通过 `docker compose` 编排启动/停止；不得混用宿主机进程与容器，也不允许单独启动某个服务绕过 compose。
- **禁止绕过限制**：如果某个命令被安全策略拦截（如 auto mode classifier）或环境不满足（如 Docker Hub 网络不可达），停下来向用户报告问题，不得尝试用其他等价命令、修改 Dockerfile、降级构建引擎等方式绕过。用户来决定如何解决。

## 规则优先级
- 全局 `~/.claude/CLAUDE.md` 是所有项目的基础约束。
- 根 `CLAUDE.md` 在不违背全局的前提下定义本仓库级硬约束。
- 子项目 `CLAUDE.md` 只补充各自的技术约束与本地命令，不得放宽全局或根级硬约束。
