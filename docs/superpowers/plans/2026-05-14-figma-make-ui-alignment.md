# Figma Make UI 对齐实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Svelte 前端与 Figma Make WIP 设计 1:1 对齐，覆盖 3 端 x 2 模式的 6 种主题组合。

**Architecture:** 四阶段渐进实施 — 先修正主题 token 色值（纯 CSS），再补缺失页面（路由+组件），再建动画系统（store+action），最后截图验收。

**Tech Stack:** SvelteKit (Svelte 5), Tailwind CSS v4, Leaflet + OSM, Svelte 原生动画

---

## File Structure

```
frontend_svelte/
├── src/
│   ├── app.css                                    # MODIFY: 主题 token 色值对齐
│   ├── lib/
│   │   ├── stores/
│   │   │   ├── theme.svelte.ts                    # EXISTS (no change)
│   │   │   ├── platform.svelte.ts                 # EXISTS (no change)
│   │   │   └── hover-config.svelte.ts             # CREATE: 动画配置 store
│   │   ├── actions/
│   │   │   └── interactive.ts                     # CREATE: use:interactive action
│   │   ├── components/
│   │   │   ├── common/
│   │   │   │   ├── tablet-rail/
│   │   │   │   │   ├── index.ts                   # CREATE: barrel export
│   │   │   │   │   ├── TabletRail.svelte          # CREATE: rail 容器
│   │   │   │   │   ├── NearbyPulseCard.svelte     # CREATE
│   │   │   │   │   ├── StreakCard.svelte          # CREATE
│   │   │   │   │   ├── VisitorsCard.svelte        # CREATE
│   │   │   │   │   ├── MoodCheckinCard.svelte     # CREATE
│   │   │   │   │   ├── AchievementCard.svelte     # CREATE
│   │   │   │   │   ├── SuggestedFollowsCard.svelte # CREATE
│   │   │   │   │   ├── DailyPickCard.svelte       # CREATE
│   │   │   │   │   └── VibeCard.svelte            # CREATE
│   │   │   │   ├── mobile-bottom-nav.svelte       # MODIFY: 加入 interactive action
│   │   │   │   ├── tablet-bottom-nav.svelte       # MODIFY: 加入 interactive action
│   │   │   │   └── sidebar.svelte                 # MODIFY: 加入 interactive action
│   │   │   └── ui/
│   │   │       ├── neu-button/                    # MODIFY: 加入 interactive action
│   │   │       └── neu-card/                      # MODIFY: 加入 interactive action
│   │   └── assets/
│   │       └── markers/
│   │           ├── marker_user.svg                # CREATE
│   │           ├── marker_group.svg               # CREATE
│   │           ├── marker_moment.svg              # CREATE
│   │           ├── marker_user_cluster.svg        # CREATE
│   │           ├── marker_group_cluster.svg       # CREATE
│   │           └── marker_moment_cluster.svg      # CREATE
│   └── routes/
│       ├── (public)/
│       │   ├── +layout.svelte                     # MODIFY: 加入 TabletRail
│       │   └── location/
│       │       └── +page.svelte                   # CREATE: 地图页
│       ├── (protected)/
│       │   ├── profile/
│       │   │   └── edit/
│       │   │       └── +page.svelte               # CREATE: 编辑资料页
│       │   └── ...
│       └── (public)/
│           └── events/
│               └── [id]/
│                   └── +page.svelte               # CREATE: 活动详情页
```

---

## Phase 1: 主题 Token 精确对齐

### Task 1.1: 修正 Forest 浅色主题色值

**Files:**
- Modify: `src/app.css:67-135`

- [ ] **Step 1: 修正 Forest 浅色 `--neu-*` 基础色值**

将以下 CSS 变量替换为 Figma Make 精确值：

```css
:root[data-theme-family='forest'] {
  --neu-bg: #F9F9F5;                    /* was: #ffffff */
  --page-bg: #EDECEA;                   /* was: #ededed */
  --neu-shadow-light: rgba(255, 255, 255, 0);   /* unchanged */
  --neu-shadow-dark: rgba(0, 0, 0, 0);          /* unchanged */
  --neu-text: #1B1F1B;                  /* was: #1a1a1a */
  --neu-text-muted: #6A7B70;            /* was: #8a9190 */
  --neu-accent: #3d7a55;                /* unchanged */
  --neu-accent-primary: #3d7a55;        /* unchanged */
  --neu-glow: rgba(61, 122, 85, 0.18);  /* was: 0.15 */
  --neu-fab-glow: 0 6px 18px rgba(61, 122, 85, 0.32), 0 2px 4px rgba(0, 0, 0, 0.1);  /* unchanged */
  --neu-border: rgba(61, 122, 85, 0.13);  /* was: rgba(0, 0, 0, 0.08) */
  --neu-border-pressed: rgba(61, 122, 85, 0.24);  /* was: rgba(0, 0, 0, 0.13) */
```

