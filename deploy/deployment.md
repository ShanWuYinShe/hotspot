# HotSpot verify / preview / release 设计说明

## 目标

这套体系服务三个清晰目标：

1. **verify**：支持多 agent + 多 worktree 并行开发时的默认本地验证
2. **preview**：提供一个唯一的本地预演环境，验证待交付结果的完整入口行为
3. **release**：产出正式部署输入物，不直接运行环境

## 为什么这样拆

过去 `deploy/` 同时承载测试、本地 runtime 预演和部分看起来像正式部署的语义，导致：
- README 与 `deployment.md` 重复
- 脚本职责重叠
- runtime 也被设计成多 env，多余且复杂
- preview image 与正式部署 image 的语义混淆

现在改成：
- **verify 多实例**
- **preview 单实例**
- **release 只产物**

## 一、verify

### 语义
- 面向开发内环
- 默认验证当前工作树快照
- 包含未提交修改
- 支持多 worktree 并行

### 隔离模型
按 `env-name` 隔离：
- snapshot
- compose project
- cache
- lock
- 测试依赖容器

### 入口

```bash
./deploy/scripts/verify.sh --env-name feature-a
```

### 技术实现
- `compose/verify.yml`
- 固定 runner image：
  - `dockerfiles/backend-go.verify.Dockerfile`
  - `dockerfiles/frontend.verify.Dockerfile`
- 当前工作树通过 snapshot 注入容器执行测试

## 二、preview

### 语义
- 面向本地集成分支或候选结果
- 只有一个实例
- 验证 HTTPS + nginx + frontend + backend-go + websocket + MinIO
- 不再为多个 worktree 同时起多套 preview

### 入口

```bash
./deploy/scripts/preview-build.sh
./deploy/scripts/preview-up.sh
./deploy/scripts/preview-inspect.sh
```

### 输入
- 默认：当前工作树快照
- 可选：通过 `preview-build.sh --ref <git-ref>` 使用确定性 git 版本

### 运行拓扑
- nginx 为唯一宿主入口
- `/` -> frontend
- `/api/v1/` -> backend-go
- `/ws/v1` -> backend-go
- `/s3/` -> MinIO

### compose
- `compose/preview.yml`
- 固定 compose project：`hotspot-preview`
- 固定 preview image：
  - `hotspot/backend-go:preview`
  - `hotspot/frontend-svelte:preview`

### 数据语义
保留：
- 默认保留 postgres / redis / minio 数据
- 支持 `--clean` / `--clean-all`

## 三、release

### 语义
- 不运行环境
- 只生成正式部署输入物
- 必须基于确定性 git 版本

### 入口

```bash
./deploy/scripts/release-build.sh --ref HEAD
./deploy/scripts/release-build.sh --ref <sha-or-tag> --label release-xxx
```

### 输出
- backend image
- frontend image
- manifest.json

manifest 至少包含：
- ref
- resolved SHA
- release label
- backend image
- frontend image

### 为什么 release 不运行
- preview 负责行为验证
- release 负责交付确定性
- 这样能把“当前工作树验证”和“正式部署输入”彻底拆开

## 四、Cloudflare / HTTPS / 真实部署边界

当前本地 preview 环境通过 nginx + HTTPS 模拟最终入口，已经足够验证：
- Secure cookie
- 同域 API / ws / s3 入口
- nginx 代理行为

但它不等于完整正式部署。当前仓库没有闭环实现：
- 真实证书签发/续期
- Cloudflare API 自动化
- 源站 IP 信任链
- 远程发布编排

因此当前 `release-build.sh` 只负责**产出正式部署输入物**，不负责真正远端发布。

## 五、目录约定

```text
deploy/
  README.md
  deployment.md
  compose/
    verify.yml
    preview.yml
  dockerfiles/
    backend-go.verify.Dockerfile
    frontend.verify.Dockerfile
    backend-go.preview.Dockerfile / backend-go.release.Dockerfile
    frontend.preview.Dockerfile / frontend.release.Dockerfile
  scripts/
    common.sh
    verify.sh
    preview-build.sh
    preview-up.sh
    preview-down.sh
    preview-inspect.sh
    release-build.sh
  nginx/
    hotspot.preview.nginx.conf
```

## 六、必须保留的行为
- 本地优先开发与本地集成分支流程不变
- verify 默认继续走 Docker
- verify 默认继续验证当前工作树快照
- preview 保留 HTTPS + nginx + MinIO + inspect 链
- 数据保留 / 清理语义保留
- 多 worktree 并行隔离只属于 verify

## 七、验证要求

### verify
- 不同 worktree 使用不同 `env-name` 可以并行运行
- snapshot / compose project / cache / lock 不串
- backend-go 与 frontend 验证都能通过

### preview
- `preview-build.sh` 成功产出 preview image
- `preview-up.sh` 启动成功
- `preview-inspect.sh` 能通过：
  - `GET /`
  - `POST /api/v1/system/health`
  - `GET /s3/`

### release
- `release-build.sh --ref <sha>` 能稳定产出 release image 与 manifest
- release 结果不依赖当前未提交工作树
