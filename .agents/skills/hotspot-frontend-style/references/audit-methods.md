# 前端视觉审计方法与假阳性规律（hotspot web）

本文件是 skill `hotspot-frontend-style` 的明细补充：压图对比度像素实测、按钮「原语绕过」审计判据、脚本假阳性规律。方法与数据来自既往实测，**未在近期重跑**，重跑后请以新结论更新此处。

## 1. 压图文字对比度的像素级实测法

自动指标（`getComputedStyle` 取色）对压在图片上的文字**不可信**：它拿到的是父级不透明度叠加值而非图片像素，只能把这类元素单列（hotspot 审计脚本按 `contrastOverMedia` 分类，不计入 AA 结论）。要判定必须实测像素：

1. 找到候选元素 → `document.createRange(); selectNodeContents(el)` 取**文字节点自身的 `getBoundingClientRect()`**；
2. 截图该矩形 → 把元素 `style.color='transparent'`（**只改颜色，保布局**）→ 再截同一矩形，得到「纯背景」像素；
3. 逐像素算 `contrast(文字色 over 背景像素, 背景像素)`，取 min 与 P5；文字带 alpha 时逐像素合成 `eff = fg*a + bg*(1-a)` 再算。

**关键坑（会得出完全相反的结论）**：测量区若用**元素矩形**（含 padding），padding 下根本没有字形、那里是胶囊圆角外的原图像素，会把最差比压到 1.0–4.0 造成假阴性。同一批 176 个候选：用元素矩形 → 15 处「不达标」（6 组）；改用字形区 → **仅 1 处**（`/groups/group_1` 分类胶囊，白字压 `glass-badge` 0.18 白玻璃底，实测 4.16/需 4.5，84.9% 字形像素不达标）。修复后同法复测：4.16 → 14.18、达标像素 13.3% → 100%。

**结论**：压图徽标一律用深色遮罩（`bg-surface-media-scrim-strong` = 0.55 黑，白字在纯白底上仍有 ~4.9:1），不要用浅色玻璃底 + 白字（`glass-badge` 0.18 白），那是结构性不达标。

## 2. 前端「原语绕过」审计判据

前端与鸿蒙的原语纪律**不同**：鸿蒙强制 SystemButton 原语；前端允许原生 `<button>` + token 类（Svelte/Tailwind 惯例），约束落在**交互反馈类**（SKILL.md 自查清单第 7 条）。据此对全仓 209 处原生 `<button>` 做了审计，**结论：无缺口**。合规基线（命中任一即合规）：

- 反馈工具类：`hover-btn` / `hover-bg` / `hover:bg-*` / `tap-*` / `scale-btn` / `focus-ring` / `focus-visible:ring-*`
- `link-btn` 工具类（自带 transition + underline 反馈）
- 共享常量：admin 页的 `ROW_BTN / ROW_BTN_PRIMARY / ROW_BTN_DANGER`（`routes/(app)/admin/shared/table.ts`，内含完整 hover/tap/focus/disabled 四态）
- `ListItem` 组件（列表行，自查清单第 5 条）

**审计脚本的坑（本次 106/209 误报）**：

1. 多行 button 标签被属性值里的 `=>` 提前截断（`[^>]*` 遇 `>` 即停），class 没采到；
2. class 是常量引用（`class={ROW_BTN}`）时不含字面关键词；
3. `link-btn` / `focus-ring` / `focus-visible` 不在关键词表里。

→ 与鸿蒙端两轮结论一致：**启发式只产候选清单，合规判定必须人工开上下文**；且**不要拿鸿蒙的原语标准套前端**——两端的「合规」定义不同。

## 3. 触控热区审计方法（agent-browser 量取）

调研共识（移动端交互规范）：触控目标的有效热区 ≥ 44×44。项目用 `.hit-slop`（`::before` 外扩 6px）实现「视觉小、热区大」。审计步骤：`set viewport 390 844` → `open <路由>` → `eval` 遍历 `button, a[href], [role=button]` 收集 `getBoundingClientRect()`，与 SKILL.md 的换算表核对。

## 4. 审计脚本通用规律（web/鸿蒙共通，单点权威）

> 本节是 web 与鸿蒙两侧「审计假阳性」通用规律的**单点权威**；鸿蒙端只保留端侧实例与数据（skill `hotspot-harmony-ui` 的「审计基线与假阳性规律」）。

- 启发式扫「属性链上是否出现某关键词」误报率极高：真实承载常在**组件内部**、**状态驱动**或**父级行**。
- 布局类问题（图片尺寸、间距）不能用「附近有 Text/缺少数字宽高」粗判，要按**该元素自身**判断。
- 结论落盘必须附命令与日期，否则下次仍要重跑证实。