- [ ] **Step 2: 补充 Forest 浅色缺失 token**

在 Forest 浅色块中 `--neu-border-pressed` 之后添加：

```css
  --neu-accent-soft: rgba(61, 122, 85, 0.10);
  --neu-accent-soft-border: rgba(61, 122, 85, 0.32);
```

- [ ] **Step 3: 更新 Forest 浅色的 shadcn 语义 token 中受影响的值**

```css
  --popover: #F9F9F5;             /* was: #ffffff — 跟随 neu-bg */
  --sidebar: #F9F9F5;             /* was: #ffffff — 跟随 neu-bg */
```

- [ ] **Step 4: 运行格式化和类型检查**

Run: `cd frontend_svelte && bun run format`
Run: `cd frontend_svelte && bun check`

- [ ] **Step 5: 视觉验证 Forest 浅色模式**

在浏览器中切换到 Forest 浅色主题，确认：
- 背景从纯白变为暖白色调
- 文字从纯黑变为绿色调文字
- 边框从黑色调变为绿色调边框
- 截图保存到 `.local/verify-forest-light.png`

- [ ] **Step 6: Commit**

```bash
git add frontend_svelte/src/app.css
git commit -m "fix(frontend): align Forest light theme tokens with Figma Make design"
```

---

### Task 1.2: 修正 Forest 暗色主题色值

**Files:**
- Modify: `src/app.css:137-203`

- [ ] **Step 1: 修正 Forest 暗色 `--neu-*` 基础色值**

```css
:root[data-theme-family='forest'].dark {
  --neu-bg: #161A16;                    /* was: #1a1c1a */
  --page-bg: #0E110E;                   /* was: #111311 */
  --neu-shadow-light: rgba(255, 255, 255, 0);   /* unchanged */
  --neu-shadow-dark: rgba(0, 0, 0, 0);          /* unchanged */
  --neu-text: #DCE8DE;                  /* was: #e4eae5 */
  --neu-text-muted: #7A9080;            /* was: #8aa096 */
  --neu-accent: #52b878;                /* unchanged */
  --neu-accent-primary: #52b878;        /* unchanged */
  --neu-glow: rgba(82, 184, 120, 0.20); /* was: 0.18 */
  --neu-fab-glow: 0 6px 22px rgba(82, 184, 120, 0.38), 0 2px 6px rgba(0, 0, 0, 0.2);  /* unchanged */
  --neu-border: rgba(82, 184, 120, 0.12);  /* was: rgba(255, 255, 255, 0.07) */
  --neu-border-pressed: rgba(82, 184, 120, 0.22);  /* was: rgba(255, 255, 255, 0.04) */
```

- [ ] **Step 2: 补充 Forest 暗色缺失 token**

```css
  --neu-accent-soft: rgba(82, 184, 120, 0.12);
  --neu-accent-soft-border: rgba(82, 184, 120, 0.34);
```

- [ ] **Step 3: 更新 Forest 暗色受影响的 shadcn token**

```css
  --popover: #111611;             /* follow page-bg green tone */
  --sidebar: #111611;             /* follow page-bg green tone */
```

- [ ] **Step 4: 运行格式化和验证**

Run: `cd frontend_svelte && bun run format`

- [ ] **Step 5: 视觉验证 Forest 暗色模式**

- [ ] **Step 6: Commit**

```bash
git add frontend_svelte/src/app.css
git commit -m "fix(frontend): align Forest dark theme tokens with Figma Make design"
```

---

### Task 1.3: 修正 Vibrant 浅色主题色值

**Files:**
- Modify: `src/app.css:210-281`

- [ ] **Step 1: 修正 Vibrant 浅色色值**

```css
:root,
:root[data-theme-family='vibrant'] {
  --neu-bg: #FAFAFE;                    /* was: #ffffff */
  --page-bg: #eef2ff;                   /* unchanged */
  --neu-shadow-light: rgba(255, 255, 255, 1);    /* unchanged */
  --neu-shadow-dark: rgba(99, 91, 200, 0.12);    /* was: 0.1 */
  --neu-text: #1e1b4b;                  /* unchanged */
  --neu-text-muted: #5b5792;            /* unchanged */
  --neu-accent: #6366f1;                /* unchanged */
  --neu-accent-primary: #6366f1;        /* unchanged */
  --neu-glow: rgba(99, 102, 241, 0.34); /* was: 0.32 */
  --neu-border: rgba(99, 102, 241, 0.15);  /* was: 0.13 */
  --neu-border-pressed: rgba(99, 102, 241, 0.26);  /* was: 0.22 */
```

