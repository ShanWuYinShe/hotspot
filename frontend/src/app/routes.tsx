import { createBrowserRouter } from "react-router";
import { MainLayout }   from "./components/layout/MainLayout";
import { Home }         from "./pages/Home";
import { Discover }     from "./pages/Discover";
import { Messages }     from "./pages/Messages";
import { ChatDetail }   from "./pages/ChatDetail";
import { Profile }      from "./pages/Profile";
import { Settings }     from "./pages/Settings";
import { Location }     from "./pages/Location";
import { UserProfile }  from "./pages/UserProfile";
import { PostDetail }   from "./pages/PostDetail";
import { EventDetail }  from "./pages/EventDetail";
import { Notifications } from "./pages/Notifications";
import { EditProfile }  from "./pages/EditProfile";
import { NearbyAll }    from "./pages/NearbyAll";
import { NearbyGroups } from "./pages/NearbyGroups";
import { Search }       from "./pages/Search";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: MainLayout,
    children: [
      { index: true,                     Component: Home         },
      { path: "discover",                Component: Discover     },
      { path: "messages",                Component: Messages     },
      { path: "messages/:chatId",        Component: ChatDetail   },
      { path: "profile",                 Component: Profile      },
      { path: "profile/edit",            Component: EditProfile  },
      { path: "settings",                Component: Settings     },
      { path: "location",                Component: Location     },
      { path: "user/:userId",            Component: UserProfile  },
      { path: "post/:postId",            Component: PostDetail   },
      { path: "events/:eventId",         Component: EventDetail  },
      { path: "notifications",           Component: Notifications},
      { path: "nearby",                  Component: NearbyAll    },
      { path: "nearby-groups",           Component: NearbyGroups },
      { path: "search",                  Component: Search       },
    ],
  },
]);