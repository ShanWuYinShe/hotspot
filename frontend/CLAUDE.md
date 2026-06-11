# CLAUDE.md

本文件只定义本子项目的技术约束与本地命令；仓库级 workflow 以根 `CLAUDE.md` 为准。

> **注意**：本项目为 **主前端**（SvelteKit），当前所有前端开发在此进行。设计需与 `frontend_react/`（React，Figma Make 生成）保持 UI/UX 一致——React 版作为设计参考保留，不再退役本项目。

## 构建 & 测试

- `bun run build` — 构建
- `bun check` — 类型检查（svelte-check）
- `bun lint` — 格式检查（prettier --check）
- `bun run format` — 格式化（prettier --write）

> **测试**：package.json 无 `test` 脚本，暂无自动化测试。

## 约束

- **API 路径**：统一从 `endpoints.ts` 引用。
- **请求封装**：所有请求走 `client.ts` 和 `services/*`。
- **HTTP 方法**：统一 POST。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **分页**：cursor 只透传，不解析、不构造。

## 页面布局规范（所有页面、所有设备的硬约束）

> 核心原则参考「发现」页对主内容区的处理：**内容居中收束在一条合理的测量栏内，宽屏不铺满全宽、不留大片空白；同时始终保留舒适的响应式留白，内容不紧贴容器边缘。** 借鉴 Linear / Notion / Stripe / Vercel 等优秀产品的版式：长内容有最佳阅读宽度，留白随视口平滑缩放，节奏统一——既克制又有设计感。

### 必用原语（定义在 `src/app.css`）

- **`.page-shell`**：内容栏容器，`margin-inline:auto` 居中 + 限定 `max-width`。任何页面的滚动内容**必须**包在某一档 shell 内，不允许内容直接铺满 `desktop-main-card`。
  - 默认（feed / 卡片列表 / 网格）：`min(1180px, 100vw - 32px)`
  - `--wide`（数据表格、管理后台、仪表盘）：`min(1320px, …)`
  - `--narrow`（详情、阅读、长文）：`min(720px, …)`
  - `--form`（表单、设置、个人资料）：`min(560px, …)`
- **`.page-pad`**：响应式留白（`clamp` 横向随视口缩放，底部更宽容纳「加载更多」/安全区）。`--tight` 为紧凑变体。
- 现有 `.feed-density-shell` / `.feed-density-pad`（discover、moments/nearby）是同一原则的 feed 专用实现，等价保留，**新页面优先用 `.page-shell` + `.page-pad`**。

### 用法

```svelte
<!-- 滚动区负责留白与居中，内容栏负责收束宽度 -->
<div class="flex-1 overflow-y-auto no-scrollbar page-pad">
  <div class="page-shell page-shell--narrow">…页面内容…</div>
</div>
```

带右栏的页面（如「发现」）：主栏用 shell 收束，右栏固定宽度、`border-left` 分隔，不与主栏争抢空白。

### 档位选择

| 页面类型                                                                   | 档位                                       |
| -------------------------------------------------------------------------- | ------------------------------------------ |
| 发现 / 附近动态 / 附近的人 / 附近群组 / 好友 / 关注 / 粉丝 / 我的瞬间·群组 | 默认 `.page-shell`（网格随宽度自适应列数） |
| 瞬间详情 / 用户主页 / 群组详情 / 活动详情 / 分享页                         | `.page-shell--narrow`                      |
| 设置全系 / 个人资料 / 编辑资料 / 搜索 / 登录                               | `.page-shell--form`                        |
| 管理后台所有页（表格密集）                                                 | `.page-shell--wide`                        |

### 三档设备一致性

- **桌面（≥1280px）**：内容在 `desktop-main-card` 内再经 shell 收束 → 宽屏两侧自然留白，不拉伸成超长行。
- **平板（600–1279px）**：单列内容用 shell（多取默认档量级）+ `page-pad`，底部为浮动玻璃导航留出空间。
- **移动（<600px）**：shell 的 `100vw - 32px` 兜底保证左右各约 16px 留白；不追求居中收束（视口已足够窄），重点是不贴边。

### 红线

- 禁止页面直接用裸 `max-w-3xl` / `max-w-2xl` / 内联 `max-width: 640px` 等各自为政的写法——统一走 `.page-shell` 档位，保证全站节奏一致。
- 禁止内容直接 `w-full` 平铺到容器边缘（既无 max-width 又无 `page-pad`）。
- 改动布局后用 `bun check` + `bun run build` 校验。
