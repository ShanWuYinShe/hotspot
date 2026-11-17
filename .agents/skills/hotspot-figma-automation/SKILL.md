---
name: hotspot-figma-automation
description: Figma 设计稿无感云端审查与自动化同步工作流：MCP session 维护、use_figma 执行脚本、Google Fonts 预加载防崩溃、数学级中心线重构公式、移动端去系统化与无菌化准则、图片叠字卡片的主题玻璃条（浅色/深色自适应）、侧边栏导航项对齐、卡片内文案溢出与方形行带治理、设计规范页分区块整理与生成器字体/排版修正、全站配色 token 化（叠字卡片玻璃条/管理后台/图标随 6 个颜色模式自适应）、细节巡检（中文缺字/文字重叠/文字溢出/顶部与底部安全区）、平板页面一致性对齐、三端审美一致性（字号/圆角/文案/字体基线）、可交互原型演示（变体组件 + 原型反应）、截图视觉闭环验证、同步后自动巡检门禁。触发：自动化审查/修改 Figma 设计稿；同步后跑巡检、排查同行标签基线或页头居中偏差；状态栏/时钟清理；图层错位重构；use_figma 字体报错；同步设计稿至 Figma；Figma MCP 调试。
---

# hotspot-figma-automation

Figma 设计稿云端自动化修改、清洗与同步规程。基于 Figma MCP 远程执行机制，无需在桌面端手动启动插件，通过脚本驱动服务端沙盒对画布节点进行批量分析、数学对齐、数据无菌化与闭环视觉验证。

## 何时用

- 需要自动批量检查或修正 Figma 画布（图层对齐、文案规范、样式统一）。
- 需要全量清除移动端画板上的系统状态栏（`System / Status Bar`）、时钟（`9:41`）与状态图标。
- 自动化执行 `use_figma` 脚本遇到未加载字体报错或 Session 失效。
- 批量重构表格、列表或卡片元素并确保多列中心线像素级对齐。
- 改动完成后通过 MCP 截图进行视觉闭环验收。
- 设计稿同步完成后跑巡检门禁，确认同行标签基线、页头标题居中无偏差。
- 需要在 Figma 侧打里程碑标记（里程碑名 + 本次改动摘要）。

## 规则 / 做法

### 1. 云端静默执行机制
- Figma MCP 服务端沙盒直接连接 Figma 文档模型（WebSocket 双向同步）。
- Agent 提交的 Plugin API 脚本直接在云端沙盒运行并实时广播，无需用户在本地 Figma 客户端开启插件或保持桌面应用处于激活态。

### 2. 字体安全加载与写属性铁律
- **禁止向未加载字体的 TEXT 节点写入任何属性**：Figma 服务端沙盒缺失本地字体（如 `PingFang SC`）。对含未加载字体的节点执行任何写操作（包含 `characters`、`fontSize`、`textAlignVertical`、`resize` 等）均会触发运行时致命异常：`Cannot write to node with unloaded font`。
- **全样式预加载范式**：不仅加载 `Regular`，必须同时加载待写入的 `Medium` 和 `Bold`：
  ```javascript
  await figma.loadFontAsync({ family: "Noto Sans SC", style: "Regular" });
  await figma.loadFontAsync({ family: "Noto Sans SC", style: "Medium" });
  await figma.loadFontAsync({ family: "Noto Sans SC", style: "Bold" });
  ```
- 修改属性前，先强制重定向至已加载字体：`text.fontName = { family: "Noto Sans SC", style: style };`，再修改 `textAlignVertical` 或重设文本。

### 3. 单行胶囊/徽章/药丸按钮原生垂直居中
- **禁止盲调 Y 轴偏移**：严禁采用 `text.y = bg.y + 3` 这种经验主义盲调。由于中文字体内置顶间距差异，手动偏移会导致高亮态与未选中态、不同字号间高度与中心线错位。
- **原生垂直居中标准范式**：
  ```javascript
  text.fontName = { family: "Noto Sans SC", style: style };
  text.textAlignVertical = 'CENTER';
  text.y = bg.y;
  text.resize(text.width, bg.height);
  ```
  让文本节点的外框高度直接等于背景矩形高度（`text.height = bg.height`），`y` 坐标与背景顶对齐，借由 Figma 原生排版引擎以亚像素精度自动完成数学垂直居中。

### 4. 复合胶囊（图标 + 文字）安全布局方程
- **严禁独立写死绝对坐标**：图标与文字若分别写死 X 坐标，极易在文本调整或字宽变化时产生物理重叠（如筛选按钮文字遮盖图标、公告图钉图标与「置顶」重合产生重影）。
- **复合胶囊定位公式**：
  $$W_{total} = W_{icon} + \text{gap} + W_{text}$$
  $$X_{start} = X_{bg} + \text{round}\left(\frac{W_{bg} - W_{total}}{2}\right)$$
  $$X_{icon} = X_{start},\quad Y_{icon} = Y_{bg} + \text{round}\left(\frac{H_{bg} - H_{icon}}{2}\right)$$
  $$X_{text} = X_{icon} + W_{icon} + \text{gap},\quad Y_{text} = Y_{bg}$$
  保证容器左右留白完全对称，图标与文字间距清晰（$\text{gap} \ge 4\text{px}$），文字全高居中。

