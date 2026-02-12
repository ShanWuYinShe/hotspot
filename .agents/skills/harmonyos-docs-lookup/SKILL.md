---
name: harmonyos-docs-lookup
description: Fetch and read HarmonyOS NEXT / OpenHarmony official documentation on demand. Use when needing ArkTS API references, ArkUI component docs, state management guides, or any HarmonyOS development documentation.
compatibility: Requires WebFetch or WebSearch tool access.
metadata:
  category: documentation
  source: github.com/openharmony/docs
---

# HarmonyOS 文档获取

`developer.huawei.com` 无法通过 WebFetch 获取（超时）。使用 GitHub 上 OpenHarmony/docs 仓库的 raw 地址获取文档原文。

## 获取流程

```
Step 1: 根据下方路径规则构造 URL
Step 2: WebFetch https://raw.githubusercontent.com/openharmony/docs/master/{path}
Step 3: 404 → 先获取该目录的 Readme-CN.md 索引，搜索正确文件名后重试
Step 3 alt: 仍然找不到 → WebSearch "site:developer.huawei.com harmonyos {关键词}" 从搜索摘要获取信息
```

浏览目录结构：`https://github.com/openharmony/docs/tree/master/zh-cn/application-dev/reference/apis-{kit}`

## 文档路径规则

所有路径相对于仓库根目录，中文前缀 `zh-cn/`，英文 `en/`。

### ArkUI 组件 API

索引：`zh-cn/application-dev/reference/apis-arkui/Readme-CN.md`

组件文档在 `apis-arkui/arkui-ts/` 子目录下，命名规则 `ts-{category}-{name}.md`：

| category | 含义 | 示例 |
|----------|------|------|
| `container` | 布局容器 | ts-container-column.md, ts-container-list.md, ts-container-tabs.md |
| `basic-components` | 基础组件 | ts-basic-components-text.md, ts-basic-components-image.md, ts-basic-components-navigation.md |
| `media-components` | 媒体组件 | ts-media-components-video.md |
| `drawing-components` | 绘制 | ts-drawing-components-circle.md |

### 其他 Kit API

索引：`zh-cn/application-dev/reference/apis-{kit}/Readme-CN.md`

具体 API：`zh-cn/application-dev/reference/apis-{kit}/js-apis-{module}.md`

### 开发指南

| 主题 | 路径 |
|------|------|
| UI 开发（布局、动画、手势、弹窗） | `zh-cn/application-dev/ui/arkts-{topic}.md` |
| 状态管理（@State、@Prop、@Link） | `zh-cn/application-dev/ui/state-management/arkts-{topic}.md` |
| 渲染控制（ForEach、LazyForEach） | `zh-cn/application-dev/ui/rendering-control/` |
| ArkTS 语言基础 | `zh-cn/application-dev/quick-start/` |

## Kit 目录名速查

| Kit | 目录名 | 场景 |
|-----|--------|------|
| ArkUI | `apis-arkui` | UI 组件、布局、动画 |
| Ability | `apis-ability-kit` | UIAbility、生命周期、Want |
| Network | `apis-network-kit` | HTTP、WebSocket |
| Location | `apis-location-kit` | 定位 |
| Media | `apis-media-kit` | 音视频 |
| Image | `apis-image-kit` | 图片处理 |
| Camera | `apis-camera-kit` | 相机 |
| Notification | `apis-notification-kit` | 通知 |
| Core File | `apis-core-file-kit` | 文件管理 |
| ArkData | `apis-arkdata` | 数据管理、Preferences |
| ArkWeb | `apis-arkweb` | Web 组件 |
| Basic Services | `apis-basic-services-kit` | 系统基础服务 |
| Connectivity | `apis-connectivity-kit` | 蓝牙、Wi-Fi |
| BackgroundTasks | `apis-backgroundtasks-kit` | 后台任务 |
| Form | `apis-form-kit` | 卡片 |
| Crypto | `apis-crypto-architecture-kit` | 加密 |
| User Auth | `apis-user-authentication-kit` | 用户认证 |

## 注意事项

- 默认使用 `master` 分支（最新版本），如需特定版本可替换分支名
- Readme-CN.md 是每个 Kit 的索引，不确定文件名时先查索引
- 404 通常是文件名猜错了，不是文档不存在
