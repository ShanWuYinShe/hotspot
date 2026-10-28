# 鸿蒙端主题/深色验收量化基线（hotspot harmonyapp）

本文件是 skill `hotspot-harmony-ui`「主题、A11y 与触控验收」的量化明细：改主题、改颜色、或做深色验收时**不要只看截图感觉**，按下表跑一遍。

## 1. 文字对比度（WCAG 2.x）

取 `ThemeConstants.ets` 每个主题块的 `pageBackground / surfaceBackground / textPrimary / textSecondary / textTertiary`，按 WCAG 相对亮度公式算比值，正文要求 **≥4.5:1**。本机实测基线（3 主题 × 2 明暗，全部达标）：

| 主题·模式 | primary | secondary | tertiary |
|---|---|---|---|
| 清野 浅/深 | 15.31 / 15.24 | 7.50 / 8.84 | 5.12 / 5.04 |
| 星际 浅/深 | 14.12 / 16.19 | 5.41 / 8.93 | 4.74 / 5.35 |
| 水墨 浅/深 | 12.84 / 13.99 | 4.82 / 7.02 | **4.68** / 5.20 |

最低值是水墨浅色的 tertiary 4.68 —— **改任何 text/bg token 后必须重跑，低于 4.5 即回归**。

## 2. 深色下的层级靠加亮的描边（不能只看 alpha）

`ThemeConstants.ets` 注释原文：「深色下黑阴影对比弱，卡片抬升改用更亮的描边承载（border 单独加亮），阴影只做轻微加深」。`borderColor` 是 `#AARRGGBB`，验收时要算**与底色合成后**的颜色再求对比度。实测基线：

| 主题·模式 | 描边合成后 vs 底色 |
|---|---|
| 清野 浅 / 深 | 1.511 / 1.476 |
| 星际 浅 / 深 | **1.271（偏弱）** / 1.414 |
| 水墨 浅 / 深 | 1.463 / 1.553 |

经验阈值：**≥1.35 清晰、1.2~1.35 偏弱、<1.2 过弱**。星际浅色 1.271 偏弱属跨端一致的既有设计（前端 `--cst-palette-border` 同值），**修要两端一起改**，不要只改鸿蒙端。

## 3. 主题族运行时矩阵（6 组合）

用模拟器在设置页逐组合切换截图确认：

| 组合 | 结果 |
|---|---|
| vibrant 浅/深 | ✅（多页实测） |
| 清野 浅色 | ✅ 绿调米白底 #F6F7F3、深绿选中 pill 白字、图标徽章绿调 |
| 清野 深色 | ✅ 绿调暗底 #161A16、**亮绿 pill + 深绿文字**（对比清晰）、徽章绿调 |
| 水墨 浅色 | ✅（同批截图确认） |
| 水墨 深色 | ✅ 暖黑底 #181A1C、珊瑚红 pill + 深色文字、徽章红调 |

选中的明暗分段 pill 在深色下均为「亮 accent 底 + 深 accent 文字」——与静态对比度基线一致。头像游客色不随主题变（身份色独立于主题 accent，业界惯例）。

## 4. 主题档位覆写与 harmony-only 档

- **forest light 软岛**：仅 `default_clean_light` 的 `surfaceIsland` = `mixOpaque(surfaceBackground 92%, textPrimary 8%)` = #E4E6E2；其余 5 组公式不变；surface parity 6/6。
- **textTertiary**：forest light 调为 #58645C（岛上 4.93），文档标注 harmony-only。
- 改 token 后按两端代码逐值比对（web `themes/*.css` 的 `--cst-surface-island` ↔ 鸿蒙 `AppThemeManager` 的 `surfaceIsland`，6 套组合；仓库无独立入库校验脚本），并同步更新 `docs/frontend-ui-audit.md` 记录；两端同改见 SKILL.md 跨端同步表。

## 5. 设计 token 对照 HarmonyOS 官方规范

依据华为官方 UX/UI 规范（鸿蒙设计规范文档，官网原文核实）：系统通用圆角 **16vp**；卡片之间间距 **12vp**；卡片内主标题 **16sp** / 副标题 **12sp**（90% / 60% 不透明度）；默认鸿蒙黑体，单页不超过 2 种字体；卡片内最大字号 **24**。

本项目 `DesignConstants` 对照结果：

| 官方项 | 官方值 | 本项目 | 结论 |
|---|---|---|---|
| 卡片之间间距 | 12vp | `SPACING.MD = 12`（实测枢纽/首页卡片列 `space: SPACING.MD`） | ✅ 一致 |
| 系统通用圆角 | 16vp | `RADIUS.MD = 16`（`COMPACT_CARD` / `INPUT` 用它） | ✅ 一致 |
| 卡片主标题 | 16sp | `FONT.SIZE.LG = 16` | ✅ 有对应档 |
| 卡片副标题 | 12sp | `FONT.SIZE.SM = 12` | ✅ 有对应档 |
| 卡片内最大字号 | 24 | `XXL = 24`；更大的 `XXXL 30` / `HERO 36` 只用于页面级标题 | ✅ 限界内 |

**已登记的有意偏差（跨端统一，非缺陷）**：

1. **普通卡片圆角 20vp**（`SHAPE.CARD = RADIUS.LG`）而官方通用 16vp —— 对齐前端 `rounded-lg = 20`，跨端一致优先；紧凑卡/输入框仍用 16 与官方一致。
2. **正文字号 14**（`FONT.SIZE.MD`）小于官方正文量级 16 —— 对齐前端 `text-md = 14`；展示型标题仍走 20/24/30/36 档。

新增页面/组件必须从现有档位取值，不得引入近邻档（13/15/18/22/28 已归并过一轮）。