### 5. 指标 / 数据卡片（Stats / Metrics）水平中轴绝对锁定
- **严禁数值与标签独立绝对定位**：数值通常为大号加粗字体（20~24px），标签为小号辅助文本（12~13px）。两者各赋 X 坐标会导致数值向右偏离 15~25px，破坏视觉 1:1 对应关系。
- **中轴对齐公式**：
  $$X_{center} = X_{lbl} + \frac{W_{lbl}}{2}$$
  $$X_{num} = \text{round}\left(X_{center} - \frac{W_{num}}{2}\right)$$
  或为两者设置等宽列框（`width = colWidth`），均开启 `textAlignHorizontal = 'CENTER'`，确保垂直中轴绝对重合。

### 6. 桌面端侧边栏 Logo 区域三段式防撞规约
- 桌面端侧边栏头部必须严格遵循三段式间距，禁止折叠按钮图标侵入品牌 Logo：
  - 品牌图标：$X = 33$（宽 24，右边界 57）
  - 品牌文字「HotSpot」：$X = 65$
  - 折叠展开图标：$X = 140$

### 7. 去系统化与无菌化准则
- **全屏画板禁止系统状态栏**：移动端与平板端所有功能画板必须彻底剔除 iOS/Android 系统状态栏（如 `System / Status Bar (iOS)`）、系统时钟（`9:41`、`12:00`）及信号/WiFi/电池图标。状态栏属于系统层级，不得侵入业务画布。
- **数据无菌化**：严禁在设计稿中遗留随机哈希、开发种子脏数据（如 `游客·fck3`）或无意义占位符。列表文案必须使用贴合产品业务场景的正式文本。

### 8. 闭环视觉验证铁律
- **禁止盲改即止**：任何节点变更执行完毕后，必须调用 `get_screenshot` 对受影响画板执行高清快照捕获。
- 下载快照至本地后亲眼核验，确认字距、行高、中心线对齐与边缘收口均达标。

### 9. 卡片内严禁绘制多余线框矩形（消灭 Box-in-a-Box）
- **严禁卡片内部嵌套描边线框**：白色主卡片已有圆角背景、轻描边或阴影。卡片内部的指标数字（如“256 用户”、“12 人 11 瞬间 5 在线”）和操作按钮严禁再被一层 1px stroke 的空心矩形圈起。空心线框会把卡片割裂成粗糙的线框表格，破坏现代扁平/卡片化视觉。
- **正解做法**：指标数值与按钮直接以透明背景浮动在主卡片内，依靠外边距（上下各 16~24px）保持充足留白与呼吸感。

### 10. 选项卡/胶囊分段指示器数学中轴与对称留白定理
- **严禁不同 Tab 状态错位偏移**：给选中态胶囊背景和未选中标签分别使用随意的 Y 坐标和不同高度，会导致选中文本下沉/偏心且 baseline 错位。
- **正解做法**：整行 Tab 文字必须共享同一 Y 坐标与行基线（`textAlignVertical = 'CENTER'`，文字高度统一为 13~14px）。选中态胶囊背景高度统一为 26~28px，其 Y 坐标严格遵循居中方程：
  $$Y_{pill} = Y_{text} - \frac{H_{pill} - H_{text}}{2}$$
  保证文字上下内边距光学绝对对称（上下各 6~7.5px），且与未选中文本处于完全同一水平视线。
- **整行 Tab 的 `textAlignVertical` 必须统一**（要么全 `CENTER`，要么全 `TOP` 后用字形定位）：同一行里「选中项 CENTER、未选中项 TOP」是最常见的 3~7px 基线漂移来源。
- **禁止用「文本框 y 相同」判断是否对齐**：文本框 y 相同但 `textAlignVertical` 不同，字形仍会相差半个行高；判定对齐必须看渲染后的字形中心（`absoluteRenderBounds`），巡检脚本即按此口径检查。

### 11. 复合状态徽章（微图标 + 文字）安全间距防粘连
- **指示点严禁粘连或遮挡文字**：如「● 实时雷达」徽章，绿点等状态指示器与右侧文字之间必须保持至少 4~6px 的清晰留白，严禁出现指示点覆盖文字边缘笔画的重叠事故。
- **文字必须禁止意外折行**：文本必须开启 `textAutoResize = 'WIDTH_AND_HEIGHT'`，确保四字文案不因宽度过窄而发生折行，背景外框宽度由内边距自动撑开：
  $$W_{bg} = \text{padLeft} + W_{dot} + \text{gap} + W_{text} + \text{padRight}$$

### 12. 按钮内微型复合图标尺寸与描边权重控制
- **严禁高密度矢量微缩失真**：在 32px 高度的小按钮内（如列表「关注」按钮），严禁将复杂复合矢量（如 UserPlus）强行塞入 10x10px 且保留 1.75px 粗描边，否则会导致头像与加号重叠挤压成模糊色块。
- **正解做法**：按钮微图标标准规格为 12x12px 或 14x14px，描边粗细控制在 1.2px，加号与人像主体保持至少 2px 物理间距，确保矢量细节锐利分明。

