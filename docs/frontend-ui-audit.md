# 前端 / 鸿蒙 UI 审计基线（2026-10-21）

本文件记录**已实测通过**的 UI 质量项与逐页结论。**这些项已确认达标，不必每次改动都重审**；
新增页面或大改布局时，用对应方法复测一节即可。方法全部基于 `agent-browser`（前端）与
`hdc uitest`（鸿蒙），不使用自研浏览器管线。

## 一、可测质量项（已全部通过）

| 项 | 方法 | 结果（2026-10-21） |
|---|---|---|
| 宽屏内容填充 | 视口 1397×750 / 1920×1080，量最外层内容跨度 ÷ 可用宽度 | 1397：全站 0.96；1920：抽查 10 路由全 **0.97**，无横向滚动 |
| 内容卡留白对称 | 量最宽内容卡的左右留白差（`abs(left-right)`） | 全站 **skew 0**；修复过 `/me/relationships`（180/180）与 `/admin/{config,stats,rbac}`（52/52） |
| 深色主题对比度 | WCAG 相对亮度公式 | 正文 14.21:1 / 次级 7.84:1 / 强调 6.66:1 / 警告 9.68:1 —— 全部 ≥ AA |
| 深色层级表达 | 查表面 token | 页面 `#0B1424` → 卡片 `#152134`（用表面提亮表达 elevation，符合深色模式实践）|
| 动效一致性 | grep 硬编码 transition 时长 | **0 处**硬编码，全部走 `--cst-motion-*` |
| 触控热区 | 视口 390×844 遍历可点元素 | `.hit-slop` 外扩 6px：36→48、34→46、32→44（全部 ≥44）；28px chip 为 40（可接受例外）|
| 键盘焦点可见 | 按 Tab 6 次读 `activeElement` 的 outline | 6/6 均有 **outline solid 2px** |
| 空态完整性 | 检查 EmptyState 调用 | 组件四要素（icon/title/description/action）；未传 icon 时渲染默认 Inbox（2026-10-21 加）|
| 动效/骨架屏 | 端上观察 | 鸿蒙列表页加载态为**骨架屏**（avatar + 文本条 + 图片占位）|

## 二、逐页走查结论（前端 1397 视口）

| 页面 | 结论 |
|---|---|
| 首页 / 消息 / 通知 / 附近三叶 | ✅ |
| 搜索（发现态双列、结果列表限宽居中）| ✅ |
| 个人中心 / 编辑资料 / 我的群组/瞬间/举报 | ✅ |
| 设置（描述列 240–300px 自适应）| ✅ 已修参差断行 |
| 关系（列表限宽居中）| ✅ 已修右侧空白 |
| 群组详情 / 瞬间详情 | ✅（hero 全幅 + 双栏侧栏）|
| 管理后台（概览/用户/地理统计）| ✅，合计数值用 accent 强调属设计意图 |

## 三、鸿蒙端结论

- 原语层强制 `MIN_HIT_TARGET=44`，行级动作用 `.responseRegion` 外扩（不占布局）；
- 免登录页面已全部走查（首页/附近三叶/群组详情/瞬间详情/用户主页/搜索/消息游客/我的游客/登录页）；
- 登录门后页面用**访客登录**（登录页「访客」tab）可走查；管理后台需管理员账号；
- 已修：页面容器档位（搜索/管理概览等 6 页 form→default）、搜索页重复卡、空态卡、
  无封面精选卡紧凑化、附近列表设点后自动重载、设置页两处文案/截断。

## 四、未完成/待确认

- 鸿蒙账号密码登录在模拟器上稳定失败（提交瞬间账号字段被改写，未发出请求）→ 待人工/真机复核；
- 鸿蒙「编辑资料」页保留底部 tab 栏（其他推入页没有）→ 可能丢失草稿，需改导航结构；
- 鸿蒙区块标题文字链接热区 < 44vp（需结构调整）。

## 状态组件审计（2026-10-21，两端逐处核对）

**规则**：加载/错误/空态应落在**所在页面或容器自身的表面**上，不得再叠一层卡片
（状态"浮在页面上"的直接原因）；卡片只用于本身就是一个独立表面（弹层、Sheet、浮层）。

| 组件 | 端 | 处数 | 结论 |
|---|---|---|---|
| `EmptyState` | 鸿蒙 | 55 | 全部 `bare: true`（34 处原为卡片，已改）|
| `<EmptyState>` | 前端 | 51 | 全部带 `bare`（29 处补充）|
| `SystemLoadingView` | 鸿蒙 | 31 | 默认无卡（`card` 选项保留给浮层）|
| `<LoadError>` | 前端 | 34 | 内部就是 `<EmptyState bare>` |
| `<InlineLoading>`、`<ListEnd>`、`<LoadMoreButton>` | 前端 | 41 / – / – | 本身无卡 |
| `Skeleton`、`SkeletonShimmer`、`ListEnd` | 鸿蒙 | – | 本身无卡 |
| `+error.svelte` | 前端 | 1 | 整页错误页，无卡 |
| 硬编码状态文案 | 鸿蒙 10 / 前端若干 | – | 均为页内纯文本，**0 处**带卡片 |

**复核方法**（可重跑）：用括号/标签平衡解析枚举调用点（`grep` 首行会漏掉跨行 `bare`，
本会话曾因此误报 6 处），再检查调用点上文是否出现卡片样式；对告警处逐一读取父级容器判断
是"状态放在页面卡片内"（正确）还是"额外套了一层"（错误）。

**未改动（有意）**：状态仍为**垂直居中**摆放 —— 这是版式选择，与"叠卡片"无关。
## 2026-10-26 补充（Round 12：骨架屏 / 场景化空态 / 超宽屏 / 页头口径 / 鸿蒙氛围画布）

**口径**：web 用 agent-browser（headless，独立 session）；鸿蒙用真机 HUAWEI Mate 60（hdc target
`MJE0224A11012728`，1216×2688，非模拟器）。证据根目录：`.workwork/ui-polish-1026/`
（web: `fe/`；鸿蒙: `ha/`；独立复核: `review/`，均为 gitignored 本地产物）。

### 一、Web 端本轮实测（改后基线）

| 项 | 方法 | 结果 |
|---|---|---|
| 全量回归 | 39 页（390×844 / 1440×900 亮暗 + 2560×1440）同口径 metrics | 横向溢出 0、坏图 0、h1 全 =1（基线 desktop 首页 h1=0、分栏聊天 h1=2 已修）、console error 0 |
| 骨架屏真实加载态 | 页面内 patch fetch 延迟 → 可信点击 SPA 导航 → 加载窗口截图 + DOM 量测 | 桌面 12 页 + 移动 5 页均出现 `[data-skeleton-root]` 且 `aria-busy=true`；行高 68–72、前导圆 40/44、文本条 14·55% / 11·35%、条间距 8、圆角 12；骨架→内容首块位移 ≤2px |
| reduced-motion | `prefers-reduced-motion: reduce` 下同法采集 | `animation-duration≈0.01ms`、`animations=0`，骨架保留形、不闪动 |
| 场景化空态 | 消息访客 / 我的访客 / 我的瞬间访客 / 搜索无结果 × 390+1440 × 亮暗 | spotlight 装饰层 0→1，几何 320/288/168/±102 与 spec 一致；`aria-hidden`、`pointer-events:none`、无动画 |
| 2560 超宽屏 | 1440/2560 量 `h1` 与正文首块 x/宽 | `/settings` Δ605→0（页头与正文同 56rem 居中）；`/search` 2xl 两段式（左栏 4 分栏 + 热门动态整行 4 卡，左下空白消除）；1440/390 像素不变 |
| 页头图标口径 | 截图 + computed style | 首页品牌头 ghost（2 枚：搜索/通知）；标准 PageHeader surface（hit-slop 48px≥44）；`/nearby` 枢纽偏离已修 |

### 二、鸿蒙端本轮实测（真机）

| 项 | 结果 |
|---|---|
| 氛围画布 AmbientBackdrop | 三主题 × 明暗 6/6 像素校验一致（星尘/星座/轨道/天际线、等高线/暖阳/林线、墨山/圆相/墨点）；深色可见世界层；`hitTestBehavior(None)` + `accessibilityLevel('no')`；无逐帧动画（平台适配） |
| 场景化空态 spotlight | 消息/我的访客态、我的群组空态、搜索无结果均渲染；双环为正圆（`.border()` 在 `.borderRadius()` 之前）、访客消息态氛围层可见 |
| 页头对齐 | 首页 2×ghost（无地图）；附近枢纽/附近人/附近群组 trailing = surface；与 web 移动端并排一致 |
| 搜索页三态 | 输入即搜 + Skeleton 加载 + 无结果 spotlight；结果用户行 68vp 紧凑行对齐 web；无「只有 tab 下方全空」中间态 |
| 触控热区 | 区块标题链接 `responseRegion` ≥44vp（原登记 backlog 关闭） |
| 构建 | hvigor `assembleApp`：TYPE CHECK SUCCESSFUL + BUILD SUCCESSFUL；`bun run check:theme` 四项全绿 |

### 三、待观察（未计入本轮缺陷）

- settings 2560 正文列宽 1024→896（页头/正文对齐的预期副作用）；search ≥1792 新布局需产品确认；
- 鸿蒙氛围画布多实例内存（实测 4→5 块 Canvas、PSS 218.7→330.2MB 含页面数据）与搜索 Tab 计数格式差异见 `docs/ui-improvement-handoff.md` 待做。
## 2026-10-27 补充（Round 2：整卡图片卡 / 去页面级边框 / 选中态无障碍）

**口径**：web 用 agent-browser（headless，独立 session，worktree 8889 实现环境）；鸿蒙本轮**真机不可用**
（hdc 离线；模拟器系统截图黑屏、componentSnapshot 离屏取证失败，均已复现并存档），鸿蒙侧证据为
`assembleApp` 构建日志 + uitest 布局树 + token 一致性脚本，真机像素验收清单见 ha report §7。

### 一、Web 本轮实测（改后基线）

| 项 | 方法 | 结果 |
|---|---|---|
| 整卡图片卡 | 10 核心页 × 5 视口（390/768/1024/1440/2560）× 亮暗 = 100 张矩阵 | 图片铺满整卡、无 border、圆角 20；feed 4:5、网格 3:4；顶/底遮罩覆盖作者/徽章/标题/元信息/操作行；多图首图 + 1/N；无图 fallback 主题渐变+0.08 水印+主题色文字；横向矩阵 0 控制台错误、0 横向溢出 |
| 压图文字对比度 | 卡内封面替换纯白 data-URI（最坏情况），元素「有字/无字」两帧取字形区逐像素算 WCAG | 初测发现顶部作者行不达标（昵称 2.46/时间 1.71）→ 顶部遮罩改三段 0.86/0.64@40%/transparent + 高度 128（双端）后复测达标：昵称 7.47/7.69、时间 4.89/5.04；全局最低 4.81（群名 light），feed 标题 10.70/12.14；明细 `fe/after/contrast-metrics.json`、`review/verdict-r2.md` |
| 去页面级边框 | 桌面 1440/2560 截图 + DOM 量测 | 顶栏/侧栏/主内容 border=0、bg=transparent、无圆角/投影；ambient 单实例（DOM 计数 1）；侧栏↔主内容交界无接缝；PageHeader 无硬底边线（半透明画布+blur）；移动首页品牌头/聊天头/在线条硬线移除 |
| Tab 选中态 | 浏览器 `filter: grayscale(1)` 实拍（亮/暗） | segmented 选中 = accent 实底 + ✓ + semibold；pills/underline = semibold（underline 有下划线）；灰度下选中项一眼可辨 |
| 骨架零位移 | 延迟 API 5s → 骨架矩形 vs 内容首卡矩形 | 首页 390：骨架 334×418 @(28,115) = 内容 334×418 @(28,115)；SkeletonFeed 已改整卡形状 |
| 回归 | 100 张矩阵 metrics + 39 页同口径巡检 | 0 横向溢出、0 坏图、h1 全 =1、console error 0 |

### 二、鸿蒙本轮（代码/构建/布局树，待真机像素验收）

| 项 | 结果 |
|---|---|
| 整卡图片卡 | `MomentFeedCard variant='overlay'`（首页 4:5、我的瞬间 3:4）、`NearbyImageCard mediaOverlay`（三叶 3:4）、`MediaOverlay` 原语（顶/底遮罩、fallback 渐变+0.08 水印、强遮罩徽章、白图标操作键）；布局树实测 feed 0.800、网格 0.750；无 border；多图 1/N；fallback 主题色文字 |
| 去页面级边框 | `SystemPageHeader` 去硬底边线；`DesktopLayout` 侧栏/主面板/顶栏去白浮卡、侧栏分组线删除；内容单元卡片保留并逐页登记 |
| Tab 选中态 | `TabsPillGroup` variant（segmented/pills/underline）：segmented = 实底 + ✓ + semibold；布局树可见独立 ✓ 节点 |
| 跨端 token | 新增 scrim/fallback/watermark 与 web 逐值一致（`check_scrim_parity.py` 6/6 PASS）；`check:theme` 四项全绿 |
| 构建 | `assembleApp` TYPE CHECK SUCCESSFUL + BUILD SUCCESSFUL；无 .ets 晚于构建 |
| 待真机 | 三主题×明暗矩阵、灰度 Tab、压图对比度像素实测、覆盖态双态截图、web↔鸿蒙并排 compare —— checklist 见 `ha/report.md` §7 |

### 三、去边框判定口径（本轮登记，后续沿用）

