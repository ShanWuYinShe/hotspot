# UI 一致性改进交接（2026-10-22 ~ 2026-10-27，Round 1-22）

> **口径变更（2026-10-29）**：前端 MSW mock 已移除（MockBadge 组件同步删除），dev 环境改用后端
> 种子数据 + 真实后端。本文中「dev/mock」「mock 登录」「mock 空态不可复现」等历史结论保留原记录，
> 新基线以 seed 环境为准（见根 `AGENTS.md`「开发种子数据」）。

本文件记录 UI 一致性/审美改进的完成基线与待做项，供后续会话直接接手。规则细节见
`.agents/skills/hotspot-harmony-ui/SKILL.md`、`.agents/skills/hotspot-frontend-layout/SKILL.md`；
审计结论见 `docs/frontend-ui-audit.md`、`docs/backend-audit.md`。

## 已完成基线（不要回退）

### 2026-10-27 本轮（Round 22，UI 质量门禁脚本化：axe / 触控 / reduced）

- **依赖**：`axe-core@4.10.2` 进 frontend devDependency（不进 check/build）；`bun.lock` 更新。
- **脚本**：`scripts/lib/qa-common.sh`（AB/node/登录/页面清单/详情发现/输出目录）、`scripts/browser-a11y.sh`
  （逐页 axe、serious/critical>0 或 axe 未注入/加载失败 → exit 1，支持 --base/--theme/--mode/--reduced/--mobile/--limit）、
  `scripts/browser-touch.sh`（390 有效热区扫描 + allowlist，未登记离群 → exit 1，支持 --full/--baseline）、`scripts/audit-all.sh`；
  package `audit:a11y` / `audit:touch` / `audit:ui`（任一失败整体 exit 1，打印报告路径）；`browser-qa.sh` 抽到 qa-common。
- **门禁结果（8898）**：`audit:ui` exit 0 —— QA 18+5 移动 issues=[]；a11y 35 页 0 violations/0 错误；touch 全量 35 页 768 元素、
  分母 688、通过 641、例外 47、覆盖率 93.2%、未登记 0；负向验证（注入无 alt/无名称 → critical + exit 1；未登记离群 → exit 1）通过。
- **顺带修复 --full 发现**：admin/audit 文字链 eff41→45、admin/stats 明细 44×44/热力点 eff54、聊天头部 eff48、share CTA eff52、
  错误页按钮 eff52（视觉不变）；map axe-not-loaded 加重载重试。
- **文档**：frontend/AGENTS.md（命令/前置）、frontend-ui-audit（命令/基线/allowlist 维护）、style skill；默认门禁切全量，--baseline 保留 r5 22 页快速对照。

### 2026-10-27 上一轮（Round 21，聊天滚动策略 + 回底/新消息计数）

- **滚动策略**：近底部（≤120px）才自动滚；上滑阅读收到新消息**不跳转**、按条数累加并显示回底按钮；自己发送始终滚底；
  初始/切会话直接回底；分页 prepend 保持视口；会话切换计数/状态清零。
- **回底按钮**：44px、距底 >200px 显示，仅箭头 /「N 条新消息」（>99→99+），`aria-live`、`aria-label` 与文本一致，150ms 显隐（reduced 0），
  键盘可达（Shift+Tab/Enter），点击滚底+清零+补 markRead，回底 200ms 内隐藏；位置不遮挡输入栏/浮层。
- **markRead 延后**：移除 WS 即时已读；仅初始/近底/回底补标（1s 节流），上滑/离开不补标。
- **鸿蒙镜像**：`ChatJumpToBottomButton.ets`（48vp/99+ 徽标/MotionUtil reduced/禁用+移出 a11y）+ ChatPage 私聊/群聊共用策略
  （`onScrollIndex` 末尾可见近似，无距底像素 API 平台差异登记）；旧 78% 胶囊删除；build/check:theme/parity 全绿。
- **验证**：WS 注入矩阵（1/批量/120/私聊/群聊/1440/390）scrollTop 不变与计数正确；axe 50 runs 0；QA 23/23；CLS 0；test **103 文件/546**；build 通过。

### 2026-10-27 上一轮（Round 20，页头滚动/SPA 焦点播报/回顶/reflow）

- **页头滚动**：sticky PageHeader 顶部透明（无白带）→ 滚动 >4px 玻璃（`surface-page 78% + blur16 + 细底边 + elevation`，150ms，reduced 立即）；
  三主题×亮暗矩阵一致、滚动过程 0 layout-shift、滚动回顶恢复透明；18 处调用无需改；
  并修复既有缺陷：`SplitPageLayout` 桌面分支 header 外层未 sticky（/notify 1440 滚动时 header 滚出）→ 加 `sticky top-0 z-sticky`，复验吸顶+玻璃可见、其它分栏页与 390 无回归。
- **SPA 焦点/播报**：客户端导航后焦点落 `main#main-content`（tabindex=-1、preventScroll、outline none）；`#route-announcer` 播报 h1 标题
  （首屏不播、1.5s 去重、后退/前进恢复）；不聚焦 h1 避免与播报重复；skip link/Tab 顺序不回归。
