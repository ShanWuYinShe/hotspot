import { useState, useEffect, useCallback } from "react";
import { privateChatsService, groupsService } from "@/lib/api/services";
import { isSuccess, TOKEN_INVALID_CODE } from "@/lib/api/client";
import type { PrivateChatItem, GroupItem } from "@/lib/types/api";

interface MessagesData {
  privateChats: PrivateChatItem[];
  groupChats: GroupItem[];
  loadingPrivate: boolean;
  loadingGroup: boolean;
  error: string | null;
  totalUnread: number;
  refreshPrivate: () => void;
  loadGroups: () => void;
}

export function useMessagesData(): MessagesData {
  const [privateChats, setPrivateChats] = useState<PrivateChatItem[]>([]);
  const [groupChats, setGroupChats] = useState<GroupItem[]>([]);
  const [loadingPrivate, setLoadingPrivate] = useState(true);
  const [loadingGroup, setLoadingGroup] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [groupsLoaded, setGroupsLoaded] = useState(false);

  const totalUnread = privateChats.reduce((sum, c) => sum + c.unread_count, 0);

  const loadPrivate = useCallback(async () => {
    setLoadingPrivate(true);
    setError(null);
    try {
      const res = await privateChatsService.getChats({
        pagination: { limit: 50 },
      });
      if (isSuccess(res) && res.data) {
        setPrivateChats(res.data.items);
      } else if (res.code !== TOKEN_INVALID_CODE) {
        setError("加载失败，请重试");
      }
    } catch {
      setError("网络异常");
    } finally {
      setLoadingPrivate(false);
    }
  }, []);

  const loadGroups = useCallback(async () => {
    if (groupsLoaded) return;
    setLoadingGroup(true);
    try {
      const res = await groupsService.getMyGroups({
        pagination: { limit: 50 },
      });
      if (isSuccess(res) && res.data) {
        setGroupChats(res.data.items);
        setGroupsLoaded(true);
      }
    } catch {
      // non-critical
    } finally {
      setLoadingGroup(false);
    }
  }, [groupsLoaded]);

  useEffect(() => {
    loadPrivate();
  }, [loadPrivate]);

  return {
    privateChats,
    groupChats,
    loadingPrivate,
    loadingGroup,
    error,
    totalUnread,
    refreshPrivate: loadPrivate,
    loadGroups,
  };
}
