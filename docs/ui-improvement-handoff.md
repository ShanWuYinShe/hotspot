# UI 一致性改进交接（2026-10-21，Round 1-9）

本文件记录 UI 一致性/审美改进的完成基线与待做项，供后续会话直接接手。规则细节见
`.agents/skills/hotspot-harmony-ui/SKILL.md`、`.agents/skills/hotspot-frontend-layout/SKILL.md`；
审计结论见 `docs/frontend-ui-audit.md`、`docs/backend-audit.md`。

## 已完成基线（不要回退）

- **状态组件统一为无卡形态**：两端 `EmptyState`（鸿蒙 55 处、前端 51 处）全部无卡片，
  `SystemLoadingView` 默认无卡（`card` 选项留给浮层）；硬编码状态文案 0 处带卡片。
- **鸿蒙间距体系归一**：纵向 46 处、横向 13 个文件，离群档归零；规则 **12 / 16 / 24** 三档。
  前端三档**只约束页面级/区块级**，组件内部沿用 Tailwind 标准刻度（`py-2.5` 等 73 处不要动）。
- **前端页面级纵向节奏归一（Round 9）**：`routes/(app)/**` 逐页归一完成（21 个源码文件、2 个 skill + 本文件），
  落地口径见 `hotspot-frontend-layout` skill「纵向节奏 → 前端落地口径」。要点：
  - 页面根容器 `gap-4`(16)；「附近」枢纽页例外 `gap-6`(24)（根容器直接承载多个带标题分区）；
  - 区块之间 24（`gap-6`/`mb-6`）、区块标题→内容 12（`mb-3`）、区块内元素 16；
  - `contentClassName` 收敛白名单（`pt-4 pb-6` / 长表单 `pt-4 pb-10` / 封面 `pb-6` /
    总览 `pt-6 pb-8` / 移动端固定操作条条件式 `pb-28`）；
  - 消除所有「父 gap + 子 mt/mb」双倍叠加（如 `pt-6`+`mt-6`=48、`mb-5`+`mt-5`=40）。
  验证：svelte-check 0 error / ESLint 0 问题 / vitest 499 通过 / `bun run build` 通过；
  agent-browser 改前改后 64 对（16 路由 × 桌面/移动 × 亮/暗）几何+像素对比：水平位移 0、
  横向溢出 0、坏图 0、h1=1、控制台错误 0，13 项预期间距变化逐项命中。
- **首页**：错误态/空态无卡；顶栏三图标 `neutral → surface`（对齐 web `.action-icon-btn`）。
- **登录页**：分段控件激活胶囊填满分段槽（44vp）。
- **「附近」**：错误态补重试按钮。
- **开发接口地址**：`127.0.0.1` → `10.0.2.2`（模拟器可直连宿主，端上验证链路打通）。
- **编辑资料**：编辑期间隐藏底部 tab 栏（防丢草稿）。

## 待做（按优先级）

1. **鸿蒙登录「账号/密码串位」复验**（修复已合并 `d39e1d7f`，端到端未验证）：
   - 修复前症状（稳定复现 3 次）：正确填两框 → 点登录 → 账号框变密码值、报「密码需为 8-128 位…」、
     hilog 无 signin 请求；推理为两个 `SystemInput` 之间的条件节点导致组件复用串位；
   - 自动化 `uitest uiInput` 在键盘弹出后坐标会失效，**建议真机手点验证**；
   - 验证标准：提交后账号框仍是 `uitest_admin`，提示不再是密码策略错。
2. **状态页垂直位置**：现为居中，可改「靠上 1/3」（设计选择，需产品确认）。
3. **卡片圆角跨端统一**：鸿蒙 20 / 前端 12（skill 登记为有意偏差，如需完全一致再统一）。

## Round 9 遗留观察（未改，供后续按需处理）

- **groups 详情 Tab 内容顶距**：「成员」子 Tab 内容 16（`mt-4`）、「简介/公告」24；
  私有组件 `MembersList/ApplicationsList/InvitesList/BannedList` 若统一，需同页同步决策。
- **`ContextRail` rail 分区间距 16**（共享原语 `gap:1rem`）：若严格按「区块之间 24」需改共享原语，
  会同时影响 notify/users/moments 三处 rail。
- **`PageHeader` 自身上下内距**（有 below：16/14；无 below：18/18）属页头组件内部构图，
  未纳入本轮三档；若要求页头也严格 12/16/24，需单独立项评估全站页头高度。
- **鸿蒙 `MomentDetailPage.ets` 跨端漂移**：其注释/数值记载「对齐前端 mt-5(20)」，前端本轮把
  评论分区顶距改为 24（`mt-6`）；鸿蒙侧未改，后续同步时一起核对。
- **notify 极端态**：已读为空、仅未读列表时，列表 `mb-6` + `ListEnd` 自带 `py-6` = 48 收尾留白；
  属状态组件内部节奏，未动。

## 端上验证环境要点

- 模拟器直启：`/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator -start "Pura 90 Pro"`
  （约 10s 就绪，`hdc list targets` 出现 `127.0.0.1:5555`）。
- **真机与模拟器同时在线时**必须 `hdc -t 127.0.0.1:5555 install ...` 指定设备，否则报
  `need connect-key`。
- 模拟器装**仓库构建**即可正常加载数据（`DEV_SERVER=http://10.0.2.2:8888`），无需临时改常量。
- 想验证错误/空态：`docker compose -f docker-compose.yml -f docker-compose.dev.yml stop backend`
  → 各页进入错误态 → 验证完 `start backend`。
