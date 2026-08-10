/**
 * 统一格式化工具
 * 用于数字、时间等展示格式的统一处理
 */

/**
 * 格式化计数（用于粉丝数、点赞数、成员数等）
 * 1000 → 1k, 10000 → 1w, 100000 → 10w
 */
export function formatCount(count: number | undefined): string {
  if (count === undefined || count === null) return '0';
  if (count < 0) return '0';
  if (count < 10000) return count.toString();
  if (count < 100000000) {
    const wan = count / 10000;
    return wan >= 10 ? `${Math.floor(wan)}w` : `${wan.toFixed(1)}w`;
  }
  const yi = count / 100000000;
  return `${yi.toFixed(1)}亿`;
}

/**
 * 格式化相对时间（用于动态、消息等时间显示）
 * - < 1min: "刚刚"
 * - < 1h: "N分钟前"
 * - < 1d: "N小时前"
 * - < 7d: "N天前"
 * - 同年: "MM-DD"
 * - 跨年: "YYYY-MM-DD"
 */
export function formatRelativeTime(dateTime: string | undefined): string {
  if (!dateTime) return '';
  const createdAt = new Date(dateTime).getTime();
  if (Number.isNaN(createdAt)) return '';

  const now = Date.now();
  const diffMs = now - createdAt;
  const minute = 60_000;
  const hour = 3_600_000;
  const day = 86_400_000;

  if (diffMs < minute) return '刚刚';
  if (diffMs < hour) {
    return `${Math.floor(diffMs / minute)}分钟前`;
  }
  if (diffMs < day) {
    return `${Math.floor(diffMs / hour)}小时前`;
  }
  if (diffMs < 7 * day) {
    return `${Math.floor(diffMs / day)}天前`;
  }

  const date = new Date(createdAt);
  const nowDate = new Date(now);
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const dayStr = date.getDate().toString().padStart(2, '0');

  if (date.getFullYear() === nowDate.getFullYear()) {
    return `${month}-${dayStr}`;
  }
  return `${date.getFullYear()}-${month}-${dayStr}`;
}

/**
 * 聊天日期分组胶囊文案（对齐前端 groupMessagesByDay / formatDateLabel）：
 * 今天 / 昨天 / M月D日（跨年带年份）。
 */
export function formatDateLabel(iso: string): string {
  if (!iso) return '今天';
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return '今天';
  const now = new Date();
  const startOfDay = (x: Date): number => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const dayDiff = Math.round((startOfDay(now) - startOfDay(date)) / 86400000);
  if (dayDiff === 0) return '今天';
  if (dayDiff === 1) return '昨天';
  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1}月${date.getDate()}日`;
  }
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`;
}
