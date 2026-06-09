import { lazy, Suspense } from "react";
import { Navigate } from "react-router";
import { createBrowserRouter } from "react-router";
import { MainLayout } from "./components/layout/MainLayout";
import { Home } from "./pages/Home";
import { Discover } from "./pages/Discover";
import { Messages } from "./pages/Messages";
import { ListSkeleton } from "./components/ui/skeleton";

function MomentsRedirect() {
  return <Navigate to="/profile?section=moments" replace />;
}

const ChatDetail = lazy(() =>
  import("./pages/ChatDetail").then((m) => ({ default: m.ChatDetail })),
);
const Profile = lazy(() =>
  import("./pages/Profile").then((m) => ({ default: m.Profile })),
);
const Settings = lazy(() =>
  import("./pages/Settings").then((m) => ({ default: m.Settings })),
);
const Location = lazy(() =>
  import("./pages/Location").then((m) => ({ default: m.Location })),
);
const UserProfile = lazy(() =>
  import("./pages/UserProfile").then((m) => ({ default: m.UserProfile })),
);
const PostDetail = lazy(() =>
  import("./pages/PostDetail").then((m) => ({ default: m.PostDetail })),
);
const EventDetail = lazy(() =>
  import("./pages/EventDetail").then((m) => ({ default: m.EventDetail })),
);
const Notifications = lazy(() =>
  import("./pages/Notifications").then((m) => ({ default: m.Notifications })),
);
const EditProfile = lazy(() =>
  import("./pages/EditProfile").then((m) => ({ default: m.EditProfile })),
);
const NearbyAll = lazy(() =>
  import("./pages/NearbyAll").then((m) => ({ default: m.NearbyAll })),
);
const NearbyGroups = lazy(() =>
  import("./pages/NearbyGroups").then((m) => ({ default: m.NearbyGroups })),
);
const Search = lazy(() =>
  import("./pages/Search").then((m) => ({ default: m.Search })),
);
const Login = lazy(() =>
  import("./pages/Login").then((m) => ({ default: m.Login })),
);
const Create = lazy(() =>
  import("./pages/Create").then((m) => ({ default: m.Create })),
);
const NotFound = lazy(() =>
  import("./pages/NotFound").then((m) => ({ default: m.NotFound })),
);

// Social pages
const Followers = lazy(() =>
  import("./pages/Followers").then((m) => ({ default: m.Followers })),
);
const Following = lazy(() =>
  import("./pages/Following").then((m) => ({ default: m.Following })),
);
const Friends = lazy(() =>
  import("./pages/Friends").then((m) => ({ default: m.Friends })),
);
const SharePage = lazy(() =>
  import("./pages/SharePage").then((m) => ({ default: m.SharePage })),
);

// Admin pages
const RequireAdmin = lazy(() =>
  import("./pages/admin/RequireAdmin").then((m) => ({
    default: m.RequireAdmin,
  })),
);
const AdminDashboard = lazy(() =>
  import("./pages/admin/AdminDashboard").then((m) => ({
    default: m.AdminDashboard,
  })),
);
const AdminUsers = lazy(() =>
  import("./pages/admin/AdminUsers").then((m) => ({ default: m.AdminUsers })),
);
const AdminGroups = lazy(() =>
  import("./pages/admin/AdminGroups").then((m) => ({ default: m.AdminGroups })),
);
const AdminReports = lazy(() =>
  import("./pages/admin/AdminReports").then((m) => ({
    default: m.AdminReports,
  })),
);
const AdminAnnouncements = lazy(() =>
  import("./pages/admin/AdminAnnouncements").then((m) => ({
    default: m.AdminAnnouncements,
  })),
);
const AdminAudit = lazy(() =>
  import("./pages/admin/AdminAudit").then((m) => ({ default: m.AdminAudit })),
);

// Settings pages
const ChangePassword = lazy(() =>
  import("./pages/settings/ChangePassword").then((m) => ({
    default: m.ChangePassword,
  })),
);
const PrivacySettings = lazy(() =>
  import("./pages/settings/PrivacySettings").then((m) => ({
    default: m.PrivacySettings,
  })),
);

