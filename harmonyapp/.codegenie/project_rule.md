# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 这个客户端做什么
这是 HotSpot 的 HarmonyOS ArkTS 客户端。`entry/src/main/ets/features` 按业务域拆页面，`entry/src/main/ets/core` 负责 API、网络、状态、路由、存储、WebSocket、主题和工具，`components` 放可复用 UI。`Index.ets` / `EntryAbility.ets` 是入口，`AppRouter` 负责全局导航。

## 代码组织
- `entry/src/main/ets/entryability/EntryAbility.ets`：应用入口 Ability。
- `entry/src/main/ets/features/`：按业务域划分的页面。
- `entry/src/main/ets/components/`：公共 ArkUI 组件。
- `entry/src/main/ets/core/config/ApiEndpoints.ets`：唯一 API 路径定义。
- `entry/src/main/ets/core/api/ApiClient.ets`：按业务域封装的静态 API 方法。
- `entry/src/main/ets/core/network/HttpUtil.ets`：统一 HTTP 客户端。
- `entry/src/main/ets/core/state/`：GlobalStateManager、AuthAccountManager、PersistentStateManager 等全局状态。
- `entry/src/main/ets/core/router/AppRouter.ets`：全局导航入口。
- `entry/src/main/ets/core/types/`：API / WebSocket 类型。
- `entry/src/main/ets/core/network/WebSocketService.ets`：实时连接。
- `entry/src/main/ets/core/geo/CoordinateConverter.ets`：坐标转换。

## 关键约束
- API 路径只在 `core/config/ApiEndpoints.ets` 定义，禁止业务代码硬编码。
- 统一通过 `HttpUtil.post()` / `ApiClient` 调接口。
- 业务成功只看 `code === 0`。
- Token 存在 `PreferencesUtil`；`10100` 要刷新或跳登录，`10201` 要强制登出。
- 系统图标用 `SymbolSpan` / `SymbolGlyph`，并通过集中管理的图标枚举引用。
- 地图/位置不在客户端计算距离或角度。
- 可交互内容必须遵守 Safe Area。

## 常用命令
- 先安装依赖：`ohpm install --all --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true`
- 代码检查：`DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" DEVECO_SDK_NODE="/Applications/DevEco-Studio.app/Contents/tools/node/bin/node" CODELINTER="/Applications/DevEco-Studio.app/Contents/plugins/codelinter/run/index.js" "$DEVECO_SDK_NODE" "$CODELINTER" -c code-linter.json5 entry/src/main/ets/`
- `hvigorw` 编译由用户手动运行，不要代跑。
