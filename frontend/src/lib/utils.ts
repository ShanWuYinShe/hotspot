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
  return typeof d === 'number' ? `${Math.round(d * 10) / 10}m` : d;
}
