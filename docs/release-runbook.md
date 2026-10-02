# Release & Hotfix Runbook

本文件是**发布、回滚、热修复**的唯一操作手册。版本号约定与发布门禁的硬性要求同时写在根 `AGENTS.md`「版本号与发布」；本文提供完整流程、命令与判断分支。

相关脚本：

| 文件 | 作用 |
|------|------|
| `scripts/deploy-production.sh` | 部署任意已发布 tag（下载 Release 镜像 → load → 备份 → 滚动更新 → 校验） |
| `scripts/rollback-production.sh` | 一键回滚（复用部署链路，审计记为 kind=rollback） |
| `.github/workflows/release.yml` | tag 触发：门禁校验 → 构建镜像 → 打包 + sha256 → 挂 GitHub Release |
| `.github/workflows/ci.yml` | main/PR 全量 CI（后端单测+集成、前端 check/test/build、compose/nginx 校验、gitleaks） |

---

## 0. 版本号约定

- 版本号**以 git tag 为源**（`vX.Y.Z`，全项目共用一个序列），不手改任何版本文件；`git describe --tags --always --dirty` 派生出人类可读版本串，完整 commit 做回溯锚点。
- SemVer 判定：破坏性改动升 major，新功能升 minor，缺陷修复升 patch。
- tag 形态必须是 `vX.Y.Z`（CI 与部署脚本都断言；带后缀的候选 tag 不会出包）。
- tag 必须打在 **origin/main 已包含的提交**上（CI 与部署脚本双重断言）。
- 发布过的 tag **永不删除、永不移动**；发错了就升版本号重发（见 $5）。
- 版本指纹贯穿四层产物（后端 health、前端 `/version.json`、镜像 OCI label、鸿蒙 `versionName`），对照表见 README「版本号管理与发布」。

---

## 1. 发布门禁（打 tag 之前，硬性）

顺序执行，全部通过才允许 `git tag`。完整命令见根 `AGENTS.md`「发布门禁」：

1. **联调**：`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d` 全员 healthy，在 `http://localhost:8888` 走通本次改动（前端改动补 `bun run --cwd frontend browser:qa`）。
2. **集成测试**（后端有改动时）：`--profile test up -d backend-test` + `BACKEND_URL=http://localhost:8082 go test -count=1 ./test/integration/`。
3. **生产镜像本地构建**（与 CI 同构，先跑等于把失败提前）：`GIT_VERSION=<候选> GIT_COMMIT=$(git rev-parse HEAD) GIT_BUILD_TIME=<UTC> IMAGE_TAG=<候选> docker compose --profile prod build backend frontend-build`。
4. **指纹核对**：`docker inspect hotspot-backend:<候选> hotspot-frontend:<候选>` 的 OCI `version` 必须等于候选版本、`revision` 必须等于本次 commit。
5. **鸿蒙端**：涉及 `harmonyapp/` 时人工确认本机 hvigor 构建通过（CI 不覆盖）。

例外：纯文档/注释改动可跳过 1、2；文档改动不进镜像时 3、4 也可跳过。

---

## 2. 标准发布

```bash
# 1) 代码已合入 main（worktree 内开发 → git merge --ff-only）
git push                                   # 先推分支

# 2) 在 main 当前提交打 annotated tag，一个一个推
git tag -a v1.2.0 -m "v1.2.0"
git show v1.2.0 --stat                     # 核对 tag 指向
git push origin v1.2.0

# 3) 确认 Release 工作流真的跑了（为什么必须逐个推见 git-workflow skill）
gh run list --workflow=release.yml --branch v1.2.0 --limit 1
gh run watch <run-id> --exit-status

# 4) 部署（脚本内部流程见下）
./scripts/deploy-production.sh v1.2.0
```

部署脚本内部按序执行：

