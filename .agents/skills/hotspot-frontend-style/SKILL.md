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
   `leading-*` 类）。优先用语义档位 `leading-card`(1.7 卡片正文/摘要/副标题——
   **原 1.55 已因阅读疲劳修复上调**，见 theme-registry.css `--leading-card`；
   鸿蒙端 `FONT.LINE_RATIO.CARD = 1.7` 同步) / `leading-copy`(1.75 正文阅读) /
   `leading-none`(单行紧凑如数字、徽标)；
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
    - 标准 header 页：`contentClassName="pt-4 pb-6"`（页顶 16 由页头自身内距承接，
      正文上下 `pt-4 pb-6`）
    - 长表单页（settings、me/edit）：`pt-4 pb-10`（尾部加大留白）
    - 封面型页（无标准 header，顶由封面承担）：`pb-6`
    - 资料总览长内容页（ProfilePage 总览）：`pt-6 pb-8`（呼吸型变体）
    - 移动端固定底部操作条补净空：`pt-4 {platform.platform === 'mobile' ? 'pb-28' : 'pb-6'}`
      （users/[userId]、moments/[momentId]）
    - 状态/空态区：`py-20`（样板 `feedState` + `EmptyState` 内置）
    **页面级/区块级间距三档（12/16/24）的落地口径见 skill `hotspot-frontend-layout`
    「纵向节奏 → 前端落地口径」**（根容器 `gap-4`、区块间 24、标题→内容 12，
    且间距只允许单侧提供）。
    **`PageHeader` 内部 ContentColumn 的上下内距由组件派生**（有 below：`pt-4 pb-3.5`；
    无 below：`pt-4.5 pb-4.5`），属页头组件内部构图，**页面不要再传 `contentClassName`
    覆盖**（唯一例外是已登记的窄列面板）；`below` snippet 首元素必须 `mt-3`（与样板一致，
    见页面布局契约第 2 条）。
    **窄列特例**（不进 `--cst-page-gutter`，用 `px-4`（16px）压缩 gutter，已登记例外，
    其他窄列场景需在 AGENTS.md 补登记）：
    - 桌面消息分栏左栏（`w-80`=320px）
    - 桌面地图左侧面板（`w-85`=340px，`LocationDesktopPanel`：PageHeader 传
      `contentClassName="px-4"`，搜索/Tab/列表/确认区统一 `px-4`）
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

## 压图文字的对比度必须像素级实测，且测量区取「字形区」而非「元素矩形」

自动指标（getComputedStyle 取色）对压在图片上的文字**不可信**——它拿到的是父级
不透明度叠加值而非图片像素，只能把这类元素单列（hotspot 审计脚本即按
`contrastOverMedia` 分类，不计入 AA 结论）。要判定就得实测像素：

1. 找到候选元素 → 用 `document.createRange(); selectNodeContents(el)` 取**文字节点自身的
   `getBoundingClientRect()`**；
2. 截图该矩形 → 把元素 `style.color='transparent'`（**只改颜色，保布局**）→ 再截同一矩形，
   得到「纯背景」像素；
3. 逐像素算 `contrast(文字色 over 背景像素, 背景像素)`，取 min 与 P5；文字带 alpha 时
   逐像素合成 `eff = fg*a + bg*(1-a)` 后再算。

**关键坑（2026-10-18 实测，会得出完全相反的结论）**：测量区若用**元素矩形**（含 padding），
padding 下根本没有字形、那里是胶囊圆角外的原图像素，会把最差比压到 1.0–4.0 造成假阴性。
同一批 176 个候选：用元素矩形 → 15 处「不达标」（6 组）；改用字形区 → **仅 1 处**
（`/groups/group_1` 分类胶囊，白字压 `glass-badge` 0.18 白玻璃底，实测 4.16/需 4.5，
84.9% 字形像素不达标）。修复后同法复测：4.16 → 14.18、达标像素 13.3% → 100%。

