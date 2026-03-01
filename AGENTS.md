# HotSpot - AI Agent 操作指南

> 本指南为 AI 编码工具在 HotSpot 项目中的全局操作规范。旨在确保所有代码变更安全、可回退、严格遵循 Spec-First 原则。**请在执行任何任务前仔细阅读并遵守。**

## 项目全景

HotSpot 是一个基于地理位置的社交应用，采用 Monorepo 结构管理多个子项目。

| 目录 | 技术栈 | 说明 |
|------|--------|------|
| `spec/` | Markdown (.md) | **唯一真实来源**：包含所有 API 契约、编码规范、错误码定义 |
| `backend/` | Rust / Axum / SQLx / PostGIS | 后端服务 |
| `frontend_svelte/` | SvelteKit / Svelte 5 / TypeScript | Web 前端 |
| `harmonyapp/` | HarmonyOS ArkTS (.ets) | 鸿蒙客户端 |
| `deploy/` | Shell / Docker / Nginx | 部署运维（Docker 编排、数据库脚本、Nginx 配置、部署文档） |

## 核心原则：Spec-First 开发

**`spec/` 目录是 API 和行为规范的唯一权威来源。** 所有子项目的实现必须无条件遵循。

### 权威性优先级

```
spec/ > 实现代码 > 注释 > 口头/文字描述
```

- **冲突处理**：实现与 spec 不符时，**必须修改实现以匹配 spec**，禁止反向修改 spec（除非用户明确要求）。
- **不确定性处理**：对字段、路径、行为有任何疑问，**必须首先查阅对应的 spec 文件**，严禁猜测或引入未定义内容。

### 规范文件索引

以下为关键规范文件，**不要在本文档中重复其内容**，按需引用：

| 文件名 | 核心内容 | 使用场景 |
|--------|----------|----------|
| `spec/api_spec_index.md` | API 模块索引 | 查找某个功能属于哪个模块 |
| `spec/api_spec_base.md` | 全局规则（分页/幂等/错误信封/Token） | 实现任何 API 前必须阅读 |
| `spec/api_spec_types.md` | 共享类型定义 | 定义 DTO、枚举等跨模块类型时引用 |
| `spec/api_spec_errors.md` | 错误码定义 | **唯一错误码来源**，禁止自行定义 |
| `spec/api_spec_websocket.md` | WebSocket 事件契约 | 实现或消费 WebSocket 时引用 |
| `spec/backend_rule.md` | 后端编码规范 | Rust 代码风格、数据库操作约束 |
| `spec/frontend_rule.md` | 前端编码规范 | Svelte/TypeScript 编码约束 |
| `spec/harmonyapp_rule.md` | 鸿蒙客户端编码规范 | ArkTS 编码约束、组件规范 |
| `spec/backend_doc.md` | 后端功能边界 | 哪些逻辑必须在后端实现（如距离计算） |
| `spec/frontend_doc.md` | 前端功能边界 | 前端不应实现的逻辑 |
| `spec/harmonyapp_doc.md` | 鸿蒙客户端功能边界 | 鸿蒙端功能范围与非目标 |
| `spec/api_spec_*.md` | 各业务模块 API | 各模块具体端点定义 |

### 查找 API 的标准流程

1.  **定位模块**：从 `spec/api_spec_index.md` 确定功能所属模块。
2.  **查阅端点**：打开对应模块 spec（如 `api_spec_auth.md`），查看请求/响应格式、认证要求、幂等性要求。
3.  **复用类型**：如需使用共享类型，查阅 `spec/api_spec_types.md`。
4.  **确认错误码**：若涉及错误处理，查阅 `spec/api_spec_errors.md` 获取标准错误码。

### 新增 API 端点的标准工作流

新增或修改一个 API 端点时，需按以下顺序逐项检查。各技术栈的详细文件清单见 `.cursor/rules/` 下的对应规则文件。

**通用步骤**：

- 确认 `spec/api_spec_*.md` 中端点定义存在且为最新
- 确认请求/响应字段、类型与 `spec/api_spec_types.md` 一致
- 确认错误码来自 `spec/api_spec_errors.md`
- 确认认证级别（token_optional / token_required）正确
- 确认幂等性要求（MUST 幂等的接口须支持 Idempotency-Key）

**后端**（详见 `.cursor/rules/backend.md`）：DTO → Handler → Service → Repository → Route

