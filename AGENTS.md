# hotspot

多模块项目：HarmonyOS ArkTS 客户端 + Svelte Web 前端 + Go 主后端。

通用约束（Git 规范、Commit 格式、验证流程、工具链偏好、Docker 等）以全局 `~/AGENTS.md` 为准（各 AI 工具的全局指令文件均链接到该文件），本文件只做项目特有补充，不得放宽或违背全局约束。

## 仓库地图
- `backend-go/`：Go 主后端，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend/`：Svelte Web 主前端，SvelteKit 2 + Svelte 5 + TypeScript + Tailwind CSS，当前所有前端开发在此进行。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。
- `reception-bot/`：社区温度接待机器人（多人格拟真人账号：欢迎/关怀/收件箱回复），**定位为本地调试/联调用途，不做常驻服务**，按需手动启停；运行规程见 `.agents/skills/hotspot-reception-bot/SKILL.md`（唯一事实源）；`state/` 为运行时状态（git 已忽略，含口令归档文件，勿入库）。

## skill 索引（改对应领域前先加载对应 skill）

| skill | 管什么 |
|---|---|
| `hotspot-backend-migrations` | `backend-go/migrations/` 命名、幂等与向后兼容纪律 |
| `hotspot-frontend-style` | 前端样式自查清单与交互状态契约 |
| `hotspot-frontend-theme` | 前端主题/Token 三层架构与配色硬约束 |
| `hotspot-frontend-routing` | 前端文件路由表与组件放置规则 |
| `hotspot-frontend-layout` | 前端 Standard/Split 布局契约 |
| `hotspot-frontend-map` | 前端自研地图契约：marker 全量投影、GCJ-02 同口径、滚轮缩放与漂移实测 |
| `hotspot-harmony-ui` | 鸿蒙端设计 token、原语与跨端同步机制 |
| `hotspot-harmony-device-verify` | 鸿蒙端端上实测：hdc + uitest 走查流程、本机坐标、走查账号与踩坑 |
| `hotspot-worktree-debug` | worktree 内 compose 隔离调试、端口重映射与清理 |
| `hotspot-reception-bot` | 接待机器人运行、人格与线上接入（运行时位置见该 skill 内说明） |

## 构建 & 验证

各模块命令以子项目 `AGENTS.md` 为唯一维护处（`backend-go/AGENTS.md`、`frontend/AGENTS.md`、`harmonyapp/AGENTS.md`），此处不重复列出（速记命令曾与子项目漂移，已删除；一律以子项目为准）。

## 版本号与发布

版本号以 git tag 为源（流程见 `git-workflow`「版本号以 tag 为源」），版本串注入全部产物（后端 health/启动日志、前端 `/version.json` 与设置页徽标、镜像 OCI label），用于线上排障精确回溯。发布为产物分发：tag 触发 CI（`.github/workflows/release.yml`，内置门禁：tag 形态 `vX.Y.Z`、必须落在 `origin/main`、该提交 CI 全绿）构建镜像并附 `.sha256` 挂 GitHub Release，本地 `./scripts/deploy-production.sh vX.Y.Z` 下载分发到服务器 `docker load`——服务器不接触源码、不构建；回滚用 `./scripts/rollback-production.sh`（复用同一链路）。鸿蒙端版本约定见 `harmonyapp/AGENTS.md`。完整流程（含热修复）见 `docs/release-runbook.md`。

### 发布门禁（硬性）：先本地 Docker 验证，再打 tag

禁止把未经本地验证的改动打 tag 发布：在 `git tag vX.Y.Z` 之前，必须按顺序完成：

1. 联调：`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d` 全员 healthy，在 `http://localhost:8888`（nginx 网关）走通本次改动；前端改动辅以 `bun run browser:qa` 巡检（见 `frontend/AGENTS.md`）。
2. 集成测试（后端有改动时）：按 `backend-go/AGENTS.md`「集成测试」的约定启动 `backend-test` 并执行（命令以该文件为唯一维护处，此处不重复）。
3. 生产镜像本地构建（与 CI Release 同构的 Dockerfile；CI 在打 tag 后才构建，本地先构建等于把失败提前；三个指纹变量必须一起传，否则 OCI version 回落为 `dev`）：
   ```bash
   GIT_VERSION=<候选版本> GIT_COMMIT=$(git rev-parse HEAD) GIT_BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
     IMAGE_TAG=<候选版本> docker compose --profile prod build backend frontend-build
   ```
