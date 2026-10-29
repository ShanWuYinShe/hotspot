# UI 跨端对齐 Playbook（Web 移动端 ↔ HarmonyOS ↔ 未来 Android）

> 本文回答「怎么做对齐」，与 token 权威（两端 `AGENTS.md` + `check-cross-platform-*.mjs`）互补（`docs/design-system.md` 为已停更的历史快照，仅供考古）。
> 经验来自 harmonyapp 与前端移动端的整轮实战对齐（2026-09），方法论可直接迁移到后续 Android 端。

---

## 1. 核心原则：对齐的是设计，不是实现

前端移动端（390px 视口）是视觉基准；客户端要复现的是**视觉结果**，不是 DOM 结构。
凡是手机平台有原生等价物的，一律用平台方案（见 §4），不要把 CSS 的写法硬翻过来。

```
对齐目标 = token 值（色/字/距/角） + 组件语义（这层卡片是什么） + 交互行为
不对齐   = DOM 嵌套、CSS 实现技巧（伪元素/grid gap 画分割线等）
```

## 2. 三层对齐法（由硬到软，顺序执行）

### 第 1 层：Token 级 diff（最硬，可进 CI）

前端 `themes/*.css` 的 `--cst-palette-*` ↔ 客户端主题常量，**逐值相等**。

- 已有 CI 防线：`bun run --cwd frontend check:theme`（`check-cross-platform-colors.mjs`），
  覆盖 14 组直比映射 + 合成色规则（accent-soft/active 指示器预合成）。
- **透明色陷阱**（防线演进的核心教训）：
  - 前端大量用 `rgba()` / `color-mix()` 表达「accent N% 叠加底色」；客户端常量只能写**预合成不透明 hex**。
    公式：`blend = round(fg×a + bg×(1−a))`，例：accent #2B5EB2 @0.13 over #F7FAFF = `#DCE6F5`。
  - ArkTS 8 位 hex 是 **`#AARRGGBB`**（alpha 前置），与 CSS 的 `#RRGGBBAA` 相反——校验脚本与
    人眼 review 都要先换算通道，否则「看着一样」实际不同（本次抓出 `#00800000` 全透明遮罩 ×3 处）。
- 字号/间距档位：客户端 `DesignConstants` 全档注释必须标注前端对应值
  （`FONT.SIZE.XL: 20 // text-xl`），并执行与前端 theme-registry 相同的「1–2px 近邻归并」原则。

### 第 2 层：规格实测（防「代码看着对」）

**永远不要对着源码猜值**，两边都读真实渲染值：

| 端 | 工具 | 拿什么 |
|---|---|---|
| 前端 | agent-browser `eval getComputedStyle(el)` | 字号/字重/padding/圆角/颜色的最终计算值 |
| 客户端 | `hdc shell uitest dumpLayout -p …` + `file recv` | 组件树 bounds（物理 px） |

- **先确立换算基准**：模拟器 560dpi → 1vp=3.5px，逻辑宽 377vp ≈ 前端 390px，可近似 1:1 对比。
  （AVD 密度在 `~/.Huawei/Emulator/deployed/<name>/config.ini` 的 `hw.lcd.density`。）
- dump 的 Text bounds 是行高框，不等于 fontSize；判断「容器是否贴边/间距是否对」最可靠
  （本方法抓出 hero 卡 `width('100%')` 贴屏幕全宽、边框贴边不可见的真 bug）。
- 前端组件实测要带**平台判定上下文**：`innerWidth≥1280 → desktop`，标题档会从 2xl 变 xl。
  agent-browser 会话重置后视口会丢，**每次先 `set viewport 390 844` 并 `eval innerWidth` 确认**。

### 第 3 层：整页视觉对比（收尾）

- 前端截图（390 视口）按物理宽等比缩放（×3.386 → 1320px），与模拟器截图**同 y 偏移裁剪同区块**逐一对比。
- 视觉层抓的是前两层漏掉的「组合感」差异：节奏、密度、对齐轴。发现差异后必须回到第 1/2 层
  找到具体 token/规格根因再改，禁止「目测调参」。

## 3. 对齐前的环境清单（数据不同源 = 全部白做）

1. **同一后端 + 同一账号**：客户端与前端都连 docker nginx（8888）；前端已无 API mock
   （dev 直连真实后端），两端用同一测试账号登录（`qa@hotspot.local`，见根 `AGENTS.md`
   「开发种子数据」），否则数据不同源。
2. **同一浏览锚点**：前端锚点来自后端 `client_config.defaults.default_location`；
   客户端锚点存 preferences（`haps/entry/preferences/geotalk` 的 `browse_anchor` 键），
   不一致时首页数据完全不同。debug 构建下可直接改文件（`aa force-stop` 后 `file send` 回去）。
3. **客户端必须 debug 构建**：release 走线上域名，现象是「全局网络失败」，极易误诊。
   debug 服务器地址按设备拓扑自动切换：模拟器 `10.0.2.2`（QEMU NAT），
   真机 `127.0.0.1` + `hdc -t <sn> fport tcp:8888 tcp:8888`（USB 反向转发，不依赖 WiFi）。
