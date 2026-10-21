# UI 一致性改进交接（2026-10-21，Round 1-8）

本文件记录 UI 一致性/审美改进的完成基线与待做项，供后续会话直接接手。规则细节见
`.agents/skills/hotspot-harmony-ui/SKILL.md`、`.agents/skills/hotspot-frontend-layout/SKILL.md`；
审计结论见 `docs/frontend-ui-audit.md`、`docs/backend-audit.md`。

## 已完成基线（不要回退）

- **状态组件统一为无卡形态**：两端 `EmptyState`（鸿蒙 55 处、前端 51 处）全部无卡片，
  `SystemLoadingView` 默认无卡（`card` 选项留给浮层）；硬编码状态文案 0 处带卡片。
- **鸿蒙间距体系归一**：纵向 46 处、横向 13 个文件，离群档归零；规则 **12 / 16 / 24** 三档。
  前端三档**只约束页面级/区块级**，组件内部沿用 Tailwind 标准刻度（`py-2.5` 等 73 处不要动）。
- **首页**：错误态/空态无卡；顶栏三图标 `neutral → surface`（对齐 web `.action-icon-btn`）。
- **登录页**：分段控件激活胶囊填满分段槽（44vp）。
- **「附近」**：错误态补重试按钮。
- **开发接口地址**：`127.0.0.1` → `10.0.2.2`（模拟器可直连宿主，端上验证链路打通）。
- **编辑资料**：编辑期间隐藏底部 tab 栏（防丢草稿）。

## 待做（按优先级）

1. **前端页面级纵向节奏**：规则已定义（页面根容器 / 区块之间 / 页顶底 用 12/16/24），
   需逐页读 `routes/(app)/**/+page.svelte` 根容器的 `gap`/`py` 并归一。
2. **鸿蒙登录「账号/密码串位」复验**（修复已合并 `d39e1d7f`，端到端未验证）：
   - 修复前症状（稳定复现 3 次）：正确填两框 → 点登录 → 账号框变密码值、报「密码需为 8-128 位…」、
     hilog 无 signin 请求；推理为两个 `SystemInput` 之间的条件节点导致组件复用串位；
   - 自动化 `uitest uiInput` 在键盘弹出后坐标会失效，**建议真机手点验证**；
   - 验证标准：提交后账号框仍是 `uitest_admin`，提示不再是密码策略错。
3. **状态页垂直位置**：现为居中，可改「靠上 1/3」（设计选择，需产品确认）。
4. **卡片圆角跨端统一**：鸿蒙 20 / 前端 12（skill 登记为有意偏差，如需完全一致再统一）。

## 端上验证环境要点

- 模拟器直启：`/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator -start "Pura 90 Pro"`
  （约 10s 就绪，`hdc list targets` 出现 `127.0.0.1:5555`）。
- **真机与模拟器同时在线时**必须 `hdc -t 127.0.0.1:5555 install ...` 指定设备，否则报
  `need connect-key`。
- 模拟器装**仓库构建**即可正常加载数据（`DEV_SERVER=http://10.0.2.2:8888`），无需临时改常量。
- 想验证错误/空态：`docker compose -f docker-compose.yml -f docker-compose.dev.yml stop backend`
  → 各页进入错误态 → 验证完 `start backend`。
