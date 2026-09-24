---
name: hotspot-frontend-style
description: hotspot 前端样式与交互状态契约：样式自查清单 19 条、交互状态契约、按钮 hover 两档、禁止双重边框等。触发：新增或修改任何前端样式；写 Tailwind 类名；使用 design token；处理 hover/focus/active/disabled 状态；列表行/弹层/输入框样式；样式自查。
---

# hotspot-frontend-style

## 样式自查清单（新增/修改样式后逐条过）

> **核心原则：少档位 + 强制收敛**。设计系统的价值在约束而非自由——档位要少、
> 近邻值必须归并（1-2px 差异人眼分不清，并存只会让界面碎、无节奏）。「不够用就
> 归到最近档位」是默认动作，新增档位需充分论证。同一类东西禁止变体过多。
> 各档位的**双端对照数值与语义用法**的唯一权威是根 `docs/design-system.md`
> （圆角第 6 节 / 字号第 4 节 / 间距第 5 节 / 模糊第 9 节 / 动效第 10 节 /
> 阴影第 8 节）；本清单只保留前端执行规则与速记档位，改 token 数值时以
> design-system.md 为准并两端同步。

1. **圆角（4 档 + full）**：控件/卡片圆角一律用 `rounded-*` 类（`--radius-*` 档位：
   **sm 12 / md 16 / lg 20 / xl 24px** + `rounded-full` 圆形 + `rounded-sheet` 28px 弹层）
   或 `var(--cst-shape-sm/md/lg/xl)`；禁止 `rounded-[13px]` 等任意值、禁止 style 内
   硬编码 `border-radius:20px`、禁止近邻档位并存（原 compact/tight/small/control/
   relaxed/panel/feature 7 档已收敛到 4 档）。例外仅限物理形状细件
   （开关半高圆角、状态条 1-3px 指示条、拖拽把手）
2. **颜色**：组件内禁止 hex/rgba 字面量（数据层同样禁止）；arbitrary 类
   （`bg-[#...]`/`shadow-[...]`）禁止含色值。阴影色值统一走 `--shadow-*` token
   （见上方第 10 条「阴影」），不得在 `shadow-[...]` 或内联 `box-shadow` 里手写色值
3. **字号（强制，9 档）**：全站字号一律用 `text-*` 刻度类——**4 档自定义 token
   （`--text-*`：xxs=10 / xs=11 / sm=12 / md=14）+ Tailwind 默认 5 档**
   （text-lg 及以上 18/20/24/30/36），与 `check-cross-platform-scale.mjs` 的
   `FONT_MAP`、`frontend/AGENTS.md`「字号 9 档」同口径（跨端一致性见 skill `hotspot-harmony-ui`）。
   - xxs=10：极小 uppercase 标签（eyebrow、分组标）
   - xs=11：辅助文字（副标题、计数、时间戳）
   - sm=12：正文（原 13px 已归并，消除 1px 噪声）
   - md=14：强调正文（昵称、按钮主标、侧栏导航）
   **禁止 1-2px 近邻档位并存**（原 xxxs/15/16/base 已砍/归并）；**禁止 `text-[Npx]`
   arbitrary 类、禁止 style 内联 `font-size:Npx`**（含动态 ternary——移到 class 三元）。
   **已登记例外**（与 `base.css` 的 h1-h4 元素级映射同等地位）：
   - `Avatar` fallback 首字母 `font-size:{size*0.42}px`：随头像尺寸等比缩放，物理比例
     恒定；字号档位类不适用于此例外，保持内联按尺寸计算。
   > `PageHeader` 的 `titleSize`/`subtitleSize` 已重构为**字号档位 prop**
   > （`md/base/lg/xl/2xl/3xl` → 映射 `TITLE_SIZE_CLASS`），不再有内联 `font-size`。
   > 全站页面统一传 `device === 'desktop' ? 'xl' : '2xl'`（桌面 20 / 平板·移动 24），
   > 不得逐页自选档位——标题层级是页面锚点，跨页漂移会破坏节奏。
