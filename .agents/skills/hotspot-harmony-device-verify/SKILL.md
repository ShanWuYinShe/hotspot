---
name: hotspot-harmony-device-verify
description: hotspot 鸿蒙端模拟器实测流程：项目事实与测试账号造数、模拟器/反向转发环境准备、hdc + uitest 走查流程（登录/断言/截图/锚点/密度标定）、安全键盘与 stale 坐标坑、报错排障。触发：需要验证鸿蒙端某页面在不同角色下的 UI；在模拟器上跑 harmonyapp 走查或截图；hdc 连模拟器、uitest dumpLayout/click/fling；排查「个人中心/管理入口为何不显示」；本地 dev 环境造测试账号或附近群组数据。
---

# hotspot 鸿蒙端模拟器实测流程（hdc + uitest）

> 通用 hdc 行为见用户级 skill `harmony-device-debug`；本文件只记 hotspot 项目专属事实与实测流程。**环境类结论必须附验证日期 + 复现步骤，沿用前先证伪一次**（项目 AGENTS.md「环境类结论必须可复现」）。

## 项目事实（唯一维护处）

| 项 | 值 |
|---|---|
| bundleName | `com.lsyb.social` |
| 入口 Ability | `EntryAbility` |
| 应用日志域 | `A0FF` 前缀（`grep A0FF` 抓 app 日志） |
| 模拟器调试后端地址 | `http://127.0.0.1:8888`（hdc rport 反向转发） |
| 后端 dev 网关 | 宿主 `http://127.0.0.1:8888`（nginx） |
| 测试库 | postgres 容器 `hotspot-postgres-1`，库 `hotspot`，用户 `postgres` |

**调试后端地址**：`AppConstants.ets` 现行代码统一 `http://127.0.0.1:8888 + hdc 反向转发`（模拟器 `hdc -t 127.0.0.1:5555 rport tcp:8888 tcp:8888`，真机 USB `fport`）。模拟器重启后转发条目不会自动恢复，装机前先设好，并以 `hdc fport ls` 出现 `[Reverse]` 条目为准；app 内请求实测 HTTP 200 / code=0（63ms）。`http://10.0.2.2:8888` 保留为 rport 失效时的备选直连（SLIRP 用户态 NAT，见下「环境准备」）。

## 测试账号与造数（本地 dev 库）

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

造新账号：正式用户走 `POST /public/v1/auth/register`（用户名型，`verification_code: null`，必带 `Idempotency-Key`）；授予管理员需 SQL 直插（`ADMIN_BOOTSTRAP_USER_ID` 只在库中无管理员时生效）：

```sql
INSERT INTO admin_users (user_id, grant_all_permissions) VALUES ('<user_id>', TRUE)
ON CONFLICT (user_id) DO UPDATE SET grant_all_permissions = TRUE, revoked_at = NULL;
```

**无定位也能造出「附近群组」验证数据**：app 的 `browse_anchor` 为 (0,0) 时启动即被丢弃（日志「持久化浏览锚点无效(如 (0,0))，丢弃」），回退 `currentLocation`；模拟器定位关闭 → reference_point 为 null。而 `groups/search` **不走**第 3 级默认位置兜底（实测无 reference_point 时返回 0 条），需命中第 2 级：`POST /api/v1/users/update_location`（仅 `/api/v1` 登录请求命中本环）。

```bash
# 1) 给走查账号注入位置（服务端默认位置：35.154922, 112.682524「南天门」）
curl -s -X POST "$API/api/v1/users/update_location" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"location":{"point":{"latitude":35.154922,"longitude":112.682524},"location_name":"南天门"}}'
# 2) 用另一账号在该坐标建群（Idempotency-Key 必须是 {UUID}-{13位ms}-{8位随机}，见 spec/api_spec_base.md）
#    → 之后 app（登录同账号、无锚点）的附近群组即可看到这些群，距离 0m
```

已按此造好两个走查群：**走查群A-改前**（`group_553935e9eada256e4091f266`）、**走查群B-改后**（`group_2fa2449f039e11f8845d5e80`），均为 `uitest_admin` 所有、`direct` 入群、位于默认位置；uitest_a 未加入，可直接用于「加入态刷新」的端上复核。

## 环境准备：模拟器、反向转发与恢复

### 1) 启动模拟器（沙盒内直接直启）

`~/.Huawei/Emulator/deployed`（实例镜像，含 `*.qcow2`/`ram.bin`）已由 DSH sandbox-extra-roots 插件放行，沙盒内直接 `Emulator -start <实例名>` 即可（实测约 10s 就绪、`hdc list targets` 出现 `127.0.0.1:5555`）。**不要再把实例克隆到 `.workwork/` 或改 `config.ini` 重定向路径**（旧做法已废弃）。

