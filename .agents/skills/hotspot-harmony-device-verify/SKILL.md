---
name: hotspot-harmony-device-verify
description: hotspot 鸿蒙端模拟器实测流程：用 hdc + uitest 在 DevEco 模拟器上验证 harmonyapp 页面与角色态（登录/切号/截图），含本机实测坐标与踩坑。触发：需要验证鸿蒙端某页面在不同角色下的 UI；在模拟器上跑 harmonyapp 走查或截图；hdc 连模拟器、uitest dumpLayout/click/fling；排查“个人中心/管理入口为何不显示”；本地 dev 环境造测试账号。
---

# hotspot 鸿蒙端模拟器实测流程（hdc + uitest）

通用 hdc 行为见用户级 skill `harmony-device-debug`；本文件只记 hotspot 项目专属事实与实测流程。

## 项目事实（唯一维护处）

| 项 | 值 |
|---|---|
| bundleName | `com.lsyb.social` |
| 入口 Ability | `EntryAbility` |
| 应用日志域 | `A0FF` 前缀（`grep A0FF` 抓 app 日志） |
| 模拟器调试后端地址 | `http://127.0.0.1:8888`（hdc rport 反向转发，见下） |
| 后端 dev 网关 | 宿主 `http://127.0.0.1:8888`（nginx） |
| 测试库 | postgres 容器 `hotspot-postgres-1`，库 `hotspot`，用户 `postgres` |

**调试后端地址（2026-10-17 实测更正）**：AppConstants.ets 现行代码统一
`http://127.0.0.1:8888 + hdc 反向转发`（模拟器 `hdc -t 127.0.0.1:5555 rport tcp:8888 tcp:8888`，
真机 USB `fport`）。模拟器重启后转发条目不会自动恢复，装机前先设好，并以
`hdc fport ls` 出现 `[Reverse]` 条目为准；app 内请求实测 HTTP 200 / code=0（63ms）。
旧记录「设备侧 8888 被占、rport 报 listen failed、走 10.0.2.2」已不复现
（用户级 harmony-device-debug 2026-10-16 同步更正）；`http://10.0.2.2:8888`
保留为 rport 失效时的备选直连（SLIRP 用户态 NAT）。

## 测试账号（本地 dev 库）

| 登录名 | 密码 | 角色 | user_id |
|---|---|---|---|
| `uitest_a` | `UiTest#12345` | 正式用户（normal，非管理员） | `user_4541afca9608ded2a7517066` |
| `uitest_admin` | `UiTest#12345` | 平台管理员（admin_users + grant_all_permissions） | `user_c8b03b846f825cdd6d9f8317` |
| （app 内「快速体验」） | — | 游客 temp | 每次新建 |

管理员权限服务端自查（验证角色数据是否就绪）：

```bash
API=http://127.0.0.1:8888
TOKEN=$(curl -s -X POST "$API/public/v1/auth/signin" -H 'Content-Type: application/json' \
  -d '{"login_id":"uitest_admin","password":"UiTest#12345","client_type":"app","device_id":"probe-1"}' \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["access_token"])')
curl -s -X POST "$API/admin/api/v1/rbac/me" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{}'
# 管理员 → {"code":0,"data":{"permissions":["*"],...}}
# 非管理员 → {"code":10200,"message":"非平台管理员"}
```

造新账号：正式用户走 `POST /public/v1/auth/register`（用户名型，`verification_code: null`，
必带 `Idempotency-Key`）；授予管理员需 SQL 直插（`ADMIN_BOOTSTRAP_USER_ID` 只在库中无管理员时生效）：

```sql
INSERT INTO admin_users (user_id, grant_all_permissions) VALUES ('<user_id>', TRUE)
ON CONFLICT (user_id) DO UPDATE SET grant_all_permissions = TRUE, revoked_at = NULL;
```

## 实测流程（已验证）

1. **连接**：`hdc list targets -v` 出现 `Connected`（本机模拟器为 `127.0.0.1:5555`）。
2. **冷启动**：`aa force-stop com.lsyb.social` → `aa start -b com.lsyb.social -a EntryAbility`。
3. **登录**：优先从「消息」Tab 的登录提示进入（见下「关键前提」）。
   坐标先 `dumpLayout` 取 bounds：登录页账号框≈`(628,1398)`、密码框≈`(628,1601)`、登录键≈`(628,1814)`；
   「登录 / 注册」入口在提示页≈`(628,1142)`。
   `uitest uiInput inputText <x> <y> '<text>'` → `uinput -K -d 2 -u 2` 收键盘 → 点登录。