4. **accent/status 实色背景上的内容**：文字/图标必须用对应 `on-*` token
   （`text-text-on-accent`/`text-text-on-status`/`text-content-on-marker`），
   禁止恒白 `content-inverse`；选中行内若有与 accent 同色的 marker 图标需
   实色底衬托（PinMedia `selected`）
5. **列表行**：交互行一律走 `ListItem`（高亮圆角/背景/反色由组件统一处理），
   不在页面里自写 hover 背景或叠行级卡片类
6. **token 注册与类名有效性（强制）**：新 token 若需类名消费，同步补
   `theme-registry.css` 注册，并核对类名不带 `cst-` 前缀（`text-text-on-status`
   而非 `text-cst-...`）。Tailwind v4 对**未注册 utility 静默失效**（不报错、
   不生成 CSS、svelte-check 无感），因此：
   - 写任何带 token 语义的类名前，确认它在注册层/定义层/项目自定义 `.class` 中真实存在；
   - 项目自定义工具类拼写必须与 `utilities.css` 一致（如 `no-scrollbar`，
     曾误写 `scrollbar-none` 静默失效）；
   - `bun run check:theme` 已串联 `scripts/check-utility-classes.mjs` 自动拦截，
     本地 `bun run check` 与 CI 都会失败——不要绕过该检查。
7. **hover/tap 类**：交互反馈用既有工具类（`hover-btn`/`hover-bg`/`scale-btn`/
   `tap-*`/`header-action`/`surface-ghost`/`card-hover`），不新发明交互类；
   禁止双重边框（高亮与元素自带边框不得叠成两圈），
   具体契约见本节末尾「交互状态契约」
8. **字距（强制）**：字距一律用 `tracking-*` 刻度类，禁止 `tracking-[Nem]` arbitrary
   类与内联 `letter-spacing:Nem`：
   - `tracking-eyebrow`=0.14em：uppercase 分组标签（外观/通知/通用 section 标、
     「在线/未读/最近」小标、「新消息/较早」通知分组标）
   - `tracking-label`=0.12em：次级 uppercase 小标（DesktopLayout 侧栏分组）
   - `tracking-field`=0.08em：表单字段标签（AuthForm 昵称/密码/验证码）
   - `tracking-title`=-0.01em：大标题微收紧（用户名、昵称 h1/h2）
   - Tailwind 内置 `tracking-tight`(-0.025em)/`tracking-wide`(0.025em) 用于
     状态栏时钟收紧、连接横幅微扩等边界场景
   **动态字距**（如选中态切换）用 class 三元（`{active?'tracking-x':''}`），禁止
   `style="letter-spacing:..."` 内联
9. **行高**：禁止 `leading-[N]` arbitrary 类与内联 `line-height:N`（一律移到
   `leading-*` 类）。优先用语义档位 `leading-card`(1.55 摘要/副标题) /
   `leading-copy`(1.75 正文阅读) / `leading-none`(单行紧凑如数字、徽标)；
   多行正文段落允许用 Tailwind 默认 `leading-5/6/7`（固定 px 行高，对可读性
   敏感、随字号变化需要不同行距，不强制收敛）。
10. **阴影（强制，3+2+3 档）**：阴影一律用 `shadow-*` token 类或
    `var(--shadow-*)`/`var(--cst-elevation-*)` 引用，禁止 `shadow-[0_3px_10px_...]`
    arbitrary 类、禁止内联 `box-shadow:Npx...` 手抄值、禁止 utilities.css 自行组合
    双向投影表达式。两套投影语言，按语义选择：
    - **双向凸起/凹陷**（控件/卡片/头像立体感）：`shadow-raised-sm/md/lg`（3 档：
      sm=4/4/9 控件、md=5/5/12 卡片、lg=8/8/18 大卡片）+ `shadow-inset-sm/md`（2 档凹陷）。
      消费 `--cst-elevation-shadow-{dark,light}` 原料——vibrant 显影、forest 透明（改由 border 表达层级）
    - **accent 光晕**（accent 实色背景发光）：`shadow-glow-sm/md/lg`（3 档：
      sm=0/3/10、md=0/4/14、lg=0/6/18）
    - **Material 单向/弹层投影**：`var(--cst-elevation-card)`/`-button`/`-modal`/`-sheet`/`-menu`
      （语义 token，themes/{vibrant,forest,inkwash}.css 各自定义真值）；弹层/下拉菜单用
      投影表达层级，不依赖描边环
    **地图标记光晕**：结构 token `var(--cst-marker-shadow)`（0 2px 10px 定位点）/
    `var(--cst-marker-halo)`（0 0 8px 小光点），色值由调用方按 marker tone 提供
    （`var(--cst-marker-*-glow)`），定义于 shared.css——禁止组件内联手抄偏移量
    **命名原则**：按投影方向/语义命名（raised/inset/glow/elevation-card），禁止设计风格前缀（neu-/skeuo-/glass-）。
    **已登记例外**：`me/edit` 保存按钮 `box-shadow:0 4px 14px {saved?'online-glow':'accent-glow'}`（固定尺寸+动态双色）
