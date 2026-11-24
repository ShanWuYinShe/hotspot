# Release & Hotfix Runbook

本文件是**发布、回滚、热修复**的唯一操作手册。版本号约定与发布门禁的硬性要求同时写在根 `AGENTS.md`「版本号与发布」；本文提供完整流程、命令与判断分支。

相关脚本：

| 文件 | 作用 |
|------|------|
| `scripts/deploy-production.sh` | 部署任意已发布 tag（下载 Release 镜像 → load → 备份 → 滚动更新 → 校验） |
| `scripts/release-smoke.sh` | 发版后线上冒烟（版本三处对齐 / 前端入口 / Release 资产齐全与镜像包校验和） |
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

全部通过才允许 `git tag`。镜像构建与容器内验证由 CI 承担，本地不构建生产镜像：

1. **改动已过 PR 门禁**：PR 上 CI 全绿——`Go Backend (Unit Tests)` 与 `Go Backend (Postgres Integration)` 覆盖单测、`-race`、覆盖率、迁移幂等/manifest、真实库 IT 与 HTTP E2E；`Docker Build + Runtime Smoke (Prod Images)` 构建并运行生产镜像（启动迁移 + manifest 校验、`/readyz`/`/healthz`、OCI version 指纹）。
2. **`origin/main` CI 全绿**：`gh run list --branch main --limit 1 --json conclusion` 为 `success`（Release 的 `require-ci` job 会在 tag 后再次强制断言该提交 CI 全绿）。
3. **鸿蒙端**：涉及 `harmonyapp/` 时人工确认本机 hvigor 构建通过（CI 不覆盖）。

需要本地联调或复现 CI 失败时（不是打 tag 的硬性前置）：`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d` 起 dev 环境（入口 `http://localhost:8888`）；全量后端检查本地复现用 `./scripts/verify-backend-ci.sh --with-db`（见 `backend-go/AGENTS.md`「本地复现 CI」）。

例外：纯文档/注释改动不进镜像，无额外本地验证要求，仍以 CI 全绿为准。

---

## 2. 标准发布

```bash
# 1) 代码已合入 main（worktree 内开发 → 推分支 → PR → CI 全绿后 rebase 合并）
git push                                   # 先推分支

# 2) 在 main 当前提交打 annotated tag，一个一个推
git tag -a v1.2.0 -m "v1.2.0"
git show v1.2.0 --stat                     # 核对 tag 指向
git push origin v1.2.0

# 3) 确认 Release 工作流真的跑了（为什么必须逐个推见 git-workflow skill）
gh run list --workflow=release.yml --branch v1.2.0 --limit 1
gh run watch <run-id> --exit-status

# 4) 部署：tag 即上线（Release 工作流 auto-deploy job 经 SSH 自动触发，
#    与手动执行同一条脚本链路）。手动兜底（CI 失败/密钥失效时）：
./scripts/deploy-production.sh v1.2.0
```

部署脚本内部按序执行：

