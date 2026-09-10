import { redirect } from '@sveltejs/kit';
import { relationshipsHref } from '@/lib/relationships';

// 旧路径 → 关系聚合页对应 tab 的 308 重定向。
// /me/friends 曾与 /me/relationships?tab=friends 渲染同一 Section、同一数据，
// 只是页头不同，且无任何入站入口；收敛为单一 canonical 路径后保留重定向，
// 旧书签/外链仍可到达，不会落到 404。
export function load() {
	redirect(308, relationshipsHref('friends'));
}