- `-stop` 可能 hang，清场用 `pkill -9 -f Emulator`。
- 只读系统镜像在 `~/Library/Huawei/Sdk/system-image/...`，读不受限，无需搬运。
- 若报 `EPERM ... Operation not permitted`（注意是 **EPERM 而非 EACCES**，属沙盒拒绝签名）：把报错路径加进 sandbox-extra-roots 的 `extraWritableRoots`（见 `sandbox-adaptation` skill），热生效后重试即可；不要申请 `danger-full-access`。
- **实例目录 EPERM 的具体形态**：`Emulator -start "Pura 90 Pro"` 报 `Error opening logfile /Users/<me>/.Huawei/Emulator/deployed/Pura 90 Pro/Log//qemu.log: Operation not permitted` 且**进程直接没起来**（`pgrep -f Emulator` 为空）——但 hdc 可能仍留 `127.0.0.1:5555 Offline` 僵尸条目，别把它当「模拟器在启动中」等下去。实测实例目录 `touch` 也是 EPERM（沙盒拒绝签名，不是文件权限问题）；解法同上（把实例目录加进 `extraWritableRoots`）。

### 2) 反向转发（rport）与热启动残留

- 热启动快照（`isHotBoot=true`）会带回上次会话的 8888 反向转发：`hdc rport tcp:8888 tcp:8888` → `[Fail]TCP Port listen failed at 8888`；`hdc shell "netstat -tlnp | grep 8888"` → `127.0.0.1:8888 LISTEN`（占用者是 `hdcd`）。该残留监听**不再转发**：app 打 127.0.0.1:8888 会 15s 超时（hilog 里 `tcpConnE:115`、`lastRecvE:0`）。
- **解法（实测）**：`hdc shell reboot` → 等 `Connected`（约 42s）→ 再 `rport tcp:8888 tcp:8888` 即 OK（冷启动清掉残留套接字）。别用 `rport tcp:8899 tcp:8888` 绕——app 的 `DEV_SERVER` 固定 `http://127.0.0.1:8888`，且 `server_base_url` 偏好**没有 UI 写入入口**、shell 非 root 改不了 app 沙箱内的 preferences 文件（属主是 app uid，权限 660）。
- 连通性判据（设备内无 curl）：app 起来后 `hdc shell "hilog -x | grep A0FF08 | grep -E '响应 code|请求成功'"` 出现 `响应 code=0 (55ms)` 即通。

### 3) 仍连不上 dev 后端时（现象与排查顺序）

现象：App 停在「加载中…」或各页报「加载…失败」，而宿主 `docker compose ps` 全 healthy、`curl localhost:8888/healthz` 正常。原因通常不在后端：

- App 的开发接口常量是 `DEV_SERVER = 'http://127.0.0.1:8888'`（`constants/AppConstants.ets`），在模拟器里 `127.0.0.1` 指模拟器自身，到达不了宿主；rport 失效时（如上述残留监听）请求就会超时。
- 排查顺序：先 `ping 10.0.2.2`（通=模拟器网络没问题）→ 再 `hdc fport ls`/`netstat` 看 rport → 再核对常量值 → 最后才怀疑后端。
- **可用做法（临时）**：把 `DEV_SERVER` 临时改成 `http://10.0.2.2:8888`（QEMU 宿主别名，模拟器内 `ping 10.0.2.2` 通）→ 构建安装 → 验证完**还原常量、不提交**（worktree 内改动，验证后删 worktree）。

### 4) 重启后 Asset Store 故障（认证全挂、游客不受影响）

