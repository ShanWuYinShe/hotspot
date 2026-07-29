import { createBrowserRouter, Navigate } from "react-router";
import { ROUTES } from "./lib/routes";
import { MainLayout } from "./components/layout/MainLayout";
// Pages are grouped by domain; each file name mirrors its route segment.
import { AuthPage } from "./pages/auth/AuthPage";
import { NearbyFeedPage } from "./pages/nearby/NearbyFeedPage";
import { NearbyPeoplePage } from "./pages/nearby/NearbyPeoplePage";
import { NearbyGroupsPage } from "./pages/nearby/NearbyGroupsPage";
import { MomentsFeedPage } from "./pages/moments/MomentsFeedPage";
import { MomentDetailPage } from "./pages/moments/MomentDetailPage";
import { MessagesPage } from "./pages/messaging/MessagesPage";
import { ChatDetailPage } from "./pages/messaging/ChatDetailPage";
import { ProfilePage } from "./pages/profile/ProfileFeature";
import { EditProfilePage } from "./pages/profile/EditProfilePage";
import { UserProfilePage } from "./pages/user/UserProfilePage";
import { GroupDetailPage } from "./pages/groups/GroupDetailPage";
import { SettingsPage } from "./pages/settings/SettingsPage";
import { LocationPage } from "./pages/location/LocationPage";
import { SearchPage } from "./pages/search/SearchPage";
import { NotificationsPage } from "./pages/notifications/NotificationsPage";

export const router = createBrowserRouter([
  // Login/register renders without the app shell/nav.
  { path: ROUTES.login, Component: AuthPage },
  {
    // App shell: all authenticated destinations share MainLayout's <Outlet/>.
    path: "/",
    Component: MainLayout,
    children: [
      // ── Nearby (home tab = "附近") ────────────────────────────────
      { index: true, Component: NearbyFeedPage }, // "/"
      { path: "nearby/people", Component: NearbyPeoplePage },
      { path: "nearby/groups", Component: NearbyGroupsPage },
      { path: "nearby/moments", Component: MomentsFeedPage },

      // ── Moments feed & detail ─────────────────────────────────────
      { path: "moments/:postId", Component: MomentDetailPage },

      // ── Messaging ─────────────────────────────────────────────────
      { path: "messages", Component: MessagesPage },
      { path: "messages/:chatId", Component: ChatDetailPage },

      // ── Self profile (/me + /me/:section + /me/edit share ProfilePage) ──
      { path: "me", Component: ProfilePage },
      { path: "me/:section", Component: ProfilePage },
      { path: "me/edit", Component: EditProfilePage },

      // ── Other users & groups ──────────────────────────────────────
      { path: "users/:userId", Component: UserProfilePage },
      { path: "groups/:groupId", Component: GroupDetailPage },

      // ── Utility / standalone pages ────────────────────────────────
      { path: "settings", Component: SettingsPage },
      { path: "map", Component: LocationPage },
      { path: "notify", Component: NotificationsPage },
      { path: "search", Component: SearchPage },

      // Catch-all: any unknown path goes home (covers legacy /profile etc.).
      { path: "*", element: <Navigate to={ROUTES.home} replace /> },
    ],
  },
]);
