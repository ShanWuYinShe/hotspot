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
| `list/ListItem` | **唯一行交互原语**。hover/选中填充、12px 圆角、键盘、aria、`tone: joined/card`。行级卡片类禁止（eslint 强制）。列表容器内自动包 `role=listitem`（ListContainer 经 context 声明；ListEnd/InlineLoading 同机制），裸用不变。行点击由内部透明覆盖按钮（`[data-cover]`）承载，行内操作按钮自动抬升（z 序），根不持 button 语义（axe nested-interactive 清零） |
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
| `forms/SearchInput` / `forms/Input` / `forms/Toggle` / `forms/ImagePicker` / `forms/TagPill` | 输入类原语。`forms/Input`（2026-10-12 新增）：单行输入，受控 value+onChange；`label`/`labelSnippet`（tracking-field 档位或自定义行）与 input 以 id 关联；`prefix` snippet 前缀图标；`clearable` 清除钮；**错误态三件套** `hasError`+`errorText`→ aria-invalid + aria-describedby + role="alert" 就近错误行（样式由 input-surface 的 :has(aria-invalid) 承担）。padding 由调用方 class 提供（同 SearchInput 约定，如 `rounded-md px-4 py-3`）；另支持 `compact` 形态（input-surface 直贴 input，无包裹/label/prefix/清除，紧凑位专用）。契约对齐鸿蒙 `SystemInput` |
| `navigation/TabBar` | 分类/过滤胶囊（唯一 Tab 原语；底栏走 `layout/mobile/NavTab`） |
| `overlays/SheetShell`、`ui/dialog`、`ui/popover`、`ui/dropdown-menu` | 弹层体系 |
| `overlays/MenuPanel` | 弹出菜单面板唯一实现（标题 + 选项行 + 可选底部动作；容器走 `ui/popover` Content）。已有：地图源菜单、地图筛选菜单。新菜单只传数据，不复制行结构 |
| `data-display/EmptyState` / `InlineLoading` / `LoadError` / `ListEnd`、`overlays/ToastViewport`、`shared/ConnectionBanner` | 状态反馈 |
| `shared/Avatar` | 头像（尺寸/形状/状态点/环） |

> 四种列表状态各司其职，不得互相替代：空 `EmptyState` / 加载中 `InlineLoading` /
> 失败 `LoadError` / 已到底 `ListEnd`（文案「没有更多了」）。分页与无限滚动列表
> 在内容耗尽时必须渲染 `ListEnd`——「加载更多」按钮消失而无收尾提示，用户无法
> 区分「已到底」与「加载卡住」。

### 内容卡封面比例档（2026-10-18 登记）

同一内容实体在不同视图允许的封面形态只有三档，**同一视图只取一档**：

| 档位 | 形态 | 使用场景 |
|---|---|---|
| 图片优先竖卡 | `aspect-[3/4]` 图铺满 + 底部信息托盘（`NearbyImageCard` 默认） | 附近的人网格、首页群组轮播 |
| 内容卡 | `aspect-[4/3]` 图 + footer（`NearbyImageCard` + footer snippet） | 我的瞬间、附近群组普通卡、我的群组平板/桌面 |
| 横向紧凑卡 | 固定高 `h-32`（128px），`horizontal` 图左 40% + 信息右 | 我的群组手机、列表型群组卡 |

- 发现流 BSP 瓦片墙的卡片比例是布局算法产物（`minRatio 0.68 / maxRatio 1.48`，
  已登记的有意设计），不属于上述三档，不得据此另造卡片形态。
- 手机端需要「一屏多张」时选横向紧凑卡，不得把 4:3 卡直接缩窄（信息会被压碎）。
- 首页双列人物卡的卡壳由 `ListItem tone="card"` 唯一承担，**禁止外层再包手写卡**
  （双描边/双投影）；简单关系列表（好友/关注/粉丝等）一律走
  `ListContainer bordered` + 连接行，不再逐行拆独立卡片。

## 统一交互契约

1. **行高亮**：hover / 选中 / 键盘焦点同形状（12px 圆角 `--cst-shape-sm`），
   由 ListItem 内聚；容器 `inset` 时高亮天然离边 8px，禁止行高亮顶容器边。
2. **卡片 hover**：只走 `card-hover`；参数全部 token 化
   （`--cst-hover-lift` / `--cst-hover-image-zoom` / `--cst-card-tap-scale`），
   主题在 `themes/{vibrant,forest,inkwash}.css` 各自演绎。
3. **按压**：行 = opacity-tap；卡片 = `--cst-card-tap-scale`；按钮 = `hover-btn`。
4. **明暗**：一律 token，禁止组件写死色值/色 mix。

## 近重复收敛 backlog（按危害排序）