- 症状：app 读 token 报 `TokenAssetStore: Failed to get asset geotalk_current_access: Error: The file operation failed.`（区别于正常的 `The asset is not found.`）；更强形态是 `The cryptography operation failed.`。
- 影响面：**所有需认证的请求全线 `code=10100`**——登录本身可能成功（`signin` 367ms ✓），但随后 `getToken()` 读不到 Asset 而 401 → 管理入口不渲染、需登录的页面拿不到数据。**游客态流程不受影响**（走 public 端点）。
- 排查：`hilog -x | grep -iE 'TokenAssetStore|rbac|code=10100'`——看到 `TokenAssetStore … not found/cryptography` 紧接 `响应 code=10100` 即可判定是 Asset 故障，不是账号权限或后端问题（同一账号 `curl /admin/api/v1/rbac/me` 返回 `permissions:["*"]` ✓）。
- 轻形态（`file operation failed`）：`hdc uninstall com.lsyb.social` + 重装后恢复正常（错误回到 `not found` 常态）。强形态（`cryptography operation failed`）：重装/重启均不恢复，按下面的修法处理。
- **修法已验证**：`bm clean -n com.lsyb.social -d`（清 app 数据）+ 冷启 + 登录 → **无效**（登录后紧跟的认证请求仍 `code=10100`）；**换成干净的系统状态有效**：`pkill -9 -f Emulator` → 新建一个模拟器实例（Device Manager，旧实例保留；或备份后重置实例快照）→ `Emulator -start <实例名>` → `hdc shell reboot` 后重设 rport。验证：TokenAssetStore 无报错、登录后认证请求全部 `code=0` ✓。结论：坏的是**实例的系统状态**（Asset/HUKS 层），与 app 数据无关，也不是「模拟器不支持」。

### 5) 登录态可直接看文件

`/data/app/el2/100/base/com.lsyb.social/haps/entry/preferences/geotalk` 是明文 XML（`auth_accounts` / `current_user_id` / `server_base_url`，shell 用户可读）。

## 走查流程（已验证）

1. **连接**：`hdc list targets -v` 出现 `Connected`（本机模拟器为 `127.0.0.1:5555`）。
2. **冷启动**：`aa force-stop com.lsyb.social` → `aa start -b com.lsyb.social -a EntryAbility`。
3. **登录**：可从「消息」Tab 的登录提示进入（角色行刷新已内置，不要求先后顺序）；坐标一律 `dumpLayout` 取 bounds 后点击，历史实测基线见 `references/coords-baselines.md`。
4. **进个人中心**：点底部「我的」Tab，然后 `uitest uiInput fling 628 2150 628 900 3500` 滚到动作行。
5. **断言**：`uitest dumpLayout -p /data/local/tmp/ui.json` 后逐节点取 text，判断「升级为完整账号」/「管理员工具台」/「设置」/「退出登录」是否存在。
6. **截图**：`snapshot_display -f /data/local/tmp/x.jpeg` → `hdc shell "cat …" > local.jpeg`（`hdc file recv` 会 hang，见下）。
7. **设置浏览锚点**：首页空态点「设置浏览位置」进地图选择器后——
   - 放锚点用 `uitest uiInput longClick <x> <y>`（长按放置；单击无效，双击是地图缩放），长按即写 `anchor_history` 偏好并触发逆地理解析；再点「设为浏览锚点」确认。**锚点模式确认后不关页**（toast「浏览锚点已设置」），需自己返回首页——切 Tab 前注意先返回，否则 Tab 点击落空；
   - 模拟器**系统地理编码无数据**：地点搜索（geocodeMultiple 走系统 geocoder）对任何关键词都返回「未找到该地点」，且锚点名只会是「未知位置」——这是模拟器环境限制，真机不受影响；历史锚点持久化在偏好文件 `anchor_history`；
   - 快速体验游客的锚点与会话态（AppStorage）**冷启动即清**，重装(-r)保留偏好但锚点需重新设置；暗色等偏好（`dark_mode_preference`）在 `geotalk` 偏好文件中持久化。
8. **密度/尺寸标定**：本机模拟器（Pura 90 Pro 实例）密度 ≈3.917（1256px / ≈320vp 逻辑宽），dump bounds 换算 vp 用 3.917，不要想当然用 3.25（曾因密度误算误判骨架卡/空态过大）。

## 登录/切号后的角色行刷新（已内置，无需特定顺序）

- `ProfilePage.aboutToAppear()` 时 `interactionEventHub.on('auth_user_changed', ...)` → 后台 `refreshAdminPrivileges()`（不阻塞 UI，结果经 `@State` 自动刷新）；`aboutToDisappear()` 注销监听。登出时先同步 `hasAdminPrivileges=false` 再走异步事件，避免登出瞬间残留上一账号的管理员行。
- 因此**不再需要**「登录必须发生在『我的』Tab 首次挂载之前」的顺序纪律；冷启 → 消息 → 登录 → 我的、或先点「我的」再从页内登录入口登录，两种路径都应得到正确角色行。
- 若管理员行不出现：先看请求有没有发（`hilog -r` 清缓冲 → 后台 `hdc shell hilog > log.txt` → 操作 → `grep -E 'rbac|ProfilePage' log.txt`），再核对服务端 `POST /admin/api/v1/rbac/me` 返回——`10200` 是「非平台管理员」业务终态（AdminOverviewPage 已与网络错误分开展示），不是 bug。

