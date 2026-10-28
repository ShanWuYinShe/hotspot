---
name: hotspot-frontend-style
description: hotspot 前端 web 样式与交互状态契约：样式自查清单（圆角/字号/字距/行高/阴影/模糊/opacity/焦点/z-index 等强制档位）、交互状态与按钮 hover 两档、按钮语言两派、卡片交互契约、媒体图片管线与占位、骨架/空态、质量门与性能红线。触发：新增或修改任何前端样式；写 Tailwind 类名；使用 design token；处理 hover/focus/active/disabled 状态；列表行/弹层/输入框样式；图片与媒体占位；样式自查与 UI 质量门（axe/触控/主题矩阵）。
---

# hotspot-frontend-style

> 数值档位与双端对照的**唯一权威**是根 `docs/design-system.md`（圆角§6 / 字号§4 / 间距§5 / 模糊§9 / 动效§10 / 阴影§8 / 图片卡§17 / 软岛§19 / 氛围画布§16 / 运行时§21.12）；本文件只保留前端执行规则与速记档位，改 token 数值时以它为准并两端同步。页面结构见 skill `hotspot-frontend-layout`，token 定义见 `hotspot-frontend-theme`，鸿蒙端见 `hotspot-harmony-ui`。

## 样式自查清单（新增/修改样式后逐条过）

> **核心原则：少档位 + 强制收敛**。设计系统的价值在约束而非自由——档位要少、近邻值必须归并（1-2px 差异人眼分不清，并存只会让界面碎、无节奏）。「不够用就归到最近档位」是默认动作，新增档位需充分论证；同一类东西禁止变体过多。

1. **圆角（4 档 + full）**：控件/卡片一律用 `rounded-*` 类（`--radius-*`：**sm 12 / md 16 / lg 20 / xl 24px** + `rounded-full` + `rounded-sheet` 28px 弹层）或 `var(--cst-shape-sm/md/lg/xl)`；禁止任意值 `rounded-[13px]`、禁止内联 `border-radius:20px`、禁止近邻档位并存（原 compact/tight/small/control/relaxed/panel/feature 7 档已收敛到 4 档）。例外仅限物理形状细件（开关半高圆角、1-3px 状态条、拖拽把手）。
2. **颜色**：组件内禁止 hex/rgba 字面量（数据层同样禁止）；arbitrary 类（`bg-[#...]`/`shadow-[...]`）禁止含色值；阴影色值统一走 `--shadow-*` token。
3. **字号（强制，9 档）**：一律用 `text-*` 刻度类——**4 档自定义 token（`--text-*`：xxs=10 / xs=11 / sm=12 / md=14）+ Tailwind 默认 5 档**（text-lg 及以上 18/20/24/30/36），与 `check-cross-platform-scale.mjs` 的 `FONT_MAP`、`frontend/AGENTS.md`「字号 9 档」同口径。xxs=10 极小 uppercase 标签（eyebrow）；xs=11 辅助文字（副标题/计数/时间戳）；sm=12 正文（原 13px 已归并）；md=14 强调正文（昵称/按钮主标/侧栏导航）。**禁止 1-2px 近邻档并存**；禁止 `text-[Npx]` arbitrary、禁止内联 `font-size`（含动态 ternary——移到 class 三元）。已登记例外：`Avatar` fallback 首字母 `font-size:{size*0.42}px`（随头像等比缩放）。
   `PageHeader` 的 `titleSize`/`subtitleSize` 已重构为字号档位 prop（`md/base/lg/xl/2xl/3xl` → `TITLE_SIZE_CLASS`），全站统一传 `device === 'desktop' ? 'xl' : '2xl'`（桌面 20 / 平板·移动 24），不得逐页自选——标题层级是页面锚点。
