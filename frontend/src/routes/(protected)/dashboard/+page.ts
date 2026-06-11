import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

/**
 * /dashboard 与首页 `/` 内容完全重复（迁移自 Figma/React 的遗留路由，且无任何入口引用）。
 * 统一以 `/` 作为「附近/首页」的唯一路由，此处仅保留重定向以兼容旧链接/书签。
 */
export const load: PageLoad = () => {
  redirect(307, '/');
};
