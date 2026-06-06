/**
 * 关注列表页
 */

import { useCallback, useEffect, useRef } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Users } from "lucide-react";
import { motion } from "motion/react";
import { usersService } from "@/lib/api/services";
import { usePagination } from "@/lib/hooks/use-pagination";
import { useAuthStore } from "@/lib/stores/auth";
import { normalizeAvatar } from "@/lib/utils";
import { EmptyState } from "../components/ui/EmptyState";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";

export function Following() {
  const navigate = useNavigate();
  const currentUserId = useAuthStore((s) => s.currentUserId);
  useDocumentTitle("关注");

  const { load, items, loading, error, hasMore, loadingMore } = usePagination({
    uniqueKey: "user_id",
    initialLoading: false,
  });

  const initialFetched = useRef(false);

  const loadFollowing = useCallback(
    async (refresh = false) => {
      await load(
        (cursor) =>
          usersService.getFollowing({
            user_id: currentUserId ?? undefined,
            pagination: { limit: 20, cursor },
          }),
        refresh,
      );
    },
    [currentUserId, load],
  );

  useEffect(() => {
    if (!initialFetched.current && !loading && !error) {
      initialFetched.current = true;
      loadFollowing(true);
    }
  }, [loadFollowing, loading, error]);

  return (
    <div
      className="flex flex-col h-full"
      style={{ backgroundColor: "var(--neu-bg)" }}
    >
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
        <p style={{ fontSize: 16, fontWeight: 600, color: "var(--neu-text)" }}>
          关注
        </p>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3">
        {loading && items.length === 0 ? (
          <p
            className="text-center py-8"
            style={{ color: "var(--neu-text-muted)", fontSize: 14 }}
          >
            加载中…
          </p>
        ) : items.length === 0 ? (
          <EmptyState icon={Users} title="暂无关注" />
        ) : (
          <div className="flex flex-col gap-2">
            {items.map((user) => (
              <div
                key={user.user_id}
                onClick={() => navigate(`/user/${user.user_id}`)}
                className="flex items-center gap-3 p-3 rounded-xl cursor-pointer active:scale-[0.98] transition-transform"
                style={{
                  background: "var(--neu-bg)",
                  boxShadow:
                    "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)",
                  border: "1px solid var(--neu-border)",
                }}
              >
                <img
                  src={normalizeAvatar(user.avatar_url)}
                  alt={user.nickname}
                  className="rounded-full flex-shrink-0"
                  style={{
                    width: 40,
                    height: 40,
                    border: "1px solid var(--neu-border)",
                  }}
                />
                <div className="flex-1 min-w-0">
                  <p
                    className="truncate"
                    style={{ fontSize: 14, color: "var(--neu-text)" }}
                  >
                    {user.nickname}
                  </p>
                  {user.bio && (
                    <p
                      className="truncate"
                      style={{ fontSize: 12, color: "var(--neu-text-muted)" }}
                    >
                      {user.bio}
                    </p>
                  )}
                </div>
              </div>
            ))}
            {hasMore && (
              <button
                onClick={() => loadFollowing(false)}
                disabled={loadingMore}
                className="py-2 text-center"
                style={{ color: "var(--neu-text-muted)", fontSize: 13 }}
              >
                {loadingMore ? "加载中…" : "加载更多"}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
