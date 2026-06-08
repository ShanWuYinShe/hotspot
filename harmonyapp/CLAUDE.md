# CLAUDE.md

本文件只定义本子项目的技术约束与本地命令；通用工作流与 Git 规范以根 `CLAUDE.md` 为准。

## 构建 & 测试

DevEco Studio 工具路径：

```
NODE=/Applications/DevEco-Studio.app/Contents/tools/node/bin/node
HVIGORW=/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js
OHPM=/Applications/DevEco-Studio.app/Contents/tools/ohpm/bin/ohpm
DEVECO_SDK_HOME=/Applications/DevEco-Studio.app/Contents/sdk
JAVA_HOME=/Applications/DevEco-Studio.app/Contents/jbr/Contents/Home
PATH=$JAVA_HOME/bin:$PATH
```

- `$OHPM install --all --registry https://ohpm.openharmony.cn/ohpm/ --strict_ssl true` — 安装依赖
- `$NODE $HVIGORW --mode project -p product=default assembleApp --analyze=normal --parallel --incremental --daemon` — 编译整个应用
- `$NODE $HVIGORW assembleHap` — 仅编译 HAP 包（快速检查用）
- `$NODE $HVIGORW clean` — 清理构建产物

> 如果修正 `DEVECO_SDK_HOME` / `JAVA_HOME` 后仍复用旧环境，先运行 `$NODE $HVIGORW --stop-daemon`，再重新构建。

> **Lint & Format**: DevEco Studio 内置，无独立 CLI。代码检查由 IDE 在编辑时自动执行。

## 约束

- **API 路径**：统一在 `ApiEndpoints.ets` 定义。
- **HTTP 调用**：通过 `HttpUtil.post()` / `ApiClient` 发起。
- **响应判断**：业务成功唯一依据 `code === 0`。
- **认证**：Token 存储于 `PreferencesUtil`；`10100` 触发刷新或跳登录，`10201` 强制登出。
- **地图交互**：距离和角度由后端计算，客户端仅展示；坐标转换仅在 `CoordinateConverter.ets` 单一边界层执行一次。
