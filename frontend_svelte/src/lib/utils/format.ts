/**
 * 格式化工具函数
 */

/** 校验日期字符串是否有效 */
function isValidDate(dateStr: string): boolean {
	if (!dateStr) return false;
	const ts = new Date(dateStr).getTime();
	return !isNaN(ts);
}

/**
 * 格式化时间为相对时间（刚刚、X分钟前、X小时前等）
 */
export function formatRelativeTime(dateStr: string): string {
	if (!isValidDate(dateStr)) return '—';

	const date = new Date(dateStr);
	const now = new Date();
	const diff = now.getTime() - date.getTime();
	const minutes = Math.floor(diff / 60000);
	const hours = Math.floor(diff / 3600000);
	const days = Math.floor(diff / 86400000);

	if (minutes < 1) return '刚刚';
	if (minutes < 60) return `${minutes}分钟前`;
	if (hours < 24) return `${hours}小时前`;
	if (days < 7) return `${days}天前`;
	return formatDateTime(dateStr);
}

/**
 * 格式化时间为完整日期时间
 */
export function formatDateTime(dateStr: string): string {
	if (!isValidDate(dateStr)) return '—';
	const date = new Date(dateStr);
	return date.toLocaleString('zh-CN', { hour12: false });
}

/**
 * 格式化时间为日期（不含时间）
 */
export function formatDate(dateStr: string): string {
	if (!isValidDate(dateStr)) return '—';
	return new Date(dateStr).toLocaleDateString('zh-CN');
}

/**
 * 格式化距离（米 -> 米/公里）
 */
export function formatDistance(meters: number): string {
	if (!isFinite(meters) || meters < 0) return '—';
	if (meters < 1000) return `${Math.round(meters)}m`;
	return `${(meters / 1000).toFixed(1)}km`;
}

/**
 * 解析用户输入的标签字符串（支持逗号、中文逗号、空格分隔，自动去除 # 前缀）
 */
export function parseTags(input: string, maxCount = 10): string[] {
	return input
		.split(/[,，\s]+/)
		.map((tag) => tag.replace(/^#/, '').trim())
		.filter(Boolean)
		.slice(0, maxCount);
}