| 位置 | 判定 |
|---|---|
| 桌面顶栏/侧栏/主内容、PageHeader、移动品牌头/聊天头/在线条、群详情概览白卡 | **页面级框 → 去** |
| 设置分组 ListContainer、瞬间详情单帖卡、群详情右栏、成员胶囊、rail 卡片 | **内容单元 → 留**（浅底/投影/极浅描边，不读成整页框） |
| 消息分栏竖线 | 功能分栏，降级为 `border-subtle/40` 发丝线保留 |
| 聊天输入条/底部 dock/弹层 | 控件/遮罩层，保留 |
## 2026-10-27 补充（Round 3：内容软岛 / 页面级硬线清零 / 次级页精致化）

**范围**：Web 22 文件 + 鸿蒙 15 文件；方案见 `docs/design-system.md` §19。证据：
`.workwork/ui-polish-1026/r3/fe/{raw,compare}`、`r3/ha/{compare,interim}`。**鸿蒙真机仍离线**：
端上像素（软岛阴影/圆角/gap、按钮 hover、未读态、骨架零位移）未验证，待真机矩阵。

### 一、内容软岛（Web 实测）

| 项 | 方法 | 结果 |
|---|---|---|
| 岛形态 | DOM 抽测 + 截图 | 无 1px border；`rounded-lg`(20) + `bg-surface-island` + `shadow-xs`；`p-2`、行透明、行间 gap 2、hover 填充离边 8 |
| 暗色可辨 | 岛 vs 画布亮度比（像素采样） | vibrant 1.33 / forest 1.16 / inkwash 1.31（修复前 1.01–1.07 糊平） |
| 岛内文字对比 | 既有 token 口径计算 | light 13.5–15.5、dark 10.7–13.1；次要文字 5.3–7.6，全部 ≥4.5 |
| forest 亮色岛 | 合成实测 | 与画布同色（ratio 1.00），边界仅靠 shadow-xs；登记待产品决策是否改 `primary 92% + text 8%` |
| 暗色行透明 bug | 截图/像素 | 修复前暗色岛被不透明行盖住（ratio 1.01），已修为行透明（1.18–1.33） |

### 二、页面级硬线扫描结论（全站）

| 位置 | 类型 | 结论 |
|---|---|---|
| /me 自定义页头、他人主页分区头、群详情分区标题、搜索页头/sticky 分区、reports 过滤行 | 页面级 | **删除** |
| 聊天输入条 / 粘底操作条（border-t） | 功能线 | 保留 |
| 消息分栏竖线（border-r） | 功能分栏 | 保留（`border-subtle/40` 发丝线） |
| 表格/统计行（admin stats 等） | 数据功能线 | 保留 |
| 卡片内新手引导/演示分隔 | 内容单元内 | 保留 |

### 三、六页打磨与骨架

| 页 | 结论 |
|---|---|
| /me | 身份卡/数据列软岛；编辑/设置按钮软填底无描边；stats capsule；页头线删除；桌面版同步 |
| /me/edit | 字段收进表单软岛；输入框功能边界与错误态不变；danger 改危险软底、注销入口软岛行（二次确认保留） |
| /settings | 7 组卡全部软岛；segmented/开关/badge/退出登录收敛；页内无页面级横线 |
| /notify | 列表软岛；接受=主按钮、拒绝=软填底；未读点/分页 tabs/空态保留；16+ 行长列表仍成组 |
| /me/relationships | 列表软岛；发消息软填底；tabs/搜索框控件语言保留 |
| /search | 结果/发现列表统一软岛；最近搜索/热门话题/人物/群组/动态语言一致 |

- **N1 搜索骨架**：行高 60→64（对齐真实行）+ 分组头占位（48×16 + 24×11）；骨架→内容零位移。
- **N2 消息骨架**：移动端非分栏形态补在线条 92px 占位（与真实条几何一致），避免加载后整体下推。

### 四、鸿蒙镜像（代码/构建/布局树，待真机像素）

| 项 | 结果 |
|---|---|
| surfaceIsland | 6/6 与 web 逐值一致（`check_surface_parity.py` PASS）；`ListGroup`/`SystemCard`/`SettingsSectionCard` 软岛；ProfileIdentityCard 去框 20；stats capsule；通知容器式软岛+未读态；搜索页头/分区线清零；骨架同步 |
| 构建 | `assembleApp` TYPE CHECK + BUILD SUCCESSFUL；`check:theme` 全绿 |
| 真机 | 软岛像素/阴影/圆角/gap、按钮 hover、未读态、骨架零位移待真机矩阵（Round 13+14 同批补） |
## 2026-10-27 补充（Round 4：长尾/后台软岛化 + 交互状态/可访问性）

**范围**：Web 18 文件（+51/−48）+ 鸿蒙长尾页面（P0 原语已完成，P1/P2 镜像中）；方案见 design-system §19.5（例外清单）/ §20（交互基线）。
证据：`.workwork/ui-polish-1026/r4/fe/{raw/before-matrix,raw/after-matrix,raw/audit,compare}`、`r4/ha/{interim,compare}`。**鸿蒙真机仍离线**，端上像素待补。

### 一、长尾/后台软岛化（Web 实测）

| 页面/区域 | 结果 |
|---|---|
| 瞬间详情 | 单帖卡改内容软岛；媒体容器/九宫格/回复框去 1px 描边、保留圆角/阴影/focus-ring；统计/评论分隔发丝线保留 |
| 首页长尾 | onboarding 卡、未登录关注卡、悬浮 Tab 容器、HeroStats、GroupCarousel、NearbyUserRows 去框；整卡图片卡语言不变 |
| 群详情 | hero 封面去描边保留圆角/阴影/占位；指标/操作/成员分组与 5 个子列表按钮软岛/软填 |
| 他人主页 | 身份/数据卡与 /me 同构；私信/屏蔽提示收敛 |
| admin 10 页 | 概览身份条+9 模块卡、config 5 分区卡+保存条、stats 卡/图表容器/筛选、tags danger callout；6 张表共用 `table.ts`：外壳去描边、表头 bg-subtle+行线+行 hover 保留、行按钮软填/soft danger、390 横向滚动、空态/分页统一 |

### 二、交互状态与可访问性（独立实测基线）

| 项 | 方法 | 结果 |
|---|---|---|
| focus-visible | 键盘 Tab 逐次 + computed outline；亮/暗 | 2px solid accent（rgb(43,94,178)）；admin 表格行按钮（tab36 进入「封禁」）同 2px；moment/group/users 390 可见；skip link 首 Tab 可见 |
| hover / press / disabled | 计算样式 + 截图 | 行按钮 soft danger 背景 0.065→0.13；press scale .94–.95；disabled opacity-faint + pointer-events |
| reduced-motion | `set media reduced-motion` 遍历 | ambient 24s/36s、ping 1s、pulse 2s、hover 0.18s → reduced 下 ambient 消失、其余 1e-05s 单次、hover 1e-05s（全局 utilities 规则）。**2026-10-28 R15 起 ambient 动画默认关闭**（opt-in），本条仅适用于历史版本与 motion 显式开启页面 |
| 加载/空/错误 | 逐页核查 | moment = Skeleton；group/users = InlineLoading（理由：单实体/短列表、<1s、避免二次位移）；admin = InlineLoading + EmptyState + LoadError + LoadMoreButton；mock 后台筛选参数未实现 → 空态不可复现（已注明，未伪造） |

### 三、回归

- QA 23/23 `issues: []`、console 0；补充 16 路由 ×2 视口 32/32（0 溢出 / 0 坏图 / h1 正常 / 0 error）；
- check 0 errors 0 warnings + eslint 0 + theme 四联 OK；test 100 文件/518；build 通过。

### 四、鸿蒙（代码/构建/布局树，像素待真机）

- P0：`FocusRing`（focusBox 2vp accent + 2vp 外距，SystemButton/SurfaceIconButton/Pressable/SystemInput 接入）；`MotionUtil`（reduced 下 spring duration=0 / pressScale=1 / shimmer 关闭，原语已接）；AdminOverview 软岛。
- P1/P2：MomentDetail/GroupDetail/他人主页软岛映射（按 web 终稿值），布局树与 compare 见 r4/ha；build/check:theme/parity 全绿；端上像素待真机。
## 2026-10-27 补充（Round 5：三主题验收 + 可访问性/触控/性能质量门）

**范围**：Web 39 文件（+167/−136）+ 鸿蒙 19 文件（+72/−5）+ 文档 4（+136/−6）+ 技能 3（+35）；基线 `4c61487d`。
方法与工具：axe-core 4.10.2（`agent-browser --init-script axe.min.js` 注入）；触控扫描脚本；Performance API 采样。
**鸿蒙真机仍离线**：TalkBack / 系统 reduced-motion 开启路径 / 焦点环 / 软岛像素待真机。

### 一、三主题 × 明暗验收

| 组合 | 软岛/画布亮度比 | 说明 |
|---|---|---|
| vibrant light / dark | 1.11 / 1.33 | 未变 |
| forest light | **1.17（#E4E6E2）** | R5 决策：仅 forest light 覆写 `surface-primary 92% + text-primary 8%`（原 1.00，纸面主题画布与卡片同色，仅阴影不足） |
| forest dark / inkwash light / inkwash dark | 1.16 / 1.06 / 1.31 | 未变；inkwash light 1.06 偏弱但文字对比达标，保持 |

- 岛内文字：forest light primary 13.11:1 / secondary 6.42:1；其余组合 ≥4.5（axe color-contrast 主题扫描 36 runs 0 flagged）。
- 主题相关 axe 对比问题修复：settings「已登录」徽章、瞬间详情 tag chip、admin/入群申请状态徽章（改实底 + on-accent 字）。

### 二、可访问性（axe-core）

| 项 | 修复前 | 修复后 |
|---|---|---|
| 主扫描 50 runs（35×1440 light + 5×390 light + 10×1440 dark） | 23/50 页 0 violation；critical 3、serious 多类 | **50/50 页 0 violation** |
| 主题附加 36 runs | 4 处 color-contrast | **0 flagged** |

- 修复项：nested-interactive 7 页/51 节点（整卡改「覆盖按钮 + 行内动作 auto-lift」，卡片根不再持 button/aria-label）、
  aria-required-children 3（列表行补 `role=listitem`）、definition-list/dlitem（MetricRow 改 `div role=group`）、
  aria-prohibited-attr 13（listitem/徽章去 aria-label，改覆盖按钮或 sr-only 文本）、scrollable-region-focusable
  （admin config `pre` 加 tabindex/role=region/aria-label）、landmark（login/register/error 页补 main）、
  image-redundant-alt（头像装饰化 alt=""/aria-hidden）。
- 新增验证入口：axe 扫描脚本与基线（工具在 `.workwork/ui-polish-1026/r5/tools/axe.min.js`，页面注入即扫）。

### 三、触控目标（390×20 页）

| 项 | 修复前 | 修复后 |
|---|---|---|
| 可点元素有效热区 ≥44 | 321/465 = 69.0% | **425/466 = 91.2%** |
| 离群 | 144 | **41（全部登记例外）** |

- 新增 `.hit-slop-44` 原语（孤立图标/文本链扩 44，不占布局）；紧凑按钮视觉 ≥32 或 slop。
- 例外清单：地图 marker 17（36×53，防相邻误点）+ 地图控件 2、相邻 TagPill 22（/me/edit×16、/search×6，WCAG 24 已满足，扩 44 会强制 26px 间距溢出）、
  表单控件 34、skip link 19（聚焦展开）。鸿蒙侧 responseRegion 补 5 处，边缘 tap 实测通过；同类例外登记。

### 四、性能基线（本地 dev/mock，仅相对参考）

| 指标 | 实测 |
|---|---|
| TTFB | 5–12ms |
| FCP | 152–284ms |
| LCP | 180–436ms |
| CLS | 0–0.0195（全部 <0.1） |
| 长任务 | 0 |

结论：无修复项；关键页首屏/布局稳定性达标，记录为基线（生产环境需另测）。

### 五、修复过程中发现并修掉的回归

- **/login 横向溢出 13px**（复核 Blocker）：`.hit-slop-44::before` 的 5 个未定位 link-btn 宿主以登录页 `<main class="relative">` 为基准撑宽页面；
  修法：`.hit-slop-44:not(.absolute):not(.fixed):not(.sticky){position:relative}`（宿主自动获得定位基准，显式 absolute 不被覆盖），
  utilities 注释写明「宿主必须定位」；复验 /login 1440/390 ovf=0、28/28 hit-slop-44 路由 0 溢出、39 页回归不退化。

### 六、回归

- 39 页：主巡检 22/22 `issues=[]`、console 0；补充 16 路由×2 视口 32/32（0 溢出/0 坏图/h1 正常/0 error）。
- check 0 errors 0 warnings + eslint 0 + theme 四联 OK；build 通过；test 全量（MetricRow 契约已随语义更新）。
## 2026-10-27 补充（Round 6：图片加载管线 + 感知性能 + 搜索键盘导航）

**范围**：Web 22 文件（20 改 + 2 新增：`lib/images.ts` + 单测；+266/−31）+ 鸿蒙 15 文件（14 改 + 1 新增 `ImageUrlUtil.ets`；+196/−18）+ 文档 4 + 技能 3；总计 41 改 + 3 新；基线 `eaa3098e`（数字以复核快照为准）。
规范：design-system §20.8。证据：`.workwork/ui-polish-1026/r6/fe/{har,compare,raw/after/screens}`、`r6/ha/{interim,logs}`。
**鸿蒙真机仍离线**：流量/首屏时间/渐显像素/reduced 系统路径待真机。

### 一、图片管线

