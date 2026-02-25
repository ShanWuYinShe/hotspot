# HotSpot HarmonyOS App

基于 HarmonyOS ArkTS 的鸿蒙客户端应用。

## 技术栈

- **框架**: HarmonyOS ArkTS (.ets)
- **API 版本**: API 20 (SDK 6.0.0)
- **构建工具**: hvigor
- **网络**: @ohos.net.http
- **持久化**: @ohos.data.preferences
- **UI**: ArkUI 原生组件

## 项目结构

```
entry/src/main/ets/
├── entryability/            # 应用入口 Ability
├── components/              # 公共 ArkUI 组件
├── constants/               # 常量定义
│   ├── AppConstants.ets     # API_BASE_URL、WS_BASE_URL
│   ├── DesignConstants.ets  # 设计常量
│   └── SystemConstants.ets  # 系统常量
├── core/                    # 核心逻辑层
│   ├── api/                 # API 客户端封装
│   ├── config/              # API 端点常量
│   ├── network/             # HTTP/WebSocket 工具
│   ├── state/               # 全局状态管理
│   ├── storage/             # 持久化工具
│   ├── types/                # 类型定义
│   ├── di/                  # 依赖注入
│   ├── services/            # 各类服务
│   └── utils/               # 工具函数
└── features/                 # 按业务域划分的页面
    ├── auth/                # 认证
    ├── chat/                # 聊天
    ├── discover/            # 发现
    ├── group/               # 群组
    ├── map/                 # 地图
    ├── message/             # 消息
    ├── moment/              # 瞬间
    ├── nearby/              # 附近
    ├── profile/             # 个人资料
    ├── search/              # 搜索
    ├── settings/            # 设置
    └── subscriptions/       # 订阅
```

## 开发指南

### 环境要求

- **DevEco Studio**: 5.0+
- **HarmonyOS SDK**: 6.0.0 (API 20)
- **Node.js**: 18+

### 开发命令

```bash
# 使用 DevEco Studio 打开 harmonyapp 目录
# 通过 GUI 进行开发、编译和调试
```

### 编译检查

```bash
# 编译（检查编译错误）
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" \
node "/Applications/DevEco-Studio.app/Contents/tools/hvigor/bin/hvigorw.js" \
  --mode module -p product=default assembleHap \
  --info --analyze=ultrafine --parallel --incremental \
  --enable-build-script-type-check --no-daemon

# 代码检查（检查 IDE 中的 linter 警告）
DEVECO_SDK_HOME="/Applications/DevEco-Studio.app/Contents/sdk" \
node "/Applications/DevEco-Studio.app/Contents/plugins/codelinter/run/index.js" \
  -c code-linter.json5 entry/src/main/ets/
```

### AI Agent 规范

鸿蒙客户端代码规范见 [AGENTS.md](AGENTS.md)，开发前请务必阅读。

**常见错误检查清单**：
- [ ] 是否硬编码了 API 路径 `"/api/v1/..."`？
- [ ] 是否依赖了 HTTP 状态码而非 `code === 0`？
- [ ] 是否在客户端计算了距离/角度？
- [ ] 新增图标是否在 HarmonyOS 主题图标库验证了符号名存在？
- [ ] 是否使用了 `Image($r('sys.media.*'))` 而可以改用 `SymbolSpan`？

## API 调用规则

所有 API 路径必须定义在 `core/config/ApiEndpoints.ets` 中，代码中禁止出现硬编码路径。

```typescript
// 正确
const result = await HttpUtil.post<ApiResponse<AuthData>>(
  ApiEndpoints.AUTH.LOGIN, data
);

// 错误
const result = await HttpUtil.post<ApiResponse<AuthData>>(
  '/api/v1/auth/login', data
);
```

响应处理只判断 `code === 0`，不依赖 HTTP 状态码。

## 图标规范

统一使用 HarmonyOS 主题图标库 `sys.symbol.*`，禁止在业务代码中直接写 `$r('sys.symbol.xxx')`。

所有图标名称在 `components/SystemIcon.ets` 的 `AppIcon` 枚举中统一管理。

## 相关文档

- [API 规范](../spec/api_spec_index.md)
- [WebSocket 事件](../spec/api_spec_websocket.md)
