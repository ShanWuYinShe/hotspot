/**
 * 子页面判定（单一事实源）。
 *
 * 子页面在移动 / 平板为全屏，自带返回（back）头部，因此隐藏底部主导航。
 * 底部主标签目的地（/、/discover、/chat、/profile）及其它保留底栏的页面**不**是子页面。
 *
 * 注意：/users/nearby、/groups/nearby 是顶层「浏览」页，但移动·平板**不在底栏中**，
 * 经首页「查看全部」等处下钻进入，故按子页面处理——全屏、带返回、隐藏底栏（见 SUB_PAGE_EXACT）。
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

/** 前缀匹配的子页面（设置子页、聊天会话、分享）。 */
const SUB_PAGE_PREFIXES = ['/settings/', '/chat/users/', '/chat/groups/', '/share/'];

/**
 * 详情页基路径：/users/<id>、/groups/<id>、/moments/<id>。
 * 排除 `nearby` 段——/users/nearby、/groups/nearby 不是实体详情页，
 * 其子页面判定已由上面的 SUB_PAGE_EXACT 显式处理，这里不重复匹配。
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
