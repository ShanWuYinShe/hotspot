# 前端动效交接（灵动岛与全站衔接）

> 本文件用于跨会话接手：记录**当前已生效的基线**、**交接机制速查**、**未完成项（可直接照做）**、**复测命令**。
> 动效契约的权威描述在 `docs/design-system.md` §10 / §22；页面布局契约在 skill `hotspot-frontend-layout`。
> 本文件不是规范，不要把它当作设计依据——规范以 design-system 为准。

## 0. 当前基线（勿回退）

- **灵动岛形态**：h-9 胶囊，最右侧 18px 阅读进度圆环（accent 弧 + 极淡底轨，0% 空环 / 100% 整圆），
  每帧直接写 `stroke-dashoffset`，**无 CSS 过渡**；行动区存在时圆环在行动区外侧（最右）。
  唯一形态实现 `frontend/src/components/layout/page/IslandCapsule.svelte`。
- **标题 → 岛：滚动驱动交接（scroll-linked）**，不再有「触发后播一段时长动画」：
  - 页头 / 详情页 / 首页在滚动回调里算进度 `p(0→1)`，`publishHandoff({ p, source })`；
  - 岛侧 `use:islandHandoff` 逐帧写 `transform` / `opacity` / `--cst-island-content`；
  - 吸顶页头滚动窗口 120→260px（DetailHeader 160→280px，均乘 `platform.uiScale`）；`integrated` 页头 /
    首页按锚点矩形相对 52px 顶栏的位置每帧算；
  - 滚动停在哪交接就停在哪；反向滚动逐帧可逆（实测同一 scrollTop 下 transform 字符串完全一致）；
  - 路由切换：`register/release` 会把交接归零 + 岛无时长退场动画 → 直接卸载，不再有「飞回上一页标题」幽灵。
- **页头退场**：`handoffRecede` 逐帧写 inline opacity；`p<0.5` 保持清晰可读，后 35% 进度跨度内快速淡出
  （避免半透明页头与滚动的正文长时间叠字双曝光）。`.page-header-scroll` 的 transition 列表**不含 opacity**。
- **首页信息流岛**同样走滚动交接（原 IntersectionObserver + 时长动画已删除）。
- **reduced-motion**：过闸即切换（`p>0.5` 直接终态），不做连续位移。
- **文档已同步**：design-system §22、ui-components、frontend/AGENTS.md、layout skill。

提交：`e2a6a8f0`（上一轮全站动效治理）→ `e32bddef`（圆环 + 滚动驱动交接）→ `ab416d7a`（交接 action 默认启用修复）。

## 1. 交接机制速查（改这里之前先读）

- `frontend/src/lib/island-handoff.ts`
  - `publishHandoff({ p, source })` / `subscribeHandoff(fn)` / `currentHandoff()` / `resetHandoff()`：
    模块级单例，滚动回调 publish、岛与页头订阅；**只有当前页面在 publish**（页面卸载时归零）。
  - `scrollHandoffProgress(scrollTop, start, end)`：吸顶页头滚动窗口进度（窗口外 0/1、窗口内线性）。
  - `anchorHandoffProgress(top, height, barBottom, approach=24, tail=10)`：非吸顶锚点按相对顶栏位置算进度。
  - `handoffCss(target, source, p, reduced)`：纯函数（单测覆盖）。几何 smoothstep 插值；`p=1` 返回空 transform
    把 hover 的 1px 上移交还 CSS；内容层 `(eased-0.55)/0.4` 显字；`p/0.35` 壳显影。
  - `islandHandoff(node, { enabled? })`：胶囊 action，**默认启用**（显式 `enabled:false` 才跳过；
    这个默认值有 action 级回归单测，见 `lib/__tests__/island-handoff.test.ts`）。
  - `handoffRecede(node, { enabled, mode: 'band' | 'content' })`：页头退场 action。
- 状态：`frontend/src/stores/header-island.svelte.ts` —— `setHandoff(token, p, source)` 驱动
  `visible`（p>0.001 挂载）与 `settled`（p>0.55，页头 inert/aria-hidden）；`progress` 与 token 无关（首页岛同样消费）。
  页面注册/释放都会把交接归零。
- 触发方：`PageHeader.svelte`（sticky 窗口 / integrated 锚点模式 + sticky 容器探测）、
  `DetailHeader.svelte`（160→280 窗口，锚点 y 收紧进顶栏带）、`routes/(app)/+page.svelte` 的 `tabsHandoff`
  （首页分段器 action）。