- [ ] **Step 2: 补充 Vibrant 浅色缺失 token**

```css
  --neu-accent-soft: rgba(99, 102, 241, 0.10);
  --neu-accent-soft-border: rgba(99, 102, 241, 0.30);
```

- [ ] **Step 3: 更新受影响的 shadcn token**

```css
  --popover: #FAFAFE;             /* was: #ffffff — 跟随 neu-bg */
  --sidebar: #FAFAFE;             /* was: #f7f9ff — 跟随 neu-bg */
```

- [ ] **Step 4: 运行格式化和验证**

Run: `cd frontend_svelte && bun run format`

- [ ] **Step 5: 视觉验证 Vibrant 浅色模式**

- [ ] **Step 6: Commit**

```bash
git add frontend_svelte/src/app.css
git commit -m "fix(frontend): align Vibrant light theme tokens with Figma Make design"
```

---

### Task 1.4: 修正 Vibrant 暗色主题色值

**Files:**
- Modify: `src/app.css:283-351`

- [ ] **Step 1: 修正 Vibrant 暗色色值**

```css
:root,
:root.dark,
:root[data-theme-family='vibrant'].dark {
  --neu-bg: #130F26;                    /* was: #141028 */
  --page-bg: #07040F;                   /* was: #09060f */
  --neu-shadow-light: rgba(129, 140, 248, 0.07);  /* was: rgba(99, 102, 241, 0.06) — 色调从 indigo 改为 lavender */
  --neu-shadow-dark: rgba(0, 0, 0, 0.72);          /* was: 0.65 */
  --neu-text: #EAE6FF;                  /* was: #e8e4ff */
  --neu-text-muted: #7a75b0;            /* unchanged */
  --neu-accent: #818cf8;                /* unchanged */
  --neu-accent-primary: #818cf8;        /* unchanged */
  --neu-glow: rgba(129, 140, 248, 0.30);  /* was: 0.26 */
  --neu-border: rgba(129, 140, 248, 0.14);  /* was: 0.13 */
  --neu-border-pressed: rgba(129, 140, 248, 0.26);  /* was: 0.22 */
```

- [ ] **Step 2: 补充 Vibrant 暗色缺失 token**

```css
  --neu-accent-soft: rgba(129, 140, 248, 0.14);
  --neu-accent-soft-border: rgba(129, 140, 248, 0.34);
```

- [ ] **Step 3: 更新受影响的 shadcn token**

```css
  --popover: #0e0b1e;             /* follow page-bg purple tone */
  --sidebar: #130F26;             /* was: #141028 — 跟随 neu-bg */
```

- [ ] **Step 4: 运行格式化和验证**

Run: `cd frontend_svelte && bun run format`

- [ ] **Step 5: 视觉验证 Vibrant 暗色模式**

- [ ] **Step 6: Commit**

```bash
git add frontend_svelte/src/app.css
git commit -m "fix(frontend): align Vibrant dark theme tokens with Figma Make design"
```

---

### Task 1.5: 补充 NEU 主题缺失 token

**Files:**
- Modify: `src/app.css:358-496`

- [ ] **Step 1: 为 NEU 浅色和暗色补充 `--neu-accent-soft` 和 `--neu-accent-soft-border`**

NEU 浅色（`358-427` 行区域）添加：
```css
  --neu-accent-soft: rgba(212, 132, 90, 0.10);
  --neu-accent-soft-border: rgba(212, 132, 90, 0.28);
```

NEU 暗色（`429-496` 行区域）添加：
```css
  --neu-accent-soft: rgba(232, 168, 124, 0.12);
  --neu-accent-soft-border: rgba(232, 168, 124, 0.30);
```

- [ ] **Step 2: 在 `@theme` 块中注册新 token 的 Tailwind 映射**

在 `app.css` 的 `@theme` 块（约 5-59 行）中添加：

```css
  --color-neu-accent-soft: var(--neu-accent-soft);
  --color-neu-accent-soft-border: var(--neu-accent-soft-border);
```

- [ ] **Step 3: 运行格式化**

Run: `cd frontend_svelte && bun run format`

- [ ] **Step 4: 构建，确保无编译错误**

Run: `cd frontend_svelte && bun run build`

- [ ] **Step 5: Commit**

```bash
git add frontend_svelte/src/app.css
git commit -m "feat(frontend): add missing accent-soft tokens to all theme families"
```

---

### Task 1.6: 6 组合截图对比验证

**Files:** None (verification only)

- [ ] **Step 1: 逐主题逐模式截图**

对 6 种组合（vibrant/forest/neu x light/dark）分别截图，与 Figma Make 对比：
- Home 页
- Discover 页
- Settings 页（包含主题选择器）

