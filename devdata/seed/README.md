# 开发数据配套设施（devdata/seed）

seed 测试数据所需的媒体素材与说明集中在本目录，**外置、不打包进后端二进制**。

- 后端不再使用 `//go:embed` 打包素材：生产镜像构建上下文是 `backend-go/`，本目录不在其中，
  生产二进制/镜像因此不含任何测试图片。
- 本目录只服务本地 dev / 测试环境：`cmd/seed` 在 `APP_ENV=production` 下直接拒绝执行。
- dev compose 以只读方式把仓库根 `devdata/` 挂载到 backend 容器的 `/devdata`
  （见 `docker-compose.dev.yml` 的 backend volumes）。

## 目录结构

```
devdata/
└── seed/
    ├── README.md          # 本文档（素材规格、数据规格、运行方式）
    └── media/             # 25 张 JPEG 素材（固定文件名，禁止改名）
        ├── avatar_*.jpg   #  9 张用户头像 400x400
        ├── group_*.jpg    #  4 张群头像   800x800
        └── moment_*.jpg   # 12 张瞬间图/聊天图 1200x800
```

素材是仓库的普通文件（随 git 版本管理），不属于生成产物；缺失/损坏时 seed 会明确报错，
不会静默降级。

## 素材目录解析（`-assets-dir` / `SEED_ASSETS_DIR`）

`cmd/seed` 按以下优先级确定素材目录，并在错误信息里列出所有尝试过的路径：

1. `-assets-dir DIR`（命令行参数，最高优先级）
2. 环境变量 `SEED_ASSETS_DIR`
3. 默认候选（相对进程工作目录，官方运行方式的 cwd 都是 backend-go 源码根）：
   - `../devdata/seed/media` —— 宿主机 backend-go 目录下 `go run`；dev 容器内 cwd=`/src`
     且 devdata 挂载在 `/devdata`，同一相对路径命中 `/devdata/seed/media`；
   - `/devdata/seed/media` —— dev compose 容器内挂载点；
   - `/src/devdata/seed/media` —— 「仓库根整体挂载在 /src」布局的兜底。

`scripts/seed-dev.sh` 会显式传 `-assets-dir /devdata/seed/media`（最可靠）；用户自己传了
`-assets-dir`/`--assets-dir` 时以用户参数为准。挂载布局不同时可用
`SEED_ASSETS_DIR_CONTAINER` 覆盖脚本注入的容器内路径。

## 运行命令

| 场景 | 命令 |
|------|------|
| 仓库根执行（推荐，自动走 dev compose） | `./scripts/seed-dev.sh` |
| 清空 seed 数据后重建（含 S3 对象） | `./scripts/seed-dev.sh --reset` |
| 只预览写入计划 | `./scripts/seed-dev.sh --dry-run` |
| 宿主机直接 dry-run（不需要 DB，验证素材加载） | `cd backend-go && go run ./cmd/seed --dry-run -assets-dir ../devdata/seed/media` |
| 容器内直接跑 | `docker compose -f docker-compose.yml -f docker-compose.dev.yml exec -T backend go run ./cmd/seed --dry-run -assets-dir /devdata/seed/media` |

三种模式：

- 默认 `--if-missing`：缺失则写入，已存在的行跳过（保留开发中产生的互动）；
- `--reset`：先按固定前缀清理 seed 数据（含对象存储对象）再完整重建；
- `--dry-run`：只打印计划（素材目录、账号、媒体数量、各表行数），不写数据库/对象存储。

真实写入前脚本会尽力确保 SeaweedFS bucket 存在；写完会重启 backend 使 `app_configs`
（默认浏览锚点等）重新加载。

## 素材规格与新增/替换

| 文件名前缀 | 数量 | 尺寸 | 用途 |
|------------|------|------|------|
| `avatar_*.jpg` | 9 | 400x400 | 用户头像（`users.avatar_url` 指向代理路径） |
| `group_*.jpg` | 4 | 800x800 | 群头像 |
| `moment_*.jpg` | 12 | 1200x800 | 瞬间图（22 个图片位轮换复用）/ 聊天图片 |

**替换图片**（推荐，零代码改动）：

1. 用同名文件覆盖 `devdata/seed/media/` 下的对应素材，尺寸必须与上表一致（加载时校验，
   尺寸不符会立即失败，避免把错误比例写进 `media_upload_sessions`）；
2. 重跑 `./scripts/seed-dev.sh --reset`。素材内容变化会改变 `md5`/大小，而默认模式
   （`--if-missing`）不会更新已存在的 media 行，必须用 `--reset` 重新上传对象并重写行；