4. **accent/status 实色背景上的内容**：文字/图标必须用对应 `on-*` token（`text-text-on-accent`/`text-text-on-status`/`text-content-on-marker`），禁止恒白 `content-inverse`；选中行内与 accent 同色的 marker 图标需实色底衬托（`PinMedia` 的 `selected`）。
5. **列表行**：交互行一律走 `ListItem`（高亮圆角/背景/反色由组件统一），不在页面自写 hover 背景或叠行级卡片类。
6. **token 注册与类名有效性（强制）**：新 token 需类名消费时同步补 `theme-registry.css` 注册，并核对类名不带 `cst-` 前缀（`text-text-on-status` 而非 `text-cst-...`）。Tailwind v4 对**未注册 utility 静默失效**（不报错、不生成 CSS、svelte-check 无感）：写 token 语义类名前确认其在注册层/定义层/项目自定义 `.class` 中真实存在；项目自定义工具类拼写与 `utilities.css` 一致（如 `no-scrollbar`，曾误写 `scrollbar-none` 静默失效）；`bun run check:theme` 已串联 `scripts/check-utility-classes.mjs` 自动拦截，本地 `bun run check` 与 CI 都会失败——不要绕过该检查。
7. **hover/tap 类**：交互反馈用既有工具类（`hover-btn`/`hover-bg`/`scale-btn`/`tap-*`/`header-action`/`surface-ghost`/`card-hover`），不新发明交互类；禁止双重边框（高亮与元素自带边框不得叠成两圈），契约见下「交互状态契约」。
8. **字距（强制）**：一律 `tracking-*` 刻度类，禁止 `tracking-[Nem]` 与内联 `letter-spacing`：`tracking-eyebrow`=0.14em（uppercase 分组标签、在线/未读/最近小标、新消息/较早分组标）；`tracking-label`=0.12em（次级 uppercase 小标，DesktopLayout 侧栏分组）；`tracking-field`=0.08em（表单字段标签）；`tracking-title`=-0.01em（大标题微收紧）；边界场景用内置 `tracking-tight`(-0.025em)/`tracking-wide`(0.025em)。动态字距用 class 三元，禁止内联。
9. **行高**：禁止 `leading-[N]` arbitrary 与内联 `line-height`。优先语义档位 `leading-card`(1.7 卡片正文/摘要/副标题——由阅读疲劳修复上调，theme-registry.css `--leading-card`；鸿蒙 `FONT.LINE_RATIO.CARD=1.7` 同步) / `leading-copy`(1.75 正文阅读) / `leading-none`(单行紧凑如数字、徽标)；多行正文段落允许 Tailwind 默认 `leading-5/6/7`（固定 px 行高，对可读性敏感、随字号变化需要不同行距）。
10. **阴影（强制，3+2+3 档）**：一律 `shadow-*` token 类或 `var(--shadow-*)`/`var(--cst-elevation-*)`，禁止 arbitrary 投影串、禁止内联手抄、禁止 utilities.css 自行组合双向投影。两套投影语言按语义选择：双向凸起/凹陷（控件/卡片/头像立体感）`shadow-raised-sm/md/lg`（sm=4/4/9 控件、md=5/5/12 卡片、lg=8/8/18 大卡片）+ `shadow-inset-sm/md`，消费 `--cst-elevation-shadow-{dark,light}` 原料（vibrant 显影、forest 透明→改由 border 表达层级）；accent 光晕 `shadow-glow-sm/md/lg`（sm=0/3/10、md=0/4/14、lg=0/6/18）；Material 单向/弹层投影 `var(--cst-elevation-card)`/`-button`/`-modal`/`-sheet`/`-menu`（themes/*.css 各自定义真值），弹层/下拉用投影表达层级、不依赖描边环。地图标记用结构 token `var(--cst-marker-shadow)`（0 2px 10px）/`var(--cst-marker-halo)`（0 0 8px 小光点），色值由调用方按 marker tone 提供（`var(--cst-marker-*-glow)`），定义于 shared.css，禁止组件内联手抄。命名按投影方向/语义（raised/inset/glow/elevation-card），禁止设计风格前缀（neu-/skeuo-/glass-）。已登记例外：`me/edit` 保存按钮 `box-shadow:0 4px 14px {saved?'online-glow':'accent-glow'}`（固定尺寸+动态双色）。
11. **font-weight（强制）**：用 `font-*` 类（`font-medium`=500/`font-semibold`=600/`font-bold`=700/`font-normal`=400），禁止内联 `font-weight`；动态字重（选中态切换）用 class 三元。
12. **页面纵向间距**：`contentClassName` 白名单与页面级/区块级间距三档（12/16/24）的落地口径见 skill `hotspot-frontend-layout`「纵向节奏 → 前端落地口径」与「内容轴」，此处不重复。**窄列特例**（不进 `--cst-page-gutter`，用 `px-4` 压缩 gutter，已登记例外，其他窄列场景需在 AGENTS.md 补登记）：桌面消息分栏左栏（`w-80`=320px）、桌面地图左侧面板（`w-85`=340px，`LocationDesktopPanel` 的 PageHeader 传 `contentClassName="px-4"`，搜索/Tab/列表/确认区统一 `px-4`）。**布局尺寸 token**（layout.css 定义，禁止组件内裸数字）：`--cst-desktop-sidebar-width`（224px 展开）/`--cst-desktop-sidebar-collapsed`（72px 折叠）/`--cst-tablet-dock-max`（680px 平板 Dock）/`--cst-cover-height`（clamp 220→340px）/`--cst-cover-height-lg`（clamp 220→380px）/`--cst-cover-height-sm`（clamp 120→160px 编辑资料小封面）。
13. **尺寸与间距**：控件/图标尺寸优先用 `size-*` 刻度类（`size-9`=36px 等），减少内联宽高。**间距**（gap/px/py/mt/mb）用 Tailwind 默认 4px 步进刻度（`gap-1/1.5/2/2.5/3...`），**不强制收敛、不建 spacing token**——间距的 2px 差异是布局设计的一部分（卡片内 8、section 间 16 等刻意微调），与字号/圆角的「1px 无感噪声」性质不同，强行归并会破坏视觉层次。横向页面 gutter 是唯一例外（`--cst-page-gutter`，由 ContentColumn 统一管理）。存量内联尺寸（地图 PIN、StatusBar 信号/电池等物理装饰细件）可保留；布局尺寸例外：对话框 `78vh/max-height:700px`（桌面居中模态统一档）、移动 bottom-sheet `85%`（新会话）/`92%`（建群）、聊天空态 `min-height:240px/max-width:220px`、上传占位 `height:110px`、封禁页 `max-width:360px`、横幅 `h-7` 类化后仅剩物理例外——新增此类优先 Tailwind 类，确需内联则登记。
14. **毛玻璃模糊半径（强制，3 档）**：`backdrop-filter:blur()` 半径一律 `var(--cst-blur-sm/md/lg)`（sm=8 轻徽章、md=16 sticky header/overlay、lg=24 底部导航/tablet dock），禁止内联 `blur(Npx)` 字面量（原 8/10/16/18/24/28px 六种已收敛到 3 档）。
15. **opacity**：优先 Tailwind `opacity-*` 标准档类（10/20/30.../100），减少内联 `style="opacity:0.N"`；装饰性图标/文字透明度归标准档（0.5→opacity-50），导航激活态背景等精确视觉调校（0.12~0.15 区间）可保留内联。
16. **焦点可见性（强制）**：所有可交互元素必须有可见键盘焦点环，统一 `focus-ring` 类（`focus-visible` 时 accent outline）或等效 `focus-visible:ring-2 focus-visible:ring-accent-default`；**禁止无替代地使用 `outline-none`**（Tailwind v4 会压掉 base.css 的全局按钮轮廓）。图标按钮必须同时提供 `aria-label`；紧凑图标按钮（<44px）叠加 `hit-slop` 扩展热区（注意下方「hit-slop 陷阱」）。
17. **减弱动效（强制）**：新增加载/装饰动画必须考虑 `prefers-reduced-motion`；utilities.css 已提供全局降级（动画瞬时化、`map-pulse-halo` 停用、spinner 放慢），不得绕过；JS 动画（Svelte transition/class 定时器）必须显式判断（`motionDuration(ms)`，reduced 返回 0），否则 reduced 下仍会播。
18. **动效曲线（强制）**：过渡缓动一律 `var(--cst-ease-*)`：`--cst-ease-standard`=(0.22,1,0.36,1) 标准减速曲线（绝大多数 transition）；`--cst-ease-bounce`=(0.34,1.56,0.64,1) 弹簧 overshoot（FAB、toggle 滑块、地图标记放大、sheet 拖拽回弹等「有弹性」的入场/位移）。Tailwind 类消费经 `--ease-standard`（theme-registry 注册）。**duration 不强制 token 化**：0.10s~0.30s 精细差异是 Figma 动效契约的有意设计（btn 0.17s、action 0.14s、tap 0.10s），`--cst-motion-standard:220ms` 仅作基准参考，不代表统一时长。
19. **z-index 层叠**：分两类不混用。全站页面级层叠（弹层/抽屉/Toast 等跨组件浮层）用 `--cst-z-*` token（dropdown=10/overlay=40/modal=50/toast=60），经 `z-[var(--cst-z-modal)]` 等引用；局部 stacking context 内层叠（卡片角标盖图片、地图容器内部排序、sticky header）用 Tailwind 默认 `z-10/20/30` 或 arbitrary `z-[1]/z-[2]`，表达「盖几层」的相对语义，不套全站 token——它们在各自 `position:relative` 容器内独立成层。地图组件内部 `z-index:0~50` 内联属此类（地图外层 `overflow:hidden` 形成独立 stacking context）。
20. **背景装饰层（氛围画布）**：页面不允许自造背景纹理/插画；全站唯一实现 `components/layout/ambient/AmbientBackdrop.svelte`（PageViewport 等处已挂载，新页面自动继承）。组件内确需装饰层必须同时满足：`aria-hidden="true"` + `pointer-events:none` + 父容器 `relative isolate` + 自身 `z-index:-1`、颜色只用 `--cst-ambient-*`/既有 token、动效走 `transform/opacity` 且 reduced-motion 停用。完整规范见 `docs/design-system.md` 第 16 节与 skill `hotspot-frontend-theme`。

## 交互状态契约（现状即规范，改动前先对齐）

| 元素 | hover | focus / active |
|------|-------|----------------|
| 输入框（input-surface） | — | 边框加深（border-strong）+ 内缘 1px 加深线 + 内凹阴影（`--shadow-inset-sm`），底色保持中性 |
| 图片卡（image-hover-frame） | 内嵌 accent 2px 描边 + 图片缩放 1.045（仅 fine pointer） | — |
| 带边框卡片（card-hover） | 边框变 accent（`hover:border-accent-default`）或投影抬升，不画外环 | — |
| 可交互 surface-flat | 描边加深（border-strong） | — |
| 弹层 / 下拉菜单 | 无描边环，层级靠 elevation-menu 投影 | — |

### 按钮 hover 底色：两档，按「控件是否有可见轮廓/底色」选

utilities.css 的全局规则会给**无 `bg-*` 且无 `hover:*` 类**的 `<button>` 自动填 `--cst-interactive-hover-fill`（仅 fine pointer）。因此有底色/描边的按钮被该规则排除，必须**显式**给 hover；两档按控件是否「有可见轮廓」选择，不要混用：

| 控件形态 | hover 底色 | 例 |
|----------|-----------|-----|
| 透明底图标按钮 / 行式按钮 / 未选中 pill | `hover:bg-interactive-hover`（中性灰） | 图标按钮、ListItem、`CreateSheet` 心情/话题按钮 |
| 有可见轮廓或底色（border / ring / `bg-surface-primary`） | `hover:bg-accent-soft`（accent 淡底，更强信号） | `ProfilePage` 描边白底按钮、`AppearanceMenu` ring-1 控件、下拉菜单项 |

- **禁止透明度越档**：`hover:bg-accent-soft/50` 之类 `/N` 变体一律用整档 `hover:bg-interactive-hover`（accent-soft 本身 10~14% 淡底，再砍半档等于用 opacity 造新档位，违反第 15 条）。
- 已选中态（`aria-pressed=true` / `bg-accent-soft` 打底）的 hover 走 `hover:bg-interactive-active`，不用 interactive-hover。
- 导航 Tab 不参与本表：`.tab-item` 规则统一提供 `--cst-interactive-hover-fill`（无 layer，优先级高于 `@layer utilities`），**不要在 TabBar 上再写 `hover:bg-*`**（写了也不生效，是死类名）。

硬性规则：
- 输入框聚焦态由整框承载：内层 input/textarea 不画自身焦点框（utilities.css `.input-surface:focus-visible` 统一压制，与第 16 条配合）；底色保持中性，不引入 accent 色，避免与错误态等 accent 元素抢视觉。
- 禁止双重边框：card-hover 只抬投影不画外环；带边框的卡片 hover 用 `hover:border-accent-default`；surface-flat 的 hover 描边仅对可交互元素生效。
- 图片缩放必须由外层 `overflow:hidden` + 圆角裁剪，图片任何像素不得溢出卡片。
- **禁止渲染空 src 图片**：`<img src="">` 会被浏览器解析为当前页面 URL 而显示破图。无图/空 URL 一律走占位：大图区用 `MediaPlaceholder`（data-display，ImageOff 图标 + accent-soft 底），头像用 `Avatar`（首字母兜底），列表小缩略图内联 `{#if url}` 分支。
- **计数变化反馈**：点赞/评论等瞬时数值变化用 `count-pop` 类 + `{#key 值}` 重建元素触发弹跳（utilities.css cst-count-pop，snappy + bounce，自动遵循 reduced-motion）。
- **大图查看统一入口**：任何「点击查看大图」交互一律复用 `components/overlays/ImageViewer`（bits-ui Dialog 底座：Esc/外点关闭、左右键/滑动切图、单击图片切换放大（以点击位置为锚）与滚轮连续缩放、放大后拖拽平移、缩略图条），禁止各页自建 lightbox。入口仅限动态详情页（单图封面/多图网格）；列表卡片与画廊一律整卡跳详情，不在卡片上直接开大图。
- **地图筛选菜单统一**：`map/shared/MapFilterMenu.svelte` 是桌面/移动唯一筛选实现（Popover 外点/Escape 关闭、徽章、至少一项约束、「全部显示」重置）；地图空白点击一律经 `MapCanvas onMapTap`（内部区分拖拽，拖动不误触），禁止父级用原生 onclick 判定。
- **键盘导航**：列表/搜索页可加 ↑↓ 焦点移动（落到真实链接/按钮、focus-ring 可见），Enter 原生打开、Esc 清空/返回；不引入 combobox/aria-activedescendant；Tab 顺序不变。新增交互后必须跑 axe（0 serious/critical）。
- **按钮光标**：`base.css` 全局 `button:not(:disabled)` / `[role="button"]:not([aria-disabled])` 给 `cursor:pointer`（Chrome 默认箭头会让 hover 反馈「不像可点」）。
- **行类组件 hover 态对比度自检（强制）**：行 hover 底色（半透明 palette-hover）叠加后，行内 `.text-text-secondary` 实测仅 **3.2–4.7:1**（不达标；非 hover 基线 5.31–7.62 达标）。ListItem 已内置修复：hover（仅 fine pointer）/ `focus-within` 时行内次级文字升 `--cst-text-primary`（pressed/on-accent 行排除）。**新增任何「行类」组件必须自检 hover 态对比度**（不能只测静态态）；修复优先在组件内做状态化文字升档，不改 palette（跨端联动）、不删 hover 底色（减特效）。真实鼠标 hover 实测 6 组合 10.61–13.86 全 ≥4.5。
- **tint 淡底 + 同色语义文字**必须用 on-soft 文字档：`text-text-accent-on-soft` / `text-text-status-{online,warning,danger}-on-soft`（= 语义色 80% + text-primary 20%，semantic.css 派生；定义见 skill `hotspot-frontend-theme`）；仅限 tint 软底（徽章/胶囊/软底容器/危险菜单 hover），实底/画布正文继续用 `text-status-*` / `text-accent-default`。悬停才出现 tint 的场景写成 `text-status-* hover:text-text-status-*-on-soft` + `hover:bg-*`。

## 按钮语言两派与组件档位

动作按钮只有两种允许形态，**按场景分派、同页同场景不得混用**，全部经由 `ui/button` 原语（含派生 `ui/FollowButton`），禁止绕过原语手写描边/实底动作钮：

- **派 A 页面级动作（默认）**：`rounded-md`(16) + `text-sm`，高度档 sm 32 / md 40 / lg 44（表单提交、弹层确认、详情动作区、移动动作条、错误页动作）。
- **派 B 行内关系 chip**：`rounded-full` + `text-xs`，`px-3.5 py-1.5`（行内）/`px-4 py-2`（详情宽位）（关注/加入/接受/私聊入口等嵌在内容里的小动作）。
- 圆角禁档：动作按钮禁 `rounded-sm`(12)/`rounded-lg`(20)/`rounded-xl`(24)；圆形图标钮走 `ActionIconButton`/`size-* rounded-full`；导航项/菜单行/Toast/图片格等非动作控件按各自组件语言，不套本契约。
- 卡片/面板**内嵌**次级动作钮用 `flat` 去投影（投影语义归容器）。
- 完整表格与示例见 `docs/ui-components.md`「按钮语言两派契约」。

**组件级档位约定**（走既有 token 刻度，不新增档位；两端口径经源码逐项核对）：

| 约定 | 值 | 两端口径 |
|---|---|---|
| 页面级返回键 | 42px + `hit-slop`（`ActionIconButton` 承载；标准页隐藏、详情页显示） | 两端一致：鸿蒙 `DesignConstants.LAYOUT.BACK_BUTTON_SIZE = 42` |
| 分区标题 | 14px semibold（`text-md` 档）；`eyebrow` 仅限微型分组标（11px uppercase + tracking） | 14px = 既有 `text-md` ↔ 鸿蒙 `FONT.SIZE.MD=14` |
| bare 空态源图标 | 24px；非 bare 圆形容器默认 64（`NeedsAuth` 64 容器为登记例外） | 两端一致：鸿蒙 `EmptyState` 用 `ICON_SIZE_MD=24` / `EMPTY_ICON_SIZE=64` |
| 横屏壳一级导航 | 固定 5 slot：首页 / 附近 / 发布(FAB) / 消息 / 我的 | Web 壳契约（鸿蒙端布局独立，不套用） |
| admin 表格移动端列裁剪 | 低优先级列 `max-sm:hidden`（600px = platform 移动/平板阈值），只留主体/关键状态/操作三类列；共享常量 `admin/shared/table.ts` 的 `TH_SECONDARY`/`TD_SECONDARY` | Web admin 专属模式（鸿蒙 admin 为卡片页） |

**图标线宽四档**（Lucide strokeWidth；新增图标按档选，禁止引入第五档）：1.5 大尺寸/装饰性图标（size-5+，如 MomentCard 占位 MapPin 20）；1.75 标准 UI 图标（size-4/5 列表行、导航，全站默认）；2 小尺寸强调（size-3.5/4 状态类，如 Toast 状态图标）；2.5-3 极小可见性特例（size-3 选中态 Check，如 MenuPanel 选中勾）。跨端：harmonyapp 用系统符号（`sys.symbol.*`，线宽由 HarmonyOS 设计定，无 strokeWidth 参数），两端无法字面统一线宽，以「同尺寸视觉重量相当」为准。

**触控热区（强制）**：任何可点元素的**视觉高度 < 44px 时必须带 `hit-slop`**（或 `hit-slop-y`/`hit-slop-44`）。`.hit-slop` 用 `::before` 外扩 6px，实测换算：36×36 顶栏图标按钮 → 48×48 ✅；34×34 消息行更多操作 → 46×46 ✅；32（Button `h-8`）→ 44×44 ✅ 刚好达标；40（`py-1.5` 文本按钮）→ 52 ✅；28（筛选 chip）→ 40 ✗ 可接受（chip 为次级筛选、行长 ≥44，且带横向间距）。审计方法：`set viewport 390 844` → `open <路由>` → `eval` 遍历 `button, a[href], [role=button]` 收集 `getBoundingClientRect()` 与上表核对。

## 卡片交互契约（信息流卡/网格卡统一）

1. **层叠：卡片必须自成堆叠上下文**。所有图片内容卡根节点必须同时具备：`card-hover isolate group relative overflow-hidden`。原因：行内操作经根上 `[&_button…]:z-20` 抬升；若根无 `isolate`，这些 `z-20` 会在页面堆叠上下文里与页内 sticky header 的 `--cst-z-sticky(20)` 平级，DOM 更晚的卡片胜出 → 卡片 UI 压住标题栏；`isolate` 后卡片内 `z-20` 只在卡内生效，sticky header 稳定压住整卡。验收：移动端首页滚动到卡片作者行与吸顶 tab 重叠，`document.elementsFromPoint` 首位必须是吸顶条而非卡片按钮。同类陷阱：`ProfileMomentTile` / `ProfileGroupTile` 这类「卡片 + 卡片外 z-20 菜单」的包装层也要 `isolate`。
2. **按压：唯一入口 `use:cardPress`**。`.card-hover:active` 已删除，改 `.card-hover[data-pressed="true"] { transform: scale(var(--cst-card-tap-scale)) }`；`lib/motion.ts` 的 `cardPress`：卡本体/覆盖按钮按下 → `data-pressed`；行内 `button:not([data-cover])/a/input…` 按下不缩放整卡。禁止组件内手写 pointerdown/pointerleave + 内联 transform（会产生两套缩放量感）。
3. **操作行：唯一原语 `ImageCardActions`**。信息流大卡（HomeFeedItem）、网格卡（MomentCard）、我的瞬间卡全部使用；不允许在卡片里再手写点赞/评论/收藏/分享按钮。状态语义（全站唯一）：点赞 = 危险红填充（压图态白描边）；收藏 = accent 填充；评论/分享 = 中性；**不能只靠色相**（填充形状 + 颜色双信号）；压图态所有操作内容满亮度反色（不再叠 opacity）。交互：图标/计数一起 hover 放大（hover-action）、按压缩小、focus-ring、计数 count-pop（仅激活方向）、`aria-label/aria-pressed`、点击 `stopPropagation`。档位：`size="md"`（图标 18 / 间距 24 / 计数 12）；`size="sm"`（图标 16 / 间距 12 / 计数 12）用于手机 2 列窄卡，四类操作必须全部落在卡内（171px 卡实测量取右侧不越界）。
4. **压图徽章：唯一类 `.media-badge`**。深色遮罩底 + 反色字 + h-5 紧凑档；图标方角标用 `.media-badge--icon`；**禁止 backdrop-filter**；分类/位置/距离/1-N/已关注/身份徽章全部迁移，禁止再手写 `bg-surface-overlay-strong` 或 `bg-surface-media-scrim-strong + px + text-xxs` 组合；非压图 fallback 用 `.badge` + 主题色。
5. **作者区与分享**：作者区在有 userId 时可点（头像 hover-action、昵称下划线，stopPropagation → 主页）；无目标渲染纯文本。分享统一 `lib/moment-actions.shareMoment`（系统分享 → 剪贴板 + toast）；点赞/收藏统一 `setMomentLiked/setMomentSaved`（失败 toast + 调用方回滚乐观态）。

## 视觉语言：唯一实现与安静动作

- **安静次级动作唯一形态**：`ui/button variant="soft"`（中性软填 `bg-surface-subtle/60`、无描边、hover 全量）。编辑资料/设置/发消息/私信/拒绝/新会话/全部已读/表格行内动作一律用它；**不再手写软填胶囊或描边胶囊**。描边只留给可切换筛选 chip（TagPill/TabBar）与功能边界（输入框、主题 swatch）。**同级动作不混用**：主操作 = accent 实底；次级/安静 = soft；危险 = soft danger 底 + danger 文字；同一页/同一组动作不得混用「描边胶囊」和「软填胶囊」。关系 chip（关注/加入）两态同族：未激活 `surface-accent`、已激活 `soft`（已关注/已加入不再用 accent 描边）。
- **指标不再用灰药丸**：用 `MetricRow variant="strip"`（24px semibold tabular 数字 + 12px 次级标签 + `divide-x divide-border-subtle` 发丝分隔），每屏最多一处强调色（如「在线」绿点）；禁止每个数字套圆角底块、禁止「卡片里的卡片」。
- **分区标题唯一实现** `home/SectionHead`（icon 16 + 标题 14 semibold + count 12 + 右侧「查看全部」文字链；rail 变体 11）；页面不得再手写「标题 + 查看全部」行。
- **资料身份卡唯一实现** `profile/IdentityHero`（封面 132/92 + 头像 88/72 压封面 ring-4 + 身份/简介/标签 + MetricRow strip + actions）；页面不得自写身份卡；他人主页动作：未关注=关注 primary/私信 soft，已关注=私信 primary/已关注 soft（禁止第三种描边胶囊）。
- **空态**：页面级用新版 EmptyState（64px 软底图标砖 + 14 semibold 标题 + 12 描述 + `extra` 建议 chips/价值点），超大空态开 spotlight；访客态整块用 `auth/NeedsAuthNotice`（3 条 highlights + 主/次 CTA）。
- **图片卡文字层级**：标题 `text-base sm:text-lg` semibold leading-tight（压图仍白字，禁止降低遮罩对比度）；作者行头像 24–32 + 昵称 14 semibold + 时间 12；动作计数 12 tabular、动作间距 24；卡比例只随断点变一档（移动 3/4、≥600 4/5）。
- **选中态不得只依赖颜色**：`segmented` 选中 = accent 实底 + ✓ icon + `font-semibold`；`pills` 选中 = `font-semibold`；`underline` 选中 = `font-semibold` + 下划线。验收必须附 `filter: grayscale(1)` 灰度截图，选中项仍一眼可辨；新增 Tab/分段控件按此执行。
- **内容轴**：页面不得自设页面级 `max-w-*`，正文/页头/粘底条同轴（规则见 skill `hotspot-frontend-layout`「内容轴」）。
- **页面级容器与分组**：页面级去边框/连续画布、内容软岛、页头与分区硬线清零见 skill `hotspot-frontend-layout`；导航三态 token 与灰度验收口径见 skill `hotspot-frontend-theme`。

## 消息页排版与降噪

- 正文 `text-md`(14) + `leading-card`；时间戳 `text-xs`(11)；群聊发送者名 `text-sm`(12)；输入框 `text-md`(14)、高 44。
- accent 实底气泡内文字用 `text-text-on-accent` 全量，**不得叠 opacity**（旧 0.7 → 3.97:1 不达标）；对方气泡时间戳仍用 secondary（实测 5.36–8.84）。
- 日期分节 = 中性 11px 文本标签（禁止白底胶囊）；`ListEnd` 仅分页耗尽后出现；短会话底对齐（空白留在上方）。
- 会话行三态：选中 = accent 实底 + 长指示条（on-accent）+ semibold；未读 = 淡 tint + 短指示条（accent）+ 数字徽标 + semibold；普通 = 无条 medium。选中行文字 on-accent 全量。
- 圆角统一 `rounded-md`(16)；⋯ 菜单绝对定位不占文档流（否则底对齐最后一条会上移 ~40px）。

## 动效与运行时响应

- **统一入口 `lib/motion.ts`**：`MOTION.enter 180 / exit 150 / pop 220`、`prefersReducedMotion()`、`motionDuration()`、`pop` action、`playPageEnter()`、`toastExit`；CSS 用 `cst-page-enter` / `cst-state-pop` / `cst-comment-enter`。禁止手写新 duration/scale 数字。
- **页面进场预算（不减少特效、只去掉空白窗口）**：页面进场只允许客户端导航（`afterNavigate` 且 `nav.type !== 'enter'`）；客户端导航进场 ≤120ms、起点 `opacity ≥0.6` + 位移 ≤2px（禁止 opacity:0 起——点击后先空白再淡入就是「慢一拍」）；骨架→内容 fade ≤110ms；首屏不播（护 LCP）；只包内容层，不动 sticky/滚动/焦点。时长双源：CSS `.page-enter/.content-fade`（`--cst-page-enter-duration` / `--cst-content-fade-duration`）↔ `lib/motion.ts` 的 `MOTION.enter/contentFade`，`motion.test.ts` 源码锁定，改一处必须同步。
- **骨架→内容**只在「实际展示过骨架」时 fade，缓存命中不播；不得逐项 stagger 大列表。**状态 pop** 只在用户激活操作触发（点击 nonce/显式 handler）；初始数据同步/首帧已激活不播，取消不 pop。**toast 退场**用 `out:toastExit`，容器常驻（最后一条可退）；空 aria-live 无内容。
- **导航意图预取**：SvelteKit `data-sveltekit-preload-data` 只覆盖 `<a>`；button + goto 的导航（桌面侧栏/顶栏/位置卡/头像）必须经 `lib/nav-preload.ts`：hover/focus/touchstart 时 `preloadRoute()` + 5s 空闲兜底 `scheduleIdlePreload()`；外壳用 `<svelte:window>`（模板顶层）事件委托。dev 必须配 `vite.config.ts` `server.warmup.clientFiles`（全部路由 + 布局/消息/导航组件；glob 用 `src/routes/*/**` 绕开 `(app)` 括号陷阱），否则首访路由按需编译 58–720ms；只用 preloadCode（本应用无路由 load 数据）。
- **滚动与合成层**：唯一滚动容器 `main` **禁止 `scroll-smooth`**（会把 scrollTop 赋值/无参 scrollTo 动画化 766ms）；平滑只给显式入口（BackToTop 传 `behavior:'smooth'`）。键盘 PageDown 的 ~134ms 是浏览器原生，勿回退。`will-change` 只允许持续动画的固定图层；图片 hover 缩放、头像入口、批量交互类一律不加。
- **验收**：normal/reduced 双路径 computed + 中间态截图；无动画类/事件/定时器残留；QA 截图不得因动效抖动（动画 ≤260ms，动效 QA 建议在 reduced-motion 下取基线）；不得动画首屏 LCP 元素、不得引入 CLS。

## 媒体与图片

- **统一图片管线**：所有 `<img>` 用 `lib/images.ts`：`presetImageUrl/presetSrcSet`（avatar/thumb/card/cover/hero/full）+ 调用方 `sizes`；仅 unsplash 追加 `auto=format&fit=crop&q&w`，其他域名/本地/blob/地图瓦片原样；禁止裸写 `<img src>`。尺寸按**显示尺寸 × 2x/3x** 取档：avatar（显示 32/48→64/96/144）、thumb 200/400、card 400/800、cover 800/1600、hero 1080+、full 1600（ImageViewer 打开即加载）；调用方必须给 `sizes`（移动单列 100vw、桌面对应列宽/固定 px），否则浏览器按 100vw 选档浪费流量；首屏仅可见图 `loading=eager` + `fetchpriority=high`（feed 前 2/详情首图/hero/ImageViewer），其余 `lazy` + `decoding=async`；非 unsplash 不生成 srcset。
- **占位与降级**：网络媒体（头像/封面/缩略图/内容图）加载中必须有意义占位，不得纯灰圆/灰块：头像=首字母/UserRound；封面缩略图=`.media-placeholder` 主题渐变/图标；加载完成渐显覆盖（0.22s，reduced 直接显示）；失败保留占位或降级图标。Avatar 占位层始终渲染（`aria-hidden`），图片 absolute 覆盖；decorative/alt 语义不变；占位文字用 text-primary（accent-bg 上各主题/明暗 ≥4.5，text-secondary 在 inkwash 不达标）。有 Skeleton 的列表用骨架、不叠加占位；占位与图片同尺寸（无 CLS）。
- **整卡图片卡（与 design-system §17 同源）**：图像优先的内容卡图片 `object-cover` 铺满整卡，信息以顶部/底部遮罩覆盖层压在图上；禁止「图片 + 白色信息托盘」两段式（横排 `horizontal` 列表卡为登记例外）。卡片不设 border；圆角 `rounded-lg`(20)；hover 抬升走 `card-hover`（图片微缩放 + 卡片上浮）。遮罩只用 `--cst-surface-card-scrim-top` / `--cst-surface-card-scrim-bottom` / `--cst-surface-card-fallback`（三主题×明暗）；禁止组件内写 rgba/gradient 字面量。压图白字/白图标对比度 **≥4.5:1**（纯白底图最坏情况，字形区像素法实测，方法见 `references/audit-methods.md`）；徽章一律深色强遮罩 + 白字，禁止浅玻璃底 + 白字；点赞/收藏等状态用填充形状 + 白描边，不只色相。位置信息只在右上徽章出现一次；底部元信息只留「时间 · 标签」；多图 → 首图 + 1/N 徽章。无图 fallback：主题渐变 + 0.08 主题水印图标（64px，`--cst-opacity-watermark`）+ **主题色文字**（不压白字）。
- **占位动画限时**：图片占位 pulse 用 `.media-placeholder--pulse`（3 × skeleton token ≈4.5s 后静态）；**禁止 `animate-pulse`（infinite）做图片占位**（慢图下批量常驻）；reduced-motion 由全局规则压制。验证口径：加载完成后 `document.getAnimations()` 中 pulse 计数归零（实测 390 首页 7→0）。
- **新图片接入必须同时给**「占位 + 失败降级 + 尺寸档/srcset」（design-system §20.8/§20.10）。

## 骨架屏 / 场景化空态 / 页头图标

- **首屏骨架**：高频列表/信息流一律骨架屏（`data-display/Skeleton*`：`SkeletonList` / `SkeletonFeed`），禁止再用 spinner 占位：与真实内容同栅格/同列数/同卡片外壳，做到「骨架→内容零位移」（首块 y/高度差 ≤2px）。几何与鸿蒙 `Skeleton`/`SkeletonShimmer` 同源（不要单独改一端）：行内上下 12、前导圆 36/40/44/48、文本条 14·55% + 11·35%、条间距 8、圆角 `rounded-sm`(12)；脉冲 `--cst-motion-skeleton`(1500ms)，reduced-motion 下保留骨架形、停止脉冲。语义：外层 `role=status + aria-busy=true + aria-label`（「加载动态中…」等）；占位块 `aria-hidden`、不可聚焦；ListContainer 内自动补 `listitem`。**不得换骨架**：分页加载更多（LoadMoreButton/spinner）、表单/设置/admin 表格/聊天首屏（异构或短促，骨架行高与真实气泡差异会造成滚动跳动）。新增页面接入前先看 `docs/ui-components.md`「骨架屏使用口径」。
- **场景化空态装饰**（`EmptyState.spotlight`）：只用于**超大空态**（整页/整块内容区：访客引导、我的内容空、搜索无结果）；窄面板/小空态不得开启。几何与鸿蒙 `EmptySpotlight` 同一 spec：光晕 320 + 外环 288 + 内环 168 + 四向星点 4px@(±102, ±102)；颜色只取 `--cst-ambient-*`；无动画；装饰层 `aria-hidden` + `pointer-events:none` + `z-index:-1`（父级 `relative isolate`）。规范全文见 `docs/design-system.md` 第 16.6 节；改几何/颜色必须两端同步。
- **页头动作图标钮**：品牌头（首页 logo + wordmark）用 ghost 变体（透明底裸图标，36px）；标准 PageHeader（内页）用 surface 变体（浅底圆钮 + hairline，36px），不用 ghost；图片/封面上悬浮的返回/更多用 overlay 变体（平台适配，保留）。36px 视觉 + `hit-slop` = 48px（≥44）；纯图标按钮必须带 `aria-label`。详细口径与鸿蒙映射见 `docs/ui-components.md`「页头动作图标钮口径」。

## 质量门与性能红线

- **每次改 UI 后跑**：`bun run audit:a11y`（axe 0 serious/critical）、`bun run audit:touch`（390 触控无未登记离群）、`bun run audit:ui`（含 browser-qa）；reduced-motion 用 `bun run audit:a11y -- --mobile --reduced`（残留类/长动画即失败）。触控例外只允许走 `scripts/ui-quality-touch-allowlist.json` 稳定 selector + 理由（禁止 nth-child），维护说明见 `docs/frontend-ui-audit.md`；命令依赖 dev 环境 + agent-browser，不进 `check` 链；`--limit` 仅调试。
- **合并前主题矩阵**：`bun run audit:a11y -- --matrix --parallel 2`（约 15–20min，0 serious/critical）；快查 `--only/--limit`；默认 `audit:ui` 仍只扫 vibrant light。三主题×明暗关键页必测；状态徽章用实底 + on-accent 字（避免淡底 + 主题字在部分主题/明暗下 <4.5）。
- **a11y 结构基线**：整卡可点必须用「覆盖按钮 + 行内动作 auto-lift」，卡片根不得持 button/aria-label（nested-interactive）；`role=list` 子节点补 `role=listitem`；listitem 不挂 aria-label；`dl` 只包 dt/dd；无 role 的 div 不挂 aria-label；每页有 `main`；可滚动非键盘可达容器加 `tabindex=0` + `role=region` + `aria-label`；装饰图 `alt=""` + `aria-hidden`。新的可点结构先跑 axe 再加。键盘验收：Tab 全站可达、skip link 首 Tab 可见、去描边不得丢焦点环、父容器 overflow 不得裁切、亮/暗都测（键盘截图 + computed outline）。
- **性能红线（违反即回退）**：全屏/大面积无限动画默认禁止——氛围画布基础类不带 animation，只有 `.ambient-backdrop--motion`（`<AmbientBackdrop motion />`，仅空态/稀疏页）才播放；新增无限动画必须先给 idle GPU 实测。`will-change` 不得挂在批量交互类上（`.hover-btn/.hover-action` 禁止；首页 48 个实例 = 48 个常驻层）。`backdrop-filter` 只给少量大面积 glass（右栏玻璃卡/吸顶胶囊），禁止给批量小角标。源码契约守卫 `src/styles/__tests__/perf-contract.test.ts`（改 CSS 后必须过）。CLS 应 <0.1（图片显式尺寸/懒加载/骨架零位移）；性能基线：关键页 LCP/CLS 采样（本地 dev 环境仅作相对参考）。
- 完整命令、主题矩阵与 reduced-motion 验收明细、GPU/性能实测基线与定位方法见 `references/quality-gates.md`；压图对比度像素实测、原语绕过判据与审计脚本假阳性规律见 `references/audit-methods.md`。

## 反模式与陷阱

- **替换元素绝对定位必须显式 w/h 拉满**：`<img>` 等替换元素 `position:absolute` + `inset-0`（含 `inset-y-0`/`inset-x-0`）**拉不开**——CSS2 §10.3.7：abs 定位替换元素 `width/height:auto` 时按固有尺寸渲染，left/right 锚定不参与拉伸；`object-cover` 因此无从裁剪。表现为小图按原尺寸在卡片内露底、大图按固有比例错裁，且大图恰好溢出时被 `overflow-hidden` 裁得「看起来正常」，同页面只有小图露馅，极难肉眼归因（装饰 div 是非替换元素，inset 全拉锚有效——同一原语里 div 正常、img 露底更迷惑）。**规则**：图片优先卡片原语里，abs 定位的 img 一律显式 `h-full w-full`（横排局部列如 `h-full w-2/5`），不靠 inset 拉伸；inset 全拉锚仅用于非替换装饰层。验证：jsdom 不做布局，svelte-check 与既有单测无感——类串断言拦回归（`NearbyImageCard.test.ts`「图片盒显式 w/h 拉满」用例），最终以浏览器实测 `img.getBoundingClientRect()` 对比卡片 rect 为准（修复后应≈99%w/99%h）。
- **`hit-slop` 不能用在绝对定位元素上**：它自带 `position: relative`，而 `styles/theme/utilities.css` 在 `tailwindcss` 之后导入、同 `@layer utilities` 同权重下后者胜 → 会把 `absolute` 压成 `relative`。地图图钉（`MapPinMarker`，`absolute` + `translate(-50%,-100%)` 地理投影）加 `.hit-slop` 后实测 17 个 marker 只剩 12 个仍落在覆盖层内、坐标整体错位（投影失效）。此类元素改用**只补伪元素、不碰 position** 的等价写法：`.marker-hit::before { content: ''; position: absolute; inset: -6px; }`（见 `MapPinMarker.svelte`）。
- **压图文字对比度不可用自动指标**：`getComputedStyle` 取色拿到的是父级不透明度叠加值而非图片像素，对压在图片上的文字**不可信**（hotspot 审计脚本即按 `contrastOverMedia` 分类，不计入 AA 结论）。要判定必须像素级实测，且测量区取「字形区」而非元素矩形（用元素矩形含 padding 会把最差比压到 1.0–4.0 造成假阴性）；压图徽标一律用深色遮罩（`bg-surface-media-scrim-strong` = 0.55 黑，白字在纯白底上仍有 ~4.9:1），不要浅玻璃底 + 白字（`glass-badge` 0.18 白是结构性不达标）。完整方法见 `references/audit-methods.md`。
- **不要拿鸿蒙的原语标准套前端**：前端与鸿蒙的原语纪律**不同**——鸿蒙强制 SystemButton 原语；前端允许原生 `<button>` + token 类（Svelte/Tailwind 惯例），约束落在**交互反馈类**（第 7 条）。据此对全仓 209 处原生 `<button>` 审计结论为无缺口。合规基线（命中任一即合规）：反馈工具类（`hover-btn`/`hover-bg`/`hover:bg-*`/`tap-*`/`scale-btn`/`focus-ring`/`focus-visible:ring-*`）、`link-btn` 工具类、共享常量 `ROW_BTN / ROW_BTN_PRIMARY / ROW_BTN_DANGER`（`routes/(app)/admin/shared/table.ts`）、`ListItem` 组件。审计脚本的坑（曾 106/209 误报）与判据详见 `references/audit-methods.md`。

## references

- `references/quality-gates.md`：质量门命令、主题矩阵与 reduced-motion 验收、性能实测基线（GPU 定位与 A/B 方法）、页面进场/预取细则。
- `references/audit-methods.md`：压图文字对比度像素实测法、前端按钮「原语绕过」审计判据、审计脚本假阳性规律。
