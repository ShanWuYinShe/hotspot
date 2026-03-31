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

## 与根流程的关系
- 主目录保持 `main` + clean，所有实际改动都在 worktree 中完成；本地分支、本地集成 worktree、本地验证入口和风险边界，以仓库根 `CLAUDE.md` 为准；涉及 API、地图、Safe Area、登录态或 WebSocket 的改动时，再对照 `spec/`。

## 关键约束
- API 路径只在 `core/config/ApiEndpoints.ets` 定义，禁止业务代码硬编码。
- 统一通过 `HttpUtil.post()` / `ApiClient` 调接口，不要在业务页面直接封装 HTTP。
- 业务成功只看 `code === 0`。
- Token 存在 `PreferencesUtil`；`10100` 要刷新或跳登录，`10201` 要强制登出。
- 系统图标用 `SymbolSpan` / `SymbolGlyph`，并通过集中管理的图标枚举引用。
- 地图/位置不在客户端计算距离或角度。
- 可交互内容必须遵守 Safe Area。

## 地图与坐标转换边界
- `CoordinateConverter.ets` 只负责业务坐标与地图坐标之间的单次边界转换；不要在同一流程里重复正反转换。
- 当系统定位返回 WGS-84 而地图 SDK 需要 GCJ-02 时，进地图前做一次 WGS-84 → GCJ-02；从地图相机回写业务状态时，再做一次反向转换。
- marker、overlay、位置选择器中心点都必须使用地图 SDK 原生能力。
- center pin 必须绑定地图相机中心；业务状态中的当前位置以地图 SDK 返回的中心值为准。

## 图标与 Safe Area 规则
- 系统图标优先使用 Harmony 官方符号字体，通过统一图标枚举引用，不在业务代码里直接写 `sys.symbol.*`。
- 地图 marker、品牌图形等业务图形可以使用自定义资源，但不要混入系统图标管理方式。
- Safe Area 只通过统一能力计算和复用；不要在页面里硬编码状态栏、导航栏高度。

## 常用命令
- 本仓库默认最终验证入口以根 `CLAUDE.md` 约定为准。
- Harmony 工程的检查/编译配置入口主要在 `code-linter.json5`、`hvigorfile.ts`、`entry/hvigorfile.ts`、`hvigor/hvigor-config.json5`、`build-profile.json5`。
- 在非主目录 worktree 中，安装/检查/编译通过 `python3 .claude/scripts/harmony_worktree.py --task <install|check|build|all>` 执行；不直接调用 ohpm/hvigor 原始命令。
