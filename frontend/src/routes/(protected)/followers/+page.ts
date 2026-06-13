import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

/**
 * `/followers` 已合并入统一的「关系」页（`/relations`，粉丝分页）。
 * 仅保留重定向以兼容旧链接/书签。
 */
export const load: PageLoad = () => {
  redirect(307, '/relations?tab=followers');
};
