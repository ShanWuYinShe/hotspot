// people 列表进度（会话级）：已加载集合 + 渲染切片。组件卸载（进详情再
// 返回）后仍保持，回退即同步渲染、不重拉；定位 key 变化或手动重试时重置。
// 排序/标签筛选是客户端派生，不影响进度。
import type { NearbyUser } from './UserCard.svelte';

export interface PeopleListProgress {
	key: string;
	users: NearbyUser[];
	visibleCount: number;
}

let progress = $state<PeopleListProgress | null>(null);

export function getPeopleProgress(): PeopleListProgress | null {
	return progress;
}

export function setPeopleProgress(next: PeopleListProgress): void {
	progress = next;
}

export function clearPeopleProgress(): void {
	progress = null;
}

/** 定位 key（~100m 网格）：抖动不重拉，城市级变化才重置进度。 */
export function peopleProgressKey(point: { latitude: number; longitude: number } | null): string {
	if (!point) return '';
	return `${point.latitude.toFixed(3)},${point.longitude.toFixed(3)}`;
}
