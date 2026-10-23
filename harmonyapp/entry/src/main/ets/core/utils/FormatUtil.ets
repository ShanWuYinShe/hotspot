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
 * 格式化相对时间（用于瞬间、消息等时间显示）
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

/**
 * 日期：YYYY-MM-DD（入群日期等只显示日期的场景）。
 */
export function formatDate(dateTime: string | undefined): string {
  if (!dateTime) return '';
  const date = new Date(dateTime);
  if (Number.isNaN(date.getTime())) return '';
  const month = (date.getMonth() + 1).toString().padStart(2, '0');
  const day = date.getDate().toString().padStart(2, '0');
  return `${date.getFullYear()}-${month}-${day}`;
}

/**
 * 详情页时间：YYYY-MM-DD HH:mm。此前 toLocale 输出美式 8/31/2026（ MomentDetailPage
 * 踩坑记录），各页私有实现统一收敛到此。
 */
export function formatDateTime(dateTime: string | undefined): string {
  if (!dateTime) return '';
  const date = new Date(dateTime);
  if (Number.isNaN(date.getTime())) return '';
  return `${formatDate(dateTime)} ${formatHhMm(date)}`;
}

/**
 * 详情页相对时间（瞬间详情作者行/评论行，Gemini UI 审查：完整时间戳冗长）：
 * 今天→HH:mm / 昨天→"昨天 HH:mm" / 同年→M月D日 HH:mm / 跨年→YYYY年M月D日。
 * 对齐前端 lib/utils formatMomentTime（跨端同步：文案与粒度逐字一致）。
 */
export function formatDetailTime(dateTime: string | undefined): string {
  if (!dateTime) return '';
  const date = new Date(dateTime);
  if (Number.isNaN(date.getTime())) return '';
  const now = new Date();
  const startOfDay = (x: Date): number => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const dayDiff = Math.round((startOfDay(now) - startOfDay(date)) / 86400000);
  const hhMm = formatHhMm(date);
  if (dayDiff === 0) return hhMm;
  const monthDay = `${date.getMonth() + 1}月${date.getDate()}日`;
  if (dayDiff === 1) return `昨天 ${hhMm}`;
  if (date.getFullYear() === now.getFullYear()) return `${monthDay} ${hhMm}`;
  return `${date.getFullYear()}年${monthDay}`;
}

/**
 * 卡片时间（会话/动态卡片）：今天→HH:mm / 昨天→"昨天" / 同年→M月D日 /
 * 跨年→YYYY年M月D日（2026-10-20 起与 web `formatCardTime` 逐字一致）。
 * 统一 MomentFeedCard 与 MessagePage 逐字重复的手写实现，
 * 跨年分支不再走 toLocaleDateString（美式日期回归）。
 */
export function formatCardTime(iso: string): string {
  if (!iso) return '';
  const time = new Date(iso);
  if (!Number.isFinite(time.getTime())) return '';
  const now = new Date();
  if (time.toDateString() === now.toDateString()) {
    return formatHhMm(time);
  }
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  if (time.toDateString() === yesterday.toDateString()) {
    return '昨天';
  }
  // 与 web `formatCardTime` 逐字对齐（2026-10-20 跨端一致性）：同年 M月D日、跨年 YYYY年M月D日
  const monthDay = `${time.getMonth() + 1}月${time.getDate()}日`;
  if (time.getFullYear() === now.getFullYear()) {
    return monthDay;
  }
  return `${time.getFullYear()}年${monthDay}`;
}

function formatHhMm(date: Date): string {
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
}

/**
 * 清洗纯文本内容（用于瞬间描述、评论等展示）：
 * 剥除 HTML 标签、转义常见 HTML 实体、规整首尾多余空白，
 * 防止未脱敏内容或抓取文本在端上直接暴露 <b>、<div> 等裸标签。
 */
export function cleanPlainText(text: string | undefined | null): string {
  if (!text) return '';
  let cleaned = text.replace(/<[^>]+>/g, '');
  cleaned = cleaned
    .replace(/&nbsp;/g, ' ')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&apos;/g, "'");
  return cleaned.trim();
}

