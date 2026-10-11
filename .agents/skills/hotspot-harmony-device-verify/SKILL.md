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
| 模拟器调试后端地址 | `http://10.0.2.2:8888`（SLIRP 用户态 NAT，实测可达） |
| 后端 dev 网关 | 宿主 `http://127.0.0.1:8888`（nginx） |
| 测试库 | postgres 容器 `hotspot-postgres-1`，库 `hotspot`，用户 `postgres` |

**AppConstants.ets 注释里写的 `127.0.0.1:8888 + hdc rport` 在本机模拟器上不通**：
设备侧 8888 已被占用，`hdc rport tcp:8888 tcp:8888` 报 `[Fail]TCP Port listen failed at 8888`。
本机模拟器走 `10.0.2.2:8888`（Preferences 里 `server_base_url` 也是这个值）。

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

## 已发现缺陷（2026-10-11，未修，复现步骤可用）

1. **同 mount 登录管理员后无管理员行**：先点过「我的」再登录管理员，该行不出现，
   切到首页再切回也不恢复（TabContent 常驻，不重新 mount）。根因见上「关键前提」。
2. **同 mount 切号残留上一账号管理员行（权限残留）**：管理员在「我的」内退出登录→改登普通账号，
   「管理员工具台」行仍显示。点击被服务端挡住（`10200`），但 app 把非管理员错误
   当网络错误展示（`AdminOverviewPage.ets:90` → “加载权限失败，请检查网络”），文案误导。
   根因同 1：`logout()` 未重置 `hasAdminPrivileges`。

修复方向：ProfilePage 订阅登录态变化（`@Watch(KEY_IS_LOGGED_IN)` 或
`@Watch(KEY_USER_TYPE)`）重新 `refreshAdminPrivileges()`，并在 `logout()` 里清零该状态；
AdminOverviewPage 区分 `10200` 与网络错误。

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


## 沙盒下启动模拟器：用 -instancePath 重定向，不要申请放宽权限（2026-10-12 实测）

**踩坑**：文件策略为 `workspace-write` 时启动模拟器会报

```
Emulator: Error opening logfile /Users/liyou/.Huawei/Emulator/deployed/Pura 90 Pro/Log//qemu.log: Operation not permitted
```

目录权限本身正常（`drwxr-xr-x`、属主是自己），返回的是 **EPERM 而非 EACCES** —— 这是
DSH 沙盒的拒绝签名（对照：写工作区成功）。**我当时误判为「必须申请 danger-full-access」，
并向 harness 申请了一次更宽权限，其实完全不需要**（用户当场质疑，复查后确认）。

**正确做法**：`Emulator -help` 里有 `-instancePath <path>`，把实例目录重定向进工作区即可：

```bash
SRC="/Users/liyou/.Huawei/Emulator/deployed/Pura 90 Pro"
DST="/Users/liyou/work/hotspot/.workwork/emu-instance"
mkdir -p "$DST"
cp -c -R "$SRC" "$DST/Pura 90 Pro"        # -c = APFS 克隆，5.6GB 近乎瞬时、不占额外空间
# 改克隆件的 config.ini：instancePath / logPath / configPath 三处全部指向 $DST
/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator \
  -start "Pura 90 Pro" -instancePath "$DST"   # 注意是**父目录**，不是实例目录本身
```

要点：
- `-instancePath` 传**父目录**；传实例目录本身会报
  `"X" is not found. Please create the device(folder): "<path>/X"`。
- 只读的系统镜像在 `~/Library/Huawei/Sdk/system-image/...`，沙盒**允许读**，无需搬运；
  实例目录里只有可写 overlay（`*.qcow2`）与 `ram.bin`。
- `config.ini` 里 `logPath` / `configPath` 默认指向 `~/Library/...`，也要一并改到工作区。
- 克隆件与真实实例**共享同一份用户数据快照**：里面装好的 App、登录态都会带过来。
- 收尾时删掉 `.workwork/emu-instance` 即可，真实实例不受影响。
- **结论**：这类「工具要写 home 目录」的问题优先走「① 换目录/重定向」，能靠工具自身参数
  解决就不要申请放宽沙盒；本机模拟器的重定向参数就是 `-instancePath`。
