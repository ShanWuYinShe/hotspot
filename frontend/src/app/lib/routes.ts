/** Canonical application URLs mirror backend domains and resource names. */
export type ProfileSection = "moments" | "groups" | "relationships" | "friends" | "following" | "followers";

export const ROUTES = {
  home: "/",
  discover: "/discover",
  messages: "/messages",
  me: "/me",
  meEdit: "/me/edit",
  settings: "/settings",
  map: "/map",
  notifications: "/notifications",
  nearby: "/nearby",
  nearbyPeople: "/nearby/people",
  nearbyGroups: "/nearby/groups",
  nearbyMoments: "/nearby/moments",
  search: "/search",
  user: (userId: string | number) => `/users/${userId}`,
  moment: (momentId: string | number) => `/moments/${momentId}`,
  group: (groupId: string | number) => `/groups/${groupId}`,
  message: (chatId: string | number) => `/messages/${chatId}`,
  meSection: (section: ProfileSection) => `/me/${section}`,
} as const;

export const SUB_PAGE_PATHS = [
  ROUTES.settings,
  ROUTES.map,
  ROUTES.notifications,
  ROUTES.meEdit,
  ROUTES.search,
  ROUTES.nearby,
  ROUTES.nearbyPeople,
  ROUTES.nearbyGroups,
  ROUTES.nearbyMoments,
] as const;

/** Paths that replace bottom navigation with a focused detail experience. */
export function isSubPagePath(pathname: string) {
  return (
    SUB_PAGE_PATHS.some((path) => pathname === path) ||
    pathname.startsWith(`${ROUTES.messages}/`) ||
    pathname.startsWith(`${ROUTES.user("").replace(/\/$/, "")}/`) ||
    pathname.startsWith(`${ROUTES.moment("").replace(/\/$/, "")}/`) ||
    pathname.startsWith(`${ROUTES.group("").replace(/\/$/, "")}/`) ||
    pathname.startsWith(`${ROUTES.me}/`)
  );
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
  | "home" | "nearby-people" | "nearby-groups" | "nearby-activity"
  | "messages" | "notifications" | "moments" | "my-groups"
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
  if (pathname === ROUTES.nearbyPeople || pathname === ROUTES.nearby) return "nearby-people";
  if (pathname === ROUTES.discover || pathname === ROUTES.nearbyMoments) return "nearby-activity";

  if (pathname.startsWith(`${ROUTES.me}/`)) {
    const section = pathname.slice(`${ROUTES.me}/`.length);
    if (section === "moments") return "moments";
    if (section === "groups") return "my-groups";
    if (section === "relationships" || section === "friends" || section === "following" || section === "followers") return "relationships";
  }

  if (pathname.startsWith(`${ROUTES.moment("").replace(/\/$/, "")}/`) && source === "me") return "moments";
  if (pathname.startsWith(`${ROUTES.group("").replace(/\/$/, "")}/`) && source === "me") return "my-groups";
  if (pathname.startsWith(`${ROUTES.user("").replace(/\/$/, "")}/`) && source === "relationships") return "relationships";

  return null;
}