- **回到顶部**：滚动 >1.5 屏出现 44px 圆钮（桌面 bottom-6 距 MockBadge 28px、移动 bottom-24 距底栏 14px）；平滑回顶（reduced 直接跳）、
  19ms 隐藏防闪烁；键盘可达、不可见移出无障碍树。
- **Reflow**：320px × 34 页 + 200% 文本缩放 × 8 关键页 document/main 横向溢出 = 0（装饰层超界被裁剪）。
- **鸿蒙镜像**：TopBar 玻璃 N/A（grep `.sticky(`=0，页头在滚动容器外）；新增 `BackToTopButton.ets`（48vp、MotionUtil、a11y label），
  Home/Nearby/Notification/Search/MomentsFeed 5 页接入；SPA 焦点/TalkBack、320/reflow N/A；build/check:theme/parity/布局树全绿。
- **验证**：axe 50 runs 0；QA 23/23；CLS home 0.0195 持平/notify 0.0069/me/detail 0、0 layout-shift、0 长任务；test **103 文件/546**；build 通过。

### 2026-10-27 上一轮（Round 19，媒体占位/降级 + 搜索 URL 状态）

- **媒体占位与降级**：Avatar 占位层**始终渲染**（有 name 首字母 / 无 name UserRound，aria-hidden）+ 图片 absolute 覆盖 + 渐显；
  失败保留占位不闪烁；群封面/通知·搜索·rail·主页瞬间缩略图/详情封面与九宫格/群 hero 统一 `.media-placeholder` 主题渐变占位；
  有 Skeleton 列表不叠加；占位文字 accent-bg 上 3 主题×亮暗 ≥4.5（text-primary：vibrant 11.75/9.25、forest 11.58/10.11、inkwash 10.21/8.49）；
  慢网（延迟 3.5s）4 页×2 视口「加载中 vs 完成」无灰块、无 CLS。
- **搜索 URL 状态**：新增 `lib/search-url.ts`（`?query=&tab=&tag=`；people/posts 为内部 Tab、兼容 users/moments 别名；非法 tab→all；空 query 清空全部参数）；
  深链恢复只 1 组搜索请求；`replaceState` 同步不增历史；popstate 前进后退恢复；输入防抖/联想不写 URL；Esc 清空回发现态；
  键盘/防抖/最近搜索不回归。
- **鸿蒙镜像**：`SystemAvatar` 占位始终渲染 + onComplete 渐显（reduced 直接显示）+ onError 保留；通知头像改 SystemAvatar；
  6 处缩略图占位/fade/失败保留；群 hero 渐变登记；搜索 URL N/A；build/check:theme/parity/布局树全绿。
- **验证**：axe 50 runs 0；QA 23/23；check 0/0；test **103 文件/546**；build 通过；CLS search 390 既有 0.0995→0.0077、home 0.0195 持平；0 长任务。

### 2026-10-27 上一轮（Round 18，微动效与状态反馈）

- **动效入口**：新增 `frontend/src/lib/motion.ts`（enter 180/exit 150/pop 220；`prefersReducedMotion/motionDuration/pop/playPageEnter/toastExit`）+ CSS
  `cst-page-enter`/`cst-state-pop`/`cst-comment-enter`；鸿蒙 `MotionUtil` 扩展为唯一动画入口（contentEnter/statePop/popScale/enterOffset，reduced 直接终态）。
- **页面进场（仅客户端导航）**：RoutedContentViewport `afterNavigate` 仅 `nav.type!=='enter'` 播放（首屏不播护 LCP；popstate/前进后退播），
  仅内容层 opacity 0→1 + translateY 4px（180ms）；sticky/滚动/focus 不受影响。
- **骨架→内容 + Toast**：home/notify/messages/search/admin-users 有骨架才 fade（缓存命中不播）；toast 每条 `out:toastExit`（150ms）+ 容器常驻（最后一条可退）。
- **状态 pop**：点赞/收藏/关注/加入 仅激活方向（1→1.18→1，220ms；点击 nonce 驱动，初始/同步不播；取消不 pop）Web 11 处 + 鸿蒙 11 处；
  旧手写 150ms/1.3 模式全部替换；评论新增仅新行 fade+6px（160ms）。
- **reduced 矩阵**：normal vs reduced 双路径（首屏 animationstart 0、导航/pop/toast/评论均无类无事件、computed 1e-05s）；
- **性能**：LCP 同 r6 量级（首屏无动画）、CLS 不回归（home 0.0195 与基线一致、search 390 0.0995 既有 discovery 位移）、0 长任务。
- **验证**：axe 50 runs 0；QA 23/23；check 0/0；test **102 文件/540**；build 通过；鸿蒙 build/check:theme/parity/布局树全绿。

### 2026-10-27 上一轮（Round 17，图片加载管线 + 感知性能 + 搜索键盘导航）

