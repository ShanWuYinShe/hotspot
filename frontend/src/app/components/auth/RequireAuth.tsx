import { useAuthStore } from "@/lib/stores/auth";

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isRestoring = useAuthStore((s) => s.isRestoring);

  if (isRestoring) {
    return (
      <div
        className="min-h-[100dvh] flex items-center justify-center"
        style={{ backgroundColor: "var(--page-bg)" }}
      >
        <div className="flex flex-col items-center gap-3">
          <div
            className="w-8 h-8 rounded-full border-2 animate-spin"
            style={{
              borderColor: "var(--neu-border)",
              borderTopColor: "var(--neu-accent-primary)",
            }}
          />
          <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
            加载中...
          </span>
        </div>
      </div>
    );
  }

  // 来宾用户也算已认证（guest 模式）
  if (!isAuthenticated) {
    return (
      <div
        className="min-h-[100dvh] flex items-center justify-center"
        style={{ backgroundColor: "var(--page-bg)" }}
      >
        <div className="flex flex-col items-center gap-3">
          <span style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
            请先登录
          </span>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