- **加锁**：服务器 `~/hotspot/.deploy.lock` 原子 mkdir，占锁失败即中止（TTL 6h 后可接管陈旧锁）；
- **校验发布包**：Release 的 `.sha256` 资产（旧版本无此资产时告警放行）；加载后若存在 `.ids.txt`（镜像身份清单）则逐一核对 `docker inspect .Id`，锁定「发布包内容 == CI 构建产物」。清单同时记录 config digest 与 OCI manifest digest：`docker inspect .Id` 的语义取决于镜像存储后端（经典 graph driver 为 config digest，containerd 存储如 Docker 29 为 manifest digest），任一匹配即通过；
- **配置同源**：compose / nginx 片段从**被部署的 tag** 用 `git archive` 导出，不取工作树内容（否则本地 HEAD 领先 tag 时会把未发布的配置带上线）；
- **配置生效校验**：nginx 配置是 file bind-mount（容器创建时锁定 inode），同步必须**原位写入**（`cat >`）——`rm+mv` 换掉宿主 inode 后，运行中的容器仍读旧文件，`nginx -s reload` 会静默不生效（2026-11-07 实测线上 nginx 配置自 11/04 起与仓库漂移）。同步后在动镜像前逐文件比对 conf.d 片段「容器内配置 sha256 == tag 内配置」，不一致立即中止并提示重建 nginx 容器（若本次发布**新增**了 nginx 挂载文件，旧容器尚未挂载该文件也会触发该提示：先重建 nginx 容器——此时宿主机 compose 已是 tag 版本——再重跑部署）；切流后再复核主配置 `/etc/nginx/nginx.conf`（挂在 compose 的 file bind-mount 上，`compose up` 因挂载变化重建 nginx 后生效）；
- **核对镜像指纹**：`version` 必须等于 tag、`revision` 必须等于该 tag 的 commit；
- **数据库快照**：`pg_dump -Fc` 到服务器 `backups/` + 本机 `.workwork/`，失败即中止；
- **独立迁移 job**：滚动更新前先在服务器跑一次性容器
  `IMAGE_TAG=<tag> docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile migrate run --rm -T migrate`
  （同镜像内 `./migrate` 二进制，幂等：pending 应用 + checksum/pending/schema manifest 校验）；
  失败直接中止（此时 backend 尚未切换），不把迁移失败留给新容器 crash-loop；
- **滚动更新**：`IMAGE_TAG=<tag> compose --profile prod up -d --scale frontend-build=0` + `nginx -s reload`（软重载，不中断在途 HTTP/WS 连接）；容器 `stop_grace_period: 30s` 覆盖应用侧排水预算；
- **三级版本校验**：后端 health 版本、前端静态卷 `version.json`、**nginx 容器内**真实响应（`/version.json` 与 `/healthz`）——容器里的文件对不等于网关服务得出去；
- **就绪探针口径**：容器 healthcheck 用 `GET /readyz`——数据库不可达返回 503（实例摘流量），Redis 不可达仅 `degraded`（200，限流/幂等 fail-open 仍可服务）；`GET /healthz` 保留为纯存活探针。
- **可观测性（可选）**：服务器 `.env` 设 `OBSERVABILITY_ENABLED=1` 时随本次部署拉起/更新 prometheus + grafana（采集栈与业务同挂 `hotspot_internal` 网络）；未启用则仅分发采集配置、不常驻。Prometheus firing alerts 由 `health-check.sh` 巡检经 `ALERT_WEBHOOK` 外发（最坏延迟=巡检周期，无独立 Alertmanager）；
- **CI 机器门禁（合入前）**：`docker-build` job 现在会**运行**生产镜像（非 root、启动迁移 + manifest 校验、`/readyz`/`/healthz`、OCI version 指纹）；`backend-integration` job 额外校验迁移幂等 + `-verify-only` + schema manifest 索引一致性（见 `.github/workflows/ci.yml`）；
- **失败自动回滚**：任一级校验失败 → 用上一版本镜像重新 `up -d` + 告警（数据不回滚）；
- **审计**：追加一行到服务器 `backups/deploy-history.log` 与本机 `.workwork/deploy-history.log`；
- **旧镜像回收**：保留最近 `DEPLOY_KEEP_IMAGES`（默认 3）个历史版本 + 当前/上一版本。

### 自动发版节奏（AI 自迭代期）

自迭代期间的默认发版口径（人也可以照此执行）：

- **触发条件**（满足即发布，不必累积过多）：main 上已合并 ≥1 批用户可见改动（新功能=minor / 仅修复=patch）且 **main CI 全绿**、涉及 `harmonyapp/` 时本机 `assembleApp` 通过。
- **发布内容一次到位**：后端 + 前端（tag → Release 工作流构建镜像并 auto-deploy 生产）+ **Android APK（同一 Release 附件）**。
- **执行顺序**（固定，不要跳步）：
  1. 发版 PR：把 `androidapp/app/build.gradle.kts` 默认 `versionName/versionCode` 对齐新版本（versionCode 每次 +1）；
  2. 合并后等 main CI 全绿；涉及鸿蒙改动时本机确认 hvigor 构建；
  3. `git tag -a vX.Y.Z -m "vX.Y.Z"` 逐个推送 → 等 Release 工作流（require-ci / build-push-release / auto-deploy）全绿；
  4. 构建并附 APK：`scripts/release-android-apk.sh vX.Y.Z <versionCode>`（校验 tag/Release/签名/版本后上传）；
  5. 跑发版后冒烟：`./scripts/release-smoke.sh --version vX.Y.Z`（一次核对 version.json / healthz / tag commit 三处一致、前端入口可达、Release 含镜像包 + `.sha256` + `.ids.txt` + APK 且镜像包校验和正确）。