- **图片管线**：新增 `frontend/src/lib/images.ts`（仅 unsplash 加 `auto=format&fit=crop&q&w`）+ 鸿蒙 `ImageUrlUtil.ets`；
  6 档尺寸预设（avatar/thumb/card/cover/hero/full）、srcset/sizes、`fadeImage` 渐显（0.22s，reduced 直接显示）；
  全站 26 个 `<img>` 接入：首屏 feed 前 2/详情首图/hero/ImageViewer eager+high，其余 lazy+async；Avatar 占位+失败首字母。
- **感知性能（transferSize 前后）**：`/` 1440 −62%、`/` 390 −54~64%、`/me` −46~70%、详情 1440 −77.6%、详情 390 −52%；
  CLS 不回归（详情 0.0213→0）；LCP 方向改善（dev 噪声）。`/nearby/people` 1440 +43% 为「封面 200w→400w 清晰度收益」的有意取舍（备选：非首屏回退 thumb）。
- **搜索键盘导航**：`/search` ↑↓ 在输入框/话题/结果行/最近条目间移动焦点（真实链接、focus-ring 可见）、Enter 打开、Esc 清空回输入框；
  未引入 combobox；Tab 顺序不变；axe 复扫 50 runs 0 violation。
- **鸿蒙镜像**：ImageUrlUtil 15 处接入、首屏优先/LazyForEach 策略（无 fetchpriority/lazy 的平台差异已登记）、高频图占位+onComplete 渐显
  （reduced 直接显示）；build/check:theme/parity/布局树全绿。
- **验证**：axe 50 runs 0；QA 23/23；check 0/0；test **101 文件/526**；build 通过；39 页回归全绿。

### 2026-10-27 上一轮（Round 16，三主题验收 + a11y/触控/性能质量门）

- **三主题×明暗验收**：9 页 × forest/inkwash/vibrant × 亮暗对比；**forest light 软岛 v2**（仅 forest light：
  `color-mix(surface-primary 92%, text-primary 8%)` = #E4E6E2，岛/画布 1.00→1.17，岛内 primary 13.11/secondary 6.42）；
  其余 5 组不变（1.11/1.33/1.16/1.06/1.31）；主题相关 color-contrast 4 处修复（设置徽章/瞬间 tag chip/状态徽章）。
- **axe 可访问性清零**：主扫描 50 runs **50/50 页 0 violation**（修复前 23/50；critical 3、serious 22 实例），
  主题附加 36 runs 0 flagged。修复：nested-interactive（整卡改「覆盖按钮 + 行内动作 auto-lift」，51 节点）、
  aria-required-children（列表行 role=listitem）、definition-list（MetricRow → div role=group）、aria-prohibited（13 节点）、
  scrollable-region 可聚焦、landmark（login/register/error 补 main）、image-redundant-alt（头像装饰化）。
- **触控目标**：390×20 页有效热区 ≥44 由 **69.0% → 91.2%**（425/466；离群 41 全为登记例外：地图 marker/控件 19、
  相邻 TagPill 22、表单控件 34、skip link 19——分类见 audit）；新增 `.hit-slop-44`；鸿蒙 responseRegion 补 5 处 + 边缘 tap 实测。
- **性能基线**（dev/mock，仅相对参考）：TTFB 5–12ms、FCP 152–284ms、LCP 180–436ms、CLS 0–0.0195（<0.1）、长任务 0；无需修复。
- **鸿蒙追加**：装饰头像 `SystemAvatar.decorative`（13 处）、11 个 accessibilityText、responseRegion 5 处、
  textTertiary forest light #58645C；build/check:theme/parity 全绿；真机 TalkBack/像素仍待补。
- **验证**：39 页回归全绿；check 0/0、test 100 文件/518、build 通过；独立复核 verdict-r5。

### 2026-10-27 上一轮（Round 15，长尾/后台软岛化 + 交互状态与可访问性基线）

- **长尾/后台软岛化（Web 18 文件）**：瞬间详情单帖卡/媒体容器、首页 onboarding/HeroStats/轮播/附近人列表、
  群详情 hero/指标/操作/子列表、他人主页、**admin 10 页**（模块卡/表格外壳/筛选/危险 callout）统一为软岛
  （无 1px 全框描边 + `surface-island` + shadow-xs + 圆角 20）；表格外壳去描边但表头/行线/行 hover 保留；
  例外清单（表格行线/输入控件/媒体格/hero/分隔线/chip/头像环/弹层/地图面板）登记在 design-system §19.5。
- **交互状态与可访问性基线（design-system §20）**：
  focus-visible 全站可见（web 2px accent + offset 2，键盘 Tab 实测含 admin 表格行按钮/skip link；
  鸿蒙 `FocusRing` focusBox 2vp + 2vp，原语接入）；reduced-motion（web 全局 `prefers-reduced-motion` 覆盖；
  鸿蒙 `MotionUtil`，原语接入；默认行为不回归）；hover/press/disabled 统一；长尾加载/空/错误态核查
  （moment=Skeleton；group/users=InlineLoading 带理由；admin=InlineLoading+EmptyState+LoadError+LoadMoreButton）。
