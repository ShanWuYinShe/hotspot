# UI 一致性改进交接（2026-10-22 ~ 2026-10-27，Round 1-14）

本文件记录 UI 一致性/审美改进的完成基线与待做项，供后续会话直接接手。规则细节见
`.agents/skills/hotspot-harmony-ui/SKILL.md`、`.agents/skills/hotspot-frontend-layout/SKILL.md`；
审计结论见 `docs/frontend-ui-audit.md`、`docs/backend-audit.md`。

## 已完成基线（不要回退）

### 2026-10-27 本轮（Round 14，次级页面内容软岛化 + 个人中心域精致化）

- **内容软岛（双端同 spec）**：开放页面内容分组统一为「无 1px 描边 + `surfaceIsland` 表面 +
  `shadow-xs` + 圆角 20 + 行透明/gap 2 + hover 离边 8」；新增 `--cst-surface-island`
  （light = `surface-primary 86%` 半透合成画布；dark = `surface-primary 94% + text-primary 6%`；6 组值），
  `ListContainer bordered` 语义更新为软岛 → 设置/通知/关系/搜索/我的/举报等全站列表生效。
  暗色岛/画布亮度比 1.16–1.33（修复前 1.01–1.07 糊平）。spec 见 `docs/design-system.md` §19。
- **页面级硬线清零**：/me 自定义页头线、他人主页分区头线、群详情分区线、搜索页头/sticky 线、
  reports 行线全部删除；保留功能线白名单（聊天输入条/粘底条、消息分栏发丝线、表格/统计行、
  内容单元内演示分隔）。
- **安静按钮语言**：次级动作统一 `bg-surface-subtle/60` 软填无描边（hover 全量）；主操作 accent 实底；
  危险 soft danger；同级动作不再混用描边胶囊与软填胶囊。
- **六页打磨**：/me、/me/edit（表单软岛+danger 软底）、/settings、/notify、/me/relationships、/search
  （含 N1 搜索骨架 64px+分组头、N2 消息骨架在线条 92px 占位，骨架→内容零位移）。
- **鸿蒙镜像**：`surfaceIsland`（6/6 parity PASS）、`ListGroup`/`SystemCard`/`SettingsSectionCard` 软岛、
  Profile 身份卡去框、stats capsule、通知容器式软岛+未读态、搜索页头/分区线清零、骨架同步；
  build TYPE CHECK + BUILD SUCCESSFUL、check:theme 全绿。
- **验证**：web check 0 error / test 518 全过 / build 通过 / browser QA 23 页 issues=[] / 6 页×3 视口×亮暗
  before-after 36×2 张；暗色岛 ratio 1.16–1.33、岛内文字对比度 ≥4.5。**鸿蒙真机仍离线**：
  软岛阴影/圆角/gap/按钮 hover/未读态/骨架像素待真机（Round 13+14 两轮端上像素验收均待补）。

### 2026-10-27 上一轮（Round 13，用户反馈：整卡图片卡 / 去页面级边框 / 选中态无障碍）

- **整卡图片卡（双端同 spec）**：图片铺满整卡 + 顶部 96px/底部 62%（feed 58%）遮罩覆盖信息；
  feed 4:5、网格 3:4、圆角 20、无 border；顶部作者/徽章、底部标题 2 行+「时间·标签」+ 白图标操作行；
  多图首图 + 1/N；无图 fallback = 主题渐变 + 0.08 水印 + 主题色文字；覆盖态按钮 `onMedia`
  （深色强遮罩胶囊+白字+白描边）。压图文字纯白底图最坏情况字形区像素实测全部 ≥4.5:1
  （最低 4.81，feed 标题 10.70/12.14）。spec 见 `docs/design-system.md` §17。
  鸿蒙：`MediaOverlay` 原语 + `MomentFeedCard variant='overlay'` + `NearbyImageCard mediaOverlay`，
  scrim/fallback 与 web 6/6 逐值一致。
- **去页面级边框（连续画布终版）**：桌面顶栏/侧栏/主内容去掉白浮卡 border/圆角/白底/投影，
  共享整窗画布 + 氛围单实例（无接缝）；PageHeader 去硬底边线（半透明画布+blur，滚动才投影）；
  移动品牌头/聊天头/在线条/我的瞬间分区线/群详情概览白卡全部收敛。判定口径：页面级框去、
  内容单元（设置分组卡/单帖卡/rail）留，逐处结论见 `docs/frontend-ui-audit.md` 与 fe report §3。
