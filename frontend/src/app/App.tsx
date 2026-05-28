import { RouterProvider } from "react-router";
import { router } from "./routes";
import { ThemeProvider } from "./contexts/ThemeContext";
import { PlatformProvider } from "./contexts/PlatformContext";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { Toaster } from "./components/ui/sonner";
import { toast } from "sonner";
import { useAppInit } from "@/lib/hooks/use-app-init";
import { useNetworkStatus } from "@/lib/hooks/use-network-status";
import { MotionConfig } from "motion/react";

function AppInit() {
  useAppInit();

  useNetworkStatus((online) => {
    if (online) {
      toast.success("网络已恢复");
    } else {
      toast.error("网络已断开，部分功能可能不可用");
    }
  });

  return null;
}

export default function App() {
  return (
    <ThemeProvider>
      <PlatformProvider>
        <MotionConfig reducedMotion="user">
          <ErrorBoundary>
            <AppInit />
            <RouterProvider router={router} />
            <Toaster />
          </ErrorBoundary>
        </MotionConfig>
      </PlatformProvider>
    </ThemeProvider>
  );
}