- **验证**：before/after 矩阵各 28 张（7 页×390/1440×亮暗）+ 39 页回归（23 QA issues=[] + 16 路由×2 视口 0 错误）；
  check 0/0、test 518、build 通过。鸿蒙 P0（焦点/减少动态/AdminOverview）+ P1/P2 软岛映射，build/check:theme/parity 全绿；
  **真机仍离线**（Round 13/14/15 端上像素验收均待补）。

### 2026-10-27 上一轮（Round 14，次级页面内容软岛化 + 个人中心域精致化）

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
  超长循环动效 24s/36s 且 reduced-motion 停用；背景常驻形成纵深感。**2026-10-28 起 Web 端动画默认关闭**（`<AmbientBackdrop motion />` opt-in，性能修复见 design-system §21.11）；
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

### 2026-11-24 跨端补齐（Web/Android/鸿蒙对齐轮次）

- **Web 审计详情复制日志 ID**：`admin/audit` 详情弹层新增「复制日志 ID」；抽 `lib/clipboard.ts`（Clipboard API + execCommand 回退）并补 4 条单测（PR #203）。
- **鸿蒙「我的分享」入口 + 搜索**：`MySharesPage` 早已实现且注册路由但无入口，设置页「隐私与安全」组补入口（PR #204）；页面补搜索（契约无 keyword → 加载全部分页后本地筛选，PR #205）。
- **鸿蒙通知中心搜索**：对齐 Web `/notify` 与 Android `NotifyScreen`；搜索加载全部分页后按标题/文案/类型筛选（PR #207）。过程中修复两个 ArkUI 渲染问题：① 搜索词先于防抖筛选写入导致分组 key 不变、List 复用旧行 → `applyFilter` 递增 `listVersion` 并入分组 key 与 List id；② 空态垂直居中落在键盘下方 → 搜索空态改顶部对齐。
- **跨端整卡遮罩 CI 校验**：新增 `check-cross-platform-card-scrim.mjs`（web scrim rgba 6 组合 × 5 档 ↔ 鸿蒙 `cardScrim*` 预合成 ARGB），串入 `bun run check:theme`（PR #206）；AMBIENT 已有 `check-cross-platform-ambient.mjs` 覆盖，FALLBACK/WATERMARK 仍为人工核对。

### 2026-11-24 聊天与骨架补强（Round 99-102）

- **聊天气泡按内容收窄**：修复短消息气泡被 `width('100%')` 元信息行撑到 75% 上限的问题（PR #209）。
- **发送失败内联重试**：鸿蒙输入栏上方错误条 + 「重试」，失败保留原文、重试重发；对齐 Web 形态（PR #210）。
- **重复气泡修复**：① 乐观气泡改为先插 pending 再请求（避免 WS 回显先到导致双条）；② 历史加载后数据源与页面数组共用引用，`addData` 二次追加 → 统一改 `setData`（PR #211）。
- **骨架屏补强**：通知中心首屏（PR #212）与关系页（PR #213）加载态由 spinner 改为与行几何一致的 5 行骨架；搜索页已有骨架（历史实现）。

### 2026-11-24 列表骨架与状态机修复（Round 103-108）

- **列表骨架通用化**：新增 `components/states/ListSkeleton`（5 行圆头像 + 两行文字条 + shimmer），我的分享、系统公告、我的举报三页接入（PR #216）；关系页（#213）、通知中心（#212）此前已接入，主要列表骨架补齐。
- **管理后台公告列表过滤已软删除**：`querySearchAnnouncements` 补齐 `deleted_at IS NULL`（PR #218），修复「删除后行残留」与「编辑已删除公告报 10300」；契约 `AdminAnnouncementItem` 无删除标记，按 spec 口径过滤（契约扩展需用户批准）。
- **Web 我的举报搜索状态机**：搜索加载/失败/鉴权态入内容区（InlineLoading / LoadError 重试 / 10100 登录引导），首屏 loading 判断改为 `loading && !searching`，补 3 条契约测试（PR #217）。
- **Android 审计日志筛选状态**：起止时间稳定化（修复 `Instant.now()` 作 key 导致时间筛选无限重拉：端上实测 8 秒 2977 次 → 每次筛选 1 次）、`loadMore` 复用筛选条件、筛选切换清空旧列表、搜索防抖前进入加载态、自定义结束时间取当日最后一微秒、行内 `details` 去重（PR #219）。
### 2026-11-24 回顶扩展（Round 109-111）

- **设置页回顶**：长页面滚动超过 600vp 显示 `BackToTopButton`，点击平滑回顶（PR #221）。
- **个人中心本人页回顶**：滚动超过 240vp 显示回顶按钮（个人页内容约 1.5 屏，600vp 阈值够不到），点击平滑回顶（PR #222）。
- **踩坑（ArkUI）**：`.overlay(builder)` 的 builder 在隐藏态返回空时，**空 overlay 节点会拦截手势**——不仅拦滚动（设置页实测完全无法滚动），也拦点击（搜索页 Tab/结果行实测全部点不动，PR #227）。修正：回顶按钮改用 Stack 同级子节点或始终渲染的全屏 Row + `hitTestBehavior(HitTestMode.Transparent)`，空白区域手势穿透、按钮可点。
### 2026-11-24 搜索交互与群组行对齐（Round 112-117）

