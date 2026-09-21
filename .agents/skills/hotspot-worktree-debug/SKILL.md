---
name: hotspot-worktree-debug
description: hotspot 项目的 worktree 调试环境操作细则：compose 隔离、同名冲突时的端口重映射、合并后重启主容器。
  触发：在 hotspot worktree 内启动/清理 docker compose 环境；worktree 目录名与主工作树同名导致 compose 项目名冲突；
  合并分支后需要让主开发容器生效最新代码；删除 worktree 前清理其 compose 资源。
---

# hotspot worktree 调试与清理

本 skill 覆盖 worktree 内的隔离调试、同名冲突处理、以及合并后的容器重启。
主工作树开发环境本身的启停约定见项目 AGENTS.md「调试入口」。

worktree 内调试同样通过 Docker Compose，与主工作树自动隔离。隔离原理（项目名取目录名、`-p` 接管风险）与同名冲突通用处理见全局 skill `docker-workflow`「worktree 隔离」节；下方是本项目的具体端口、命令与约束：

- 在对应 worktree 根目录执行（`docker-compose.yml` 在 worktree 内同样存在），cwd 必须是该目录
- 默认端口映射（8888 网关 / 8081 后端直连 / 5432 postgres / 8333 seaweedfs / 6379 redis / 8025 mailpit）与主实例相同，同一时间只允许一套 dev 环境运行；需要并行时按下方「目录名同名冲突」节重映射端口


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
     mailpit:
       ports: !override
         - "127.0.0.1:8026:8025"
   ```

   mailpit 无 profile、宿主端口写死 8025，不重映射则 worktree `up -d` 直接端口冲突；
   observability profile（prometheus 9090 / grafana 3001）若启用，同理需一并重映射。

2. 启动（cwd 必须是该 worktree 根目录；`NGINX_PORT` 同时驱动 nginx 宿主端口与
   后端 `S3_PUBLIC_ENDPOINT`，必须一起改为不冲突端口）：

   ```bash
   NGINX_PORT=8889 docker compose -p hotspot-<worktree标识> \
     -f docker-compose.yml -f docker-compose.dev.yml \
     -f .workwork/docker-compose.worktree.yml up -d
   ```

   访问入口：http://localhost:8889；后端直连 127.0.0.1:8083、Postgres 5433、
   Redis 6380、SeaweedFS 8334。主实例（8888/8081/5432/6379/8333/8025）不受影响。

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


当 Worktree 分支代码完成验证并合并到 `main` 分支后，如果主开发环境容器正在运行（`http://localhost:8888`）：
- **必须主动帮开发者重启受影响的主容器**，避免因 Vite dev 进程缓存或静态代理未刷新导致开发者看不到最新效果：
  ```bash
  # 前端改动
  docker compose -f docker-compose.yml -f docker-compose.dev.yml restart frontend nginx
  # 后端改动（或全栈改动）
  docker compose -f docker-compose.yml -f docker-compose.dev.yml restart backend frontend nginx
  ```
- 重启后需通过 `docker compose ps` 确认容器恢复 `healthy` 状态，确保开发者刷新浏览器即可直接看到最新效果。
