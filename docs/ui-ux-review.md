# UI/UX 全面审查报告（harmonyapp 优先）

> **修复进度（2026-08-30）**：P0 五组已全部修复并逐处核验（另补修了审查后发现的 NotificationCenter 互动类淡底色、ProfilePage 退出确认两处同类问题）。详见文末「P0 修复记录」。

审查日期：2026-08-30
审查范围：`harmonyapp/`（ArkTS，34 个页面 + 30 个公共组件）、`frontend/`（SvelteKit，抽查代表页面与核心组件）
审查方式：全量阅读页面源码，核对设计 token 体系与实际落地的一致性、布局结构、交互反馈、状态完备性。

---

## 一、总体结论

**设计系统基础很好，问题集中在"逻辑正确性"与"体验细节"两层。**

- harmonyapp 的 token 体系（`DesignConstants.ets`、`ThemeConstants.ets`）已相当完善：字号/间距/圆角/按压曲线全部收敛，全仓 `features/` 下**无一处硬编码颜色或字号**，这一点超过绝大多数项目。
- 前端 token 体系（`--cst-*` 单源 + 语义别名）同样严谨，hover/tap/focus 契约有文档化注释。
- **但真正的 UI 问题不在样式，而在**：假交互（点了没反应的筛选）、错误状态展示（统计截断、颜色倒挂）、危险操作无确认、死功能入口外露、触控目标不足、骨架屏体系空转。

问题分级：🔴 高（影响功能正确性或可能误操作）｜🟡 中（明显体验缺陷）｜🔵 低（打磨项）

---

## 二、harmonyapp 逐页问题清单

### 1. 登录页 `features/auth/pages/LoginPage.ets`

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L502-526 | 登录按钮 `onPress` 与外层 `.onClick` 双重绑定，`userLogin()` 会执行两次（重复请求 + 重复 Toast） | 去掉外层冗余 onClick |
| 🟡 | L455-468 | 账号输入框未设 `inputType`，邮箱登录不弹出带 @ 的键盘 | 按登录方式动态切换键盘类型 |
| 🟡 | L592-617 | 验证码输入框与「发送验证码」按钮高度不一致，行内不齐 | 统一高度档 |
| 🔵 | L271-281 | isLoading 时整页替换为全屏 loading，已填表单视觉消失 | 改为按钮内 loading 或遮罩 |
| 🔵 | L285-315 | 返回按钮无无障碍描述 | 补 `accessibilityText` |

✅ 亮点：表单校验（touched + 失焦报错、实时 disabled、验证码 60s 倒计时）是教科书级实现。

### 2. 首页 `features/home/pages/HomePage.ets`

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L587/606/625 | HeroStats 三段统计直接展示 `users.length`（接口 limit 10），数值被截断为 ≤10，与「实时附近」语义矛盾，误导用户 | 用服务端 count 字段或标注「附近前 10」 |
| 🟡 | L474-518 / L660 | 整张 Hero 卡可点跳转，与 header 地图入口、「查看全部」三处入口重叠；且卡片含统计信息，看数字时误触率高 | 收敛为仅箭头区可点 |
| 🔵 | L905/1050 | 空态为单行灰字，未复用 EmptyState 组件 | 用 EmptyState 小号变体 |
| 🔵 | L1079-1102 | 活动行图标/颜色按 index 轮换，与内容类型无关，刷新后同一条动态可能换色 | 按内容类型映射 |

### 3. 布局框架 `components/layout/`

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | PhoneLayout.ets L128 | 手机端底栏 `backgroundColor(pageBackground)` 与 `backgroundBlurStyle` 同时设置，实色盖住毛玻璃，模糊失效 | 背景改半透明色 |
| 🟡 | PhoneLayout.ets L91-106 | 通知 banner 流式排布把整页（含底栏）往下推，布局跳动 | 改顶部 overlay 浮层 |
| 🟡 | DesktopLayout.ets L567-598 | 切 tab 用 if/else 销毁重建页面，切回首页重新请求三接口、loading 闪烁 | Stack + visibility 保活 |
| 🟡 | DesktopLayout.ets L591 | 「未知路由」无返回引导 | 加回退按钮 |
| 🔵 | DesktopLayout.ets L174-185 | 桌面创建 sheet 与手机端结构不一致（跳转项 vs 直出表单） | 两端对齐 |

