// 平台管理表格样式常量（token 类，桌面为主；小屏 overflow-x-auto 由 AdminTable 外壳承担）。
// 页面私有：仅 /admin 路由使用，放路由 shared/ 目录（对齐项目「多组件路由拆 shared/」约定）。
//
// R13 视觉改版（模块 F）：
// - 表头保留功能线（§19.5 登记例外），背景降为半档 subtle 让表格更像「数据区」而非卡片；
// - 行高与行 hover 维持，数字列统一右对齐 + tabular-nums（数字是后台的节奏锚点：母题 A）；
// - 行内动作按钮统一 28px 高（视觉） + 44px 热区（hit-slop-44），次级软填 / 危险 soft danger。

/** 表格外壳：横向滚动容器（Round 4 软岛：去 1px 描边，保留圆角/表头/行线等表格功能线） */
export const TABLE_WRAP = 'overflow-x-auto rounded-lg bg-surface-island shadow-xs';

/** <table> 本体 */
export const TABLE = 'w-full text-sm border-collapse';

/** 表头基础样式（对齐方向由 TH / TH_NUM 追加，避免 text-left 与 text-right 同元素并存、
 *  CSS 层叠顺序决定胜负的不可预期结果） */
const TH_BASE = 'px-4 py-2.5 text-xs font-medium text-text-secondary whitespace-nowrap border-b border-border-default bg-surface-subtle';

/** 表头单元格 */
export const TH = `${TH_BASE} text-left`;

/** 数字列表头（右对齐，与 TD_NUM 成对使用） */
export const TH_NUM = `${TH_BASE} text-right`;

/** 行（hover 走整档 --cst-interactive-hover；/50 属透明度越档，见 AGENTS.md 交互状态契约） */
export const TR = 'border-b border-border-default last:border-b-0 hover:bg-interactive-hover transition-colors';

/** 单元格（py-3 = 行高 ~48px，r59：信息拥挤单行间距偏紧；全后台表格统一） */
export const TD = 'px-4 py-3 align-middle text-text-primary whitespace-nowrap';

/** 数字单元格（右对齐 + 等宽数字：多列表格扫读时列位稳定） */
export const TD_NUM = `${TD} text-right tabular-nums`;

/** 操作列容器 */
export const TD_ACTIONS = `${TD} text-right`;

/** 单元格内的次级信息（ID / 次级元信息），避免每页手写同一串类名 */
export const META = 'text-xs text-text-secondary';

/** 单元格内的等宽次级信息（用户 ID / 群组 ID 等） */
export const META_MONO = `${META} font-mono`;

/** 移动端低优先级列（<sm=600px 隐藏，恢复点与 platform 移动/平板阈值同刻度）：
 *  仅保留「主体 / 关键状态 / 操作」三类列，其余细节列手机上藏进横向滚动外的空间，
 *  免去 7-8 列表格在 390px 下的反复横扫；平板（≥sm）起整表恢复。 */
export const TH_SECONDARY = `${TH} max-sm:hidden`;

/** 同 TH_SECONDARY 的数据单元格 */
export const TD_SECONDARY = `${TD} max-sm:hidden`;

/** 移动端隐藏的数字列表头（与 TD_SECONDARY_NUM 成对） */
export const TH_SECONDARY_NUM = `${TH_BASE} text-right max-sm:hidden`;

/** 移动端隐藏的数字单元格 */
export const TD_SECONDARY_NUM = `${TD} text-right tabular-nums max-sm:hidden`;

// 行内动作说明（R13）：ui/button 已提供 soft（中性软填）变体，但后台表格行内动作需要
// 28px 紧凑高度 + hit-slop-44 热区 + 同构的 soft-danger，故保留本文件的行内按钮常量；
// 页面级动作（新建/发布/授予/裁决）统一使用 ui/button。

/** 行内操作按钮（次级安静：软填底、无描边；press 缩放与 focus-ring 保留） */
export const ROW_BTN = 'hover-btn hit-slop-44 tap-92 whitespace-nowrap shrink-0 rounded-md bg-surface-subtle/60 px-3 py-1.5 text-xs font-medium text-text-primary focus-ring transition-colors hover:bg-surface-subtle active:scale-95 disabled:opacity-faint disabled:pointer-events-none';

/** 行内危险操作按钮（soft danger：无描边；保留 danger 语义与二次确认） */
export const ROW_BTN_DANGER = 'hover-btn hit-slop-44 tap-92 whitespace-nowrap shrink-0 rounded-md bg-danger-bg/50 px-3 py-1.5 text-xs font-medium text-text-status-danger-on-soft focus-ring transition-colors hover:bg-danger-bg active:scale-95 disabled:opacity-faint disabled:pointer-events-none';

// ── 页面工具栏（搜索 / 筛选 / 页面级动作）─────────────────────────────
// 9 个子页统一：控件行 flex-wrap，页面级动作靠右（移动端独占一行贴右），
// 与下方表格间距由页面根容器 gap-4 提供（不在工具栏上叠 mb）。

/** 工具栏单行（搜索框 + 下拉筛选） */
export const TOOLBAR = 'flex flex-wrap items-center gap-2.5';

/** 搜索框容器（桌面限宽 320，移动占满一行） */
export const SEARCH_WRAP = 'w-full max-w-80';

/** 下拉筛选控件（与 SearchInput 同圆角/同高度） */
export const SELECT = 'input-surface shrink-0 rounded-md px-3 py-2 text-xs text-text-primary';