- **选中态不依赖颜色**：`TabBar` / 鸿蒙 `TabsPillGroup` 三形态——segmented = accent 实底 + ✓ + semibold；
  pills = semibold；underline = semibold + 下划线；浏览器 `filter: grayscale(1)` 实拍验证可辨。
  spec 见 `docs/design-system.md` §18。
- **骨架屏同步整卡形状**：`SkeletonFeed` feed 4:5 / card 4:3 / portrait 3:4 + 顶部作者行/底部文本条；
  首页骨架首卡与真实首卡 rect 完全一致（334×418 @28,115，零位移）。
- **验收证据**：web 10 核心页 ×5 视口 ×亮暗 = 100 张矩阵（0 console error/0 横向溢出）；
  check 0 error、test 100 files/518 passed、build 通过、browser-qa 23 页 issues 0。
  **鸿蒙真机本轮不可用**（hdc 离线；模拟器截图黑屏、componentSnapshot 离屏取证失败，均有复现记录），
  已用 assembleApp 构建 + 布局树 + token 一致性脚本取证；真机像素验收 checklist 见 `ha/report.md` §7，
  待设备恢复后补做（本轮不能算「已完成端上验收」）。

### 2026-10-26 本轮（Round 12，双端 UI 改进 + 跨端对齐）

- **列表首屏骨架屏（web 新增，共享项）**：`Skeleton` / `SkeletonList` / `SkeletonFeed` 三原语 +
  `.skeleton-block` + `--cst-motion-skeleton`(1500ms)；接入 12 个高频首屏（首页 feed、附近枢纽+三叶、
  消息、通知、搜索、我的瞬间/群组、关系、瞬间详情含评论区）。几何与鸿蒙 `Skeleton`/`SkeletonShimmer`
  同源（行内 12 / 文本条 14·55% + 11·35% / 条间距 8 / 圆角 12），`role=status + aria-busy`、
  占位块 `aria-hidden`、reduced-motion 停用脉冲。分页/表单/admin/聊天首屏保留 spinner（理由见 `docs/ui-components.md`）。
- **场景化空态 spotlight（双端同 spec，共享项）**：`EmptyState.spotlight`（web 22 个超大空态；
  鸿蒙访客态/我的内容空/搜索无结果）。几何：光晕 320 + 外环 288 + 内环 168 + 四向星点 4px@±102，
  纯 ambient token、无动画、装饰层不进读屏/不拦截触摸。spec 见 `docs/design-system.md` 第 16.6 节。
- **鸿蒙端氛围画布同步（Round 11 遗留，已完成）**：`AmbientBackdrop.ets`（单 Canvas 静态绘制）
  + `ThemeConstants.AMBIENT`（6 组合 × 12 色，与 web `--cst-ambient-*` 语义/合成口径一一对应），
  统一挂载 `PageShellContainer`；三主题 × 明暗真机 6/6 像素校验通过。平台适配差异（静态无动效、
  每页一块画布）已登记，见 `docs/design-system.md` 第 16.4/16.5 节。
- **页头动作图标跨端口径（Lead 决策）**：品牌头（首页）= ghost（前端 `variant="ghost"` / 鸿蒙
  `kind:'neutral'`），标准 PageHeader（内页）= surface（前端默认 / 鸿蒙 `kind:'surface'`，
  `SystemPageHeader.IconAction` 已扩展）；web 移动端首页仅「搜索 + 通知」（地图入口保留在附近域页头），
  鸿蒙同步移除首页地图钮。
- **超宽屏对齐修复（web）**：`/settings` 页头与正文同 56rem 居中（2560 左边缘 Δ605→0）；
  `/search` ≥1792 改两段式（发现区 4 分栏 + 热门动态整行 4 卡，消除左下 500px 空白）；
  桌面首页补 h1（0→1）、桌面分栏聊天去重复 h1（2→1）。
