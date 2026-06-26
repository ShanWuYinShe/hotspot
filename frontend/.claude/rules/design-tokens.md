---
paths:
  - 'src/**/*.{svelte,css}'
---

# 设计系统

所有颜色、阴影、留白**只用语义 token**（定义在 `src/app.css`，随 `[data-theme-family]` 主题族解析），**禁止**硬编码十六进制色值。

## 颜色 / 表面 token

- 画布：`--page-bg`
- 表面：`--surface-bg`（卡片/头部/输入面）
- 描边：`--border-theme`
- 文字：`--text-primary`（标题/正文） / `--text-secondary`（次要/说明/占位） / `--text-on-accent`（品牌色上文字）
- 品牌：`--brand` / `--glow` / `--theme-accent-soft` / `--theme-accent-soft-border`
- 语义：`--color-semantic-online` / `--semantic-hot` / `--semantic-warning`
- 地图：`--map-user` / `--map-group` / `--map-post`

## 表面与质感

- 当前主题族 **forest（扁平）**：阴影归零，组件扁平。其他主题族可有 neumorphic 阴影（`--shadow-light` / `--shadow-dark` / `--shadow-md` / `--shadow-lg`）。
- 复用类：`.surface-flat`（扁平交互面）、`.surface-hero` / `.ambient-glow`（概览卡）、`.input-surface`（输入容器）。
- 圆形图标按钮统一 `36px`（移动可 `38px`）。

## 圆角

- 基础 `--radius: 0.625rem`
- 卡片 `22px`、概览/主张卡 `24px`、输入/搜索 `18px`、小面 `16px`、chip/pill/头像 `999px`
- **token 阶梯**（`--radius-*`，禁用魔法数）：`xs 8` / `sm 12` / `md 18` / `lg 22` / `xl 24` / `xxl 22`(对齐鸿蒙 `RADIUS.XXL`) / `pill 40` / `button 28`。原语内联 `border-radius:Npx` 必须改用 `var(--radius-*)` 或 `rounded-[var(--radius-*)]`。

## 字号 / 字重

- **token 阶梯**（`--text-*` 命名空间，自动生成 `text-*` 工具类；禁用内联 `font-size:Npx`）：

  | token                    | class          | 用途                                                         |
  | ------------------------ | -------------- | ------------------------------------------------------------ |
  | `--text-xs` 10px         | `text-xs`      | eyebrow/micro（配 `.eyebrow` 类的 letter-spacing+uppercase） |
  | `--text-caption` 11px    | `text-caption` | 时间戳/badge/小注                                            |
  | `--text-meta` 12px       | `text-meta`    | 次要/说明                                                    |
  | `--text-body` 13px       | `text-body`    | 正文（最高频）                                               |
  | `--text-body-lg` 14px    | `text-body-lg` | 强调正文/主按钮                                              |
  | `--text-title` 16px      | `text-title`   | 卡片/列表标题                                                |
  | `--text-h2` 18px         | `text-h2`      | 页级标题（钻取/面板）                                        |
  | `--text-h1` 20px         | `text-h1`      | 区块 H1/stat                                                 |
  | `--text-display` 24px    | `text-display` | 门面页大标题/群组名（或流式 `.page-title--hero`）            |
  | `--text-display-lg` 36px | —              | 错误页 hero                                                  |

- **红线**：`.svelte` 内禁止内联 `style="font-size:Npx"`（已有 0 处）；scoped `<style>` 块与 JS 模板里用 `font-size: var(--text-*)` 引用。
- H1 字重 `600`，正文/标题按需 `400–700`。

## 间距 / 节奏

- 横向留白 `clamp`（`.page-pad` / `.page-pad-x`）
- 头部内元素纵向 `mb-2 ~ mb-3`
- 网格/列表 gap `.feed-density-grid` / `gap-2.5 ~ gap-5`
- 桌面侧边栏 `192px`，顶栏 `52px`，右栏 `220px` + `border-left`

## 动效

- 点击反馈缩放：`active:scale-[0.88~0.97]`，`transition-transform duration-150`
- 主题切换 `transition-colors duration-500`

## 图标

- 统一 `lucide-svelte`，按需单文件引入。线宽默认 `1.5`，选中/强调 `2`。
- **同一导航面内图标唯一表义**：一个图标不得在同一侧边栏/底栏里指代两个不同目的地。
- **图标默认单色**走 `--text-secondary` / `--text-primary`，不用 per-item 彩色图标。
- **图标实现一律用 lucide 组件**，禁止手写内联 `<svg>` 复刻 lucide 已有图标。