- **跨端水印透明度纳入 CI**：新增 `check-cross-platform-watermark.mjs` 并串入 `check:theme`；FALLBACK 渐变因两端无共享公式登记人工（PR #224）。
- **搜索结果用户行对齐 web**：头像 40→44（`AVATAR_SIZE.SMALL_PLUS`），行高 64→68vp（PR #225）。
- **用户搜索不再过滤本人**：去掉 `doSearch` 的 `user_id === 当前用户` 过滤，与 web/Android 及 spec 对齐（PR #226）。
- **空 overlay 拦截点击（重要回归）**：`SearchPage` / `MomentsFeedPage` 的回顶 overlay 在隐藏态返回空，空 overlay 节点会拦截下层**点击**（搜索结果 Tab、结果行全部点不动，滚动却正常，所以此前只测滚动未发现）。修正为始终渲染全屏 Row + `HitTestMode.Transparent`（PR #227）；通知中心 builder 本就常驻渲染，未受影响。
- **群组结果行紧凑化**：由「头像+成员数+两行简介+加入/详情双按钮+更多菜单」（约 200vp）改为 web EntityRow 同款（44 squircle 头像 + 名称 + N 位成员 + 箭头，68vp），整行点击进群详情，并清理仅旧行使用的加入/举报/复制等方法（PR #228）。
### 2026-11-24 回顶扩展收尾（Round 118-121）

- **我的分享回顶**：`MySharesPage` 列表滚过 240vp 显示回顶按钮（PR #230，24 条分享造数验证，验证后清理）。
- **他人主页回顶**：`ExternalProfileView` 改为 `Stack { Scroll; 回顶 overlay }`，4 个带图瞬间时约 1.5 屏（PR #231）。
- **我的群组回顶**：`GroupPage` 列表区 `Refresh` 外包 `Stack` 挂回顶按钮（PR #232，23 个群组造数验证，验证后清理）。
- **#33 完成**：设置页、个人中心本人页、我的分享、他人主页、我的群组共 5 处长列表均已接入；统一模式为「Stack 同级子节点 + 全屏 Row + `HitTestMode.Transparent`」，禁止空 overlay builder（会拦滚动与点击）。
### 2026-11-24 质量修复收尾（Round 122-125）

- **Web 详情页重复 h1（PR #234）**：用户/群组/瞬间详情页 `DetailHeader` 已提供 sr-only h1，页内 `IdentityHero`/`GroupHero`/页级 sr-only 又渲染 h1 → 巡检 h1=2；可见标题降 h2、移除重复 sr-only，三页 DOM 实测各只剩 1 个 h1（视觉零变化）。
- **鸿蒙瞬间详情图片失败态（PR #235，关闭 #31）**：`MomentDetailPage` 首图与多图缩略图补 `onError` + 失败占位（图片图标 +「图片加载失败」/ 1:1 图标）；停 SeaweedFS + 无效 `file_id` 造 404，端上实测「失败占位 → 还原后正常加载」三态。
- **axe 主题矩阵对比度误报治理（PR #236）**：媒体卡压字层背景为「图片 + 渐变遮罩（兄弟节点）」，axe 只能回溯到卡片 surface → 浅色主题白字误报 white-on-light（forest/inkwash light 3 条 serious）；仅对命中 `[data-card-footer]`/`[data-card-children]` 的 `color-contrast` 节点过滤并计数，真实对比度核算与口径写入 `frontend-ui-audit.md`；矩阵复跑 6 组合 × 43 页 PASS。
- **Web 群聊非成员直链加入引导（PR #237）**：`10200 not_group_member` 由「错误文案 + 重试」改为 `EmptyState` +「加入群组」CTA（direct 加入后自动重拉历史/群信息、approval 转待审核、invite_only 内联服务端原因），未入群时隐藏输入栏；补 3 个组件契约用例。
- **审计 backlog 复核（Round 125）**：群聊非成员直链、纯文字瞬间卡、`/users/:id` 统计空态矛盾、1440 首页右栏不吸顶 4 项已关闭（证据见 `frontend-ui-audit.md`「审计 backlog 复核」）；保留页头返回钮标题偏移、768 平板后台表格列裁剪待后续。

### 2026-11-24 通知键盘导航与 backlog 复核（Round 126-127）

- **审计 backlog 复核（PR #241）**：768 平板后台表格实测无横向溢出（`/admin/users` wrap 720=scroll 720，操作列完整可见；`/admin/audit` 按断点隐藏详情/IP 列），「积压表格列裁剪」关闭；backlog 关闭 5/6，仅余页头返回钮 leading 槽位。
- **通知列表键盘导航（PR #242）**：复用 `lib/search-nav` 给 `/notify` 接入 ↑↓（搜索框 ↓ 进入列表首项、候选内逐项移动、弹层打开不接管）；实测焦点序列 搜索框→接受→拒绝→通知行→↑ 拒绝，`bun run check` EXIT=0。
- **UI 质量门回归（Round 128）**：`bun run audit:ui` 全绿——browser-qa（桌面 21 + 移动 5）exit 0、audit:a11y（38 页）0 serious/critical、audit:touch（38 页 988 元素）无未登记离群。
- **鸿蒙我的瞬间三 tab 端上复核**：#26 关闭（见待做项）。