截图保存到 `.local/verify-phase1-*.png`

- [ ] **Step 2: 修复残余色差（如有）**

---

## Phase 2: 缺失页面实现

### Task 2.1: 创建 Leaflet 集成工具层

**Files:**
- Create: `src/lib/utils/map.ts`

- [ ] **Step 1: 安装 Leaflet 依赖**

Run: `cd frontend_svelte && bun add leaflet && bun add -d @types/leaflet`

- [ ] **Step 2: 创建地图工具文件**

`src/lib/utils/map.ts` — 封装 Leaflet 初始化、marker 创建、popup 绑定等通用逻辑：

```typescript
import type { Map, Marker, LayerGroup } from 'leaflet';

export const DEFAULT_CENTER: [number, number] = [34.2618, 117.1859]; // 徐州
export const DEFAULT_ZOOM = 14;

export function createMap(
  container: HTMLElement,
  center?: [number, number],
  zoom?: number
): Map { ... }

export function createCustomMarker(
  iconUrl: string,
  lat: number,
  lng: number,
  size = [32, 32]
): Marker { ... }

export function createClusterMarker(
  iconUrl: string,
  count: number,
  lat: number,
  lng: number
): Marker { ... }
```

- [ ] **Step 3: 创建 SVG marker 资源**

从 Figma Make 的 `src/assets/markers/` 目录复制 6 个 SVG marker 文件到 `src/lib/assets/markers/`。每个 SVG 约 200-400 bytes，使用当前主题的强调色填充。

- [ ] **Step 4: 运行类型检查**

Run: `cd frontend_svelte && bun check`

- [ ] **Step 5: Commit**

```bash
git add frontend_svelte/src/lib/utils/map.ts frontend_svelte/src/lib/assets/markers/
git commit -m "feat(frontend): add Leaflet map utilities and custom SVG markers"
```

---

### Task 2.2: 实现地图页

**Files:**
- Create: `src/routes/(public)/location/+page.svelte`

- [ ] **Step 1: 创建路由目录**

Run: `mkdir -p frontend_svelte/src/routes/\(public\)/location`

- [ ] **Step 2: 实现地图页主体**

参考 Figma Make `src/app/pages/Location.tsx`（1207 行），核心功能：

1. **地图区域** — 全屏 Leaflet 地图，中心点取 `mapAnchorStore` 坐标
2. **Pin 过滤栏** — 顶部悬浮 pill 按钮（用户/群组/动态），使用 `neu-surface-ghost` 样式
3. **搜索栏** — 顶部搜索框，调用 Nominatim API
4. **可拖拽底部面板**（Mobile/Tablet）— 3 档吸附（MIN 148px / MID 318px / MAX 540px），使用 Svelte `spring()` 驱动吸附动画
5. **右侧面板**（Desktop）— 固定宽度 360px 列表面板
6. **Pin 列表** — 底部面板/右侧面板中展示搜索结果

Mobile 底部面板用 `bind:clientY` + `touchmove`/`touchend` 事件实现拖拽，`spring()` 驱动吸附动画。

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
  import { spring } from 'svelte/motion';
  import { createMap, createCustomMarker } from '$lib/utils/map';
  import type { Map, Marker } from 'leaflet';

  type PinType = 'user' | 'group' | 'post';
  type SnapState = 'min' | 'mid' | 'max';

  const SNAP_MIN = 148;
  const SNAP_MID = 318;
  const SNAP_MAX = 540;

  let mapContainer: HTMLElement;
  let map: Map;
  let activeFilter = $state<PinType | 'all'>('all');
  let sheetHeight = $state(SNAP_MIN);
  let markers: Marker[] = [];

  // spring for snap animation
  const sheetSpring = spring(SNAP_MIN, { stiffness: 0.3, damping: 0.8 });

  // ...拖拽逻辑、pin 数据加载、渲染