- **APK 版本号规则不变**：`versionCode` 必须递增（否则无法覆盖安装），`versionName` 与 tag 一致。

### Android APK 分发（附到 Release）

Android 端不进 CI 发版链路（CI 只跑 debug 门禁），APK 由本机签名后上传到对应 GitHub Release。
Release 跑完后一条命令完成「构建 + 签名校验 + 版本校验 + 上传」：

```bash
scripts/release-android-apk.sh v1.14.0 5   # 参数：tag（vX.Y.Z）+ versionCode（递增整数）
```

脚本会拒绝：tag 不在 origin/main、Release 尚未生成、keystore/钥匙串口令缺失、APK 版本号与入参不一致等情况（避免发错包/发 debug 包）。
以下为手动兜底细节。签名材料**不入库、不打印、不配 GitHub secret**：

- keystore：`~/.android/hotspot-release.jks`（alias `hotspot`，RSA 4096 / 10000 天）；
  口令存 macOS 钥匙串（`security find-generic-password -a <用户名> -s hotspot-android-release -w`）。
  **keystore 丢失后无法覆盖升级安装**，需另行离线备份，禁止提交到仓库或 CI。
- 构建（版本号通过环境变量注入，默认已与当前 tag 对齐）：

```bash
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
export ANDROID_HOME="$HOME/Library/Android/sdk"
PASS=$(security find-generic-password -a "$USER" -s hotspot-android-release -w)
HOTSPOT_KEYSTORE_FILE="$HOME/.android/hotspot-release.jks" \
HOTSPOT_KEYSTORE_PASSWORD="$PASS" HOTSPOT_KEY_ALIAS=hotspot HOTSPOT_KEY_PASSWORD="$PASS" \
HOTSPOT_VERSION_NAME=vX.Y.Z HOTSPOT_VERSION_CODE=N \
cd androidapp && ./gradlew --no-daemon assembleRelease
# N = 上一次 versionCode + 1（递增，勿重复）
```

- 校验签名与版本：

```bash
"$ANDROID_HOME/build-tools/37.0.0/apksigner" verify --print-certs \
  androidapp/app/build/outputs/apk/release/app-release.apk
"$ANDROID_HOME/build-tools/37.0.0/aapt2" dump badging \
  androidapp/app/build/outputs/apk/release/app-release.apk | head -2
```

- 本地安装冒烟（可选，不进入发版硬门禁）：

```bash
# debug 与 release 签名不同，覆盖安装前先卸载旧包，否则会 INSTALL_FAILED_UPDATE_INCOMPATIBLE
adb uninstall com.hotspot.app || true
adb install androidapp/app/build/outputs/apk/release/app-release.apk
adb shell am start -W -n com.hotspot.app/.MainActivity
adb shell dumpsys package com.hotspot.app | grep -E 'versionCode|versionName'
# release 不可调试：应输出 package not debuggable
adb shell run-as com.hotspot.app pwd
```

- 附到 Release（tag 的 Release 工作流跑完后）：

```bash
gh release upload vX.Y.Z \
  androidapp/app/build/outputs/apk/release/app-release.apk#hotspot-android-vX.Y.Z.apk --clobber
gh release view vX.Y.Z --json assets --jq '.assets[].name'
```

- 版本规则：`versionCode` 必须递增（否则无法覆盖安装），`versionName` 与 tag 一致；
  两者默认值写在 `androidapp/app/build.gradle.kts`（随每次发版更新），发新版用 `HOTSPOT_VERSION_NAME`/`HOTSPOT_VERSION_CODE` 覆盖即可。

### 部署后自查

