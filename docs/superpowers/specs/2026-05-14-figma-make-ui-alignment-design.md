# Figma Make UI 完全对齐设计

**日期**: 2026-05-14
**状态**: 已确认
**目标**: 将 Svelte 前端与 Figma Make WIP 设计 1:1 对齐，覆盖桌面端/Pad端/手机端 x 浅色/暗色模式的 6 种组合，支持动态切换。

---

## 设计权威

`/Users/liyou/Work/Hotspot_Make_WIP/` 为设计权威来源。色值、组件、动画参数、布局结构均以该项目的 React 实现为最终参考。

---

## Phase 1: 主题 Token 精确对齐

### 目标

将 `frontend_svelte/src/app.css` 中的 CSS 变量色值与 Figma Make 的 `src/styles/theme.css` 完全对齐。

### 差异清单

NEU 主题（浅色+暗色）：0 处差异，已完全对齐。

#### Forest 浅色（5 处色值差异 + 1 处 alpha）

| Token | Figma Make | Svelte 当前 | 修正为 |
|-------|-----------|------------|--------|
| `--neu-bg` | `#F9F9F5` | `#ffffff` | `#F9F9F5` |
| `--page-bg` | `#EDECEA` | `#ededed` | `#EDECEA` |
| `--neu-text` | `#1B1F1B` | `#1a1a1a` | `#1B1F1B` |
| `--neu-text-muted` | `#6A7B70` | `#8a9190` | `#6A7B70` |
| `--neu-glow` | `rgba(61,122,85,0.18)` | `rgba(61,122,85,0.15)` | `rgba(61,122,85,0.18)` |
| `--neu-border` | `rgba(61,122,85,0.13)` | `rgba(0,0,0,0.08)` | `rgba(61,122,85,0.13)` |

#### Forest 暗色（6 处色值差异）

| Token | Figma Make | Svelte 当前 | 修正为 |
|-------|-----------|------------|--------|
| `--neu-bg` | `#161A16` | `#1a1c1a` | `#161A16` |
| `--page-bg` | `#0E110E` | `#111311` | `#0E110E` |
| `--neu-text` | `#DCE8DE` | `#e4eae5` | `#DCE8DE` |
| `--neu-text-muted` | `#7A9080` | `#8aa096` | `#7A9080` |
| `--neu-glow` | `rgba(82,184,120,0.20)` | `rgba(82,184,120,0.18)` | `rgba(82,184,120,0.20)` |
| `--neu-border` | `rgba(82,184,120,0.12)` | `rgba(255,255,255,0.07)` | `rgba(82,184,120,0.12)` |

#### Vibrant 浅色（4 处差异）

| Token | Figma Make | Svelte 当前 | 修正为 |
|-------|-----------|------------|--------|
| `--neu-bg` | `#FAFAFE` | `#ffffff` | `#FAFAFE` |
| `--neu-shadow-dark` | `rgba(99,91,200,0.12)` | `rgba(99,91,200,0.1)` | `rgba(99,91,200,0.12)` |
| `--neu-glow` | `rgba(99,102,241,0.34)` | `rgba(99,102,241,0.32)` | `rgba(99,102,241,0.34)` |
| `--neu-border` | `rgba(99,102,241,0.15)` | `rgba(99,102,241,0.13)` | `rgba(99,102,241,0.15)` |

#### Vibrant 暗色（7 处差异）

| Token | Figma Make | Svelte 当前 | 修正为 |
|-------|-----------|------------|--------|
| `--neu-bg` | `#130F26` | `#141028` | `#130F26` |
| `--page-bg` | `#07040F` | `#09060f` | `#07040F` |
| `--neu-shadow-light` | `rgba(129,140,248,0.07)` | `rgba(99,102,241,0.06)` | `rgba(129,140,248,0.07)` |
| `--neu-shadow-dark` | `rgba(0,0,0,0.72)` | `rgba(0,0,0,0.65)` | `rgba(0,0,0,0.72)` |
| `--neu-text` | `#EAE6FF` | `#e8e4ff` | `#EAE6FF` |
| `--neu-glow` | `rgba(129,140,248,0.30)` | `rgba(129,140,248,0.26)` | `rgba(129,140,248,0.30)` |
| `--neu-border` | `rgba(129,140,248,0.14)` | `rgba(129,140,248,0.13)` | `rgba(129,140,248,0.14)` |