### 4. 动态（Moments）模块

**`MomentsFeedPage.ets`（附近动态）**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L399 | 瀑布流用 `ForEach` 渲染，已建好的 `hotMomentsDataSource`/LazyForEach 闲置，长列表无虚拟化，滚动掉帧 | 切换 LazyForEach |
| 🔴 | L578/L657 | `buildRecommendedUsersSection`/`buildRecommendedGroupsSection` 从未被调用——推荐区块在移动端完全不渲染 | 接入或删除死代码 |
| 🟡 | L540 | 标签药片只切换高亮，无任何过滤逻辑，**假交互** | 过滤数据或触发搜索 |
| 🟡 | L273 | 假搜索框形似输入框但无按压反馈，易被当成可直接输入 | 加 Pressable 反馈或改为跳转按钮样式 |
| 🔵 | L266 | 「N 条内容」是当前页数量而非总数 | 用总数或改文案 |

**`MomentDetailPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🟡 | L118-125 | `calcImageGridHeight` 硬编码 180/240 行高，宽屏/折叠屏图片比例失调 | 按 aspectRatio 驱动 |
| 🟡 | L344-345 | 评论输入框 + 三键操作条双层底栏堆叠，视觉臃肿 | 合并为一行 |
| 🔵 | L492/L546 | 评论数重复展示；点赞按钮文案切换导致宽度跳动 | 固定最小宽度 |

**`MomentPage.ets`（我的瞬间）**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🟡 | L223 | 首屏仅转圈无骨架屏，弱网感知空白久 | 接入 Skeleton 组件 |
| 🟡 | L279 | `textMaxLines: 0` 全文展开，长文撑爆列表 | 限 3-5 行 + 展开按钮 |
| 🔵 | L288 | 加载到底无「没有更多了」；无定位时静默只读缓存，用户误以为账号为空 | 加收尾提示与状态说明 |

**`CreateMomentPage.ets`（发布）**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L89-110 | 进页即申请 5 项权限（媒体/相机/定位），权限轰炸，违反按需申请原则 | 拍照/选图/用位置时各自按需申请 |
| 🔴 | L671-681 | `calcGridHeight` 固定 itemHeight 80，三列实际宽度约 100+，缩略图被 Cover 严重裁切、比例失调 | 按列宽动态计算 |
| 🟡 | L446 | 「编辑暂不支持修改图片」提示**无条件渲染**，新建模式也显示 | 加模式判断 |
| 🟡 | L243 | 9 张图串行压缩上传，仅文字提示无逐张进度，失败即整体中止 | 并发上传 + 逐张进度 + 单张重试 |

**`PhotoPreviewPage.ets`（图片预览）**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🟡 | L90 | 单指 PanGesture 与 Swiper 翻页手势竞争，横滑切图易卡顿；缺单击退出 | scale=1 时禁用手势 |
| 🟡 | L175/L201 | 关闭按钮 margin 未叠加 safeAreaTop，页码 `y:'90%'` 硬编码，刘海屏位置漂移 | 叠加安全区 |

### 5. 附近 / 地图 / 群组模块

**`NearbyUsersListPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L207-212 | Tabs「在线」实际映射回 distance 排序，**筛选无效且误导用户** | 实现 online 过滤或移除该 tab |
| 🟡 | L91 | 定位缺失仅文案提示，无「去设置」跳转 | 加引导按钮 |

**`NearbyGroupsListPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L119-124 | `loadMoreGroups` 丢失 keyword/tag 参数，**翻页后筛选失效** | 透传筛选参数 |
| 🟡 | L289 | Scroll+ForEach 无懒加载；48 个 TAGS 硬编码与后端分类易脱节 | LazyForEach + 分类走接口 |

**`LocationPickerPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L150/L230 | 定位失败仅 hilog 日志，用户无任何感知，无权限引导 | Toast + 去设置入口 |
| 🟡 | L822-825 | 筛选全不选时地图空白无提示 | 加空态说明 |