## 角色 → 动作行预期

| 角色 | 升级为完整账号 | 管理员工具台 | 设置 | 退出登录 |
|---|---|---|---|---|
| 匿名 | —（整页 LoginPromptView） | — | — | — |
| 游客 temp | ✅ | ❌ | ✅ | ✅ |
| 正式 normal | ❌ | ❌ | ✅ | ✅ |
| 管理员 | ❌ | ✅ | ✅ | ✅ |

动作行顺序：升级行在最前（仅 temp），然后 更新位置 → 设置 → [管理员工具台] → 退出登录。管理员行点击落到「管理后台」概览页（`PageNames.ADMIN_OVERVIEW`）。

## 登录流程（坐标、安全键盘与「串位」根因）

**已实测可用**（分辨率 1256×2760，Pura 90 Pro 实例；坐标随构建会变，一律以 dump 为准，历史基线见 `references/coords-baselines.md`）：

1. 冷启后停在首页 → 点底部「消息」`(877, 2599)`；
2. 游客态消息页点 **「登录 / 注册」**（实测中心 `(628, 1279)`——注意不要用 `text.includes("登录")` 匹配，会命中页面文案「消息功能需要登录」而点到非按钮文字上）；
3. 登录页两个输入框按 dump 实时坐标；提交按钮文本为 **「登录」**（同页还有 tab「登录/注册」与「访客」，按坐标或出现顺序区分）；
4. 点「我的」`(1127, 2599)`，确认角色行符合预期（动作行刷新由 auth 事件驱动，不依赖挂载顺序）。

**安全键盘坑**：密码框会弹出**安全键盘**（dump 里出现「安全键盘」节点），它盖住两个输入框——这就是「两次文本拼进同一字段」的常见原因（第二次点击落在安全键盘上，焦点没变）。

**实测可用的登录输入流程**：

1. 点账号框 → 用 `uitest uiInput inputText <x> <y> 'uitest_admin'` 输入 → **`uitest uiInput keyEvent Back` 一次**（此时 IME 打开，Back 只关键盘 ✓）→ dump 校验仍在登录页 ✓；
2. 点密码框 → `inputText 'UiTest#12345'`（会切安全键盘）→ **`keyEvent Back` 一次**关安全键盘 → dump 校验仍在登录页 ✓ 且「安全键盘」节点消失 ✓；
3. 此时 dump 能看到两个字段分别持有 `uitest_admin` 与 `*************` ✓（值已正确分离），再点「登录」。

**提交按钮的正确点法**：登录表单默认在**折叠线以下**，dump 只含可见节点，所以要先 `fling` 往上滚一次，再从 dump 里按特征找按钮（`clickable=true` + `w>600` + `h>120` + `cy>1200`）；历史实测按钮 `(628,2442)`，点它确实触发提交。

**「账号/密码串位」的根因 = 自动化 stale 坐标（不是 app bug）**：`uitest uiInput inputText <x> <y> <text>` 会**先按 (x,y) 点一下**再输入；键盘弹出后窗口上移，原先的输入框 y 坐标此时落在**键盘按键**上——那一下会先敲进一个字符，再输入文本（曾观察到 `tUiTest#12345`，多出的 `t` 就是落点处键盘键）。

- **复现实验**：密码框输入 `abc`（3 字符）→ dump 显示 `****`（4 个）；先点输入框、**重新 dump 取键盘弹出后的实时坐标**再 inputText，`abc` → `***`（正好 3 个）。
- **正确用法（已跑通）**：点输入框 → dump 取**当前**输入框坐标 → `inputText` 用该坐标 → `Back` 收键盘。每次键盘开合后窗口坐标都会变，**不能复用键盘前的坐标**。
- **纪律**：任何「输入串位 / 多字符 / 少字符」的端上现象，先用「输入 N 字符 → dump 看长度/掩码位数」做自证；长度不对先查坐标与键盘，再考虑 app 缺陷。
- `keyEvent Back` 只有在 **IME/安全键盘打开时**才安全（只关键盘）；键盘未开时它等于导航后退，连按两次会退出应用。每次 Back 后都要 dump 校验页面没变。

## 踩坑与纪律

### 走查铁律：截图与 dumpLayout 必须**同一次调用**获取

用截图目测判断布局位置时极易与 dumpLayout 的坐标不一致——两次命令之间页面可能仍在滚动/重排（尤其刚 fling 过或有入场动画）。曾连续 3 次误判（动态详情长文「行高没生效」、首页空态卡片「偏上」、消息页「最近」标签 x 值），根因都是截图与 dump 处于不同滚动位置。

