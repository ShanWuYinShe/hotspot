/**
 * Canonical application URLs mirror backend domains and resource names.
 *
 * 个人主页分区共 6 个：moments / groups / friends / following / fans 五个独立分区，
 * 外加 relationships 聚合分区（好友 + 关注 + 粉丝三 tab，内部用 ?tab= 切换）。
 * 全部为常驻固定路由（/me/moments、/me/relationships …），不走动态段；
 * relationships 独立渲染为样板页（StandardPageLayout，不走 ProfileDesktop
 * 分区模式）；桌面导航「关系」和通知页「新关注你的人」入口落在 relationships；
 * 移动端个人数据 5 行链接走独立分区。路径见 ME_SECTION_PATHS。
 */
export type ProfileSection =
	| "moments"
	| "groups"
	| "friends"
	| "following"
	| "fans"
	| "relationships";

/** 个人主页固定分区页路径（常驻路由，非动态段）。 */
export const ME_SECTION_PATHS: Record<ProfileSection, string> = {
  moments: "/me/moments",
  groups: "/me/groups",
  friends: "/me/friends",
  following: "/me/following",
  fans: "/me/fans",
  relationships: "/me/relationships",
};

export const ROUTES = {
  home: "/",
  login: "/login",
  /** 发现聚合页（平板/移动端顶级导航；聚合附近的人/群组/动态与搜索入口）。 */
  discover: "/discover",
  messages: "/messages",
  me: "/me",
  meEdit: "/me/edit",
  settings: "/settings",
  map: "/map",
  nearbyPeople: "/nearby/people",
  nearbyGroups: "/nearby/groups",
  nearbyMoments: "/nearby/moments",
	search: "/search",
	notifications: "/notifications",
  user: (userId: string | number) => `/users/${userId}`,
  moment: (momentId: string | number) => `/moments/${momentId}`,
  group: (groupId: string | number) => `/groups/${groupId}`,
  event: (eventId: string | number) => `/events/${eventId}`,
  message: (chatId: string | number) => `/messages/${chatId}`,
  /** 群聊窗口（与私聊 message() 区分;静态段 group 优先于 [chatId] 参数段）。 */
  groupMessage: (groupId: string | number) => `/messages/group/${groupId}`,
} as const;

/**
 * Static path prefixes for detail pages. Using these avoids fragile runtime
 * string surgery (`.replace(/\/$/, "")` on a helper result) when testing
 * whether a pathname lives under a parametrised route.
 */
const DETAIL_PREFIXES = ["/users/", "/moments/", "/groups/", "/events/"] as const;

export const SUB_PAGE_PATHS = [
  ROUTES.settings,
  ROUTES.map,
  ROUTES.notifications,
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

/** Consistent active state for the four primary mobile/tablet navigation targets. */
export function isPrimaryRouteActive(pathname: string, target: string) {
  if (target === ROUTES.home) return pathname === ROUTES.home;
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
    // friends/following/fans 及任何 /me/ 尾段都归属「关系」导航项。
    return "relationships";
  }

  if (pathname.startsWith(`${ROUTES.moment("")}`) && source === "me") return "moments";
  if (pathname.startsWith(`${ROUTES.group("")}`) && source === "me") return "my-groups";
  if (pathname.startsWith(`${ROUTES.user("")}`) && source === "relationships") return "relationships";

  return null;
}
