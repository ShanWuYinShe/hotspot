import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

/**
 * 「附近动态」与「发现」内容重复（同为附近动态 feed）。已收敛到 `/discover`。
 * 此处仅保留重定向以兼容旧链接 / 书签；与 `friends → /relations`、`dashboard → /`
 * 的遗留路由收敛策略一致（见 frontend/CLAUDE.md「路由收敛」）。
 */
export const load: PageLoad = () => {
  redirect(307, '/discover');
};
