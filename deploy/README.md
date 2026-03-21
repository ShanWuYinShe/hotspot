# HotSpot deploy workflow

`deploy/` 现在只保留三层语义：

- **verify**：默认本地验证层，服务多 agent + 多 worktree 并行开发
- **preview**：单实例本地预演层，验证最终候选结果的 HTTPS / nginx / websocket / MinIO / cookie 行为
- **release**：正式部署输入层，只产 image / artifact / manifest，不运行环境

## 快速入口

### 短版工作流
- `verify`：用于并行开发验证，支持多 agent + 多 worktree。默认验证当前工作目录 / worktree，并在 Docker 隔离环境里完成 backend-go/frontend 的测试与构建检查。示例：`./deploy/scripts/verify.sh --env-name feature-a`
- `preview`：用于最终候选结果预演，只有一个实例。先 `./deploy/scripts/preview-build.sh`，再 `./deploy/scripts/preview-up.sh`，最后 `./deploy/scripts/preview-inspect.sh`
- `release`：用于产出正式部署输入物，不运行环境。示例：`./deploy/scripts/release-build.sh --ref HEAD --label release-2026-03-21`

### 常用命令

#### verify
```bash
./deploy/scripts/verify.sh --env-name feature-a
```

#### preview
```bash
./deploy/scripts/preview-build.sh
./deploy/scripts/preview-up.sh
./deploy/scripts/preview-inspect.sh
```

停止或清理 preview：

```bash
./deploy/scripts/preview-down.sh
./deploy/scripts/preview-down.sh --remove
./deploy/scripts/preview-down.sh --clean-all
```

#### release
```bash
./deploy/scripts/release-build.sh --ref HEAD
./deploy/scripts/release-build.sh --ref <sha-or-tag> --label release-2026-03-21
```

## 当前原则
- 只有 `verify` 支持多实例并行。
- `preview` 只有一个实例，不再按 worktree 并行。
- `release` 不运行环境，只产物。
- `verify` 默认验证当前工作目录 / worktree，包含未提交修改。
- 所有测试、check、build 验证默认都通过 `deploy/` 下的 Docker 流程完成；不要把宿主环境直接运行 `go test`、`pnpm check`、`pnpm build` 当成最终验证。
- 测试和构建流程产生的缓存、pnpm store、临时产物必须落在 `tmp/` 或容器临时目录，不能写回仓库根。
- `preview` / `release` 面向候选结果与确定性版本，不再承担多 env 开发隔离语义。

## 目录结构
- `compose/verify.yml`：verify 编排
- `compose/preview.yml`：preview 编排
- `dockerfiles/`：runner / app Dockerfiles
- `scripts/`：verify / preview / release 入口
- `nginx/hotspot.preview.nginx.conf`：preview 统一入口配置

更多设计说明见 [deployment.md](deployment.md)
