import { redirect } from '@sveltejs/kit';
import { ROUTES } from '@/lib/routes';

// 旧路径 → 首页的 308 重定向。附近聚合门户（/nearby）与首页渲染同一份
// 附近数据（人/群组/动态三个板块，入口一一对应），属页面间重复功能；
// 收敛为首页单一聚合后保留重定向，旧书签/外链仍可到达，不会落到 404
// （对齐 /me/friends → /me/relationships 的收敛模式）。
export function load() {
	redirect(308, ROUTES.home);
}