- `lib/images.ts`：仅 `images.unsplash.com` 追加/覆盖 `auto=format&fit=crop&q=78|80|82|85&w=<width>`；其他域名/本地/blob/无效 URL 原样；
  6 档预设（avatar/thumb/card/cover/hero/full）+ `presetSrcSet` + `avatarImageWidths`（32→64/96、48→96/144、xl→160）+ `fadeImage`（0.22s，reduced 1e-05s）。
- 接入 26 个 `<img>`：feed 封面/行卡、附近卡、详情大图+九宫格、群 hero、头像、通知、搜索、他人主页、rail、ImageViewer（full 1600/q85 eager）、地图图钉；
  首屏仅 feed 前 2、附近的人/附近瞬间前 2（复核 N1 已补）、详情首图、hero、Viewer eager+high，其余 lazy+async；非 unsplash 不生成 srcset；
  onerror/srcset 降级保留；静态 logo 补 `decoding=async`。

### 二、前后数据（transferSize，各 2 轮）

| 页面/视口 | before → after | 变化 | CLS |
|---|---|---|---|
| `/` 1440 | 443,364 → 168,762 | **−62.0%** | 0.0195→0.0195 |
| `/` 390 | 324,624 → 117,284 / 150,667 | −63.9% / −53.6% | 0 |
| `/nearby/people` 1440 | 108,037 → 154,305 | **+43%（清晰度收益：封面 200w→400w）** | 0 |
| `/me` 1440 / 390 | 18,679 → 10,043 / 5,596 | −46.2% / −70.0% | 0 |
| 详情 1440 | 264,510 → 59,207 | **−77.6%** | 0.0213→0 |
| 详情 390 | 93,961 → 45,051 | −52.1% | 0 |

- LCP 方向改善/持平（本地 dev 噪声、冷启动离群已在报告注明）；CLS 全部不回归（详情 0.0213→0）。
- 结论：清晰度提升 + 流量整体大幅下降；nearby 唯一上涨为「低清→清晰」的有意取舍（如需省流可对非首屏回退 thumb 档，登记备选）。

### 三、搜索键盘导航

- `/search`：↑↓ 在输入框/相关话题/结果行/最近条目间移动焦点（真实链接/按钮，focus-ring 可见），Enter 打开（实测 → /users/user_lumos），
  Esc 清空并回输入框；未引入 combobox/aria-activedescendant；Tab 顺序不变；axe 复扫 50 runs 0 violation。

### 四、回归

- axe 39 页/50 runs **50/50 0 violation（0 serious/critical）**；browser-qa 23/23 页 issues=[]、console 0；
- check 0 errors/0 warnings + eslint 0 + 主题四联 OK；test **101 文件 / 526 用例**；build 通过。

### 五、鸿蒙镜像（代码/构建/布局树，像素待真机）

- `ImageUrlUtil.ets`：与 web 同 host/参数规则（w/auto=format/fit=crop/q=78），其他域名原样；15 处接入（卡片/详情/头像/缩略图/通知）；
  宽度档对齐（首屏 1080/大图、卡片 900:800、头像 2x）；首屏前 2 优先、LazyForEach 离屏不创建（ArkUI 无 fetchpriority/lazy，登记平台差异）；
  高频图占位 + onComplete 渐显 200ms（reduced 直接显示）；失败 fallback 保留；build/check:theme/parity/布局树全绿。
## 2026-10-27 补充（Round 7：微动效与状态反馈）

**范围**：Web 16 文件（14 改 + 2 新增 `lib/motion.ts` + 单测；+184/−37）+ 鸿蒙 16 文件（+491/−30）；基线 `6d292e98`。
规范：design-system §20.9。证据：`.workwork/ui-polish-1026/r7/fe/{compare,raw}`、`r7/ha/{interim,logs}`。
**鸿蒙真机仍离线**：动效手感/中间帧像素/reduced 系统开启路径待真机。

### 一、动效实现

| 项 | 实现 | reduced 行为 |
|---|---|---|
| 页面进场 | `RoutedContentViewport` afterNavigate 仅 `nav.type!=='enter'`（首屏不播护 LCP；popstate 播）；内容层 opacity 0→1 + translateY 4px，180ms | 不播（无类/无事件） |
| 骨架→内容 | home/notify/messages/search/admin-users `contentFade`（仅实际展示骨架时、缓存命中不播） | 直接显示 |
| Toast 退场 | 每条 `out:toastExit`（fade+y −8px，150ms）+ 容器常驻（最后一条可退）；空 aria-live 无噪音 | 立即移除 |
| 状态 pop | 点赞/收藏/关注/加入 11 处（MomentCard/MomentPostBody 点赞收藏评论赞/FollowButton/JoinButton/群详情加入/鸿蒙 11 处）；仅激活方向，点击 nonce 驱动 | 无 class/无事件 |
| 评论新增 | 仅本次新增行 fade+6px（160ms；600ms 标记清除） | 跳过 |

- reduced 全矩阵：normal vs reduced 对照截图与 computed（`cst-page-enter` 0.18s vs 无；`cst-state-pop` 0.22s vs 无；toast 30ms 仍存在→300ms 移除 vs 30ms 已 0）；
  JS 路径均走 `prefersReducedMotion()` 判断，无动画类/定时器残留。

### 二、性能红线

| 指标 | 结果 |
|---|---|
| LCP（dev） | home 560/412、nearby 628/276、detail 796/616、search 640/492（同 r6 量级；首屏无进场动画） |
| CLS | home 1440 0.0195（与基线一致）、390/其它 0；search 390 0.0995（既有 discovery 位移，非本轮） |
| 长任务 | 0 |

### 三、回归

- axe 39 页/50 runs **0 violation**；browser-qa 23/23 页 issues=[]、console 0；
- check 0 errors/0 warnings + eslint 0 + 主题四联 OK；test **102 文件 / 540**；build 通过。

### 四、鸿蒙镜像（代码/构建/布局树）

- `MotionUtil` 扩展为唯一动画入口：`contentEnter`（180ms）、`statePop`（220ms，1→1.15→1，keyframeAnimateTo 两段）、`popScale`、`enterOffset`；reduced 直接终态；
- 内容进场 10 处（Home 冷启动跳过、tab 切换/刷新重播、缓存命中不播；触发/跳过条件见 ha report §2 表）；状态 pop 11 处，旧手写 150ms/1.3 模式全部替换；
- 无新增 token；build/check:theme/parity/布局树快照正常。
## 2026-10-27 补充（Round 8：媒体占位/降级 + 搜索 URL 状态）

**范围**：Web 12 文件（10 改 + 2 新增 `lib/search-url.ts` + 单测；+122/−41）+ 鸿蒙 7 文件（+212/−67）；基线 `1bdf0b5e`。
规范：design-system §20.10；证据：`.workwork/ui-polish-1026/r8/fe/{raw,compare}`、`r8/ha/{interim,logs}`。
**鸿蒙真机仍离线**：占位/渐显/失败像素与慢网表现待真机。

### 一、媒体占位与降级

- Avatar：占位层始终渲染（有 name 首字母 / 无 name UserRound，`aria-hidden`）+ 图片 absolute 覆盖 + `fadeImage` 渐显；失败保留占位不闪烁；decorative/alt 语义不变、无布局变化。
- 统一 `.media-placeholder` 主题渐变占位：群封面、通知缩略图、搜索缩略图、rail thumb、用户主页瞬间、详情封面与九宫格、群 hero；有 Skeleton 列表不叠加。
- **慢网实测**（延迟图片 3.5s）：4 页 × 390/1440「加载中 vs 完成」——加载中无纯灰圆/灰块，占位可见，完成后渐显；无 CLS。
- 占位文字对比（accent-bg 上，text-primary）：vibrant 11.75/9.25、forest 11.58/10.11、inkwash 10.21/8.49（均 ≥4.5；text-secondary 在 inkwash 仅 4.01/4.26 故不用）。

### 二、搜索 URL 状态（`?query=&tab=&tag=`）

| 场景 | 结果 |
|---|---|
| 深链 `/search?query=咖啡` | 恢复 query+tab+tag，执行搜索（**仅 1 组请求**，无 discovery 额外请求） |
| 输入防抖/联想 | 不写 URL（避免历史污染） |
| Enter/选最近·热门/点 tag/切 Tab/Esc | `replaceState` 同步，history.length 不变 |
| 刷新 / 前进 / 后退 | 恢复（popstate 也只 1 组请求） |
| 非法 tab / 空 query | tab→all；空 query 清空全部参数回发现态 |

### 三、回归

- axe 39 页/50 runs **0 violation**；browser-qa 23/23 页 issues=[]、console 0；
- LCP 408/228（home）、324/208、608/220、700/144（1440/390）；CLS home 0.0195 持平、其余 0/0.0025/0.0077（search 390 既有 0.0995 降至 0.0077）；0 长任务；
- check 0 errors/0 warnings + eslint 0 + 主题四联 OK；test **103 文件 / 546**；build 通过。

### 四、鸿蒙镜像（代码/构建/布局树）

- `SystemAvatar`：占位层始终渲染（有 name 首字母 trim+slice(0,1) / 无 name 人形图标，accentMuted 底 + textSecondary 字）+ Image 覆盖 + onComplete 渐显（reduced 直接显示）+ onError 保留占位；decorative/alt/尺寸不变；
- 通知头像改 SystemAvatar（相邻含姓名时 decorative）；缩略图 6 处统一占位 + fade + 失败保留；群 hero 渐变登记为已有意义占位；搜索 URL N/A；
- build 断言无 `.ets` 晚于构建；check:theme/parity/6 页布局树正常。
## 2026-10-27 补充（Round 9：页头滚动/SPA 焦点播报/回顶/reflow）

**范围**：Web 7 文件（5 改 + 2 新增 `BackToTop.svelte`/`route-announcer.ts`，+104/−10）+ 鸿蒙 5 文件（+195/−9）；总计 17 修改 + 3 新增（+400/−21）；基线 `86705ced`。
规范：design-system §20.11；证据：`.workwork/ui-polish-1026/r9/fe/{compare,raw}`、`r9/ha/{interim,logs}`。
**鸿蒙真机仍离线**：回顶出现时机/位置/手感/遮挡、reduced 显隐待真机。

### 一、页头滚动感知（Web）

- 顶部（scrollTop ≤4px）：透明（bg rgba(0,0,0,0)、无 border/shadow/blur），不再有白带；
- 滚动 >4px：`surface-page 78% + blur(16px) saturate(1.4)` + 细底边 + elevation，过渡 150ms（reduced 1e-05s）；
- 三主题 × 亮暗矩阵数据一致（top 透明 / scrolled 玻璃）；页头滚动过程 **0 layout-shift**；滚动回顶恢复透明；18 处 PageHeader 调用无需改。
- **修复既有缺陷（复核 N1）**：`SplitPageLayout` 桌面 3 分支 header 外层缺 sticky（`shrink-0` 包裹高度=header 高 → 内部 sticky 无行程），`/notify` 1440 滚动时 header 随内容滚出；
  给外层加 `sticky top-0 z-sticky`（保留 ContentColumn 对齐）后复验：/notify 1440 scroll 200/400 header top=64 吸顶 + 玻璃态可见，回顶恢复透明；
  /search /settings 1440、390 /notify 无回归，layout-shift 不变。

### 二、SPA 焦点与播报（Web a11y）

| 场景 | activeElement | announcer |
|---|---|---|
| 首屏 | BODY | 空（不播） |
| 导航（首页→附近的人） | main#main-content | 「附近的人」 |
| 后退 / 前进 | main#main-content | 「首页」/「附近的人」 |

- 焦点程序化（preventScroll、outline none），skip link 首 Tab 仍可用、Enter 后焦点 main；不聚焦 h1（避免与 announcer 重复播报）；
- announcer `aria-live=polite`/atomic、1.5s 去重。

### 三、回到顶部（Web）

- 滚动 >1.5 屏出现 44px 圆钮（桌面 bottom-6、距 MockBadge 28px；移动 bottom-24、距底栏 14px）；点击平滑回顶（reduced 直接跳），19ms 内隐藏（<200ms 防闪烁）；键盘可达、不可见移出无障碍树。

### 四、Reflow（Web）

- 320px × 34 页 + 200% 文本缩放（root 32px）× 8 关键页：documentElement/main 横向溢出 = 0；仅装饰 ambient SVG 超出被裁剪。

### 五、回归

- axe 39 页/50 runs **0 violation**；browser-qa 23/23 页 issues=[]、console 0；CLS home 0.0195 持平、notify 0.0069、me/detail 0；0 longtask；LCP 288–644ms；
- check 0 errors/0 warnings + eslint 0 + 主题四联 OK；test **103 文件 / 546**；build 通过。

### 六、鸿蒙镜像（代码/构建/布局树）

- TopBar 滚动玻璃 **N/A**：`grep -rn '.sticky('` = 0，所有 TopBar 在滚动容器外、内容不穿过页头（依据+复现写入 ha report）；
- 回顶：新增 `BackToTopButton.ets`（48vp、ARROW_UP、surface+hairline+shadow、opacity 180ms via MotionUtil/reduced 直接、不可见禁触+移出无障碍树、label「回到顶部」）；
  Home/Nearby/Notification/Search/MomentsFeed 5 页接入（阈值 1.5×视口高、scrollTo 300ms/reduced 0ms、右下避底栏）；
- SPA 焦点/TalkBack、320/reflow 登记 N/A；build 断言无 .ets 晚于构建、check:theme/parity/2 页布局树正常（搜索/通知/瞬间墙回顶未稳定端上触发，非崩溃，登记验证限制）。
## 2026-10-27 补充（Round 10：聊天滚动策略 + 回底/新消息计数）

