import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router";
import {
  ArrowLeft,
  Users,
  UserCheck,
  Flag,
  Heart,
  Shield,
  Activity,
} from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import {
  adminUsersService,
  adminGroupsService,
  adminReportsService,
  systemService,
} from "@/lib/api/services";
import { isSuccess } from "@/lib/api/client";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import type { HealthStatus } from "@/lib/types/api";

interface StatCard {
  label: string;
  value: string | number;
  icon: React.ElementType;
  color: string;
}

export function AdminDashboard() {
  const navigate = useNavigate();
  useDocumentTitle("管理面板");

  const [stats, setStats] = useState<StatCard[]>([]);
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchDashboard = useCallback(async () => {
    setLoading(true);
    try {
      const [usersRes, groupsRes, reportsRes, healthRes] = await Promise.all([
        adminUsersService.search({ pagination: { limit: 1 } }),
        adminGroupsService.search({ pagination: { limit: 1 } }),
        adminReportsService.search({
          status: "pending",
          pagination: { limit: 1 },
        }),
        systemService.health(),
      ]);

      const toCount = (res: {
        code?: number;
        data?: { items?: unknown[]; next_cursor?: string };
      }) => {
        if (!isSuccess(res) || !res.data) return "0";
        if (res.data.items && res.data.items.length === 0) return "0";
        return res.data.next_cursor ? "1+" : "1";
      };
      const userCount = toCount(usersRes);
      const groupCount = toCount(groupsRes);
      const reportCount = toCount(reportsRes);

      // Use next_cursor existence to know if there are more items, but we only have count hint from limit=1
      setStats([
        {
          label: "用户总数",
          value: userCount,
          icon: Users,
          color: "#6366F1",
        },
        {
          label: "群组总数",
          value: groupCount,
          icon: UserCheck,
          color: "#10B981",
        },
        {
          label: "待处理举报",
          value: reportCount,
          icon: Flag,
          color: "#F59E0B",
        },
        {
          label: "系统健康",
          value:
            isSuccess(healthRes) && healthRes.data
              ? healthRes.data.status
              : "未知",
          icon: Heart,
          color: "#EC4899",
        },
      ]);

      if (isSuccess(healthRes) && healthRes.data) {
        setHealth(healthRes.data);
      }
    } catch {
      toast.error("加载仪表盘失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  const navItems = [
    { label: "用户管理", icon: Users, path: "/admin/users", color: "#6366F1" },
    {
      label: "群组管理",
      icon: UserCheck,
      path: "/admin/groups",
      color: "#10B981",
    },
    { label: "举报处理", icon: Flag, path: "/admin/reports", color: "#F59E0B" },
    {
      label: "公告管理",
      icon: Activity,
      path: "/admin/announcements",
      color: "#8B5CF6",
    },
    { label: "RBAC 权限", icon: Shield, path: "/admin/rbac", color: "#EF4444" },
  ];

  return (
    <div
      className="flex flex-col min-h-screen"
      style={{ background: "var(--neu-bg)" }}
    >
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
        <p style={{ fontSize: 16, fontWeight: 600, color: "var(--neu-text)" }}>
          管理面板
        </p>
      </div>

      {/* Content */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex-1 px-4 py-5"
        style={{
          maxWidth: 600,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        {/* Stats */}
        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div
              className="rounded-full"
              style={{
                width: 32,
                height: 32,
                border: "3px solid var(--neu-border)",
                borderTopColor: "var(--neu-accent-primary)",
                animation: "spin 0.8s linear infinite",
              }}
            />
            <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
          </div>
        ) : (
          <>
            <div className="grid grid-cols-2 gap-3 mb-6">
              {stats.map((card) => (
                <motion.div
                  key={card.label}
                  whileTap={{ scale: 0.97 }}
                  className="flex flex-col items-center py-5 rounded-[20px]"
                  style={{
                    background: "var(--neu-bg)",
                    boxShadow:
                      "5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)",
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  <div
                    className="flex items-center justify-center rounded-full mb-3"
                    style={{
                      width: 40,
                      height: 40,
                      background: `${card.color}18`,
                    }}
                  >
                    <card.icon
                      style={{ width: 18, height: 18, color: card.color }}
                      strokeWidth={1.5}
                    />
                  </div>
                  <span
                    className="mb-1"
                    style={{
                      fontSize: 22,
                      fontWeight: 700,
                      color: "var(--neu-text)",
                    }}
                  >
                    {typeof card.value === "number"
                      ? card.value
                      : String(card.value)}
                  </span>
                  <span
                    style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                  >
                    {card.label}
                  </span>
                </motion.div>
              ))}
            </div>

            {/* System health */}
            {health && (
              <div
                className="flex items-center gap-3 px-4 py-3 rounded-[16px] mb-6"
                style={{
                  background: "var(--neu-bg)",
                  boxShadow:
                    "inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light)",
                  border: "1px solid var(--neu-border)",
                }}
              >
                <div
                  className="rounded-full"
                  style={{
                    width: 10,
                    height: 10,
                    background:
                      health.status === "healthy"
                        ? "#10B981"
                        : health.status === "degraded"
                          ? "#F59E0B"
                          : "#EF4444",
                  }}
                />
                <span style={{ fontSize: 13, color: "var(--neu-text)" }}>
                  系统状态：
                  {health.status === "healthy"
                    ? "正常"
                    : health.status === "degraded"
                      ? "降级"
                      : "异常"}
                </span>
              </div>
            )}

            {/* Navigation */}
            <p
              className="px-1 mb-2"
              style={{
                fontSize: 11,
                letterSpacing: "0.12em",
                textTransform: "uppercase",
                color: "var(--neu-text-muted)",
              }}
            >
              快捷入口
            </p>
            <div className="flex flex-col gap-2">
              {navItems.map((item) => (
                <motion.button
                  key={item.path}
                  whileTap={{ scale: 0.98 }}
                  onClick={() => navigate(item.path)}
                  className="flex items-center gap-3 px-4 py-3.5 rounded-[16px] w-full outline-none text-left"
                  style={{
                    background: "var(--neu-bg)",
                    boxShadow:
                      "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                    border: "1px solid var(--neu-border)",
                  }}
                >
                  <div
                    className="flex items-center justify-center rounded-full"
                    style={{
                      width: 36,
                      height: 36,
                      background: `${item.color}18`,
                    }}
                  >
                    <item.icon
                      style={{ width: 16, height: 16, color: item.color }}
                      strokeWidth={1.5}
                    />
                  </div>
                  <span style={{ fontSize: 14, color: "var(--neu-text)" }}>
                    {item.label}
                  </span>
                </motion.button>
              ))}
            </div>
          </>
        )}
      </motion.div>
    </div>
  );
}
