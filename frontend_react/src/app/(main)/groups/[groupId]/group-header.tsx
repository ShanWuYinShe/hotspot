'use client';

/**
 * 群组头部 (Client Component)
 * 处理加入/退出群组交互
 */

import { useState, useTransition, useOptimistic } from 'react';
import { useRouter } from 'next/navigation';
import { MapPin, Users, Lock, MessageCircle, Settings, LogOut } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { GroupPasswordDialog } from '@/components/common/group-password-dialog';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { formatDistance, formatDateTime } from '@/lib/utils';
import { groupsService } from '@/lib/api/services/groups';
import { isSuccess } from '@/lib/api/client';
import type { GroupDetailData } from '@/types/api';

interface GroupHeaderProps {
  group: GroupDetailData;
}

export function GroupHeader({ group }: GroupHeaderProps) {
  const router = useRouter();
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();
  
  const isJoined = group.is_joined ?? group.my_state?.is_joined ?? false;
  const isOwner = group.my_role === 'owner';
  const isAdmin = group.my_role === 'admin';
  
  const [optimisticJoined, setOptimisticJoined] = useOptimistic(isJoined);
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);

  const initials = group.name.slice(0, 2).toUpperCase();

  const handleJoinOrLeave = () => {
    if (!requireAuth(optimisticJoined ? '退出群组' : '加入群组')) {
      return;
    }

    if (group.has_password && !optimisticJoined) {
      setPasswordDialogOpen(true);
      return;
    }

    startTransition(async () => {
      const previous = optimisticJoined;
      const next = !previous;
      setOptimisticJoined(next);
      const response = next
        ? await groupsService.join(group.group_chat_id)
        : await groupsService.leave(group.group_chat_id);
      if (!isSuccess(response)) {
        setOptimisticJoined(previous);
        toast.error('操作失败', response.message);
      }
    });
  };

  const handleChat = () => {
    if (!requireAuth('进入群聊')) {
      return;
    }
    router.push(`/chat/groups/${group.group_chat_id}`);
  };

  const handleSettings = () => {
    if (!requireAuth('管理群组')) {
      return;
    }
    router.push(`/groups/${group.group_chat_id}/settings`);
  };

  return (
    <Card>
      <CardContent className="p-6">
        <div className="flex items-start gap-6">
          <Avatar className="h-20 w-20">
            {group.avatar_url && <AvatarImage src={group.avatar_url} alt={group.name} />}
            <AvatarFallback className="text-2xl">{initials}</AvatarFallback>
          </Avatar>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-2xl font-bold truncate">{group.name}</h1>
              {group.has_password && (
                <Lock className="h-5 w-5 text-muted-foreground" />
              )}
              {(isOwner || isAdmin) && (
                <Badge variant="secondary">
                  {isOwner ? '群主' : '管理员'}
                </Badge>
              )}
            </div>

            {group.description && (
              <p className="text-muted-foreground mb-3">{group.description}</p>
            )}

            {/* 标签 */}
            {group.tags && group.tags.length > 0 && (
              <div className="flex flex-wrap gap-1 mb-3">
                {group.tags.map((tag) => (
                  <Badge key={tag} variant="outline">
                    {tag}
                  </Badge>
                ))}
              </div>
            )}

            {/* 元信息 */}
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <span className="flex items-center gap-1">
                <Users className="h-4 w-4" />
                {group.members_count} 成员
              </span>
              {group.location && (
                <span className="flex items-center gap-1">
                  <MapPin className="h-4 w-4" />
                  {formatDistance(group.location.distance)}
                </span>
              )}
              <span>创建于 {formatDateTime(group.created_at)}</span>
            </div>
          </div>

          {/* 操作按钮 */}
          <div className="flex gap-2">
            {optimisticJoined ? (
              <>
                <Button onClick={handleChat}>
                  <MessageCircle className="h-4 w-4 mr-2" />
                  进入群聊
                </Button>
                {(isOwner || isAdmin) && (
                  <Button variant="outline" onClick={handleSettings} aria-label="群组设置">
                    <Settings className="h-4 w-4" />
                  </Button>
                )}
                {!isOwner && (
                  <Button 
                    variant="outline" 
                    onClick={handleJoinOrLeave}
                    disabled={isPending}
                  >
                    <LogOut className="h-4 w-4 mr-2" />
                    退出
                  </Button>
                )}
              </>
            ) : (
              <Button onClick={handleJoinOrLeave} disabled={isPending}>
                {group.has_password ? (
                  <>
                    <Lock className="h-4 w-4 mr-2" />
                    申请加入
                  </>
                ) : (
                  '加入群组'
                )}
              </Button>
            )}
          </div>
        </div>
        <GroupPasswordDialog
          open={passwordDialogOpen}
          onOpenChange={setPasswordDialogOpen}
          isPending={isPending}
          onSubmit={(password) => {
            startTransition(async () => {
              setOptimisticJoined(true);
              const response = await groupsService.join(group.group_chat_id, password);
              if (isSuccess(response)) {
                setPasswordDialogOpen(false);
              } else {
                setOptimisticJoined(false);
                toast.error('加入失败', response.message);
              }
            });
          }}
        />
      </CardContent>
    </Card>
  );
}
