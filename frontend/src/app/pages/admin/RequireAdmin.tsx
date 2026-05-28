import { useEffect } from "react";
import { useNavigate } from "react-router";
import { Shield, ArrowLeft } from "lucide-react";
import { motion } from "motion/react";
import { Outlet } from "react-router";
import { useAdminPermissionsStore } from "@/lib/stores/admin-permissions";
import { useAuthStore } from "@/lib/stores/auth";

export function RequireAdmin() {
  const navigate = useNavigate();
  const currentUserId = useAuthStore((s) => s.currentUserId);
  const sessionValid = useAuthStore((s) => s.sessionValid);
  const { isAdmin, isLoading, ensureLoaded } = useAdminPermissionsStore();

  useEffect(() => {
    if (currentUserId && sessionValid) {
      ensureLoaded(currentUserId);
    }
  }, [ensureLoaded, currentUserId, sessionValid]);

  if (isLoading) {
    return (
      <div
        className="flex flex-col items-center justify-center flex-1"
        style={{ minHeight: "100vh" }}
      >
        <div
          className="rounded-full"
          style={{
            width: 40,
            height: 40,
            border: "3px solid var(--neu-border)",
            borderTopColor: "var(--neu-accent-primary)",
            animation: "spin 0.8s linear infinite",
          }}
        />
        <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
        <p
          className="mt-4"
          style={{ fontSize: 13, color: "var(--neu-text-muted)" }}
        >
          验证权限中...
        </p>
      </div>
    );
  }

  if (!isAdmin) {
    return (
      <div
        className="flex flex-col items-center justify-center flex-1 px-6"
        style={{ minHeight: "100vh" }}
      >
        <div
          className="flex items-center justify-center rounded-full mb-5"
          style={{
            width: 72,
            height: 72,
            background: "var(--neu-bg)",
            boxShadow:
              "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
          }}
        >
          <Shield
            style={{
              width: 32,
              height: 32,
              color: "var(--neu-accent-primary)",
            }}
            strokeWidth={1.5}
          />
        </div>
        <h2
          className="mb-2"
          style={{ fontSize: 20, fontWeight: 600, color: "var(--neu-text)" }}
        >
          访问受限
        </h2>
        <p
          className="mb-6 text-center"
          style={{
            fontSize: 14,
            color: "var(--neu-text-muted)",
            maxWidth: 280,
          }}
        >
          你没有管理员权限，无法访问此页面。如需权限请联系超级管理员。
        </p>
        <motion.button
          whileTap={{ scale: 0.94 }}
          onClick={() => navigate(-1)}
          className="flex items-center gap-2 px-5 py-2.5 rounded-full outline-none"
          style={{
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 14,
            boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
          }}
        >
          <ArrowLeft style={{ width: 14, height: 14 }} strokeWidth={1.75} />
          返回
        </motion.button>
      </div>
    );
  }

  return <Outlet />;
}
