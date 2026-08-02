import { error } from '@sveltejs/kit';

/**
 * 数字型 ID 路由守卫：要求路径参数为纯数字格式（后端 ID 均为数字）。
 * 供各详情页 +page.ts 的 universal load 使用——SPA 模式下参数变化时 load 会重新执行，
 * 非法 ID（如 /moments/abc、/users/-1）直接进入 404，而不是带着 NaN 发请求。
 */
export function requireNumericId(value: string): string {
	if (!/^\d+$/.test(value)) {
		throw error(404, '无效的 ID');
	}
	return value;
}

/** 个人主页分区白名单（与 lib/routes.ts 的 ProfileSection 保持一致）。 */
const PROFILE_SECTIONS = new Set(['moments', 'groups', 'relationships']);

export function requireProfileSection(value: string): string {
	if (!PROFILE_SECTIONS.has(value)) {
		throw error(404, '未知的个人主页分区');
	}
	return value;
}
