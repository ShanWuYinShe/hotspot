# HotSpot - Agent Guide

> 面向 AI 编码工具的全局操作指南。保持变更契约安全、最小化、可回退。

## 项目概况

HotSpot 是一个基于地理位置的社交应用，采用 monorepo 结构。

| 目录 | 技术栈 | 说明 |
|------|--------|------|
| `spec/` | Markdown (.mdc) | API 契约与编码规范（**唯一真实来源**） |
| `backend/` | Rust / Axum / SQLx / PostGIS | 后端服务 |
| `frontend_svelte/` | SvelteKit / Svelte 5 / TypeScript | Web 前端 |
| `harmonyapp/` | HarmonyOS ArkTS (.ets) | 鸿蒙客户端 |
| `scripts/` | Shell | 守护脚本 |

## Spec-First 开发原则

**`spec/` 是所有 API 和编码规范的唯一权威来源。** 任何子项目的实现必须严格遵循 spec。

### 权威性优先级

```
spec/ > 实现代码 > 注释 > 口头描述
```

- 实现与 spec 冲突时，**以 spec 为准**，修改实现代码。
- 不确定某个字段、路径或行为时，**先查阅对应 spec 文件**，禁止猜测。
- 禁止在实现中引入 spec 未定义的端点、字段或行为。

### 规范文件索引

编码规范和 API 契约已有独立文件，**不要在本文件中重复其内容**，按需引用：

| 文件 | 内容 |
|------|------|
| `spec/api_spec_index.mdc` | API 契约总索引（查接口从这里开始） |
| `spec/api_spec_base.mdc` | 全局规则（分页/幂等/错误信封/Token） |
| `spec/api_spec_types.mdc` | 共享类型定义（跨模块复用类型） |
| `spec/api_spec_errors.mdc` | 错误码（唯一错误码来源） |
| `spec/api_spec_websocket.mdc` | WebSocket 事件契约 |
| `spec/backend_rule.mdc` | 后端编码规范 |
| `spec/frontend_rule.mdc` | 前端编码规范 |
| `spec/backend_doc.mdc` | 后端功能边界文档 |
| `spec/frontend_doc.mdc` | 前端功能边界文档 |
| `spec/api_spec_*.mdc` | 各业务模块 API 契约 |

### 查找接口的正确流程

1. 从 `spec/api_spec_index.mdc` 查找对应模块。
2. 打开对应模块 spec（如 `api_spec_auth.mdc`）查看端点定义。
3. 查 `api_spec_types.mdc` 确认共享类型。
4. 查 `api_spec_errors.mdc` 确认错误码。

## 构建命令

### Backend

```bash
cd backend
cargo check          # 类型检查（最常用，优先使用）
cargo build --release
cargo fmt            # 格式化
cargo clippy         # lint
cargo test           # 全部测试
cargo test <name>    # 单个测试
```

### Frontend (SvelteKit)

```bash
cd frontend_svelte
pnpm install
pnpm check           # 类型检查（最常用，优先使用）
pnpm build
pnpm dev
```

### HarmonyOS

DevEco Studio 默认安装路径（macOS）：`/Applications/DevEco-Studio.app/`

```bash
# macOS 默认路径（直接使用，无需询问）
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk"
HVIGORW="/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js"

# 非 macOS / 自定义路径
DEVECO_SDK_HOME="$DEVECO_STUDIO_DIR/Contents/sdk"  # macOS
DEVECO_SDK_HOME="$DEVECO_STUDIO_DIR/sdk"           # 其他平台
HVIGORW="$DEVECO_STUDIO_DIR/Contents/tools/hvigor/bin/hvigorw.js"  # macOS
HVIGORW="$DEVECO_STUDIO_DIR/tools/hvigor/bin/hvigorw.js"          # 其他平台

# 编译
cd harmonyapp
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" node "$HVIGORW" \
  clean --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --enable-build-script-type-check --no-daemon
```

## 跨项目关键规则

以下规则适用于所有子项目，违反任何一条均视为错误：

### 协议与路径

- 仅 POST，RPC + JSON，不使用 REST 风格。
- 统一前缀 `/api/v1/`。
- 所有时间使用 UTC ISO 8601。
- 所有 ID 命名为 `xxx_id`，类型统一为 `string`。
- `*_url` 必须为完整绝对 URL。

### 分页

- 只使用游标分页，**禁止 OFFSET**。
- `cursor` 对客户端不透明，仅透传，不解析。
- 默认 `limit` 20，最大 100。