- **鸿蒙搜索页三态修复**：输入即搜（消除「只有 tab 行、下方全空」中间态）+ Skeleton 加载态 +
  无结果 spotlight 空态；用户结果行重写为 web 同构紧凑行（40 头像 + 昵称 + `@user_id · tag` + 箭头，68vp）。
- **鸿蒙区块标题链接热区**：「查看全部 / 探索群组 / 更多动态」等 `Row.onClick` 通过 `responseRegion`
  扩到 ≥44vp（原 <44 的登记 backlog 已关闭）。
- **验收证据**：web 39 页改后同口径巡检 0 回归（溢出 0/坏图 0/h1 全 1/console 0）；
  骨架真实加载态（`aria-busy=1`、reduced-motion `animation-duration≈0`）实测；鸿蒙真机 Mate60
  `assembleApp` TYPE CHECK/BUILD SUCCESSFUL + 三主题 × 明暗截图；`bun run check/test/build` 全绿。

### 早期基线

- **状态组件统一为无卡形态**：两端 `EmptyState`（鸿蒙 55 处、前端 51 处）全部无卡片，
  `SystemLoadingView` 默认无卡（`card` 选项留给浮层）；硬编码状态文案 0 处带卡片。
- **鸿蒙间距体系归一**：纵向 46 处、横向 13 个文件，离群档归零；规则 **12 / 16 / 24** 三档。
  前端三档**只约束页面级/区块级**，组件内部沿用 Tailwind 标准刻度（`py-2.5` 等 73 处不要动）。
- **前端页面级纵向节奏归一（Round 9）**：`routes/(app)/**` 逐页归一完成（21 个源码文件、2 个 skill + 本文件），
  落地口径见 `hotspot-frontend-layout` skill「纵向节奏 → 前端落地口径」。要点：
  - 页面根容器 `gap-4`(16)；「附近」枢纽页例外 `gap-6`(24)（根容器直接承载多个带标题分区）；
  - 区块之间 24（`gap-6`/`mb-6`）、区块标题→内容 12（`mb-3`）、区块内元素 16；
  - `contentClassName` 收敛白名单（`pt-4 pb-6` / 长表单 `pt-4 pb-10` / 封面 `pb-6` /
    总览 `pt-6 pb-8` / 移动端固定操作条条件式 `pb-28`）；
  - 消除所有「父 gap + 子 mt/mb」双倍叠加（如 `pt-6`+`mt-6`=48、`mb-5`+`mt-5`=40）。
  验证：svelte-check 0 error / ESLint 0 问题 / vitest 499 通过 / `bun run build` 通过；
  agent-browser 改前改后 64 对（16 路由 × 桌面/移动 × 亮/暗）几何+像素对比：水平位移 0、
  横向溢出 0、坏图 0、h1=1、控制台错误 0，13 项预期间距变化逐项命中。
- **鸿蒙登录「账号/密码串位」复验（Round 10，端到端通过）**：模拟器实测 `uitest_admin / UiTest#12345`
  一次登录成功——提交前账号/密码正确；提交后 hilog `POST /public/v1/auth/signin → code=0`、
  后端 `signin 200`、Toast「欢迎！」、进入消息列表；**无**账号框改写、**无**密码策略错。
  - **根因更正（重要）**：上一轮观察到的「提交后账号框变密码值」主要是**自动化坐标过期伪影**，
    不能据此判定 app bug——`uitest uiInput inputText <x> <y> <text>` 会先按坐标点一下，
    键盘弹出后窗口上移，旧坐标落到安全键盘上先敲进一个字符（实测输入 `abc` 出现 4 个掩码；
    改用键盘弹出后的实时坐标则正好 3 个）。正确用法与自证方法已写入
    `hotspot-harmony-device-verify` skill；常驻错误槽位修复（`d39e1d7f`）保留为健壮性修复。
- **groups 详情 Tab 内容顶距统一（Round 10）**：成员 / 申请审批 / 邀请管理 / 封禁名单子 Tab 的
  内容顶距统一为 **24**（含空态；成员列表无「添加成员」行时由列表自身承担 24），
  子 Tab 栏距主 Tab 栏也改为 24——同页 tab→内容只有一种间距。
