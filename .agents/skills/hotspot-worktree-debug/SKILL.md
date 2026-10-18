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


## worktree 重映射端口下跑 E2E 集成测试（NGINX_PORT 契约）

- 媒体两个 E2E 用例（TestMedia_E2E_PresignUploadComplete / Batch）的客户端入口
  origin **跟随 NGINX_PORT 环境变量**（默认 8888，backend-go/test/integration
  docker_test.go mediaPresignClientOrigin，2026-10-03 定稿；origin 固定的旧方案
  已废——763e3fa2 修好 origin 跟随后，固定 8888 会把 PUT 打进默认端口那套栈，
  complete_upload 在本栈查不到对象报 10400）。
- 重映射起栈（NGINX_PORT=8889 docker compose up …）后跑测试带**同名变量**即可，
  无需再 -skip：
  ```bash
  NGINX_PORT=8889 BACKEND_URL=http://localhost:8084 \
  POSTGRES_DSN="postgres://postgres:<PASSWORD>@127.0.0.1:5433/hotspot_test?sslmode=disable" \
  go test -count=1 ./test/integration/   # cwd = backend-go；PASSWORD 从 worktree 自己的 .env 取，勿回显
  ```
- 真库 IT（internal/**/*_it_test.go，testdb.Open 读 POSTGRES_DSN）跑在同一
  hotspot_test 库上：库内有 E2E 累积数据，**IT 断言不能用绝对总数**，要用
  「播种前基线 + 增量」口径（参考 admin/repo_postgres_stats_it_test.go，2026-10-01）。

## worktree 内跑 Go 工具链（2026-10-14 起无需重定向缓存，含 pre-commit 门禁）

**`~/Library/Caches/go-build` 与 `~/go` 已由 DSH sandbox-extra-roots 插件放行（2026-10-14
逐项实测可写）**，worktree 内 `go build/vet/test`（含 `.husky/pre-commit` 钩子触发的门禁）
直接跑即可，不要再 `export GOCACHE/GOMODCACHE`。

历史坑（未配插件的机器仍适用，2026-09-25 实测）：用 `$PWD/.workwork/...` **相对路径**重定向时，
缓存落在该 worktree 内部（pre-commit 在 worktree 根执行，`$PWD` 即 worktree 根）；Go 模块
缓存解压目录默认只读（0444/0555），`git worktree remove` 报 Permission denied，目录删不掉。
当时正确做法——绝对路径指向主工作树共享缓存（门禁本身不降级，实测同日 build+vet+test 通过）：

```bash
export GOCACHE="$HOME/work/hotspot/.workwork/go-build"
export GOMODCACHE="$HOME/work/hotspot/.workwork/go-mod"
```

- `.husky/pre-commit` 不覆盖这两个变量、继承 shell 环境，未配插件的机器提交前 `export` 即可让
  门禁跑通。
- 已踩坑的 worktree 目录删除被挡时：
  `chmod -R u+w .worktrees/<分支名> && rm -rf .worktrees/<分支名>`
  （`git worktree list` 已注销、仅剩目录时无需再跑 `git worktree remove`）。


当 Worktree 分支代码完成验证并合并到 `main` 分支后，如果主开发环境容器正在运行（`http://localhost:8888`）：
- **必须主动帮开发者重启受影响的主容器**，避免因 Vite dev 进程缓存或静态代理未刷新导致开发者看不到最新效果：
  ```bash
  # 前端改动
  docker compose -f docker-compose.yml -f docker-compose.dev.yml restart frontend nginx
  # 后端改动（或全栈改动）
  docker compose -f docker-compose.yml -f docker-compose.dev.yml restart backend frontend nginx
  ```
- 重启后需通过 `docker compose ps` 确认容器恢复 `healthy` 状态，确保开发者刷新浏览器即可直接看到最新效果。

### 单文件 bind mount 编辑后：restart 不够，必须 force-recreate（2026-09-25 实测）

编辑已被单文件 bind mount 进容器的配置（如 `docker/nginx.dev.conf`）后，文件 inode 变化，
容器内挂载点悬空（`docker exec <容器> nginx -t` 报 open failed）——**`docker compose restart nginx`
不会重做挂载**，必须：

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --force-recreate nginx
```

重建后 `nginx -t` 与 `docker compose ps` healthy 再验。曾致 fixer 改完 `proxy_cookie_flags`
后反复验证失败，排查半天才定位到挂载悬空（教训来源：dev 网关剥凭证 Cookie Secure 修复）。


## worktree 内跑鸿蒙 hvigor（2026-10-18 实测，DSH 沙盒环境）

DevEco hvigor 默认把项目缓存写到 `~/.hvigor/project_caches/`，hvigorw wrapper 启动
阶段还会经 npm 拉 pnpm（写 `~/.npm`）——DSH workspace-write 沙盒下两者均 EPERM
（报错形如 `EPERM: operation not permitted, mkdir '/Users/liyou/.hvigor/...'`，
看似构建错误实为沙盒拦截）。按「重定向优先」用工具自身配置面解决，不提权：

```bash
# cwd = worktree 根；目录需先建好（.workwork/ 已 gitignore）
mkdir -p .workwork/hvigor-home .workwork/npm-cache
export HVIGOR_USER_HOME="$PWD/.workwork/hvigor-home"   # hvigor 缓存根（须绝对路径）
export npm_config_cache="$PWD/.workwork/npm-cache"     # wrapper 内 npm 缓存
# JAVA_HOME / DEVECO_SDK_HOME / PATH 按 harmonyapp/AGENTS.md「构建 & 测试」节照常 export
```

- **pre-commit 钩子内的 hvigor 走同一套变量**：钩子继承当前 shell 环境，commit 前
  先 `export` 上述变量再 `git commit`，harmonyapp 门禁即可通过（批次 6 提交实测
  TYPE CHECK SUCCESSFUL + BUILD SUCCESSFUL，门禁全绿）。
- worktree 清理时 `.workwork/` 随目录一并删除，无残留。
- 纪律同全局：这是②「工具自身配置面」方案，实测有效，不要再为此申请提权。
