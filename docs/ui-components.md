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
| `layout/ambient/AmbientBackdrop` | 页面氛围画布（三主题世界层装饰，`z-index:-1`，由 PageViewport/认证页/错误页/启动占位统一挂载） |
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
| `data-display/NearbyImageCard` | 图片优先内容卡**唯一实现**（附近瞬间/附近的人/附近群组/首页轮播/群组详情） |
| `card-hover`（utilities） | 可点击卡片唯一 hover 语言：投影抬升 + `--cst-hover-lift`（vibrant）/ 纸面微染+边框（forest） |
| `card-raised` / `surface-hero` / `ListItem tone=card` | 静态卡片语言（白底 + 投影/描边），禁止再发明第四种 |
| `image-hover-frame` | 相册小方格专用（照片墙缩略图），内容卡不得使用 |

### 按钮 / 输入 / 导航 / 弹层 / 反馈
| 原语 | 职责 |
|---|---|
| `ui/button` | 全部动作按钮的唯一壳（表单/CTA/弹层确认/详情动作区；`ui/FollowButton` 等复合钮经它派生） |
| `navigation/ActionIconButton` | 图标按钮（ghost/outline） |
| `hover-btn` / `scale-btn`（utilities） | 行内文字/图标动作的 hover/tap CSS 语言 |
| `hit-slop` / `hit-slop-y`（utilities） | 紧凑控件的触控热区扩展（视觉尺寸不变）。`hit-slop` 四周各 6px，用于图标按钮等**孤立**控件；`hit-slop-y`（2026-10-19 新增）只扩纵向，用于**横向相邻**的紧凑控件（筛选胶囊、分段控制器）——前者的水平 6px 会与相邻项重叠，命中落到 DOM 靠后那颗。注意：两者都靠 `::before` 绝对定位，**会被祖先的 `overflow` 裁剪**，横滚容器需留出纵向 padding（TabBar 的 `py-1` 即为此） |
| `forms/SearchInput` / `forms/Input` / `forms/Toggle` / `forms/ImagePicker` / `forms/TagPill` | 输入类原语。`forms/Input`（2026-10-12 新增）：单行输入，受控 value+onChange；`label`/`labelSnippet`（tracking-field 档位或自定义行）与 input 以 id 关联；`prefix` snippet 前缀图标；`clearable` 清除钮；**错误态三件套** `hasError`+`errorText`→ aria-invalid + aria-describedby + role="alert" 就近错误行（样式由 input-surface 的 :has(aria-invalid) 承担）。padding 由调用方 class 提供（同 SearchInput 约定，如 `rounded-md px-4 py-3`）；另支持 `compact` 形态（input-surface 直贴 input，无包裹/label/prefix/清除，紧凑位专用）。契约对齐鸿蒙 `SystemInput` |
| `navigation/TabBar` | 分类/过滤胶囊（唯一 Tab 原语；底栏走 `layout/mobile/NavTab`）。三个变体：`pills`（筛选/分类，可横滚）、`segmented`（二选一/系统级分段，等宽轨道）与 `underline`（**页级两态切换**，2026-10-20 新增：全宽等分、无轨道无图标、激活项 = 主文字色 + 底部 2px accent 滑动下划线；用于首页「关注/发现」，替代原 288px 浮动胶囊）。`disabled`（2026-10-19 新增）整体禁用并置灰，表达「可读但不可改」（如瞬间可见性探测中）；**移动端触控高度**：按钮 `py-3`（视觉 41px）+ 滚动根 `py-1` + `.hit-slop-y` → 有效 ≈49px，桌面 `lg:py-1.5` 保持紧凑 |
| `overlays/SheetShell`、`ui/dialog`、`ui/popover`、`ui/dropdown-menu` | 弹层体系 |
| `overlays/MenuPanel` | 弹出菜单面板唯一实现（标题 + 选项行 + 可选底部动作；容器走 `ui/popover` Content）。已有：地图源菜单、地图筛选菜单。新菜单只传数据，不复制行结构 |
| `data-display/EmptyState` / `InlineLoading` / `LoadError` / `ListEnd`、`overlays/ToastViewport`、`shared/ConnectionBanner` | 状态反馈 |
| `shared/Avatar` | 头像（尺寸/形状/状态点/环） |

> 四种列表状态各司其职，不得互相替代：空 `EmptyState` / 加载中 `InlineLoading` /
> 失败 `LoadError` / 已到底 `ListEnd`（文案「没有更多了」）。分页与无限滚动列表
> 在内容耗尽时必须渲染 `ListEnd`——「加载更多」按钮消失而无收尾提示，用户无法
> 区分「已到底」与「加载卡住」。

### 按钮语言两派契约（2026-10-18 登记，审计 C1 收敛）

动作按钮只有两种被允许的形态，**按场景分派、同页同场景不得混用**；两种形态都必须
经由 `ui/button` 原语（含其派生 `ui/FollowButton`），禁止绕过原语手写描边/实底钮：

