import { createBrowserRouter, Navigate, useLocation, useParams } from "react-router";
import { ROUTES, type ProfileSection } from "./lib/routes";
import { MainLayout } from "./components/layout/MainLayout";
import { Home } from "./pages/Home";
import { Discover } from "./pages/Discover";
import { Messages } from "./pages/Messages";
import { ChatDetail } from "./pages/ChatDetail";
import { Profile } from "./pages/Profile";
import { Settings } from "./pages/Settings";
import { Location } from "./pages/Location";
import { UserProfile } from "./pages/UserProfile";
import { PostDetail } from "./pages/PostDetail";
import { Notifications } from "./pages/Notifications";
import { EditProfile } from "./pages/EditProfile";
import { NearbyAll } from "./pages/NearbyAll";
import { NearbyGroups } from "./pages/NearbyGroups";
import { Search } from "./pages/Search";
import { GroupDetail } from "./pages/GroupDetail";

const legacySectionMap: Record<string, ProfileSection> = {
  moments: "moments",
  groups: "groups",
  friends: "friends",
  following: "following",
  fans: "followers",
};

function LegacyMomentRedirect() {
  const { postId } = useParams<{ postId: string }>();
  return <Navigate replace to={ROUTES.moment(postId ?? "")} />;
}

function LegacyUserRedirect() {
  const { userId } = useParams<{ userId: string }>();
  return <Navigate replace to={ROUTES.user(userId ?? "")} />;
}

function LegacyProfileRedirect() {
  const location = useLocation();
  const section = new URLSearchParams(location.search).get("section");
  const target = section && legacySectionMap[section]
    ? legacySectionMap[section] === "friends" || legacySectionMap[section] === "following" || legacySectionMap[section] === "followers"
      ? `${ROUTES.meSection("relationships")}?tab=${legacySectionMap[section]}`
      : ROUTES.meSection(legacySectionMap[section])
    : ROUTES.me;
  return <Navigate replace to={target} />;
}

/** Legacy URLs stay share-safe while all new navigation uses API-domain URLs. */
export const router = createBrowserRouter([
  {
    path: "/",
    Component: MainLayout,
    children: [
      { index: true, Component: Home },
      { path: ROUTES.discover.slice(1), Component: Discover },
      { path: ROUTES.nearby.slice(1), Component: () => <Navigate replace to={ROUTES.nearbyPeople} /> },
      { path: ROUTES.nearbyPeople.slice(1), Component: NearbyAll },
      { path: ROUTES.nearbyGroups.slice(1), Component: NearbyGroups },
      { path: ROUTES.nearbyMoments.slice(1), Component: Discover },
      { path: ROUTES.messages.slice(1), Component: Messages },
      { path: "messages/:chatId", Component: ChatDetail },
      { path: ROUTES.me.slice(1), Component: Profile },
      { path: ROUTES.meEdit.slice(1), Component: EditProfile },
      { path: "me/:section", Component: Profile },
      { path: ROUTES.settings.slice(1), Component: Settings },
      { path: ROUTES.map.slice(1), Component: Location },
      { path: "users/:userId", Component: UserProfile },
      { path: "groups/:groupId", Component: GroupDetail },
      { path: "moments/:postId", Component: PostDetail },
      { path: ROUTES.notifications.slice(1), Component: Notifications },
      { path: ROUTES.search.slice(1), Component: Search },

      // Compatibility URLs. Keep these until external links and saved client state have migrated.
      { path: "location", Component: () => <Navigate replace to={ROUTES.map} /> },
      { path: "nearby-groups", Component: () => <Navigate replace to={ROUTES.nearbyGroups} /> },
      { path: "post/:postId", Component: LegacyMomentRedirect },
      { path: "user/:userId", Component: LegacyUserRedirect },
      { path: "profile/edit", Component: () => <Navigate replace to={ROUTES.meEdit} /> },
      { path: "profile", Component: LegacyProfileRedirect },
      { path: "events/:eventId", Component: () => <Navigate replace to={ROUTES.discover} /> },
    ],
  },
]);
