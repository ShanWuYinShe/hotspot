/**
 * 粉丝列表页
 */

import { useCallback } from 'react';
import { useNavigate } from 'react-router';
import { ArrowLeft } from 'lucide-react';
import { motion } from 'motion/react';
import { usersService } from '@/lib/api/services';
import { usePagination } from '@/lib/hooks/use-pagination';
import { useAuthStore } from '@/lib/stores/auth';
import { normalizeAvatar } from '@/lib/utils';
import { useDocumentTitle } from '@/lib/hooks/use-document-title';

export function Followers() {
  const navigate = useNavigate();
  const currentUserId = useAuthStore((s) => s.currentUserId);
  useDocumentTitle('粉丝');

  const pagination = usePagination({ uniqueKey: 'user_id' });

  const loadFollowers = useCallback(
    async (refresh = false) => {
      await pagination.load(
        (cursor) => usersService.getFollowers({ user_id: currentUserId ?? undefined, pagination: { limit: 20, cursor } }),
        refresh,
      );
    },
    [currentUserId, pagination],
  );

  // Initial load
  if (pagination.items.length === 0 && !pagination.loading && !pagination.error) {
    loadFollowers(true);
  }

  return (
    <div className="flex flex-col h-full" style={{ backgroundColor: 'var(--neu-bg)' }}>
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: '1px solid var(--neu-border)' }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none"
          style={{ width: 36, height: 36, background: 'var(--neu-bg)', boxShadow: '4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)', border: '1px solid var(--neu-border)' }}
        >
          <ArrowLeft style={{ width: 15, height: 15, color: 'var(--neu-text-muted)' }} strokeWidth={1.5} />
        </motion.button>
        <p style={{ fontSize: 16, fontWeight: 600, color: 'var(--neu-text)' }}>粉丝</p>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3">
        {pagination.loading && pagination.items.length === 0 ? (
          <p className="text-center py-8" style={{ color: 'var(--neu-text-muted)', fontSize: 14 }}>加载中…</p>
        ) : pagination.items.length === 0 ? (
          <p className="text-center py-8" style={{ color: 'var(--neu-text-muted)', fontSize: 14 }}>暂无粉丝</p>
        ) : (
          <div className="flex flex-col gap-2">
            {pagination.items.map((user) => (
              <div
                key={user.user_id}
                onClick={() => navigate(`/user/${user.user_id}`)}
                className="flex items-center gap-3 p-3 rounded-xl cursor-pointer active:scale-[0.98] transition-transform"
                style={{ background: 'var(--neu-bg)', boxShadow: '3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)', border: '1px solid var(--neu-border)' }}
              >
                <img
                  src={normalizeAvatar(user.avatar_url)}
                  alt={user.nickname}
                  className="rounded-full flex-shrink-0"
                  style={{ width: 40, height: 40, border: '1px solid var(--neu-border)' }}
                />
                <div className="flex-1 min-w-0">
                  <p className="truncate" style={{ fontSize: 14, color: 'var(--neu-text)' }}>{user.nickname}</p>
                  {user.bio && <p className="truncate" style={{ fontSize: 12, color: 'var(--neu-text-muted)' }}>{user.bio}</p>}
                </div>
              </div>
            ))}
            {pagination.hasMore && (
              <button
                onClick={() => loadFollowers(false)}
                disabled={pagination.loadingMore}
                className="py-2 text-center"
                style={{ color: 'var(--neu-text-muted)', fontSize: 13 }}
              >
                {pagination.loadingMore ? '加载中…' : '加载更多'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