**`GroupPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L858-865 | 「我的群组」（已加入列表）仍显示「加入」按钮，逻辑矛盾 | 改为「进入」或隐藏 |
| 🟡 | L744 | 申请项头像/群名固定写死「群组申请」，缺申请人与目标群信息 | 展示真实数据 |

**`GroupDetailPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L425-429 | `joinGroup` 把 10400 一律视为「已是成员」直接进聊天，**密码错误等业务错误会误跳转** | 按错误码分支处理 |
| 🟡 | L1000/L1048 | 同页两种距离格式（`xxx m` vs formatDistance）；离开群组无二次确认 | 统一格式 + 确认弹窗 |

**`CreateGroupPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🟡 | L525 | 定位失败永久停留「正在获取位置…」，无重试入口 | 失败态 + 重试按钮 |
| 🟡 | L586/L629 | 人数步进 ±10 无法精确设值，非数字输入静默忽略 | 支持直接输入 + 非法输入提示 |

**`NearbyImageCard.ets`**：🔴 L81-85 `getBadgePosition` 两个分支返回相同坐标，`badgeAlign=topLeft` 失效。

### 6. 聊天 / 消息模块

**`ChatPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | ChatMessageItem.ets L124-133 | 自己消息的时间与已读勾使用 `sendButtonActiveText`（为气泡底色设计的亮色）绘制在气泡外的列表背景上，**浅色主题下近乎不可见** | 改用 `textTertiary` |
| 🔴 | L861-904 | 发送无乐观气泡，依赖 WS 回显才上屏，弱网下「发了没影」；失败仅 Toast 无重发 | 本地 pending 气泡 + 失败红点重试 |
| 🔴 | L1192-1201 | 「没有更多了」渲染在最新消息**之下**，语义应在列表顶部 | 移到首位 |
| 🟡 | L765-805 | 图片发送无本地预览/进度/失败重发；图片消息不设 aspectRatio 与最大高，超高图撑爆列表；无点击预览 | 限高 + 预览 + 占位 |
| 🟡 | L453/L456 | 收到新消息无条件 scrollToBottom，上翻读历史被拽回底部；同时立即 markAsRead「未看即已读」 | 仅贴底时滚动 + 「↓新消息」悬浮钮 |
| 🟡 | L1390-1396 | 1px 透明可聚焦 Button 作键盘失焦落点，会被读屏朗读为空按钮 | 改用不可聚焦方案 |

**`MessagePage.ets`（会话列表）**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🟡 | L773-817 | 注释声称 LazyForEach，实为 Scroll+ForEach 全量渲染；删除用长按 bindMenu 可发现性差 | 真懒加载 + 滑动删除 |
| 🟡 | L529-562 | 「删除会话/退群」无二次确认，误触即退群 | 确认弹窗 |
| 🟡 | — | 置顶会话无任何视觉标记（无 pin 图标/底色） | 加置顶标识 |

**`NotificationBanner.ets` / `NotificationCenter.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | NotificationBanner.ets L128-135 | **优先级颜色倒挂**：critical 用 `textTertiary` 灰、medium 反而用 accent 亮色，紧急通知视觉上最弱 | critical→error 色 |
| 🟡 | NotificationCenter.ets L763 | 「互动」类（点赞/评论）用 `error` 红色徽标，语义误导为告警 | 改用 accent/中性色 |
| 🟡 | NotificationBanner.ets L54-64 | 关闭图标 16vp 无热区扩展，难按 | 扩至 44vp 命中区 |
| 🟡 | L1470-1483 | 「举报」直接以硬编码 reason 提交，误触即产生真实举报 | 加确认/理由选择 |

### 7. 我的 / 设置模块

**`ProfilePage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L560/1070/1185 | `logout()` **无二次确认**，点击立即清态跳首页 | AlertDialog 确认 |
| 🔴 | L1237 | 身份卡相机按钮选图后未进入编辑态，Toast 说「保存后生效」却无保存入口，**头像变更直接丢失** | 选图后进入编辑或直接上传 |
| 🟡 | L498 vs ProfileEditPage L31 | 简介上限 500 与 150 两处不一致 | 统一为单一常量 |
| 🟡 | L1171 | 帖子/喜欢/收藏三个 Tab 均传同一 `myMoments`，Tab 切换内容无差异 | 分别取数或隐藏未实现 Tab |
| 🟡 | L1540 | 升级表单 placeholder「邮箱/用户名」与 `isValidLoginId`（拒绝 @）矛盾 | 修正文案 |