### 13. 地图选点现代水滴锚点设计范式
- **彻底废除落后的粗劣棒棒糖+十字准星旧式图标**。
- **标准现代地图 Pin 三层解构**：
  1. **地面光晕与投影**：底部椭圆接触阴影（#0F172A，透明度 0.25）+ 渐变脉冲定位环（#2563EB，透明度 0.12，带 0.3 透明度描边）。
  2. **悬浮水滴本体**：矢量水滴造型（SVG 平滑贝塞尔曲线），填充品牌蓝渐变（#3B82F6 -> #1D4ED8），外加轻柔投影（Y=4, R=8, Opacity 0.25）。
  3. **核心指示点**：水滴头部内嵌纯白圆环（11x11px）与中心蓝芯点（4.5x4.5px），尖端精准指向地图中心点。

### 14. 指标数值严禁混入状态圆点脏字符
- **数值与状态严格解耦**：指标数值纯展示数字或量词，状态在线标识等应由卡片头部标签或徽章承载，严禁在数值中硬编码「2• 在线」这类带有孤立圆点的混杂文本。

### 15. 同步后自动巡检门禁（同行基线 / 页头居中 / 叠字卡片 / 侧边栏）

- **同步完成后必须跑巡检脚本**（命令见「命令 / 脚本通道 4」）；返回 `summary.pass === true` 才视为同步通过；`violations` 非空时按 `hint` 逐条修复并重新巡检，直至归零。
- 巡检为只读脚本，不修改任何节点；输出 `{ summary, violations, warnings }`，`summary.byRule` 给出分类计数。
- 规则与阈值：
  - `tab-row-baseline`：同一胶囊标签行（Tab / Chip / 筛选，含选中态与数字）内各标签字形中心偏差 > 2px → 违规，按 §10 修复。
  - `tab-row-va-mixed`：同一行标签 `textAlignVertical` 混用 → 警告。当前可能看似对齐，但字体/字号变化后会再次漂移，应在下次改动时统一。
  - `header-center`：移动/平板页头标题与返回按钮中心偏差 > 2px → 违规。单行标题比字形中心；标题+副标题两行块比整块中心；桌面端页头为标题胶囊结构，不在此规则内，仍需截图核对。
  - `card-chip-edge` / `card-text-edge`：卡片右上角时间胶囊（或纯文字）右缘贴边/溢出（右内边距 < 6px）→ 违规，按 §16 修复。
  - `card-chip-inset-asym`：时间胶囊右内边距与上内边距相差 > 3px → 警告，建议统一为相同 inset。
  - `overlay-missing-glass`：卡片内图片覆盖 > 80% 且存在胶囊外的白色叠字，却没有玻璃条 → 违规，按 §17 修复。
  - `overlay-glass-theme`：玻璃条颜色与卡片所在主题不匹配（浅色主题应为浅条、深色主题应为深条）→ 违规。
  - `overlay-text-theme`：叠字颜色与主题不匹配（浅色主题深字、深色主题浅字）→ 违规。
  - `overlay-chip-text`：黑色胶囊（时间、已关注等）内文字不是反白 → 违规。
  - `nav-item-indent`：同一侧边栏内导航文字或图标 x 偏离基准 > 2px（典型为激活项缩进）→ 违规，按 §18 修复。
  - `card-text-overflow`：圆角卡片内左对齐文案渲染超出卡片右缘，或压住同行右侧小图标（10-16px）→ 违规，按 §19 修复。
  - `card-inner-band`：圆角卡片内出现方形（圆角 0）行带矩形 → 警告，按 §19 处理。

### 16. 卡片右上角时间标签必须内缩（禁止贴边/出血）

- **禁止把时间胶囊的右缘对齐到卡片右边界**：胶囊右缘超出卡片（负内边距）会被卡片裁切或压住圆角，视觉上像「溢出」，是生成端最常见的系统性错位。
- **内缩方程**（inset 取卡片顶部内边距，通常 10px）：
  $$X_{chip} = W_{card} - \text{inset} - W_{chip}$$
  $$X_{text} = X_{chip} + \frac{W_{chip} - W_{text}}{2}$$
  文字与胶囊同步平移，保持胶囊内水平居中；右内边距必须与上内边距对称（差值 ≤ 3px）。
- 卡片右上角若为纯文字（无胶囊底），同样按 inset 内缩，右内边距不得小于 6px。

### 17. 图片叠字卡片必须用「主题玻璃条」

- **适用范围**：卡片内图片覆盖率 > 80%、且文案压在图片上的所有卡片（信息流大卡、附近瞬间/枢纽图卡、群详情封面、搜索热门动态等）。
- **玻璃条几何**：按叠字包围盒生成，条宽 = 卡片宽，底/顶边贴齐卡片边缘；文字块上下各留 12px 内边距；卡片 `clipsContent` 负责裁圆角。
- **配色随主题（浅/深两档）**：

  | 主题 | 玻璃条 | 正文 / 标题 | 计数与操作 |
  |---|---|---|---|
  | 浅色 | 白 62% + `BACKGROUND_BLUR` 18 | #16233A | #52647F |
  | 深色 | #0B111C 66% + `BACKGROUND_BLUR` 18 | #E8F1E9 | #A8BDAE |