**范围**：Web 4 文件（chat.svelte.ts/ChatMessageList/ChatView/GroupChatView，+120/−14）+ 鸿蒙 2 文件（1 改：ChatPage +75/−38；1 新：`ChatJumpToBottomButton.ets`）+ 文档 4 + 技能 3；总计 12 改 + 1 新（+291/−57）；基线 `3d102907`。
规范：design-system §20.12；证据：`.workwork/ui-polish-1026/r10/fe/{compare,raw}`、`r10/ha/{interim,logs}`。
**鸿蒙真机仍离线**：上滑不跳/计数/按钮位置/键盘遮挡/150ms 手感待真机；聊天页布局树受登录态限制（游客无会话）。

### 一、滚动策略（Web）

| 场景 | 行为 |
|---|---|
| 初始/切会话 | 直接回底（无动画） |
| 自己发送 | 始终滚底（发送反馈） |
| 近底部（≤120px）收新消息 | 自动滚底（smooth；reduced auto） |
| 上滑（>120px）收新消息 | **不跳转**、计数累加、显示回底按钮 |
| 分页加载更早 | prepend + 视口保持（现有逻辑保留） |
| 会话切换 | 计数/按钮状态清零 |

实测矩阵（私聊/群聊 ×1440/390，WS 注入）：上滑 dist500 收 1 条 → scrollTop 完全不变；批量 2→3、120→「99+」；点击回底 dist0/清零/隐藏；底部注入自动滚；自己发送(dist450)→dist0；reduced 回底 18ms；上滑注入 3 条 CLS 0。

### 二、回底按钮（Web）

- 44px、距底 >200px 显示、right-14 bottom-3（与 dev MockBadge 间距 12px、距输入栏 35px，无重叠）；仅箭头 / 「N 条新消息」（>99→99+）；
  `aria-live=polite`、aria-label 与文本一致；150ms in/out（reduced 0）；键盘 Shift+Tab 可达 + Enter 回底；回底后隐藏。

### 三、markRead 延后

- 移除 WS 即时已读；仅初始/近底部/点回底补标（1s 节流）；上滑期间与离开页面不补标；切会话计数/状态清零。

### 四、回归

- axe 39 页/50 runs **0 violation**；browser-qa 23/23 页 issues=[]、console 0；CLS 0；check 0/0；test **103 文件/546**；build 通过。

### 五、鸿蒙镜像（代码/构建/布局树）

- 新 `ChatJumpToBottomButton.ets`：48vp、ARROW_DOWN、count>0 徽标（99+）、opacity 150ms via MotionUtil/reduced 直接、不可见禁触+移出无障碍树、
  label「回到底部 / 回到底部，N 条新消息」；ChatPage（私聊+群聊共用）ListScroller + `onScrollIndex` 末尾可见近似（无距底像素 API，平台差异登记）；
  仅近底自动滚/markRead、上滑计数不跳、自己发送/初始/分页保持；旧 78% 胶囊删除；build 断言无 `.ets` 晚于构建、check:theme/parity 正常。

## 2026-10-27 补充（Round 11：UI 质量门禁脚本化）

**背景**：r5 的 axe/触控检查此前是一次性会话脚本（未入库）；本轮固化为仓库命令，后续每轮可重复跑。
**范围**：frontend 依赖 + 脚本 + docs + skill；基线 `9245d736`；未 commit。

### 命令

| 命令 | 作用 | 失败条件 | 输出 |
|---|---|---|---|
| `bun run audit:a11y` | axe-core 扫描全部桌面页（默认 1440，38 页：QA 18 + 审计扩展 17 + 详情 3；详情 ID 取不到则 35） | serious/critical > 0、axe 未注入、页面加载失败 | `.workwork/browser-check/a11y-<ts>/` |
| `bun run audit:a11y -- --mobile` | 追加 390×844 5 页（首页/附近的人/搜索/设置/通知） | 同上 | 同上 |
| `bun run audit:a11y -- --reduced` | reduced-motion：关键类残留 + 非允许 CSS 动画 >20ms | 有残留/长动画 | `raw/reduced.jsonl` |
| `bun run audit:touch` | 390×844 触控有效热区，**默认全量页面**（QA 18 + 审计扩展 17 + 详情页，最多 38 页），对照 allowlist | 未登记离群、页面加载失败 | `.workwork/browser-check/touch-<ts>/` |
| `bun run audit:touch -- --baseline` | 仅 r5 基线 22 页（快速/对照） | 同上 | 同上 |
| `bun run audit:ui` | browser-qa --mobile + a11y + touch | 任一子检查失败整体 exit 1 | 三个报告路径 + 摘要 |

参数：`--base`、`--session`、`--theme vibrant|forest|inkwash`、`--mode light|dark`、`--reduced`、`--mobile`、`--limit <n>`（调试）、`--allowlist <path>`（touch，负向验证用）、`--help`。

### 基线（8898 worktree / 2026-10-27）

- a11y：5 页抽样 0 violations（vibrant light；forest dark 3 页 0）；`--reduced` 5 页 0 异常；
- touch：18 页全量 447 元素 / 分母 419、≥44 通过 378、登记例外 41（全部命中 r5 例外）、排除表单/skip link 28、覆盖率 90.2%、未登记离群 0；
- 负向验证：DOM 注入无 alt 图片 + 无名称按钮 → axe `image-alt`/`button-name` critical，脚本 exit 1；临时移除 TagPill allowlist 规则 → touch 未登记离群，exit 1（临时文件已删除，未改仓库）。

### allowlist 维护

- 文件：`frontend/scripts/ui-quality-touch-allowlist.json`；每条规则 = `id` + 稳定 `selector` + `reason`（可选 `exclude: true` 表示不计入分母）；
- **禁止**写入 nth-child / 纯坐标 / 易漂移选择器；优先 `.marker-hit`、`[role=...][aria-label=...]` 这类语义/稳定类；
- 新增例外前先修样式（`hit-slop-44` / 提高视觉尺寸）；确需例外时同步更新本文件与本文登记表，并在 PR/轮次报告中说明；
- 表单控件与 skip link 为 `exclude`（r5 口径，不计入 44 门槛分母）。

### 注意

- 脚本使用独立 session（`hotspot-audit-*-<pid>`），结束时自动关闭；不要与交互会话混用；
- `--limit` 仅调试用；正式门禁不带；输出目录带时间戳，可重复运行；
- 脚本依赖 dev 环境与 agent-browser，**不加入 `check` 链**（check 保持静态检查）；
- 鸿蒙端无浏览器脚本，对应检查仍走 build/check:theme/parity/布局树（见 harmonyapp/AGENTS.md），本门禁为 Web 侧。

### 触控全量收尾修复（2026-10-27）
默认门禁已切到全量页面；此前 `--full` 暴露的 r5 未覆盖离群已全部按 ≥44 修复（视觉尽量不变、无布局位移、无相邻重叠）：
| 位置 | 修复前 | 修复后 | 方式 |
|---|---|---|---|
| /admin/audit 表格文字链（5+） | 22×15 → eff 41 | 22×19 → eff 45 | hit-slop-44 + py-0.5 |
| /admin/stats 明细按钮（2） | 42×23 → eff 23 | 390 下 44×44（桌面不变） | max-sm:min-h-11/max-sm:min-w-11 |
| /admin/stats 热力聚合点（2） | 28×28 → eff 28 | 28×28 视觉不变 → eff 54 | hit-slop-44 |
| 聊天头部整行按钮（2） | 304×36 → eff 36 | 304×36 视觉不变 → eff 48 | hit-slop |
| /share 分享 CTA | 106×40 → eff 40 | 106×40 视觉不变 → eff 52 | hit-slop |
| 404/错误页按钮（3） | 40 高 → eff 40 | 40 高视觉不变 → eff 52 | hit-slop |

全量结果：8898 touch 35 页、8888 待复跑；详见 R11 report.md。

## 2026-10-27 补充（Round 12：主题矩阵 a11y 收口 + on-soft 状态文字）

**背景**：r11 门禁默认只扫 vibrant light；专项矩阵发现 forest light / vibrant dark / inkwash dark 等主题下「tint 软底 + 同色语义文字」有 serious color-contrast。修复采用语义 on-soft 文字 token，背景 mix 不动。

### 矩阵命令

- `bun run audit:a11y -- --matrix`：3 主题 × 2 明暗 × 桌面全量页面（35–38）+ 移动 5 页；聚合 report.json（matrix：每组合 pages/impact/errors）与 summary 矩阵表；serious/critical>0、axe 错误/加载失败 → exit 1。
- 耗时：单 worker 约 25–30min；`--parallel 2` 约 15–20min（各 worker 独立 session/输出文件）。合并前专项门禁，**不进 audit:ui 默认**（默认仍 vibrant light）。
- 快查：`--only /admin/reports,/me/edit --limit 2`（定向路由 + 每组页数），`--limit`/并行仅调试/专项。

### 修复（全站同类组合，非逐页 hack）

- 新增 `--cst-text-{accent,status-online,status-warning,status-danger}-on-soft` = 语义色 80% + text-primary 20%；registry 注册 `text-text-*-on-soft`；
- 替换全站「tint 软底 + 同色文字」：TagPill 激活、admin pending/已确认/警告徽章、后台行内危险按钮（ROW_BTN_DANGER）、确认/举报/Toast/危险菜单项、资料卡徽章、TabBar 激活等（35 文件；含 hover/高亮态写法 `text-status-* hover:text-text-status-*-on-soft`）；
- 背景 token/mix 不动；before→after 对比度（最差软底，含 danger /50）：vibrant light 5.33、vibrant dark 6.64、forest light 5.68、forest dark 5.83、inkwash light 5.18、inkwash dark 6.12（全部 ≥4.5）。
- 鸿蒙：`ColorConfig` 新增 4 个 OnSoft 字段（同 80/20、palette 不变），`harmonyapp/scripts/check_status_parity.py` 断言双端 hex 与对比度。

## 2026-10-28 补充（Round 13：全站视觉改版——锚点/同轴/安静动作）

**范围**：Web 全站（外壳、认证、首页/附近/搜索、资料域、消息/通知、详情页、后台、地图）。方案与规则：`docs/design-system.md` §21 / skill `hotspot-frontend-style`「Round 13」/ `hotspot-frontend-layout`「内容轴」；本轮工作区证据根：`.workwork/ui-review-r13/`（before 基线 / after-<模块> / after-full 全量矩阵 / critique-design.md 独立评审 / DESIGN-SPEC.md）。
**基线**：`9fc07a3b` + R12 未提交改动（状态色 on-soft 与主题矩阵门禁随本轮一起入库）。
**鸿蒙**：本轮只做前端（用户指定）；R12 已同步的 on-soft token 与矩阵门禁不变，鸿蒙侧无新增改动。

### 一、方法

1. **基线取证**：3 主题 × 明暗 × 桌面 1440 / 平板 768·1024 / 移动 390 × 登录/访客/管理员三态，共 528 张（`before/`；含 detail/查询态/空态补充 `extra/`）。
2. **独立评审**：另起只读评审代理通读矩阵 + 关键处像素实测 + 源码定位，输出 `critique-design.md`（P0 2 项 / P1 9 项 / P2 29 项，含可执行改法），Lead 逐项派单到模块。
3. **并行实施**：6 个模块负责人（外壳/信息流/资料/消息/详情/后台）按 `DESIGN-SPEC.md` 与评审清单改造，Lead 提供共享原语（MetricRow strip、SectionHead、EmptyState、NeedsAuthNotice、Button soft、图片卡文字层级、Confirm/Report 弹层）。
4. **验收**：全量矩阵重拍（`after-full/`）与 before 同口径对比；check/test/build；axe 主题矩阵；390 触控；文档与 skill 登记。

### 二、关键改造（按母题）

| 母题 | 落地 |
|---|---|
| 一个锚点 | `IdentityHero`（/me、/users、移动资料页）——封面 132/92 + 头像压封面 + 桌面两栏（左身份/右指标+动作），1024/1440 不再右半空；`/settings` 改单列分组（标题描述在上、软岛在下）；`/nearby/groups` 精选卡降高 192 + 右面板两栏；`/me/groups` 图片网格改横排列表限宽；关系/搜索/举报行补「第二信息位」 |
| 同一张卡 | 全站图片卡统一：标题 `text-base sm:text-lg`、作者行头像 24–32 + 昵称 14、meta 12、动作计数 12 tabular、比例移动 3/4 / ≥600 4/5；MomentCard / HomeFeedItem / UserCard / GroupCard / 我的瞬间 全量对齐 |
| 一条轴 | `/search` 删自设 max-w（标题 284 = 正文 284，修前 +116px）；`/admin` 概览卡片 284 = Tab 栏；`/me` 身份卡与正文同轴；`/moments/[id]` 粘底条与帖子卡逐 px 对齐；后台/详情页统一满宽 + page gutter |
| 安静动作 | `ui/button variant="soft"` 成为唯一安静次级动作（编辑资料/设置/发消息/私信/拒绝/新会话/全部已读/表格行内）；`FollowButton`/`JoinButton` 两态收敛为 surface-accent + soft；描边只留筛选 chip 与功能边界 |
| 指标与分区 | 灰药丸指标块全量替换为 `MetricRow strip`（附近枢纽/资料页/群组详情/后台 KPI/stats）；分区标题统一 `SectionHead`（icon + 标题 + count + 查看全部；rail 变体） |
| 空态/访客 | `EmptyState` 升级（64px 软底图标砖 + 14 semibold 标题 + 12 描述 + extra 建议 chips）；`NeedsAuthNotice` 升级为访客引导面板（3 条价值点 + 主/次 CTA），/me、/messages、/notify、me 子页统一 |
| 外壳/认证 | 登录/注册桌面双栏品牌页 + `/register` 深链（修外部 404）；404 品牌化（路径 sr-only）；分享页挂氛围画布；侧栏分组/选中态、移动底栏选中态与 FAB 光晕；地图左面板悬浮岛 + 类型图标砖 + 图例 + 移动面板收敛 |
| 消息/通知 | 聊天页 `--cst-surface-chat` 画布、头部资料入口、气泡/日期分隔/输入条精修、空会话面板（引导 + 最近 3 条建议）、窄平板 ?peer= 深链修复；通知未读条 + 成对动作（接受 primary / 拒绝 soft）+ 右 rail MetricRow strip |
| 管理后台 | 概览真仪表盘（KPI 行 + 9 模块卡 + 举报待办角标 + 最近审计活动）；10 页统一工具栏/表头/行 hover/行内动作；枚举中文化、目标列链到内容、audit 列折叠、stats 筛选标签化、config 布尔统一 Toggle |
| 详情 | 瞬间详情帖子卡与评论层级重排、移动端动作行去重、粘底条同轴；群组详情 hero 层级、指标 strip、群公告摘要行、右 rail 动作重排、审核中状态 bug 修复 |