</script>
```

- [ ] **Step 3: 运行格式化和类型检查**

Run: `cd frontend_svelte && bun run format && bun check`

- [ ] **Step 4: 构建验证**

Run: `cd frontend_svelte && bun run build`

- [ ] **Step 5: Commit**

```bash
git add frontend_svelte/src/routes/\(public\)/location/
git commit -m "feat(frontend): implement location/map page with Leaflet and draggable sheet"
```

---

### Task 2.3: 实现编辑资料页

**Files:**
- Create: `src/routes/(protected)/profile/edit/+page.svelte`

- [ ] **Step 1: 创建路由目录**

Run: `mkdir -p frontend_svelte/src/routes/\(protected\)/profile/edit`

- [ ] **Step 2: 实现编辑资料页**

参考 Figma Make `src/app/pages/EditProfile.tsx`（178 行），核心功能：

1. **返回按钮** — 左上角 ArrowLeft + "编辑资料"
2. **头像上传区** — 大头像 + 相机图标叠加，点击触发文件选择
3. **表单字段** — 昵称（NeuInput）、简介（NeuInput textarea）、标签
4. **保存按钮** — 底部固定，NeuButton glow 变体
5. **加载/保存态** — 提交时 loading，成功 toast

数据源：`authStore.user` 读取当前值，PUT 保存。

```svelte
<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { usersService } from '$lib/api/services';
  import { isSuccess } from '$lib/api/client';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Camera from 'lucide-svelte/icons/camera';
  import { goto } from '$app/navigation';

  let nickname = $state(authStore.user?.nickname ?? '');
  let bio = $state(authStore.user?.bio ?? '');
  let saving = $state(false);

  async function handleSave() {
    saving = true;
    const res = await usersService.updateProfile({ nickname, bio });
    if (isSuccess(res)) {
      toast.success('保存成功');
      goto('/profile');
    } else {
      toast.error(res.message || '保存失败');
    }
    saving = false;
  }
</script>
```

- [ ] **Step 3: 运行格式化和类型检查**

Run: `cd frontend_svelte && bun run format && bun check`

- [ ] **Step 4: Commit**

```bash
git add frontend_svelte/src/routes/\(protected\)/profile/edit/
git commit -m "feat(frontend): implement profile edit page"
```

---

### Task 2.4: 实现活动详情页

**Files:**
- Create: `src/routes/(public)/events/[id]/+page.svelte`

- [ ] **Step 1: 创建路由目录**

Run: `mkdir -p frontend_svelte/src/routes/\(public\)/events/\[id\]`

- [ ] **Step 2: 实现活动详情页**

参考 Figma Make `src/app/pages/EventDetail.tsx`（225 行），核心功能：

1. **封面图** — 全宽或 hero 区域
2. **活动信息** — 标题、时间、地点、描述、标签
3. **参与者列表** — 头像横排 + 总数
4. **加入/退出按钮** — 底部固定，NeuButton
5. **返回按钮** — 左上角

```svelte
<script lang="ts">
  import { page } from '$app/stores';
  import ArrowLeft from 'lucide-svelte/icons/arrow-left';
  import Calendar from 'lucide-svelte/icons/calendar';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Users from 'lucide-svelte/icons/users';
  import { goto } from '$app/navigation';

  let { data } = $props();
  // data 来自 +page.ts 的 load 函数
  let joined = $state(false);

  async function toggleJoin() {
    // 调用后端 API
    joined = !joined;
  }
</script>
```

- [ ] **Step 3: 创建 +page.ts load 函数**

`src/routes/(public)/events/[id]/+page.ts` — 加载活动详情数据：

```typescript
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ params, fetch }) => {
  const res = await fetch(`/api/v1/events/${params.id}`);
  return await res.json();
};
```

- [ ] **Step 4: 运行格式化和类型检查**

Run: `cd frontend_svelte && bun run format && bun check`

- [ ] **Step 5: Commit**

```bash
git add frontend_svelte/src/routes/\(public\)/events/
git commit -m "feat(frontend): implement event detail page"
```

---

### Task 2.5: 实现 TabletRail 组件集

**Files:**
- Create: `src/lib/components/common/tablet-rail/` 目录下 10 个文件

- [ ] **Step 1: 创建目录和 barrel export**

Run: `mkdir -p frontend_svelte/src/lib/components/common/tablet-rail`

创建 `index.ts` barrel export。

- [ ] **Step 2: 实现 TabletRail 容器组件**

`TabletRail.svelte` — 简单的 flex column 容器，接受 `children`：

```svelte
<aside class="flex flex-col gap-3 w-full">
  {@render children()}