11. **font-weight（强制）**：字重一律用 `font-*` 类（`font-medium`=500/
    `font-semibold`=600/`font-bold`=700/`font-normal`=400），禁止内联
    `style="font-weight:N"`。动态字重（选中态切换）用 class 三元
    （`{active?'font-semibold':'font-normal'}`）
12. **页面纵向间距（强制）**：正文 `contentClassName` 收敛到样板 `/nearby/moments`
    的约定值，禁止 `py-5`/`py-6`/`py-7 lg:py-8` 等散乱值：
    - 标准 header 页：`contentClassName="pt-4 pb-6"`（顶 padding 由 header `pt-4`
      承接，正文上下 `pt-4 pb-6`）
    - 长表单页（settings、me/edit）：`pt-4 pb-10`（尾部加大留白）
    - 封面型页（无标准 header，顶由封面承担）：`pb-6`
    - 资料总览长内容页（ProfilePage 总览）：`pt-6 pb-8`（呼吸型变体）
    - 状态/空态区：`py-20`（样板 `feedState` + `EmptyState` 内置）
    **header 内 ContentColumn** 统一 `pt-4 pb-0`（顶留白、底不留白，below 区自带头部
    间距）；`below` snippet 首元素必须 `mt-3`（与样板一致，见页面布局契约第 2 条）。
    **窄列特例**（不进 `--cst-page-gutter`，用 `px-4`（16px）压缩 gutter，已登记例外，
    其他窄列场景需在 AGENTS.md 补登记）：
    - 桌面消息分栏左栏（`w-75`=300px）
    - 桌面地图左侧面板（`w-85`=340px，`LocationDesktopPanel`：PageHeader 传
      `contentClassName="px-4 pt-4 pb-0"`，搜索/Tab/列表/确认区统一 `px-4`）
    **布局尺寸 token**（layout.css 定义，禁止组件内裸数字）：
    `--cst-desktop-sidebar-width`（224px 展开）/ `--cst-desktop-sidebar-collapsed`（72px 折叠）/
    `--cst-tablet-dock-max`（680px 平板 Dock）/ `--cst-cover-height`（clamp 220→340px 封面）/
    `--cst-cover-height-lg`（clamp 220→380px 大封面）/ `--cst-cover-height-sm`
    （clamp 120→160px 编辑资料小封面）
13. **尺寸与间距**：UI 控件/图标尺寸优先用 Tailwind `size-*` 刻度类（`size-9`=36px
    等），减少内联 `style="width:Npx;height:Npx"`。**间距**（gap/px/py/mt/mb）用
    Tailwind 默认 4px 步进刻度（`gap-1/1.5/2/2.5/3...`），**不强制收敛、不建 spacing
    token**——间距的 2px 差异是布局设计的一部分（卡片内 8px、section 间 16px 等
    刻意微调），与字号/圆角的"1px 无感噪声"性质不同，强行归并会破坏视觉层次。
    横向页面 gutter 是唯一例外（`--cst-page-gutter`，由 ContentColumn 统一管理）。
    存量内联尺寸（地图 PIN、StatusBar 信号/电池等物理装饰细件）可保留为例外
    布局尺寸例外（非控件尺寸，可用类时仍优先用类）：对话框高度
    `78vh/max-height:700px`（桌面居中模态统一档）、移动 bottom-sheet 高度
    `85%`（新会话）/`92%`（建群，表单更高）、聊天空态 `min-height:240px/max-width:220px`、
    上传占位 `height:110px`、封禁页 `max-width:360px`、横幅 `h-7` 类化后
    仅剩物理例外——新增此类布局尺寸时优先 Tailwind 类，确需内联则登记
