---
paths:
  - "entry/src/main/ets/**/*.{ets,ts}"
---

# 设计系统（harmonyapp）

所有颜色、字号、间距、圆角、阴影**只用 DesignConstants / AppThemeManager token**，对齐前端 SvelteKit forest 设计系统（单一事实源）。**禁止**在页面/组件里硬编码十六进制色值或裸数字 `fontSize(N)` / `padding({left:N})`。

## 颜色

- 画布：`COLOR.PRIMARY_BG_*` / `SECONDARY_BG_*`
- 表面：`COLOR.SURFACE_NEUTRAL_SOFT` / `SURFACE_INFO_SOFT` 等
- 文字：`COLOR.TEXT_PRIMARY_*` / `TEXT_SECONDARY_*` / `TEXT_TERTIARY_*`
- 品牌：`COLOR.BRAND_PRIMARY`（forest `#3D7A55`，与前端 `--theme-accent-primary` 一致）
- 语义：`SUCCESS` / `WARNING` / `ERROR` / `INFO`
- 主题解析：`AppThemeManager.getPaletteById(resolvedThemeId)` → `@StorageProp(KEY_RESOLVED_THEME_ID)`

## 字号（FONT.SIZE，禁用裸 `fontSize(N)`）

`XS_MINUS 9` / `XS2 10` / `XS 11` / `SM 12` / `SM_PLUS 13` / `MD 14` / `MD_PLUS 15` / `LG 16` / `LG_PLUS 18` / `XL 20` / `XL_PLUS 22` / `XXL 24` / `XXXL 32` / `HERO 48`。字重 `FONT.WEIGHT.{LIGHT,REGULAR,MEDIUM,SEMIBOLD,BOLD}`。

## 间距（SPACING，4px 网格）

`XS 4` / `SM 8` / `MD 12` / `LG 16` / `XL 24` / `XXL 32` / `XXXL 48`。**禁止**组件内 `padding({left:16})` 这类裸数——用 `SPACING.LG`。

## 圆角（RADIUS / SHAPE）

`SM 8` / `MD 12` / `LG 16` / `XL 18` / `XXL 22` / `FULL 999`。语义形状 `SHAPE.{CARD 22, HERO_CARD 24, INPUT 18, BUTTON 28, CHIP FULL, TAB_BAR 24}`。与前端 `--radius-*` 完全一致。

## 页面头部（强制 SystemPageHeader）

- **所有页面头部必须用 `SystemPageHeader`**（或专用组件 `SearchPageHeader` / `ChatPageHeader`）。
- 禁止手搓 `Row { Text(...).fontSize(XL)... Button(...) }` 头部。
- SystemPageHeader API：`title` / `subtitle` / `showBackButton`+`onBack` / `action`(文本按钮) / `leftAction`+`rightAction`(IconAction) / `trailing`(@BuilderParam 任意尾随) / `titleAlign` / `titleSize` / `titleWeight` / `backgroundVariant`('bar'|'transparent'|'card')。
- 安全区由 SystemPageHeader 内聚（自带 `@StorageLink('safeAreaTop')` + `expandSafeArea`），页面无需重复写。
- 内联搜索框头部 → `SearchPageHeader`；浮动卡片+头像头部 → `ChatPageHeader`。

## 页面内容栏（PageShellContainer）

档位（对齐前端 `.page-shell`）：`form 560` / `narrow 720` / `default 1180` / `wide 1320`；feed：`feed-narrow 640` / `feed-medium 960`。
