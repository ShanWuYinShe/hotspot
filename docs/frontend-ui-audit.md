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