### 三、评审 P0/P1 关闭情况

- P0-1 设置页主题/地图源 segmented 文字截断：`TabBar` segmented 标签 `whitespace-nowrap` + ✓ 绝对定位 + 选中项隐藏 swatch；设置页整行宽度 —— 已修（1440/768/1024 × 三主题复验）。
- P0-2 访客 /messages 白色实底页头横带：根因是 messages 页显式 `!bg-surface-primary`，访客分支去掉后与连续画布一致 —— 已修。
- P1-1 次级动作三形态 → soft 唯一形态（`FollowButton`/`JoinButton`/rail/关系/通知/后台）—— 已收敛。
- P1-2 内容轴漂移 → search/admin/me/详情 全部同轴 —— 已修。
- P1-3 通知接受/拒绝权重 → primary + soft，移动端行尾同列 —— 已修。
- P1-4 宽容器空岛 → IdentityHero 两栏 / settings 单列 / 精选卡两栏 / 行第二信息位 —— 已修。
- P1-5 附近群组首屏 → 精选卡降高 + 两栏，网格首行信息可见 —— 已修。
- P1-6 移动瞬间详情动作重复 → 移动只留粘底条 —— 已修。
- P1-7 自己的内容无管理入口 → `ProfileMomentTile`/`ProfileGroupTile` 加 `…` 菜单（编辑/删除/退出群组 + ConfirmDialog）—— 已修。
- P1-8 后台表格问题 → 列优先级折叠/操作列表头/枚举中文/目标链接/stats 筛选标签/config 布尔 Toggle —— 已修。
- P1-9 1024×768 断点：宽平板移除 pb-28（/messages 内容盒 669→665 撑满）、首页操作行不再被 dock 遮挡 —— 已修。

> 逐条证据见 `.workwork/ui-review-r13/after-*` 截图与各模块交付说明；未关闭项见下方「未完成」。

### 四、门禁（最终态）

- `bun run check`：0 error / 0 warning，eslint + theme 四联全 OK。
- `bun run test`：112 files / 588 tests 全绿（含本轮新增 IdentityHero / ProfileCompleteness / AdminModuleCard / deep-link / ChatEmptyPanel / NeedsAuthNotice / MetricRow strip 等契约测试）。
- `bun run build`：通过（`vite build` + adapter-static，7.6s）。
- `bun run audit:a11y -- --matrix --parallel 2 --only '<11 条代表路由>' --base …`：**90 页 × 6 组合（vibrant/forest/inkwash × light/dark）全部 0 违规（serious/critical/moderate/minor 均为 0，0 加载错误）**；命令与原始报告见 `.workwork/browser-check/a11y-20261028-030606/`。
  详情页种子 ID 在 mock/空库环境不可发现，本轮用 `AUDIT_DETAIL_MOMENT/USER/GROUP` 环境变量兜底（脚本已支持）。
- `bun run audit:touch`（390×844）：**38 页 / 826 元素 / 计入分母 742 / ≥44 通过 695 / 登记例外 47 / 覆盖率 93.7% / 未登记离群 0 → PASS**（`.workwork/browser-check/touch-20261028-032912/`）。触控修复：admin/reports 目标链接加 hit-slop-44+py-0.5、share CTA 加 hit-slop、瞬间详情作者头像按钮加 hit-slop、群详情退出按钮加 hit-slop。
- 视觉矩阵：`after-full/` 与 `before/` 同口径（528 张）逐页对比，全部图片有效（无空图/主题错位，脚本校验通过）。

### 五、未完成 / 后续

## 2026-10-28 补充（Round 16：运行时响应 / 消息页 / 导航可读性）


**范围**：Web 前端 30 文件（24 改 + 4 新：`ChatColumn.svelte`、`nav-preload.ts`、`nav-preload.test.ts`、
`ChatMessageList.test.ts`）+ 文档 4 + skill 3。基线 `d311819b`。方案契约见 `docs/design-system.md` §21.12；
证据根 `.worktree/r16-runtime-polish/.workwork/`（perf / messages / nav / audit / lead）。
**触发（用户原话）**：「整个页面反应都慢一拍」「特别的消息页面现在看起来明显很奇怪」「侧边栏的 hover 效果
也太不明显，对色弱、色盲的用户很不友好」，要求「不减少特效」地优化。

### 一、根因（可复现）

| 假设 | 实测结论 | 证据 |
|---|---|---|
| 特效拖慢 GPU | 否（R15 已治理；本轮内容页 animation=0、longtask=0、滚动 60fps） | lead/after |
| 单次交互慢 | 否：Event Timing INP click 16–64ms、hover 24ms、键盘 24–48ms；弹窗 10ms | audit §3 |
| **dev 首访路由按需编译** | **是**：冷路由 58–900ms（页面 + 依赖 transform），warm 8–37ms；生产同路由 8–49ms | perf before/after |
| **页面进场从 opacity:0 起 180ms** | **是**：点击后先空白一拍再淡入，warm 导航「可见」= DOM + 180ms | perf |
| **全局 scroll-smooth** | **是（间接）**：把任何 scrollTop 赋值/无参 scrollTo 动画化 **766–768ms**，破坏滚动恢复/回底 | perf scroll-exp |
| 消息页顶部锚定 + 228px 死空白 | **是**（存量 R13）：短会话最后气泡底 612 / 输入条顶 840 | audit P1-4 |
| 消息页「没有更多了」噪音 | **是**（存量）：未分页会话顶部即渲染 ListEnd | audit P1-5 |
| 消息三列不同轴 | **是**：1440 消息 852@576 / 输入 768@618 / 群聊头 768@618（每侧差 26px）；1920 差 128px/侧 | messages before |
| 侧栏 hover 太淡 | **是**：旧 token 9% text-primary 混 surface-primary，hover 底/画布仅 1.07–1.19:1 | nav before |
| 全站 button 无 pointer 光标 | **是**：侧栏/顶栏/TabBar computed cursor=default（只有 card-hover 等少数是 pointer） | audit P2-1 |

### 二、整改与实测（dev worktree 8898、1440×900、mock）

| 项 | 改前 | 改后 |
|---|---|---|
| 关系页导航 click→DOM / click→完全可见 | 390.9 / 532.4ms | **29.9 / 127.6ms** |
| 通知 click→DOM / 完全可见 | 288.5 / 461.8ms | **26.9 / 167.4ms** |
| 设置 click→DOM / 完全可见 | 454.5 / 598.1ms | **31.7 / 133.6ms** |
| 服务端路由模块 transform（冷） | 58–720ms | **3.3–9.6ms**（warmup 生效） |
| 页面进场 | 180ms / opacity 0 / 4px | **120ms / opacity 0.6 / 2px** |
| 骨架→内容 fade | 150ms | **110ms** |
| main scrollBehavior | smooth | **auto**（程序化滚动即时；回顶仍显式 smooth） |
| 首页 will-change 元素 | 12 | **0** |
| 生产构建（本地静态+真实后端）导航 click→DOM / 完全可见 | 5–49ms / +180ms | **5–34ms / 148–180ms**（含 120ms 进场） |

- 预取：hover 350ms 未点击时已请求目标路由模块；reload 后 7s 内空闲预取 6 个常用路由完成；
  `preloadCode` 不执行 load，无额外 API 请求；`saveData` 跳过。
- 消息页：三列 1303 全部 x=576/w=715、1440 全部 x=576/w=852、1920 全部 x=730/w=1024（±1px）；
  短会话底对齐（最后气泡距输入条 27px）；短会话无「没有更多了」；气泡正文 12→14px、时间戳 10→11px；
  自己气泡正文/时间戳对比度 5.76–7.60、对方 12.5–15.7 / 5.36–8.84（改前 3.97 不达标）；日期标签去胶囊。
- 侧栏：普通文字 4.93–8.45、hover 文字 10.28–12.05、选中文字 8.61–12.05、选中图标/指示条 3.73–5.55；
  hover 底/画布 1.17–1.25、选中 1.27–1.33（旧 1.07–1.19 / 1.13）；灰度 L* 梯 hover→选中 4.3–6.8，
  选中另有指示条 + 内缘描边 + semibold + 描边粗细 4 项非颜色信号（灰度截图 `nav/after/v2-sidebar-contactsheet-all.png`）。
- 光标：`base.css` 全局补 `button:not(:disabled)/[role=button] → cursor:pointer`（内容页实测 pointer）。

### 三、门禁（全绿）

| 检查 | 结果 |
|---|---|
| `bun run check` | **0 error / 0 warning**（svelte-check + eslint + theme 四联 OK） |
| `bun run test` | **117 files / 631 tests 全绿**（安静环境 128s；含新增 nav-preload 8 条、ChatMessageList 契约、motion CSS-JS 同源断言） |
| `bun run build` | 通过（13.6s，adapter-static） |
| `bun run audit:a11y -- --base 8898 --mobile` | **40 页 / 0 违规（minor/moderate/serious/critical 全 0）/ 0 加载错误 → PASS** |
| `audit:a11y -- --matrix --only /messages,/me --limit 2` | **12 页 × 6 组合（3 主题 × 明暗）0 违规 → PASS** |
| `bun run audit:touch -- --base 8898` | 35 页 / 890 元素 / 计入分母 810 / ≥44 通过 763 / 登记例外 47 / **未登记离群 0 / 覆盖率 94.2% → PASS** |
| `bash scripts/browser-qa.sh --base 8898 --mobile` | **23/23 页 issues=[]**（0 横向溢出/0 坏图/0 控制台错误） |
| 截图矩阵 | 消息 1303/1440/1920/390 + 亮暗 + 三主题；侧栏三态 × 6 组合 + 灰度；证据在 `.workwork/{messages,nav,lead}/` |

### 四、独立审计发现（**未关闭**，超出本轮用户反馈范围，登记 backlog）

> 来源：独立只读审计（`.workwork/audit/report.md`，62 次页面加载/67 张截图）。均为**改动前既有**问题，
> 本轮未修改（避免扩大改动面）；复验方法见该报告 §6。

| 级别 | 问题 | 关键数据/根因 | 建议 |
|---|---|---|---|
| P1 | /me 总览页头与正文内容轴漂移 | h1 左缘 284 vs 身份卡 330（Δ46@1440）、Δ158@1920；ProfilePage.svelte:152 正文 max-w-5xl/6xl/7xl 但 PageHeader 未传 maxWidth | 推荐删页级 max-w（正文直接落 ContentColumn）或 PageHeader 传同档 maxWidth |
| P1 | /me/relationships、/me/reports 宽屏填充率低 | 1440 内容 712/1172=0.61；1920 672/1652=0.41；RelationshipsPage/ReportsPage contentMaxWidth=48rem 固定 | 列表页去掉 48rem 或加宽屏档（R13 验收要求 ≥0.70） |
| P1 | /search 移动端首屏 CLS 0.16 | 640ms 两段 shift（section 下移 121px + 卡片 261→4px）；发现态异步分区未预留高度 | 各分区骨架固定高度/min-h；目标 <0.05 |
| P2 | BSP 瓦片墙左缘比页头右移 7px | bsp-layout.ts inset=gap/2 + 页面 gap=14 | 外层 -mx-[7px] |
| P2 | 慢图下移动首页 7 个占位 pulse 常驻 | HomeFeedItem/NearbyImageCard 占位 animate-pulse | 占位限时转静态（生产 S3 影响小） |
| P2 | /nearby 枢纽「在线数」同屏重复 | 页头 chip + Hero 指标 | 保留 Hero，chip 换雷达/刷新 |
| P2 | 首屏卡片图仍 lazy（mock 外链 LCP 1.2–5.2s） | R6 管线仅前 2 张 eager；mock unsplash 外链是预览环境产物 | 真实 S3 环境复测 LCP 后决定是否加 eager |
| P2 | ListItem hover 时间戳对比度部分 <4.5 | palette-hover 半透明叠加 text-secondary：inkwash-dark 3.2 / inkwash-light 3.75 / vibrant-dark 4.03 / forest-dark 4.16 / vibrant-light 4.43（非 hover 5.31–7.62 达标） | 需动 palette 或统一 hover 语言（跨端影响），登记 backlog |

### 五、遗留风险

- 长会话翻页后 ListEnd 仅单测覆盖（mock 会话太短），真实长会话视觉未验；
- 气泡 ⋯ 菜单在 hover:none 真机常显位置未真机复核（headless 按 fine pointer 不显示）；
- 空闲预取在真实网络下多 6 个后台 chunk 请求（`saveData` 已跳过）；若线上不接受，可在
  `scheduleIdlePreload` 加一行开关；