4. **进个人中心**：点底部「我的」Tab，然后 `uitest uiInput fling 628 2150 628 900 3500` 滚到动作行。
5. **断言**：`uitest dumpLayout -p /data/local/tmp/ui.json` 后逐节点取 text，判断
   「升级为完整账号」/「管理员工具台」/「设置」/「退出登录」是否存在。
6. **截图**：`snapshot_display -f /data/local/tmp/x.jpeg` → `hdc shell "cat …" > local.jpeg`。
7. **设置浏览锚点（2026-10-17 实测）**：首页空态点「设置浏览位置」进地图选择器后——
   - 放锚点用 `uitest uiInput longClick <x> <y>`（长按放置；单击无效，双击是地图缩放）。
     长按即写 `anchor_history` 偏好并触发逆地理解析；
   - 再点「设为浏览锚点」确认。**锚点模式确认后不关页**（toast「浏览锚点已设置」），
     需自己返回首页——切 Tab 前注意先返回，否则 Tab 点击落空；
   - 模拟器**系统地理编码无数据**：地点搜索（geocodeMultiple 走系统 geocoder）
     对任何关键词都返回「未找到该地点」，且锚点名只会是「未知位置」——这是模拟器
     环境限制，真机不受影响；历史锚点持久化在偏好文件 `anchor_history`。
   - 快速体验游客的锚点与会话态（AppStorage）**冷启动即清**，重装(-r)保留偏好但
     锚点需重新设置；暗色等偏好（dark_mode_preference）在 `geotalk` 偏好文件中持久化。
8. **密度/尺寸标定**：本机模拟器（Pura 90 Pro 实例）密度 ≈3.917（1256px / ≈320vp 逻辑宽），
   dump bounds 换算 vp 用 3.917，不要想当然用 3.25（2026-10-17 实测，曾因密度误算
   误判骨架卡/空态过大）。

## 关键前提：验证角色行必须在「我的」Tab 首次挂载前登录

`ProfilePage.aboutToAppear()` 里 `await this.refreshAdminPrivileges()`（ProfilePage.ets:314）
**只在组件首次挂载时执行一次**；`hasAdminPrivileges` 是普通 `@State`，无 `@Watch`。
PhoneLayout 的 `TabContent() { ProfilePage() }` 常驻挂载，**登录/切号不会触发 aboutToAppear**。

因此正确姿势是：
**冷启动 → 进「消息」Tab（此时「我的」Tab 未挂载）→ 登录 → 再首次点「我的」**。
若先点过「我的」再登录，管理员行不会出现（实测 `/admin/api/v1/rbac/me` 请求数为 0）。

判断“到底有没有发权限请求”：`hilog -r` 清缓冲 → 后台 `hdc shell hilog > log.txt` → 操作 →
`grep -E 'rbac|ProfilePage' log.txt`。首次挂载有此请求、同 mount 登录时没有。

## 角色 → 动作行预期（2026-10-11 模拟器实测基线）

| 角色 | 升级为完整账号 | 管理员工具台 | 设置 | 退出登录 |
|---|---|---|---|---|
| 匿名 | — （整页 LoginPromptView） | — | — | — |
| 游客 temp | ✅ | ❌ | ✅ | ✅ |
| 正式 normal | ❌ | ❌ | ✅ | ✅ |
| 管理员 | ❌ | ✅ | ✅ | ✅ |

动作行顺序：升级行在最前（仅 temp），然后 更新位置 → 设置 → [管理员工具台] → 退出登录。
管理员行点击落到「管理后台」概览页（`PageNames.ADMIN_OVERVIEW`）。

## 曾发现的缺陷（**已修复**：217abdc6；复现步骤保留作回归用例）

1. **同 mount 登录管理员后无管理员行**：先点过「我的」再登录管理员，该行不出现，
   切到首页再切回也不恢复（TabContent 常驻，不重新 mount）。根因见上「关键前提」。
2. **同 mount 切号残留上一账号管理员行（权限残留）**：管理员在「我的」内退出登录→改登普通账号，
   「管理员工具台」行仍显示。点击被服务端挡住（`10200`），但 app 把非管理员错误
   当网络错误展示（`AdminOverviewPage.ets:90` → “加载权限失败，请检查网络”），文案误导。
   根因同 1：`logout()` 未重置 `hasAdminPrivileges`。

