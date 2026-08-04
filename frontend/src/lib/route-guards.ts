import { error } from '@sveltejs/kit';

/** 个人主页分区白名单（与 lib/routes.ts 的 ProfileSection 保持一致）。 */
const PROFILE_SECTIONS = new Set(['moments', 'groups', 'friends', 'following', 'fans', 'relationships']);

export function requireProfileSection(value: string): string {
	if (!PROFILE_SECTIONS.has(value)) {
		throw error(404, '未知的个人主页分区');
	}
	return value;
}