## 待做（按优先级）

> 已解决并关闭：骨架零位移 N1/N2（R14）、forest 亮色软岛同色（R5 v2 1.17）、主题/axe/触控/焦点/减少动态基线（R5）。
> 下一轮从 1 开始按序接。

1. **鸿蒙真机像素/TalkBack/流量/动效/占位/聊天滚动验收（R13–R21 遗留，最高优先）**：八轮鸿蒙改动均只完成构建/布局树/token 校验；
   真机 hdc 离线导致三主题×明暗矩阵、灰度 Tab、压图文字对比度、软岛阴影/圆角/gap、按钮 hover、未读态、
   骨架零位移、整卡图片流量/渐显像素、头像 2x vs 3x、TalkBack 顺序、reduced-motion 系统路径、web↔鸿蒙并排 compare 挂起。
   设备恢复后按 `.workwork/ui-polish-1026/{r2,r3,r4,r5,r6,r7,r8,r9,r10}/ha/report.md` checklist 执行（r7 动效手感/中间帧、
   r8 占位/慢网、r9 页头滚动/回顶、r10 聊天滚动/回底计数）。
2. **长尾自定义卡收尾（R15 已基本完成）**：剩余 demo-only 演示卡（`/?demo=`）与地图面板按例外保留；
   新增页面按 §19/§19.5 判定；鸿蒙后台仅 AdminOverview（其余 web-only N/A）、首页长尾组件 N/A。
3. **鸿蒙氛围画布实例数/内存优化（R12 遗留）**：当前每 `fillHeight` 页一块全屏 Canvas（Tab 页常驻，实测 live 2→5，
   PSS 218.7→330.2MB）；可选上提到外壳内容区单实例/共享绘制缓存；改前先量首帧与滚动帧率。
4. **氛围浓淡评审**：世界层透明度集中在 `themes/*.css` 的 `--cst-ambient-*`（12 个）；鸿蒙按同口径重算 AMBIENT ARGB。
5. **鸿蒙骨架屏接入扩展（已补齐主要列表）**：搜索页（历史实现）、通知中心（PR #212）、关系页（PR #213）已接入；其余长列表（如我的分享/公告/举报）可按同一模式补，非阻塞。
6. **搜索 Tab 计数格式对齐（已对齐，关闭）**：web `TabBar` 的 count 为独立小号数字（视觉「用户 7」）；鸿蒙 `SearchPage.tabLabel()` 在 `hasSearched && count > 0` 时输出 `${base} ${count}`（空格分隔，代码注释明确对齐 web），两端一致；此前「用户 (10)」描述已过期。
7. **搜索发现态 spotlight（待设计确认）**：web 搜索发现态未加，鸿蒙未单端先行；需要则两端同轮加。
8. **消息页头 trailing 差异（待确认）**：鸿蒙消息页头 trailing 与 web 移动端既有差异，需要时单独立项。
9. **BSP 瀑布墙骨架近似（已接受）**：无 1:1 骨架，用同列数卡栅格近似，轻微重排；收益低保留。
10. **鸿蒙 FALLBACK / WATERMARK token 纳入 CI（已完成 WATERMARK；FALLBACK 登记人工）**：AMBIENT 72/72、CARD-SCRIM 6×5 已覆盖；WATERMARK 由 `check-cross-platform-watermark.mjs` 逐值校验并串入 `check:theme`（PR #224）。
    `cardFallback*`（无图 fallback 渐变）经色板反解确认**无跨端共享公式**（App 端三段渐变，亮色 accent 预混约 20%/24%/18%、暗色手调），web 为纯色 surface-primary，登记为人工视觉核对例外（见 `docs/design-system.md` §15）。
11. **鸿蒙装饰几何常量 token 化（低优先）**：EmptySpotlight/AmbientBackdrop 的 320/288/168/4/102 与透明色可抽常量/登记。
12. **鸿蒙设置页/管理后台无氛围层（平台差异）**：后续对齐应在**外壳/外层**挂单实例，禁止在滚动容器内挂。
13. **跨端搜索结果行字号/头像档差异（已完成，关闭）**：用户行已对齐 web EntityRow（44 头像 / 14 名称 / 12 副标 / 68vp 行高，PR #225）；群组行已改紧凑 EntityRow（PR #228）；此前「web 64/36/12」口径已过期。
14. **网格卡字号例外（R13）**：171px 网格卡正文 12 / 作者 14；放大卡宽后可统一。
15. **BSP 卡片比例例外（R13）**：BSP 格按布局单元铺满、非固定比例；统一需重排布局。
16. **横排紧凑卡 fallback/精选横排例外（R13）**：horizontal 卡仍两段式与旧占位；群详情精选桌面横排保留。
17. **图片档位/清晰度真机复核（R17）**：dev 图片为本地代理图，unsplash 压缩在预发/线上生效；真机核首屏流量、
    头像 2x vs 3x、nearby 页 +43% 取舍（非首屏可回退 thumb 档）。