14. **毛玻璃模糊半径（强制，3 档）**：`backdrop-filter:blur()` 半径一律用
    `var(--cst-blur-sm/md/lg)` token（sm=8 轻徽章、md=16 sticky header/overlay、
    lg=24 底部导航/tablet dock），禁止内联 `blur(Npx)` 字面量（原 8/10/16/18/24/28px
    六种已收敛到 3 档）
15. **opacity**：优先用 Tailwind `opacity-*` 标准档类（10/20/30.../100），减少内联
    `style="opacity:0.N"`。装饰性图标/文字透明度应归标准档（0.5→opacity-50 等）；
    导航激活态背景等精确视觉调校允许保留内联值（0.12~0.15 区间属刻意微调）
16. **焦点可见性（强制）**：所有可交互元素必须有可见键盘焦点环。统一用
    `focus-ring` 类（`focus-visible` 时 accent outline）或等效的
    `focus-visible:ring-2 focus-visible:ring-accent-default`；**禁止无替代地使用
    `outline-none`**（Tailwind v4 的 `outline-none` 会压掉 base.css 的全局按钮轮廓）。
    图标按钮必须同时提供 `aria-label`；紧凑图标按钮（<44px）叠加 `hit-slop` 扩展触控热区
17. **减弱动效（强制）**：新增加载/装饰动画必须考虑 `prefers-reduced-motion`；
    utilities.css 已提供全局降级（动画瞬时化、`map-pulse-halo` 停用、
    spinner 放慢），不得绕过
18. **动效曲线（强制）**：过渡缓动曲线一律用 `var(--cst-ease-*)` token，禁止内联
    `cubic-bezier(...)` 字面量：
    - `--cst-ease-standard`=(0.22,1,0.36,1)：标准减速曲线（绝大多数 transition）
    - `--cst-ease-bounce`=(0.34,1.56,0.64,1)：弹簧 overshoot 曲线（FAB、toggle 滑块、
      地图标记放大、sheet 拖拽回弹等「有弹性」的入场/位移）
    - Tailwind 类消费经 `--ease-standard`（theme-registry 注册）
    **duration 不强制 token 化**：`0.10s`~`0.30s` 的精细时长差异是 Figma 动效契约的
   有意设计（btn 0.17s、action 0.14s、tap 0.10s 等），`--cst-motion-standard:220ms`
    仅作基准参考，不代表统一时长；组件按 Figma 契约保留各自动效时长
19. **z-index 层叠**：分两类，不混用：
    - **全站页面级层叠**（弹层/抽屉/Toast 等跨组件浮层）：用 `--cst-z-*` token
      （dropdown=10 / overlay=40 / modal=50 / toast=60），经 `z-[var(--cst-z-modal)]`
      等引用
    - **局部 stacking context 内层叠**（卡片角标盖图片、地图容器内部排序、sticky header）：
      用 Tailwind 默认 `z-10/20/30` 或 arbitrary `z-[1]/z-[2]`，表达「盖几层」的相对
      语义，**不套全站 token**——它们在各自 `position:relative` 容器内独立成层，
      与全站 z-index 刻度无语义对应。地图组件内部的 `z-index:0~50` 内联属此类
      （地图外层 `overflow:hidden` 形成独立 stacking context）

### 交互状态契约（现状即规范，改动前先对齐）

| 元素 | hover | focus / active |
|------|-------|----------------|
| 输入框（input-surface） | — | 边框加深（border-strong）+ 内缘 1px 加深线 + 内凹阴影（`--shadow-inset-sm`），底色保持中性 |
| 图片卡（image-hover-frame） | 内嵌 accent 2px 描边 + 图片缩放 1.045（仅 fine pointer） | — |
| 带边框卡片（card-hover） | 边框变 accent（`hover:border-accent-default`）或投影抬升，不画外环 | — |
| 可交互 surface-flat | 描边加深（border-strong） | — |
| 弹层 / 下拉菜单 | 无描边环，层级靠 elevation-menu 投影 | — |