- **主题判定**：读取节点 `resolvedVariableModes`，向上取最近一层已解析的变量模式，按模式名含 `light` / `dark` 区分。不要用 `fills` 里存的颜色判断——绑定变量的节点取到的是默认色，不是当前主题的解析值。
- **叠字颜色必须显式写死**：不要复用主题 token 的绑定填充，绑定变量在其它主题模式下会解析成另一种颜色，产生「浅色主题里出现浅字」这类回归。
- **黑色胶囊文字**：时间、已关注等黑胶囊内文字固定反白（#FFFFFF），不参与主题换算。
- **图标**：叠字卡片内的操作图标统一线性化（清空填充，`strokes` 用该主题次级色，`strokeWeight` 1.5）。
- **装饰性渐变例外**：图片上只有装饰渐变、没有任何胶囊外叠字时，不算缺玻璃条。

### 18. 侧边栏导航项禁止缩进（激活态不得位移）

- 同一侧边栏内所有导航项共用同一 x 基准：**文字 x = 基准，图标 x = 基准 - 35**（桌面端 1440 宽稿实测文字基准 70、图标 35）。
- 激活项允许新增的只有「胶囊底 + 左侧 3px 指示条」，**禁止把图标/文字右移**（生成端常见缺陷：激活项图标 +42px、文字 +30px，视觉上像凹进去一块）。
- 修复时只平移激活项的图标与文字，保留胶囊底与指示条位置；修完跑巡检 `nav-item-indent` 复核到 0。

### 19. 卡片内文案不得溢出，禁止方形行带矩形

- **判定口径**：圆角卡片（r ≥ 16）内左对齐文案，真实渲染右端不得超过卡片右缘；同一行右侧若还有 10-16px 图标（箭头、更多等），文案右端 = 图标左缘 - 4px，超出即算溢出。只重合卡片本身不算，必须是同一父级下的行内图标才参与判定。
- **修法**：优先缩短文案并加省略号，不要靠缩字号糊过去；一般保留 12-14 个汉字（11px 字号、148-168px 可用宽度）。
- **字体不可加载时**：生成稿里的 `PingFang SC` 在沙盒无法加载，改不了 `characters`；可新建同字号/同色 `Noto Sans SC` 文案节点（截断后）替换，并把原节点 `visible = false`。桌面端多数文案本就是 Noto Sans SC，替换后视觉一致。
- **方形行带矩形**：圆角卡片内不应出现圆角 0、宽度超过卡片 55% 的实心矩形行带。要么直接删除（与相邻平铺列表卡片保持一致），要么统一改成圆角 12 + 左右内缩相同；修完跑巡检 `card-inner-band` 复核到 0。

### 20. 设计规范页（01）结构与整理

- **目标结构**：页面顶层只保留 7 个命名区块 FRAME（宽 1650、区块间距 96）：`00 概览 / Overview`、`01 主题 / Themes`、`02 语义色 / Semantic Colors`、`03 字号 / Typography`、`04 圆角 / Radius`、`05 动效 / Motion & Rhythm`、`06 组件 / Components`；不要再让 500+ 节点平铺在页面顶层。
- **区块内聚**：语义色每个 token 的「色块 + 名称 + 色值」内聚为 `token/<变量名>`；字号样本 `type/<nn>px`；圆角样本 `radius/<nn>`；动效时长每行 `duration/<ms>`；组件按 6 个类目分组（按钮 / 输入 / 徽章 / 标签页 / 列表与卡片 / 导航与浮层），类目 FRAME 名称前缀 `组件/`，首行放 18px 中文类目标签。
- **生成器约定**：CJK 文案一律用 `Noto Sans SC`（插件沙盒加载不到系统字体 `PingFang SC`；600 字重映射 Medium，700 映射 Bold）；`addText` 自动按是否含 CJK 选字族；页面级多行文案默认版心 1600（否则 x 偏右时可用宽度会被压到 24px、整段变竖排单字）。
- **重生成后必跑整理脚本**：`scripts/design-kit/organize-design-system.js`（MCP 通道执行）。脚本幂等：已分组的区块/token/组件会跳过，只补齐缺失分组、修正 CJK 字体与异常窄文本、按 96 间距统一堆叠区块。
- **整理脚本只做移动与分组**：不改文案内容、不改色值；重生成与整理应作为「生成 → 整理 → 巡检」流水线连续执行。

### 21. 全站配色 token 化（颜色模式 / 主题自适应）