18. **鸿蒙缩略图渐显（5 处已核对，剩 2 处低优先）**：ProfileGallery（`ProfileMomentGallery` onComplete + opacity 渐显）、MyShares 封面、通知头像（`SystemAvatar` 内置渐显）、搜索/详情等已接；MomentDetail 首图/缩略图失败态已补（PR #235）；**Chat 图片气泡与 PhotoPreview 大图的淡入仍待弱网/慢图环境验证**，暂不阻塞。
19. **键盘导航扩展（R17）**：`/search`（R6）之外，**通知列表已接入 ↑↓**（PR #242，复用 `lib/search-nav`：搜索框 ↓ 进入列表首项、候选内逐项移动、弹层打开不接管）；feed/首页列表仍待读屏/真机验证后评估（行内控件多、与滚动冲突风险高）。
20. **a11y/触控/主题自动化纳入门禁（R16）**：axe-core 加 devDependency + 检查脚本，触控覆盖率脚本入库，
    并把 main/landmark、整卡覆盖按钮、列表 role、触控 44 加入 PR 自查清单。
21. **触控例外清单（R16）**：地图 marker/控件（防相邻误点）、相邻 TagPill（WCAG 24 + 间距）、表单控件、skip link；
    新增「相邻可点元素」场景先跑 390 扫描再决定扩热区方式。
22. **性能基线口径（R16/R17）**：LCP/CLS/transferSize 均为本地 dev/mock 数据，仅相对参考；生产需真实网络另测。
23. **inkwash light 岛合成口径（R16）**：有效合成 #FAF8F2；不同采样点（ambient 高光）会读到 #FCF9F3，报告需标注取样口径。
24. **交互状态契约守护（R15）**：focus-visible/reduced-motion/四态已入 design-system §20 与 skills；新页面按契约自查。
25. **admin mock 空态不可复现（R15）**：mock 列表未实现筛选参数，空态不可达；联调环境补验证。
26. **鸿蒙我的瞬间 tabs（R13，已完成，关闭）**：`MomentPage` 已实现发布/喜欢/收藏三 tab，数据源分别为 `moments/fetch_created`、`users/fetch_reacted_moments`、`moments/fetch_favorites`；端上实测（Mate X7 模拟器 / QA）：发布 9、喜欢 4（他人瞬间、心形填充）、收藏 2（书签填充），切换即加载对应列表。
27. **图片管线契约守护（R17）**：新增图片必须走 `lib/images.ts` 档位（unsplash 参数/srcset/渐显/首屏优先级），禁止裸写 `<img src>`；
    reviewer 已把「非 unsplash 原样/失败降级」纳入审计。
28. **动效契约守护（R18）**：新增动画必须走 `lib/motion.ts` / 鸿蒙 `MotionUtil`；页面进场只能客户端导航、JS 动画必须显式 reduced；
    pop 只能在用户激活操作触发；禁止动画首屏 LCP 元素/引入 CLS/列表逐项 stagger。经验：Svelte transition 需用 `motionDuration()` 处理 reduced，
    否则 reduced 下仍播。
29. **鸿蒙动效残留（R18，已完成，关闭）**：MomentFeedCard 的 `preview`/默认变体当前**无调用点**——HomePage、MomentPage、SearchPage 三处全部 `variant:'overlay'`（`grep -rn 'MomentFeedCard({' -A4 harmonyapp/entry/src/main/ets` 可见）；overlay 卡动作条由 `MediaActionButton` 覆盖（`@Watch('onActiveChanged') → MotionUtil.statePop`，`MediaOverlay.ets:177/215`），点赞/收藏已有 1→1.15→1 pop。SearchPage 用户行**登记 N/A**（Web 同样无关注入口，非鸿蒙缺口）；GroupPage joinGroup 函数级已接 pop。若后续重新启用 preview/postCard 卡片，按 MediaActionButton 同口径在激活态变化时接 pop。
30. **search 390 既有 CLS 0.0995（R18 登记，R19 已降至 0.0077）**：r8 的媒体占位/结构改动顺带消除该位移，后续回归按 0.0077 基线看。
    动效相关 QA 建议在 reduced-motion 下取基线截图，避免中间帧偶发 diff。
31. **鸿蒙详情图片失败态（R19，已完成，关闭）**：`MomentDetailPage` 首图/缩略图已补 `onError` 失败占位（PR #235，停 SeaweedFS 造 404 端上三态实测）。
    另：`delay-images.js` 慢网模拟目前在 `.workwork` 未入库，建议后续并入 browser QA 脚本做常规回归。
32. **图片失败降级统一审计（R19，已完成，关闭）**：复核全站 `<img>` 后确认远程图全部具备失败降级——9 个文件走 `use:fadeImage`（`img-fade` 初始 `opacity:0`、仅 `load` 后加 `img-loaded`，失败时图片保持不可见、父级 `.media-placeholder` 渐变透出，不会出现 broken icon）；MediaCard/MomentPostBody 用显式 `imageLoaded` 门控（失败保持占位层）；其余裸 `<img>` 均为**本地/静态资源**（品牌 logo：BrandLockup/HomeMobile/HomeTablet/DesktopLayout；相册本地 blob 预览：ImagePicker；地图画布：MapCanvas）或 `aria-hidden` 装饰层。复核命令：`grep -rl '<img' frontend/src --include=*.svelte` 与 `grep -rln 'use:fadeImage' frontend/src`。新页面/新组件继续按 design-system §20.10 自查。