- **鸿蒙瞬间详情评论区跨端对齐（Round 10）**：评论分区顶距 16→**24**（对齐前端 `mt-6`）、
  分区标题 12/medium→**14/semibold**（对齐前端 `text-md` + semibold）；端上实测正文→评论标题
  间距 61px→93px（+8vp，密度 3.917），标题行高 67px→82px。
- **状态页垂直位置（Round 10 产品决策）**：维持**页面内居中**，不改「靠上 1/3」。
- **氛围画布（Round 11，页面「不空荡」改造）**：内容稀疏/空态页补主题化世界感背景，
  唯一实现 `components/layout/ambient/AmbientBackdrop.svelte`，挂在四端外壳内容区
  （桌面/平板/移动竖屏/移动横屏，固定不随路由滚动；`PageViewport` 底色透明以透出装饰层）、
  登录页、错误页、启动占位（与 BootSplash 组合成游戏加载画面观感）；桌面顶栏/侧栏使用
  `variant="chrome"` 轻量天光版（只铺天光不画世界层），整窗共享同一环境光。
  四层结构：天光/地平线光源 → 地平线雾 → 主题世界世界（vibrant 星尘+星座+轨道环+城市天际线 /
  forest 等高线+暖阳+树林剪影 / inkwash 墨山+朱砂圆相+飞白墨点）→ 暗角；
  装饰层 `z-index:-1`、`aria-hidden`、`pointer-events:none`，不引入第二整层底色。
  12 个 `--cst-ambient-*` token（三主题 × 明暗，随主题/明暗自动适配），
  超长循环动效 24s/36s 且 reduced-motion 停用；背景常驻形成纵深感。
  规范见 `docs/design-system.md` 第 16 节；调浓淡只改 token alpha，无需改组件。
- **鸿蒙首页下拉刷新「鬼畜晃动」第四轮终极定位与根治（2026-10-23 用户反馈）**：
  端上逐帧抓包量化（uinput 拖动持续按压采样），定位到核心根因：
  ① **`pullToRefresh: true` 默认行为导致的拉伸与吸附对抗**：ArkUI API 12+ 的 `pullToRefresh` 默认值为 `true`，
  用户下拉只要超过 `refreshOffset`(64vp)，手指根本未松开系统便立即启动刷新并执行吸附回弹动画；用户的触控向下拉伸
  与系统的吸附回弹动画在同一时刻发生激烈冲突（下拉→吸附弹回→下拉→吸附弹回），形成高频上下振荡（鬼畜晃动）。
  显式配置 `.pullToRefresh(false)` 恢复标准松手才刷新契约，彻底杜绝触摸期间的自动吸附；
  ② **双重弹性手势竞争**：内层 `List` 默认自带 `EdgeEffect.Spring`，与外层 `Refresh` 的弹性下拉手势重叠抢夺，
  在 `List` 上显式设置 `.edgeEffect(EdgeEffect.None)` 解决；
  ③ **指示器断层跳变**：此前单传 `refreshingContent` 缺省 `builder`，导致下拉阶段显示系统原生指示器、刷新态突变为
  56vp 自定义内容，临界点两套布局交替挂载；统一回归 `builder: RefreshIndicator`；
  ④ **行级 `onTouch` 污染页面状态**：ListItem 移除手动 `onTouch`，改用内部内聚的 `Pressable` 原语。
  复测：连续截图位移序列完全单调（504→536→580→620→656），松手回落平滑稳定（470, 538），点击跳转与返回正常。
- **鸿蒙首页下拉刷新抖动第三轮（2026-10-23）**：用户反馈前两版仍有抖动。端上 60fps
  屏幕录制逐帧分析（内容区行投影互相关 + 指示器亮行重心）定位：抖动集中在刷新指示器与
  列表顶部，根因是 Refresh 用 builder 时刷新过程反复重建自定义组件（ArkUI 文档建议
  API 12+ 用 refreshingContent），且 builder 与 refreshingContent 并存会让刷新区高度在
  两套布局间跳变。修复：刷新态改用 `refreshingContent`（ComponentContent 稳定复用）、
  移除 builder、Refresh 直接子组件改为 List/Scroll（不再套固定高度 Column）。同时保留
  前两轮修复（List 行布局、去掉 onOffsetChange 逐帧状态）。契约写入 harmony-ui skill。