- **硬规则**：应用页（02/03/04）画布上所有 SOLID 填充与描边都必须绑定 HotSpot Tokens 变量（`paint.boundVariables.color`），不得写死 hex；否则切换 vibrant/forest/inkwash × 明暗共 6 个模式时不会跟随。巡检规则 `hardcoded-color` 会逐屏统计并拦截（summary 的 `checkedColorPaints` 为已检查的纯色数量）。
- **叠字卡片（图片叠字）映射**：玻璃条 → `surface-nav-glass`（保留 `BACKGROUND_BLUR` 18 与原透明度，浅色解析为白、深色解析为深底）；条内深色药丸 → `surface-overlay`；标题/正文 → `text-primary`；时间/标签/计数 → `text-secondary`；药丸内文字与图标 → `content-inverse`；图标矢量描边 → `text-secondary`（药丸内 `content-inverse`）。
- **管理后台（Web 抓取稿）映射**：表格底/卡片 → `surface-primary`，表头/分割线 → `surface-subtle`，文字灰阶 → `text-primary / text-secondary`，状态徽章 → `status-*-bg`（10~14% 透明）+ `text-status-*-on-soft`，按钮/链接 → `accent-default`，删除/封禁 → `destructive`，输入框 → `field-bg / field-border`，图标框透明化 + 描边 `text-secondary`，导航/筛选激活 pill → `accent-soft` @16% + `text-accent-on-soft`。
- **实现要点（踩过的坑）**：
  - **只处理未绑定的 paint**，不要「先重置再重绑」：重置会丢掉原有 token 绑定并让 fallback 色参与渲染，画面会花；保留已绑定 paint 是最安全的增量方式。
  - 绑定用 `figma.variables.setBoundVariableForPaint(paint, 'color', variable)`，并把 paint.color 设为**当前模式的解析色**作为 fallback，避免黑色 fallback 参与渲染。
  - 模式识别：沿父链读 `resolvedVariableModes[collection.id]`；按页面兜底（02 vibrant-light / 03 forest-light / 04 vibrant-dark）；禁止 `getLocalVariableCollectionByIdAsync`（部分运行时不存在，用 `getLocalVariableCollectionsAsync`）。
  - 循环里声明 `var alpha;` 不会自动重置（var 提升），必须在每次 paint 迭代显式 `alpha = undefined`，否则浅色 tint 的透明度会泄漏到文字上导致整屏发灰。
  - 不要同时对 TEXT 写 `characters`：字体未加载会抛 `Cannot write to node with unloaded font`；只改 fills 是安全的。
- **主题自适应验证**：对代表屏临时 `setExplicitVariableModeForCollection(coll, '5:3')`（forest-dark 等）→ 截图确认玻璃条/文字/药丸/后台表格全部翻转 → 再切回原模式（5:2 等），过程中不要留下残留模式。
- **收口标准**：巡检 `hardcoded-color` / `overlay-glass-theme` / `overlay-text-theme` 全部 0 违规；三端抽样截图在浅色与深色下均无不可读文字、无刺眼残留亮/暗块。

### 22. 细节巡检与修复（文字重叠 / 裁切 / 字体 / 溢出）

- **巡检已覆盖的细节规则**（`audit-figma-design.js`，summary 含 `checkedTexts` / `checkedColorPaints`）：
  - `hardcoded-color`：颜色未绑定变量；
  - `cjk-font-missing`：中文文案字体不是 Noto Sans SC / PingFang SC；
  - `text-overflow-box`：文案渲染尺寸超出自身文本框（NONE/HEIGHT 自适应）> 2px；
  - `text-row-overlap`：同父级「标题 + 副标题」字盒纵向重叠（行距小于字高）；
  - `screen-cut`：直接子元素被屏幕底边裁切（bottom > screenH + 1）。
- **CJK 字体修复**：全站遍历 TEXT，含中文且 family 非 Noto/PingFang 的，按原 style 映射到 Noto Sans SC（Bold→Bold、Medium/SemiBold→Medium、其余→Regular）；Inter 里的中文会字距错乱或显示缺字。
- **标题/副标题重叠修复**：同父级、|Δx| ≤ 2、第二行 fontSize ≤ 11 且纵向间距小于第一行字高时，把第二行 `y = 第一行.y + 第一行.height + 1`；一般行距 52px 内安全。
- **屏幕底裁切修复**：只处理真正越界的元素（bottom > screenH + 1），上移后留 8~16px 安全边距。**不要**按 `bottom >= screenH-1` 批量上移（会把贴底 TabBar/输入栏/操作栏全部抬起），也不要事后对已贴底元素做统一 flush 还原（会把底部簇内部的相对间距改坏，引出一批新的重叠）；改完逐屏截图确认。
- **文字横向溢出修复**：优先截断文案；若节点是 PingFang（沙盒不可加载，不能改 characters），新建同位置同字号的 Noto Sans SC 文本、复制 fills/行高，截断到文本框宽度内，再把原节点 `visible = false`。
- **文本框高度滞后**（渲染高度只超 2~3px）：加载字体后重设 `characters` 并按 `NONE` → `HEIGHT` 重排即可。
- **判定口径**：滚动内容被固定 TabBar/评论输入栏覆盖、红点紧贴文字等属于正常层级叠压，不算缺陷；只修同父级行内相撞、越界裁切、缺字与溢出。

### 23. 平板页面一致性 & 底部安全区（home indicator）

- **平板页面基准**（页面 03，屏幕 834×1112）：内容页边距 24（内容宽 786）；页头返回圆钮 42×42 @ (22,12)、返回图标 15×15 @ (36,26)、标题 28px @ (74,20)、右侧操作（保存等）右对齐（右边距 22，56×40 按钮 x=756、文字 x=772）。
- **内容缩进异常判定**：某页内容宽 720 / 左右各 57 边距 / 返回钮 x=55，说明是按 768 视口生成的。修法：宽卡片/表单 `x-=33` 且 `width+=66`；左锚定元素 `x-=33`；右锚定元素 `x+=33`；页头元素同步归位→整屏截图与其他页比对。
- **底部安全区**（小白条）：移动端 34px、平板 20px、桌面 0。
  - 悬浮底部导航：整体上移，使 bar 底边 = 屏高 − 安全区；
  - 底部输入条（聊天/评论）：条背景高度 += 安全区（底边保持贴屏），条内输入框/占位/发送按钮上移等量；玻璃底改为同一 token 的不透明版本，并把滚动内容中被压住的节点 z 序移到条下方，避免文字浮在输入框上；
  - 底部 CTA / 操作条：按钮上移到安全区之上；
  - 地图页底部面板：面板整体上移留出安全区，必要时补回不透明底衬（surface-primary 变量）。
