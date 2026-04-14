# HotSpot

基于地理位置的社交应用。

## 技术栈

| 子项目 | 技术栈 |
|--------|--------|
| `backend/` | Rust / Axum / SQLx / PostGIS |
| `backend-go/` | Go / Chi / pgx / PostgreSQL / S2 |
| `frontend_svelte/` | SvelteKit / Svelte 5 / TypeScript |
| `harmonyapp/` | HarmonyOS ArkTS |
| `spec/` | API 规范 |

## 开发协作

- 协作与实现说明先读 [CLAUDE.md](CLAUDE.md)，进入子项目后再读对应目录下的 `CLAUDE.md`。
- 仓库优先通过 **分层规则 + skill 入口 + 少量严格脚本** 协作，而不是依赖大量 hook 阻断。
- 根 `CLAUDE.md` 只定义硬约束、任务分级和协作入口；子项目 `CLAUDE.md` 负责技术约束和本地命令；workflow skill 负责编排具体流程。
- 普通分析、代码阅读、局部低副作用开发动作通常可以直接完成；中大型任务再升级到设计、计划或调试 skill。
- 临时文件、中间文件、验证截图和调试输出统一写入 `.local/`，避免污染 git 状态。
- 涉及数据库写入、迁移、跨服务联调等有副作用的操作，应在 Docker 隔离环境中执行。

## 任务与验证分级

- 任务分级：`L0` 只读分析，`L1` 低风险小改动，`L2` 需要简短计划的中等改动，`L3` 需要先设计的高风险改动。
- 验证分级：`开发完成` 只要求与改动直接相关的本地校验通过；`准备合并` 按风险自行判断是否需要 Docker 隔离验证。
- 所有写操作默认在 `.worktrees/` 下的 worktree 中完成。

## 常见入口

### skill / workflow 入口
- `hotspot-workflow`：单一 workflow skill，总入口；可由 Claude 自动应用，也可显式调用 `/hotspot-workflow <feature|cleanup> ...`
  - `/hotspot-workflow feature <功能描述>`：端到端功能开发流程
  - `/hotspot-workflow cleanup [过滤条件]`：盘点并清理已合并或废弃的临时 worktree 与分支

### worktree 入口
```bash
git worktree add .worktrees/<branch-name> -b <branch-name>
```

## 其他入口

- [spec/](spec/) - API 规范
