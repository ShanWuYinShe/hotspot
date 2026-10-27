// 平台管理表格样式常量（token 类，桌面为主；小屏 overflow-x-auto 由 AdminTable 外壳承担）。
// 页面私有：仅 /admin 路由使用，放路由 shared/ 目录（对齐项目「多组件路由拆 shared/」约定）。

/** 表格外壳：横向滚动容器（Round 4 软岛：去 1px 描边，保留圆角/表头/行线等表格功能线） */
export const TABLE_WRAP = 'overflow-x-auto rounded-lg bg-surface-island shadow-xs';

/** <table> 本体 */
export const TABLE = 'w-full text-sm border-collapse';

/** 表头单元格 */
export const TH = 'px-4 py-2.5 text-left text-xs font-medium text-text-secondary whitespace-nowrap border-b border-border-default bg-surface-subtle';

/** 行（hover 走整档 --cst-interactive-hover；/50 属透明度越档，见 AGENTS.md 交互状态契约） */
export const TR = 'border-b border-border-default last:border-b-0 hover:bg-interactive-hover transition-colors';

/** 单元格（py-3 = 行高 ~48px，r59：信息拥挤单行间距偏紧；全后台表格统一） */
export const TD = 'px-4 py-3 align-middle text-text-primary whitespace-nowrap';

/** 操作列容器 */
export const TD_ACTIONS = `${TD} text-right`;

/** 移动端低优先级列（<sm=600px 隐藏，恢复点与 platform 移动/平板阈值同刻度）：
 *  仅保留「主体 / 关键状态 / 操作」三类列，其余细节列手机上藏进横向滚动外的空间，
 *  免去 7-8 列表格在 390px 下的反复横扫；平板（≥sm）起整表恢复。 */
export const TH_SECONDARY = `${TH} max-sm:hidden`;

/** 同 TH_SECONDARY 的数据单元格 */
export const TD_SECONDARY = `${TD} max-sm:hidden`;

/** 行内操作按钮（次级安静：软填底、无描边；press 缩放与 focus-ring 保留） */
export const ROW_BTN = 'hover-btn hit-slop tap-92 whitespace-nowrap shrink-0 rounded-md bg-surface-subtle/60 px-2.5 py-1 text-xs font-medium text-text-primary focus-ring transition-colors hover:bg-surface-subtle active:scale-95 disabled:opacity-faint disabled:pointer-events-none';

/** 行内主操作按钮（accent 填充） */
export const ROW_BTN_PRIMARY = 'hover-btn hit-slop tap-92 whitespace-nowrap shrink-0 rounded-md bg-accent-default px-2.5 py-1 text-xs font-medium text-text-on-accent shadow-raised-sm focus-ring transition-colors hover:shadow-raised-md disabled:opacity-faint disabled:pointer-events-none';

/** 行内危险操作按钮（soft danger：无描边；保留 danger 语义与二次确认） */
export const ROW_BTN_DANGER = 'hover-btn hit-slop tap-92 whitespace-nowrap shrink-0 rounded-md bg-danger-bg/50 px-2.5 py-1 text-xs font-medium text-status-danger focus-ring transition-colors hover:bg-danger-bg active:scale-95 disabled:opacity-faint disabled:pointer-events-none';