**`ProfileEditPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L239/L275 | 头像、横幅更换均为「开发中」Toast——**主视觉区两个入口都是死功能** | 实现或隐藏 |
| 🟡 | L201/L337 | 头部「保存」与底部保存按钮双入口，保存中 header action 变 null 造成布局跳变 | 只留底部主按钮 |

**`SettingsPage.ets`**

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | L674 | 退出登录无确认，且与 ProfilePage 的 logout 实现不一致（此处清消息缓存，彼处不清），**两处退出行为数据残留不同** | 统一退出逻辑 + 确认弹窗 |
| 🟡 | L594-609 | 私密账号/位置共享与 PrivacySettingsPage 重复且各自维护 state，两边可能不一致 | 子页单一数据源 |
| 🟡 | L546-563 | 深色模式开关 + 主题分段卡 + 更多外观模式三套概念并存 | 收敛为一套 |

**`SafetySettingsPage.ets`**：🔴 L60-79 修改密码无前端校验、无确认密码字段；🔴 L81-111 撤销会话/全部下线无确认。
**`AdvancedSettingsPage.ets`**：🔴 L57-59 「清理缓存」未真正清理即 Toast「已清理」，**虚假反馈**。
**`AccountsPage.ets`**：🔴 L173 删除账号（含当前账号）无确认弹窗，且「删除」用 `variant: 'primary'` 而非 danger，危险操作视觉层级错误。
**`MySharesPage.ets`**：🔴 L184/381 撤销分享无确认；🟡 L304 share_code 不可复制。
**`SearchPage.ets`**：🟡 L226-231 discovery 数据恒为空仍渲染空标题容器；🟡 L328 vs L586 话题搜索两条路径关键词不一致（带不带 `#`）。
**`PrivacySettingsPage.ets`**：🟡 L90-92 加载前所有开关渲染为「关」，加载完成集体跳变。

### 8. 公共组件库 `components/`

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | Pressable.ets L14 vs SystemButton.ets L223、SystemCard.ets L194、SurfaceIconButton.ets L80、TabsPillGroup.ets L110、LoadMoreButton.ets L65 | 注释明令「禁止手写 onTouch+scale+animation 三件套」，**但库内 5 个组件全部各自手写** | 收敛到 Pressable 或下沉为 Modifier |
| 🔴 | SurfaceIconButton.ets L13、SystemPageHeader.ets L132、SystemInput.ets L110 | 图标按钮默认 36vp、返回键 36vp、输入框清除钮 36vp，**均低于 44vp 触控目标**（HarmonyOS 设计规范最小 40vp，WCAG/iOS 44） | 视觉不变、透明 padding 扩命中区 |
| 🔴 | Skeleton.ets / SkeletonShimmer.ets / ErrorBoundary.ets | **全仓 features 零引用**——骨架屏体系是死代码，加载态实际走 SystemLoadingView/裸 LoadingProgress | 推广到首屏列表或删除 |
| 🔴 | SystemDialog.ets L68 | 按钮 Row 无间距两键紧贴；无进出场动画；条件渲染在页面树内，**层级无法保证盖在后置浮层之上** | 改 bindContentCover |
| 🟡 | SystemToast.ets L34/40 | 用 emoji（✅❌）当类型图标，与设计系统 SymbolSpan 语言不一致；bottom 固定值未避让键盘与 TabBar | 换 Symbol 图标 + 动态避让 |
| 🟡 | SystemAvatar.ets L26 | `imageLoadFailed` 不随 src 变化重置，换图后仍显示首字母；L143 首字母底色含 surfaceSubtle，白字对比度不足 | src watch 重置 + 收敛底色板 |
| 🟡 | SystemCard.ets L39 | `clickable` 只控制缩放，组件自身没有 onPress 回调，API 契约不完整 | 补 onPress |
| 🟡 | SystemTextArea.ets | 无 disabled prop，聚焦边框不变色，与 SystemInput 不对齐 | 对齐两者状态 |
| 🟡 | PageShellContainer vs ResponsiveContainer | 职责重叠且最大宽度体系不同（变体制 vs 断点制），并存易选错 | 合并或注释分工 |
| 🔵 | SoftDivider.ets L15 | 默认色无主题订阅，切主题后不刷新 | 加 @StorageProp |