- **鸿蒙首页下拉刷新「鬼畜抖动」修复（2026-10-23 用户二次反馈）**：上一版为做指示器
  渐显在 `onOffsetChange` 里逐帧写 `@State`，触发「状态更新 → 整页重建 → offset 重算」
  反馈环；录屏逐帧量化显示下拉时内容每 50ms 交替位移 100~200px（上下抖动）。修复：
  移除逐帧状态与 onOffsetChange，指示器改为无状态两态（下拉只 spinner / 刷新态补文案，
  刷新区过小时不显示文字，不再半截裁切）。复测：连续截图的内容位移序列单调、无交替。
  坑写入 `hotspot-harmony-ui` skill「ArkUI 布局陷阱 3」。
- **鸿蒙首页下拉刷新 UI 修复（2026-10-22 用户反馈）**：端上录屏显示下拉刷新时
  spinner 漂在空白区中间、刷新态把内容推下约 1/3 屏、列表行出现错位/文字残影。
  根因：① 首页 Refresh 内容区是 `Scroll{Column}` 且外层强制 100% 高，高度连续变化时
  动态 ForEach 重排失稳；② 自定义 `RefreshIndicator` 未锁高度，在随下拉增高的刷新区里
  垂直居中。修复：列表改 `List`（行距仍由行 padding 承担，保持原节奏）+
  指示器锁 52vp 高（顶端锚定）。端上截图复核：刷新态指示器贴页头、无大空白、行无错位。
  契约写入 `hotspot-harmony-ui` skill「ArkUI 布局陷阱 3」。
- **桌面侧栏「附近」拆分（2026-10-22 用户审查）**：桌面侧栏由单项「附近」改为
  附近的人 / 附近的群组 / 附近瞬间 三个直达入口（`nearby-people/groups/moments`），
  三个叶子各自高亮；枢纽页 `/nearby` 不再占侧栏项、不点亮任何项（仍可从首页
  「看看附近」进入）。移动/平板底栏 IA 不变（仍落枢纽页、统一高亮「附近」）。
  鸿蒙桌面侧栏本就有三项（附近瞬间/附近的人/附近的群组），两端桌面 IA 由此对齐；
  2026-10-22 用户确认全站统一用「瞬间」，web 侧栏第三项随之为「附近瞬间」。
- **桌面侧栏导航 hover 修复（2026-10-22）**：chrome 氛围天光给侧栏叠了 accent 淡染，
  原 hover（accent 7% 混底）与静息底同色系、几乎不可辨。改为中性「正文色 9% 混底」
  （`--cst-surface-navigation-hover`），hover 时图标/文字同步升到 `text-primary`；
  实测 hover 底色差：浅色 ≈15 RGB（改前 8）、深色 ≈13 RGB（改前 3-5）、
  inkwash 浅色 ≈15 RGB（改前 4-9）。
- **卡片圆角跨端（Round 10 核对）**：交接文档原记「鸿蒙 20 / 前端 12」**不成立**——两端普通卡片
  都是 20（鸿蒙 `SHAPE.CARD=RADIUS.LG` / 前端 `rounded-lg`），紧凑卡与列表容器都是 16
  （`COMPACT_CARD` / `rounded-md`），12 是图标块/小控件档（`ICON_TILE` / `rounded-sm`）。
  两端已同口径，无需改动；跨端刻度脚本持续守护。
- **首页**：错误态/空态无卡；顶栏三图标 `neutral → surface`（对齐 web `.action-icon-btn`）。
- **登录页**：分段控件激活胶囊填满分段槽（44vp）。
- **「附近」**：错误态补重试按钮。
- **开发接口地址**：`127.0.0.1` → `10.0.2.2`（模拟器可直连宿主，端上验证链路打通）。
- **编辑资料**：编辑期间隐藏底部 tab 栏（防丢草稿）。

## 待做（按优先级）

1. **鸿蒙真机像素验收（Round 13+14 遗留，最高优先）**：两轮鸿蒙改动均只完成构建/布局树/token 校验，
   真机 hdc 离线导致三主题×明暗矩阵、灰度 Tab、压图对比度像素实测、软岛阴影/圆角/gap、按钮 hover、
   未读态、骨架零位移、web↔鸿蒙并排 compare 全部挂起。设备恢复后按
   `.workwork/ui-polish-1026/r2/ha/report.md` §7 与 `.workwork/ui-polish-1026/r3/ha/report.md` checklist 执行
   （含 `ha/run_after_r2.sh` 与 r3 脚本）。
