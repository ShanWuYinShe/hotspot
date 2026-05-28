import { useState, useCallback } from "react";
import { usersService } from "@/lib/api/services/users";

interface FollowToggle {
  toggledIds: Set<string>;
  toggle: (userId: string) => Promise<void>;
  isToggled: (userId: string) => boolean;
}

export function useFollowToggle(): FollowToggle {
  const [toggledIds, setToggledIds] = useState<Set<string>>(new Set());

  const toggle = useCallback(
    async (userId: string) => {
      const wasToggled = toggledIds.has(userId);
      setToggledIds((p) => {
        const n = new Set(p);
        wasToggled ? n.delete(userId) : n.add(userId);
        return n;
      });
      try {
        await (wasToggled
          ? usersService.unfollow({ user_id: userId })
          : usersService.follow({ user_id: userId }));
      } catch {
        setToggledIds((p) => {
          const n = new Set(p);
          wasToggled ? n.add(userId) : n.delete(userId);
          return n;
        });
      }
    },
    [toggledIds],
  );

  const isToggled = useCallback(
    (userId: string) => toggledIds.has(userId),
    [toggledIds],
  );

  return { toggledIds, toggle, isToggled };
}