- **踩坑（务必照做）**：
  - 不要按 `bottom >= 屏高-1` 批量上移底部元素：会把贴底的 TabBar/输入条全部抬起来；
  - 不要对已贴底元素做统一 flush 还原：会破坏底部簇内部相对间距，引出一批新的文字重叠；
  - 导航条簇要精确圈定：导航文案的直系祖先（item frame）+ 中间按钮 + 背景条；“与 bar 相交且尺寸小”会误移条后的卡片位置 chip，需按重叠检测结果逐个回滚；
  - 找 bar 几何用 `absoluteBoundingBox`，**不要用 `absoluteRenderBounds`**：阴影会把 render bounds 撑大 12~15px，导航安全区会被误判为不足。
- **门禁**：巡检规则 `safe-area-bottom` 会拦截（导航底边必须 ≥ 安全区、输入占位文案 bottom 必须 ≤ 屏高 − 安全区）；全宽底部条覆盖滚动内容属正常层级，`text-row-overlap` 规则已对“被全宽底条覆盖的文本对”做豁免。

### 24. 顶部安全区（状态栏 / 灵动岛）

- **inset 取值**：移动端 47px、平板 24px、桌面 0（浏览器自带 chrome）。
- **两类页面两种处理**：
  - **标准页**（有页头/内容列，非全出血媒体）：画布框架**扩高** `H += inset`（移动 844→891、平板 1112→1136）；全宽背景/头部底条（`w ≥ 0.99W && y ≤ 1`）**高度 += inset、y 保持 0**（状态栏区显示同一背景色）；**其余全部子节点 y += inset**（含底部容器，随新底边整体下移）。此法保证构图零变化、无新增贴底裁切。
  - **沉浸页**（地图选点、瞬间详情等照片/地图全出血）：**不改框架尺寸**，仅下移顶部悬浮控件（`w ≤ 200 && h ≤ 80 && 中心 y < 48`，如返回钮、筛选 pill、面包屑），照片/地图/渐变保持全出血。
- **判定 immersive**：屏名含「地图选点」或以「瞬间详情」开头；其余一律标准页。
- **画布列间距**：扩高前检查同列相邻屏间距（当前 120px ≥ 47px，无需级联下移；若间距 < inset 需把同列后续屏顺次下移）。
- **门禁**：
  - `safe-area-top`：直连子元素（非全宽背景、非屏外延入媒体、`0 < y < inset−1`、h ≤ 120）进入顶部安全区即违规；
  - `screen-cut` 语义已改为「底部固定容器被裁切」：只查 `FRAME && w ≥ 0.5W && h ≤ 200` 且 bottom 超出屏幕的容器（滚动内容列贴底裁切属正常滚动状态，不检查；固定条容器化后此规则才可这样收窄）。
- **坑**：容器内故意出血的衬底矩形（如地图面板底衬）会撑大容器 bounds 导致误报，把衬底收齐到屏幕底边即可；「扩高」方案比「同高度硬移内容」安全——后者会让表单页底部内容（如兴趣标签最后一行）被底部 CTA 盖掉。

### 25. 可交互原型演示（Figma Motion / 变体交互组件）

- **用途**：把规范页的「交互动效模式」做成真正可点/可悬停的原型演示，而不是静态示意图；画布上放组件集与默认变体实例，动效在 ▶ 演示模式中播放。
- **结构**：每个演示 = 一个变体组件集（`figma.combineAsVariants`，变体命名 `State=Xxx`）+ 卡片内一个默认变体实例；组件集统一放页面底部「原型演示组件」区，命名 `动效演示/序号 名称`。
- **反应 schema（当前运行时实测）**——注意与旧 typings 差异很大：
  - 字段是 `actions`（数组），**不是** `action`（会报「update the actions field」）；
  - 触发器枚举：`ON_CLICK` / `MOUSE_ENTER` / `MOUSE_LEAVE`（后两个 `delay: 0` 必填）/ `ON_HOVER` / `ON_PRESS` / `AFTER_TIMEOUT`（需 `timeout`）等；**没有** `CLICK`；
  - 节点动作：`{ type: 'NODE', destinationId, navigation: 'CHANGE_TO', transition: { type: 'SMART_ANIMATE', easing, duration } }` —— `navigation` 是**字符串枚举**（`NAVIGATE | SWAP | OVERLAY | SCROLL_TO | CHANGE_TO`），不是对象；
  - 自定义贝塞尔缓动：`{ type: 'CUSTOM_CUBIC_BEZIER', easingFunctionCubicBezier: { x1, y1, x2, y2 } }` —— **对象**，不是数组；枚举缓动还有 `GENTLE / QUICK / BOUNCY / SLOW / CUSTOM_SPRING` 与 `EASE_OUT_BACK` 等；
  - **destination 必须是同一组件集内的变体**：指向松散组件会报「destination rejected」。
