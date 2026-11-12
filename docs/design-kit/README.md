# HotSpot UI Design Kit

Web（SvelteKit + Tailwind）与 HarmonyOS（ArkTS）共用的 UI 设计规范包，用于：

- 作为双端 UI 对齐的**单一事实源**（token 由代码生成，不手工维护第二份）；
- 导入 Figma / Stitch 做设计迭代与交付；
- 后续新增页面/组件时按本包的 token 与页面清单对齐。

## 目录

| 路径 | 内容 |
|---|---|
| `ui-kit.html` | 可交互设计规范页：三主题 × 明暗色彩、字体/圆角/间距/阴影/动效、组件样例、页面截图画廊、跨端对照、导入说明。浏览器直接打开即可（无需服务） |
| `tokens.json` | 合并后的完整 token（global + 6 个主题模式 set），供脚本/其他工具消费 |
| `tokens/global.json` | 与主题无关的 token：字体、字重、行高、圆角、间距、边框、透明度、动效、模糊、z-index、鸿蒙常量对照 |
| `tokens/themes/<theme>-<mode>.json` | 6 个主题模式 set（vibrant/forest/inkwash × light/dark），DTCG 格式，可直接导入 Figma Tokens Studio |
| `pages/` | 页面截图：`web-*.png`（390×844 实时页面）、`app-*.jpg`（HarmonyOS 模拟器/真机） |

## 重新生成

Token 与规范页均由脚本从代码生成，**不要手工编辑 `tokens.json` / `ui-kit.html`**：

```bash
node scripts/design-kit/build-tokens.mjs     # 解析 frontend themes/semantic/shared + harmonyapp DesignConstants
node scripts/design-kit/build-ui-kit.mjs     # 生成 docs/design-kit/ui-kit.html
bash scripts/design-kit/capture-pages.sh     # 可选：重新捕获 Web 390×844 页面截图（需本地 compose 与 QA 登录态）
```

Token 源头：

- Web：`frontend/src/styles/theme/{shared,semantic,theme-registry}.css`、`themes/{vibrant,forest,inkwash}.css`
- HarmonyOS：`harmonyapp/entry/src/main/ets/constants/DesignConstants.ets`

## 导入 Figma

1. 新建 Figma 文件；
2. 安装 **Tokens Studio for Figma** 插件，导入：
   - `tokens/global.json`（一个 set：global）；
   - `tokens/themes/vibrant-light.json`、`vibrant-dark.json`、`forest-*`、`inkwash-*`（每个一个 set，按主题+明暗切换）；
3. 把 `pages/` 里的截图拖入画布作为参考层（Figma 支持 PNG/JPG 直接拖入；也可 Place image 逐页放置）；
4. 按 `ui-kit.html` 的 Components 区建立 Figma 组件库：按钮（primary/secondary/soft/ghost/danger × sm/md/lg）、输入框（默认/聚焦/错误/禁用）、徽章与胶囊、Tabs（pills/segmented/underline）、列表行、卡片、底部弹层、Toast、空态/骨架。

## 导入 Stitch

1. 在 Stitch 新建项目；
2. 上传 `pages/` 中的截图作为参考图（Web 与 App 各一组）；
3. 粘贴下方「设计约束」作为生成提示；
4. Stitch 的产出属于探索稿，需回填到代码 token 后重新生成本包，避免出现第三份事实源。

## 设计约束（Stitch / Figma 生成提示）

- 移动端基准视口 390×844（Web）；鸿蒙逻辑宽约 320vp，同 token 下视觉密度接近。
- 只允许使用本包语义色（surface-page / surface-primary / text-primary / text-secondary / accent-default / status-online / status-warning / status-danger 等），禁止引入新色值；三主题（星际/清野/水墨）× 明暗必须同时可用。
- 字号 9 档：10 / 11 / 12 / 14 / 18 / 20 / 24 / 30 / 36；正文 12–14，辅助 11，标题 18+；禁止 1–2px 近邻档并存。
- 圆角 5 档：12 / 16 / 20 / 24 / 9999（底部 sheet 28）；卡片 20、输入框 16、按钮胶囊。
- 间距 4px 网格：2/4/6/8/10/12/14/16/18/20/24/32。
- 组件语言：主按钮 accent 实底 + 白字；筛选胶囊 accent-soft 淡底 + accent 字；分段控制器 accent 实底浮起；页级切换用下划线 Tab。
- 页面结构：全幅页头 + 内容软岛（surface-island）+ 底部悬浮胶囊 Dock；推入页隐藏 Dock。
- 状态色 on-soft 必须满足对比度 ≥4.5（见 `harmonyapp/scripts/check_status_parity.py`）。
- 触控热区 ≥44×44；图标按钮必须有无障碍标签。

## 跨端对齐与校验

| 校验 | 命令 |
|---|---|
| 主题变量集合一致 | `cd frontend && bun run check:theme`（含色彩/刻度/氛围画布/类名） |
| 状态色 on-soft 与对比度 | `python3 harmonyapp/scripts/check_status_parity.py` |
| 组件原语一致性 | `cd frontend && bun run check:ui-consistency` |

改任一端 token 后必须同步另一端并通过以上校验，再重新生成设计包。

## 页面清单（截图对应）

| Web 路由 | 页面 | HarmonyOS |
|---|---|---|
| `/` | 首页信息流 | HomePage |
| `/nearby` | 附近枢纽 | NearbyPage |
| `/nearby/people` | 附近的人 | NearbyUsersListPage |
| `/nearby/moments` | 附近瞬间 | MomentsFeedPage |
| `/nearby/groups` | 附近群组 | NearbyGroupsListPage |
| `/messages` | 消息列表 | MessagePage |
| `/notify` | 通知中心 | 通知中心 overlay |
| `/me` | 个人中心 | ProfilePage |
| `/settings` | 设置 | SettingsPage |
| `/search` | 搜索 | SearchPage |
| `/me/edit` | 编辑资料 | ProfileEditPage |
| `/me/relationships` | 关系 | RelationshipHubPage |
| `/me/moments` | 我的瞬间 | MyMomentsPage |
| `/me/groups` | 我的群组 | MyGroupsPage |
| `/me/reports` | 我的举报 | MyReportsPage |
| `/map` | 地图选点 | LocationPickerPage |
| `/groups/:id` | 群详情 | GroupDetailPage |
| `/moments/:id` | 瞬间详情 | MomentDetailPage |
| `/users/:id` | 他人主页 | ProfilePage（external） |
| `/admin` | 管理后台 | AdminOverviewPage |

## 维护约定

- `pages/` 截图是**参考**，像素级以 token 与代码为准；截图过期时重跑 `capture-pages.sh`。
- 新增主题/语义色：先改代码 token 与校验脚本，再重新生成本包。
- 本包不存放设计决策讨论过程；决策以代码注释与 `docs/design-system.md` 为准。

