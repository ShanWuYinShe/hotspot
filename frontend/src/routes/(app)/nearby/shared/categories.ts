/**
 * 附近域统一分类词库（people 兴趣筛选与 groups 分类筛选共用；Gemini 审查 r03/r04：
 * 两页词库割裂——people 原「运动/艺术」在标签池中无数据，选中必空结果）。
 * 取值对齐 mocks USER_TAGS 标签池（单一事实源）；线上环境分类以后端 tags 为准，
 * 此常量仅作筛选项候选，空结果由空态表达。
 */
export const NEARBY_CATEGORIES = [
	'全部',
	'摄影',
	'户外',
	'旅行',
	'音乐',
	'美食',
	'阅读',
	'健身'
] as const;
