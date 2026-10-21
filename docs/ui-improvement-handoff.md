# UI 一致性改进交接（2026-10-22，Round 1-10）

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
- **鸿蒙登录「账号/密码串位」复验（Round 10，端到端通过）**：模拟器实测 `uitest_admin / UiTest#12345`
  一次登录成功——提交前账号/密码正确；提交后 hilog `POST /public/v1/auth/signin → code=0`、
  后端 `signin 200`、Toast「欢迎！」、进入消息列表；**无**账号框改写、**无**密码策略错。
  - **根因更正（重要）**：上一轮观察到的「提交后账号框变密码值」主要是**自动化坐标过期伪影**，
    不能据此判定 app bug——`uitest uiInput inputText <x> <y> <text>` 会先按坐标点一下，
    键盘弹出后窗口上移，旧坐标落到安全键盘上先敲进一个字符（实测输入 `abc` 出现 4 个掩码；
    改用键盘弹出后的实时坐标则正好 3 个）。正确用法与自证方法已写入
    `hotspot-harmony-device-verify` skill；常驻错误槽位修复（`d39e1d7f`）保留为健壮性修复。
- **groups 详情 Tab 内容顶距统一（Round 10）**：成员 / 申请审批 / 邀请管理 / 封禁名单子 Tab 的
  内容顶距统一为 **24**（含空态；成员列表无「添加成员」行时由列表自身承担 24），
  子 Tab 栏距主 Tab 栏也改为 24——同页 tab→内容只有一种间距。
- **鸿蒙瞬间详情评论区跨端对齐（Round 10）**：评论分区顶距 16→**24**（对齐前端 `mt-6`）、
  分区标题 12/medium→**14/semibold**（对齐前端 `text-md` + semibold）；端上实测正文→评论标题
  间距 61px→93px（+8vp，密度 3.917），标题行高 67px→82px。
- **状态页垂直位置（Round 10 产品决策）**：维持**页面内居中**，不改「靠上 1/3」。
- **卡片圆角跨端（Round 10 核对）**：交接文档原记「鸿蒙 20 / 前端 12」**不成立**——两端普通卡片
  都是 20（鸿蒙 `SHAPE.CARD=RADIUS.LG` / 前端 `rounded-lg`），紧凑卡与列表容器都是 16
  （`COMPACT_CARD` / `rounded-md`），12 是图标块/小控件档（`ICON_TILE` / `rounded-sm`）。
  两端已同口径，无需改动；跨端刻度脚本持续守护。
- **首页**：错误态/空态无卡；顶栏三图标 `neutral → surface`（对齐 web `.action-icon-btn`）。
- **登录页**：分段控件激活胶囊填满分段槽（44vp）。
- **「附近」**：错误态补重试按钮。
- **开发接口地址**：`127.0.0.1` → `10.0.2.2`（模拟器可直连宿主，端上验证链路打通）。
- **编辑资料**：编辑期间隐藏底部 tab 栏（防丢草稿）。

## 待做（按优先级）

暂无阻塞项（Round 10 已清空原 1-3 项：登录复验通过 / 状态页位置决策为居中 / 卡片圆角已一致）。
下一轮可从下方「遗留观察」中挑，或按产品新反馈立项。

## 遗留观察（已评估，保留现状；非缺陷）

- **`ContextRail` rail 分区间距 16**（共享原语 `gap:1rem`）：rail 是紧凑侧栏，16 为有意设计；
  若改 24 会同时影响 notify/users/moments 三处 rail，收益低，保留。
- **`PageHeader` 自身上下内距**（有 below：16/14；无 below：18/18）属页头组件内部构图，
  不属于页面级三档；若要求页头严格 12/16/24 需单独立项评估全站页头高度，保留。
- **notify 极端态**：已读为空、仅未读列表时，列表 `mb-6` + `ListEnd` 自带 `py-6` = 48 收尾留白；
  只在极端组合出现，属状态组件内部节奏，保留。

## 端上验证环境要点

- 模拟器直启：`/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator -start "Pura 90 Pro"`
  （约 10s 就绪，`hdc list targets` 出现 `127.0.0.1:5555`）。
- **真机与模拟器同时在线时**必须 `hdc -t 127.0.0.1:5555 install ...` 指定设备，否则报
  `need connect-key`。
- 模拟器装**仓库构建**即可正常加载数据（`DEV_SERVER=http://10.0.2.2:8888`），无需临时改常量。
- 想验证错误/空态：`docker compose -f docker-compose.yml -f docker-compose.dev.yml stop backend`
  → 各页进入错误态 → 验证完 `start backend`。