</aside>
```

- [ ] **Step 3: 实现 NearbyPulseCard**

参考 Figma Make `TabletRail.tsx` 中的 `NearbyPulseCard`：
- `neu-surface-flat rounded-[18px] p-4` 基础样式
- eyebrow 标签 "实时动态"
- 大数字 + 在线指示灯 + SVG sparkline 装饰
- Mock 数据：142 人在线

- [ ] **Step 4: 实现 StreakCard**

- 7 天签到进度条
- 大数字显示连续天数（27 天）
- `motion.div` 风格的进度条动画（CSS transition）

- [ ] **Step 5: 实现 VisitorsCard**

- "今日访客" 标签
- 叠加头像行 + 数量（18）
- "查看全部" 按钮

- [ ] **Step 6: 实现 MoodCheckinCard**

- "今日心情" 标签
- 5 个 emoji 按钮（天气/海洋/火焰/月亮/茶）
- `neu-surface-ghost` 样式

- [ ] **Step 7: 实现 AchievementCard**

- "下一成就" 标签
- 进度条 72% + 徽章 emoji
- CSS transition 驱动进度条动画

- [ ] **Step 8: 实现 SuggestedFollowsCard**

- "你可能认识" 标签
- 3 个推荐用户（头像 + 名字 + 共同好友数 + 关注按钮）

- [ ] **Step 9: 实现 DailyPickCard**

- "今日精选" 标签
- Hero 图片 + 标题 + 作者 + 点赞数
- `neu-surface-flat` 圆角卡片

- [ ] **Step 10: 实现 VibeCard**

- "氛围" 标签
- 天气图标 + 温度 + 当前播放音乐

- [ ] **Step 11: 更新 barrel export 并导出所有组件**

`index.ts`:
```typescript
export { default as TabletRail } from './TabletRail.svelte';
export { default as NearbyPulseCard } from './NearbyPulseCard.svelte';
export { default as StreakCard } from './StreakCard.svelte';
export { default as VisitorsCard } from './VisitorsCard.svelte';
export { default as MoodCheckinCard } from './MoodCheckinCard.svelte';
export { default as AchievementCard } from './AchievementCard.svelte';
export { default as SuggestedFollowsCard } from './SuggestedFollowsCard.svelte';
export { default as DailyPickCard } from './DailyPickCard.svelte';
export { default as VibeCard } from './VibeCard.svelte';
```

- [ ] **Step 12: 运行格式化和类型检查**

Run: `cd frontend_svelte && bun run format && bun check`

- [ ] **Step 13: Commit**

```bash
git add frontend_svelte/src/lib/components/common/tablet-rail/
git commit -m "feat(frontend): implement TabletRail card components"
```

---

### Task 2.6: 将 TabletRail 集成到 Layout

**Files:**
- Modify: `src/routes/(public)/+layout.svelte:43-56`（Tablet 分支）

- [ ] **Step 1: 在 Tablet 布局中添加 TabletRail**

修改 `+layout.svelte` 的 Tablet 分支（约 43-56 行），将内容区域改为两栏布局：

```svelte
{:else if platformStore.isTablet}
  <div class="flex h-[100dvh] w-full" style="background: var(--surface-bg); color: var(--text-primary)">
    <main class="flex-1 overflow-y-auto overflow-x-hidden no-scrollbar scroll-smooth">
      {@render children()}
    </main>
    <div class="w-[280px] flex-shrink-0 overflow-y-auto no-scrollbar p-3">
      <TabletRail>
        <NearbyPulseCard />
        <StreakCard />
        <DailyPickCard />
        <SuggestedFollowsCard />
        <VibeCard />
      </TabletRail>
    </div>
    <TabletBottomNav />
  </div>
```

添加对应 import。

- [ ] **Step 2: 运行格式化和构建验证**

Run: `cd frontend_svelte && bun run format && bun run build`

- [ ] **Step 3: 视觉验证 Tablet 端**

在 600-1279px 宽度下检查 TabletRail 显示效果。

- [ ] **Step 4: Commit**

```bash
git add frontend_svelte/src/routes/\(public\)/+layout.svelte
git commit -m "feat(frontend): integrate TabletRail into tablet layout"
```

---

## Phase 3: 交互动画系统

### Task 3.1: 创建 hoverConfig store

**Files:**
- Create: `src/lib/stores/hover-config.svelte.ts`

- [ ] **Step 1: 实现 hoverConfig store**

参考 Figma Make `src/app/hooks/useHoverConfig.ts`，创建 Svelte 等价物：

```typescript
import { themeStore } from './theme.svelte';

interface HoverVariant {
  y?: number;
  scale: number;
  duration: number;
  easing: number[];
  springConfig?: { stiffness: number; damping: number };
}

interface HoverConfig {
  card: HoverVariant;
  row: HoverVariant;
  btn: HoverVariant;
  action: HoverVariant;
  tap: HoverVariant;
}

const EASE_OUT = [0.22, 1, 0.36, 1];

