# HotSpot UI Design Kit（跨端设计稿）

Web（SvelteKit + Tailwind）与 HarmonyOS（ArkTS）共用的 UI 设计系统与**可编辑设计稿**包。

目标：把线上真实运行的 UI **转换成 Figma 可编辑设计稿**，而不是靠反复截图对齐；后续改代码后重跑脚本即可重新生成，保持设计稿与实现同源。

## 这套包有什么

| 层 | 产物 | 说明 |
|---|---|---|
| Token 源 | `tokens/runtime.json`、`tokens/global.json`、`tokens/themes/*.json` | 从代码 CSS Token + 运行时解析出的 6 个主题模式（3 主题 × 明暗）全部颜色值；可直接导入 Tokens Studio |
| 设计规格 | `.workwork/design-kit/spec/spec.json`（生成物，不入库） | 从运行中的前端抓取：35 个页面 × 3 端（移动/平板/桌面）的盒模型、文字、图片、图标、圆角、描边、阴影、渐变，并已映射到语义 token |
| Figma 插件 | `figma-plugin/`（manifest + code.js） | 本地开发插件：在 Figma 中一键生成「设计规范 + 3 端全部页面」，颜色绑定 Figma Variables，切换 6 种主题模式即可看到主题/明暗 |
| 预览 | `node scripts/design-kit/render-preview.mjs` | 不需要 Figma 也能在浏览器查看规格渲染结果（自查用，输出到 `.workwork/`） |
| Stitch 提示词 | `stitch-prompt.md` | 给 Stitch 的项目描述与约束（用于快速探索稿） |

## 生成流程（改完 UI 后重跑）

前置：本地 dev compose 已启动（`http://localhost:8888`），`agent-browser` 可用（见根 AGENTS.md 的浏览器巡检前置）。

```bash
# 1. 抓取 6 个主题模式的 token 运行时值
node scripts/design-kit/capture-tokens.mjs

# 2. 抓取全部页面（35 页 × 3 端；默认体感约 5-8 分钟）
node scripts/design-kit/capture-screens.mjs            # 全量
node scripts/design-kit/capture-screens.mjs --device mobile   # 只抓某端
node scripts/design-kit/capture-screens.mjs --only home,settings --force  # 只抓某几页

# 3. 生成设计规格（压缩 + 去重 + token 映射；输出到 .workwork，不入库）
node scripts/design-kit/build-design-spec.mjs

# 4. 打包 Figma 插件（docs/design-kit/figma-plugin/code.js）
node scripts/design-kit/build-figma-plugin.mjs

# 5. 无 Figma 环境下的逻辑冒烟测试（可选）
node scripts/design-kit/verify-figma-plugin.mjs

# 6. 浏览器预览（可选，输出 .workwork/design-kit/preview/）
node scripts/design-kit/render-preview.mjs --mode vibrant-light
```

> 抓取脚本会以种子管理员账号（`admin@hotspot.local`）真实登录，并从运行中的页面读取计算样式；不含任何截图，页面结构由 DOM 盒模型生成。

## 导入 Figma（一次操作，之后重跑插件即可刷新）

1. 打开 **Figma 桌面端**（Dev Plugin 需要在桌面 App 中导入）；
2. 菜单 `Plugins → Development → Import plugin from manifest…`，选择本包 `figma-plugin/manifest.json`；
3. 运行插件 `HotSpot Design Kit`，等待完成（首次约 30–90 秒，会生成 4 个页面）：
   - `01 · 设计规范 Design System`：6 个主题模式面板、语义色板、字号/圆角、组件库（按钮/输入/徽章/Tabs/列表行/卡片/底部导航/弹层/Toast/空态）；
   - `02 · 移动端 Mobile`（390×844）、`03 · 平板端 Tablet`（834×1112）、`04 · 桌面端 Desktop`（1440×900），每页 35 个页面 frame；
4. 主题/明暗切换：右侧 **Variables** 面板 → `HotSpot Tokens` 集合 → 切换 mode（`vibrant-light` / `vibrant-dark` / `forest-light` / `forest-dark` / `inkwash-light` / `inkwash-dark`），全文件即时换肤；也可对单个 frame 设置显式 mode 覆盖。
5. 重新生成：改完代码重跑上面的脚本，再运行一次插件；插件会重建三个设备页（设计规范页也会刷新），Variables 保持同名复用。

### 设计稿里有什么

- **颜色**绑定到 Figma Variables（127 个颜色 token）；文字/图标颜色 99.5% 绑定 token，填充约 90% 绑定 token，其余为图片/媒体遮罩等主题无关色值；
- **文本**是真实 Figma 文本层（Inter / PingFang SC），可编辑；
- **图标**是由内联 SVG 转成的矢量层；
- **图片**使用抓取时生成的缩略图（不是截图拼贴）；
- 每一屏都是独立 frame，按设备排在对应页；层结构保留页面真实层级（绝对定位），便于逐个核对。

## 已知边界

- 只捕获**首屏视口**（滚动内容不展开）；长列表页/聊天页的后续内容不在设计稿里；
- 画布类内容（地图 canvas、视频）以占位块表示；
- 动效、hover/focus 状态不在静态帧里（组件库给出了常用状态样例）；
- 背景模糊（backdrop-filter）、复杂渐变在 Figma 中以近似实现；
- 抓取基于当前种子数据，文案与图片随 seed 数据变化；
- 插件生成的是绝对定位图层（未转 Auto Layout），适合作为设计基准与标注，不适合直接做响应式组件原型。

## 产物维护约定

- `tokens/`、`figma-plugin/` 是**生成物**但提交仓库（插件是可直接导入 Figma 的交付物）；中间规格 `spec/`、预览与截图在 `.workwork/`，不入库。不要手工编辑这些生成物，改 UI/Token 后按上面的流程重新生成；
- token 事实源：Web `frontend/src/styles/theme/`（themes → semantic → shared/registry）与鸿蒙 `harmonyapp/entry/src/main/ets/constants/DesignConstants.ets`；
- 跨端一致性校验：`cd frontend && bun run check:theme`、`python3 harmonyapp/scripts/check_status_parity.py`；
- 页面截图（可选，给 Stitch 当参考图）由 `capture-pages.sh` 输出到 `.workwork/design-kit/pages`，不入库。

## 与 html.to.design / Stitch 的关系

Figma 没有原生 HTML 导入；社区插件 `html.to.design`（免费额度 10 次/月，Pro 约 $12/月）与 `Multi-Screen HTML Importer` 可以把手写 HTML 转成 Figma 图层，但不支持组件、且每次导入需要人工操作、无法绑定多主题变量。本包选择自建本地插件：批量、免费、可重复生成、颜色绑定 Variables 支持 6 种主题模式。若只是想快速看一版探索稿，也可以用 `stitch-prompt.md` 在 Stitch 生成。

## 跨端对齐

Web 与 HarmonyOS 共用同一套语义 token（颜色/字号/圆角/间距/动效）。改任一端 token 必须同步另一端并通过上述校验，再重新生成本包，这样设计稿、Web、鸿蒙三边保持一致。