3. 若只想验证素材本身可加载：`cd backend-go && go run ./cmd/seed --dry-run -assets-dir ../devdata/seed/media`。

**新增/删除素材文件**（需要改代码，否则清单校验会报错或文件不会被使用）：

1. `backend-go/internal/seed/assets.go` 的 `assetManifest` 增加/删除文件名，必要时扩展
   `expectedAssetSize`（新前缀的尺寸约定）；
2. 在数据规格里引用它：用户头像改 `specs_users.go` 的 `AvatarAsset`，群头像改
   `specs_groups.go` 的 `AvatarAsset`，瞬间图/聊天图改 `mediaplan.go` 的
   `momentImageAssetPool` / `chatImageAssets`；
3. 同步更新 `backend-go/internal/seed/assets_test.go` 的清单数量断言（当前 25 张）；
4. `cd backend-go && go test -count=1 ./internal/seed/...` 与 dry-run 确认无误后，`--reset` 重建。

## 数据规格怎么改

seed 的实现分层（改数据按层定位，不要直接在 SQL/表里手写数据）：

| 文件 | 职责 |
|------|------|
| `specs_*.go`（`specs_users.go` / `specs_moments.go` / `specs_groups.go` / `specs_chats.go` / `specs_misc.go`） | 声明式静态规格：用户/关系/账号/瞬间/群组/私聊/通知/分享/举报/管理端数据 |
| `specs.go` | 规格公共类型、默认设置、默认浏览锚点（徐州云龙湖） |
| `data*.go`（`data.go`、`data_users.go` 等同层文件） | 把规格展开为按 FK 顺序的表行：计算 S2 cell、计数列、引用媒体 `file_id` |
| `mediaplan.go` | 素材 → 媒体对象/上传会话/`file_id` 映射（12 张瞬间图轮换复用） |
| `tables.go` | 参数化 INSERT（`ON CONFLICT DO NOTHING`，幂等） |
| `cleanup.go` | `--reset` 按固定前缀清理（含对象存储对象） |
| `plan.go` / `run.go` | 计划构建 + dry-run 输出 / 执行（清理→上传→写库） |

约定：

- **固定 ID**：`user_seed_*`、`moment_seed_01..24`、`comment_seed_*`、`group_seed_1..4`、
  `gmsg_seed_*`、`chat_seed_*`、`notify_seed_*`、`share_seed_*`、`report_seed_*`、
  `upload_seed_*`、`role_seed_*`、`audit_seed_*` 等（前缀常量见 `naming.go`）；`--reset`
  按这些前缀精确清理，QA 脚本也依赖这些稳定 ID。
- **媒体命名**：`file_id = "file_" + md5(userID:uploadID)`，
  `storage_key = uploads/<userID>/<uploadID>.jpg`，`upload_id = upload_seed_*`；规则刻意
  复写 media 模块（`naming.go` 有单测锁定），不一致会导致头像/瞬间图 404。
- **时间**：规格用相对基准时刻的偏移（`CreatedAgo` 等），执行时基准时刻向下取整到小时，
  因此同一小时内重复执行时间线稳定。

改完数据后的验证：

```
cd backend-go
gofmt -w internal/seed
go build ./... && go vet ./... && go test -count=1 ./internal/seed/...
go run ./cmd/seed --dry-run -assets-dir ../devdata/seed/media   # 看 38 张表 396 行 / 38 个媒体对象
```

数据规模断言在 `internal/seed/plan_test.go` 的 `expectedTableCounts`（增删数据时同步更新）。

## 账号与口令

| 昵称 | login_id | user_id | 角色/说明 | 口令 |
|------|----------|---------|-----------|------|
| 鹿鸣 | `qa@hotspot.local` | `user_seed_lumos` | QA 巡检主账号（普通用户） | `SEED_QA_PASSWORD`（默认 `Qa12345678`） |
| 巡检员 | `admin@hotspot.local` | `user_seed_admin` | 管理员（grant_all，同时是完整普通用户） | `SEED_ADMIN_PASSWORD`（默认 `Admin12345678`） |
| 阿禾 | `sophia@hotspot.local` | `user_seed_sophia` | 普通用户 | `Seed123456` |
| 野原 | `marco@hotspot.local` | `user_seed_marco` | 普通用户 | `Seed123456` |
| Mika | `mika@hotspot.local` | `user_seed_mika` | 普通用户 | `Seed123456` |
| 青砚 | `chen@hotspot.local` | `user_seed_chen` | 普通用户 | `Seed123456` |
| 林一 | `lin@hotspot.local` | `user_seed_lin` | 普通用户 | `Seed123456` |
| 小满 | `xiaoman@hotspot.local` | `user_seed_xiaoman` | 普通用户 | `Seed123456` |
| 隐者 | `privacy@hotspot.local` | `user_seed_privacy` | 隐私设置/不可搜索场景 | `Seed123456` |
| 已封禁用户 | `banned@hotspot.local` | `user_seed_banned` | 封禁态/管理端场景 | `Seed123456` |

