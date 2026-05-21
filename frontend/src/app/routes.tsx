import { lazy, Suspense } from "react";
import { createBrowserRouter } from "react-router";
import { MainLayout }   from "./components/layout/MainLayout";
import { Home }         from "./pages/Home";
import { Discover }     from "./pages/Discover";
import { Messages }     from "./pages/Messages";
import { ListSkeleton } from "./components/ui/skeleton";

const ChatDetail    = lazy(() => import("./pages/ChatDetail").then(m => ({ default: m.ChatDetail })));
const Profile       = lazy(() => import("./pages/Profile").then(m => ({ default: m.Profile })));
const Settings      = lazy(() => import("./pages/Settings").then(m => ({ default: m.Settings })));
const Location      = lazy(() => import("./pages/Location").then(m => ({ default: m.Location })));
const UserProfile   = lazy(() => import("./pages/UserProfile").then(m => ({ default: m.UserProfile })));
const PostDetail    = lazy(() => import("./pages/PostDetail").then(m => ({ default: m.PostDetail })));
const EventDetail   = lazy(() => import("./pages/EventDetail").then(m => ({ default: m.EventDetail })));
const Notifications = lazy(() => import("./pages/Notifications").then(m => ({ default: m.Notifications })));
const EditProfile   = lazy(() => import("./pages/EditProfile").then(m => ({ default: m.EditProfile })));
const NearbyAll     = lazy(() => import("./pages/NearbyAll").then(m => ({ default: m.NearbyAll })));
const NearbyGroups  = lazy(() => import("./pages/NearbyGroups").then(m => ({ default: m.NearbyGroups })));
const Search        = lazy(() => import("./pages/Search").then(m => ({ default: m.Search })));
const Login         = lazy(() => import("./pages/Login").then(m => ({ default: m.Login })));
const Create        = lazy(() => import("./pages/Create").then(m => ({ default: m.Create })));
const NotFound      = lazy(() => import("./pages/NotFound").then(m => ({ default: m.NotFound })));

function LazyPage({ children }: { children: React.ReactNode }) {
  return (
    <Suspense fallback={
      <div className="px-5 pt-6" style={{ height: "100%", backgroundColor: "var(--neu-bg)", maxWidth: 640, margin: "0 auto" }}>
        <ListSkeleton rows={5} />
      </div>
    }>
      {children}
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    path: "/login",
    Component: () => <LazyPage><Login /></LazyPage>,
  },
  {
    path: "/",
    Component: MainLayout,
    children: [
      { index: true,                     Component: Home         },
      { path: "discover",                Component: Discover     },
      { path: "messages",                Component: Messages     },
      { path: "create",                  Component: () => <LazyPage><Create /></LazyPage>       },
      { path: "messages/:chatId",        Component: () => <LazyPage><ChatDetail /></LazyPage>   },
      { path: "profile",                 Component: () => <LazyPage><Profile /></LazyPage>      },
      { path: "profile/edit",            Component: () => <LazyPage><EditProfile /></LazyPage>  },
      { path: "settings",                Component: () => <LazyPage><Settings /></LazyPage>     },
      { path: "location",                Component: () => <LazyPage><Location /></LazyPage>     },
      { path: "user/:userId",            Component: () => <LazyPage><UserProfile /></LazyPage>  },
      { path: "post/:postId",            Component: () => <LazyPage><PostDetail /></LazyPage>   },
      { path: "events/:eventId",         Component: () => <LazyPage><EventDetail /></LazyPage>  },
      { path: "notifications",           Component: () => <LazyPage><Notifications /></LazyPage>},
      { path: "nearby",                  Component: () => <LazyPage><NearbyAll /></LazyPage>    },
      { path: "nearby-groups",           Component: () => <LazyPage><NearbyGroups /></LazyPage> },
      { path: "search",                  Component: () => <LazyPage><Search /></LazyPage>       },
      { path: "*",                        Component: () => <LazyPage><NotFound /></LazyPage>     },
    ],
  },
]);
