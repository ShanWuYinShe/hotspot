import { describe, expect, it } from 'vitest';
import { toNearbyRailRows } from '../nearby-rail';
import type { MomentItem } from '@/api/types';

// 回归：同作者多条附近动态曾以 user_id 作为 each key 直渲染，
// 触发 Svelte each_key_duplicate 崩溃（moments/[momentId] 右 rail）。

function moment(partial: Partial<MomentItem> & Pick<MomentItem, 'moment_id'>): MomentItem {
	return {
		description: '',
		images: [],
		tags: [],
		mentions: [],
		created_at: '',
		comments_count: 0,
		shares_count: 0,
		reactions: [],
		user: { user_id: '', nickname: '', avatar_url: '' },
		location: null,
		is_favorited: false,
		recommendation: null,
		is_deleted: false,
		...partial
	};
}

describe('toNearbyRailRows', () => {
	it('同一作者的多条动态只保留一条（each key 唯一）', () => {
		const rows = toNearbyRailRows([
			moment({ moment_id: 'moment_1', user: { user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: 'a' } }),
			moment({ moment_id: 'moment_2', user: { user_id: 'user_sophia', nickname: '阿禾', avatar_url: 'b' } }),
			moment({ moment_id: 'moment_8', user: { user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: 'a' } })
		]);

		expect(rows.map((r) => r.id)).toEqual(['user_lumos', 'user_sophia']);
	});

	it('保留首次出现的作者信息，丢弃后续重复项', () => {
		const rows = toNearbyRailRows([
			moment({ moment_id: 'moment_1', user: { user_id: 'u1', nickname: '首个', avatar_url: 'a1' }, location: { point: { latitude: 0, longitude: 0 }, location_name: 'X', distance: 1200, bearing_degrees: 0 } }),
			moment({ moment_id: 'moment_2', user: { user_id: 'u1', nickname: '应被忽略', avatar_url: 'a2' } })
		]);

		expect(rows).toHaveLength(1);
		expect(rows[0].name).toBe('首个');
		expect(rows[0].sub).toContain('1.2km');
	});

	it('user 缺失时回退 moment_id 作为行身份', () => {
		const rows = toNearbyRailRows([
			moment({ moment_id: 'moment_1' }),
			moment({ moment_id: 'moment_2' })
		]);

		expect(rows.map((r) => r.id)).toEqual(['moment_1', 'moment_2']);
		expect(rows.every((r) => r.name === '')).toBe(true);
	});

	it('计算出的身份相同（含回退）时同样折叠，保证 key 唯一', () => {
		const rows = toNearbyRailRows([
			moment({ moment_id: 'moment_1', user: { user_id: 'u1', nickname: '甲', avatar_url: '' } }),
			moment({ moment_id: 'u1' })
		]);

		expect(rows).toHaveLength(1);
	});

	it('空输入返回空数组', () => {
		expect(toNearbyRailRows([])).toEqual([]);
	});
});