修复方向：ProfilePage 订阅登录态变化重新 `refreshAdminPrivileges()`，登出清零该状态；
AdminOverviewPage 区分 `10200` 与网络错误。

**修复落地核实（2026-10-12，读码确认三条全在）**：
1. ProfilePage 经 `interactionEventHub.on('auth_user_changed')` 触发 `refreshAdminPrivileges()`
   （ProfilePage.ets:316 → :298），登录/切号不再依赖组件重挂载；
2. `hasAdminPrivileges` 在刷新起点（:444）、错误路径（:459）、登出（:666）三处清零；
3. AdminOverviewPage 对 `10200` 按「非管理员业务终态」处理，不再当网络错误（:91 注释）。

**运行时回归已实跑通过（2026-10-12，模拟器 + dev 库测试账号）**：
- 缺陷 1（先消息 Tab 登录管理员 → 首次进「我的」）：「管理员工具台」**出现**，
  行序与上方基线表一致（设置 → 管理员工具台 → 退出登录）；
- 缺陷 2（我的内退出 → 页内登录入口改登 uitest_a）：「管理员工具台」**消失**，
  动作行回落普通用户形态（…更新位置 → 设置 → 退出登录）。
两缺陷均闭环，无需再复现。

## 本机实测踩坑（补充通用 skill）

- **`hdc file recv` 从模拟器拉文件会 hang 到超时**（exit=124）；改用
  `hdc shell "cat /data/local/tmp/x.jpg" > local.jpg`（字节一致）。
- **滚轮/拖拽不滚动 ArkUI 列表**：`uinput -M -s`、`uinput -M -g` 实测无效，
  `uinput -T -g` 报 “wrong number of parameters”。滚动一律 `uitest uiInput fling`。
- **软键盘弹出后原坐标失效**（窗口上移），先收键盘或重新 dumpLayout。
- **`uitest uiInput click` 坐标越界会静默点空**（不报错），必须用 bounds 中心点。
- **登录态可直接看文件**：`/data/app/el2/100/base/com.lsyb.social/haps/entry/preferences/geotalk`
  是明文 XML（`auth_accounts` / `current_user_id` / `server_base_url`，shell 用户可读）。
`- 辅助脚本（本机）：`。


## 沙盒下启动模拟器：已可直启（2026-10-14 插件放行实测）

> **状态提醒（2026-10-18/19 实测）**：本条只保证 `~/.Huawei/Emulator/deployed`（实例镜像）可写；
> **实例目录被拦的问题后来复现过**（见本文 §「1. 实例目录又被沙盒拦」），当时的解法是
> 把实例克隆到工作区 + 重定向 `config.ini` 里的 `configPath`/`logPath`/`instancePath`。
> 启动前先按那条走一遍，能直启就直启、报 EPERM 就克隆。

**`~/.Huawei/Emulator/deployed` 已由 DSH sandbox-extra-roots 插件放行**（2026-10-14 晚实测：
`workspace-write` 下 `Emulator -start` 约 10s 就绪、`hdc list targets` 出现 `127.0.0.1:5555`、
`*.qcow2` 正常写入实例目录），直接 `Emulator -start <实例名>` 即可，无需任何重定向。
`-stop` 可能 hang，清场用 `pkill -9 -f Emulator`。

历史方案（未配插件的机器适用，2026-10-12 实测）：放行前直启报

```
Emulator: Error opening logfile /Users/liyou/.Huawei/Emulator/deployed/Pura 90 Pro/Log//qemu.log: Operation not permitted
```

目录权限本身正常（`drwxr-xr-x`、属主是自己），返回的是 **EPERM 而非 EACCES** —— 这是
DSH 沙盒的拒绝签名（对照：写工作区成功）。当时曾误判「必须申请 danger-full-access」，
其实用 `-instancePath <path>` 把实例目录重定向进工作区即可：

```bash
SRC="/Users/liyou/.Huawei/Emulator/deployed/Pura 90 Pro"
DST="/Users/liyou/work/hotspot/.workwork/emu-instance"
mkdir -p "$DST"
cp -c -R "$SRC" "$DST/Pura 90 Pro"        # -c = APFS 克隆，5.6GB 近乎瞬时、不占额外空间
# 改克隆件的 config.ini：instancePath / logPath / configPath 三处全部指向 $DST
/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator \
  -start "Pura 90 Pro" -instancePath "$DST"   # 注意是**父目录**，不是实例目录本身
```