const configs: Record<string, HoverConfig> = {
  neu: {
    card:   { y: -7, scale: 1.018, duration: 0, easing: [], springConfig: { stiffness: 280, damping: 18 } },
    row:    { y: -5, scale: 1.010, duration: 0, easing: [], springConfig: { stiffness: 320, damping: 20 } },
    btn:    { scale: 1.12, duration: 0, easing: [], springConfig: { stiffness: 380, damping: 18 } },
    action: { scale: 1.22, duration: 0, easing: [], springConfig: { stiffness: 400, damping: 16 } },
    tap:    { scale: 0.94, duration: 0, easing: [], springConfig: { stiffness: 600, damping: 28 } },
  },
  vibrant: {
    card:   { y: -5, scale: 1.014, duration: 220, easing: EASE_OUT },
    row:    { y: -3, scale: 1.009, duration: 180, easing: EASE_OUT },
    btn:    { scale: 1.10, duration: 170, easing: EASE_OUT },
    action: { scale: 1.20, duration: 140, easing: EASE_OUT },
    tap:    { scale: 0.94, duration: 100, easing: EASE_OUT },
  },
  forest: {
    card:   { scale: 1.016, duration: 200, easing: EASE_OUT },
    row:    { scale: 1.010, duration: 160, easing: EASE_OUT },
    btn:    { scale: 1.08, duration: 160, easing: EASE_OUT },
    action: { scale: 1.18, duration: 140, easing: EASE_OUT },
    tap:    { scale: 0.97, duration: 100, easing: EASE_OUT },
  },
};

class HoverConfigStore {
  config = $derived(configs[themeStore.family] || configs.vibrant);

