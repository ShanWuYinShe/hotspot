# hotspot

多模块项目：HarmonyOS ArkTS 客户端 + Svelte Web 前端 + Go 主后端。

通用约束（Git 规范、Commit 格式、验证流程、工具链偏好、Docker 等）以全局 `~/AGENTS.md` 为准（各 AI 工具的全局指令文件均链接到该文件），本文件只做项目特有补充，不得放宽或违背全局约束。

## 仓库地图
- `backend-go/`：Go 主后端，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend/`：Svelte Web 主前端，SvelteKit 2 + Svelte 5 + TypeScript + Tailwind CSS，当前所有前端开发在此进行。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。

## 按需加载的 skill（.agents/skills/）

| skill | 何时加载 |
|---|---|
| `hotspot-worktree-debug` | worktree 内 compose 调试、端口映射与收尾清理 |
| `hotspot-backend-migrations` | 编写或执行 backend-go 数据库迁移 SQL、命名与兼容约束 |
| `hotspot-harmony-ui` | 修改 harmonyapp 界面样式、使用设计 token、跨端色彩/刻度同步 |
| `hotspot-frontend-layout` | 修改前端页面结构、布局分栏、选用 Standard/SplitPageLayout |
| `hotspot-frontend-routing` | 新增或调整前端路由、+page.svelte 组织与设备分派 |
| `hotspot-frontend-style` | 前端组件交互四态契约（hover/focus/active/disabled）与样式自查 |
| `hotspot-frontend-theme` | 前端修改颜色/字号/圆角 token、自托管字体与主题配置 |

## 构建 & 验证

各模块命令以子项目 `AGENTS.md` 为唯一维护处（`backend-go/AGENTS.md`、`frontend/AGENTS.md`、`harmonyapp/AGENTS.md`），此处不重复列出，避免多处同步。以下快速入口仅为速记，与子项目 `AGENTS.md` 不一致时以后者为准：

```bash
go build -C backend-go ./... && go vet -C backend-go ./... && go test -C backend-go ./...  # 后端
bun install --cwd frontend && bun run --cwd frontend build                                 # 前端
```

## 版本号与发布

版本号以 git tag 为源（流程见 `git-workflow`「版本号以 tag 为源」），版本串注入全部产物（后端 health/启动日志、前端 `/version.json` 与设置页徽标、镜像 OCI label），用于线上排障精确回溯。发布为产物分发：tag 触发 CI（`.github/workflows/release.yml`）构建镜像挂 GitHub Release，本地 `./scripts/deploy-production.sh vX.Y.Z` 下载分发到服务器 `docker load`——服务器不接触源码、不构建；鸿蒙端版本约定见 `harmonyapp/AGENTS.md`。详见 README「版本号管理与发布」。

### 发布门禁（硬性）：先本地 Docker 验证，再打 tag

禁止把未经本地验证的改动打 tag 发布：在 `git tag vX.Y.Z` 之前，必须按顺序完成：

1. 联调：`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d` 全员 healthy，在 `http://localhost:8888`（nginx 网关）走通本次改动；前端改动辅以 `bun run browser:qa` 巡检（见 `frontend/AGENTS.md`）。
2. 集成测试（后端有改动时）：`docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile test up -d backend-test`，再在 `backend-go/` 目录执行 `BACKEND_URL=http://localhost:8082 go test -count=1 ./test/integration/`（完整约定见 `backend-go/AGENTS.md`）。
3. 生产镜像本地构建（与 CI Release 同构的 Dockerfile；CI 在打 tag 后才构建，本地先构建等于把失败提前；三个指纹变量必须一起传，否则 OCI version 回落为 `dev`）：
   ```bash
   GIT_VERSION=<候选版本> GIT_COMMIT=$(git rev-parse HEAD) GIT_BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
     IMAGE_TAG=<候选版本> docker compose --profile prod build backend frontend-build
   ```
4. 版本指纹就地核对：`docker inspect hotspot-backend:<候选版本> / hotspot-frontend:<候选版本>` 的 OCI `org.opencontainers.image.version` 必须等于候选版本、`revision` 等于本次 commit（Release 工作流同款断言，本地先跑）。
5. 鸿蒙端人工确认（CI 不覆盖 harmonyapp）：其构建/类型检查仅存在于开发者本机 DevEco 链路（pre-commit 调 hvigor `assembleApp`，工具链缺失时告警跳过），CI 无对应 job 兜底——tag 前必须人工确认涉及 `harmonyapp/` 的改动已在本地通过 hvigor 构建（见 `harmonyapp/AGENTS.md`）。

全部通过后才打 tag → 等 CI「Release」成功 → `./scripts/deploy-production.sh vX.Y.Z`（脚本会复核 tag 位置与版本指纹）。

例外：纯文档/注释改动（`*.md`、代码注释）可跳过第 1、2 步；文档改动不进镜像时第 3、4 步也可跳过。

## 调试入口

不要直接启动前端 dev server 直连浏览器。前端是纯 UI，必须配合后端服务。

| 项目 | 调试方式 | 入口 |
|------|----------|------|
| 前端 | docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d | http://localhost:8888（nginx 网关） |
| Go 后端 | 同上（Docker 内 hot-reload 由 air 负责） | 同上，/api 经 nginx 反代 |

详见各子项目 `AGENTS.md`：`frontend/AGENTS.md`、`backend-go/AGENTS.md`。

## worktree 调试与清理

worktree 内调试同样通过 Docker Compose，与主工作树自动隔离。
**完整操作细则见 skill hotspot-worktree-debug**（compose 隔离与端口、目录名同名冲突时的
显式项目名与端口重映射、合并后重启主容器、清理顺序）。要点速记：

- 不要指定 -p：compose 项目名默认取当前目录名（主工作树 hotspot，worktree 该目录名），自动隔离；显式 -p hotspot 会接管主实例。
- worktree 目录名恰为 hotspot（如 .worktrees/<id>/hotspot）时项目名冲突，必须显式 -p 加端口重映射，否则会接管主实例容器。
- 合并后若主环境容器在运行，必须主动重启受影响容器，确保最新代码生效。

> 细节、命令与排查见 skill；本处只保留必须记住的约束。

## 项目特有约束
- 契约权威：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- 跨端边界：距离与位置计算统一由后端完成，前端和客户端只展示结果。
- **int64 跨端序列化统一为字符串**：JS（Svelte）与 ArkTS 的 number 是 IEEE754 double，无法无损表示 Go int64（安全整数上限 2^53）。凡 spec 定义 string、后端为 int64 的字段（如 WS `server_seq`），后端 JSON tag 必须加 `,string` 下发字符串；前端/harmonyapp 类型保持 string 仅透传/去重、不做数值运算。禁止以「后端实际下发数字」为由把客户端类型改成 number——正确修法是后端序列化为字符串。
- Docker Compose 统一管理：运行时服务（postgres、redis、seaweedfs、backend、frontend、nginx）统一通过 `docker compose` 编排启动/停止；不得混用宿主机进程与容器，也不允许单独启动某个运行时服务绕过 compose。静态检查（`bun run check/test/build`、Go 单测等不依赖服务间协作的命令）可在宿主机执行。
- **改动生效闭环**：代码改动合并到主分支后，若主环境 Docker 容器在运行，必须主动重启相关容器，确保最新效果立即可见（命令与健康检查见 skill `hotspot-worktree-debug`）。
- 范围外的问题记到 `.workwork/` 或如实告知用户，不擅自扩大改动面。