**前端**（详见 `.cursor/rules/frontend.md`）：endpoints.ts → services/*.ts → types/api.ts

**鸿蒙**（详见 `.cursor/rules/harmonyapp.md`）：ApiEndpoints.ets → ApiClient.ets → types/api.ets

**跨项目同步**：如果端点涉及新增或修改 DTO/类型，须同步更新所有消费方的类型定义，并对受影响项目重新运行类型检查。

## 核心原则：简洁至上

> **用最少的代码、最直接的方式解决问题。复杂度是成本，不是资产。**

### 基本信条

- **能用一个函数解决的，不要拆成三个文件。** 代码行数不是成就，解决问题才是。
- **先写能跑的直球代码，只在真正需要时才抽象。** 预测未来需求是徒劳的，YAGNI（You Aren't Gonna Need It）。
- **flat > nested，direct > indirect，concrete > abstract。** 嵌套层级和间接调用链每增加一层，认知负担就翻倍。

### 禁止的过度设计模式

以下模式在本项目中**视为代码异味**，除非有压倒性的理由并注释说明：

- **无意义的中间层**：只是透传调用、不包含任何逻辑的 wrapper 函数或 trait。如果一个函数只是调用另一个函数然后返回结果，它不应该存在。
- **为一个实现写 trait/interface**：只有一个实现的 trait 是噪音，不是抽象。等到真正需要第二个实现时再提取。
- **过度泛型化**：`where T: Into<String> + Clone + Send + Sync + 'static` 这种签名说明你在解决一个不存在的问题。先用具体类型，泛型等真正需要时再加。
- **Builder / Factory / Strategy / 各种 Pattern 滥用**：设计模式是工具不是信仰。一个 `new()` 加几个参数能搞定的事，不要搞 Builder；一个 `match` 能搞定的事，不要搞 Strategy。
- **层层嵌套的目录结构**：`src/services/user/impl/default/mod.rs` 这种路径说明架构出了问题。模块划分应该扁平、清晰。
- **DTO 到 DTO 的无意义转换**：如果两个结构体字段完全一样，它们就不应该同时存在。

### 判断标准

做架构决策时，问自己这三个问题：

- **删掉这层抽象，代码还能正常工作吗？** 如果能，删掉它。
- **半年后的维护者看到这段代码，能在 30 秒内理解意图吗？** 如果不能，简化它。
- **这个设计解决的是当前的真实问题，还是想象中的未来问题？** 如果是后者，砍掉它。

### 各技术栈的具体要求

- **Rust**：善用 enum + match，不要用 trait object 模拟 Java 的多态。优先组合而非继承（Rust 本身就鼓励这样做）。`impl` 块直接写方法，不要为了"可测试性"套一层没用的 trait。
- **TypeScript/Svelte**：用简单的函数和响应式变量，不要搞 class 继承体系。组件保持小而专注，props 直传，不要搞 context 层层注入（除非真的需要全局状态）。
- **ArkTS**：同理，组件逻辑直接写，不要模仿 Android 那套 ViewModel + Repository + UseCase 的俄罗斯套娃。
- **SQL**：能一条查询解决的不要拆成多条再在应用层拼装。善用 PostgreSQL 和 PostGIS 的能力，不要把数据库当哑巴存储用。

## 构建与验证命令

以下为各子项目的关键命令，用于日常开发和任务完成验证。

### Backend (Rust)

```bash
cd backend
cargo check          # 类型检查（最常用，优先使用）
cargo fmt            # 代码格式化
cargo clippy         # Lint 检查
cargo test           # 运行所有测试
cargo test <name>    # 运行单个测试
cargo build --release # 生产构建
```

### Frontend (SvelteKit)

```bash
cd frontend_svelte
pnpm install         # 安装依赖（需询问）
pnpm check           # TypeScript 类型检查（最常用，优先使用）
pnpm build           # 生产构建
pnpm dev             # 启动开发服务器
```

### HarmonyOS (ArkTS)

> **注意**：HarmonyOS 的编译和代码检查是两个独立工具，**必须分别运行**。

> **编译前暂存提示**：在运行 `hvigorw` 编译 HarmonyOS 项目之前，请先执行 `git add` 暂存当前改动，方便在出现意外时用 git 恢复。`ohpm` 与 `codelinter` 可直接运行。

```bash
# 环境变量（macOS 默认安装路径，无需修改）
DEVECO_STUDIO_DIR="/Applications/DevEco-Studio.app"
DEVECO_SDK_HOME="$DEVECO_STUDIO_DIR/Contents/sdk"
DEVECO_SDK_NODE="/Applications/DevEco-Studio.app/Contents/tools/node/bin/node"
OHPM="$DEVECO_STUDIO_DIR/Contents/tools/ohpm/bin/ohpm"
HVIGORW="$DEVECO_STUDIO_DIR/Contents/tools/hvigor/bin/hvigorw.js"
CODELINTER="$DEVECO_STUDIO_DIR/Contents/plugins/codelinter/run/index.js"

# 非 macOS 环境，请用户自行提供 DEVECO_STUDIO_DIR 路径

cd harmonyapp

# 0. 安装依赖（编译前必须先执行）
"$OHPM" install --all \
  --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true

# 1. 编译检查（零错误要求，执行前请先运行 `git add` 暂存当前改动）
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" "$DEVECO_SDK_NODE" "$HVIGORW" \
  --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --daemon

# 2. 代码风格检查（模拟 IDE 中的 Linter 警告，应尽量修复）【✅ Agent 可直接执行】
DEVECO_SDK_HOME="$DEVECO_SDK_HOME" "$DEVECO_SDK_NODE" "$CODELINTER" \
  -c code-linter.json5 entry/src/main/ets/
```

## 跨项目关键规则（必须遵守）

以下规则适用于所有子项目，**任何违反均视为错误**，需立即修正。

### 协议与路径

- 仅使用 **POST** 方法，RPC 风格 + JSON 请求体。**禁止使用 REST 风格**。
- 所有 API 路径必须以 `/api/v1/` 开头。
- 所有时间字段均使用 UTC ISO 8601 格式（如 `2023-10-05T14:48:00.000Z`）。
- 所有标识符字段必须命名为 `xxx_id`，类型统一为 `string`。
- 所有 URL 字段（如 `avatar_url`）必须为完整的绝对 URL。

### 分页

- 强制使用**游标分页**，**禁止使用 OFFSET/LIMIT** 进行分页。
- `cursor` 参数对客户端**完全不透明**，后端应将其作为不透明令牌传递，前端只透传，不解析、不构造。
- 默认每页数量 `limit` 为 20，最大不超过 100。
- 响应中必须返回 `next_cursor`，若无更多数据则省略该字段（JSON 中不包含该 key）。

### 幂等性

- 所有标记为 **MUST** 幂等的接口，客户端**必须**在请求头中携带 `Idempotency-Key`。
- 幂等键格式：`{client_id}-{timestamp_ms}-{random}`，长度 32–64 字符。后端需保证在指定时间内（如 24 小时）相同键仅处理一次。

### 位置与距离

- **距离和方位计算**：必须由后端基于 PostGIS 完成，**前端/客户端严禁自行计算距离或角度**。
- 请求位置使用 `Location` 类型（包含 `point: GeoPoint` 和 `location_name` 字段）。
- 响应位置使用 `RelativeLocation` 类型（包含 `point: GeoPoint`、`location_name`、`distance`、`bearing_degrees` 等字段，由后端填充）。

### 错误处理

- 所有业务响应均使用 HTTP 200 状态码，数据格式统一为 `ApiResponse<T>` 信封。
- **业务成功判断唯一依据**：`code == 0`。严禁依赖 HTTP 状态码或 `data` 字段是否为 `null`。
- 错误码**只能**从 `spec/api_spec_errors.md` 中选用，**禁止自行定义新错误码**。
- 严禁向客户端暴露任何内部细节（如堆栈跟踪、SQL 语句、文件路径）。

## 任务完成验证流程（强制性）

> **在声明任务完成前，必须按以下顺序逐一执行验证。跳过任何一步视为任务未完成。**

### 步骤一：编译/类型检查

对**所有涉及改动的项目**运行对应的检查命令，确保**零错误**。

- Backend: `cargo check`
- Frontend: `pnpm check`
- HarmonyOS: `codelinter` 和 `hvigorw`（编译前先 `git add`，详见下方 HarmonyOS 章节）。

**原则**：
- 警告应尽量消除，但**禁止使用 `#[allow(...)]` 等属性压制警告**（除非有充分理由并注释）。
- HarmonyOS 的两项检查（编译 + Linter）必须**分别执行**，缺一不可。


### 步骤二：禁止模式扫描

对涉及改动的项目运行以下检查，结果必须**零匹配**。若有匹配，修复后重新扫描。

```bash
# 后端：禁止非宏的 sqlx 查询（必须使用 sqlx::query! 等宏）
rg 'sqlx::query\(' backend/src/
rg 'sqlx::query_as::<' backend/src/
rg 'sqlx::query_scalar::<' backend/src/
rg 'sqlx::query_as_unchecked!' backend/src/

# 前端 + 鸿蒙：禁止硬编码 API 路径（必须从常量或配置中引用）
rg '"/api/v1/' frontend_svelte/src/ harmonyapp/entry/src/main/ets/
```

### 步骤三：规范对齐

逐项核对变更内容是否与 `spec/` 完全一致：

- 端点路径、请求/响应字段名、类型是否匹配 spec。
- 认证级别（`token_optional` / `token_required`）是否正确应用。
- 幂等性要求是否满足（MUST 接口是否携带 `Idempotency-Key`）。
- 分页参数和响应格式是否符合游标分页规范。

### 步骤四：跨项目回归与类型一致性

- 如果变更涉及 DTO、共享类型、WebSocket 事件等跨项目共享内容，**必须同步更新所有消费方**（backend、frontend_svelte、harmonyapp）。
- 对受影响的其他项目重复步骤一，确保没有破坏已有功能。
- **类型对齐检查**：修改任何 DTO/类型定义后，逐一比对以下三端的类型定义是否与 `spec/api_spec_types.md` 和对应模块 spec 一致：
  - `backend/src/dto/` -- Rust struct
  - `frontend_svelte/src/lib/types/api.ts` -- TypeScript interface
  - `harmonyapp/entry/src/main/ets/core/types/api.ets` -- ArkTS interface

## 权限边界

为了保障项目安全和数据完整，AI 的操作权限被严格限制。

### ✅ 可直接执行（无需询问）

- 读取文件、搜索代码（如 `rg`、`grep`）、列出目录内容。
- 运行构建/检查命令（`cargo check`、`pnpm check`、HarmonyOS `codelinter` / `hvigorw` / `ohpm` 等）、格式化、Lint、单个测试。HarmonyOS 细则参见前文“构建与验证命令”。
- 运行上述“禁止模式扫描”中的 rg 命令。
- 查阅 `spec/` 目录下的任何规范文件。

### ❓ 必须先询问用户

- 安装/升级依赖（如 `cargo add`、`pnpm install`）。
- 运行完整的测试套件或端到端测试。
- 修改 CI/CD 配置、部署脚本、环境变量模板。
- 变更数据库 migration 文件。
- **执行数据库重置或迁移命令**（如 `sqlx migrate run`、`sqlx database reset`、`psql` 等）：Agent 必须将命令告知用户，由用户手动执行后将结果反馈给 Agent。
- **涉及大量文件删除或批量修改的操作**：Agent 必须先列出将受影响的文件清单和操作说明，经用户明确确认后才可执行。禁止自动做决定。

### 🚫 绝对禁止

- **修改 `spec/` 下的任何文件**，除非用户明确要求修改规范。
- **编造 spec 中不存在的 API 端点、字段或行为**。
- **在任何项目中硬编码 API 路径字符串 `"/api/v1/..."`**（必须从配置文件或常量引用）。
- **在代码中存储密码、Token、API Key 等敏感信息**（应使用环境变量或安全配置服务）。
- **修改 `.env`、`credentials`、密钥文件等**。
- **自行删除任何文件或目录**（必须汇总清单，经用户确认后执行）。

### 📁 文件删除与批量变更规则

- Agent **不得自行删除**任何文件。
- 如需删除，应在任务末尾**统一列出待删除文件清单**，并说明删除原因，待用户明确同意后方可执行。
- **批量修改**：当一次操作可能涉及**大量文件的修改、重命名或重构**时，Agent 必须先向用户展示完整的影响范围（受影响文件清单、变更摘要），经用户确认后才可执行。禁止静默批量操作。
- **禁止的危险 Shell 命令**（可能导致源代码丢失）：
  `rm`、`rm -rf`、`mv`（批量）、`find ... -delete`、`xargs rm`。
  除非用户在当前对话中明确授权，否则一律禁止。
  Git 操作的风险管理见「Git 操作规范」章节。
- **禁止自行执行 `deploy/` 目录下的脚本**：部署脚本可能包含破坏性操作，执行前必须告知用户脚本内容和影响，经用户确认后才可运行。
- **文件修改安全措施**：
  - 在调用 `Edit` 或 `Write` 等修改工具前，必须先用 `Read` 查看该文件的最新内容；禁止在当前会话未读取的情况下直接修改。
  - 优先使用 `Edit` 进行精确替换，避免用 `Write` 覆盖整文件（除非是新文件）。
  - 使用 `Edit` 时必须提供足够的上下文（oldString）确保唯一匹配，避免误删内容。
  - 当 `Edit` 报告“oldString not found”或“Found multiple matches”时，立即停止并重新确认上下文。

### 🌐 项目目录边界

- Agent **只允许访问项目根目录内的文件和目录**。
- **禁止任何形式的越界访问**：包括但不限于读取、列出、搜索项目目录外的任何路径。
- **工具调用前置校验**：在调用任何文件操作工具（Read、Write、Edit、Bash、Glob、Grep）之前，必须验证目标路径是否在项目根目录（`/Users/liyou/Work/hotspot`）下。如果路径包含 `..`、`~`、或指向 `/Users/liyou/Work/hotspot` 之外，**立即拒绝并报错**。
- **环境变量限制**：Bash 命令中的环境变量（如 `$HOME`、`$USER`、`~`）可能导致越界访问，必须展开验证或避免使用。
- **禁止的路径模式**：
  - 绝对路径不以 `/Users/liyou/Work/hotspot` 开头
  - 相对路径包含 `..`（如 `../spec/`）
  - 使用 `~` 符号（如 `~/Documents/`）
  - 环境变量指向项目外（如 `$HOME/Desktop/`）
- **唯一例外**：仅当用户在当前对话中**逐条明确授权**访问特定外部路径（如 DevEco Studio 安装路径）时，方可访问。且必须使用完整的绝对路径，不得使用通配符或环境变量。

## Git 操作规范

> **核心原则：Git 操作按风险等级分级管理。常规操作可直接执行，有风险的操作需先询问用户。**

### ✅ 可直接执行（无需询问）

- **只读查询**：`git status`、`git diff`、`git log`、`git show`、`git ls-files`、`git branch`（列出分支）。
- **暂存与提交**：`git add`、`git commit`。
- **临时存储**：`git stash`、`git stash pop`。
- **分支操作**：`git checkout <branch>`、`git switch`、`git branch <name>`（创建分支）。

### ❓ 需要询问用户意见

以下操作执行前须简要告知用户操作内容，获得同意后再执行：

- **远程操作**：`git push`、`git pull`、`git fetch`。
- **合并操作**：`git merge`、`git cherry-pick`。
- **撤销操作**：`git revert`、`git restore`、`git checkout -- <file>`。

### ⚠️ 高风险操作（必须详细说明风险并获得明确确认）

以下操作可能导致数据丢失或历史改写，执行前必须向用户说明具体风险和影响范围，获得明确确认后方可执行：

- **历史改写**：`git rebase`、`git commit --amend`（已推送的提交）。
- **强制操作**：`git push --force`、`git push --force-with-lease`、`git reset --hard`。
- **清理操作**：`git clean`、`git branch -D`（强制删除分支）。
- 任何带 `--force`、`--hard`、`--no-verify` 参数的命令。

### 子 Agent 约束

调用子 Agent 时，必须在 prompt 中告知其遵守本节 Git 操作规范，高风险操作需回报主 Agent 由用户确认。

## Agent 行为准则

- **语言要求**：所有回复、解释、注释、文档说明**必须使用中文**；仅代码、标识符、命令、配置键、URL、技术术语等无法或不宜翻译的部分可保持原文。
- **简洁优先**：实现任何功能时，**先选最简单的方案**。只有当简单方案被证明不够用时，才引入更复杂的设计。禁止"预防性架构"——不要为了应对假想的未来需求而增加当前的复杂度。当存在多种实现方案时，优先选择代码量最少、抽象层级最浅、依赖最少的那个。
- **建议原则**：对用户的建议需基于实际情况和最佳实践，经比较分析后给出明确结论和理由，避免无脑同意。
- **任务分解**：实现复杂功能时应拆分为子任务，使用 Todo 列表跟踪进度，并在完成时同步更新状态。
- **临时文件管理**：生成的辅助脚本或临时文档应存放于 `/tmp` 目录，任务完成后立即删除，**禁止留在项目代码目录中**。
- **设计/计划文档存放**：头脑风暴记录、设计说明、实现计划等文档一律写入 `/tmp/hotspot-plans/` 等仓库外路径，禁止在仓库内创建 `docs/`、`plans/` 等目录存放，避免污染 git；如需长期保留由用户另行备份。
- **命令使用限制**：
  - 禁止使用 `head` 或 `tail` 命令读取文件（应使用 `cat` 或直接读取）。
  - 禁止自动更新文档版本号。
  - 阅读文档时注意文档的版本信息，确保基于最新版本。
  - **工具优先**：读取/搜索/修改/写入文件时，必须优先使用平台提供的工具（如 `Read`、`Edit`、`Write`、`Glob`、`Grep`）。只有当这些工具无法满足需求时，才能使用 `python`、`node`、`sed`、`awk` 等脚本命令修改文件，并需在执行前先说明原因。除非用户另有授权，否则不要通过脚本直接写入文件，以便在对话中能够看到清晰的 diff。
- **文档格式**：本文档（及所有项目文档）不使用数字章节编号，以保持灵活性。

---

**遵守以上规则，将极大提升代码质量和协作效率。如有任何疑问，请首先查阅 `spec/` 目录下的规范文件。**
