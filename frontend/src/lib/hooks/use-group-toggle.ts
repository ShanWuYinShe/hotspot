import { useState, useCallback } from "react";
import { groupsService } from "@/lib/api/services/groups";

interface GroupToggle {
  joinedIds: Set<string>;
  toggle: (groupId: string) => Promise<void>;
  isJoined: (groupId: string) => boolean;
}

export function useGroupToggle(): GroupToggle {
  const [joinedIds, setJoinedIds] = useState<Set<string>>(new Set());

  const toggle = useCallback(
    async (groupId: string) => {
      const wasJoined = joinedIds.has(groupId);
      setJoinedIds((p) => {
        const n = new Set(p);
        wasJoined ? n.delete(groupId) : n.add(groupId);
        return n;
      });
      try {
        await (wasJoined
          ? groupsService.leave({ group_id: groupId })
          : groupsService.join({ group_id: groupId }));
      } catch {
        setJoinedIds((p) => {
          const n = new Set(p);
          wasJoined ? n.add(groupId) : n.delete(groupId);
          return n;
        });
      }
    },
    [joinedIds],
  );

  const isJoined = useCallback(
    (groupId: string) => joinedIds.has(groupId),
    [joinedIds],
  );

  return { joinedIds, toggle, isJoined };
}