| 派别 | 形态 | 适用场景 | 例 |
|---|---|---|---|
| A 页面级动作（默认） | `rounded-md`(16px) + `text-sm`，高度档 sm 32 / md 40 / lg 44 | 表单提交、弹层确认、详情页动作区、移动端动作条、错误页动作 | 登录/保存/举报提交/屏蔽/加入群组/私信动作条 |
| B 行内关系 chip | `rounded-full` + `text-xs`，`px-3.5 py-1.5`（行内）/ `px-4 py-2`（详情宽位） | 卡片/列表行内的关系与轻量动作（嵌在内容里的小动作） | 关注/加入/接受/拒绝/私聊入口 chip |

- 圆角禁档：动作按钮不得使用 `rounded-sm`(12)/`rounded-lg`(20)/`rounded-xl`(24)；
  圆形图标钮走 `ActionIconButton`/`size-* rounded-full`。导航项、菜单行、Toast、
  图片格等非动作控件按各自组件语言，不套本契约。
- 卡片/面板**内嵌**的次级动作钮用 `flat` 去投影（投影语义归容器，见 `ui/button` 注释）。
- 链接跳转用 `<a>` 时沿用派 A 的原语类语言（如错误页「返回首页」），不自造第三种形态。

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

## 同类元素基准与密度原则（2026-10-20 实测，收敛中）

用户审查目标：**同类元素必须同风格**，同时"在提高信息密度的前提下保持界面简洁，让用户自然找到功能与内容"。

### 1. 密度原则（定档依据）

- **紧凑优先，但不断行**：能一行说完的信息不用两行；行高由内容决定，档位只有 3 档（见下），禁止近邻并存（如 62/64/68 三档并存 → 合并）。
- **留白只为分组，不为装饰**：区块内用 8/12 节奏，区块间 24；"看起来空"应通过增加可见信息量解决，不要靠加 padding。
- **一眼可辨的三层信息**：标题（15px semibold）→ 正文（13~14）→ 元信息（11~12 次级色）。超过三层即视为信息过载，降级而非缩小。
- **交互元素只在不打断扫读的前提下出现**：行尾 `⋯` / 图标按钮常显时必须是"操作"；辅助操作走 hover/focus 显现（`desktop-reveal`）。

### 2. 实测基准（390×844 / dark / vibrant，CDP 实测）

| 家族 | 基准 | 已登记例外 |
|---|---|---|
| 卡片容器 | `radius 16`（`rounded-md`）+ `border 1px` + `bg surface-primary` | 资料/英雄卡 `radius 20`（`rounded-lg`，`/me` 顶部）|
| 胶囊 chip | 紧凑 `20`（badge）/ 标准 `24`（标签）/ 筛选 `24~28`（TagPill sm/md）/ **行内动作 `28`（`h-7`）** | — |
| 列表行 | **`58`（单行无头像）/ `62`（单行带头像）/ `68`（双行）/ `74`（带控件设置行）** | — |
| 图标按钮 | `36`（header 动作）/ `44`（触控下限） | 卡片内操作 `28~32` |
| 按钮（Button 原语）| `sm h-8` / `md h-10` | — |
| 区块标题 | 15px / semibold（`SectionHead`） | — |

**2026-10-20 首轮收敛结果**（逐项核实：CDP 取 `getBoundingClientRect` + `getComputedStyle` 拿到类名再改，不按估计改元素）：
- ✅ `/notify` 行内「接受/拒绝」原为内容撑高（29 vs 31，兄弟按钮差 2px）→ 固定 `h-7` → 实测两者同为 `28`；
- ✅ `/me` 统计网格 `MetricRow` 由 `rounded-sm`(12) 改为 `rounded-md`(16) → 实测 `rad=16`；
- ✅ 销案两项：`/nearby/groups` 的 32px 胶囊是 `Button` `sm` 档（允许）；`/search` 的 58/68 是单行/双行两档（已补进基准表）。

### 3. 收敛顺序（按收益排序）

1. ✅ **胶囊高度**：内容撑高的胶囊改为固定档（已修 `/notify` 接受/拒绝，实测 29/31 → 同 28）；
2. ✅ **卡片底色**：全站扫描（32 路由 × 移动端，判别条件「radius ≥ 12 + 有边框 + 透明底 + 无图片后代」）
   命中 **0** —— 卡片底色本已统一；透明底只在"叠在图片上"的封面区出现（允许）；
3. ✅ **列表行**：实测值全部落在 58/62/68/74 四档内（58 单行无头像、62 单行带头像、68 双行、74 带控件设置行）；
4. ✅ **区块标题**：管理后台 6 处 `text-sm`(13px) 对齐到 `text-md`(15px)；管理后台之外无 `text-sm` 的 h2。

**结论（2026-10-20）**：本轮清单四项全部处理完毕，基准表转为**硬约束**——新增 UI 必须先选档位，
不新建中间值；如需例外，在本文件登记并写明理由。

> 收敛完成后，本清单转为**硬约束**：新 UI 必须先选档位，不新建中间值。
