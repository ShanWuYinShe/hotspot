import { useState, useCallback } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, ClipboardList } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { adminAuditService } from "@/lib/api/services";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { usePagination } from "@/lib/hooks/use-pagination";
import type { AuditLogItem, AuditAction } from "@/lib/types/api";

const ACTION_FILTERS: { label: string; value: AuditAction | "all" }[] = [
  { label: "全部", value: "all" },
  { label: "封禁用户", value: "user.ban" },
  { label: "解封用户", value: "user.unban" },
  { label: "解散群组", value: "group.dismiss" },
  { label: "创建公告", value: "announcement.create" },
  { label: "删除公告", value: "announcement.delete" },
  { label: "角色操作", value: "role.create" },
  { label: "配置更新", value: "config.update" },
];

export function AdminAudit() {
  const navigate = useNavigate();
  useDocumentTitle("审计日志");

  const [actionFilter, setActionFilter] = useState<AuditAction | "all">("all");
  const page = usePagination<AuditLogItem>({ uniqueKey: "log_id" });

  const fetchLogs = useCallback(
    (refresh = true) => {
      page.load(
        (cursor) =>
          adminAuditService.getList({
            action: actionFilter === "all" ? undefined : actionFilter,
            pagination: { limit: 20, cursor },
          }),
        refresh,
      );
    },
    [page, actionFilter],
  );

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
          审计日志
        </p>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex-1 px-4 py-4"
        style={{
          maxWidth: 600,
          marginLeft: "auto",
          marginRight: "auto",
          width: "100%",
        }}
      >
        {/* Action filter */}
        <div className="flex gap-2 mb-4 overflow-x-auto pb-1">
          {ACTION_FILTERS.map((opt) => (
            <motion.button
              key={opt.value}
              whileTap={{ scale: 0.94 }}
              onClick={() => setActionFilter(opt.value)}
              className="px-3 py-1.5 rounded-full whitespace-nowrap flex-shrink-0"
              style={{
                fontSize: 12,
                background:
                  actionFilter === opt.value
                    ? "var(--neu-accent-primary)"
                    : "var(--neu-bg)",
                color:
                  actionFilter === opt.value
                    ? "var(--text-on-accent, white)"
                    : "var(--neu-text-muted)",
                boxShadow:
                  actionFilter === opt.value
                    ? "0 2px 8px rgba(0,0,0,0.15)"
                    : "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              {opt.label}
            </motion.button>
          ))}
        </div>

        {page.error && (
          <p
            className="mb-3 text-center"
            style={{ fontSize: 13, color: "#EF4444" }}
          >
            {page.error}
          </p>
        )}

        {page.loading && (
          <div className="flex items-center justify-center py-16">
            <div
              className="rounded-full"
              style={{
                width: 28,
                height: 28,
                border: "3px solid var(--neu-border)",
                borderTopColor: "var(--neu-accent-primary)",
                animation: "spin 0.8s linear infinite",
              }}
            />
            <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
          </div>
        )}

        {!page.loading && page.items.length === 0 && (
          <div className="flex flex-col items-center py-16">
            <ClipboardList
              style={{ width: 40, height: 40, color: "var(--neu-text-muted)" }}
              strokeWidth={1.2}
            />
            <p
              className="mt-3"
              style={{ fontSize: 14, color: "var(--neu-text-muted)" }}
            >
              暂无审计日志
            </p>
          </div>
        )}

        <div className="flex flex-col gap-2">
          {page.items.map((log) => (
            <div
              key={log.log_id}
              className="px-4 py-3 rounded-[16px]"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              {/* Timestamp */}
              <p
                className="mb-1.5"
                style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
              >
                {new Date(log.created_at).toLocaleString()}
              </p>

              {/* Actor & action */}
              <div className="flex items-center gap-2 mb-1">
                <span
                  style={{
                    fontSize: 13,
                    fontWeight: 500,
                    color: "var(--neu-text)",
                  }}
                >
                  {log.operator?.nickname ?? "系统"}
                </span>
                <span
                  className="px-2 py-0.5 rounded-full"
                  style={{
                    fontSize: 11,
                    background: "var(--neu-accent-primary)18",
                    color: "var(--neu-accent-primary)",
                  }}
                >
                  {log.action}
                </span>
              </div>

              {/* Target */}
              <p style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
                目标：{log.target_type}
                {log.target_name
                  ? ` · ${log.target_name}`
                  : ` · ${log.target_id.slice(0, 12)}`}
              </p>

              {/* IP & device */}
              {(log.ip_address || log.device_summary) && (
                <p
                  className="mt-1"
                  style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                >
                  {log.ip_address ?? ""}
                  {log.ip_location ? ` (${log.ip_location})` : ""}
                  {log.device_summary ? ` · ${log.device_summary}` : ""}
                </p>
              )}
            </div>
          ))}
        </div>

        {page.hasMore && !page.loading && page.items.length > 0 && (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => fetchLogs(false)}
            disabled={page.loadingMore}
            className="mt-4 py-2.5 rounded-[14px] w-full"
            style={{
              background: "var(--neu-bg)",
              boxShadow:
                "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
              fontSize: 13,
              color: "var(--neu-text-muted)",
            }}
          >
            {page.loadingMore ? "加载中..." : "加载更多"}
          </motion.button>
        )}
      </motion.div>
    </div>
  );
}