✅ 亮点：Pressable 滑出取消（Move 越界释放按压态）是超出行业平均水准的细节；EmptyState 采用度高（18+ 页面）。

---

## 三、frontend（Svelte）问题清单

前端整体质量高于 harmonyapp（token 单源、focus-ring 统一、跳主内容链接、IME 回车保护、hit-slop 热区都有），问题集中在无障碍与少数语义错误：

| 级别 | 位置 | 问题 | 改进 |
|---|---|---|---|
| 🔴 | `components/ui/button.svelte:57` | loading 态用 `pointer-events-none` 而非 `disabled`——**挡不住键盘 Enter，可重复提交**（与注释矛盾） | loading 时同步 `disabled` |
| 🔴 | `components/messages/ChatInputBar.svelte:41-49` | 发送按钮仅靠变色区分空/非空态，未 disabled，读屏用户无法感知 | `disabled={!inputVal.trim()}` |
| 🔴 | `routes/(app)/messages/+page.svelte:191,301` | 桌面/移动两处「发起新会话」图标按钮缺 `aria-label` | 补标签 |
| 🟡 | `components/data-display/NearbyImageCard.svelte:64-87` | `role=button` 卡片内嵌套真实 `<button>`，违反 ARIA 交互嵌套 | 卡片改 `<a>` + 覆盖层按钮 |
| 🟡 | `routes/(app)/settings/+page.svelte:27-42` | 「推送通知」仅存 localStorage 却与服务端开关同列无标识，用户误以为已同步 | 加「仅本机」副文案 |
| 🟡 | `components/messages/MessageBubble.svelte:69-73` | 编辑态 textarea 无 aria-label、无 Esc 取消路径 | 补标签 + Esc |
| 🟡 | `routes/(app)/search/+page.svelte:142-144` | Tab 计数 `用户 ${count || ''}`：0 结果与加载中不可区分 | 加载中显式骨架 |
| 🔵 | `routes/(app)/me/edit/+page.svelte:255` | bio 超限计数用品牌 accent 色而非 warning/danger，语义不符 | 换语义色 |
| 🔵 | `nearby/moments/shared/MomentCard.svelte:48-50` | `item.tag` 为空仍渲染空 accent 胶囊 | `{#if item.tag}` |

---

### 后续进展（2026-09-10 前端全量复审）

上表第 1/2/3/8/9 条已修（loading 同步 disabled、发送按钮 disabled、图标按钮
aria-label 全量覆盖 + eslint 规则、bio 超限换 warning 色、空 tag 不再渲染）；
第 4 条部分修（点击穿透已处理，`role=button` 外层 + 内嵌真实 button 的 ARIA
交互嵌套仍在）。**第 5/6/7 条已在本批次修复**：

- 第 5 条：设置页「推送通知」补 `仅本机` 徽标（`SettingRow` 新增 `badge` snippet），
  与服务端开关明确区分。
- 第 6 条：`MessageBubble` 编辑态 textarea 补 `aria-label="编辑消息内容"` 与 Esc 取消。
- 第 7 条：搜索页 Tab 计数改为「搜索中不显示、完成后 0 也显示」，0 结果与加载中可区分。

同批次还修复了本次复审新发现的问题（11 个死类名、21 个无 hover 反馈按钮、
三条重复路由、标题档位四套并存、`window.confirm` 混用等），详见
`docs/ui-components.md`「已完成的收敛」。

