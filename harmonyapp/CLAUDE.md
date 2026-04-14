# CLAUDE.md

- 本文件只定义本子项目的技术约束与本地命令；仓库级 workflow、任务分级与验证边界以根 `CLAUDE.md` 为准。

## 构建 & 测试
- `./hvigorw assembleHap` — 构建
- `./hvigorw test` — 测试

## 约束
- **API 路径**：统一在 `ApiEndpoints.ets` 定义。
- **HTTP 调用**：通过 `HttpUtil.post()` / `ApiClient` 发起。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **认证**：Token 存储于 `PreferencesUtil`；`10100` 触发刷新或跳登录，`10201` 强制登出。
- **系统图标**：使用 `SymbolSpan` / `SymbolGlyph`，通过集中枚举引用。
- **地图交互**：距离和角度由后端计算，客户端仅展示；坐标转换仅在 `CoordinateConverter.ets` 单一边界层执行一次；marker、overlay、位置选择器使用地图 SDK 原生能力。
- **Safe Area**：通过统一能力计算和复用，不在页面内硬编码高度。