压图徽标一律用**深色遮罩**（`bg-surface-media-scrim-strong` = 0.55 黑，白字在纯白底上
仍有 ~4.9:1），不要用浅色玻璃底 + 白字（`glass-badge` 0.18 白），那是结构性不达标。

    ⚠️ **`hit-slop` 不能用在绝对定位元素上**（2026-10-18 实测）：它自带 `position: relative`，
    而 `styles/theme/utilities.css` 在 `tailwindcss` 之后导入、同 `@layer utilities` 同权重下
    后者胜 → 会把 `absolute` 压成 `relative`。地图图钉（`MapPinMarker`，`absolute` +
    `translate(-50%,-100%)` 地理投影）加 `.hit-slop` 后实测 17 个 marker 只剩 12 个仍落在
    覆盖层内、坐标整体错位（投影失效）。此类元素改用**只补伪元素、不碰 position** 的等价写法：
    `.marker-hit::before { content: ''; position: absolute; inset: -6px; }`（见 `MapPinMarker.svelte`）。
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
20. **背景装饰层（氛围画布）**：页面不允许自造背景纹理/插画；全站唯一实现是
    `components/layout/ambient/AmbientBackdrop.svelte`（PageViewport 等处已挂载，
    新页面自动继承）。若确需在组件内加装饰层，必须同时满足：
    `aria-hidden="true"` + `pointer-events:none` + 父容器 `relative isolate` + 自身
    `z-index:-1`（或明确的局部相对层）、颜色只用 `--cst-ambient-*` / 既有 token、
    动效走 `transform/opacity` 且 reduced-motion 停用。完整规范见
    `docs/design-system.md` 第 16 节。

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


## 前端按钮「原语绕过」审计判据（2026-10-12 实测，避免误报）

前端与鸿蒙的原语纪律**不同**：鸿蒙强制 SystemButton 原语；前端允许原生 `<button>` +
token 类（Svelte/Tailwind 惯例），约束落在**交互反馈类**（自查清单第 7 条）。据此对全仓
209 处原生 `<button>` 做了审计，**结论：无缺口**。合规基线（命中任一即合规）：

- 反馈工具类：`hover-btn` / `hover-bg` / `hover:bg-*` / `tap-*` / `scale-btn` /
  `focus-ring` / `focus-visible:ring-*`
- `link-btn` 工具类（自带 transition + underline 反馈）
- 共享常量：admin 页的 `ROW_BTN / ROW_BTN_PRIMARY / ROW_BTN_DANGER`
  （`routes/(app)/admin/shared/table.ts`，内含完整 hover/tap/focus/disabled 四态）
- `ListItem` 组件（列表行，第 5 条）

**审计脚本的坑（本次 106/209 误报）**：
1. 多行 button 标签被属性值里的 `=>` 提前截断（`[^>]*` 遇 `>` 即停），class 没采到；
2. class 是常量引用（`class={ROW_BTN}`）时不含字面关键词；
3. `link-btn` / `focus-ring` / `focus-visible` 不在关键词表里。
→ 与鸿蒙端两轮结论一致：**启发式只产候选清单，合规判定必须人工开上下文**；
  且**不要拿鸿蒙的原语标准套前端**——两端的「合规」定义不同。


## 图标线宽档位约定（2026-10-14 Gemini 线宽审计沉淀）

Lucide strokeWidth 实测分布（86×1.5 / 82×1.75 / 48×2 / 13×2.5 / 3×2.2 / 2×3）。
审计结论：离群值均有「小尺寸可见性」上下文，盲目归并伤可读性；沉淀为四档约定：

| 档位 | 适用 | 例 |
|---|---|---|
| 1.5 | 大尺寸/装饰性图标（size-5+） | MomentCard 占位 MapPin 20 |
| 1.75 | 标准 UI 图标（size-4/5 列表行、导航） | 全站默认 |
| 2 | 小尺寸强调（size-3.5/4 状态类） | Toast 状态图标 |
| 2.5-3 | 极小可见性特例（size-3 选中态 Check 等） | MenuPanel 选中勾 |

**跨端说明**：harmonyapp 用系统符号（sys.symbol.*，线宽由 HarmonyOS 设计定，无
strokeWidth 参数）——两端无法字面统一线宽，以「同尺寸视觉重量相当」为准；前端
新增图标按上表选档，禁止再引入第五档。

## 替换元素绝对定位必须显式 w/h 拉满（2026-10-17 实测）

**坑**：`<img>` 等替换元素 `position:absolute` + `inset-0`（含 `inset-y-0`/`inset-x-0`）
**拉不开**——CSS2 §10.3.7：abs 定位替换元素 `width/height:auto` 时按固有尺寸渲染，
left/right 锚定不参与拉伸；`object-cover` 因此无从裁剪。表现为小图按原尺寸在卡片内
露底、大图按固有比例错裁；且大图恰好溢出时被 `overflow-hidden` 裁得「看起来正常」，
同页面只有小图露馅，极难肉眼归因。装饰 div 是非替换元素，inset 全拉锚有效——同一
原语里 div 正常、img 露底更迷惑（NearbyImageCard 线上实例：mock 200px 图在
226px 卡片内四周露黑边，computed style 的 position/inset/objectFit 全部"正确"）。