- **动效参数对齐 token**：演示的 duration/easing 直接用规范页 token（100ms EASE_OUT 按压、180ms 贝塞尔(0.34,1.56,0.64,1) 开关与指示器、220ms EASE_OUT_BACK 状态弹跳、220ms EASE_OUT 悬浮、260ms EASE_OUT/EASE_IN_AND_OUT 抽屉）。
- **SMART_ANIMATE 前提**：两个变体的图层同名同结构（如 `Track` / `Thumb` / `Pill` / `Card` / `Sheet` / `Scrim`），名字对不上就不补间；文字内容不同会直接切换不参与动画。
- **Closed/隐藏初态必须有可见触发物**：如底部抽屉 Closed 态放「打开抽屉」按钮（Open 态把它移出裁切区），否则演示区看起来是空的（用户会以为没做动效）。
- **画布静帧看不出动画**：给每张演示卡配提示文案（如「▶ 点击体验…」），并在区块副标题注明「演示模式可点击」；`flowStartingPoints` 在当前运行时不受支持，查看入口是选中画板或组件集后 ▶ 演示。
- **已建演示**：01 按钮按压 / 02 弹簧开关 / 03 分段选项 / 04 状态弹跳(点赞) / 05 悬浮抬升(hover) / 06 底部抽屉，位于设计规范页「05 动效」区块与页面底部组件区。

### 26. 三端审美一致性（跨页风格基线）

- **基准字号**：页头大标题 20（三端统一；`fs=18/24` 的旧页头都是历史遗留）。移动端页头标题多为 PingFang 且不能直接改字号（沙盒不可加载），修法：新建 Noto Sans SC Medium 20 文本替换并删除原节点。
- **段落标题**：fs=14 Regular 为正文体；fs=14 加权标题用 Medium（不用 Bold，Bold 仅保留品牌字标）。
- **卡片圆角基线**：大卡片 20（绝对主导）；输入框/次级行 16；底部悬浮导航条 28；设置分组卡 24。巡检时可按「尺寸 ≥150×44 的圆角矩形」统计分布，少数派逐个核对是否为刻意设计。
- **文案一致性**：跨端同名屏逐字段 diff（设置 / 个人中心 / 附近瞬间 / 首页信息流重点核对），发现「个人数据」vs「我的内容」这类同一概念不同名的，统一为多数派叫法。PingFang 节点改文案用「新建 Noto 替换 + 删原节点」。
- **字体单一化**：中文文案只允许 Noto Sans SC（PingFang 在沙盒不可加载、Inter 缺中文字形）；扫描口径 `family==='PingFang SC' && 含CJK` → 按原字重映射 Regular/Medium/Bold 批量迁移。Inter 仅保留品牌字标（HotSpot logo：Inter Bold）。
- **画布网格**：02 列宽 390 / 间距 73、03 列宽 834 / 间距 96、04 列宽 1440 / 间距 120；同名屏三端同序。01 规范页 7 区块统一宽 1650（主题区 1674 因 6 卡并排略宽），纵向顺排。
- **安全区与画布联动**：顶部安全区使 02 屏高 891 / 03 屏高 1136（旧 844/1112 为历史值，属正常共存，不算不一致）。
- **收口验证**：巡检 `violations=0` + 细节扫描（`cjk=0 / overflow=0 / clipped=0`）+ 三端同名屏截图并排比对。

## 命令 / 脚本通道

### 1. 维护 Session ID
Session 失效或新建任务时，调用 `mcp.ts` 生成会话：

```bash
cd /Users/liyou/work/hotspot/.workwork/figma-mcp
bun run mcp.ts call create_session '{"fileKey":"<fileKey>","description":"automation sync"}'
```

`fileKey` 取自设计稿链接的 `/design/<fileKey>/` 段（仓库不写入具体 key）。输出中的 `sessionId` 填入环境变量：`export SESSION_ID="<UUID>"`。

### 2. 执行 Figma 自动化脚本
使用封装脚本向目标文档推送执行：

```bash
FILE_KEY="<fileKey>" \
SESSION_ID="<UUID>" \
bash /Users/liyou/work/hotspot/.workwork/figma-capture/use-figma.sh \
  "<path-to-script.js>" \
  "<script-description>" \
  "resource:figma-use"
```

### 3. 获取截图并核验

```bash
cd /Users/liyou/work/hotspot/.workwork/figma-mcp
bun run mcp.ts call get_screenshot '{"nodeId":"<nodeId>","fileKey":"<fileKey>"}'
```

根据返回的 URL 下载快照至 `.workwork/` 或 artifacts 目录，核对渲染结果。

### 4. 同步后巡检（质量门禁）

设计稿同步（插件生成）完成后立即执行，输出偏差列表：

```bash
FILE_KEY="<fileKey>" \
SESSION_ID="<UUID>" \
bash /Users/liyou/work/hotspot/.workwork/figma-capture/use-figma.sh \
  scripts/design-kit/audit-figma-design.js \
  "design audit" \
  "resource:figma-use"
```