**正确做法**：把 dumpLayout 与 snapshot_display 放进**同一条 shell 命令**，且**先 dump 后截图**（两者之间不插入任何滚动/点击）：

```bash
hdc shell uitest dumpLayout -p /data/local/tmp/x.json
hdc shell "cat /data/local/tmp/x.json" > /tmp/x.json
hdc shell snapshot_display -f /data/local/tmp/y.jpeg
hdc shell "cat /data/local/tmp/y.jpeg" > shot.jpeg
```

**判定位置类问题只信 dumpLayout 的坐标**（布局树真值），截图只用于看观感（层级/密度/配色）；两者冲突时以 dumpLayout 为准。

### 其他实测坑

- **`hdc file recv` 从模拟器拉文件会 hang 到超时**（exit=124）→ 改用 `hdc shell "cat /data/local/tmp/x.jpg" > local.jpg`（字节一致）。
- **滚轮/拖拽不滚动 ArkUI 列表**：`uinput -M -s`、`uinput -M -g` 实测无效，`uinput -T -g` 报「wrong number of parameters」；滚动一律 `uitest uiInput fling`。
- **软键盘弹出后原坐标失效**（窗口上移）：先收键盘或重新 dumpLayout。
- **`uitest uiInput click` 坐标越界会静默点空**（不报错），必须用 bounds 中心点。
- **临时账号 token 时效短**：「快速体验」的临时账号 token 约 30 分钟内失效；跨轮走查（间隔 >30min）后应用回到「个人中心需要登录」态，需重新 快速体验 → 开始体验。走查批次尽量在同一时效窗口内完成；「开始体验」按钮坐标每次以 dump 定位（弹层出现位置稳定在 y≈2358，但勿硬编码——曾在 2255 点空）。
- **首启会弹定位权限框**：文本「允许"GeoTalk"访问你的位置？」，「仅使用期间允许」中心 ≈ `(628,1964)`（bounds `[432,1931][824,1997]`）。不处理会挡住所有页面断言（dump 只有弹窗节点）；授权后不影响后续 Tab 走查（模拟器定位本身仍不可用）。
- **偶发「整页空白」= 应用瞬时坏状态，不是模拟器限制**：恢复手段 `aa force-stop` → `aa start` 冷启。**反模式**：只用 `dumpLayout`（无障碍树）判空、不截图——页面可能「页头渲染正常、内容区全空」，dump 里同样有文本，只看节点数会把坏页面误判成正常页。**判断页面是否正常一律用 `snapshot_display` 截图复核**。
- **页面只渲染页头、内容区全空 → 先查 ArkUI 裸 `{ }` 块**：块内 `.layoutWeight(1)` 不生效 → 容器高度塌陷 → 内部 WaterFlow/List 高度为 0。修法/判定见 skill `hotspot-harmony-ui`「审计基线与布局陷阱（指针）」列出的 ArkUI 陷阱明细（dump 节点数：本例页头 10 → 修复后 52）。
- **常见无害日志**：`智能服务初始化失败 … 15100003` 每 10s 重试、定位恒失败 `3301100 The location switch is off`——属模拟器环境常态，与页面渲染无因果关系。
- **新增纪律**：**只有 dump 确认当前是推入页（有返回入口）时才按 `keyEvent Back`**——在「点了没反应」的页面（如游客态通知）按 Back 等于退到上一页/退出应用，实测直接退回桌面（随后所有点击都落在桌面上）。

## 走查脚本（避免每次从零写，坑已内置）

`scripts/walk.py`（本 skill 目录下）提供最小工具集：`dump()/nodes()/texts()/find()/inputs()/buttons()/tap()/input_text()/fling()/back()/snapshot()`，可直接 import 复用；也可命令行快速点击：

```bash
python3 .agents/skills/hotspot-harmony-device-verify/scripts/walk.py 附近
```

脚本注释里固化了本文所有坑（坐标取自 dump、每步校验、Back 只在键盘打开时按、安全键盘、按钮在折叠线下先 fling）。

## 免登录可达页面与顶栏坐标

顶栏动作按钮坐标、免登录与需登录页面清单见 `references/coords-baselines.md`。游客态「通知」点之无反应、「地图/选点」未打开——后者记为**待查**，不要当成「已走查」。

## references

- `references/coords-baselines.md`：登录/顶栏/定位弹窗实测坐标、免登录与需登录页面清单、换算基线。
