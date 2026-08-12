# hotspot

多模块项目：HarmonyOS ArkTS 客户端 + Svelte Web 前端 + Go 主后端 + Rust 旧后端。

通用约束（Git 规范、Commit 格式、验证流程、工具链偏好、Docker 等）以全局 `~/.config/opencode/AGENTS.md` 为准，本文件只做项目特有补充，不得放宽或违背全局约束。

## 仓库地图
- `backend-go/`：Go 主后端，模块化单体 + S2 地理索引，当前所有后端开发在此进行。
- `frontend/`：Svelte Web 主前端，SvelteKit 2 + Svelte 5 + TypeScript + Tailwind CSS，当前所有前端开发在此进行。
- `harmonyapp/`：HarmonyOS ArkTS 客户端。
- `spec/`：API 契约的唯一权威来源（只读）。

## 构建 & 验证

```bash
# Go 主后端
go build -C backend-go ./...
go vet -C backend-go ./...
go test -C backend-go ./...

# 前端
bun install --cwd frontend
bun run --cwd frontend build
```

## 调试入口

不要直接启动前端 dev server 直连浏览器。前端是纯 UI，必须配合后端服务。

| 项目 | 调试方式 | 入口 |
|------|----------|------|
| 前端 | docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d | http://localhost:8888（nginx 网关） |
| Go 后端 | 同上（Docker 内 hot-reload 由 air 负责） | 同上，/api 经 nginx 反代 |

详见各子项目 `AGENTS.md`：`frontend/AGENTS.md`、`backend-go/AGENTS.md`。

## worktree 调试与清理

worktree 内调试同样通过 Docker Compose，与主工作树自动隔离：

- 不要指定 `-p`：compose 项目名默认取当前目录名（主工作树 = `hotspot`，worktree = 该分支目录名），自动隔离容器/网络/volume；显式 `-p hotspot` 会接管主实例容器，隔离失效
- 在对应 worktree 根目录执行（`docker-compose.yml` 在 worktree 内同样存在），cwd 必须是该目录
- 默认端口映射（8888/8081/5432）与主实例相同，同一时间只允许一套 dev 环境运行；需要并行时按下方「目录名同名冲突」节重映射端口

### worktree 目录名与主工作树同名（compose 项目名冲突）

自动隔离依赖「worktree 目录名 ≠ `hotspot`」。若 worktree 目录名恰为 `hotspot`
（如 `.worktrees/<任务id>/hotspot`），compose 项目名会与主工作树同名，
直接 `up` 会**接管主实例**：容器标签/挂载指向主工作树，worktree 改动不会生效。
此时必须显式指定独立项目名并重映射宿主端口，两套环境可同时运行：

1. 在 worktree 根目录创建不入库的 override（`.workwork/docker-compose.worktree.yml`，
   `.workwork/` 已 gitignore），用 `!override` 替换全部宿主端口：

   ```yaml
   services:
     postgres:
       ports: !override
         - "127.0.0.1:5433:5432"
     seaweedfs:
       ports: !override
         - "127.0.0.1:8334:8333"
     redis:
       ports: !override
         - "127.0.0.1:6380:6379"
     backend:
       ports: !override
         - "127.0.0.1:8083:8081"
     backend-test:
       ports: !override
         - "127.0.0.1:8084:8081"
   ```

2. 启动（cwd 必须是该 worktree 根目录；`NGINX_PORT` 同时驱动 nginx 宿主端口与
   后端 `S3_PUBLIC_ENDPOINT`，必须一起改为不冲突端口）：

   ```bash
   NGINX_PORT=8889 docker compose -p hotspot-<worktree标识> \
     -f docker-compose.yml -f docker-compose.dev.yml \
     -f .workwork/docker-compose.worktree.yml up -d
   ```

   访问入口：http://localhost:8889；后端直连 127.0.0.1:8083、Postgres 5433、
   Redis 6380、SeaweedFS 8334。主实例（8888/8081/5432/6379/8333）不受影响。

3. 清理该 worktree 环境时同样带 `-p` 与 override（`-v` 只删除该项目独立 volume）：

   ```bash
   NGINX_PORT=8889 docker compose -p hotspot-<worktree标识> \
     -f docker-compose.yml -f docker-compose.dev.yml \
     -f .workwork/docker-compose.worktree.yml down -v
   ```

worktree 合并完成后，先清理其 compose 资源，再删除 worktree：

```bash
# 1. 停止并删除该 worktree 的容器、网络与数据卷（-v 删除 postgres_data 等 volume，仅该 worktree 独立数据，可安全删除）
docker compose -f docker-compose.yml -f docker-compose.dev.yml down -v

# 2. 删除 worktree 与分支（git 侧流程见全局 Git 准则）
git worktree remove .worktrees/<分支名>
git branch -d <分支名>
```

> 目录名同名冲突时禁止使用上面的默认清理命令（它作用于主实例项目，`-v` 会删除主实例
> 数据卷）；改用上文第 3 步的独立项目命令。

## 项目特有约束
- 契约权威：`spec/` 是 API 契约的唯一权威来源；修改接口、字段、错误码前先对齐 `spec/`。
- 跨端边界：距离与位置计算统一由后端完成，前端和客户端只展示结果。
- **int64 跨端序列化统一为字符串**：JS（Svelte）与 ArkTS 的 number 是 IEEE754 double，无法无损表示 Go int64（安全整数上限 2^53）。凡 spec 定义 string、后端为 int64 的字段（如 WS `server_seq`），后端 JSON tag 必须加 `,string` 下发字符串；前端/harmonyapp 类型保持 string 仅透传/去重、不做数值运算。禁止以「后端实际下发数字」为由把客户端类型改成 number——正确修法是后端序列化为字符串。
- Docker Compose 统一管理：所有服务（postgres、redis、seaweedfs、backend、frontend、nginx）统一通过 `docker compose` 编排启动/停止；不得混用宿主机进程与容器，也不允许单独启动某个服务绕过 compose。
- 更新代码的过程中发现的问题、错误，都应该修复掉，不能因为不是本次修改引起的就搁置
