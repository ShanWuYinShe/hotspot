/**
 * 应用初始化插件（合并原 01/02/03）
 * - 认证初始化与可见性刷新
 * - 主题/设置初始化与同步
 * - 实时消息初始化与清理
 */

export default defineNuxtPlugin(async () => {
  const authStore = useAuthStore();
  const settingsStore = useSettingsStore();

  // 1) 等待一次重水合，确保 Pinia 持久化状态已恢复
  await nextTick();

  // 2) 认证初始化：若存在 refresh token 则静默刷新一次
  try {
    if (authStore.refreshToken) {
      await authStore.refreshAuth();
    }
  } catch (_) {
    // 忽略刷新失败，继续按现状运行
  }

  // 3) 浏览器可见性恢复时：如接近过期则静默刷新
  document.addEventListener("visibilitychange", async () => {
    if (document.visibilityState !== "visible") return;
    const refreshToken = authStore.refreshToken as string | null;
    const expiresAt = authStore.expiresAt as string | null;
    if (!refreshToken || !expiresAt) return;
    const ts = Date.parse(expiresAt);
    if (Number.isNaN(ts)) return;
    const remainingMs = ts - Date.now();
    if (remainingMs < 120_000) {
      try {
        await authStore.refreshAuth();
      } catch (_) {
        // ignore
      }
    }
  });

  // 4) 主题初始化（监听并应用）
  settingsStore.initThemeWatcher();
  settingsStore.applyTheme();

  // 5) 监听认证状态：同步设置 + 初始化/清理实时消息
  watch(
    () => authStore.isAuthenticated,
    (isAuth, wasAuth) => {
      if (isAuth) {
        // 同步用户设置（非阻塞）
        settingsStore
          .fetchSettings()
          .catch((err) => console.warn("⚠️ 用户设置同步失败:", err));

        // 初始化实时消息（不在此处请求通知权限）
        nextTick(() => {
          try {
            const rt = useRealTimeMessages();
            rt.initializeRealTime();
          } catch (e) {
            console.error("实时消息初始化失败:", e);
          }
        });
      }

      // 退出登录 → 清理实时消息
      if (!isAuth && wasAuth) {
        try {
          const rt = useRealTimeMessages();
          if ((rt as any).dispose) {
            (rt as any).dispose();
          } else {
            const ws = useWebSocket();
            ws.disconnect?.();
          }
        } catch {
          /* ignore */
        }
      }
    },
    { immediate: true },
  );

  if (import.meta.dev) console.debug("✅ 应用初始化完成（认证/主题/实时）");
});



