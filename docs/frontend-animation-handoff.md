# 前端动效交接（灵动岛与页头）

> 本文件用于跨会话接手：记录**当前已生效的灵动岛动效基线**、**滚动交接模块现状**、**未完成项**、**复测命令**。
> 动效契约的权威描述在 `docs/design-system.md` §10 / §22；页面布局契约在 skill `hotspot-frontend-layout`。
> 本文件不是规范，不要把它当作设计依据——规范以 design-system 为准。

## 0. 当前基线（勿回退）

- **灵动岛模型**：桌面端「默认常驻 + 点击向下展开控制中枢」，与滚动无关；移动/平板不渲染岛。
  形态/尺寸/交互契约见 §22。唯一形态实现 `frontend/src/components/layout/page/IslandCapsule.svelte`，
  业务组合 `PageIsland.svelte`；首页 `HomeDesktop`、搜索页 `search/+page.svelte` 直接注册 store。
- **展开/收起连续 morph**（用户点展开箭头触发，220ms `--cst-ease-standard`）：
  - 常驻 `rounded-lg`(20px) → 展开 `rounded-xl`(24px)，**只插值 20→24px**；
  - 宽度 260/290→440/460 用 `min-width`/`max-width` 成对插值（`width:auto` 不可插值；显式 `w-*` 会瞬跳）；
  - 高度由主体 `slide` 驱动；**头部行两态常驻**（只 34→44px 增高 + 原位换展开箭头/关闭钮），
    禁止整块内容换位（否则首帧闪出 2px 边框细线、标题消失后再出现）；
  - 禁止用 `rounded-full`（9999px）做常驻圆角：浏览器按半高钳制，过渡末帧圆角骤降 →「大圆角突然变小小圆角」。
- **hover / focus**：只由胶囊外壳承载（实底 + accent 细边 + `shadow-raised-sm` + scale 1.01）；
  标题区不画独立灰底（显式 `hover:bg-transparent` 退出全局按钮兜底填充）；展开箭头保留自身圆形淡底；
  除展开 morph 外不改宽度、不位移布局。键盘聚焦时标题按钮有可见 focus ring。
- **阅读进度圆环**：最右 18px，每帧直接写 `stroke-dashoffset`，**无 CSS 过渡**；同一读数只此一处。
- **meta 文本**：变化时 120ms 淡入（`animate-fade-in`）。
- **页头**：桌面端标准 `PageHeader` 默认 `desktopVisible=false` → `lg:hidden`，上下文由岛承担；
  `/messages` 左栏传 `desktopVisible` 是登记例外；移动/平板页头照常（岛不渲染）。
- **reduced-motion**：全局 `@media (prefers-reduced-motion: reduce)` 把动画/过渡瞬时化；新动效必须走
  CSS transition 或 `motionDuration()`（JS）判断，不要写死时长绕过降级。

## 1. 滚动交接模块现状（改这里之前先读）

`lib/island-handoff.ts` 仍保留「滚动驱动交接」实现，但**岛本体已不再消费**（常驻模型）：

- 仍接线：`PageHeader` / `DetailHeader` 在滚动回调里 `headerIsland.setHandoff(token, p, source)` →
  `publishHandoff({ p, source })`；`handoffRecede` action 逐帧写页头 band 的 opacity；`data-island-*`
  测试钩子（`data-island-visible` / `data-island-band-hidden` / `data-island-handoff`）仍由这些帧驱动。
  桌面端标准页头已 `lg:hidden`，因此这些帧当前不产生可见效果；移动/平板 `islandEnabled=false` 不触发。
- **未接线（待清理 backlog，不要在新代码里使用）**：`islandHandoff` 飞行 action、`islandFlight` /
  `handoffCss` 几何、`.island-flight-enter/-exit`、`.island-bloom`、`.island-land`、
  `--cst-island-content`；单测 `lib/__tests__/island-handoff.test.ts` 仍锁定这些纯函数。
- `lib/island-morph.ts`：`islandRectOf` / `islandTextRectOf` 仍被页头用于锚点矩形快照，供上述帧使用。
- 状态：`stores/header-island.svelte.ts` 的 `visible` / `settled` 仍由 `setHandoff` 派生（页面侧
  inert/aria 与单测使用），`progress` 与 token 无关（首页/搜索同样消费）。
- **禁止**在岛渲染路径上重新引入滚动显隐/飞行动画：岛常驻是已确认的交互基线。

## 2. 复测方法（动效改动必须做）

环境（worktree 独立调试，端口与主环境隔离）：见 skill `hotspot-worktree-debug`；本机可用 8889/5433/8083…
（`NGINX_PORT=8889` + compose override + `COMPOSE_PROJECT_NAME=<项目名>` 跑 `./scripts/seed-dev.sh`）。

展开/收起帧采样（页面里 eval，验收「宽度连续 / 圆角不钳制 / 头部不塌陷」）：

```js
window.__s = [];
(function () {
  const el = document.querySelector('[data-desktop-dynamic-island]');
  const t0 = performance.now();
  function f() {
    const b = el.getBoundingClientRect();
    const head = el.firstElementChild?.getBoundingClientRect();
    window.__s.push([
      Math.round(performance.now() - t0),
      Math.round(b.width),
      Math.round(b.height),
      head ? Math.round(head.height) : -1,
      getComputedStyle(el).borderTopLeftRadius
    ]);
    if (performance.now() - t0 < 420) requestAnimationFrame(f);
  }
  requestAnimationFrame(f);
})();
document.querySelector('[data-desktop-dynamic-island] button[aria-label="展开控制面板"]').click();
// 稍后读取 window.__s：宽度应连续增长（290→…→460）、圆角只出现 20~24px、高度从 36 起单调增长
// （首帧不得出现 2px 边框细线），头部行 34→44px。
```