- forest/inkwash 全页 axe 矩阵只抽了 /messages、/me 各 2 页（12 页/6 组合 0 违规），非全量矩阵。

## 2026-10-29 补充（Round 17：backlog 清偿）

> R16 §四登记的 8 项 backlog 全部整改完毕（P1×3 + P2×5）。方案契约补记入 design-system §21.13；
> 证据 `.worktrees/r17-backlog/.workwork/`（base/ 基线、lead/{before,after}/、search-cls/、media-a11y/、lead/*.log）。

| backlog 项 | 整改 | 整改后实测 |
|---|---|---|
| P1 /me 轴漂移（Δ46/158px） | ProfilePage 总览与分区模式统一 `contentMaxWidth=1280` 呼吸档，PageHeader 传同值，删正文包裹 max-w | h1 与身份卡左缘同轴（1440 Δ0）；/me/moments、/me/groups 分区页同步对齐 |
| P1 关系/举报页填充率 0.61/0.41 | Relationships/Reports `48rem`→1280；ProfileUserSection/SectionGroups 删内层 `max-w-2xl/3xl` | 列表岛 1116/1172=**0.95**@1440、1224/1652=**0.74**@1920（≥0.70 达标） |
| P1 /search 移动 CLS 0.16 | 发现态三请求独立 loading 态，加载期在最终位置渲染等高骨架（话题 chips/人物/群组/动态四种），落定原位替换 | 390 冷启动 8 次 median 0.08/max 0.40 → **0×8**；1440 → **0×8**；几何逐项相同（scrollH 2817→2817） |
| P2 BSP 左缘 +7px | BspCardCanvas 容器外扩 `±gap/2`（布局几何，非主题值） | moments 1440：slot0 x=284 = h1 x=284（Δ0） |
| P2 占位 pulse 常驻 | 新 `.media-placeholder--pulse`（3×skeleton token ≈4.5s 后静态），HomeFeedItem/NearbyImageCard 替换 | 390 首页改前 7 个 infinite pulse 常驻 → 改后 3 次后归零（@8s/@14s = 0），reduced-motion 0 |
| P2 /nearby 在线数双写 | 删页头「N 在线」chip，Hero MetricRow 单一表达 | 同屏「在线」节点 2 → 1 |
| P2 首屏图 lazy 缺口 | GroupCard 加 `priority` prop；/nearby 枢纽前 2 张 UserCard、/nearby/groups featured+rest[0] 传值 | /nearby 1440 eager 0→2、/nearby/groups 1440/390 eager 0→2（像素 diff 0 回归） |
| P2 ListItem hover 对比度 <4.5 | ListItem 根 `data-list-item` + 组件内规则：hover(fine pointer)/focus-within 时行内 `.text-text-secondary` 升 text-primary（pressed 行排除） | 真实鼠标 hover 实测 6 组合 fg 全部切 primary；按合成底色算 **10.61–13.86 全 PASS**（改前 2.49–4.7），pressed 行不受影响 |

### 门禁（R17 合并树，全绿）

`bun run check` 0/0（含 theme 四联）；`bun run test` **117 files / 640 tests**（新增 SearchDiscovery 骨架契约 3 条、媒体/列表/群组卡契约）；`bun run build` 通过；`audit:a11y -- --mobile` **40 页 0 违规 PASS**；`audit:touch` 35 页未登记离群 0（94.2%）；`browser:qa --mobile` 23/23 页 issues=[]。

### 仍开放（非本轮范围）

- 首屏图 LCP 需真实 S3 环境复测（mock unsplash 外链不能代表生产）；
- 搜索骨架按 limit 估算，极端标签分布仍可能有 28/36px 级一次性位移（当前 mock 六视口零位移）；
- GroupCarousel 若未来移入首屏需补 priority 接口（当前首屏外，未加）。

## 2026-10-29 补充（Round 18：卡片 hover 的 GPU/CPU 峰值收敛）

> 用户反馈：快速晃动鼠标在不同卡片间来回滑动时 GPU/CPU 占用飙升（首页 + 附近的人 /
> 附近群组 / 附近瞬间 / 我的瞬间 / 我的群组等卡片页）。要求不降低特效。
> 方案契约见 design-system §21.14；证据与脚本在 `.workwork/r18-hover/`（未入库）。

### 一、定位（可复现）

- 工具：agent-browser 驱动 headless Chrome（真 GPU，ANGLE Metal M1），CDP profiler trace
  分线程归因 + 逐进程 CPU 时间采样（renderer 全进程 / GPU 进程）+ 页面 rAF/长任务。
- 场景：1440×1000 @DPR2，拟真晃动 = 卡内小幅往返为主 + 5–6 次跨卡/s（每次跨卡触发
  进出各一套 hover 动画），6s 窗口 ×3 轮交替采样；对照组为同一页面滚到底再回顶（热身），
  避免懒加载图片解码污染。
- 结论：稳定复现「每次 hover = 4 条动画」，其中**整卡覆盖按钮（`[data-cover]`）背景色过渡**
  与**卡片 box-shadow 过渡**是整卡面积的 paint 动画，每帧重绘整卡；空闲 GPU 进程 CPU 仅
  0.013–0.02 cores，晃动时 0.14–0.15 cores。中等窗口下 renderer 0.18 cores。
- 归因对照：禁用全部 transition/animation → GPU 0.093 cores；仅把覆盖按钮底色改成
  合成器 opacity → GPU -13%；再叠加卡片/配图常驻合成层 → 合计 -27%。给卡片套 DOM
  包装层改「预绘阴影 + opacity」的方案实测反而 +5%（整卡层合成开销 > 阴影重绘），已弃用。

### 二、整改（不降低特效，视觉等价）

| 改动 | 位置 | 说明 |
|---|---|---|
| 覆盖按钮 hover 底色改伪元素 opacity | `styles/theme/utilities.css`（无 layer 区） | `[data-cover]::before` 用同一 `--cst-interactive-hover-fill`、同一 160ms；`[data-cover]:hover` 自身底色透明。卡片与 ListItem 覆盖按钮共用，视觉等价 |
| 卡片/配图 fine pointer 常驻合成层 | 同上（`.card-hover`、`.image-hover-frame__image` 段） | 仅在 `(hover:hover) and (pointer:fine)` 下 `will-change: transform`；批量小图标（`.hover-btn/.hover-action`）红线不变 |
| 死类清理 | `nearby/groups/shared/GroupCard.svelte`、`nearby/people/UserCard.svelte` | 删 `hover:shadow-raised-sm`（被 `.card-hover:hover` 的 `!important` 覆盖）与 `transition-all`（被 `.card-hover` 过渡覆盖） |
| 契约守卫 | `styles/__tests__/perf-contract.test.ts` | 新增「卡片 hover 只走合成器属性（R18）」3 条断言 |

### 三、实测（worktree 8898 vs 主环境 8888，同 harness 交替 3 轮，DPR2）

| 指标 | 改前 | 改后 | 变化 |
|---|---|---|---|
| GPU 进程 CPU | 0.141–0.146 cores | 0.101–0.109 cores | **-27%** |
| renderer 全进程 CPU | 0.182–0.187 cores | 0.177–0.182 cores | -3% |
| 空闲 GPU | 0.013–0.016 cores | 0.013–0.016 cores | 不变 |
| Chrome RSS | 基线 | ±4MB | 无常驻层膨胀 |
| 滚动 GPU（1440 DPR1） | 0.127 cores | 0.090 cores | -29% |
| hover 视觉增量（/me/moments 单卡） | — | 178,601 vs 178,622 px（重叠 178,586） | 视觉等价（差异 3,294px=1.8%，亚像素抗锯齿） |

### 四、门禁（R18 worktree 8898 实测）

- `bun run check` **0 error / 0 warning**（svelte-check + eslint）；`check:theme` 四联全绿；
- `bun run test` **117 files / 642 tests 全绿**（新增 perf-contract 2 条）；`bun run build` 通过；
- `audit:a11y --base http://localhost:8898` **35 页 0 违规 PASS**；
- `audit:touch --base http://localhost:8898` **35 页 PASS**：890 元素 / 分母 810 / 通过 763 / 登记例外 47 / 覆盖率 94.2% / 未登记离群 0（本轮无触控几何变化，纯 hover 执行路径优化）；
- `browser:qa --base http://localhost:8898 --mobile` **23/23 页 issues=[]**（桌面 18 + 移动 5，无控制台错误）。

### 五、仍开放（非本轮范围）

- renderer 主线程成本主要来自「每次 hover 状态变更」本身（命中测试 + 样式重算 + 合成提交），
  约 0.12–0.14 cores 无法通过 CSS 属性选择消除；如需进一步下降，需要减少 hover 命中面
  变化频率（交互改版）而非特效实现。
- 极端快速整卡跳变（鼠标逐帧在两卡中心间跳）实测仍会短时拉高 GPU（动画重启频率上限），
  已从 0.19 降到 0.14 cores 量级。

## 2026-10-29 输入类 UI 重构（搜索框 / 表单输入 / input-surface）

**范围**：`forms/SearchInput`、`forms/Input`、`input-surface` 表面体系及其消费方（搜索页/消息/关系/我的群组/管理后台/地图面板/弹层/登录/设置/编辑资料），跨端同步 `harmonyapp SearchPageHeader`。改前证据：`.workwork/search-redesign/{before,audit-shots,audit-data,audit.md}`（72 个可见输入实例的实测盘点）。

### 改前基线（2026-10-29 实测）

- 72 个可见输入实例 → **9 档高度**（32.66–44px）、**3 档字号**（11/12/14）、8 种 padding；搜索框两派圆角（hero 胶囊 / 其余 16）。
- 聚焦前后边框对比仅 **1.24–1.45:1**（<3:1），浅色主题下肉眼不可辨；axe 不覆盖该项。
- placeholder 叠 0.6 透明度 → 实测 **2.43–4.0:1**（<4.5）；清空钮 25×25 无 hit-slop。
- vibrant 给所有 `.input-surface` 加 `backdrop-filter: blur(16px)`（大面积毛玻璃，违反性能红线）；搜索页传非 token 的 `shadow-xs`。

### 改后契约（详见 `docs/ui-components.md`「输入族契约」/ skill `hotspot-frontend-style`）

- 搜索 = 胶囊 39px（`px-3.5 py-2` + `text-md`）/ 图标 16 / 清空钮 32px 视觉 + hit-slop 44 / `ariaLabel` 按页面语义 / 清空回焦 / Esc 清空 / ⌘K 提示；表单 = 16 圆角 43px（`px-4 py-2.5` + `text-md`）。
- 表面单一来源 `.input-surface`：静态字段底 + 1px border；hover 升 border-strong；focus accent 1px + 内缘 1px（视觉 2px）+ 3px 柔光环；error danger + danger 光环；disabled 弱化底；placeholder `text-secondary` + `opacity:1`。
- 原语消费方不得传 padding/圆角/字号覆盖；裸 `.input-surface` 只允许 43px（`px-4 py-2.5`）/ 39px（`px-3.5 py-2`）两档，textarea 走内容 padding；`/search` hero 桌面 `max-w-3xl`（布局例外已登记）；主题文件不得覆写输入表面；⌘K / Ctrl+K 全局唤出搜索（决策抽到 `lib/search-shortcut.ts` + 9 条单测；弹层打开/非搜索页表单编辑中让位，已在搜索页则聚焦全选）。弹层内 Esc 先清空输入、空输入再交给弹层关闭（`stopPropagation`），避免「清空 + 连关」；Enter/Escape 已加 IME `isComposing` 防护。

### 验收（2026-10-29，worktree 独立环境 8889，最终版本）

- **静态检查**：宿主机全量 `bun run check`（`svelte-kit sync + svelte-check --tsgo + eslint + check:theme 四联`）→ svelte-check **0 error / 0 warning**、eslint 通过、主题一致性/跨端色彩/跨端刻度/类名有效性四联全绿。
- **测试**：`bun run test` **119 files / 670 tests 全绿**（新增 `input-surface-contract` 9 条、SearchInput 11 条、Input 17 条、`search-shortcut` 9 条；既有用例全部保持）。
- **构建**：`bun run build` 通过（先按 frontend/AGENTS.md 停掉 frontend 容器，避免与 dev server 共享 `.svelte-kit`；adapter-static 写出 `frontend/build/`，日志见主工作树 `.workwork/search-redesign/build.log`）；harmonyapp `assembleApp --no-daemon` **BUILD SUCCESSFUL**（两轮：首次同步 + 聚焦色/投影/静态边修正后复审；日志见主工作树 `.workwork/search-redesign/harmony-build.log`，CI 不覆盖鸿蒙，tag 前仍需人工确认本机可编）。
- **a11y**：`audit:a11y --base http://localhost:8889` **PASS**，35 页 0 违规（serious/critical/moderate/minor 全 0；报告 worktree `.workwork/browser-check/a11y-20261029-134450`）。
- **触控**：`audit:touch --base http://localhost:8889` **PASS**，35 页 / 890 元素 / 计入分母 810 / ≥44 通过 763 / 登记例外 47 / **未登记离群 0** / 覆盖率 94.2%（报告 `.../touch-20261029-134751`）。
- **视觉 QA**：报告主工作树 `.workwork/search-redesign/qa-after/qa-after.md` + **56 张截图**（P2 修复后按运行态快照 `a1e380ec…` 重截聚焦/hover 与 12 张主题矩阵）；主体快照 diff sha `9fb46e36…`（13:44 冻结）；三主题×明暗、桌面 1440×900 + 移动 390×844、焦点/已填/键盘焦点/hover/disabled/弹层/登录错误态全覆盖；全页 overflowX=0、console error 0。指针聚焦曾被 hover 规则（特异度更高）覆盖为 border-strong，已加 `:not(:focus-within)` 修复并由 QA 复测通过（`qa-after.md` §12）。
- **四个原始问题对照（改前 → 改后）**：满宽条带 1116px → 768px；聚焦边框对比 1.61:1 → **5.35–7.11:1**（accent 边框 + 视觉 2px + 3px 光环）；placeholder 2.43–4.0:1 → **5.36–8.84:1**；清空钮 25×25 → **32×32 视觉 + hit-slop 44 命中**。
- **⌘K 实测**：5/5 入口页（/、/messages、/me/edit、/settings、/admin/users）→ `/search` 且输入框获焦；搜索页再按一次全选；/me/edit 编辑中与弹层打开时正确让位（首轮 13:28 偶发的「焦点落 main」已由导航后 rAF 显式补聚焦消除）。
- **环境事故与恢复**：dev 容器运行时在宿主机跑 `bun run check/test/sync` 会用宿主相对路径重写 `.svelte-kit/generated/server/internal.js`，导致容器 SSR 全站 500（本轮共 6 次，均用容器内 `bunx svelte-kit sync` 恢复）。结论写入本页：**dev 容器在跑时，check/test/build 只能走容器内命令，或先停 frontend 容器**；该纪律已同步 QA 报告 §11。

### 已知遗留 / 显式决策（2026-10-29）

- **静态字段边界对比 1.30–1.57:1（<WCAG 1.4.11 的 3:1）——显式接受**：这是全站 hairline 边界语言的一部分（卡片/分隔线同档，改前同值甚至更弱）；字段识别由 label + placeholder（5.36–8.84:1）+ 焦点环（accent，视觉 2px，对比 5.35–7.11:1）承担。若后续无障碍评审要求升档，改 `semantic.css` 一处 `--cst-field-border: var(--cst-border-strong)` 即可（视觉略重，已备选）。
- mobile 14px 字段在 iOS Safari 聚焦会触发自动放大（<16px 的确定性平台行为）；受全站 9 档字号约束不做 16px 字段，需真机复测观感。
- 地图移动端主搜索由手写 44px 统一到 39px 搜索档（输入框不在 `audit:touch` 扫描集；39px 为全站搜索统一档）。
- admin 原生 `<select>` 实测 37px vs input 39px（2px，原生控件高度不可精确控；同排视觉无明显落差，注释已改为「≈39px 档」）；hero 实际用 `max-w-3xl`(768)，与设计提案 720 的差异按 Tailwind 标准档取舍记录。

## 2026-10-29 第二轮：同类型 UI 全局一致性收口（搜索/输入族 + 徽章/按钮/浮层）

**触发**：用户「一定要保持所有地方的搜索框风格都一致，任何相同类型的 UI 组件都应该在所有地方保持一致」。

### 盘点（`ui-inventory`，inventory.md 434 行 + 61 截图 + 57 份 DOM census）

- **搜索族**：11 文件 12 处 `SearchInput` 实测同形同尺寸（39px/rounded-full/text-md 14/图标 16/清空钮 44 命中），仅宽度按上下文覆盖；无手写搜索控件、无遗漏 ✅。
- **输入族**：单行输入 4 档高度（39 裸 compact / 41 三处 / 43 原语 / 44 聊天条例外）、Textarea 12 处 5 种壳 padding × 3 种行高、Select 6 处裸原生（37px）、Checkbox 5 处原生（12–13px、无 token）、图片上传 3 套触发 UI。
- **其他族**：Badge 三档高度 14/20/24（手写 h-5 17 处 + h-6 7 处；旧 `.badge` 14px 与 AGENTS 徽章契约冲突）；同语义按钮两形态（设置页 33px 手写胶囊 vs Button 40；me/edit 桌面 40 胶囊 vs 移动 44 矩形）；按钮 `rounded-full` 覆盖 19 处、`rounded-sm` 破档 1 处；底部浮层 3 套圆角（28/28/0/0、28/28/24/24 混用、24 浮层）；通知头像手写 `<img>`、InviteMembers 加载/空态手写；未注册 `rounded-2xl` 3 处。

### 新增/增强原语

- `forms/Textarea`：统一 `px-3.5 py-3` 壳 + `footer` 壳内计数；`forms/Select`：`field`（43/39 + chevron）/`ghost`（行内排序）；`forms/Checkbox`：原生 + `accent-accent-default` 语义色 + focus-ring。
- `forms/Input`：新增 `density`（default 43 / compact 39）、`reserveErrorSpace`（错误行常驻防跳动）、`suffix`（密码显隐）、`type=date`。
- `data-display/Badge`：sm 20px（h-5 px-2 text-xxs）/ md 24px（h-6 px-2.5 text-xs）+ `tone` 枚举 + `style` 逃生口（仅地图 marker 动态 token 色）。
- `ui/button` 新增 `shape: 'rect'|'pill'`（派 A / 派 B·C，禁止 class 覆盖 `rounded-*`）；`ui/dialog` 新增 `floating`（浮层卡片 bottom sheet 唯一实现）；`shared/UnreadBadge` 新增 `variant: 'accent'|'inverse'`。

### 迁移与收口（25 + 44 文件）

- 输入族 25 文件：auth 登录/注册 7 字段、settings 安全设置 6 字段、me/edit、users/[userId]、admin 8 页、群公告/群设置/新建群组/创建瞬间/瞬间编辑弹层、nearby/people 排序（ghost）。
- 其他族 44 文件：Badge 24 文件（旧 `.badge` 清零）；Button 圆角覆盖清零（派 B/C chip 改 `shape="pill"`、手写 accent 胶囊 CTA 回归 Button、`LoadMoreButton` 视觉守恒）；3 处用户操作浮层改 `Dialog.Content floating`；NotifyRow 手写头像改 `Avatar`（类型角标保留外层，注释说明理由）、ChatListItem 未读改 `UnreadBadge inverse`；压图角标改 `.media-badge`；`rounded-2xl` 归一到 `rounded-lg/md`；MockBadge 改用 Badge。

### 强制一致性门禁

- `scripts/check-component-consistency.mjs` **10 条 → 16 条规则**：新增 `raw-badge`/`button-radius-override`/`untiered-radius`/`handwritten-cta`/`raw-avatar-img`/`sheet-radius-override`；已挂入 `bun run check`（CI frontend job 拦截）。
- allowlist **10 条**（聊天/评论输入条、气泡内联编辑、头像/封面 file input、搜索页选择器字符串、ImageViewer 浮层 chrome、地图筛选 CTA），每条带 `reason`，见 `frontend/scripts/ui-consistency-allowlist.json`。

### 验收（2026-10-29，8899 独立环境）

- `bun run check`：svelte-check **0/0** + eslint + theme 四联 + **16 条规则 0 违规（扫描 321 文件）**。
- `bun run test`：**122 files / 691 tests** 全绿；`bun run build` 通过。
- `audit:a11y`：35 页 **PASS，0 违规**；`audit:touch`：35 页 **PASS，未登记离群 0，覆盖率 94.2%**（报告 `.workwork/browser-check/a11y-20261029-151102` / `touch-20261029-151102`）。
- **搜索一致性实测**（Lead 直测）：8 个页面级消费方（/search、/messages、/me/relationships、/me/groups、admin users/groups/tags、/map 桌面）+ 地图移动 + 搜索页移动 = 全部 **39px / 9999px 圆角 / 14px 字**（弹层消费方由原语保证）。
- **Badge 实测**：admin/users 7 个徽章全部 20px；群详情 Badge 20px + media-badge 20px；源码旧 `.badge` **0 处**、`rounded-2xl/3xl` **0 处**；`shape="pill"` 11 处；floating sheet 实测 **24px 圆角 / 448 宽 / 水平居中 / 距底 16px**。
- 说明：本轮 QA/审查 teammate 浏览器长任务连续失败（3 次，无产出），上述审计与实测均由 Lead 直接执行；截图证据在 `.workwork/ui-consistency/{lead-qa,shots,raw}`，盘点清单在 `.workwork/ui-consistency/inventory.md`。

### 已知取舍（显式接受）

- `NotifyRow` 通知类型角标保留外层绝对定位（Avatar `overflow-hidden` 会裁切 18px 角标，已注释；后续可给 Avatar 加 overflow-visible 变体再迁）。
- `MockBadge` 由「白底 + warning 描边」改为 warning 软底（Badge tone 枚举统一）；announcements「严重」由实底红改 `danger` 软底（全站徽章统一软底语言；solid 仅 accent「已登录」档）。
- 地图 marker 4 处动态 pin 色用 Badge `style` 逃生口（值仅 CSS token 变量，禁止 hex/rgba）；admin 表格行内 27px/12px 按钮（ROW_BTN）为既有登记例外，不属动作按钮契约。


## 2026-10-29 第三轮：卡片交互修复 + 瞬间详情页改版

**触发**：用户反馈 ① 首页卡片点「已关注」却播放打开动态动画；要求所有卡片 UI 响应逐个过一遍；② 瞬间详情页顶部有明显横条、评论输入区格格不入、图片比例失真；要求联网调研后改。

### 调研依据（联网抓取，2026-10-29）

- [MUI Card](https://mui.com/material-ui/react-card/)：卡片次级动作要 "stand detached from the main action area in order to avoid event overlap" → 卡内动作必须是与整卡覆盖层分离的真按钮。
- [NN/g Cards](https://www.nngroup.com/articles/cards-component/)：整卡可点要有清晰 signifier；卡内可有 secondary clickable elements，但层级要区分。
- [Bootstrap stretched-link](https://getbootstrap.com/docs/5.3/helpers/stretched-link/)：拉伸链接不推荐叠加多个 tap target，需要 z-index/定位处理。
- [NN/g Sticky headers](https://www.nngroup.com/articles/sticky-headers/)：持久页头只有 unobtrusive、高对比、动效克制且确有必要才成立；可考虑 partially persistent 或不用。
- [MDN object-fit](https://developer.mozilla.org/en-US/docs/Web/CSS/object-fit)：`cover` 保持比例填满但会裁剪；`contain` 保持比例完整显示（可留边）。
- [web.dev aspect-ratio](https://web.dev/articles/aspect-ratio)：用固有比例/占位避免重排。
- [NN/g 移动输入清单](https://www.nngroup.com/articles/mobile-input-checklist/)：移动输入需满足触控/标签/可见性等 14 项。
- [NN/g Image-focused design](https://www.nngroup.com/articles/image-focused-design/)：大图需被正确优先级组织，而非无约束铺满。

### 卡片交互：根因与修复

- 根因（8888 旧版实测）：作者行外层 `pointer-events-none`，「已关注」是不可点 `span`（无 `pointer-events-auto`）；真实指针点击 hit-test 落到整卡覆盖按钮（agent-browser 实测提示 "covered by button.absolute.inset-0"）→ 打开详情。
- 修复：HomeFeedItem / MomentCard 关注态改 `ui/FollowButton size="chip"`（真按钮 + `pointer-events-auto` + `aria-pressed` + 「取消/关注 + 名字」label + 乐观态 + toast + `stopPropagation`）；默认只在已关注作者卡显示，取关后就地保留「关注」，未关注作者卡不新增常驻按钮。
- 真实指针复测（8897）：点「已关注」→ 路径保持 `/`、chip 变「关注」并出 toast；旧版同点位命中覆盖层（打开详情）。

**卡片响应审计矩阵（真实指针 + 代码核对）**

| 页面/卡片 | 控件 | 预期 | 结果 |
|---|---|---|---|
| 首页信息流卡 | 头像/昵称 | 进作者主页 | ✅ |
| 首页信息流卡 | 已关注/关注 chip | 取关/关注，不导航 | ✅（本轮修复） |
| 首页信息流卡 | 点赞/收藏/分享 | 仅切换状态，不导航 | ✅ |
| 首页信息流卡 | 评论计数 | 进详情评论区 | ✅ |
| 首页信息流卡 | 标签 | 进搜索 | ✅ |
| 首页信息流卡 | 封面/其它区域 | 进详情（cardPress 动画） | ✅ |
| 附近的人卡 | 关注图标钮 | 切换，不导航 | ✅（真实点击） |
| 附近群组卡 | 加入/已加入 | 切换，不导航 | ✅（真实点击） |
| 附近瞬间/资料瞬间卡 | 作者/点赞/评论/收藏/分享 | 作者→主页、动作不导航、评论→详情 | ✅ |
| 所有卡 | 行内按钮按下 | 不触发整卡 press 缩放 | ✅ |

### 瞬间详情页改版（改前 8888 → 改后 8897 实测）

| 项 | 改前 | 改后 |
|---|---|---|
| 页头 | 平板滚动后 sticky glass 带 `h=79 / border-bottom 1px`，内容从横杠下穿过 | `PageHeader variant="floating"`：透明无底边/阴影（bg rgba(0,0,0,0)、border 0、shadow none，h=52–58），仅悬浮返回/更多圆钮，标题 sr-only |
| 单图比例 | 固定 `h-cover-height-lg`（692×380）+ `object-cover`：渲染比例 1.821 vs 原图 1.488（裁切/观感失真） | `object-contain` + 自身比例 + `max-h(min(60vh/26rem) 移动 / min(72vh/38rem) 桌面)`：moment_1 1.487 vs 1.488、moment_4 1.626 vs 1.629（误差 <0.2%） |
| 评论输入（桌面） | 固定底栏宽 804 > 卡片内容宽 692（错位 56px），白底 + 1px 上边线 | 评论区顶部 `inline` 输入器：宽 692 = 卡片内容宽、radius 16、软底浅槽、无固定定位 |
| 评论输入（移动） | 底部两条通栏：动作条 48 + 输入条 64（合计 ≈112px chrome，`border-t` 硬边） | 动作条内嵌内容；仅一条浮动 dock `366×~56`（inset 12 / 底 12 / radius 16 / 细边 + 玻璃底 + 投影），发送钮 40px（与聊天统一） |
| 顶部投入 | 页头 + 内容卡两段割裂 | 内容卡从画布直接开始，仅两个圆钮悬浮，无整条 chrome |

- 同轮收口：CreateSheet「谁可以看到」两列自绘按钮（49px 扁条）→ `TabBar variant="segmented"`（与 MomentEditDialog 同形）。
- 代码：`components/ui/FollowButton.svelte`（chip/onMedia/pointer-events-auto）、`components/home/HomeFeedItem.svelte`、`components/moments/MomentCard.svelte`、`components/layout/page/PageHeader.svelte`（floating）、`routes/(app)/moments/[momentId]/{+page,MomentPostBody}.svelte`、`shared/CommentInputBar.svelte`、`components/overlays/CreateSheet.svelte`。
- 测试：HomeFeedItem 10、MomentCard 6、PageHeader 7、MomentPostBody 3、CommentInputBar 4 全绿（新增交互/媒体契约 13 条：HomeFeedItem +2 / MomentCard +2 / PageHeader +2 / MomentPostBody +3 / CommentInputBar +4）；全量 check/test/build 见提交信息与 CI。

### 已知取舍

- 详情单图未用固定 aspect-ratio 占位（API 无 width/height 字段）；首屏以 min-height + 渐显承载，后续 API 下发尺寸可升级为精确 ratio 占位消除 CLS。
- 移动端悬浮 dock 与键盘联动沿用 SplitPageLayout footer 机制；安全区由 footer 容器负责。


## 2026-10-29 第四轮：详情页工具条改为随正文滚走（用户反馈「返回/更多压住内容」）

**反馈**：滚动到评论区时，返回 / 更多两个悬浮圆钮叠在图片上，读起来像内容重叠，位置不合适；问有没有更好的方案。

**候选方案对比**：

| 方案 | 优点 | 缺点 | 结论 |
|---|---|---|---|
| A 永久吸顶透明悬浮（旧实现） | 返回常驻、沉浸 | 内容从按钮下穿过，重叠感最强（用户本次反馈） | 否 |
| B 吸顶 + 不透明画布底 / 玻璃条 | 不重叠、返回常驻 | 形成用户上一轮已反馈过的「顶部横条」，chrome 常驻 | 否 |
| C 页头留在文档流、随正文滚走（采用） | 零遮挡、无横条、内容优先；顶部仍保留工具条位置 | 滚动后返回走浏览器/系统返回与全局「回到顶部」 | ✅ 采用 |

**实现**：

- `SplitPageLayout` 新增 `headerSticky?: boolean`（默认 `true`）：为 `false` 时页头包装容器不吸顶，留在文档流随正文滚走；默认值不变，所有既有页面行为完全不受影响；
- `PageHeader variant="floating"` 支持 `sticky={false}`（标题仍 `sr-only`，返回/更多按钮不变）；
- 瞬间详情页同时传 `headerSticky={false}` + `sticky={false}`；其余页面继续默认吸顶。

**实测（8897，1440×900 / 390×600）**：

- 页头 computed `position: static`（非 sticky）；滚动后返回按钮 `top=-319`（桌面）/ `top=-261`（移动），完全滚出视口；
- 滚动状态页面内无任何悬浮按钮覆盖图片/评论（改前截图可见按钮叠在图片上）；
- 顶部初始状态不变：返回/更多在内容卡上方的画布工具行，不重叠；
- 上下滚动、评论区输入/发送、动作条均正常；其他页面（首页/附近/用户详情等）吸顶行为未变。

**测试与门禁**：PageHeader 9 tests（新增 2 条 `sticky={false}` 契约）；`bun run check` / `bun run test`（704+）/ `bun run build` 全绿；a11y 40 页 0 错误、touch PASS（见 `.workwork/browser-check/` 最新报告）。

**取舍**：滚动后不再有持久返回入口，依赖浏览器/系统返回与全局「回到顶部」（1.5 屏后出现）。若后续希望在长评论页常驻返回，可再做「下滑隐藏、上滑浮出单枚返回钮」的渐进方案——本轮不做，避免重新引入重叠。


## 2026-10-29 第五轮：详情页工具条改为整行悬浮玻璃面板（用户反馈「只有两个按钮悬浮很奇怪」）

**反馈原文**：「可以做成这一行区域都是悬浮的，只有这两个按钮是悬浮的话就很奇怪，要的是一个和谐感」。即：不要再让返回/更多两枚圆钮各自孤零零地浮在内容上，工具条本身要有统一的悬浮承载面。

**方案对比**：

| 方案 | 结果 |
|---|---|
| 两枚独立悬浮圆钮（第 3 轮） | 按钮可见，但缺乏承载面，滚动/静止时都显得零散；用户本次反馈的主体 |
| 工具条随正文滚走（第 4 轮临时方案） | 零遮挡，但返回入口会消失，且「悬浮」感被削弱 |
| **整行悬浮玻璃工具条（采用，第 5 轮）** | 返回 + 标题 + 更多同处一块圆角半透明面板（对齐内容卡宽度、吸顶），滚动时内容透过毛玻璃承接；视觉上是一个完整 chrome 对象，不再像两枚补丁 |

**实现**：

- `PageHeader variant="floating"` 重做：整行渲染 `[data-floating-toolbar]` 面板——`rounded-md / border-border-default/60 / bg-surface-primary/85 / backdrop-blur-md / shadow-raised-sm`，左返回、中标题（可见、居中、truncate）、右更多；返回/更多仍为 42px 圆钮。
- 恢复吸顶：撤回第 4 轮的 `headerSticky={false}` / `sticky={false}` 使用（`SplitPageLayout` 的 `headerSticky` 扩展一并撤回，保持布局原语最小面）；详情页回到默认 sticky。
- 不加通栏背景/顶雾：工具条自身是完整悬浮面，避免再次形成「横条」观感；页面在顶部时保持画布透明。
- 其他页面（默认 PageHeader）零变化。

**实测（8897，vibrant light/dark）**：

- 桌面：面板 748×52 @ x312，与内容卡（748 @ x312）严格同宽同轴；radius 16px、背景 85% surface-primary、backdrop blur 12px、1px 细边；标题「瞬间详情」可见居中；滚动 400px 后面板吸顶 y=72，内容从下方透出。
- 移动：面板 358×52 @ x16，与内容卡同宽；顶部 y=8，无横向溢出。
- 深色：面板底色 `oklab(0.2467 … / 0.85)`、标题 `rgb(232,241,255)`，对比清晰。
- 交互：返回点击回退成功；更多菜单展开「复制链接 / 编辑动态 / 删除动态」，锚点正常。
- 作用域：仅 `variant="floating"` 详情页；其他页面 header 不受影响。

**测试与门禁**：`PageHeader` 7 tests（floating 面板契约 2 条：唯一面板、返回/标题/更多同容器、可见标题）；16 条源码一致性检查器 OK；`bun run check` / `bun run test` / `bun run build` 全绿；a11y 40 页 0 错误、touch PASS（见 `.workwork/browser-check/` 最新报告）。

**历史说明**：第 4 轮「随正文滚走」方案已被本轮取代，保留其记录用于追溯；当前详情页契约以本轮为准（见 `docs/ui-components.md`「详情页契约」）。


## 2026-10-29 第六轮：详情页去掉横栏 + 画廊级图片展陈（用户反馈「要融合感/无横栏/图片留白难看」）

**反馈原文**：「要有融合感，同时也不要有明显的横栏，联网搜索下，给我更好的方案。而且这个图片显示的也不好，两边还有留白，展示图片的时候要像画廊一样，要有艺术感、设计感、审美感」。

### 联网调研（2026-10-29 抓取）

| 来源 | 关键结论 | 对本案的指导 |
|---|---|---|
| [MDN object-fit](https://developer.mozilla.org/en-US/docs/Web/CSS/object-fit) | `contain` 保持比例完整显示，但比例不匹配时会 letterbox/pillarbox；`cover` 保持比例填满但会裁剪 | 画廊主图必须 `contain` 保真；留边用「同图模糊铺底」承接而不是空白 |
| [web.dev aspect-ratio / object-fit](https://web.dev/articles/aspect-ratio) | 用固有比例承载媒体；`contain` 存整图、`cover` 会在低边界放大裁剪 | 主图 `w-full + h-auto + max-h + object-contain`，高度由比例决定 |
| [NN/g Photos as Web Content](https://www.nngroup.com/articles/photos-as-web-content/) | 用户会仔细审视照片，照片是内容而非装饰 | 详情页把图片升级为内容卡首屏 hero，而不是正文里的缩略配图 |
| [NN/g Image-Focused Design](https://www.nngroup.com/articles/image-focused-design/) | 大图能吸引注意，但必须被正确优先级组织 | 图片优先、控件让位：去掉常驻 chrome，工具行退成无表面的一行 |
| [NN/g Product Photos on Listing Pages](https://www.nngroup.com/articles/product-photos-listing-pages/) | 图片应够大够清楚；如 Helmut Lang 案例中「照片周围大量空白」会让用户难以辨认内容 | 禁止两侧空白条/空底色，主图必须尽量铺满展台 |
| [NN/g Sticky Headers](https://www.nngroup.com/articles/sticky-headers/) | 持久页头只有在不抢内容、chrome 占比最小时成立；要考虑「是否真的需要 sticky」 | 详情页不再使用吸顶横栏/玻璃工具条；工具行随内容滚走 |
| [MDN blur()](https://developer.mozilla.org/en-US/docs/Web/CSS/filter-function/blur) / [backdrop-filter](https://developer.mozilla.org/en-US/docs/Web/CSS/backdrop-filter) | filter 可对元素本身模糊；backdrop-filter 处理元素背后区域 | 用同图模糊衬底（ambient matte）填充比例差，而不是露空底色 |

### 方案对比

| 方案 | 问题 |
|---|---|
| 悬浮玻璃横栏（上一轮） | 无论多圆润，仍是一条明显的横向 chrome；用户明确要求「不要明显横栏」 |
| 两枚孤立悬浮圆钮（更早） | 无承载面、零散（上一轮反馈） |
| **内容内极简工具行 + 画廊 hero（采用）** | 工具行与画布同底、无表面、非吸顶，不构成横栏；图片作为首屏 hero 整幅出血；返回/更多仍在首屏可见，滚动后交给浏览器/系统返回与全局回顶 |

### 实现

- **页面**：移除 `PageHeader`/header snippet；内容顶部改为一条 `ghost` 工具行（返回 + 居中可见 `h1`「瞬间详情」+ 更多，36px，无背景/边框/吸顶，随内容滚走）；加载/错误/空态同样保留；更多菜单逻辑抽为 `moreMenu` snippet；骨架屏同步改为「hero → 作者/正文 → 动作条 → 评论区」零位移结构。
- **图片（MomentPostBody）**：媒体区移到卡片首屏（作者/正文后移）；单图 hero：
  - 左右/上方负 margin 出血到卡片边缘（`-mx-4 sm:-mx-5 md:-mx-7 -mt-4…`），卡片 `overflow-hidden` 裁出圆角；
  - 主图 `w-full h-auto max-h-[min(78vh,44rem)] object-contain`，保持原比例、不裁切；
  - 同图 `scale-110 object-cover opacity-soft filter:blur(var(--cst-blur-lg))` + 既有 `--cst-surface-media-scrim-card` 作环境衬底：宽图铺满无留白，竖图两侧显示同图模糊展墙，不再出现空底色；
  - 顶部/底部/圆角/阴影保持卡片语言；多图首图 hero + 下方 3 列缩略图（方图 `object-cover` 语义不变）；
  - 点击主图/缩略图仍打开 ImageViewer。
- **PageHeader `variant="floating"`** 保留为通用沉浸页原语（当前详情页不再使用），文档已注明。

### 实测（8897，vibrant light/dark，1440×900 / 390×844）

| 项 | 结果 |
|---|---|
| 横栏 | 无 PageHeader/无横栏；工具行 36px、同画布底、非吸顶；标题可见居中；滚动后随内容离开 |
| 主图宽度 | 748px = 内容卡整宽（旧实现为 692px 内缩 + 部分比例两侧留白） |
| 比例 | moment_1 渲染 1.488 vs 原图 1.490；moment_2 1.500 vs 1.501；moment_3 1.500 vs 1.501；moment_4 1.626 vs 1.627（误差 <0.2%，不裁切） |
| 竖图（注入 600×900 测试图） | 主图框 748×702，两侧由同图模糊衬底填满，无空底色 |
| 布局顺序 | 图片 hero → 作者/位置/时间 → 正文 → 标签 → 动作条 → 评论；骨架与真实结构同序 |
| 主题 | light/dark 均验证；无横向溢出、无布局抖动 |
| 交互 | 返回、更多菜单（复制链接/编辑/删除/举报）、图片查看器均正常 |

**测试与门禁**：MomentPostBody 4 tests（新增画廊主图/模糊衬底契约）；`bun run check` / `bun run test` / `bun run build` 全绿；a11y 40 页 0 错误、touch PASS（见 `.workwork/browser-check/` 最新报告）。

**历史说明**：第 5 轮「整行悬浮玻璃工具条」已被本轮取代（用户明确不要显性横栏）；当前详情页契约为「内容内极简工具行 + 画廊 hero」，见 `docs/ui-components.md`「详情页契约」。

