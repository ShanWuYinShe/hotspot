import { useState, useCallback } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Search, Users, Trash2 } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";
import { adminGroupsService } from "@/lib/api/services";
import { isSuccess } from "@/lib/api/client";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { usePagination } from "@/lib/hooks/use-pagination";
import type { AdminGroupItem } from "@/lib/types/api";

export function AdminGroups() {
  const navigate = useNavigate();
  useDocumentTitle("群组管理");

  const [keyword, setKeyword] = useState("");
  const page = usePagination<AdminGroupItem>({
    uniqueKey: (item) => item.group.group_id,
  });

  const doSearch = useCallback(
    (refresh = true) => {
      page.load(
        (cursor) =>
          adminGroupsService.search({
            name: keyword || undefined,
            pagination: { limit: 20, cursor },
          }),
        refresh,
      );
    },
    [page, keyword],
  );

  const handleDissolve = useCallback(
    async (group: AdminGroupItem) => {
      const confirmed = window.confirm(
        `确定要解散群组「${group.group.name}」吗？此操作不可撤销。`,
      );
      if (!confirmed) return;

      try {
        const res = await adminGroupsService.dismiss({
          group_id: group.group.group_id,
        });
        if (isSuccess(res)) {
          toast.success(`已解散 ${group.group.name}`);
          doSearch(true);
        } else {
          toast.error("解散失败");
        }
      } catch {
        toast.error("解散请求失败");
      }
    },
    [doSearch],
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
          群组管理
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
        {/* Search */}
        <div
          className="flex items-center gap-2 px-3 py-2.5 rounded-[14px] mb-4"
          style={{
            background: "var(--neu-bg)",
            boxShadow:
              "inset 3px 3px 6px var(--neu-shadow-dark), inset -3px -3px 6px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <Search
            style={{
              width: 16,
              height: 16,
              color: "var(--neu-text-muted)",
              flexShrink: 0,
            }}
            strokeWidth={1.5}
          />
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && doSearch(true)}
            placeholder="搜索群组名称..."
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 14, color: "var(--neu-text)" }}
          />
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={() => doSearch(true)}
            className="px-3 py-1 rounded-full"
            style={{
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
              fontSize: 12,
            }}
          >
            搜索
          </motion.button>
        </div>

        {page.error && (
          <p
            className="mb-3 text-center"
            style={{ fontSize: 13, color: "var(--neu-danger)" }}
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
            <Users
              style={{ width: 40, height: 40, color: "var(--neu-text-muted)" }}
              strokeWidth={1.2}
            />
            <p
              className="mt-3"
              style={{ fontSize: 14, color: "var(--neu-text-muted)" }}
            >
              暂无群组数据
            </p>
          </div>
        )}

        <div className="flex flex-col gap-2">
          {page.items.map((group) => (
            <div
              key={group.group.group_id}
              className="flex items-center gap-3 px-4 py-3 rounded-[16px]"
              style={{
                background: "var(--neu-bg)",
                boxShadow:
                  "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              {/* Avatar */}
              <div
                className="rounded-full overflow-hidden flex-shrink-0"
                style={{ width: 40, height: 40 }}
              >
                {group.group.avatar_url ? (
                  <img
                    src={group.group.avatar_url}
                    alt=""
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div
                    className="w-full h-full flex items-center justify-center"
                    style={{ background: "#10B981" }}
                  >
                    <span style={{ fontSize: 16, color: "white" }}>
                      {group.group.name.charAt(0)}
                    </span>
                  </div>
                )}
              </div>

              {/* Info */}
              <div className="flex-1 min-w-0">
                <span
                  className="truncate block"
                  style={{
                    fontSize: 14,
                    fontWeight: 500,
                    color: "var(--neu-text)",
                  }}
                >
                  {group.group.name}
                </span>
                <p
                  className="truncate mt-0.5"
                  style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                >
                  成员 {group.stats?.active_members_today ?? 0} · 消息{" "}
                  {group.stats?.messages_count ?? 0}
                  {group.stats?.reports_count
                    ? ` · 举报 ${group.stats.reports_count}`
                    : ""}
                </p>
              </div>

              {/* Dissolve */}
              <motion.button
                whileTap={{ scale: 0.92 }}
                onClick={() => handleDissolve(group)}
                className="flex items-center justify-center rounded-full flex-shrink-0"
                style={{ width: 32, height: 32, background: "var(--neu-danger-soft)" }}
                title="解散群组"
              >
                <Trash2
                  style={{ width: 15, height: 15, color: "var(--neu-danger)" }}
                  strokeWidth={1.5}
                />
              </motion.button>
            </div>
          ))}
        </div>

        {page.hasMore && !page.loading && page.items.length > 0 && (
          <motion.button
            whileTap={{ scale: 0.97 }}
            onClick={() => doSearch(false)}
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
