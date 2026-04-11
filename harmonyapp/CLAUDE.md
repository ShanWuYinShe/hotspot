# CLAUDE.md

## 架构
HarmonyOS ArkTS 客户端。

- `entry/src/main/ets/entryability/`：应用入口
- `entry/src/main/ets/features/`：业务页面
- `entry/src/main/ets/components/`：公共组件
- `entry/src/main/ets/core/api/`：API 封装（ApiClient、HttpUtil）
- `entry/src/main/ets/core/state/`：全局状态管理
- `entry/src/main/ets/core/router/`：导航
- `entry/src/main/ets/core/geo/`：坐标转换

## 约束
- **API 路径**：统一在 `ApiEndpoints.ets` 定义。
- **HTTP 调用**：通过 `HttpUtil.post()` / `ApiClient` 发起。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **认证**：Token 存储于 `PreferencesUtil`；`10100` 触发刷新或跳登录，`10201` 强制登出。
- **系统图标**：使用 `SymbolSpan` / `SymbolGlyph`，通过集中枚举引用。
- **地图交互**：距离和角度由后端计算，客户端仅展示；坐标转换仅在 `CoordinateConverter.ets` 单一边界层执行一次；marker、overlay、位置选择器使用地图 SDK 原生能力，center pin 绑定相机中心。
- **Safe Area**：通过统一能力计算和复用，不在页面内硬编码高度。