- 返回 JSON：`summary.pass` / `summary.byRule` / `violations` / `warnings`。
- `violations` 条目含 `rule`、`page`、`screen`、节点 id、偏差值与 `hint`；修完重跑，直到 `violations` 为空。
- `warnings` 不阻塞同步，但同类（如 `tab-row-va-mixed`）应在下一次改动中收敛，避免后续漂移。

### 5. 打里程碑标记（saveVersionHistoryAsync）

同步或批量修复完成后，用脚本在 Figma 侧打里程碑标记（不需要桌面 UI）：

```js
// vh.js —— 通过 use-figma.sh 在服务端沙盒执行
var now = new Date();
var timeStr = (now.getMonth() + 1) + '月' + now.getDate() + '日 ' +
              String(now.getHours()).padStart(2, '0') + ':' +
              String(now.getMinutes()).padStart(2, '0');
await figma.saveVersionHistoryAsync(
  '细节精修版 · <主题> (' + timeStr + ')',
  '【本次版本更新内容摘要】\n\n1. <模块>：<要点>\n2. <模块>：<要点>'
);
return { ok: true, title: '...' };
```

```bash
FILE_KEY="<fileKey>" \
SESSION_ID="<UUID>" \
bash /Users/liyou/work/hotspot/.workwork/figma-capture/use-figma.sh \
  vh.js "save version history" "resource:figma-use"
```

- 标题沿用 design-kit 插件同步时的命名：`细节精修版 · <主题> (M月D日 HH:MM)`；
- 描述用「1. 模块：要点」结构，一条记录覆盖本轮全部改动，含巡检结论（0 违规）；
- 通道限制：Figma REST API 的 `/v1/files/:key/versions` 只读、MCP 工具列表里没有版本工具，唯一可写通道就是插件 API `saveVersionHistoryAsync`（选区外无需桌面端插件）。

## 陷阱 / 限制

- **陷阱：远程 MCP 缺 `X-Figma-Plugin-Bundle` 头会落到受限插件运行时**：`use_figma` 的默认插件 bundle 缺少部分编辑器 API（例：`figma.saveVersionHistoryAsync` 报 `not a supported API`）。请求头必须带 `X-Figma-Plugin-Bundle: figma_prod@2_2_208`（与官方 Gemini / Claude 客户端一致）；本地 `.workwork/figma-mcp/mcp.ts` 已内置该头并支持 `FIGMA_PLUGIN_BUNDLE` 环境变量覆盖。遇到「编辑器 API 不存在」先核对这个头，不要改用桌面 UI 自动化绕行。
- **陷阱：`figma.mixed` Symbol 类型与数值比较**：Figma 中多角不同的矩形其 `cornerRadius` 为 `Symbol(figma.mixed)`。执行 `n.cornerRadius > 3` 会触发致命异常 `cannot convert symbol to number`。筛选条件必须加类型防御：`typeof n.cornerRadius === 'number' && n.cornerRadius > 3`。
- **陷阱：`textAlignVertical` 触发字体未加载报错**：很多人误以为只有改 `characters` 才需要加载字体，实际上修改 `textAlignVertical` 也会促使引擎重算排版，遇到未加载字形会直接抛错。必须在修改前先为文本赋值已加载的字体（`Noto Sans SC`）。
- **陷阱：遍历删除引发级联失效**：删除父节点后，其包含的所有子节点会被 Figma 底层自动级联销毁。如果遍历列表仍引用了已被销毁的子节点，会抛出 `The node with id "..." does not exist`。必须按树层级自顶向下整块删除，或将遍历逻辑包裹 `try { node.remove(); } catch (e) {}`。
- **陷阱：非活跃孤立画板混淆**：画布中可能存在未连入主视口的独立测试画板。遍历或修改前必须确认节点挂载在当前活跃画板列表（`figma.currentPage.children`），避免改动脱离主屏的游离图层。
- **陷阱：生成端「选中居中、未选中顶对齐」造成同行基线漂移**：`figma-runtime.js` 的 `addText` 只对 `align === 'center'` 或徽章文本做垂直居中，同一行里选中项（居中）与未选中项（顶对齐）因此相差约 6px。巡检的 `tab-row-baseline` 会直接报出；根治需在生成端统一单行文本的垂直居中策略（影响所有单行文本，须全量截图回归），未根治前以巡检兜底。
- **陷阱：下拉框占位符图层叠加**：在重构筛选区域时，下拉框背景、文字占位符与下方 Tab 栏坐标容易计算偏差导致重叠。必须准确测量筛选容器实际高度，按容器内部相对坐标（如 `Y = container.y + (container.height - text.height) / 2`）精确定位。

## 权威源

- **Figma 执行脚本入口**：`/Users/liyou/work/hotspot/.workwork/figma-capture/use-figma.sh`
- **MCP 客户端桥接器**：`/Users/liyou/work/hotspot/.workwork/figma-mcp/mcp.ts`
- **同步后巡检脚本**：`scripts/design-kit/audit-figma-design.js`
- **Figma 设计稿目标**：HotSpot UI Design Kit（`fileKey` 取自设计稿链接的 `/design/<fileKey>/` 段，不写入仓库）
