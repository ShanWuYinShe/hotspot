import type { MomentItem } from '@/api/types';
import { formatDistance } from '@/lib/utils';

// 「此刻附近」rail 行（RailUserRow 人物行）。

export interface NearbyRailRow {
	id: string;
	name: string;
	sub: string;
	avatar: string;
}

/**
 * MomentItem[] → 「此刻附近」人物行。
 *
 * 同一作者的多条附近动态只保留首条：该 rail 是人物列表（「查看全部」指向
 * 附近的人），且 {#each} 以 user_id 为 key——一位作者可同时命中多条动态，
 * 不去重会产生重复键并触发 Svelte each_key_duplicate 运行时崩溃。
 * user 缺失时回退 moment_id 作为身份（每行动态本身，id 天然唯一）。
 */
export function toNearbyRailRows(items: MomentItem[]): NearbyRailRow[] {
	const seen = new Set<string>();
	const rows: NearbyRailRow[] = [];
	for (const m of items) {
		const id = m.user?.user_id || m.moment_id;
		if (seen.has(id)) continue;
		seen.add(id);
		rows.push({
			id,
			name: m.user?.nickname ?? '',
			sub: [formatDistance(m.location?.distance), m.tags?.[0]].filter(Boolean).join(' · '),
			avatar: m.user?.avatar_url ?? ''
		});
	}
	return rows;
}
