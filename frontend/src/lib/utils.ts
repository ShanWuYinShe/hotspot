import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * 距离展示格式化。spec RelativeLocation.distance 为 number（米），后端统一返回米；
 * mock 数据偶用字符串（如 "0.4km"），此处兼容两种形态避免 "0.4kmm" 类拼接错误。
 */
export function formatDistance(d?: number | string | null): string {
  if (d == null) return '';
  return typeof d === 'number' ? `${d}m` : d;
}