1. **手写按钮**：58 个文件用 `hover-btn` 手写 `<button>`，`ui/button` 引用 23 个文件
   （2026-10-08 刷新：设置页/LoadMoreButton 已迁；**2026-10-12 复核：60 / 24**，
   统计口径 `grep -rl` 全 src *.svelte）。
   规则：改到某文件时把该文件的简单动作按钮顺手迁 `ui/button`/`ActionIconButton`。
   约束（2026-10-07 LoadMoreButton 迁移实测）：`ui/button` secondary 自带的
   `shadow-raised-sm` 是自定义主题 token，**不参与 tailwind-merge 冲突消解**——
   className 里写 `shadow-none` 也消不掉它，且生成 stylesheet 里它排在后面
   （浏览器实测 computed box-shadow 落到 raised-sm）。无投影的胶囊按钮
   （如 LoadMoreButton）在原语支持「去投影」开关之前**不得迁移**，否则静默
   加一层投影。`LoadMoreButton.test.ts` 的「视觉形态守恒」用例已锁定当前形态
   （token 边界匹配），任何加投影/改圆角/改字色的迁移会在此失败。
 2. **表单 Input 原语**：**已落地（2026-10-12）**——`forms/Input` 建成（10 个契约测试），
    首个消费方 `me/edit` 昵称输入已迁移（登录态截图对比审查通过）。
    确认码输入亦已迁（compact 形态，2026-10-12，运行时截图验证通过）——me/edit 全页输入原语化。
    **其余消费方评估后保留手写（2026-10-12 全量盘点）**：剩余 20 处 input 全部是
    **compact 变体**（`input-surface` 直贴 input，admin 六页/群组弹层 ×3/安全设置 ×6/
    map 面板 ×2）+ 应保持原生的 file/checkbox。compact 迁移收益≈0：样式半边已合规、
    表单无错误态需求，迁移只改写 markup 不增行为，反有回归风险。策略改为：
    ① wrapper 型新表单一律用 `forms/Input`；② compact 消费方改到该文件且有新增行为需求
    （如清除钮/错误播报）时，先给原语加 compact/size 变体再迁；
    ③ AccountSecuritySection 密码簇若要补 AuthForm 式显隐切换，属 UX 增强需设计确认。
3. **分组卡壳已收敛**（2026-09-10 / 2026-10-18 收口）：移动端 `ProfileMobile` 的
   `card-raised` 已改为与桌面 `ProfilePage` 一致的「白卡 + 发丝描边 + 主题投影」。
   **2026-10-18**：设置页「账号信息/外观与显示/地图与位置/系统与关于」四组手写白卡
   统一为 `ListContainer bordered inset` + `ListItem`（与通知/隐私/账号安全分组同语言）；
   手写白卡只保留给真正的独立内容卡（个人主页身份卡、管理概览身份卡等）。新增卡片
   默认先问：这是「列表集合」还是「独立内容卡」——列表集合必须走 ListContainer。
4. **触屏列表未启用 inset**：notify / me 个人数据等保持分隔线形态；
   评估是否全面切换 inset（触屏无 hover，影响仅视觉）。

## 已完成的收敛（2026-09-10 全量审查批次）

- **关系分区路由归一**：`/me/friends`、`/me/following`、`/me/fans` 三条与
  `/me/relationships?tab=` 完全重复且无入站入口的孤儿路由，改为 308 重定向
  （仅保留 `+page.ts`，UI 全部由 `me/relationships/RelationshipsPage.svelte` 承载）；
  tab id 与深链构造的单一事实源见 `lib/relationships.ts`。
- **个人主页入口去重**：`ProfilePage` / `ProfileMobile` 同一页曾各有 2 个设置入口
  （页眉图标 + 身份卡按钮），现统一只保留身份卡的拇指操作区那一个。
- **列表语言收口（2026-10-18）**：关系页三 tab 的独立卡网格改为连接列表行；
  管理后台概览手机端模块入口改为行式列表；我的举报行补内边距/行分隔并把 `ListEnd`
  移出列表卡；通知入群邀请行动作组窄屏换行；首页双列人物卡去掉外层重复卡壳。
- **标题层级统一**：全站 `PageHeader` 的 `titleSize` 统一为
  `device === 'desktop' ? 'xl' : '2xl'`（原首页/附近/消息页用 `3xl`，动态详情页
  固定 `xl`，四套并存）。
- **无 hover 反馈的按钮清零**：21 个自带底色/描边却因命中全局 hover 排除集而
  完全没有 hover 的按钮，按「实底升一档投影 / 描边白底走 `hover:bg-accent-soft` /
  遮罩按钮走 `hover:opacity-hover`」三档补齐。
- **死类名清零 + 护栏**：11 个从未注册的类名（`shadow-raised-xs` 18 处、
  `bg-surface-subtle`、`bg-surface-secondary`、`bg-accent-hover`、`text-text-tertiary`、
  `duration-fast`、`ease-bounce`、`scrollbar-none`、`rounded-2xl`、`desktop-reveal` 等）
  已全部修好，并新增 `scripts/check-utility-classes.mjs` 挂入 `bun run check:theme`
  防复发。

## 流程要求

任何 UI 改动必须**截图对比审查**（改前/改后、双主题、受影响断面），
不允许只靠代码 diff 判断视觉结果；与市面成熟设计（Telegram/Slack/macOS
System Settings 等）的模式对表后再动手。