要点（历史）：
- `-instancePath` 传**父目录**；传实例目录本身会报
  `"X" is not found. Please create the device(folder): "<path>/X"`。
- 只读的系统镜像在 `~/Library/Huawei/Sdk/system-image/...`，沙盒**允许读**，无需搬运；
  实例目录里只有可写 overlay（`*.qcow2`）与 `ram.bin`。
- `config.ini` 里 `logPath` / `configPath` 默认指向 `~/Library/...`，也要一并改到工作区。
- 克隆件与真实实例**共享同一份用户数据快照**：里面装好的 App、登录态都会带过来。
- 收尾时删掉 `.workwork/emu-instance` 即可，真实实例不受影响。
- **结论（2026-10-14 更新）**：这类「工具要写 home 目录」的问题**优先查 sandbox-extra-roots
  放行清单**（放行即直启），其次才是换目录/重定向；模拟器场景已由放行直启取代 `-instancePath`。


## 走查铁律：截图与 dumpLayout 必须**同一次调用**获取（2026-10-12，已踩 3 次）

用截图目测判断布局位置时，**极易与 dumpLayout 的坐标不一致**——因为两次命令之间
页面可能仍在滚动/重排（尤其是刚 fling 过、或有入场动画时）。我因此在本次会话里
**连续 3 次误判**（都误以为「元素偏上/没生效」）：

1. 动态详情长文：目测行距 62px ≠ 计算 98px，误判「行高没生效」——实际是截图与 dump
   处于不同滚动位置，dump 取到的是被裁剪的节点高度（873px vs 1372px）。
2. 首页空态卡片：目测「卡片偏上」，实测卡片中心 1420 vs 容器中心 1419.5，**完全居中**。
3. 消息页「最近」标签：一次测量 x=109（误判未对齐），另一次 x=73（正确）。

**正确做法**：把 dumpLayout 与 snapshot_display 放进**同一条 shell 命令**，且**先 dump
后截图**（或先截图后 dump，但两者之间不插入任何滚动/点击）：

```bash
hdc shell uitest dumpLayout -p /data/local/tmp/x.json
hdc shell "cat /data/local/tmp/x.json" > /tmp/x.json
hdc shell snapshot_display -f /data/local/tmp/y.jpeg
hdc shell "cat /data/local/tmp/y.jpeg" > shot.jpeg
```

**判定位置类问题只信 dumpLayout 的坐标**（它是布局树的真值），截图只用于看观感
（层级/密度/配色）。两者冲突时，**以 dumpLayout 为准**，不要凭截图下结论。

### 临时账号 token 时效短（2026-10-13 实测）

「快速体验」的临时账号 token 约 30 分钟内失效：跨轮走查（间隔 >30min）后应用
回到「个人中心需要登录」态，需重新 快速体验 → 开始体验。走查批次尽量在同一
时效窗口内完成；「开始体验」按钮坐标每次以 dump 定位（弹层出现位置稳定在
y≈2358，但勿硬编码——曾在 2255 点空）。

## 模拟器环境坑（2026-10-18 实测，走查「登录 → 附近群组 → 加入」链路时连撞）

### 1. 实例目录又被沙盒拦（skill 上文「已放行直启」的说法已过期）

`Emulator -start "Pura 90 Pro"` 报
`Error opening logfile /Users/liyou/.Huawei/Emulator/deployed/Pura 90 Pro/Log//qemu.log: Operation not permitted`
且**进程直接没起来**（`pgrep -f Emulator` 为空）——但 hdc 可能仍留 `127.0.0.1:5555 Offline` 僵尸条目，
别把它当"模拟器在启动中"等下去。实测实例目录 `touch` 也是 `Operation not permitted`。

解法即上文历史方案（`cp -c -R` 克隆实例到 `.workwork/emu-instance` + 改 config.ini 三处路径 +
`-start <名> -instancePath <父目录>`），2026-10-18 复测有效，约 25s 起、`hdc list targets -v` 见 Connected。

### 2. 热启动快照会带回上次会话的 8888 反向转发（rport 建不上）

克隆实例 `isHotBoot=true`，RAM 快照里 `hdcd` 仍持有上次会话 rport 的监听：

