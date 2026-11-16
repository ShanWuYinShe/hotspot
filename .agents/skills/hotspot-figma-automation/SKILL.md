---
name: hotspot-figma-automation
description: Figma 设计稿无感云端审查与自动化同步工作流：MCP session 维护、use_figma 执行脚本、Google Fonts 预加载防崩溃、数学级中心线重构公式、移动端去系统化与无菌化准则、截图视觉闭环验证、同步后自动巡检门禁。触发：自动化审查/修改 Figma 设计稿；同步后跑巡检、排查同行标签基线或页头居中偏差；状态栏/时钟清理；图层错位重构；use_figma 字体报错；同步设计稿至 Figma；Figma MCP 调试。
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

### 15. 同步后自动巡检门禁（同行基线 / 页头居中）

- **同步完成后必须跑巡检脚本**（命令见「命令 / 脚本通道 4」）；返回 `summary.pass === true` 才视为同步通过；`violations` 非空时按 `hint` 逐条修复并重新巡检，直至归零。
- 巡检为只读脚本，不修改任何节点；输出 `{ summary, violations, warnings }`，`summary.byRule` 给出分类计数。
- 规则与阈值：
  - `tab-row-baseline`：同一胶囊标签行（Tab / Chip / 筛选，含选中态与数字）内各标签字形中心偏差 > 2px → 违规，按 §10 修复。
  - `tab-row-va-mixed`：同一行标签 `textAlignVertical` 混用 → 警告。当前可能看似对齐，但字体/字号变化后会再次漂移，应在下次改动时统一。
  - `header-center`：移动/平板页头标题与返回按钮中心偏差 > 2px → 违规。单行标题比字形中心；标题+副标题两行块比整块中心；桌面端页头为标题胶囊结构，不在此规则内，仍需截图核对。
  - `card-chip-edge` / `card-text-edge`：卡片右上角时间胶囊（或纯文字）右缘贴边/溢出（右内边距 < 6px）→ 违规，按 §16 修复。
  - `card-chip-inset-asym`：时间胶囊右内边距与上内边距相差 > 3px → 警告，建议统一为相同 inset。

### 16. 卡片右上角时间标签必须内缩（禁止贴边/出血）

- **禁止把时间胶囊的右缘对齐到卡片右边界**：胶囊右缘超出卡片（负内边距）会被卡片裁切或压住圆角，视觉上像「溢出」，是生成端最常见的系统性错位。
- **内缩方程**（inset 取卡片顶部内边距，通常 10px）：
  $$X_{chip} = W_{card} - \text{inset} - W_{chip}$$
  $$X_{text} = X_{chip} + \frac{W_{chip} - W_{text}}{2}$$
  文字与胶囊同步平移，保持胶囊内水平居中；右内边距必须与上内边距对称（差值 ≤ 3px）。
- 卡片右上角若为纯文字（无胶囊底），同样按 inset 内缩，右内边距不得小于 6px。

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

## 陷阱 / 限制

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
