# 前端 UI 组件清单（原语目录与统一契约）

> 组件是壳，design token 是壳的大脑（`src/styles/theme/`）。本清单是前端 UI
> 组件的唯一目录：新增任何 UI 前先查这里；一个交互意图只允许一种反馈实现，
> 新需求通过扩展原语的 prop 表达，而不是新建近似组件。
> 审计基线：2026-09-06（ui-list-inset 批次），90 个组件全量盘点。

## 原语目录

### 布局壳（每页恰好一次，页面不得自建）
| 原语 | 职责 |
|---|---|
| `layout/MainLayout` → desktop/tablet/mobile 三壳 | 设备分派、侧栏/底栏/顶栏 |
| `layout/page/RoutedContentViewport` | 三端唯一滚动 owner（`data-desktop-main`） |
| `layout/page/PageViewport` | 路由画布（`data-page-viewport`） |
| `layout/page/StandardPageLayout` / `SplitPageLayout` / `ContentColumn` | 页面骨架与内容栏档位 |
| `layout/page/PageHeader` | 唯一页面头部（sticky/搜索/ below 插槽） |

### 列表（行交互的唯一体系）
| 原语 | 职责 |
|---|---|
| `list/ListContainer` | 列表容器，四形态：`bordered`（开放页面卡片）/ embedded（面板内集合）× `inset`（内缩圆角行，Telegram/macOS 语言）/ 默认（分隔线）。可 hover 表面用 `inset` |
| `list/ListItem` | **唯一行交互原语**。hover/选中填充、12px 圆角、键盘、aria、`tone: joined/card`。行级卡片类禁止（eslint 强制） |
| 行包装组件 | `ChatListItem` / `home/ActivityRow` / `profile/UserCardRow` / `forms/SettingRow`——只做内容编排，交互全部委托 ListItem |

### 卡片（CSS 卡语言 + 一个内容卡原语）
| 原语 | 职责 |
|---|---|
| `data-display/NearbyImageCard` | 图片优先内容卡**唯一实现**（附近动态/附近的人/附近群组/首页轮播/群组详情） |
| `card-hover`（utilities） | 可点击卡片唯一 hover 语言：投影抬升 + `--cst-hover-lift`（vibrant）/ 纸面微染+边框（forest） |
| `card-raised` / `surface-hero` / `ListItem tone=card` | 静态卡片语言（白底 + 投影/描边），禁止再发明第四种 |
| `image-hover-frame` | 相册小方格专用（照片墙缩略图），内容卡不得使用 |

### 按钮 / 输入 / 导航 / 弹层 / 反馈
| 原语 | 职责 |
|---|---|
| `ui/button` | 表单与主 CTA 按钮 |
| `navigation/ActionIconButton` | 图标按钮（ghost/outline） |
| `hover-btn` / `scale-btn`（utilities） | 行内文字/图标动作的 hover/tap CSS 语言 |
| `forms/SearchInput` / `forms/Toggle` / `forms/ImagePicker` / `forms/TagPill` | 输入类原语 |
| `navigation/TabBar` | 分类/过滤胶囊（唯一 Tab 原语；底栏走 `layout/mobile/NavTab`） |
| `overlays/SheetShell`、`ui/dialog`、`ui/popover`、`ui/dropdown-menu` | 弹层体系 |
| `data-display/EmptyState` / `InlineLoading` / `LoadError`、`overlays/ToastViewport`、`shared/ConnectionBanner` | 状态反馈 |
| `shared/Avatar` | 头像（尺寸/形状/状态点/环） |

## 统一交互契约

1. **行高亮**：hover / 选中 / 键盘焦点同形状（12px 圆角 `--cst-shape-sm`），
   由 ListItem 内聚；容器 `inset` 时高亮天然离边 8px，禁止行高亮顶容器边。
2. **卡片 hover**：只走 `card-hover`；参数全部 token 化
   （`--cst-hover-lift` / `--cst-hover-image-zoom` / `--cst-card-tap-scale`），
   主题在 `themes/{vibrant,forest}.css` 各自演绎。
3. **按压**：行 = opacity-tap；卡片 = `--cst-card-tap-scale`；按钮 = `hover-btn`。
4. **明暗**：一律 token，禁止组件写死色值/色 mix。

## 近重复收敛 backlog（按危害排序）

1. **手写按钮**：59 个文件用 `hover-btn` 手写 `<button>`，`ui/button` 仅 10 处。
   规则：改到某文件时把该文件的简单动作按钮顺手迁 `ui/button`/`ActionIconButton`。
2. **表单 Input 原语缺失**：13 个文件手写 `<input>`；应建 `forms/Input`
   （前缀图标/清除/错误态），先落登录/注册/搜索之外的高频表单。
3. **分组卡壳两种**：设置页 `card-raised`（投影）vs 关系列表 `ListContainer bordered`
   （描边）。待统一为一种（对齐鸿蒙端「白卡+发丝描边+主题投影」）。
4. **触屏列表未启用 inset**：notify / me 个人数据等保持分隔线形态；
   评估是否全面切换 inset（触屏无 hover，影响仅视觉）。

## 流程要求

任何 UI 改动必须**截图对比审查**（改前/改后、双主题、受影响断面），
不允许只靠代码 diff 判断视觉结果；与市面成熟设计（Telegram/Slack/macOS
System Settings 等）的模式对表后再动手。