4. **截屏/操作链路**：`snapshot_display -f` + `file recv` 截图；`uitest uiInput click/swipe/keyEvent`
   操作；`dumpLayout` 拿坐标。注意底部手势区会吞点击（`safeAreaBottom` 上报 0 时按 ≥20vp 兜底）。

## 4. 充分利用系统特性（「像原生」的部分）

对齐前端 ≠ 做成网页套壳。平台有等价物时**必须用平台方案**，理由：主题跟随、无障碍、
系统动效、抗渲染 bug。本项目已落地的取舍：

| 前端实现 | HarmonyOS 采用 | 为什么不用前者 |
|---|---|---|
| lucide SVG + CSS fill 染色 | `SymbolSpan($r('sys.symbol.*'))` 系统符号 | 自研 Image+ColorFilter 曾因矩阵值域（0-255 vs 0-1）全图标变白；系统符号原生染色、随主题/无障碍缩放 |
| div 弹窗 + backdrop-filter | `SystemDialog` + `overlayStrong` 遮罩 | 系统弹窗动效与焦点管理 |
| bottom sheet (Radix Dialog) | 半模态 sheet + `SHEET_TOP_RADIUS:28` | 键盘避让、手势下滑关闭 |
| CSS `env(safe-area-inset)` | `expandSafeArea` + `safeAreaTop/Bottom` | 挖孔屏/手势条避让 |
| :hover | 按压态（scale+springMotion） | 触屏没有 hover |
| media-query 暗色 | 主题管理器随系统切换（前端没有的能力） | 手机用户预期 |

- 系统符号名要**对照 SDK 清单核验**（`sysResource.js` 的 symbol 段），无效名静默渲染空白
  （`music_note`→`music`、sparkles 无对应→`wand_and_stars`）。
- 客户端独有、前端没有的系统能力（触感、字体缩放、无障碍标签）是加分项，按平台惯例加。

### Android 迁移对照（后续建 `androidapp/` 时）

| 概念 | Web | HarmonyOS | Android（计划） |
|---|---|---|---|
| 调色板 | `--cst-palette-*` | `ThemeConstants` ColorConfig | `themes.xml` + `MaterialTheme`(colorScheme) |
| 语义色 | `--cst-text-primary` 等 | palette.textPrimary | `?attr/colorOnSurface` / 自定义 semantic attr |
| 字号档 | Tailwind text-* | `FONT.SIZE`（10/11/12/14/16/20/24/30/36） | `dimens.xml` 同刻度 |
| 底部导航 | MobilePortrait nav + NavTab | Tabs + TabBarBuilder | `BottomNavigationView`/`NavigationBar` |
| 图标 | lucide svelte | SymbolGlyph | Material Symbols（同名系） |
| 半屏发布 | Radix Dialog bottom | sheet 28 圆角 | `ModalBottomSheet` |
| 反向转发 | — | `hdc fport` | `adb reverse tcp:8888 tcp:8888` |
| 布局实测 | computed style | `uitest dumpLayout` | `uiautomator dump` |
| 校验防线 | `check:theme`（已覆盖三主题） | 同左 | 同一脚本加 Android 资源解析分支 |

## 5. 踩坑实录（每条都真实付出过排查成本）

1. **自研 SVG 染色矩阵值域错** → 浅色模式全端图标/文字「隐形」；教训：颜色处理链路越短越好，
   平台原语优先。修复手段是全量切 SymbolGlyph（组件 API 不变，调用点零改动）。
2. **语义 token 误用**：`accentText`（=accent 底上的白字）被当普通强调色用在浅色表面 → 白字白底。
   规则：**token 语义角色按「所在底色」选**，accent 族只出现在 accent 底上。
3. **横滚容器 `Row` 钉 `width('100%')`** → 溢出卡片被裁（4 张群组卡只见 1 张）。
   横滚内容行不设宽，卡片自带固定宽；`Scroll.align(Alignment.Start)` 防内容不足一屏时居中。
4. **组件依赖外部宽度测量**（NearbyImageCard 无显式宽）→ 放进无约束容器后塌缩。
   固定尺寸卡包一层固定宽容器。
5. **release/debug 网络拓扑**：见 §3.3；真机 VPN 网段（/32 掩码）ping 不通宿主是常态，
   fport 才是正解；fport 偶发断连，重启 app 即恢复。
6. **前端访客登录是两步**（切表单→提交），「随便逛逛」按钮不是直接登录。
7. **uitest 点击被手势条吞**：底部按钮热区要兜底安全区。
8. **RegExp 用字符串构造时 `\s` 被 JS 字符串转义吃掉**（写成 `'\\s'`）→ 校验规则静默失效，
   「检查通过」≠「规则在跑」；新防线上线时先注入一个已知错值确认它能红。

## 6. 工作流（建议顺序）

```
0. 环境：docker 起、两端同一测试账号（seed 数据）、锚点一致、debug 构建、视口 390 确认
1. 前端页截图（viewport + full）+ computed style 备查
2. 客户端同页截图 + dumpLayout
3. 差异 → 归因到 token（改 ThemeConstants/DesignConstants，进 CI）或规格（改布局代码）
4. 每修一批：构建 → 部署 → 截图回归 → check:theme 绿
5. 新增的对应关系沉淀进两端 `AGENTS.md` 的映射表与 check 脚本 MAPPING
```