**规则**：图片优先卡片原语里，abs 定位的 img 一律显式 `h-full w-full`（横排局部列
如 `h-full w-2/5`），不靠 inset 拉伸；inset 全拉锚仅用于非替换装饰层。

**验证**：jsdom 不做布局，svelte-check 与既有单测无感——类串断言拦回归
（`NearbyImageCard.test.ts`「图片盒显式 w/h 拉满」用例），最终以浏览器实测
`img.getBoundingClientRect()` 对比卡片 rect 为准（修前 200x298-in-226x301，修后
99%w/99%h；附近的人/附近群组两页、明暗两主题截图复核通过）。

## 按钮语言两派契约（2026-10-18 审计 C1 收敛沉淀）

动作按钮只有两种允许形态，**按场景分派、同页同场景不得混用**，全部经由 `ui/button`
原语（含派生 `ui/FollowButton`），禁止绕过原语手写描边/实底动作钮：

- **派 A 页面级动作（默认）**：`rounded-md`(16) + `text-sm`，高度档 sm 32 / md 40 / lg 44
  （表单提交、弹层确认、详情动作区、移动动作条、错误页动作）。
- **派 B 行内关系 chip**：`rounded-full` + `text-xs`，`px-3.5 py-1.5`（行内）/
  `px-4 py-2`（详情宽位）（关注/加入/接受/私聊入口等嵌在内容里的小动作）。
- 圆角禁档：动作按钮禁 `rounded-sm`(12)/`rounded-lg`(20)/`rounded-xl`(24)；圆形图标钮走
  `ActionIconButton`/`size-* rounded-full`。导航项/菜单行/Toast/图片格等非动作控件按各自
  组件语言，不套本契约。
- 卡片/面板**内嵌**次级动作钮用 `flat` 去投影（投影语义归容器）。
- 完整表格与示例见 `docs/ui-components.md`「按钮语言两派契约」。

**批次 6 落地**（同页混用清零）：`+error.svelte` 三颗自写胶囊 → Button；
`users/[userId]` 动作区 rounded-sm×3/full/xl → Button（ghost 红字用
`variant="ghost" + text-status-danger hover:bg-danger-bg` 覆写）；`groups/[groupId]`
移动动作条胶囊大钮 ×3 → Button lg（卡内次级钮 flat）；`AccountSecuritySection`
rounded-sm/full → Button；`FollowButton` block 形态 rounded-xl → rounded-md
（胶囊只保留给 pill/wide）。

## 组件级档位约定（批次 3–5 审查沉淀，2026-10-18 登记）

| 约定 | 值 | 两端口径 |
|---|---|---|
| 页面级返回键 | 42px + `hit-slop`（`ActionIconButton` 承载；标准页隐藏、详情页显示） | 两端一致：鸿蒙 `DesignConstants.LAYOUT.BACK_BUTTON_SIZE = 42`（曾发现 docs 写 40，已修正） |
| 分区标题 | 14px semibold（`text-md` 档）；`eyebrow` 仅限微型分组标（11px uppercase + tracking） | 14px = 既有 `text-md` ↔ 鸿蒙 `FONT.SIZE.MD=14`，跨端刻度脚本已约束 |
| bare 空态源图标 | 24px；非 bare 圆形容器默认 64（`NeedsAuth` 64 容器为登记例外） | 两端一致：鸿蒙 `EmptyState` 用 `ICON_SIZE_MD=24` / `EMPTY_ICON_SIZE=64` |
| 横屏壳一级导航 | 固定 5 slot：首页 / 附近 / 发布(FAB) / 消息 / 我的 | Web 壳契约（鸿蒙端布局独立，不套用） |
| admin 表格移动端列裁剪 | 低优先级列 `max-sm:hidden`（600px = platform 移动/平板阈值），只留主体/关键状态/操作三类列；共享常量 `admin/shared/table.ts` 的 `TH_SECONDARY`/`TD_SECONDARY` | Web admin 专属模式（鸿蒙 admin 为卡片页） |

> 登记原则：数值档位本身走既有 token 刻度（不新增档位）；本表只登记「组件级用法
> 契约」。两端口径经源码逐项核对（DesignConstants / EmptyState / SystemPageHeader）。