4. 版本指纹就地核对：`docker inspect hotspot-backend:<候选版本> / hotspot-frontend:<候选版本>` 的 OCI `org.opencontainers.image.version` 必须等于候选版本、`revision` 等于本次 commit（Release 工作流同款断言，本地先跑）。
5. 鸿蒙端人工确认（CI 不覆盖 harmonyapp）：其构建/类型检查仅存在于开发者本机 DevEco 链路（pre-commit 调 hvigor `assembleApp`，工具链缺失时告警跳过），CI 无对应 job 兜底——tag 前必须人工确认涉及 `harmonyapp/` 的改动已在本地通过 hvigor 构建（见 `harmonyapp/AGENTS.md`）。

全部通过后才打 tag → 等 CI「Release」成功 → `./scripts/deploy-production.sh vX.Y.Z`（脚本会复核 tag 位置、发布包校验和、镜像 OCI 指纹与线上三级版本）。

**热修复遵循同一门禁**：本项目 tag 必须落在 `origin/main` 上，热修同样要回合主干后打 patch tag，不存在绕过主干的旁路发布；线上先止血用 `./scripts/rollback-production.sh --previous`。流程见 `docs/release-runbook.md`。

例外：纯文档/注释改动（`*.md`、代码注释）可跳过第 1、2 步；文档改动不进镜像时第 3、4 步也可跳过。

## UI 审计基线

前端与鸿蒙的 UI 质量项（视口填充、留白对称、对比度、动效、触控热区、焦点可见、空态）与逐页
走查结论见 `docs/frontend-ui-audit.md`。**已确认达标项不必重复审计**；新增页面按下表方法复测对应项。

## 调试入口

不要直接启动前端 dev server 直连浏览器。前端是纯 UI，必须配合后端服务。

| 项目 | 调试方式 | 入口 |
|------|----------|------|
| 前端 | docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d | http://localhost:8888（nginx 网关） |
| Go 后端 | 同上（Docker 内 hot-reload 由 air 负责） | 同上，/api 经 nginx 反代 |

详见各子项目 `AGENTS.md`：`frontend/AGENTS.md`、`backend-go/AGENTS.md`。

### 开发种子数据（dev 环境）

前端**不内置任何 API mock**：dev 下 API / WebSocket / S3 全部直连真实后端。首次启动（或数据被改乱后）
执行种子命令注入一套可复现的测试数据（Postgres + SeaweedFS 双写，幂等，重复执行不产生重复数据）：

```bash
./scripts/seed-dev.sh           # 缺失则写入；已有 seed 数据则跳过（保留开发中产生的互动）
./scripts/seed-dev.sh --reset   # 按固定前缀清空 seed 数据后重建（含 S3 对象）
./scripts/seed-dev.sh --dry-run # 只打印写入计划
```

| 账号 | login_id | 口令（`.env` 可覆盖） | 角色 |
|------|----------|------------------------|------|
| 鹿鸣 | `qa@hotspot.local` | `SEED_QA_PASSWORD`（默认 `Qa12345678`） | 普通用户，QA 巡检主账号 |
| 巡检员 | `admin@hotspot.local` | `SEED_ADMIN_PASSWORD`（默认 `Admin12345678`） | 管理员（grant_all），同时是完整普通用户 |

口令仅本地开发使用，生产部署不得沿用；数据规格、媒体素材与实现见 `backend-go/internal/seed/` 与
`backend-go/AGENTS.md`「开发种子数据」。

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
- **环境类结论必须可复现，否则视为未验证假设**：在 skill / 文档 / 对话里写下「本机不可验证」「必须真机」「环境限制」「某页面白屏」这类结论时，必须同时给出**验证日期 + 复现步骤**；沿用别人（或上一轮自己）写下的环境结论前，**先证伪一次**。
  - 本项目真实教训：一条陈旧的「推入式页面白屏、必须真机」结论被沿用多轮，掩盖了一个真 bug（ArkUI 裸 `{ }` 块 → `layoutWeight` 失效 → 内容区全空），直到端上截图才暴露。
  - 推论：判定「页面是否正常」一律以**截图**为准；`dumpLayout` 的节点数只能证明"有无渲染"，页头正常而内容区全空时它同样有文本，会被误判为正常页。
- 范围外的问题记到 `.workwork/` 或如实告知用户，不擅自扩大改动面。