```bash
# 一键发版后冒烟（推荐：版本三处对齐 + 前端入口 + Release 资产/校验和，任一失败 exit 1）
./scripts/release-smoke.sh --version vX.Y.Z

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

**回滚只回滚镜像，不回滚数据**：迁移无 down 迁移（见 `hotspot-backend-migrations` skill）。schema 变更已不再强制「旧镜像兼容」——项目明确不考虑二进制向后兼容，迁移允许破坏性 DDL（文件内 `-- allow-destructive: <理由>` 声明）。因此：**普通增量迁移**回滚后是「旧镜像 + 新 schema」，通常可用；**破坏性迁移**（删表/列/类型变更）发布后旧镜像无法跑在新 schema 上，回滚必须按 `docs/backup-restore.md` 恢复数据库到发布前备份。打 tag 前以 main CI 全绿为准：`backend-integration` job 已校验迁移幂等与 manifest 一致性、`docker-build` job 已运行生产镜像完成迁移与就绪校验，本地不再构建新镜像验证。

**迁移执行方式**：生产部署先运行独立迁移 job（见 §2），server 启动期仅幂等校验并在缺失时补跑。两者共用同一 runner：按 `schema_migrations` 版本表增量执行迁移（只跑未登记文件）；每个文件与其版本登记在同一事务内提交，已应用文件以 sha256 checksum 冻结（历史迁移被改动/删除会拒绝启动）；启动末尾用 `migrations/schema_manifest.json` 校验期望表/列/索引是否齐全。**不做基线、不做存量库接管**：若目标库「已有业务表但版本表为空」，或已登记迁移缺少 checksum，启动直接失败；迁移只从空库按文件顺序全量执行，旧库需将数据迁移到新库。迁移只读校验可用 `go run ./cmd/migrate -verify-only`。

### 3.4 手工回滚（脚本不可用时）

```bash
ssh workwork-server
cd ~/hotspot
IMAGE_TAG=v1.1.10 docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod up -d --scale frontend-build=0
docker exec hotspot-nginx-1 nginx -s reload
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
git worktree add -b hotfix/<缺陷名> .agents/worktrees/hotfix-<缺陷名> v1.1.10

# 3) 最小修复 + 按 $1 门禁验证（改动小不等于可以跳过验证）

# 4) 推分支 → 开 PR → CI 全绿后 rebase 合并进 main（见 github-ci-flow skill）
git push -u origin hotfix/<缺陷名>
gh pr create --draft --base main --head hotfix/<缺陷名> --title "<中文标题>" --body-file /tmp/pr-body.md
gh pr checks <PR号> --watch
gh pr ready <PR号> && gh pr merge <PR号> --rebase --delete-branch

# 6) 打 patch tag（打在 main 当前提交上，一个一个推）
git tag -a v1.1.11 -m "v1.1.11 热修：<一句话缺陷描述>"
git push origin v1.1.11
gh run list --workflow=release.yml --branch v1.1.11 --limit 1   # 确认触发并成功