### 额外对齐项

以下 Figma Make 中定义但 Svelte 可能缺少的 token（需逐一核对补全）：

- `--neu-accent-soft` / `--neu-accent-soft-border`：Forest 和 Vibrant 的半透明强调色
- `--hover-shadow-card/row/btn/pill`：每个主题的 hover shadow 定义
- `--input-background` / `--switch-background`：shadcn 基础 token

### 影响范围

仅 `app.css`。所有组件通过 CSS 变量消费色值，修改即全局生效。

### 验证方式

6 种组合（3 主题 x 2 模式）各截图对比，确保无可见色差。

---

## Phase 2: 缺失页面实现

### 2.1 地图页 `/location`

**路由**: `(public)/location/+page.svelte`
**技术栈**: Leaflet + OpenStreetMap

#### 功能清单

- Leaflet 地图（中心点取 mapAnchorStore 坐标）
- 自定义 SVG marker（6 种：user/group/moment + cluster 变体），marker 文件放在 `src/lib/assets/markers/`
- Pin 类型过滤栏（用户/群组/动态，pill 形切换按钮）
- 搜索栏（顶部悬浮，调用 Nominatim API 或后端搜索接口）
- 可拖拽底部面板（仅 Mobile），3 档吸附：
  - MIN (148px)：仅搜索栏 + 确认按钮
  - MID (318px)：列表内容部分可见
  - MAX (540px)：完整列表可滚动
- Desktop: 全屏地图 + 右侧列表面板
- Tablet: 全屏地图 + 浮动底部面板（同 Mobile 但更宽）

#### 数据源

- 地图 pins：后端 API（nearby users/groups/moments）
- 地点搜索：Nominatim 或后端封装
- 底部面板列表：复用 pin 数据

#### 参考实现

Figma Make: `src/app/pages/Location.tsx` (~1207 行)

### 2.2 编辑资料页 `/profile/edit`

**路由**: `(protected)/profile/edit/+page.svelte`

#### 功能清单

- 头像上传（点击头像触发文件选择，预览裁剪）
- 表单字段：昵称、简介、个人标签
- NeuInput 样式（neumorphic convex/focus concave）
- 保存/取消按钮（NeuButton）
- 加载态和保存成功 toast
- 返回按钮（顶部）

#### 数据源

- 当前用户信息：`authStore.user`
- 保存：PUT `/api/v1/users/me` 或对应接口

#### 参考实现

Figma Make: `src/app/pages/EditProfile.tsx` (~178 行)

### 2.3 活动详情页 `/events/[id]`

**路由**: `(public)/events/[id]/+page.svelte`（需新建目录和路由）

#### 功能清单

- 活动信息卡片（封面图、标题、时间、地点、描述）
- 参与者头像列表（前 N 个 + 总数）
- 加入/退出按钮（带 tap 动画）
- 活动标签
- 返回按钮

#### 数据源

- 活动详情：后端 API
- 参与列表：后端 API
- 加入/退出：后端 API

#### 参考实现

Figma Make: `src/app/pages/EventDetail.tsx` (~225 行)

### 2.4 TabletRail 组件集

**文件**: `src/lib/components/common/tablet-rail/`

8 个卡片组件：

| 组件 | 功能 | 数据源 |
|------|------|--------|
| `NearbyPulseCard` | 实时附近人数 + 在线指示灯 | nearby API |
| `StreakCard` | 连续签到天数 + 7 天进度条 | gamification API |
| `VisitorsCard` | 今日访客头像 + 数量 | visitors API |
| `MoodCheckinCard` | 5 个心情 emoji 选择 | 本地状态 |
| `AchievementCard` | 下一徽章进度条 | gamification API |
| `SuggestedFollowsCard` | 3 个推荐用户 + 关注按钮 | 推荐算法 API |
| `DailyPickCard` | 每日精选内容卡片 | curated API |
| `VibeCard` | 当前天气 + 正在听的音乐 | 第三方 API |

**初始实现**：先用 mock 数据（同 Figma Make），后续接 API。

**渲染位置**：仅在 tablet 模式（600-1279px）下渲染，放在主页面的右侧 rail 区域。

**参考实现**：Figma Make: `src/app/components/TabletRail.tsx`

---

## Phase 3: 交互动画系统

### 技术方案

