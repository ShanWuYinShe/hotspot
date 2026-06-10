/**
 * 子页面判定 — 与 React 版 MainLayout 的 SUB_PAGE_PATHS 对齐。
 *
 * 子页面在移动端为全屏，自带返回（back）头部，因此隐藏底部主导航。
 * 主标签页与顶层浏览页（/、/discover、/chat、/profile、各类 nearby 浏览页等）
 * 没有返回入口，必须保留底部导航，避免用户被困。
 */

/** 精确匹配的子页面（均自带返回头部）。 */
const SUB_PAGE_EXACT = new Set([
  '/settings',
  '/notifications',
  '/profile/edit',
  '/search',
  '/users/nearby',
  '/groups/nearby'
]);

/** 前缀匹配的子页面（设置子页、聊天会话、分享、活动详情）。 */
const SUB_PAGE_PREFIXES = ['/settings/', '/chat/users/', '/chat/groups/', '/events/', '/share/'];

/**
 * 详情页基路径：/users/<id>、/groups/<id>、/moments/<id>。
 * 排除 `nearby` 段——它是顶层浏览页（无返回头部），不应隐藏导航。
 */
const DETAIL_BASES = ['/users', '/groups', '/moments'];

function isDetailPage(pathname: string): boolean {
  for (const base of DETAIL_BASES) {
    if (!pathname.startsWith(`${base}/`)) continue;
    const rest = pathname.slice(base.length + 1);
    if (rest.length > 0 && rest !== 'nearby') return true;
  }
  return false;
}

export function isSubPage(pathname: string): boolean {
  if (SUB_PAGE_EXACT.has(pathname)) return true;
  if (SUB_PAGE_PREFIXES.some((prefix) => pathname.startsWith(prefix))) return true;
  return isDetailPage(pathname);
}
