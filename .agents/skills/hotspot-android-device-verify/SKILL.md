---
name: hotspot-android-device-verify
description: hotspot Android 端模拟器实测流程：AVD 创建与启动、adb reverse 联调（含端内 10.0.2.2 超时判据）、登录与页面走查（uiautomator dump + input + screencap）、造数与清理。触发：需要在 Android 模拟器/真机上安装运行 hotspot Android 客户端并截图走查；验证 androidapp 某页面 UI（首页/详情/评论/我的/设置）；排查模拟器网络不通、请求超时、登录失败；需要端上复现设计稿契约。
---

# hotspot Android 端模拟器实测（emulator + adb + uiautomator）

> 通用 adb/emulator/JDWP 行为见用户级 skill `android-emulator-debug`；本文件只记 hotspot 项目专属事实与实测流程。
> **环境类结论必须附验证方式（命令/取数口径）+ 复现步骤，沿用前先证伪一次**（项目 AGENTS.md「环境类结论必须可复现」）。

## 项目事实（唯一维护处）

| 项 | 值 |
|---|---|
| applicationId / 入口 Activity | `com.hotspot.app` / `.MainActivity` |
| 调试后端 | 宿主 dev compose 网关 `http://127.0.0.1:8888`（nginx） |
| 端内基址 | 默认 `http://10.0.2.2:8888`；联调固定用 `adb reverse` + `-PHOTSPOT_DEBUG_BASE_URL=http://127.0.0.1:8888` 覆盖构建 |
| 走查账号 | `qa@hotspot.local` / `Qa12345678`（seed 普通用户；`moment_seed_01` 自带评论供评论区走查） |
| 造数 | 主库；用 `client_type=web` 身份 curl 造评论，避免影响模拟器内的 App 会话 |
| 截图 / dump 落盘 | `.workwork/android-verify/`（任务收尾删除） |
| AVD | `HotSpotPhone`（`system-images;android-36;google_apis;arm64-v8a` + **`pixel_2` 设备档**，面板覆写为 1080×2400/420dpi；无挖孔） |

## 环境准备

### 系统镜像与 AVD（一次性）

```bash
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
export ANDROID_HOME="$HOME/Library/Android/sdk"
yes | "$ANDROID_HOME/cmdline-tools/latest/bin/sdkmanager" --install "system-images;android-36;google_apis;arm64-v8a"
echo no | "$ANDROID_HOME/cmdline-tools/latest/bin/avdmanager" create avd \
  -n HotSpotPhone -k "system-images;android-36;google_apis;arm64-v8a" -d pixel_2 --force
# 设备档 pixel_2 无挖孔；面板覆写为 1080×2400/420dpi（与设计 390 宽同密度）
CFG=~/.android/avd/HotSpotPhone.avd/config.ini
sed -i '' -E 's/^hw\.lcd\.height *=.*/hw.lcd.height = 2400/' "$CFG"
sed -i '' -E 's/^hw\.lcd\.width *=.*/hw.lcd.width = 1080/' "$CFG"
sed -i '' -E 's/^hw\.lcd\.density *=.*/hw.lcd.density = 420/' "$CFG"
```

镜像解压后约 4.3G；同一时间只跑一个模拟器（与鸿蒙模拟器互斥，16G 内存机器上同时开会挤）。

**设备档不要用 `pixel_7`**：其挖孔 cutout 会让系统把状态栏高度报成 136px（52dp），而 `-no-window` 又不渲染
可见挖孔——截图会表现为「顶部安全区过大」的假象（App 本身是按系统 inset 布局的）。判据：
`adb shell dumpsys window | grep "type=statusBars frame"` 应为 `[0,0][1080,63]`（24dp）；若为 136 则 AVD 档位不对。

### 启动与等待 boot

```bash
"$ANDROID_HOME/emulator/emulator" -avd HotSpotPhone \
  -no-window -no-audio -no-boot-anim -gpu swiftshader_indirect -no-snapshot \
  -netdelay none -netspeed full      # 首次 / 需要重置状态时加 -wipe-data
# 就绪判据：轮询 adb shell getprop sys.boot_completed 输出 1（带超时上限；冷启动耗时不固定）
```