---

## 四、改进方案（按优先级）

### P0 — 本周必修：正确性与危险操作（10 项）

1. **修复假交互/错误跳转**：NearbyUsersListPage「在线」tab、NearbyGroupsListPage 翻页丢筛选、GroupDetailPage 10400 误跳转、MomentsFeedPage 标签假过滤、GroupPage 已加入群显示「加入」。
2. **危险操作统一加确认弹窗**（共 8 处）：退出登录 ×2、删除账号、撤销分享、退群/删会话、撤销会话/全部下线、清理缓存、举报。建议封装 `SystemDialog.confirm({ title, danger: true })` 统一调用。
3. **修复虚假反馈**：AdvancedSettingsPage 清理缓存假成功；ProfilePage 相机选图后丢失；ProfileEditPage 「开发中」死入口（实现或隐藏）。
4. **修复颜色语义错误**：NotificationBanner 优先级倒挂（critical→error）、ChatMessageItem 自己消息时间戳近乎不可见、NotificationCenter 互动类误用 error 红。
5. **LoginPage 双重提交**：删除外层冗余 onClick；frontend button.svelte loading 同步 disabled。

### P1 — 体验基线：触控与反馈（2 周）

6. **触控目标达标**：SurfaceIconButton/返回键/清除钮/NotificationBanner 关闭钮从 36/16vp 扩到 ≥44vp 命中区（视觉尺寸不变，加透明 padding）。参考：HarmonyOS 设计规范最小热区 40vp、WCAG 2.5.5 / Apple HIG 44pt。
7. **聊天体验三件套**：发送乐观气泡 + 失败重发；仅贴底时 scrollToBottom +「↓新消息」悬浮钮；图片消息限高 + 点击预览 + 加载占位。
8. **骨架屏落地**：Skeleton 组件当前零引用，优先接入 MomentPage、MessagePage、ProfilePage 首屏，替换裸转圈。
9. **列表虚拟化**：MomentsFeedPage 瀑布流、MessagePage 会话列表、NearbyGroupsListPage 三处 ForEach → LazyForEach。
10. **库内收敛**：SystemButton/SystemCard/SurfaceIconButton 等 5 个组件收敛到 Pressable；SystemDialog 改 bindContentCover 并补进出场动画与按钮间距。

### P2 — 一致性与打磨（1 个月）

11. **数据展示诚实**：HomePage HeroStats 截断统计、MomentsFeedPage「N 条内容」、frontend search Tab 计数，统一改为服务端总数或加限定文案。
12. **状态管理去重**：SettingsPage 与 PrivacySettingsPage 重复开关、两处 logout 逻辑不一致、简介字数上限 500/150 不一致——收敛为单一数据源/常量。
13. **权限按需申请**：CreateMomentPage 进页 5 连申请改为触发时申请；CreateGroupPage/LocationPickerPage 定位失败加重试与去设置引导。
14. **空态与收尾统一**：所有列表底部加「没有更多了」；HomePage 单行灰字改 EmptyState 小号变体；各 Tab 空态文案按类型定制。
15. **无障碍补齐**：图标按钮 aria-label/accessibilityText、SystemBadge 未读数读屏语义、TabsPillGroup 选中态播报。

### 设计规范参考（落地基准）

- **触控目标**：HarmonyOS《UX 设计规范》最小热区 40vp；建议对齐 WCAG 2.5.5 Target Size 44×44。
- **对比度**：文字/背景对比度 ≥ 4.5:1（WCAG AA），辅助文字 ≥ 3:1——SystemAvatar 首字母底色、ChatMessageItem 时间戳需实测。
- **字体层级**：现有 10/11/12/14/16/20/24/32 八档已合规，保持不再新增近邻档。
- **按压反馈**：延续现有 SCALE_PRESSED_* + spring 曲线，全部可点元素统一走 Pressable，禁用系统默认 stateEffect 混用。
- **危险操作**： danger variant + 二次确认 + 异步确认 loading（防连点），参照 Material Design 3 destructive action 模式。
- **权限申请**：HarmonyOS 官方「按需申请、用时申请」原则，禁止进页批量申请。