- 几何工具：`frontend/src/lib/island-morph.ts` 只保留 `islandRectOf` / `islandTextRectOf`（Range 文字矩形）/
  `islandFlight`（中心对齐 + 长宽比缩放，极端比例才 clamp）。**不要再往这里加时长动画**。
- 测试钩子：页头 `data-island-visible` / `data-island-band-hidden` / `data-island-handoff`；
  岛根 `data-desktop-dynamic-island`；圆环 `data-island-progress-ring` / `data-island-progress`。

## 2. 复测方法（动效改动必须做）

环境（worktree 独立调试，端口与主环境隔离）：见 skill `hotspot-worktree-debug`；本机可用 8889/5433/8083…
（`NGINX_PORT=8889` + compose override + `COMPOSE_PROJECT_NAME=<项目名>` 跑 `./scripts/seed-dev.sh`）。

帧级采样（在页面里 eval，验证「跟手 / 可逆 / 停手不动」）：

```js
const main = document.getElementById('main-content');
const read = () => {
  const isl = document.querySelector('[data-desktop-dynamic-island]');
  return isl ? { op: getComputedStyle(isl).opacity, tr: isl.style.transform,
                 ct: isl.style.getPropertyValue('--cst-island-content') } : null;
};
// 下行：逐个 scrollTop 采样 → 停手 600ms 再采（必须与停手前一致）→ 上行：同一 scrollTop 的 transform 必须一致
```

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
截图裁剪注意：`sips --cropOffset offsetY offsetX`（**先 Y 后 X**，别写反）；元素级截图对 fixed 覆盖层不可靠，
裁景时优先整屏截图再按 Y 裁剪。

## 3. 未完成待做（按优先级；每条都可直接开工）

### P0 — 流畅度硬伤（本轮审计发现、尚未修）

1. **筛选 / 排序切换硬切**（/nearby/people、/nearby/moments、/nearby/groups 等）
   - 现状证据：点「摄影」后 12 张卡 → 空态在 ≤50ms 内一帧替换，`getAnimations()` 为空，无过渡、无骨架；
     排序切换同类。筛选状态会写进 `islandMeta`（已生效），但内容区本身是硬切。
   - 建议：把 `lib/motion.ts` 的 `contentFade` 扩展为可被「筛选/排序 nonce」重放（如
     `use:contentFade={{ trigger }}`），或对列表容器做 110–160ms 交叉淡入 + 保留旧数据降透明度；
     空态出现也走同一动作。**不要**给每个卡片单独挂动画（N 卡 × 过渡 = 掉帧）。
   - 验收：切换后 1 帧内出现 1 个 110–160ms 的 opacity/transform 动画、无 CLS、无长任务；reduced 直接切。

2. **详情页画廊切图硬切**（`components/moments/MomentPostBody.svelte` 画廊区）
   - 现状证据：点第 2 张缩略图，页码 1/3→2/3，但新图 opacity 全程 1、无动画；未缓存时旧图瞬间卸载 → 灰底闪一下；
     背景 `data-gallery-backdrop` 也是瞬变。
   - 建议：双层 `<img>` 交叉淡入（旧图保留到新图 `onload`/`decode()` 后 120–150ms；backdrop 同源交叉），
     失败/超时回退硬切；`{#key}` 结构需改为双图层或保留旧图节点。
   - 验收：缓存与未缓存两种都无闪白/灰底；页码与图同步；Google 图片加载慢时不阻塞交互（可点下一张）。

3. **SheetShell 拖拽关闭「先弹回再下滑」**（底部弹层拖拽关闭）
   - 现状：拖拽释放时先回弹到原位再播放关闭动画，方向不连贯；边缘返回手势（MobilePortrait）被 150ms
     过渡拖住，不跟手。
   - 建议：拖拽释放按速度/位移决定「关闭 or 回位」，关闭时从当前拖拽位移直接向下滑出（不先复位）；
     两个手势共用同一 spring/velocity 判定。验收：慢拖/快甩/中途反向三类手势都跟手、无方向跳变。

### P1 — 审美与一致性打磨

4. **岛 leading 视觉锚点**（普通页面岛只有两行文字，左侧偏空）
   - 已给 demo 候选：A 8px accent 圆点；B 28px accent-soft 圆底 + 语义图标（与首页岛同语言）；C 裸图标。
     用户尚未拍板；若做，给 `PageHeader` 增加可选 `islandIcon`（地图/齿轮/铃铛等），不要每页自造。