## 触控热区审计（2026-10-21 实测，agent-browser 量取）

调研共识（移动端交互规范）：触控目标的有效热区 ≥ 44×44。项目用 `.hit-slop`（`::before` 外扩 6px）
实现"视觉小、热区大"，实测换算：

| 视觉尺寸 | + hit-slop | 有效热区 | 结论 |
|---|---|---|---|
| 36×36（顶栏图标按钮） | +12 | **48×48** | ✅ |
| 34×34（消息行更多操作） | +12 | **46×46** | ✅ |
| 32（Button `h-8`） | +12 | **44×44** | ✅ 刚好达标 |
| 40（`py-1.5` 文本按钮） | +12 | 52 ✅（此前未带，已补） | ✅ |
| 28（筛选 chip） | +12 | 40 ✗ | 可接受：chip 为次级筛选、行长 ≥44，且带横向间距 |

**规则**：任何可点元素的**视觉高度 < 44px 时必须带 `hit-slop`**（或 `hit-slop-y`）。新增按钮/图标按钮
先按上表换算，别只看视觉尺寸。

审计方法（agent-browser）：`set viewport 390 844` → `open <路由>` →
`eval` 遍历 `button, a[href], [role=button]` 收集 `getBoundingClientRect()`，与上表核对。
## 骨架屏 / 场景化空态 / 页头图标口径（2026-10-26 登记）

### 1) 列表首屏骨架屏（`data-display/Skeleton*`）

- **首屏加载** 高频列表/信息流一律骨架屏（`SkeletonList` / `SkeletonFeed`），禁止再用
  spinner 占位：与真实内容同栅格/同列数/同卡片外壳，做到「骨架→内容零位移」
  （首块 y/高度差 ≤2px）。
- 几何与鸿蒙 `Skeleton`/`SkeletonShimmer` 同源（不要单独改一端）：行内上下 12、前导圆
  36/40/44/48、文本条 14·55% + 11·35%、条间距 8、圆角 `rounded-sm`(12)；脉冲
  `--cst-motion-skeleton`(1500ms)，reduced-motion 下保留骨架形、停止脉冲。
- 语义：外层 `role=status + aria-busy=true + aria-label`（「加载动态中…」等）；占位块
  `aria-hidden`、不可聚焦；ListContainer 内自动补 `listitem`（复用 InlineLoading 机制）。
- **不得换骨架**：分页加载更多（LoadMoreButton/spinner）、表单/设置/admin 表格/聊天首屏
  （异构或短促，骨架行高与真实气泡差异会造成滚动跳动）。
- 新增页面接入前先看 `docs/ui-components.md`「骨架屏使用口径」。

### 2) 场景化空态装饰（`EmptyState.spotlight`）

- 只用于 **超大空态**（整页/整块内容区：访客引导、我的内容空、搜索无结果）；
  窄面板/小空态 **不得开启**（装饰反而吵）。
- 几何与鸿蒙 `EmptySpotlight` 同一 spec：光晕 320 + 外环 288 + 内环 168 + 四向星点
  4px@(±102, ±102)；颜色只取 `--cst-ambient-*`；无动画；装饰层 `aria-hidden` +
  `pointer-events:none` + `z-index:-1`（父级 `relative isolate`）。
- 规范全文见 `docs/design-system.md` 第 16.6 节；改几何/颜色必须两端同步。

### 3) 页头动作图标钮（ghost / surface）

- **品牌头**（首页 logo + wordmark）：ghost 变体（透明底裸图标，36px）；
- **标准 PageHeader**（内页）：surface 变体（浅底圆钮 + hairline，36px）——不用 ghost；
- 图片/封面上悬浮的返回/更多用 overlay 变体（平台适配，保留）；
- 触控热区：36px 视觉 + `hit-slop` = 48px（≥44）；纯图标按钮必须带 `aria-label`。
- 详细口径与鸿蒙映射见 `docs/ui-components.md`「页头动作图标钮口径」。
## Round 2 样式契约（2026-10-27）

### 1) 整卡图片卡（与 design-system §17 同源）

- 图像优先的内容卡：图片 `object-cover` 铺满整卡，信息以**顶部/底部遮罩覆盖层**压在图上；
  禁止「图片 + 白色信息托盘」两段式（横排 `horizontal` 列表卡为登记例外）。