#### 按钮 hover 底色：两档，按「控件是否有可见轮廓/底色」选

utilities.css 的全局规则会给**无 `bg-*` 且无 `hover:*` 类**的 `<button>` 自动填
`--cst-interactive-hover-fill`（仅 fine pointer）。因此有底色/描边的按钮被该规则
排除，必须**显式**给 hover；两档按控件是否"有可见轮廓"选择，不要混用：

| 控件形态 | hover 底色 | 例 |
|----------|-----------|-----|
| 透明底图标按钮 / 行式按钮 / 未选中 pill | `hover:bg-interactive-hover`（中性灰） | 图标按钮、ListItem、`CreateSheet` 心情/话题按钮 |
| 有可见轮廓或底色（border / ring / `bg-surface-primary`） | `hover:bg-accent-soft`（accent 淡底，更强信号） | `ProfilePage` 描边白底按钮、`AppearanceMenu` ring-1 控件、下拉菜单项 |

- **禁止透明度越档**：`hover:bg-accent-soft/50` 之类 `/N` 变体一律用整档
  `hover:bg-interactive-hover`（accent-soft 本身就是 10~14% 淡底，再砍半档等于
  用 opacity 造新档位，违反上方第 15 条「opacity」）。
- 已选中态（`aria-pressed=true` / `bg-accent-soft` 打底）的 hover 走
  `hover:bg-interactive-active`，不用 interactive-hover。
- 导航 Tab 不参与本表：`.tab-item` 规则统一提供 `--cst-interactive-hover-fill`
  （无 layer，优先级高于 `@layer utilities`），**不要在 TabBar 上再写 `hover:bg-*`**
  （写了也不生效，是死类名）。

硬性规则：
- 输入框聚焦态由整框承载：内层 input/textarea 不画自身焦点框
  （utilities.css `.input-surface:focus-visible` 统一压制，与上方第 16 条「焦点可见性」配合）；
  底色保持中性，不引入 accent 色，避免与错误态等 accent 元素抢视觉。
- 禁止双重边框：card-hover 只抬投影不画外环；带边框的卡片 hover 用
  `hover:border-accent-default`；surface-flat 的 hover 描边仅对可交互元素生效。
- 图片缩放必须由外层 `overflow:hidden` + 圆角裁剪，图片任何像素不得溢出卡片。
- **禁止渲染空 src 图片**：`<img src="">` 会被浏览器解析为当前页面 URL 而显示破图
  （"我的瞬间"页曾因此对无图动态整体破图）。无图/空 URL 一律走占位：
  大图区用 `MediaPlaceholder`（data-display，ImageOff 图标 + accent-soft 底），
  头像用 `Avatar`（首字母兜底），列表小缩略图内联 `{#if url}` 分支。
- **计数变化反馈**：点赞/评论等瞬时数值变化用 `count-pop` 类 + `{#key 值}` 重建元素
  触发弹跳（utilities.css cst-count-pop，snappy + bounce，自动遵循 reduced-motion）。
- **大图查看统一入口**：任何"点击查看大图"交互一律复用 `components/overlays/ImageViewer`
  （bits-ui Dialog 底座：Esc/外点关闭、左右键/滑动切图、单击图片切换放大（以点击
  位置为锚）与滚轮连续缩放、放大后拖拽平移、缩略图条），禁止各页自建 lightbox。
  **入口仅限动态详情页**（单图封面/多图网格）；列表卡片与画廊一律整卡跳详情，
  不在卡片上直接开大图。
- **地图筛选菜单统一**：`map/shared/MapFilterMenu.svelte` 是桌面/移动唯一筛选实现
  （Popover 外点/Escape 关闭、徽章、至少一项约束、「全部显示」重置）；地图空白点击
  一律经 `MapCanvas onMapTap`（内部区分拖拽，拖动不误触），禁止父级用原生 onclick 判定。