- 口令可用仓库根 `.env` 的 `SEED_QA_PASSWORD` / `SEED_ADMIN_PASSWORD` 或当前 shell 同名变量
  覆盖；其余开发账号口令固定为 `Seed123456`。
- `group_seed_2`（云龙湖晨跑团）为口令群，入群口令 `run123456`。
- 口令仅本地开发使用，**生产部署不得沿用**；生产环境执行 seed 会直接拒绝。

## 与主环境 / 生产的边界

- **不改已有数据**：默认模式写入用 `ON CONFLICT DO NOTHING`，已存在的行跳过；`--reset` 只删除
  固定 seed 前缀的行与 `updated_by_user_id LIKE 'user_seed_%'` 的 `app_configs` 行。
- **`app_configs` 不覆盖已有配置**：seed 只写自己创建的默认配置行；已有环境（如主开发库）的
  默认浏览锚点保持原样。全新库上写入徐州云龙湖锚点（与 `user_seed_lumos` 一致），否则附近/地图
  默认位置与种子数据相距 400+km 会开箱即空。需要强制对齐时用管理端「应用配置」页修改，或先删
  除 `config_id='default'` 行再重跑 seed。
- **不进入生产产物**：素材在 `devdata/`（仓库根），不在 `backend-go/` 构建上下文内；生产二进制
  不再包含任何测试图片，也不提供 `--assets-dir` 之外的新依赖。
- **仅 dev compose 可见**：dev compose 以 `:ro` 挂载 `./devdata:/devdata`；未启动 dev compose 时
  宿主机 `go run` 仍可从 `../devdata/seed/media` 读取。
- 前端没有任何 API mock：dev 下 API / WebSocket / S3 全部直连真实后端，数据由 seed 提供。

## 数据规模（dry-run 可见）

- 38 张表 396 行 + 38 个媒体对象（9 头像 / 4 群头像 / 22 瞬间图 / 3 聊天图）。
- 覆盖 10 用户、24 瞬间、4 群组、3 私聊、9 通知、2 分享、4 举报、2 管理角色、6 审计日志、
  2 系统公告、1 份 AppConfig。
- 分享码：`SEEDSUNSET23`（moment_seed_01）、`SEEDUSERPAGE`（user_seed_lumos 主页），
  QA 走 `/share/SEEDSUNSET23` 即有内容。

## 常见问题

1. **报错「未找到开发素材目录 devdata/seed/media（已尝试: …）」**：仓库不完整（缺 devdata/）
   或运行目录/容器挂载不对。确认仓库完整；容器内确认已重建 backend 使 `/devdata` 挂载生效
   （`docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d backend`）；
   也可显式传 `-assets-dir`。
2. **报错「素材 xxx 尺寸为 AxB，期望 CxD」**：替换的图片尺寸不符，按上表尺寸重新导出。
3. **报错「缺少 N 个必需文件」**：目录里缺文件或文件名被改；错误信息会列出缺失清单，
   从仓库补全即可（清单在 `internal/seed/assets.go` 的 `assetManifest`）。
4. **换了素材但页面还是旧图**：默认模式不会更新已存在的 media 行（旧 md5/大小），用
   `./scripts/seed-dev.sh --reset` 重新上传并重建；仍有旧图时清浏览器/S3 缓存。
5. **seed 后地图/附近默认位置没变**：`app_configs` 已有行不会被覆盖（见上节）；脚本会重启
   backend 让新配置生效，但已有环境需要手动在管理端修改默认位置。
6. **全新 SeaweedFS 卷首次写入 404**：脚本会在真实写入前尽力自动建 bucket；外部对象存储环境
   需运维预置 bucket。
7. **单元测试需要素材吗**：不需要。`go test ./internal/seed/...` 用临时目录现场生成 JPEG，
   不读 `devdata/`；真实素材由 dry-run / 实际 seed 校验。
