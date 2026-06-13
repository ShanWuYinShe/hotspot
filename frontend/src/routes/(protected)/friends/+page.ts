import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

/**
 * `/friends` `/follows` `/followers` 已合并为统一的「关系」页（`/relations`，内含好友/关注/粉丝分页）。
 * 此处仅保留重定向以兼容旧链接/书签；与 `dashboard → /` 的遗留路由收敛策略一致。
 */
export const load: PageLoad = () => {
  redirect(307, '/relations?tab=friends');
};