33. **鸿蒙回顶扩展（R20，已完成，关闭）**：设置页（#221）、个人中心本人页（#222）、我的分享（#230）、他人主页（#231）、我的群组（#232）共 5 处长列表已全部接入 BackToTopButton。
    接入约定：**不要用空 builder 的 `.overlay()` 挂回顶按钮**——空 overlay 节点会拦截手势（滚动与点击都会被拦，见「2026-11-24 回顶扩展」踩坑记录）；统一用 Stack 同级子节点 + 全屏 Row + `HitTestMode.Transparent`。新增长列表按此模式接入并端上验证显隐/回顶。
34. **回顶与浮层避让（R20）**：当前基线无 changelog 浮动钮；若后续引入同角浮层，按「回顶置其上留 8–12px」规则处理并实测不遮挡。
35. **页头玻璃可读性（R20）**：scrolled 状态在真实内容（图片/大标题）下的可读性、宽屏/真机滚动观感待真机复验；如觉过透可调 78% 系数（两端 token/常量需同步）。
36. **SPA 焦点/播报（R20，已实现，关闭）**：`RoutedContentViewport` 的 `afterNavigate` 已做「聚焦 main（不可聚焦则回退 h1）→ `announceRoute` 播报」；新增页面无 h1 时仍按 §20.11 自查（优先补 h1）。复核：`grep -rn 'afterNavigate|announceRoute' frontend/src/components/layout/page/RoutedContentViewport.svelte`。
37. **Reflow 持续回归（R20）**：320px/200% 缩放当前抽 34 页 + 8 关键页通过；新页面/新组件按 §20.11 复测，装饰层超界必须被裁剪。

38. **聊天发送失败重试（已完成两端）**：Web 输入栏错误行 + 「重试」（PR #e02ff939e）；鸿蒙内联错误条 + 重试（PR #210，端到端验证：停后端失败 → 恢复后重试成功）。发送重复气泡另见 PR #211。
39. **鸿蒙聊天页布局树受限（R21）**：模拟器重装登录态重置，游客无会话无法进入聊天页；聊天滚动/回底行为以代码级触发证据 + reduced 分支为准，真机/有会话账号补布局树与像素。
40. **聊天滚动阈值与按钮位置回归（R21）**：近底 120px/按钮 200px、right-14/bottom-3、99+ 上限为当前基线；新增输入栏浮层/表情面板/引用条时需复测按钮不遮挡。
41. **聊天未读语义（R21）**：markRead 延后为客户端语义（服务端不维护独立未读字段），上滑/离开不补标；后续若引入「会话内未读分隔线」需与后端游标契约对齐再实现。

42. **质量门禁每轮必跑（R22）**：前端改动合并前在主环境或 worktree 跑 `bun run audit:ui`（需 dev 环境运行）；a11y serious/critical、
    触控未登记离群必须为 0；新增合理例外才进 allowlist 并写理由。CI 无浏览器，暂为本地门禁（后续可评估 Playwright/远程浏览器接入）。
43. **allowlist 维护（R22）**：6 条规则（地图 marker/面板拖拽/地图筛选/pressed chip/表单/skip link）以稳定 selector 匹配；
    页面结构大改后若规则失配，脚本会报未登记离群，按「先确认是否合理例外，再决定修元素或改 allowlist」处理。

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
- **Android 群聊非成员态不可达（Round 126 实测）**：`GroupChatScreen` 的 `notMember` 分支需要 `group_messages/fetch`
  返回 10200，但正常 UI 流程下非成员无法进入群聊页（入口仅群详情「打开群聊」，非成员显示「加入群组」）；
  被移出群后聊天页也不会自动重拉历史（需重进页面，而重进后群详情按钮已变成「加入群组」）。该状态仅理论可达，
  Web 端已给加入引导（PR #237）；Android 侧移植无法端上验证，暂不实施。若后续接入群聊深链或成员移除 WS 事件，再补 CTA。

## 端上验证环境要点

- 模拟器直启：`/Applications/DevEco-Studio.app/Contents/tools/emulator/Emulator -start "Pura 90 Pro"`
  （约 10s 就绪，`hdc list targets` 出现 `127.0.0.1:5555`）。
- **真机与模拟器同时在线时**必须 `hdc -t 127.0.0.1:5555 install ...` 指定设备，否则报
  `need connect-key`。
- 模拟器装**仓库构建**即可正常加载数据（`DEV_SERVER=http://10.0.2.2:8888`），无需临时改常量。
- 想验证错误/空态：`docker compose -f docker-compose.yml -f docker-compose.dev.yml stop backend`
  → 各页进入错误态 → 验证完 `start backend`。
