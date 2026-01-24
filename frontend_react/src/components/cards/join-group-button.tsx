'use client';

/**
 * 加入群组按钮 (Client Component)
 * 处理加入/退出群组交互
 */

import { useState, useTransition, useOptimistic } from 'react';
import { Lock } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { GroupPasswordDialog } from '@/components/common/group-password-dialog';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { groupsService } from '@/lib/api/services/groups';
import { isSuccess } from '@/lib/api/client';

interface JoinGroupButtonProps {
  groupId: string;
  isJoined: boolean;
  hasPassword?: boolean;
}

export function JoinGroupButton({
  groupId,
  isJoined: initialIsJoined,
  hasPassword = false,
}: JoinGroupButtonProps) {
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  const [optimisticIsJoined, setOptimisticIsJoined] = useOptimistic(initialIsJoined);
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (!requireAuth(optimisticIsJoined ? '退出群组' : '加入群组')) {
      return;
    }

    if (hasPassword && !optimisticIsJoined) {
      setPasswordDialogOpen(true);
      return;
    }

    startTransition(async () => {
      const previous = optimisticIsJoined;
      const next = !previous;
      setOptimisticIsJoined(next);

      const response = next
        ? await groupsService.join(groupId)
        : await groupsService.leave(groupId);

      if (!isSuccess(response)) {
        setOptimisticIsJoined(previous);
        toast.error('操作失败', response.message);
      }
    });
  };

  return (
    <>
      <Button
        variant={optimisticIsJoined ? 'outline' : 'default'}
        size="sm"
        onClick={handleClick}
        disabled={isPending}
      >
        {optimisticIsJoined ? (
          '已加入'
        ) : hasPassword ? (
          <>
            <Lock className="h-3 w-3 mr-1" />
            加入
          </>
        ) : (
          '加入'
        )}
      </Button>
      <GroupPasswordDialog
        open={passwordDialogOpen}
        onOpenChange={setPasswordDialogOpen}
        isPending={isPending}
        onSubmit={(password) => {
          startTransition(async () => {
            setOptimisticIsJoined(true);
            const response = await groupsService.join(groupId, password);
            if (isSuccess(response)) {
              setPasswordDialogOpen(false);
            } else {
              setOptimisticIsJoined(false);
              toast.error('加入失败', response.message);
            }
          });
        }}
      />
    </>
  );
}
