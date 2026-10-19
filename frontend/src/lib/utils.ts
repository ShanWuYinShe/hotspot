import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * 距离展示格式化。spec RelativeLocation.distance 恒为 number | null（米），
 * null/undefined 表示无参考点或不可见，返回空串。
 * 数字统一四舍五入到 1 位小数，消除浮点伪影（如 0.4 * 3 → 1.2000000000000002）；
 * spec 整数米数取整后原样输出。
 */
export function formatDistance(d?: number | null): string {
  if (d == null) return '';
  if (d >= 1000) return `${(d / 1000).toFixed(1)}km`;
  return `${Math.round(d)}m`;
}

/**
 * 日期时间展示格式化（动态详情/评论等完整时间场景）。
 * 输入 ISO 字符串，输出 "2026-08-05 17:20"；非法/空输入返回空串。
 */
export function formatDateTime(iso?: string | null): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** 数字缩写（审计 P1-8）：社交计数 ≥1万 缩写为 x.x万，其余原样。全站唯一实现。 */
export function formatCount(n: number): string {
	if (n >= 10000) {
		const v = n / 10000;
		const s = v >= 100 ? Math.round(v).toString() : v.toFixed(1).replace(/\.0$/, '');
		return `${s}万`;
	}
	return n.toString();
}

/** 相对日文案（审计 P1-7）：「今天/N天前/N周前」。列表语境相对时间的统一实现。 */
export function formatRelativeDays(daysAgo: number): string {
	if (daysAgo <= 1) return '今天';
	if (daysAgo <= 7) return `${daysAgo}天前`;
	return `${Math.floor(daysAgo / 7)}周前`;
}

/**
 * 详情页相对时间（动态详情作者行/评论行，Gemini UI 审查：完整时间戳冗长）：
 * 今天→HH:mm / 昨天→"昨天 HH:mm" / 同年→M月D日 HH:mm / 跨年→YYYY年M月D日。
 * 对齐鸿蒙 FormatUtil.formatDetailTime（跨端同步：文案与粒度逐字一致）。
 */
/**
 * 卡片时间（网格卡片作者行用）：宽卡片才承受得起完整时间戳。
 * 今天→HH:mm / 昨天→"昨天" / 同年→"M月D日" / 跨年→"YYYY年M月D日"。
 * 详情页仍用 formatMomentTime（带 HH:mm，信息更全）。
 */
export function formatCardTime(iso?: string | null): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (isNaN(d.getTime())) return '';
	const now = new Date();
	const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
	const dayDiff = Math.round((startOfDay(now) - startOfDay(d)) / 86400000);
	if (dayDiff === 0) return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
	if (dayDiff === 1) return '昨天';
	const monthDay = `${d.getMonth() + 1}月${d.getDate()}日`;
	if (d.getFullYear() === now.getFullYear()) return monthDay;
	return `${d.getFullYear()}年${monthDay}`;
}

export function formatMomentTime(iso?: string | null): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (isNaN(d.getTime())) return '';
	const now = new Date();
	const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
	const dayDiff = Math.round((startOfDay(now) - startOfDay(d)) / 86400000);
	const hhMm = `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
	if (dayDiff === 0) return hhMm;
	const monthDay = `${d.getMonth() + 1}月${d.getDate()}日`;
	if (dayDiff === 1) return `昨天 ${hhMm}`;
	if (d.getFullYear() === now.getFullYear()) return `${monthDay} ${hhMm}`;
	return `${d.getFullYear()}年${monthDay}`;
}