- 卡片不设 border；圆角 `rounded-lg`(20)；hover 抬升走 `card-hover`（图片微缩放 + 卡片上浮）。
- 遮罩只用 `--cst-surface-card-scrim-top` / `--cst-surface-card-scrim-bottom` /
  `--cst-surface-card-fallback`（三主题×明暗）；禁止组件内写 rgba/gradient 字面量。
- 压图白字/白图标对比度 **≥4.5:1**（纯白底图最坏情况，字形区像素法实测）；徽章一律深色强遮罩 +
  白字，禁止浅玻璃底 + 白字；点赞/收藏等状态用填充形状 + 白描边，不只色相。
- 位置信息只在右上徽章出现一次；底部元信息只留「时间 · 标签」；多图 → 首图 + 1/N 徽章。
- 无图 fallback：主题渐变 + 0.08 主题水印图标（64px，`--cst-opacity-watermark`）+ **主题色文字**（不压白字）。

### 2) 页面级容器禁止边框（连续画布）

- 桌面顶栏/侧栏/主内容、移动/平板顶栏/PageHeader 不得有包裹整页的 border/圆角/白底/硬分隔线；
  层级靠留白、排版、交互态与**内容单元卡片**表达；内容单元卡片可保留但收敛重量。
- PageHeader 静止无底线，滚动才允许柔和投影；氛围画布桌面整窗单实例（无区域接缝）。
- 自查：截图四边不得出现「丑竖线/横线」；grep 外壳组件残留的 `border-b` / `rounded-lg border` 包裹体。

### 3) 选中态不得只依赖颜色（design-system §18）

- `segmented`：accent 实底 + ✓ icon + `font-semibold`；`pills`：`font-semibold`；
  `underline`：`font-semibold` + 下划线。
- 验收必须附 `filter: grayscale(1)` 灰度截图，选中项仍一眼可辨；新增 Tab/分段控件按此执行。
## Round 3 内容岛与安静按钮（2026-10-27）

### 1) 内容岛（开放页面内容分组，强制）

- 形态：`bg-surface-island + shadow-xs + rounded-lg(20)`，**无 1px 描边**；`p-2`(8)、行背景透明、
  行间 `gap-y-0.5`(2)、hover 填充离岛边 8；**不加 1px 行横线**。
- 开放页面的列表集合只用 `ListContainer bordered`（已改为软岛语义）；调用方不得覆写 border/bg/radius，
  也不得在外层再包手写卡片（双框）。
- 例外（登记）：弹层/Sheet、hero/封面预览、单帖卡、聊天输入条、表格/统计页。

### 2) 同级按钮不混用

- 主操作 = accent 实底；次级/安静 = `bg-surface-subtle/60` 软填底、无描边、hover 全量；
  危险 = soft danger 底 + danger 文字；同一页/同一组动作不得混用「描边胶囊」和「软填胶囊」。
- 功能边界（输入框、主题 swatch、搜索框）保留，不属「页面框」。

### 3) 页面级硬线

- 页头/分区标题下不得有 1px 硬线（用留白 + eyebrow）；只允许功能线：聊天输入条/粘底条、
  消息分栏竖向发丝线、表格/统计行、内容单元内演示分隔。
- 自查：`grep -rn 'border-b' frontend/src/components frontend/src/routes | grep -vE '功能线白名单'`。
## Round 4 长尾/后台与交互状态（2026-10-27）

### 1) 长尾/后台软岛

- 页面级自定义卡（单帖卡、群详情 hero/指标/操作、他人主页、首页 onboarding/统计/轮播、admin 模块卡与表格外壳）
  一律按 design-system §19 软岛化；例外按 §19.5（表格行线/表头线、输入控件、媒体容器、hero 封面、内容分隔线、
  chip/badge、头像环、弹层、地图面板等保留其功能边界）。
- 单帖卡=内容岛；媒体容器去 1px 描边但保留圆角/阴影/focus-ring；表格外壳去描边但表头/行线/行 hover 保留。

### 2) 焦点可见（强制）

- 所有 interactive 必须 `focus-ring`（2px accent + offset 2）；键盘 Tab 可达、skip link 首 Tab 可见；
  **去描边不得丢焦点环**；父容器 overflow 不得裁切；亮/暗都测（键盘截图 + computed outline）。

### 3) 减少动态（强制）

- 全局 `@media (prefers-reduced-motion: reduce)` 覆盖位移/脉冲/shimmer/悬停过渡；透明度淡入可留；
  新增动画不得裸写 duration/iteration；默认行为不得回归。

### 4) 加载/空/错误态