- `hdc rport tcp:8888 tcp:8888` → `[Fail]TCP Port listen failed at 8888`
- `hdc shell "netstat -tlnp | grep 8888"` → `127.0.0.1:8888 LISTEN`（占用者就是 `hdcd` PID 992）
- 该残留监听**不再转发**：app 打 127.0.0.1:8888 会 15s 超时（hilog 里 `tcpConnE:115`、`lastRecvE:0`）

**解法（实测）**：`hdc shell reboot` → 等 `Connected` → 再 `rport tcp:8888 tcp:8888` 即 OK
（冷启动清掉残留套接字）。别用 `rport tcp:8899 tcp:8888` 绕——app 的 `DEV_SERVER` 固定
`http://127.0.0.1:8888`，且 `server_base_url` 偏好**没有 UI 写入入口**、shell 非 root 改不了
app 沙箱内的 preferences 文件（属主是 app uid，权限 660）。

连通性判据（设备内无 curl）：app 起来后
`hdc shell "hilog -x | grep A0FF08 | grep -E '响应 code|请求成功'"` 出现 `响应 code=0 (55ms)` 即通。

### 3. 重启后 Asset Store 可能「file operation failed」→ 卸载重装即恢复

`hdc shell reboot` 后 app 读 token 报
`TokenAssetStore: Failed to get asset geotalk_current_access: Error: The file operation failed.`
（区别于正常的 `The asset is not found.`）。
`hdc uninstall com.lsyb.social` + `install` 后恢复正常（错误回到 `not found` 常态）。

**2026-10-19 补充（同族故障的更强形态，重装/重启均未恢复）**：
- 症状：`TokenAssetStore … Error: **The cryptography operation failed.**`（读 token 时）；
  `uninstall` + `install` 后错误变成**持续** `The asset is not found.`；`hdc shell reboot` 亦不恢复。
- 影响面：**所有需认证的请求全线 `code=10100`**——实测登录本身成功（`signin` 367ms ✓），
  但随后的 `POST /admin/api/v1/rbac/me` 因 `getToken()` 读不到 Asset 而 401 →
  管理入口不渲染、需登录的页面拿不到数据。**游客态流程不受影响**（走 public 端点）。
- 排查手法：`hilog -x | grep -iE 'TokenAssetStore|rbac|code=10100'`——看到
  `TokenAssetStore … not found/cryptography` 紧接 `响应 code=10100` 即可判定是 Asset 故障，
  不是账号权限或后端问题（同一账号 `curl /admin/api/v1/rbac/me` 返回 `permissions:["*"]` ✓）。
- **修法已验证（2026-10-20）**：
  - `bm clean -n com.lsyb.social -d`（清 app 数据）+ 冷启 + 登录 → **无效**，登录后紧跟的认证请求仍 `code=10100`；
  - **重新克隆模拟器实例有效**：`pkill -9 -f Emulator` → 删工作区克隆 → 从
    `~/.Huawei/Emulator/deployed/<实例名>` 重新 `cp -c -R` → 重写 `config.ini` 的
    `configPath`/`logPath`/`instancePath` 指向工作区 → `Emulator -start <name> -instancePath <工作区>` →
    `hdc shell reboot` 后重设 rport。**验证**：TokenAssetStore 无报错、登录后认证请求全部 `code=0` ✓。
  - 结论：坏的是**旧克隆实例的系统状态**（Asset/HUKS 层），与 app 数据无关，也不是"模拟器不支持"。

- **更正（2026-10-20，重要）**：上一版把两件事记成「app 侧不稳定」，实测**都是自动化用法错误**：
  - **`uinput -K -d 2 -u 2` 的 key code 2 = HOME 键** —— 它被当成"收键盘"用，实际每次都把应用
    切回桌面（截图里看到的是 launcher）。收键盘请用 **`uitest uiInput keyEvent Back`**，
    且**只按一次**：连按两次会依次"关 IME → 退出应用"，同样回到桌面；
  - **点击坐标必须用实测值**：底栏中心是 `y=2599`，写成 `2620` 会落进系统手势区 → 同样切桌面。
  - 因此「点击导航后被切回桌面」不是 app 缺陷；「登录态不跨冷启」也**未验证**（多次流程其实没登录成功）。
- 需要管理后台走查时按「冷启 → 消息页登录 → **首次**挂载我的」执行，且**每一步都 dump 校验当前页**
  （`uitest dumpLayout`），发现页面不对就停，别继续盲点。