function LazyPage({ children }: { children: React.ReactNode }) {
  return (
    <Suspense
      fallback={
        <div
          className="px-5 pt-6"
          style={{
            height: "100%",
            backgroundColor: "var(--neu-bg)",
            maxWidth: 640,
            margin: "0 auto",
          }}
        >
          <ListSkeleton rows={5} />
        </div>
      }
    >
      {children}
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    path: "/login",
    Component: () => (
      <LazyPage>
        <Login />
      </LazyPage>
    ),
  },
  {
    path: "/",
    Component: MainLayout,
    children: [
      { index: true, Component: Home },
      { path: "discover", Component: Discover },
      { path: "messages", Component: Messages },
      {
        path: "create",
        Component: () => (
          <LazyPage>
            <Create />
          </LazyPage>
        ),
      },
      {
        path: "messages/:chatId",
        Component: () => (
          <LazyPage>
            <ChatDetail />
          </LazyPage>
        ),
      },
      {
        path: "profile",
        Component: () => (
          <LazyPage>
            <Profile />
          </LazyPage>
        ),
      },
      {
        path: "moments",
        Component: () => (
          <LazyPage>
            <MomentsRedirect />
          </LazyPage>
        ),
      },
      {
        path: "profile/edit",
        Component: () => (
          <LazyPage>
            <EditProfile />
          </LazyPage>
        ),
      },
      {
        path: "settings",
        Component: () => (
          <LazyPage>
            <Settings />
          </LazyPage>
        ),
      },
      {
        path: "location",
        Component: () => (
          <LazyPage>
            <Location />
          </LazyPage>
        ),
      },
      {
        path: "user/:userId",
        Component: () => (
          <LazyPage>
            <UserProfile />
          </LazyPage>
        ),
      },
      {
        path: "post/:postId",
        Component: () => (
          <LazyPage>
            <PostDetail />
          </LazyPage>
        ),
      },
      {
        path: "events/:eventId",
        Component: () => (
          <LazyPage>
            <EventDetail />
          </LazyPage>
        ),
      },
      {
        path: "notifications",
        Component: () => (
          <LazyPage>
            <Notifications />
          </LazyPage>
        ),
      },
      {
        path: "nearby",
        Component: () => (
          <LazyPage>
            <NearbyAll />
          </LazyPage>
        ),
      },
      {
        path: "nearby-groups",
        Component: () => (
          <LazyPage>
            <NearbyGroups />
          </LazyPage>
        ),
      },
      {
        path: "search",
        Component: () => (
          <LazyPage>
            <Search />
          </LazyPage>
        ),
      },
      // Social
      {
        path: "followers",
        Component: () => (
          <LazyPage>
            <Followers />
          </LazyPage>
        ),
      },
      {
        path: "following",
        Component: () => (
          <LazyPage>
            <Following />
          </LazyPage>
        ),
      },
      {
        path: "friends",
        Component: () => (
          <LazyPage>
            <Friends />
          </LazyPage>
        ),
      },
      {
        path: "share/:code",
        Component: () => (
          <LazyPage>
            <SharePage />
          </LazyPage>
        ),
      },
      // Settings
      {
        path: "settings/password",
        Component: () => (
          <LazyPage>
            <ChangePassword />
          </LazyPage>
        ),
      },
      {
        path: "settings/privacy",
        Component: () => (
          <LazyPage>
            <PrivacySettings />
          </LazyPage>
        ),
      },
      // Admin (guarded by RequireAdmin)
      {
        path: "admin",
        Component: () => (
          <LazyPage>
            <RequireAdmin />
          </LazyPage>
        ),
        children: [
          {
            index: true,
            Component: () => (
              <LazyPage>
                <AdminDashboard />
              </LazyPage>
            ),
          },
          {
            path: "users",
            Component: () => (
              <LazyPage>
                <AdminUsers />
              </LazyPage>
            ),
          },
          {
            path: "groups",
            Component: () => (
              <LazyPage>
                <AdminGroups />
              </LazyPage>
            ),
          },
          {
            path: "reports",
            Component: () => (
              <LazyPage>
                <AdminReports />
              </LazyPage>
            ),
          },
          {
            path: "announcements",
            Component: () => (
              <LazyPage>
                <AdminAnnouncements />
              </LazyPage>
            ),
          },
          {
            path: "audit",
            Component: () => (
              <LazyPage>
                <AdminAudit />
              </LazyPage>
            ),
          },
        ],
      },
      {
        path: "*",
        Component: () => (
          <LazyPage>
            <NotFound />
          </LazyPage>
        ),
      },
    ],
  },
]);