5. **空态 / 筛选结果入场**：EmptyState（spotlight 静态）出现时无入场；建议 150ms 淡入 + 2px 上移，
   与 P0-1 共用同一 action（reduced 直接终态）。
6. **主题切换过渡收敛**：换主题瞬时触发 ~174 条颜色过渡（500ms）。本机无 longtask，但低端机有风险；
   建议收敛到 300ms 或只对表面层过渡（背景/文字），避免大面积 transition。
7. **岛落地微动效（可选）**：抵达时 scale 1.006→1（80ms）+ accent 细边亮度回落一档；不做 overshoot。
   用户未确认要不要，先别做。
8. **岛 meta 文本跳变**：搜索词/计数更新时岛内第二行直接换字（无 crossfade），频率低但可见；
   可加 120ms 淡入/淡出（注意不要和交接的 `--cst-island-content` 叠两层）。

### P2 — 验证/环境类

9. **岛 hover 材质真机取证**：headless 下 `:hover` 不可靠（`matches(':hover')` 恒 false），
   本轮未做视觉取证；用真机/手动 hover 确认「实底 + accent 细边 + 抬升 + 1px 上移、宽度不变」。
10. **动效改动后的质量门禁**：按 §2 跑 `svelte-check / vitest / build / browser-qa`；
    UI 改动必须截图对比（改前/改后、明暗双主题、受影响断面），不能只看 diff。
11. **大量数据/长列表下的交接稳定性**：本轮在种子数据下采样正常；若页面首屏骨架高不同导致锚点绝对位置
    在数据到达后变化，需要复测 integrated 页头（`measureAnchor` 已有 scrollTop<1 时重测逻辑，重点看首屏）。
12. **95–100% 进度环满圈与 hover 交互**：圆环当前 `aria-hidden` 纯展示、不可点；若后续要「点击回顶」等语义，
    先补 a11y（button + aria-label）再改视觉，保持「唯一读数只一处表达」。

## 4. 本轮（动效交接）踩过的坑，接手时别再踩

- **`tools.read`（DSH）会静默截断 >~39KB 的文件**：本轮 read→write 曾把 `utilities.css` 截断成半个文件，
  触发 Tailwind「Missing closing }」导致 dev server 白屏。**大文件不要 read→整写**，用 edit/python 定点替换，
  或写后核对 `wc -l` / `git diff --stat`。
- **CSS 注释里别写会与 Tailwind 解析冲突的内容**：改 `utilities.css` 后若 HMR 报
  `Missing closing } at .page-header-scrolled`，先查文件是否被截断/括号是否配平。
- **IslandCapsule 的 `use:islandHandoff` 必须默认启用**：简化时漏传 `{{ enabled }}` 导致岛「只在落位后突然出现」；
  已加 action 级回归测试（不传参数也要消费交接帧）。改这个 action 的默认值前先看 `island-handoff.test.ts`。
- **sips `--cropOffset` 参数顺序是 Y X**，写反会把图裁到左边并带黑边（本轮因此误判过岛的位置）。
- **agent-browser 元素截图对 fixed 覆盖层会裁错**：截图 demo/覆盖层时一次只显示一个 section 再截。
- **worktree 里跑 seed**：seed 脚本按默认 compose 项目名找 backend；用 `COMPOSE_PROJECT_NAME=<项目名>` 覆盖。
- **主环境生效**：合并到 main 后必须重启 frontend 容器（Vite dev 在容器内），否则 8888 可能仍跑旧模块。

## 5. 交接清单（下一会话开始可照做）

- [ ] 读 `docs/design-system.md` §22（岛上/交接契约）与本文件 §1/§3。
- [ ] `git log -3` 确认基线在 `ab416d7a` 之后；`git status` 干净。
- [ ] 开工前建 worktree（`create_worktree`，分支名如 `anim-next`），按 skill `hotspot-worktree-debug` 起 8889 环境。
- [ ] 先做 P0-1（筛选/排序过渡）或 P0-2（画廊交叉淡入），两者都改 `lib/motion.ts` 的复用面，建议一起评估。
- [ ] 每项完成后按 §2 跑门禁 + 帧采样 + 截图对比，再 `ff-only` 合并、重启主环境容器、清理 worktree。
