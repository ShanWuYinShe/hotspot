/** Canonical application URLs mirror backend domains and resource names. */
export type ProfileSection = "moments" | "groups" | "relationships";

export const ROUTES = {
  home: "/",
  login: "/login",
  messages: "/messages",
  me: "/me",
  meEdit: "/me/edit",
  settings: "/settings",
  map: "/map",
  nearbyPeople: "/nearby/people",
  nearbyGroups: "/nearby/groups",
  nearbyMoments: "/nearby/moments",
  search: "/search",
  notify: "/notify",
  user: (userId: string | number) => `/users/${userId}`,
  moment: (momentId: string | number) => `/moments/${momentId}`,
  group: (groupId: string | number) => `/groups/${groupId}`,
  message: (chatId: string | number) => `/messages/${chatId}`,
  meSection: (section: ProfileSection) => `/me/${section}`,
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
  ROUTES.meEdit,
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
  if (pathname === ROUTES.nearbyPeople) return "nearby-people";
  if (pathname === ROUTES.nearbyMoments) return "nearby-activity";

  if (pathname.startsWith(`${ROUTES.me}/`)) {
    const section = pathname.slice(`${ROUTES.me}/`.length);
    if (section === "moments") return "moments";
    if (section === "groups") return "my-groups";
    // "relationships" is the only other section; treat it (and any /me/ tail) as relationships.
    return "relationships";
  }

  if (pathname.startsWith(`${ROUTES.moment("")}`) && source === "me") return "moments";
  if (pathname.startsWith(`${ROUTES.group("")}`) && source === "me") return "my-groups";
  if (pathname.startsWith(`${ROUTES.user("")}`) && source === "relationships") return "relationships";

  return null;
}