- 首屏列表/卡优先 Skeleton；单实体/短列表可 InlineLoading（页面注释给理由）；
  EmptyState / LoadError+重试 / ListEnd 四态不得互替。
## Round 5 质量门（可访问性/触控/主题/性能，2026-10-27）

### 1) axe 可访问性基线（0 serious / 0 critical）

- 整卡可点必须用「覆盖按钮 + 行内动作 auto-lift」，卡片根不得持 button/aria-label（nested-interactive）。
- `role=list` 子节点补 `role=listitem`；listitem 不挂 aria-label；`dl` 只包 dt/dd；无 role 的 div 不挂 aria-label。
- 每页有 `main`；可滚动非键盘可达容器加 `tabindex=0` + `role=region` + `aria-label`；装饰图 alt=""+aria-hidden。
- 状态徽章用实底 + on-accent 字；用 axe-core（`--init-script`）扫描，新的可点结构先跑 axe 再加。

### 2) 触控目标

- 移动端有效热区 ≥44×44（`hit-slop`/`hit-slop-44`）；横向相邻元素不得互相覆盖；
  例外：表单控件、横向 chip/筛选 pill、内联链接、地图 marker/工具条、skip link。

### 3) 主题与性能

- 三主题×明暗关键页必测；软岛/画布 ratio 与文字对比记录基线；状态徽章避免低对比淡底。
- 性能基线：关键页 LCP/CLS 采样（本地 dev 环境仅作相对参考）；CLS 应 <0.1，图片显式尺寸/懒加载/骨架零位移。
## Round 6 图片加载与键盘导航（2026-10-27）

### 1) 图片必须走统一管线

- 所有 `<img>` 用 `lib/images.ts`：`presetImageUrl/presetSrcSet`（avatar/thumb/card/cover/hero/full）+ 调用方 `sizes`；
  仅 unsplash 追加 `auto=format&fit=crop&q&w`，其他域名/本地/blob/地图瓦片原样；禁止裸写 `<img src>`。
- 首屏仅可见图 `loading=eager` + `fetchpriority=high`（feed 前 2/详情首图/hero/ImageViewer）；其余 `lazy` + `decoding=async`。
- 占位（主题渐变/骨架/占位底）+ 0.22s 渐显；reduced-motion 直接显示；不得引入布局位移；失败走 fallback（渐变/首字母/图标）。
- 清晰度：按显示尺寸 2x/3x 取档；Retina 用 2x srcset；真机偏软再升档（不要用低清拉伸）。

### 2) 键盘导航

- 列表/搜索页可加 ↑↓ 焦点移动（落到真实链接/按钮、focus-ring 可见），Enter 原生打开、Esc 清空/返回；
  不引入 combobox/aria-activedescendant；Tab 顺序不变。新增交互后必须跑 axe（0 serious/critical）。
## Round 7 动效与状态反馈（2026-10-27）

### 1) 动效必须走统一入口

- `lib/motion.ts`：`MOTION.enter 180 / exit 150 / pop 220`、`prefersReducedMotion()`、`motionDuration()`、`pop` action、`playPageEnter()`、`toastExit`；
  CSS 用 `cst-page-enter` / `cst-state-pop` / `cst-comment-enter`。禁止手写新 duration/scale 数字。
- 页面进场只允许客户端导航（`afterNavigate` 且 `nav.type !== 'enter'`）；**首屏不播**（护 LCP）；只包内容层，不动 sticky/滚动/焦点。
- 骨架→内容只在「实际展示过骨架」时 fade；缓存命中不播；不得逐项 stagger 大列表。
- 状态 pop 只在用户激活操作触发（点击 nonce/显式 handler）；初始数据同步/首帧已激活不播，取消不 pop。
- toast 退场用 `out:toastExit`，容器常驻（最后一条可退）；空 aria-live 无内容。

### 2) reduced-motion（强制）

- CSS 动画由全局 `@media (prefers-reduced-motion: reduce)` 压到 0.01ms；**JS 动画（Svelte transition/class 定时器）必须显式判断**，
  transition duration 用 `motionDuration(ms)`（reduced 返回 0），否则 reduced 下仍会播。验收：无动画类/事件/定时器残留。

### 3) 红线与验收

- 不得动画首屏 LCP 元素；不得引入 CLS；QA 截图不得因动效抖动（动画 ≤260ms；动效 QA 建议在 reduced-motion 下取基线）。
- 验收：normal/reduced 双路径 computed + 中间态截图；axe 0；CLS/LCP 不回归；键盘/焦点无回归。
