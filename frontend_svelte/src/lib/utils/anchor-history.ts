/**
 * 锚点历史记录工具
 * 管理浏览锚点的使用历史，支持去重、限长、持久化（localStorage）
 */

import { browser } from '$app/environment';

const STORAGE_KEY = 'hotspot_anchor_history';
const MAX_ITEMS = 10;

export interface AnchorHistoryItem {
	latitude: number;
	longitude: number;
	locationName: string;
	timestamp: number;
}

/** 获取历史记录 */
export function getAnchorHistory(): AnchorHistoryItem[] {
	if (!browser) return [];
	try {
		const json = localStorage.getItem(STORAGE_KEY);
		if (!json) return [];
		const parsed = JSON.parse(json);
		if (!Array.isArray(parsed)) return [];
		return parsed
			.map((item) => {
				if (!item || typeof item !== 'object') return null;
				const { latitude, longitude, locationName, timestamp } = item as Partial<AnchorHistoryItem>;
				if (
					typeof latitude !== 'number' ||
					typeof longitude !== 'number' ||
					typeof timestamp !== 'number'
				) {
					return null;
				}
				return {
					latitude,
					longitude,
					locationName: typeof locationName === 'string' && locationName.trim()
						? locationName.trim()
						: '自定义位置',
					timestamp
				} satisfies AnchorHistoryItem;
			})
			.filter((item): item is AnchorHistoryItem => item !== null)
			.slice(0, MAX_ITEMS);
	} catch {
		// 忽略解析错误
	}
	return [];
}

/** 添加一条历史（去重 + 限长） */
export function addToAnchorHistory(latitude: number, longitude: number, locationName: string): void {
	if (!browser) return;
	try {
		const history = getAnchorHistory();

		// 去重：经纬度差距 < 0.001（约 100m）视为同一点
		const existingIndex = history.findIndex(
			(item) =>
				Math.abs(item.latitude - latitude) < 0.001 &&
				Math.abs(item.longitude - longitude) < 0.001
		);

		const newItem: AnchorHistoryItem = {
			latitude,
			longitude,
			locationName: locationName || '自定义位置',
			timestamp: Date.now()
		};

		if (existingIndex >= 0) {
			history.splice(existingIndex, 1);
		}

		history.unshift(newItem);

		if (history.length > MAX_ITEMS) {
			history.splice(MAX_ITEMS);
		}

		localStorage.setItem(STORAGE_KEY, JSON.stringify(history));
	} catch {
		// 忽略存储错误
	}
}

/** 清空历史 */
export function clearAnchorHistory(): void {
	if (!browser) return;
	try {
		localStorage.removeItem(STORAGE_KEY);
	} catch {
		// 忽略
	}
}