2. **长尾自定义卡收敛（Round 14 遗留）**：`ListContainer` 场景已随软岛组件全站生效；剩余 admin 统计/表单
   自定义卡、moment 详情单帖卡、首页新手引导卡等仍走旧描边语言，按「内容单元岛化 / 登记例外」逐页判定
   （design-system §19；不要一刀切）。
3. **鸿蒙氛围画布实例数/内存优化（Round 12 遗留，端上实测数据已量化）**：当前每个
   `fillHeight` 页面一块全屏 Canvas（Tab 页常驻，实测 live 2→5 实例、PSS 218.7→330.2MB 含页面数据，
   单画布理论 ≈12MB）。可选方案：上提到外壳内容区单实例（对齐 web）或共享绘制缓存/图片快照；
   修改前先量首帧与滚动帧率，勿只按内存数字取舍。
2. **氛围浓淡评审**：世界层透明度集中在 `themes/*.css` 的 `--cst-ambient-*`（12 个），
   产品/用户若觉得过淡或过浓，改 token alpha 即可，无需动组件；鸿蒙端需按同口径重算
   `ThemeConstants.AMBIENT` 的 ARGB。
3. **鸿蒙骨架屏接入扩展（低优先）**：通知/搜索/关系等列表目前未统一接 Skeleton 系（列表骨架
   原语已有），可按 `docs/ui-components.md`「骨架屏使用口径」逐页补，几何直接复用。
4. **搜索 Tab 计数格式对齐（低优先，待确认）**：web TabBar count 以独立小字呈现（如「用户 7」），
   鸿蒙搜索页当前为「用户 (10)」括号形式；如判定为不一致，鸿蒙按 web 口径改（需两端截图确认）。
5. **搜索发现态 spotlight（待设计确认）**：web 搜索发现态未加 spotlight，鸿蒙本轮未单端先行
   （符合「不分别改两端」约束）；若设计认为需要，两端同一轮一起加。
6. **消息页头 trailing 既有差异（待确认）**：鸿蒙消息页头 trailing 与 web 移动端存在既有差异，
   本轮未动（ha report §8 记录）；需要时单独立项对齐。
7. **BSP 瀑布墙骨架近似（已接受）**：附近瞬间/群组桌面瀑布墙无 1:1 骨架，用同列数卡片栅格近似，
   内容到达有轻微重排；如需消除需按布局算法生成骨架，收益低，保留。
8. **骨架屏零位移 N1/N2（2026-10-27 Round 14 已修）**：搜索骨架行高对齐 64 + 分组头占位；消息骨架补在线条
   92px 占位；两端已镜像，复核确认后关闭。
9. **鸿蒙 AMBIENT / CARD-SCRIM token 纳入 CI（复核 N5 + Round 13，防漂移）**：`check-cross-platform-colors.mjs`
   只覆盖 14 组 palette；AMBIENT 12 项 ×6 组合（72/72）与 CARD-SCRIM/FALLBACK/WATERMARK（6/6）目前只有人工脚本
   校验（`review/lib/check-ambient-parity.py`、`ha/logs/scrim-parity.txt`），建议并入 `bun run check:theme`。
10. **鸿蒙装饰几何常量 token 化（复核 N6，低优先）**：`EmptySpotlight.ets` / `AmbientBackdrop.ets` 的 320/288/168/4/102
   与透明色 `'#00000000'` 属已登记布局几何例外，可后续登记到 design-system 或抽常量；透明色建议用 `DesignConstants.COLOR.TRANSPARENT`。
11. **鸿蒙设置页/管理后台无氛围层（复核 N3，已登记平台差异）**：`PageShellContainer`（`fillHeight=false`，
   位于外页 Scroll）不挂画布，外层目前也未挂；与 web 全站有氛围层不一致。后续对齐应在**外壳/外层页面**挂单实例，
   禁止在滚动容器内挂（会随滚动拉伸）。