用 Svelte 原生能力实现，不引入外部动画库：

1. **`hoverConfig` store**：响应式 store，根据 `themeStore.family` 返回当前主题的动画配置对象
2. **Svelte actions**：`use:interactive` 指令封装 hover/tap 行为
3. **CSS transition**：用于简单过渡（颜色、border、opacity）
4. **`svelte/motion` 的 `spring()` / `tweened()`**：用于数值动画（拖拽面板、进度条）
5. **`svelte/transition`**：用于进出场动画（CreateSheet 滑入）

### 三种主题动画配置

#### Neu（暖砂 — 弹簧物理）

```
card: { y: -7, scale: 1.018, transition: spring(280, 18) }
row:  { y: -5, scale: 1.010, transition: spring(320, 20) }
btn:  { scale: 1.12,         transition: spring(380, 18) }
tap:  { scale: 0.94,         transition: spring(600, 28) }
```

#### Vibrant（星际 — 缓动）

```
card: { y: -5, scale: 1.014, transition: tween(220ms, [0.22,1,0.36,1]) }
row:  { y: -3, scale: 1.009, transition: tween(180ms, [0.22,1,0.36,1]) }
btn:  { scale: 1.10,         transition: tween(170ms, [0.22,1,0.36,1]) }
tap:  { scale: 0.94,         transition: tween(100ms, [0.22,1,0.36,1]) }
```

#### Forest（清野 — 纯缩放，无位移）

```
card: { scale: 1.016, transition: tween(200ms, [0.22,1,0.36,1]) }
row:  { scale: 1.010, transition: tween(160ms, [0.22,1,0.36,1]) }
btn:  { scale: 1.08,  transition: tween(160ms, [0.22,1,0.36,1]) }
tap:  { scale: 0.97,  transition: tween(100ms, [0.22,1,0.36,1]) }
```

### 实现细节

**文件结构**：

```
src/lib/stores/hover-config.svelte.ts    # 动画配置 store
src/lib/actions/interactive.ts           # use:interactive action
```

**`interactive` action 行为**：
- `mouseenter`：应用 hover 变换（y/translate + scale）
- `mouseleave`：恢复原位
- `mousedown`/`touchstart`：应用 tap 变换（scale 缩小）
- `mouseup`/`touchend`：恢复 tap 前状态
- 用 `CSS transition` 驱动过渡，不直接操作 JS 动画帧

**应用范围**：
- NeuButton / NeuCard：`use:interactive={{ type: 'btn' }}`
- 内容卡片：`use:interactive={{ type: 'card' }}`
- 列表行：`use:interactive={{ type: 'row' }}`
- NavTab：active pill 用 Svelte `animate:flip` 实现滑动指示器

### CreateSheet 动画

- 背景：CSS `opacity` 过渡 220ms
- 面板：Svelte `slide` / 自定义 `fly` transition，底部弹入
- FAB 图标切换：CSS `rotate` + `opacity` 过渡 180ms

### 地图页底部面板拖拽

- `bind:clientY` 追踪触摸位置
- `spring()` 驱动吸附动画
- 3 档吸附阈值

---

## Phase 4: 验收

### 截图对比验收

对 16 个核心页面（排除 admin），3 端 x 2 模式 = 96 组截图：

| 页面 | Mobile Light | Mobile Dark | Tablet Light | Tablet Dark | Desktop Light | Desktop Dark |
|------|-------------|-------------|--------------|-------------|---------------|--------------|
| Home | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Discover | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Messages | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| ChatDetail | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Profile | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| EditProfile | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Settings | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Location | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| UserProfile | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| PostDetail | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| EventDetail | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Notifications | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| NearbyAll | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| NearbyGroups | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Search | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| Login | ✓ | ✓ | - | - | ✓ | ✓ |

### 残余差异处理

发现的残余视觉差异在 Phase 4 中逐一修复，每个修复都需要截图对比验证。

---

## 不做的事

- 不改变 SvelteKit 路由结构（保持 `(public)/(protected)/(admin)/(auth)` 分组）
- 不引入 svelte-motion 或其他外部动画库
- 不修改后端 API（仅消费现有接口或新增前端 mock）
- 不做 Admin 页面的 Figma 对齐（Figma Make 中无 Admin 设计）
- 不做跨端距离/位置计算（保持后端负责的约束）
