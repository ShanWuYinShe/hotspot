# 宽屏回归基线（hotspot web）

本文件是 skill `hotspot-frontend-layout`「宽屏回归基线」的明细：视口、路由清单、测量口径与基线数值。

## 视口 1920×1080（10 个关键路由）

路由：`/`、`/search`、`/messages`、`/me`、`/nearby/moments`、`/notify`、`/settings`、`/admin`、`/users/user_lumos`、`/groups/group_1`。

- **内容填充率全部 0.97**（页面内容横向占满可用区域，侧栏宽度扣除后计算）；
- **无横向滚动**（`scrollWidth <= innerWidth + 2`）；
- `/search` **1920 下不是左右双列**：`SearchDiscovery.svelte:97` 为 `md:grid-cols-2 2xl:flex 2xl:flex-col`，1920 ≥ `2xl`(1792) 后变成**单列纵向**，内部卡片区 :98 转 `2xl:grid-cols-4`（4 列卡墙）。断点是 `md`(960) 起双列、`2xl`(1792) 起改单列，不是 `lg`。

## 视口 1397×750（同口径）

- 全站填充率 0.96；唯一 0.17 是 `/share` 的居中小卡，属预期。

## 取证方式

```bash
# 浏览器自动化一律走 agent-browser，禁止自研脚本 spawn Chrome
agent-browser set viewport 1920 1080
agent-browser open <url>
agent-browser eval "<测量表达式>"
```

测量表达式按「内容区宽度 / 可用视口宽度（扣除侧栏）」计算填充率，并同时取 `document.documentElement.scrollWidth`、`window.innerWidth` 判横向滚动。

## 判据

- 新增页面或改布局后复测上述两个视口：填充率不得明显下降（约 <0.9 即视为「内容塌回中间一条」）；
- 若出现横向滚动或右侧大片空白，优先查页面是否传了 `contentMaxWidth`/`maxWidth` 或窄列 `max-w-*`（见 SKILL.md「断点与内容宽度」）。