### 幂等

- MUST 幂等接口必须携带 `Idempotency-Key`（详见 `api_spec_base.mdc`）。
- 幂等键格式：`{client_id}-{timestamp_ms}-{random}`，32–64 字符。

### 位置与距离

- 距离和方位由后端 PostGIS 计算，**客户端禁止本地计算距离或角度**。
- 请求用 `Location`，响应用 `RelativeLocation`。

### 错误处理

- 业务响应统一 HTTP 200 + `ApiResponse<T>`。
- 业务成功仅看 `code == 0`，不依赖 HTTP 状态码或 `data` 有值。
- 错误码只能使用 `spec/api_spec_errors.mdc` 中已定义的，禁止自行编号。
- 禁止向客户端暴露堆栈、SQL、内部路径。

## 任务完成验证流程

> Agent 在声明任务完成前，**必须**按以下流程逐步验证。跳过任何一步等于任务未完成。

### 步骤一：编译验证

对**涉及改动的项目**运行编译，必须零错误：

```bash
cd backend && cargo check
cd frontend_svelte && pnpm check
cd harmonyapp && DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" \
  node "/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js" \
  clean --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --enable-build-script-type-check --no-daemon
```

- 编译警告应尽量消除，但不得用 `#[allow(...)]` 等属性压制。
- 如果编译失败，修复后重新运行直到通过。

### 步骤二：禁止模式扫描

对涉及改动的项目运行以下检查，结果必须**零匹配**：

```bash
# 禁止非宏 sqlx 查询（后端）
rg 'sqlx::query\(' backend/src/
rg 'sqlx::query_as::<' backend/src/
rg 'sqlx::query_scalar::<' backend/src/
rg 'sqlx::query_as_unchecked!' backend/src/

# 禁止硬编码 API 路径（前端 + 鸿蒙）
rg '"/api/v1/' frontend_svelte/src/ harmonyapp/entry/src/main/ets/
```

如果有匹配，修复后重新扫描。

### 步骤三：规范对齐

核对变更涉及的内容是否与 `spec/` 一致：
- 端点路径、字段名、类型是否匹配。
- 认证级别（token_optional / token_required）是否正确。
- MUST 幂等端点是否携带 `Idempotency-Key`。
- 游标分页是否正确透传 `next_cursor`。

### 步骤四：跨项目回归

- DTO/类型/WebSocket 事件变更须同步所有消费方（backend、frontend_svelte、harmonyapp）。
- 对受影响的其他项目重复步骤一。

## 权限边界

### 直接执行（无需询问）

- 读取文件、搜索代码、列出目录。
- 运行编译检查（`cargo check`、`pnpm check`）、格式化、lint、单个测试。
- 运行「禁止模式扫描」中的 rg 命令。
- 查阅 `spec/` 目录下的规范文件。

### 必须先询问

- 安装/升级依赖。
- 删除文件或目录。
- 任何 git 写操作（见下节）。
- 运行完整测试套件或 e2e 测试。
- 修改 CI/CD 配置、部署脚本、环境变量模板。
- 变更数据库 migration 文件。

### 绝对禁止

- 修改 `spec/` 下的任何文件（除非用户明确要求修改规范）。
- 编造 spec 中不存在的 API 端点或字段。
- 在任何项目中硬编码 API 路径字符串 `"/api/v1/..."`。
- 在代码中存储密码、Token、API Key 等敏感信息。
- 修改 `.env`、`credentials`、密钥文件等。

## Git 安全

- 默认不主动执行任何 `git` 命令，除非用户要求。
- 仅在需要时允许只读命令：`git status`、`git diff`、`git log`、`git show`。
- 未经批准禁止写操作：`git add`、`git commit`、`git push`、`git pull`、`git merge`、`git rebase`、`git cherry-pick`、`git revert`、`git reset`、`git checkout`、`git restore`、`git clean`。
- 未经用户明确要求禁止 `--force`、`--hard`、`--no-verify`。

## Agent 行为准则

- 对用户的建议不能无脑同意，要认真考虑实际情况，参照业内最佳实践，比较后说明结果和原因。
- 实现代码时应拆分成任务，使用 todo 列表记录进度，完成后同步更新状态。
- 生成的辅助临时脚本和 md 文件放在 `/tmp` 目录，完成后删除，不放在代码目录。
- 禁止使用 `head` 或 `tail` 命令读取文件。
- 文档不使用数字章节编号。
- 不要自动更新文档版本号。
- 阅读文档时注意文档的版本信息。
