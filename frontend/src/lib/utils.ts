import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * 距离展示格式化。spec RelativeLocation.distance 为 number（米），后端统一返回米；
 * 距离可能为数字或字符串（如 "0.4km"），统一处理避免 "0.4kmm" 类拼接错误。
 * 数字统一四舍五入到 1 位小数，消除浮点伪影（如 0.4 * 3 → 1.2000000000000002）；
 * spec 整数米数取整后原样输出。
 */
export function formatDistance(d?: number | string | null): string {
  if (d == null) return '';
  // number 为米：≥1km 转 km（1 位小数），否则原样米；字符串（如 '0.4km'）直接返回
  if (typeof d === 'number') {
    if (d >= 1000) return `${(d / 1000).toFixed(1)}km`;
    return `${Math.round(d)}m`;
  }
  return d;
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
