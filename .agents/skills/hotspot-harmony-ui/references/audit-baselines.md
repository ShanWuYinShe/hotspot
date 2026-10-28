# 鸿蒙端审计基线与假阳性规律（hotspot harmonyapp）

本文件是 skill `hotspot-harmony-ui` 的明细：三类审计的正确判据与脚本假阳性规律。改列表/图片/交互/性能相关代码后按此回归。

## 1. 「固定高度」审计的正确判据

项目要求「固定高度容器（搜索框、菜单行、pill）改用最小尺寸约束，系统大字体时要能随内容撑高而不是截断」。据此审计全仓 **81 处** `.height(...)` 且块内含 Text 的位置，**逐一核实后判定为无缺口、不需要改动**。判据如下，下次别再重复怀疑：

| 类别 | 例子 | 判定 |
|---|---|---|
| **图形容器**（只含 Icon/头像/徽章，无文字） | `EMPTY_ICON_SIZE`、`ICON_SIZE_MD`、`AVATAR_SIZE.*`、`UNREAD_BADGE_SIZE`、`LOADING_ICON_SIZE`、`DIALOG_ICON_TILE_SIZE` | ✅ **固定尺寸正确**——图形不随字体缩放，改 minHeight 反而错 |
| **按钮 / 输入框** | `SystemButton`(BUTTON_HEIGHT*)、`SystemInput`(INPUT_HEIGHT)、`SearchPageHeader`(CHAT.INPUT_HEIGHT) | ✅ **固定高度正确**——前端同为固定高（`button.svelte` 用 `h-10`/`h-11`，全仓 `min-h-*` 仅 14 处且都是特殊场景）。控件固定高是跨端一致的既定契约 + 行业惯例，改成 minHeight 会破坏跨端一致 |
| **可撑高容器** | `SystemTextArea`(`constraintSize({minHeight})`)、`EmptyState`(`EMPTY_FILL_MIN_HEIGHT`)、`CreateMomentSheet` 多行输入 | ✅ 已用 minHeight，正确 |
| **热区扩展** | `TabsPillGroup`：视觉高度由内层固定、外层 `constraintSize({minHeight: MIN_HIT_TARGET})` 只扩命中区 | ✅ 正确范式，可作参考 |

**审计脚本的坑**：「块内 N 行含 Text」这种粗判会把**图标容器**误报为「文本容器」（81 处里绝大多数是图标容器）——要按**该 height 所属元素自身**是否承载文字判断，不要用「附近有 Text」。

## 2. 交互反馈审计的正确判据

项目原则要求「交互状态一眼可辨：hover / focus / active / disabled 四态差异长在元素本体上」。按「可点元素链上是否有反馈」做全仓审计，得到 21 处候选，**逐一核实后判定全部为误报**——原因：按压反馈**不一定写在 onClick 所在的那条属性链上**。以下都是合法承载方式：

| 承载方式 | 例子 | 说明 |
|---|---|---|
| **组件内部承载** | `SystemCard({ clickable: true })` 外层再 `.onClick` | 卡片组件内部用 `isPressed` → `SCALE_PRESSED_CARD` + spring 动画 |
| **状态驱动的样式** | `TabsPillGroup` 的 `.onClick` 只 dispatch | 反馈由 `hoveredIndex` / 选中态驱动背景与缩放（13 处相关代码） |
| **属性开关** | `SystemIcon` 的 `clickable` | 反馈由使用方通过该开关启用 |
| **聚焦态代替按压态** | `SystemInput` | 输入框交互靠 focus border 表达，**不需要**按压缩放 |
| **父级行承载** | 设置行的 `.onClick` | 行组件统一提供按压填充（`surfaceSubtle`） |

**结论：本仓交互原语（Pressable / SystemButton / SystemCard / SurfaceIconButton / SystemToggle / TabsPillGroup / SystemInput / 设置行）的反馈机制完备，无需补齐。**

**审计脚本的坑**：用「onClick 所在属性链上是否出现 scale/onTouch/animation/backgroundColor」做启发式，误报率极高（本次 21/21 误报）——因为反馈常在**组件内部**或**由状态间接驱动**。要判断某处是否真缺反馈，必须**打开该组件与其使用上下文**看，不能只扫属性链。

## 3. 性能审计基线（全部无缺口）

ArkUI 性能清单四项实测结果与复核方法（改列表/图片相关代码后按此回归）：

| 项 | 结果 | 复核方法 |
|---|---|---|
| LazyForEach/ForEach 键稳定性 | **85/85 稳定** | 键函数是 ForEach 块**收尾行**的 `(item) => 表达式`；逐一核对均为稳定 ID（user_id/moment_id/session_id/图片 url）。两类含 index 的属合法：静态 tab 配置（`` `${tab.label}-${index}` ``）、骨架占位（`conv-skeleton-${i}`，键=字符串项自身，HomePage 的 `(_, s) => s` 同理） |
| cachedCount | 12 处长列表全部配置（5~8） | grep cachedCount |
| @Reusable | 18 个组件（含 MomentFeedCard/UserCard/EmptyState 等高频卡） | grep @Reusable |
| Image 尺寸约束 | **20/20** 有 objectFit + 宽高/aspectRatio | 动态图全用 `.width('100%') + .aspectRatio() + objectFit(Cover) + alt + onError` 契约 |
| 事件生命周期 | **9/9 组件 on/off 配对完整**（补测） | interactionEventHub 订阅组件 4 个 + ChatPage 9on/9off + ProfilePage 双事件双注销 + NativeMapComponent（清定时器/清聚合层/**release 全部 PixelMap**）；地图监听挂组件自有 controller 随组件销毁。新增订阅时必须在 aboutToDisappear 注销同 handler 引用 |

**审计脚本的坑（第三次验证同一规律）**：Image 约束检查报 3 处「缺尺寸」，实际都有 `.width('100%') + .aspectRatio()` —— 脚本只认数字 `.width(N)/.height(N)`。启发式只产候选清单，判定必须人工开上下文；键函数要按「块收尾行」提取而非开头。

## 4. 通用规律

- 启发式「附近/属性链上是否有某关键词」误报率极高；真实承载常在组件内部、状态驱动或父级行。
- 结论落盘必须附命令与复现；沿用旧结论前先证伪一次（项目 AGENTS.md「环境类结论必须可复现」）。
