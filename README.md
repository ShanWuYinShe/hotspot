# HotSpot

基于地理位置的社交应用。

## 技术栈

| 子项目 | 技术栈 |
|--------|--------|
| `backend/` | Rust / Axum / SQLx / PostGIS |
| `backend-go/` | Go / Chi / pgx / PostgreSQL / S2 |
| `frontend_svelte/` | SvelteKit / Svelte 5 / TypeScript |
| `harmonyapp/` | HarmonyOS ArkTS |
| `deploy/` | Docker / Nginx |
| `spec/` | API 规范 |

## 开发协作

- 协作与实现说明先读 [CLAUDE.md](CLAUDE.md)，进入子项目后再读对应目录下的 `CLAUDE.md`。
- 仓库优先通过 **文档引导 + skill 入口 + 少量严格脚本** 协作，而不是依赖大量 hook 阻断。
- 普通分析、代码阅读、局部低副作用开发动作通常可以直接完成；当仓库已经提供对应 skill 时，优先通过 skill 使用成熟入口。
- 隔离运行、统一验证、候选预演、发布输入物等高确定性步骤仍优先走 `deploy/scripts/` 提供的标准入口。
- 临时文件、中间文件、验证截图和调试输出统一写入 `tmp/`，避免污染 git 状态。

## 常见入口

### skill / workflow 入口
- `hotspot-workflow`：单一 workflow skill，总入口；可由 Claude 自动应用，也可显式调用 `/hotspot-workflow <feature|cleanup> ...`
  - `/hotspot-workflow feature <功能描述>`：端到端功能开发流程
  - `/hotspot-workflow cleanup [过滤条件]`：盘点并清理已合并或废弃的临时 worktree 与分支

### 隔离验证入口
```bash
./deploy/scripts/verify.sh --env-name <name>
./deploy/scripts/verify-inspect.sh --env-name <name>
```

## 其他入口

- [spec/](spec/) - API 规范
- [deploy/](deploy/) - 本地验证与运行入口
