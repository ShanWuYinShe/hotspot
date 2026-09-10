/**
 * Canonical application URLs mirror backend domains and resource names.
 *
 * 个人主页分区共 3 个：moments / groups / relationships（好友 + 关注 + 粉丝三 tab，
 * 内部用 ?tab= 切换，见 RELATIONSHIP_TABS）。全部为常驻固定路由
 * （/me/moments、/me/relationships …），不走动态段；relationships 独立渲染为
 * 样板页（StandardPageLayout，不走 ProfilePage 分区模式）；桌面导航「关系」、
 * 通知页「新关注你的人」与移动端个人数据行均落在 relationships。
 *
 * 历史遗留的 /me/friends、/me/following、/me/fans 三条路由与 relationships 的
 * 三个 tab 内容完全重复（同一 Section 组件、同一数据，只是页头不同），且无任何
 * 入站入口；现改为 308 重定向到 /me/relationships?tab=… 以保留旧链接可达性，
 * 见 lib/relationships.ts。
 */
export type ProfileSection = "moments" | "groups" | "relationships";

/** 个人主页固定分区页路径（常驻路由，非动态段）。 */
export const ME_SECTION_PATHS: Record<ProfileSection, string> = {
  moments: "/me/moments",
  groups: "/me/groups",
  relationships: "/me/relationships",
};

export const ROUTES = {
  home: "/",
  login: "/login",
  messages: "/messages",
  me: "/me",
  meEdit: "/me/edit",
  settings: "/settings",
  map: "/map",
  nearby: "/nearby",
  nearbyPeople: "/nearby/people",
  nearbyGroups: "/nearby/groups",
  nearbyMoments: "/nearby/moments",
	search: "/search",
	notify: "/notify",
  user: (userId: string | number) => `/users/${userId}`,
  moment: (momentId: string | number) => `/moments/${momentId}`,
  group: (groupId: string | number) => `/groups/${groupId}`,
  message: (peerId: string | number) => `/messages/${peerId}`,
  /** 群聊窗口（与私聊 message() 区分;静态段 group 优先于 [peerId] 参数段）。 */
  groupMessage: (groupId: string | number) => `/messages/group/${groupId}`,
} as const;

/**
 * Static path prefixes for detail pages. Using these avoids fragile runtime
 * string surgery (`.replace(/\/$/, "")` on a helper result) when testing
 * whether a pathname lives under a parametrised route.
 */
const DETAIL_PREFIXES = ["/users/", "/moments/", "/groups/"] as const;

export const SUB_PAGE_PATHS = [
  ROUTES.settings,
  ROUTES.map,
  ROUTES.notify,
  ROUTES.search,
  ROUTES.nearbyPeople,
  ROUTES.nearbyGroups,
  ROUTES.nearbyMoments,
] as const;

/** Paths that replace bottom navigation with a focused detail experience. */
export function isSubPagePath(pathname: string) {
  if (SUB_PAGE_PATHS.some((path) => pathname === path)) return true;
  if (pathname.startsWith(`${ROUTES.messages}/`)) return true;
  if (pathname.startsWith(`${ROUTES.me}/`)) return true;
  return DETAIL_PREFIXES.some((prefix) => pathname.startsWith(prefix));
}

/** Consistent active state for the primary mobile/tablet navigation targets. */
export function isPrimaryRouteActive(pathname: string, target: string) {
  // 首页/附近聚合页是顶级 tab：只对自身路径高亮，不包含其子页面。
  if (target === ROUTES.home || target === ROUTES.nearby) return pathname === target;
  return (
    pathname === target ||
    pathname.startsWith(`${target}/`) ||
    (target === ROUTES.me && pathname === ROUTES.settings)
  );
}

export type DesktopNavId =
  | "home" | "nearby-people" | "nearby-groups" | "nearby-moments"
  | "messages" | "moments" | "my-groups"
  | "relationships";

/**
 * The desktop rail only highlights destinations that actually exist in the rail.
 * Detail and utility pages have no active item unless their navigation source is
 * explicit, preventing a misleading unrelated highlight.
 */
export function getDesktopNavId(pathname: string, search = ""): DesktopNavId | null {
  const source = new URLSearchParams(search).get("from");

  if (pathname === ROUTES.home) return "home";
  if (pathname === ROUTES.messages || pathname.startsWith(`${ROUTES.messages}/`)) return "messages";
  if (pathname === ROUTES.nearbyGroups) return "nearby-groups";
  if (pathname === ROUTES.nearbyPeople) return "nearby-people";
  if (pathname === ROUTES.nearbyMoments) return "nearby-moments";

  if (pathname.startsWith(`${ROUTES.me}/`)) {
    const section = pathname.slice(`${ROUTES.me}/`.length);
    if (section === "moments") return "moments";
    if (section === "groups") return "my-groups";
    // 关系聚合页归属「关系」；其余尾段（如 edit）不亮。
    // （旧 friends/following/fans 已重定向，不再是有效 pathname。）
    if (section === "relationships") return "relationships";
    return null;
  }

  if (pathname.startsWith(`${ROUTES.moment("")}`) && source === "me") return "moments";
  if (pathname.startsWith(`${ROUTES.group("")}`) && source === "me") return "my-groups";
  if (pathname.startsWith(`${ROUTES.user("")}`) && source === "relationships") return "relationships";

  return null;
}