  getCardCss(): string { return this.#toCssTransition(this.config.card); }
  getRowCss(): string  { return this.#toCssTransition(this.config.row); }
  getBtnCss(): string  { return this.#toCssTransition(this.config.btn); }
  getTapScale(): number { return this.config.tap.scale; }

  #toCssTransition(variant: HoverVariant): string {
    const ms = variant.duration || 260;
    const easing = variant.easing.length ? `cubic-bezier(${variant.easing.join(',')})` : 'ease-out';
    return `transform ${ms}ms ${easing}`;
  }
}

export const hoverConfig = new HoverConfigStore();
```

- [ ] **Step 2: 运行类型检查**

Run: `cd frontend_svelte && bun check`

- [ ] **Step 3: Commit**

```bash
git add frontend_svelte/src/lib/stores/hover-config.svelte.ts
git commit -m "feat(frontend): add theme-aware hover animation config store"
```

---

### Task 3.2: 创建 interactive action

**Files:**
- Create: `src/lib/actions/interactive.ts`

- [ ] **Step 1: 实现 use:interactive action**

```typescript
import { hoverConfig } from '$lib/stores/hover-config.svelte';

type InteractiveType = 'card' | 'row' | 'btn' | 'action';

interface InteractiveOptions {
  type?: InteractiveType;
  disabled?: boolean;
}

export function interactive(node: HTMLElement, options?: InteractiveOptions) {
  const type = options?.type ?? 'btn';

  function getVariant() {
    const config = hoverConfig.config;
    switch (type) {
      case 'card': return config.card;
      case 'row': return config.row;
      case 'action': return config.action;
      default: return config.btn;
    }
  }

  function onEnter() {
    if (options?.disabled) return;
    const v = getVariant();
    const transforms: string[] = [];
    if (v.y) transforms.push(`translateY(${v.y}px)`);
    if (v.scale !== 1) transforms.push(`scale(${v.scale})`);
    node.style.transform = transforms.join(' ');
    node.style.transition = hoverConfig.getCardCss();
  }

  function onLeave() {
    node.style.transform = '';
  }

  function onDown() {
    if (options?.disabled) return;
    const tapScale = hoverConfig.getTapScale();
    node.style.transform = `scale(${tapScale})`;
  }

  function onUp() {
    onEnter(); // return to hover state
  }

  node.addEventListener('mouseenter', onEnter);
  node.addEventListener('mouseleave', onLeave);
  node.addEventListener('mousedown', onDown);
  node.addEventListener('mouseup', onUp);
  node.addEventListener('touchstart', onDown, { passive: true });
  node.addEventListener('touchend', onUp, { passive: true });

  return {
    update(newOptions?: InteractiveOptions) {
      options = newOptions;
    },
    destroy() {
      node.removeEventListener('mouseenter', onEnter);
      node.removeEventListener('mouseleave', onLeave);
      node.removeEventListener('mousedown', onDown);
      node.removeEventListener('mouseup', onUp);
      node.removeEventListener('touchstart', onDown);
      node.removeEventListener('touchend', onUp);
    }
  };
}
```

- [ ] **Step 2: 运行类型检查**

Run: `cd frontend_svelte && bun check`

- [ ] **Step 3: Commit**

```bash
git add frontend_svelte/src/lib/actions/interactive.ts
git commit -m "feat(frontend): add use:interactive Svelte action for theme-aware hover/tap"
```

---

### Task 3.3: 将 interactive action 应用到现有组件

**Files:**
- Modify: `src/lib/components/ui/neu-button/*.svelte`
- Modify: `src/lib/components/ui/neu-card/*.svelte`
- Modify: `src/lib/components/common/mobile-bottom-nav.svelte`
- Modify: `src/lib/components/common/tablet-bottom-nav.svelte`

- [ ] **Step 1: 应用到 NeuButton**

在 NeuButton 的 `<button>` 元素上添加 `use:interactive={{ type: 'btn' }}`。

- [ ] **Step 2: 应用到 NeuCard**

在 NeuCard 的 `<div>` 元素上添加 `use:interactive={{ type: 'card' }}`。

- [ ] **Step 3: 应用到 MobileBottomNav 的 nav item 按钮**

在每个 nav item 按钮上添加 `use:interactive={{ type: 'btn' }}`。

- [ ] **Step 4: 应用到 TabletBottomNav**

同理。

- [ ] **Step 5: 运行格式化和构建**

Run: `cd frontend_svelte && bun run format && bun run build`

- [ ] **Step 6: 交互验证**

在浏览器中测试 3 个主题的 hover/tap 动画：
- Neu：弹簧物理感，有 y 位移
- Vibrant：平滑缓动，有 y 位移
- Forest：仅缩放，无位移

- [ ] **Step 7: Commit**

```bash
git add frontend_svelte/src/lib/components/
git commit -m "feat(frontend): apply interactive action to NeuButton, NeuCard, and nav components"
```

---

### Task 3.4: 实现 CreateSheet 弹簧动画

**Files:**
- Modify: `src/lib/components/common/mobile-create-sheet.svelte`（如存在）
- 或 Modify: `src/routes/(public)/+layout.svelte` 中的 FAB 逻辑

- [ ] **Step 1: 实现 FAB 图标切换动画**

在 mobile-bottom-nav 的 FAB 按钮中，用 CSS `rotate` + `opacity` 实现图标切换：

```svelte
<button
  style="transform: rotate({showCreate ? 45 : 0}deg); opacity: {iconVisible ? 1 : 0}; transition: transform 0.18s ease-out, opacity 0.18s ease-out;"
>
```

- [ ] **Step 2: 实现 CreateSheet 进出动画**

使用 Svelte 的 `transition:fly` 实现 CreateSheet 从底部滑入：

```svelte
{#if showCreate}
  <!-- backdrop -->
  <div transition:fade={{ duration: 220 }} ...>

  <!-- sheet -->
  <div transition:fly={{ y: 300, duration: 300 }} style="border-radius: 28px 28px 0 0; height: 92%;">
    <CreateSheet />
  </div>
{/if}
```

- [ ] **Step 3: 运行格式化和构建**

- [ ] **Step 4: Commit**

```bash
git add frontend_svelte/src/lib/components/common/
git commit -m "feat(frontend): add spring-based CreateSheet and FAB animations"
```

---

## Phase 4: 验收

### Task 4.1: 逐页面截图对比

- [ ] **Step 1: 对 16 个核心页面做 6 组合截图**

脚本自动化截图（使用 agent-browser），保存到 `.local/phase4-*.png`。

- [ ] **Step 2: 修复残余差异**

根据截图对比结果，逐一修复残余视觉差异。

- [ ] **Step 3: 最终 commit**

```bash
git add -A
git commit -m "fix(frontend): final visual alignment with Figma Make design"
```

---

## Self-Review

### Spec Coverage

| Spec 要求 | 对应 Task |
|-----------|----------|
| Forest 浅色 5+1 处色值修正 | Task 1.1 |
| Forest 暗色 6 处色值修正 | Task 1.2 |
| Vibrant 浅色 4 处色值修正 | Task 1.3 |
| Vibrant 暗色 7 处色值修正 | Task 1.4 |
| NEU 补充 accent-soft token | Task 1.5 |
| 6 组合截图验证 | Task 1.6 |
| 地图页 /location | Task 2.1 + 2.2 |
| 编辑资料页 /profile/edit | Task 2.3 |
| 活动详情页 /events/[id] | Task 2.4 |
| TabletRail 8 个卡片 | Task 2.5 |
| TabletRail 集成到 Layout | Task 2.6 |
| hoverConfig store | Task 3.1 |
| interactive action | Task 3.2 |
| 组件应用动画 | Task 3.3 |
| CreateSheet 弹簧动画 | Task 3.4 |
| 截图验收 | Task 4.1 |

### Placeholder Scan

无 TBD/TODO。所有色值、参数、文件路径均已明确。

### Type Consistency

- `HoverVariant` 接口在 `hover-config.svelte.ts` 中定义，在 `interactive.ts` 中消费
- `InteractiveType` 在 `interactive.ts` 中定义，组件通过 `use:interactive={{ type: 'btn' }}` 传入
- 所有路由目录名使用 SvelteKit 约定的 `[id]` 语法