- **加锁**：服务器 `~/hotspot/.deploy.lock` 原子 mkdir，占锁失败即中止（TTL 6h 后可接管陈旧锁）；
- **校验发布包**：Release 的 `.sha256` 资产（旧版本无此资产时告警放行）；
- **配置同源**：compose / nginx 片段从**被部署的 tag** 用 `git archive` 导出，不取工作树内容（否则本地 HEAD 领先 tag 时会把未发布的配置带上线）；
- **核对镜像指纹**：`version` 必须等于 tag、`revision` 必须等于该 tag 的 commit；
- **数据库快照**：`pg_dump -Fc` 到服务器 `backups/` + 本机 `.workwork/`，失败即中止；
- **滚动更新**：`IMAGE_TAG=<tag> compose --profile prod up -d --scale frontend-build=0` + 重启 nginx；
- **三级版本校验**：后端 health 版本、前端静态卷 `version.json`、**nginx 容器内**真实响应（`/version.json` 与 `/healthz`）——容器里的文件对不等于网关服务得出去；
- **失败自动回滚**：任一级校验失败 → 用上一版本镜像重新 `up -d` + 告警（数据不回滚）；
- **审计**：追加一行到服务器 `backups/deploy-history.log` 与本机 `.workwork/deploy-history.log`；
- **旧镜像回收**：保留最近 `DEPLOY_KEEP_IMAGES`（默认 3）个历史版本 + 当前/上一版本。

### 部署后自查

```bash
ssh workwork-server 'tail -5 ~/hotspot/backups/deploy-history.log'   # 审计
curl -s https://<线上域名>/version.json                              # 边缘实际版本
ssh workwork-server '~/hotspot/scripts/health-check.sh'              # 巡检（crontab 常态运行）
```

---

## 3. 回滚

### 3.1 一键回滚（推荐）

```bash
./scripts/rollback-production.sh --list              # 先看：当前版本 / 部署历史 / 可用 tag
./scripts/rollback-production.sh --previous          # 回滚到上一次成功部署的版本
./scripts/rollback-production.sh v1.1.9              # 或显式指定版本
./scripts/rollback-production.sh --previous --yes    # 跳过确认（紧急/自动化）
```

`--previous` 从服务器部署历史推断上一版本（排除当前版本），不靠人工回忆；目标版本未知或历史为空时要求显式指定。回滚走与发布完全相同的链路（含数据库快照、版本校验、失败自动回滚网），只是审计记为 `kind=rollback`。

### 3.2 自动回滚（无需人工）

部署脚本在版本校验失败时自动重部署上一版本并推送 `ALERT_WEBHOOK` 告警。

### 3.3 数据面（必读）

**回滚只回滚镜像，不回滚数据**：迁移无 down 迁移（见 `hotspot-backend-migrations` skill 的 expand-contract 纪律），回滚后线上是「旧镜像 + 新 schema」。绝大多数情况下这是安全的——迁移只加不删、旧二进制必须兼容新 schema 是硬性要求；一旦出现读写异常，按 `docs/backup-restore.md` 评估恢复。

### 3.4 手工回滚（脚本不可用时）

```bash
ssh workwork-server
cd ~/hotspot
IMAGE_TAG=v1.1.10 docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod up -d --scale frontend-build=0
docker restart hotspot-nginx-1
```

前提：该版本镜像**仍在服务器**（旧镜像回收默认保留 3 个历史版本）。已被回收时用 $3.1 的脚本（它会从 Release 重新下载并 load）。

---

## 4. 热修复（Hotfix）

### 4.1 何时用热修

线上阻断性缺陷（可用性、数据正确性、安全），需要**尽快**出一个最小补丁版本，而不是等 main 上其它未完成的改动一起发。

### 4.2 本项目的一条硬约束

**发布 tag 必须落在 `origin/main` 上**（`release.yml` 与 `deploy-production.sh` 双重断言）。所以不存在「从旧 tag 拉一条发布分支单独发版」的旁路——那样线上会跑着 main 里没有的代码，版本无法用主干回溯，这正是门禁要排除的状态。

结论：**热修也必须回到 main**。如果 main 上有不宜随热修一起发布的内容，先把它处理掉（延后 / revert），再发 patch；不要为了省事绕开门禁。

### 4.3 流程

```bash
# 1) 先止血（可选但常见）：线上正在出事且修复尚未就绪时，先回滚恢复服务
./scripts/rollback-production.sh --previous

# 2) 从被修复的版本拉热修分支（worktree 内开发，禁止在主工作树改代码）
git worktree add -b hotfix/<缺陷名> .worktrees/hotfix-<缺陷名> v1.1.10

# 3) 最小修复 + 按 $1 门禁验证（改动小不等于可以跳过验证）

# 4) 与 main 对齐（保持 ff-only 可达）
git rebase main

# 5) 合并进 main（主工作树执行）
git merge --ff-only hotfix/<缺陷名>
git push

# 6) 打 patch tag（打在 main 当前提交上，一个一个推）
git tag -a v1.1.11 -m "v1.1.11 热修：<一句话缺陷描述>"
git push origin v1.1.11
gh run list --workflow=release.yml --branch v1.1.11 --limit 1   # 确认触发并成功

# 7) 部署
./scripts/deploy-production.sh v1.1.11
```