## 网络联调（关键事实）

**本机 API 36 AVD 实测**：端内直接请求 `http://10.0.2.2:8888` 会在 connectTimeout 后报「网络超时」，
且宿主 nginx **没有**对应访问日志；同期 `adb shell nc 10.0.2.2 8888` 可连通——**shell 连通性不能作为
端内可达判据**。复现：默认基址构建安装 → 登录页提交 → 看 `docker logs --since 60s hotspot-nginx-1 |
grep 'POST /public/v1/auth/signin'`（无输出即未到达）。

联调固定走反向转发：

```bash
adb reverse tcp:8888 tcp:8888          # 模拟器重启 / adb server 重启后需重设
cd androidapp
JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home" \
ANDROID_HOME="$HOME/Library/Android/sdk" \
./gradlew --no-daemon assembleDebug testDebugUnitTest \
  -PHOTSPOT_DEBUG_BASE_URL=http://127.0.0.1:8888
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell am force-stop com.hotspot.app
adb shell am start -W -n com.hotspot.app/.MainActivity
```

端内可达判据：`docker logs --since 60s hotspot-nginx-1 | grep -E 'POST /(public|api)/v1'` 出现请求 +
截图出现真实数据（首页信息流、详情评论列表）。

## 安装、拉活、截图

```bash
adb install -r app/build/outputs/apk/debug/app-debug.apk
adb shell am start -W -n com.hotspot.app/.MainActivity
adb exec-out screencap -p > "$PWD/.../shot.png"        # 用绝对路径，避免落到奇怪 cwd
adb shell uiautomator dump /sdcard/ui.xml && adb pull /sdcard/ui.xml /tmp/ui.xml
```

## 走查流程（登录 → 页面）

1. **登录**：`uiautomator dump` 取 `EditText` bounds → `input tap` 聚焦 → `input text`；
   逐框「点 → 输 → `input keyevent 4` 收键盘」再点下一个框（键盘弹起会移动控件，沿用旧坐标会点到上一个框）。
   提交按钮取 dump 中 `text="登录"` 且 y 最大者的中心点；点击后等 10s 再截图（成功进首页，失败在按钮下方出红字错误）。
2. **底部导航**：坐标一律从 dump 取——当前 1080×2400 布局里底部 tab 行在 `y≈2285`，拍脑袋用内容区坐标会点进卡片。
3. **页面入口**：首页卡片按文本定位（如「云龙湖的落日把水面染成金色」）取节点 bounds 中心点击；
   详情页滚到评论区：`adb shell input swipe 540 2000 540 800 300`。
4. **断言以截图为准**：方案 B 评论区的导引线、作者微徽章、回复头「回复 @昵称」、docked 输入条与键盘避让
   都要截图确认；dump 只能证明存在节点，页头正常而内容区空时 dump 也会显示文本。

## 造数（主库，不 reset seed）

```bash
BASE=http://localhost:8888
# 用 web 身份登录，避免挤掉模拟器内的 App 会话
curl -s -c /tmp/c.txt -X POST "$BASE/public/v1/auth/signin" -H 'Content-Type: application/json' \
  -d '{"login_id":"qa@hotspot.local","password":"Qa12345678","client_type":"web","device_id":null}'
curl -s -b /tmp/c.txt -X POST "$BASE/api/v1/comments/create" -H 'Content-Type: application/json' \
  -H "Idempotency-Key: $(uuidgen | tr 'A-Z' 'a-z')" \
  -d '{"moment_id":"moment_seed_01","content":"端上验证评论","parent_id":null,"mentions":[]}'
```

## 清理（收尾必做）

```bash
adb emu kill                                   # 实测可用：输出 OK: killing emulator, bye bye
adb reverse --remove-all
pgrep -fl 'GradleDaemon|KotlinCompileDaemon'   # 应为空（构建一律 --no-daemon）
rm -rf .workwork/android-verify                # 截图 / 布局 dump 属中间产物
```
