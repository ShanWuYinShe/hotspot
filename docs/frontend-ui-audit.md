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
| reduced-motion | `set media reduced-motion` 遍历 | ambient 24s/36s、ping 1s、pulse 2s、hover 0.18s → reduced 下 ambient 消失、其余 1e-05s 单次、hover 1e-05s（全局 utilities 规则） |
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

- 桌面页标题字号仍为 20（移动 24）；评审建议桌面提到 24 属跨全站口径变更（涉及 `PageHeader` 契约与 skill 一行描述），本轮未做，登记为后续候选。
- 聊天输入条附件入口：无接口/功能，未加装饰死按钮。

## R14 卡片交互一致性与层叠修复（2026-10-28）

**用户反馈**：① 卡片交互逻辑（hover、压图功能 UI 的交互效果）应保持一致；② 移动端卡片滚到标题栏下方时
z 值不对，卡片上的 UI 显示到标题栏上方（附截图）。

**根因（实测复现）**：`HomeFeedItem` 卡根缺 `isolate`。行内操作按钮经根上 `[&_button…]:z-20` 抬升，
根不是堆叠上下文 → 这些按钮参与页面堆叠上下文；与页内 sticky 条（`--cst-z-sticky=20`）同 z 且 DOM 更晚，
故滚动时卡片作者/操作压住吸顶 tab。实测 `elementsFromPoint(60,80)` 修复前返回卡片按钮、修复后返回吸顶条。

**修复内容**：
1. `HomeFeedItem` 卡/行根加 `isolate`；`ProfileMomentTile`/`ProfileGroupTile` 包装层加 `isolate`（同类陷阱：卡外 z-20 菜单）；
2. 按压统一：新增 `lib/motion.cardPress`，`.card-hover:active` 固定 0.985 → `[data-pressed]` + `--cst-card-tap-scale`；
   `NearbyImageCard` 删除手写 pointer 状态/内联 transform（原 0.94 与 0.985 两套量感）；
3. 操作行统一：新增 `data-display/ImageCardActions`（四类操作、状态语义、pop、aria、md/sm 档）；
   `HomeFeedItem`（卡+行两形态）与 `MomentCard` 全部接入；此前**信息流卡的点赞/评论/收藏只是 div（点击只打开详情）**，
   现为真按钮且失败可回滚（与网格卡同语义）；
4. 压图徽章统一：新增 `.media-badge`/`--icon`，迁移 HomeFeedItem / MomentCard / UserCard / GroupCard / GroupCarousel /
   ProfileGroupTile / users 页日期徽章（原混用 `surface-overlay-strong` 与 `surface-media-scrim-strong`、有无 blur 不一）；
5. 作者区统一：`FeedItem` 增可选 `userId`，MomentCard 作者头像/昵称可点进主页（与信息流卡同 hover 语言）；
6. 分享统一：`lib/moment-actions.shareMoment`，两张卡同一实现；
7. 窄卡适配：`size="sm"` 动作档用于手机 2 列卡（171px 实测四类操作不越界，此前分享/收藏被 overflow 裁切）。

**验证（2026-10-28，R14 worktree env http://localhost:8898）**：
- 移动端 390×844 首页滚动 520px：重叠点首位元素 = 吸顶条（修复前为卡片按钮）；截图 before/after 见
  `.workwork/r14-repro/`；
- 桌面 hover/press 实测：卡 hover 抬升 + 图片 1.045 缩放；按压 `data-pressed=true` → computed transform
  `matrix(0.94,0,0,0.94,0,0)`；行内按钮按下不产生按压态；
- 点赞点击（mock 登录）：`aria-pressed` false→true、计数 30→31、URL 不变（不误触整卡跳转）；
- 作者点击：/nearby/moments → /users/user_lumos；
- 移动 2 列卡操作行右侧边界 ≤ 卡内容右缘（收藏 right=171 / 卡右缘 203）；
- `bun run check` 0 error / 0 warning；`bun run test` 113 文件 / 599 用例全绿（新增
  ImageCardActions / HomeFeedItem / NearbyImageCard 契约测试）；
- a11y（`--base http://localhost:8898 --mobile --reduced`）：**PASS**，40 页 axe 0 违规
  （minor/moderate/serious/critical 全 0），reduced-motion 异常 0（`.workwork/browser-check/a11y-20261028-130501`）；
- 触控（`--base http://localhost:8898`）：**PASS**，35 页 / 890 元素 / 覆盖率 94.2% /
  47 条登记例外 / 未登记离群 0（`.workwork/browser-check/touch-20261028-131436`）。
- 触控修复：动作行按钮视觉 16px（sm 档纯图标）时 `hit-slop-44` 有效热区仅 42px，补 `min-w-4.5` 后 44px 通过。

**验证工具修复（同轮发现）**：`frontend/scripts/browser-a11y.sh` 原先只在循环外执行一次
`ab set media light reduced-motion`，而 agent-browser 的 `open` 打开新标签页会重置 CDP 媒体模拟，
导致 reduced-motion 检查实际在 no-preference 下执行——把 ambient 装饰动画误报为长动画
（40 页 37 页假阳性 + 1 页未返回）。已改为**每页页面稳定后、注入检查脚本前**重设媒体模拟；
修复后 ambient 动画在 reduced 下被全局规则关停，检查恢复可信。