---

## 五、问题分布统计

| 模块 | 🔴 高 | 🟡 中 | 🔵 低 |
|---|---|---|---|
| harmonyapp 页面 | 24 | 31 | 15 |
| harmonyapp 公共组件 | 4 | 6 | 3 |
| frontend | 3 | 4 | 2 |

高频根因（修复一类可消除一片）：
1. **危险操作无确认**（8 处）→ 一个统一 confirm 封装全部解决；
2. **假交互/死功能**（6 处）→ 建立「未实现功能不上线入口」的提交红线；
3. **触控目标不足**（5+ 处）→ 在 Pressable/SurfaceIconButton 层统一加命中区扩展即可全仓达标；
4. **数据截断冒充统计**（3 处）→ 统一从服务端 count 字段取数。

---

## 六、P0 修复记录（2026-08-30）

### P0-1 假交互/错误跳转
- `NearbyUsersListPage.ets`：「在线」tab 改为基于 `presence_status` 的客户端过滤（接口无在线参数，注释说明），加载更多保持过滤视图一致。
- `NearbyGroupsListPage.ets`：`loadMoreGroups` 透传 keyword/tag，翻页不再丢筛选。
- `GroupDetailPage.ets`：核查后端后确认 10400 为通用业务冲突（密码错误/群已满/已申请/仅限邀请共用），仅 message 含「已是群成员」才放行进入聊天，其余错误 Toast 不跳转。
- `MomentsFeedPage.ets`：标签药片实现真实客户端过滤（`getVisibleMoments`），过滤为空给出「查看全部」出口；与「热门」同逻辑的「活动」tab 已移除。
- `GroupPage.ets`：已加入群组按钮「加入」→「进入」，跳群组详情。

### P0-2 虚假反馈/死功能/重复提交
- `AdvancedSettingsPage.ets`：清理缓存加确认弹窗，并真正调用 `CacheManager`/`MessageCacheService`/`MomentCacheService` 的 clearAll（保留登录态与设备配置）。
- `ProfilePage.ets`：身份卡两处相机按钮接通真实上传链路（选图→压缩→上传→`updateProfile(avatar_file_id)`→刷新全局用户态）。
- `ProfileEditPage.ets`：头像入口接通同款上传；横幅因后端无 banner 字段，移除死入口并注释待后端支持。
- `LoginPage.ets`：删除登录按钮外层冗余 onClick，重复提交消除（注册/访客/验证码按钮已排查无同类问题）。

### P0-3 危险操作二次确认（7 处，均走 SystemDialog + danger 确认键 + 防连点）
退出登录 ×2（SettingsPage、ProfilePage）、删除账号（AccountsPage，按钮改 danger）、撤销分享（MySharesPage）、删除会话/退出群组（MessagePage）、撤销会话/全部下线（SafetySettingsPage）、举报（ChatPage）。

### P0-4 颜色语义/前端无障碍
- `NotificationBanner.ets`：优先级映射 critical→error / high→warning / medium→textSecondary / low→textTertiary；关闭按钮命中区 16→40vp（视觉不变）。
- `ChatMessageItem.ets`：自己消息气泡外时间戳 sendButtonActiveText→textTertiary，已读勾→palette.online。
- `NotificationCenter.ets`：互动类图标/未读点 error→accentBackground，未读淡底 dangerSoft→infoSoft。
- 前端：`button.svelte` loading 同步 disabled（阻断键盘重复提交）；`ChatInputBar.svelte` 空内容禁止发送；messages 页两处「发起新会话」补 aria-label。eslint 通过。

### 遗留说明
- harmonyapp 本机无 hvigor/DevEco，ArkTS 未做编译验证，改动均经逐处代码核验；建议在 DevEco 中全量构建一次。
- `GroupPage.ets` 的 `joinGroup()` 链现已无调用方（含同类 10400 误读），建议后续删除死代码。
- P1/P2 项（触控目标全量达标、聊天乐观气泡、骨架屏落地、列表虚拟化等）未动，按报告优先级继续。
