# 前端质量门与性能基线（hotspot web）

本文件是 skill `hotspot-frontend-style` 的明细补充：完整命令、验收口径与实测基线。骨架/红线在 SKILL.md「质量门与性能红线」一节。

## 1. UI 审计命令

- 每次改 UI 后：`bun run audit:a11y`（axe 0 serious/critical）、`bun run audit:touch`（390 宽触控无未登记离群）、`bun run audit:ui`（含 browser-qa）。
- reduced-motion：`bun run audit:a11y -- --mobile --reduced`（残留动画类/长动画即失败）。
- 触控例外只允许走 `scripts/ui-quality-touch-allowlist.json`：稳定 selector + 理由，禁止 `nth-child`；维护说明见 `docs/frontend-ui-audit.md`。
- 这些命令依赖 dev 环境 + agent-browser，**不进 `check` 链**；`--limit` / `--only` 仅调试用。

## 2. 主题矩阵（合并前）

- `bun run audit:a11y -- --matrix --parallel 2`：约 15–20min，要求 0 serious/critical；默认 `audit:ui` 仍只扫 vibrant light。
- 三主题（vibrant/forest/inkwash）× 明暗的关键页必须实测；软岛/画布亮度比与文字对比记录基线（≥4.5）。
- 状态色/徽章对比（单点权威）：状态徽章用实底 + on-accent 字（淡底 + 主题字在部分主题/明暗下会 <4.5，axe color-contrast 会拦）；改状态色/新增语义色需 3 主题×明暗 ≥4.5（矩阵 + `python3 harmonyapp/scripts/check_status_parity.py` 双端校验）。token 定义与 on-soft 用法见 skill `hotspot-frontend-theme`。

## 3. axe 可访问性结构基线（0 serious / 0 critical）

- 整卡可点必须用「覆盖按钮 + 行内动作 auto-lift」，卡片根不得持 button/aria-label（nested-interactive）。
- `role=list` 子节点补 `role=listitem`；listitem 不挂 aria-label；`dl` 只包 dt/dd；无 role 的 div 不挂 aria-label。
- 每页有 `main`；可滚动非键盘可达容器加 `tabindex=0` + `role=region` + `aria-label`；装饰图 `alt=""` + `aria-hidden`。
- 新增可点结构先跑 axe 再加；扫描方式为 axe-core 注入（`--init-script`）。
- 键盘验收：Tab 全站可达、skip link 首 Tab 可见；去描边不得丢焦点环；父容器 overflow 不得裁切；亮/暗都测（键盘截图 + computed outline）。

## 4. 性能实测与定位方法（可重跑）

**GPU 进程定位**（agent-browser 打开页面后，按 profile 找到本会话 Chrome 的 GPU 进程）：

```bash
PROFILE=$(ls -dt /var/folders/*/*/T/agent-browser-chrome-* | head -1)
MAIN=$(ps -axo pid,command | grep -- "--user-data-dir=$PROFILE" | grep -v type= | awk "{print \$1}" | head -1)
# GPU 进程 = MAIN 的 --type=gpu-process 子进程；top -l 5 -s 2 -pid $GPU -stats pid,cpu 取平均
```

- 页面侧计数：`document.getAnimations().length`、`getComputedStyle(e).backdropFilter !== "none"` 的元素数、`willChange !== "auto"` 的元素数（内容页都应在 0 / 个位数）。
- 单变量 A/B：用注入 `<style>` 的方式分别关全部动画 / 关 backdrop-filter / 关 will-change，对比 GPU 进程 CPU。
- 源码契约守卫：`src/styles/__tests__/perf-contract.test.ts`（改 CSS 后必须过）。

**首页 1440x900（mock 登录，headless Chrome GPU 进程 CPU%）实测基线**：

| 场景 | 修复前 | 修复后 |
|---|---|---|
| 空闲 | 24.6% | **0.0%** |
| 滚动 | 19.3% | 10.5%（对照：再关 backdrop = 6.1%） |
| 动画数 / backdrop 元素 / will-change 元素 | 3 / 24 / 79 | 0 / 6 / 12 |

数据与量测脚本见 `docs/frontend-ui-audit.md`「性能」节。

## 5. 页面进场与预取（运行时响应）

- 进场预算：客户端导航 ≤120ms、起点 `opacity ≥0.6` + 位移 ≤2px（禁止 opacity:0 起）；骨架→内容 fade ≤110ms；首屏不播；只包内容层。
- 时长双源锁定：CSS `.page-enter/.content-fade`（`--cst-page-enter-duration` / `--cst-content-fade-duration`）↔ `lib/motion.ts` 的 `MOTION.enter/contentFade`；`motion.test.ts` 源码锁定，改一处必须同步。
- 预取：`data-sveltekit-preload-data` 只覆盖 `<a>`；button + goto 导航必须经 `lib/nav-preload.ts`（hover/focus/touchstart 触发 `preloadRoute()` + 5s 空闲兜底 `scheduleIdlePreload()`；外壳 `<svelte:window>` 顶层事件委托）。
- dev 必须配 `vite.config.ts` `server.warmup.clientFiles`（全部路由 + 布局/消息/导航组件；glob `src/routes/*/**` 绕开 `(app)` 括号陷阱），否则首访路由按需编译 58–720ms。只用 preloadCode（本应用无路由 load 数据）。

## 6. 滚动与合成层

- 唯一滚动容器 `main` 禁止 `scroll-smooth`：会把 scrollTop 赋值/无参 scrollTo 动画化 766ms；平滑只给显式入口（BackToTop 传 `behavior:'smooth'`）。键盘 PageDown 的 ~134ms 为浏览器原生，勿回退。
- `will-change` 纪律（按 `utilities.css` 现行策略）：**只在 `@media (hover: hover) and (pointer: fine)` 内**允许 `.card-hover`（整卡 hover 抬升）与 `.image-hover-frame__image`（卡片配图缩放）常驻 `transform` 合成层——卡片层与配图层各约 -15% 快速跨卡晃动 GPU 进程 CPU（叠加约 -28%），空闲 GPU/内存无回归；基础规则（媒体查询外）不得带 `will-change`。
- 其余一律不加：`.hover-btn` / `.hover-action`（首页实测 48 个实例 = 48 个常驻层，收益为零）、头像入口 `.avatar-entry__avatar img`、批量小图标（过渡开始时浏览器会自动提升）。源码契约守卫 `src/styles/__tests__/perf-contract.test.ts`：`.hover-action` 不得含 `will-change`；`.card-hover`/`.image-hover-frame__image` 的基础规则不得带、媒体查询内必须带 `will-change: transform`（改 CSS 后必须过）。

## 7. 验收纪律

- normal/reduced 双路径 computed + 中间态截图；无动画类/事件/定时器残留。
- 动画 ≤260ms；动效 QA 建议在 reduced-motion 下取基线；QA 截图不得因动效抖动。
- 不得动画首屏 LCP 元素；不得引入 CLS（图片显式尺寸/懒加载/骨架零位移）；LCP/CLS 采样在本地 dev 仅作相对参考。
