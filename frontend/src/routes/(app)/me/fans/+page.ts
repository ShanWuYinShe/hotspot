import { redirect } from '@sveltejs/kit';
import { relationshipsHref } from '@/lib/relationships';

// 旧路径 → 关系聚合页对应 tab 的 308 重定向（理由见 me/friends/+page.ts）。
export function load() {
	redirect(308, relationshipsHref('fans'));
}
