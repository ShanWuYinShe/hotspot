// moments 列表进度（会话级）：已加载集合 + 服务端游标 + 渲染切片 + 标签。
// 组件卸载（进详情再返回）后仍保持，回退即从已加载集合同步渲染、不从第 1 页
// 重拉；定位 key 变化（城市切换）或手动重试时重置。
// Tab/标签是客户端派生过滤，不影响进度。
import type { FeedItem } from '@/components/moments/moment-feed.svelte';

export interface MomentsFeedProgress {
	savedAt: number;
	key: string;
	items: FeedItem[];
	nextCursor: string;
	visibleCount: number;
	tags: string[];
}

let progress = $state<MomentsFeedProgress | null>(null);

/** 缓存有效期：无刷新入口的会话级缓存不能整场会话保鲜，5 分钟后强制重拉。 */
const PROGRESS_TTL_MS = 5 * 60 * 1000;

export function getMomentsProgress(): MomentsFeedProgress | null {
	if (progress && Date.now() - progress.savedAt > PROGRESS_TTL_MS) {
		progress = null;
	}
	return progress;
}

export function setMomentsProgress(next: MomentsFeedProgress): void {
	progress = next;
}

export function clearMomentsProgress(): void {
	progress = null;
}

/** 定位 key（~100m 网格）：抖动不重拉，城市级变化才重置进度。 */
export function feedProgressKey(point: { latitude: number; longitude: number } | null): string {
	if (!point) return '';
	return `${point.latitude.toFixed(3)},${point.longitude.toFixed(3)}`;
}