### 4.4 热修特有注意点

- **迁移**：热修若带迁移，必须兼容**上一个已发布版本**的二进制（回滚网依赖这一点）；禁止在热修里做删除列、收紧约束这类需要停机协同的改动。
- **前端**：静态产物经静态卷切换上线，切换过程无白屏窗口（见 `frontend/Dockerfile` 的 CMD 说明）；已打开旧页面的用户在下一次导航时拿到新版本。
- **WebSocket**：后端重启会断开在线连接，客户端自动重连，无需额外操作。
- **验证**：热修同样要过 $1 门禁——「紧急」不是跳过验证的理由。
- **事后**：把根因修复与防回归测试补进 main（热修分支只做最小改动）。

---

## 5. 异常情形处置

| 情形 | 现象 | 处置 |
|------|------|------|
| **CI 长期红** | main 上 CI 连续失败（含纯文档提交） | 这是**发布链路瘫痪**：Release 的 `require-ci` 门禁会拒绝在 CI 未通过的提交上出包，等于再也发不了版。必须先修 CI，不许带红发版（2026-09-27~10-02 实测：连续 14 次全红期间所有 tag 都无法发布） |
| **gitleaks 命中** | CI Secret scan 报 leaks found | 先判断是真凭据还是测试 fixture：真凭据**必须轮换并实测失效**后才能把指纹登记进 `.gitleaksignore`（附轮换依据），禁止直接忽略；fixture 假值可直接登记。接待机器人账号口令的处置见 `reception-bot/scripts/rotate-credentials.sh` 与 `.gitleaksignore` 顶部注释 |
| tag 推了但 CI 没触发 | Release 页没有新版本 | 每个 tag 单独推送并逐个确认（批量推超过 3 个 tag 时 GitHub 不产生 push 事件）；漏触发时删本地与远程 tag 后重推 |
| Release 工作流失败 | Actions 红 | 修问题 → 升 patch 版本重新打 tag；**不要**移动已推送的 tag |
| 部署提示 tag 不在 main 上 | 前置校验失败 | 该 tag 打在未合并提交上；合并进 main 后重打版本 |
| 部署提示锁被占用 | `另一个部署正在进行` | 确认无并发部署后 `ssh <server> 'rm -rf ~/hotspot/.deploy.lock'`；陈旧锁（>6h）脚本会自行接管 |
| 部署卡在版本校验 | 三级校验之一不通过 | 脚本已自动回滚 + 告警；查 `docker logs <backend>`、`docker exec <nginx> wget -qO- http://127.0.0.1/version.json` |
| 部署后版本对但页面异常 | 边缘/缓存问题 | 查 nginx 日志与 Cloudflare 缓存；`curl /version.json` 核对边缘实际产物 |
| 需回滚但上一版本镜像已回收 | `docker inspect` 找不到镜像 | 用 `rollback-production.sh`（会重新下载并 load），或临时调大 `DEPLOY_KEEP_IMAGES` |

---

## 6. 环境变量速查

| 变量 | 默认 | 说明 |
|------|------|------|
| `SERVER` | `workwork-server` | 部署目标 ssh 别名 |
| `REMOTE_DIR` | `~/hotspot` | 服务器部署目录 |
| `ALERT_WEBHOOK` | 空 | 接收 `POST {"text":...}` 的机器人地址（部署失败/回滚告警） |
| `DEPLOY_KEEP_IMAGES` | `3` | 服务器保留的历史镜像版本数，`0` = 不清理 |
| `DEPLOY_LOCK_TTL_HOURS` | `6` | 并发锁陈旧判定阈值 |
| `ROLLBACK_OF` | 空 | 由 `rollback-production.sh` 注入，仅影响审计与告警文案 |
| `DEPLOY_DB_USER` / `DEPLOY_DB_NAME` | `postgres` / `hotspot` | 备份使用的库配置 |
| `DEPLOY_FRONTEND_VOLUME` | `hotspot_frontend_build` | 前端静态卷名 |