hover 验收：headless 下 `:hover` 常不可靠（`matches(':hover')` 可能恒 false）；用真实鼠标/人工复核，
或用选择器静态核验（标题按钮含 `hover:` 类即会退出全局按钮兜底填充）。

截图：元素级 `agent-browser screenshot '<selector>' <绝对路径>`（fixed 覆盖层用整屏截图再裁）。
截图裁剪注意：`sips --cropOffset offsetY offsetX`（**先 Y 后 X**，别写反）。

必跑命令（宿主 frontend 目录）：

```bash
./node_modules/.bin/svelte-check --tsconfig ./tsconfig.json --tsgo
./node_modules/.bin/eslint .
bun run check:theme && node scripts/check-component-consistency.mjs
./node_modules/.bin/vitest run
# build 前先停 frontend 容器（共享 .svelte-kit），跑完再 start：
docker compose -f docker-compose.yml -f docker-compose.dev.yml stop frontend && bun run build
# 浏览器巡检（需 dev 环境）：
cd frontend && AB=~/.local/bin/agent-browser bash scripts/browser-qa.sh --base http://localhost:8889 --mobile
```

合并后主环境 frontend 容器要重启（Vite 在容器内，重启才会稳定加载新代码）。

## 3. 现行已落地动效（改前先看是否已有现成实现）

- 筛选/排序切换与空态入场：`lib/motion.ts` 的 `contentFade`（110ms 淡入）、`.empty-state-enter`（150ms）。
- 详情页画廊切图：`MomentPostBody.svelte` 双层 `<img>` 交叉淡入（140ms）+ 同源模糊背景过渡。
- SheetShell 拖拽关闭：位移/速度综合判定，顺势滑出（`--cst-motion-rapid`）或 spring 回弹。
- 移动/平板底部悬浮岛导航：`rounded-[28px]` 毛玻璃胶囊（`MobilePortrait` / `TabletLayout` / `MobileLandscape`）。
- 主题切换：View Transition 260ms `--cst-ease-standard`，首屏无过渡、reduced 立即。
- 页面进场 / 内容淡入 / 状态 pop / 评论入场：统一入口 `lib/motion.ts` 与 utilities.css 的 `cst-*` 类。

## 4. 未完成待做（按优先级）

1. **滚动交接遗留清理（backlog）**：`islandHandoff` action、`.island-flight-*` / `.island-bloom` /
   `.island-land` / `--cst-island-content`、`islandFlight`/`handoffCss` 已无调用方；清理时同步
   `island-handoff.test.ts` 与 §22 说明。清理前不要在原基础上新增交接动画。
2. **动效改动后的质量门禁**：按 §2 跑 `svelte-check / vitest / build / browser-qa`；
   UI 改动必须截图对比（改前/改后、明暗双主题、受影响断面），不能只看 diff。
3. **岛 hover 材质真机取证**：headless 下 `:hover` 不可靠，需真实鼠标/人工复核。
4. **大量数据/长列表下的岛内容同步稳定性**：meta/筛选状态随页面状态实时更新；若出现频繁重渲染，
   检查注册方 `update()` 是否传入了不必要的引用。

## 5. 踩过的坑，接手时别再踩

- **`tools.read`（DSH）会静默截断 >~39KB 的文件**：read→write 整文件会把 `utilities.css` /
  `design-system.md` 截断成半个文件（触发 Tailwind「Missing closing }」白屏或直接丢章节）。
  大文件不要 read→整写，用 node/python 按 marker 拼接，或写后核对 `wc -l` / `git diff --stat`。
- **CSS 注释里别写会与 Tailwind 解析冲突的内容**：改 `utilities.css` 后若 HMR 报
  `Missing closing } at .page-header-scrolled`，先查文件是否被截断/括号是否配平。
- **sips `--cropOffset` 参数顺序是 Y X**，写反会把图裁到左边并带黑边。
- **agent-browser 元素截图对 fixed 覆盖层会裁错**：截图 demo/覆盖层时一次只显示一个 section 再截。
- **headless 下 `:hover` 不可靠**：`matches(':hover')` 可能恒 false，hover 验收用真实鼠标或人工复核，
  不要仅凭 headless 截图判定 hover 失效。
- **worktree 里跑 seed**：seed 脚本按默认 compose 项目名找 backend；用 `COMPOSE_PROJECT_NAME=<项目名>` 覆盖。
- **主环境生效**：合并到 main 后必须重启 frontend 容器（Vite dev 在容器内），否则 8888 可能仍跑旧模块。

## 6. 交接清单（下一会话开始可照做）

- [ ] 读 `docs/design-system.md` §22（岛契约）与本文件 §0/§1。
- [ ] `git log -3` 确认基线已包含「灵动岛常驻 + 连续 morph」提交；`git status` 干净。
- [ ] 开工前建 worktree（`create_worktree`，分支名如 `anim-next`），按 skill `hotspot-worktree-debug` 起 8889 环境。
- [ ] 改岛形态/动效前先跑 §2 帧采样拿「改前」数据，改完按同口径复测 + 明暗双主题截图。
- [ ] 每项完成后按 §2 跑门禁 + 截图对比，再 `ff-only` 合并、重启主环境容器、清理 worktree。
