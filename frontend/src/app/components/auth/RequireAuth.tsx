import React from "react";
import { Navigate } from "react-router";
import { useAuth } from "../../contexts/AuthContext";

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
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

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <>{children}</>;
}