# 7) 部署
./scripts/deploy-production.sh v1.1.11
```

### 4.4 热修特有注意点

- **迁移**：热修允许常规增量迁移；破坏性迁移（删表/列/类型变更）意味着回滚需要数据库备份恢复（见上一条），热修场景下除非确有数据修复必要、且已确认可接受回滚代价，否则避免夹带破坏性 DDL。
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
| Release 已创建但资产不全 / auto-deploy 被跳过 | Release 页缺 `.ids.txt` 或镜像包；部署历史无该版本 | 镜像 Id 清单可按服务器已加载镜像 `docker inspect` 重建后 `gh release upload` 补齐；镜像包完整时用 `./scripts/deploy-production.sh <tag>` 手动兜底（同一条部署链路），镜像包缺失则升 patch 版本重发 |
| 部署报「加载后 Id 与发布清单不符」 | 加载后 `docker inspect .Id` 与 `.ids.txt` 不一致，但 tar sha256 与 OCI 标签均正确 | 多为镜像存储后端语义差异：CI 写 config digest、Docker 29 containerd 写 OCI manifest digest。用镜像包 `index.json` 的 manifest digest 重建 `.ids.txt` 后 `gh release upload --clobber` 重传，再重跑 auto-deploy；**auto-deploy 用 tag 内旧脚本时只能写单列 manifest digest**（旧脚本把续行当整串比较），新版脚本兼容双列 |
| 部署提示 tag 不在 main 上 | 前置校验失败 | 该 tag 打在未合并提交上；合并进 main 后重打版本 |
| 部署提示锁被占用 | `另一个部署正在进行` | 确认无并发部署后 `ssh <server> 'rm -rf ~/hotspot/.deploy.lock'`；陈旧锁（>6h）脚本会自行接管 |
| 部署卡在版本校验 | 三级校验之一不通过 | 脚本已自动回滚 + 告警；查 `docker logs <backend>`、`docker exec <nginx> wget -qO- http://127.0.0.1/version.json` |
| 部署后版本对但页面异常 | 边缘/缓存问题 | 查 nginx 日志与 Cloudflare 缓存；`curl /version.json` 核对边缘实际产物 |
| 部署成功但网关行为/样式仍旧（配置未生效） | `docker exec <nginx> sha256sum /etc/nginx/conf.d/default.conf` 与 tag 内文件不一致；nginx 容器启动时间早于本次部署 | file bind-mount inode 漂移（旧脚本用 `rm+mv` 换掉了宿主 inode）。先 `docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --no-deps --force-recreate nginx` 重建容器（数秒中断），再重跑部署；新版脚本已原位写入并在动镜像前 fail-fast |
| 需回滚但上一版本镜像已回收 | `docker inspect` 找不到镜像 | 用 `rollback-production.sh`（会重新下载并 load），或临时调大 `DEPLOY_KEEP_IMAGES` |

---

## 6. 环境变量速查

| 变量 | 默认 | 说明 |
|------|------|------|
| `SERVER` | `workwork-server` | 部署目标 ssh 别名 |
| `REMOTE_DIR` | `~/hotspot` | 服务器部署目录 |
| `ALERT_WEBHOOK` | 空 | 接收 `POST {"text":...}` 的机器人地址（部署失败/回滚、巡检失败、Prometheus firing alerts 告警） |
| `OBSERVABILITY_ENABLED` | 空（关闭） | 服务器 `.env` 设 `1` 时部署脚本用 `--profile observability` 拉起/更新 prometheus + grafana |
| `DEPLOY_KEEP_IMAGES` | `3` | 服务器保留的历史镜像版本数，`0` = 不清理 |
| `DEPLOY_LOCK_TTL_HOURS` | `6` | 并发锁陈旧判定阈值 |
| `ROLLBACK_OF` | 空 | 由 `rollback-production.sh` 注入，仅影响审计与告警文案 |
| `DEPLOY_DB_USER` / `DEPLOY_DB_NAME` | `postgres` / `hotspot` | 备份使用的库配置 |
| `DEPLOY_FRONTEND_VOLUME` | `hotspot_frontend_build` | 前端静态卷名 |
## 7. 部署架构评估（蓝绿 / 供应链签名）

**蓝绿部署（暂不实施）**：当前「一次性迁移 job + 滚动重建 + nginx 软重载」在
单实例规模下已经做到迁移失败前置暴露、回滚网与版本三级校验；蓝绿需要双
backend 副本 + nginx upstream 切换 + 迁移兼容窗口评估，收益主要在「秒级零
停机」而本项目当前流量/发布频率不足以覆盖其复杂度。触发条件：需要发布期间
零停机 SLA、或单实例发布窗口成为用户可感知事件时再实施（届时迁移必须保持
向前兼容，独立 migrate job 可直接复用）。

**供应链签名（部分实施）**：当前已具备 tar sha256、镜像身份清单（.ids.txt，config + OCI manifest 双摘要）、
OCI version/revision/created 标签与 GH Release 资产来源；下一步 cosign keyless
签名（GitHub OIDC）需要额外证书透明与验签步骤（部署端安装 cosign 并在加载前
验签），收益是把「资产来源可信」从 GitHub 账户安全提升到签名身份级别。待
发布链路稳定后再评估引入，不在本轮引入额外工具链。
