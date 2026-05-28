import { useEffect, useRef } from "react";
import { useAuthStore } from "@/lib/stores/auth";
import { useConfigStore } from "@/lib/stores/config";
import { authService } from "@/lib/api/services/auth";
import { usersService } from "@/lib/api/services/users";
import { ensureRefreshToken, isSuccess } from "@/lib/api/client";
import { useWebSocketStore } from "@/lib/websocket/store";
import { setPresenceStore } from "@/lib/websocket/handlers";
import { usePresenceStore } from "@/lib/stores/presence";
import { useNotificationStore } from "@/lib/stores/notification";
import { useMessageStore } from "@/lib/stores/message";
import { useAdminPermissionsStore } from "@/lib/stores/admin-permissions";

const CHECK_INTERVAL = 60 * 1000;
const REFRESH_THRESHOLD = 3;

class TokenRefreshManager {
  private checkInterval: ReturnType<typeof setInterval> | null = null;

  start() {
    if (this.checkInterval) return;
    this.checkAndRefresh();
    this.checkInterval = setInterval(
      () => this.checkAndRefresh(),
      CHECK_INTERVAL,
    );
  }

  stop() {
    if (this.checkInterval) {
      clearInterval(this.checkInterval);
      this.checkInterval = null;
    }
  }

  private async checkAndRefresh() {
    const {
      isAuthenticated,
      isTokenExpiringSoon,
      updateExpiresAt,
      invalidateSession,
    } = useAuthStore.getState();
    if (!isAuthenticated) return;
    try {
      if (isTokenExpiringSoon(REFRESH_THRESHOLD)) {
        const success = await ensureRefreshToken();
        if (success) {
          const res = await authService.checkStatus();
          if (isSuccess(res) && res.data?.expires_at) {
            updateExpiresAt(res.data.expires_at);
          } else {
            updateExpiresAt(
              new Date(Date.now() + 15 * 60 * 1000).toISOString(),
            );
          }
        } else {
          invalidateSession();
        }
      }
    } catch {
      // Token refresh failed silently — next check will retry
    }
  }
}

const tokenRefreshManager = new TokenRefreshManager();

async function restoreSession() {
  const {
    startRestoring,
    finishRestoring,
    invalidateSession,
    login,
    setProfile,
    currentUserId,
  } = useAuthStore.getState();
  startRestoring();

  try {
    let statusRes = await authService.checkStatus();
    if ((!isSuccess(statusRes) || !statusRes.data?.is_valid) && currentUserId) {
      const refreshed = await ensureRefreshToken();
      if (refreshed) {
        statusRes = await authService.checkStatus();
      }
    }
    if (!isSuccess(statusRes) || !statusRes.data?.is_valid) {
      invalidateSession();
      return;
    }

    const profileRes = await usersService.getMyProfile();
    if (!isSuccess(profileRes) || !profileRes.data) {
      invalidateSession();
      return;
    }

    const expiresAt =
      statusRes.data.expires_at ??
      new Date(Date.now() + 15 * 60 * 1000).toISOString();
    login({
      user_id: profileRes.data.user_id,
      user_type: profileRes.data.user_type,
      access_token_expires_at: expiresAt,
    });
    setProfile(profileRes.data);
  } catch {
    invalidateSession();
  } finally {
    finishRestoring();
  }
}

export function useAppInit() {
  const initialized = useRef(false);

  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;

    // Fetch client config
    useConfigStore.getState().fetch();

    // Restore session
    restoreSession();

    // Start token refresh
    tokenRefreshManager.start();

    // Initialize WebSocket and real-time stores
    setPresenceStore(usePresenceStore.getState());
    useWebSocketStore.getState().initialize();
    useNotificationStore.getState().initialize();
    useMessageStore.getState().initialize();
    useAdminPermissionsStore.getState().initialize();

    return () => {
      tokenRefreshManager.stop();
      useWebSocketStore.getState().destroy();
      useNotificationStore.getState().destroy();
      useMessageStore.getState().destroy();
      useAdminPermissionsStore.getState().destroy();
    };
  }, []);
}