### 4. 登录与推入式页面：**可以验证**（2026-10-18 的"白屏 / 不可达"结论已作废，2026-10-19 复验）

2026-10-18 曾记录「点登录/快速体验后全白、推入式页面白屏 → 登录链路不可达、需要真机」。
**该结论已被 2026-10-19 实测推翻**，保留此条只为提醒后来者别据一次现象下环境结论：

- 复验结果：**推入式页面正常渲染**（瞬间详情页 31 节点；附近瞬间页 52 节点 + 8 张卡 + 截图正常）；
  **登录页正常渲染**，`uitest_a` / `UiTest#12345` **登录成功**（`signin` 389ms → 进消息列表），
  登录后的消息列表、个人中心均可走查。
- 当时看到的"整页空白"是**应用瞬时坏状态**，恢复手段：
  `aa force-stop com.lsyb.social` → `aa start -b com.lsyb.social -a EntryAbility`（冷启动）。
- **方法学教训**：当时只用 `dumpLayout`（无障碍树）判空、**没有截图**。页面可能"页头渲染正常、
  内容区全空"，dump 里同样有文本 → 只看节点数会把坏页面误判成正常页。**判断页面是否正常
  一律要用 `snapshot_display` 截图复核**。

> 保留的当期观察（供排障参考，与"页面空白"无因果关系）：`智能服务初始化失败 … 15100003`
> 每 10s 重试；定位恒失败 `3301100 The location switch is off`。

### 5. 无定位也能造出「附近群组」验证数据（本轮实用技巧）

app 的 `browse_anchor` 为 (0,0) 时启动即被丢弃（日志「持久化浏览锚点无效(如 (0,0))，丢弃」），
回退 `currentLocation`；模拟器定位关闭 → reference_point 为 null。
而 `groups/search` **不走**第 3 级默认位置兜底（实测无 reference_point 时返回 0 条），
需命中第 2 级：`POST /api/v1/users/update_location`（仅 `/api/v1` 登录请求命中本环）。

```bash
# 1) 给走查账号注入位置（服务端默认位置：35.154922, 112.682524「南天门」）
curl -s -X POST "$API/api/v1/users/update_location" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"location":{"point":{"latitude":35.154922,"longitude":112.682524},"location_name":"南天门"}}'
# 2) 用另一账号在该坐标建群（Idempotency-Key 必须是 {UUID}-{13位ms}-{8位随机}，见 spec/api_spec_base.md）
#    → 之后 app（登录同账号、无锚点）的附近群组即可看到这些群，距离 0m
```

已按此造好两个走查群：**走查群A-改前**（`group_553935e9eada256e4091f266`）、
**走查群B-改后**（`group_2fa2449f039e11f8845d5e80`），均为 `uitest_admin` 所有、`direct` 入群、
位于默认位置；uitest_a 未加入，可直接用于「加入态刷新」的端上复核。

## 2026-10-19 端上实测（IA 调整走查 + 历史结论更正）

### 走查通过项
- 首页 Tab：副标题「关注与发现」+ 段位「关注 / 发现」（游客默认「发现」✓ 冷启动矩阵生效）
  + 「热门动态」真实数据；**已无「附近活动/附近群组/人物推荐」区块**（IA 调整去重复）。
- 切「关注」段（未登录）→ `LoginPromptView`「登录后查看关注动态」+ 副标题 + 快速体验/登录入口 ✓。
- 附近 Tab（NearbyPage 枢纽）：统计卡「实时附近 / 5km 内 · 持续更新 / N 人附近 · N 动态 · N 在线」
  （由首页迁来，数据源为枢纽自身 nearby 状态）✓；三段段位逐个走查均渲染
  （附近的人「身边的新朋友」/ 附近群组「身边兴趣圈」/ 附近动态「周边新鲜事」）。

### 可复现的环境与操作事实

1. **首启会弹定位权限框**：文本「允许"GeoTalk"访问你的位置？」，「仅使用期间允许」
   中心 ≈ `(628,1964)`（bounds `[432,1931][824,1997]`）。不处理会挡住所有页面断言
   （dump 只有弹窗节点）。授权后不影响后续 Tab 走查（模拟器定位本身仍不可用）。
2. **偶发"整页空白" = 应用瞬时坏状态，不是模拟器限制**：恢复与判定口径见本文 §4
   （`aa force-stop` + `aa start` 冷启；必须截图复核，不能只看 `dumpLayout`）。
