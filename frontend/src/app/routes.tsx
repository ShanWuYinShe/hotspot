import { createBrowserRouter, Navigate } from "react-router";
import { ROUTES } from "./lib/routes";
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
import { Auth } from "./pages/Auth";

export const router = createBrowserRouter([
  // Login/register is a top-level route so it renders without the app shell/nav.
  { path: ROUTES.login, Component: Auth },
  {
    path: "/",
    Component: MainLayout,
    children: [
      { index: true, Component: Home },
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
      { path: ROUTES.notify.slice(1), Component: Notifications },
      { path: ROUTES.search.slice(1), Component: Search },
      // Catch-all: any unknown path goes home (covers legacy /profile etc.).
      { path: "*", element: <Navigate to={ROUTES.home} replace /> },
    ],
  },
]);
