---
description: 端到端功能开发：创建 worktree → 实现 → 分层测试 → 压缩为单 commit → 线性同步 → 清理
argument-hint: <功能描述>
---

用户需求：$ARGUMENTS

请以**端到端自主模式**完成这个需求。目标是让 `main` 分支最终增加一个**单独的、语义清晰的 commit**。

## 执行步骤

### 1. 准备与实现
- 确认主目录位于 `main` 分支且无未提交改动。
- 创建 worktree：分支名格式 `{子项目}/{模块}/{简述}`（如 `backend-go/auth/add-oauth`），路径建议 `../hotspot-feature-<分支名>`。
- 在 worktree 中完成代码编写，期间可自由进行多个临时 commit。

### 2. 分层测试
- **本地单元测试**：在 worktree 中运行不依赖数据库/外部服务的测试。
- **隔离集成测试**：若改动涉及数据库写入、Schema 变更或需要真实环境验证，运行：
  `./deploy/scripts/verify.sh --env-name <name>`
  该脚本在 Docker 中提供干净环境，验证通过后再继续。

### 3. 整理历史
- 查看领先 `main` 的 commit 数：`git log --oneline main..HEAD`
- **若仅有一个 commit**：确保 message 符合规范（见下），跳过 squash。
- **若有多个 commit**：执行 `git rebase -i main`，将除第一个外的 `pick` 改为 `squash`，编辑最终 message 为：
  `<type>(<scope>): <subject>`
  示例：`feat(backend-go/auth): add oauth login flow`

### 4. 线性同步
- 切回主目录：`git checkout main`
- 同步远程最新 `main`：`git pull --ff-only origin main`（或 `git pull --rebase`）
- 合并你的单个 commit：
  - 优先 `git merge --ff-only <你的分支名>`
  - 若失败（主线有新提交），执行 `git rebase main <你的分支名>` 后再次 `git merge --ff-only <你的分支名>`

### 5. 清理现场
- 切回主目录。
- 删除 worktree：`git worktree remove <worktree路径>`
- 删除本地分支：`git branch -D <分支名>`

## 完成标准
- 本地单元测试通过。
- 隔离集成测试（如需要）通过。
- `main` 分支新增一个符合规范的 commit。
- worktree 和临时分支已清理。

现在开始执行。