3. **页面只渲染页头、内容区全空 → 先查 ArkUI 裸 `{ }` 块**：
   ArkUI 的裸 `{ ... }` 不是容器，块内 `.layoutWeight(1)` 不生效 → 容器高度塌陷 →
   内部 WaterFlow/List 高度为 0 → 页头计数正常但一条内容都不渲染。修法是删掉裸块、
   让容器直接作为分支体。**判定手法**：dump 节点数（本例页头 10 节点 → 修复后 52 节点）。
4. **rport 流程**：热启动快照会残留 8888 监听 → `hdc rport` 报
   `TCP Port listen failed`；`hdc shell reboot`（约 42s 重新 Connected）后 rport **OK**，
   app 日志 `A0FF08 HttpUtil: 响应 code=0`，端上数据正常加载。

## 模拟器登录流程（2026-10-20 本轮实测坐标与坑）

**已实测可用**（分辨率 1256×2760，Pura 90 Pro 实例）：

1. 冷启后停在首页 → 点底部「消息」`(877, 2599)`；
2. 游客态消息页点 **「登录 / 注册」**（实测中心 `(628, 1279)`——注意不要用 `text.includes("登录")`
   匹配，会命中页面文案「消息功能需要登录」而点到非按钮文字上 ♂️）；
3. 登录页两个输入框实测：账号 `(628, 1716)`、密码 `(628, 1944)`，提交按钮文本为 **「登录」**
   （同页还有 tab「登录/注册」与「访客」，按坐标或出现顺序区分）；
4. 点「我的」`(1127, 2599)`——管理入口要求**登录后首次挂载**，所以顺序必须是
   冷启 → 消息 → 登录 → 我的。

**已定位的关键坑（2026-10-20 第三轮实测）**：密码框会弹出**安全键盘**（dump 里出现「安全键盘」节点），
它盖住两个输入框 —— 这就是"两次文本拼进同一字段"的真正原因（第二次点击落在安全键盘上，焦点没变）。

**实测可用的登录输入流程**：
1. 点账号框 `(628,1716)` → `inputText 'uitest_admin'` → **`keyEvent Back` 一次**（此时 IME 打开，Back 只关键盘 ✓）
   → dump 校验仍在登录页 ✓；
2. 点密码框 `(628,1944)` → `inputText 'UiTest#12345'`（会切安全键盘）→ **`keyEvent Back` 一次**关安全键盘
   → dump 校验仍在登录页 ✓且「安全键盘」节点消失 ✓；
3. 此时 dump 能看到两个字段分别持有 `uitest_admin` 与 `*************` ✓（值已正确分离）。

**提交按钮的正确点法**（2026-10-20 第四轮实测）：登录表单默认在**折叠线以下**，dump 只含可见节点，
所以要先 `fling` 往上滚一次，再从 dump 里按特征找按钮（`clickable=true` + `w>600` + `h>120` + `cy>1200`），
本轮定位到 `Button (628,2183) w=746 h=188`；点它确实触发了提交（出现校验提示）✓。

**未验证观察（记录下来，未定性为 app bug）**：提交后 dump 显示**账号字段被改写**为 `tUiTest#12345`
（= 密码串前多一个 `t`）、密码变短为 `************`，随后报
「密码需为 8-128 位且同时包含字母和数字」。两次干净输入后都复现同样的改写。可能是：
① 安全键盘期间输入串入账号框；② 登录表单在提交时的状态处理有缺陷（app 侧）；③ 自动化连续操作导致的级联状态。
**下次先用人工/真机点一遍同样流程**：若人手操作也出现改写 → app bug，需单独报；若不复现 → 属自动化问题。

**纪律**：`keyEvent Back` 只有在 **IME/安全键盘打开时**才安全（只关键盘）；键盘未开时它等于导航后退，
连按两次会退出应用（实测）。每次 Back 后都要 dump 校验页面没变。

## 走查脚本（避免每次从零写，坑已内置）

`scripts/walk.py`（本 skill 目录下）提供最小工具集：`dump()/nodes()/texts()/find()/inputs()/buttons()/
tap()/input_text()/fling()/back()/snapshot()`，可直接 import 复用；也可命令行快速点击：

```bash
python3 .agents/skills/hotspot-harmony-device-verify/scripts/walk.py 附近
```

脚本注释里固化了本文所有坑（坐标取自 dump、每步校验、Back 只在键盘打开时按、安全键盘、按钮在折叠线下先 fling）。