12. **跨端搜索结果行字号/头像档既有差异（复核 N8，非本轮引入）**：web 行高 64/头像 36/名称 12；
   鸿蒙 68/头像 40/名称 14。属两端 avatar/字号档映射既有差异，后续统一时两端同一轮改。
13. **鸿蒙搜索 Tab 计数（复核 N9，既有 IA）**：鸿蒙搜索 Tab 不显示计数，web 显示（如「用户 7」）；
   信息密度差异，记录备查。
14. **网格卡字号例外（Round 13）**：171px 宽网格卡正文 12 / 作者 14（feed 标题 16 合规），避免 16px 两行
   挤压操作行；若后续放大网格卡宽度可统一。
15. **BSP 卡片比例例外（Round 13）**：附近瞬间/群组桌面 BSP 格按布局单元铺满、非固定比例；如需统一需重排布局。
16. **横排紧凑卡 fallback/精选横排例外（Round 13）**：horizontal 卡仍两段式与旧占位；群详情精选桌面横排保留。
17. **forest 亮色软岛与画布同色（Round 14 遗留）**：light `surfaceIsland` 86% 合成后 forest 与画布同色
   （ratio 1.00），边界仅靠 `shadow-xs`；若产品觉得需要拉开表面差，可改 light 岛为 `primary 92% + text 8%`
   （会略变灰，需两端同改并回归主题校验）。
18. **鸿蒙我的瞬间 tabs 缺失（Round 13）**：web 有 发布/喜欢/收藏/共 N 条记录 tabs，鸿蒙因无喜欢/收藏数据源、
   MomentPage 现取 fetchNearbyMoments（既有数据源问题）未实现；需数据层补齐后再对齐 UI。

## 遗留观察（已评估，保留现状；非缺陷）

- **`ContextRail` rail 分区间距 16**（共享原语 `gap:1rem`）：rail 是紧凑侧栏，16 为有意设计；
  若改 24 会同时影响 notify/users/moments 三处 rail，收益低，保留。
- **`PageHeader` 自身上下内距**（有 below：16/14；无 below：18/18）属页头组件内部构图，
  不属于页面级三档；若要求页头严格 12/16/24 需单独立项评估全站页头高度，保留。
- **notify 极端态**：已读为空、仅未读列表时，列表 `mb-6` + `ListEnd` 自带 `py-6` = 48 收尾留白；
  只在极端组合出现，属状态组件内部节奏，保留。
- **settings 2560 正文列宽收窄（2026-10-26，预期修复的副作用）**：为让页头与正文同 56rem 居中，
  2560 下正文宽度 1024→896；1440 视觉更整。若产品认为宽屏信息密度下降，可改为「页头与正文同时加宽到同一值」
  而非各页单独 `contentMaxWidth`。
- **search ≥1792 两段式布局（2026-10-26）**：新布局仅超宽屏生效（1440/390 像素不变），
  需产品/用户最终确认信息密度。
- **我的瞬间页数据源显示附近流（既有问题，非本轮引入）**：ha report §8 记录，属数据层契约问题，单独立项。
- **本轮已顺手修（复核 N12）**：鸿蒙 `EmptyState.ets` 空态图标圆的 `.border()` 写在 `.borderRadius()` 之后
  （ArkUI 陷阱会把圆角重置为 0）→ 已改为先 border 后 borderRadius；同类写法（如有）下轮批量扫。
- **复核留档**：`review/verdict.md`（独立复核结论 A–H、13 Notes）、`review/checklist.md`、
  `review/baseline-web/` 与 `review/after-web/`（改前/改后 39 页同口径）、`review/metrics/compare.md`。

## 端上验证环境要点

- 模拟器直启：`/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator -start "Pura 90 Pro"`
  （约 10s 就绪，`hdc list targets` 出现 `127.0.0.1:5555`）。
- **真机与模拟器同时在线时**必须 `hdc -t 127.0.0.1:5555 install ...` 指定设备，否则报
  `need connect-key`。
- 模拟器装**仓库构建**即可正常加载数据（`DEV_SERVER=http://10.0.2.2:8888`），无需临时改常量。
- 想验证错误/空态：`docker compose -f docker-compose.yml -f docker-compose.dev.yml stop backend`
  → 各页进入错误态 → 验证完 `start backend`。
