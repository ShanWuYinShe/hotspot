'use client';

import * as React from 'react';
import Link from 'next/link';
import { useInfiniteQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import * as notificationsService from '@/lib/api/services/notifications';
import { isSuccess } from '@/lib/api/client';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  Loader2,
  Bell,
  Heart,
  MessageCircle,
  UserPlus,
  Users,
  Megaphone,
  CheckCheck,
  AtSign,
} from 'lucide-react';
import type { NotificationItem, NotificationFilterType } from '@/types/api';

const filterTabs: { value: NotificationFilterType | 'all'; label: string; icon: React.ElementType }[] = [
  { value: 'all', label: '全部', icon: Bell },
  { value: 'like', label: '赞', icon: Heart },
  { value: 'comment', label: '评论', icon: MessageCircle },
  { value: 'follow', label: '关注', icon: UserPlus },
  { value: 'mention', label: '@我', icon: AtSign },
  { value: 'system', label: '系统', icon: Megaphone },
];

export default function NotificationsPage() {
  const queryClient = useQueryClient();
  const [activeFilter, setActiveFilter] = React.useState<NotificationFilterType | 'all'>('all');

  // Fetch notifications
  const notificationsQuery = useInfiniteQuery({
    queryKey: ['notifications', activeFilter],
    queryFn: async ({ pageParam }) => {
      const res = await notificationsService.getNotifications({
        filter: activeFilter === 'all' ? undefined : activeFilter,
        cursor: pageParam as string | undefined,
        limit: 20,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage?.pagination.has_more ? lastPage.pagination.next_cursor : undefined,
  });

  // Mark all as read mutation
  const markAllReadMutation = useMutation({
    mutationFn: async () => {
      const res = await notificationsService.markNotificationsRead({ mark_all: true });
      if (!isSuccess(res)) throw new Error(res.message);
      return res;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] });
    },
  });

  const notifications = notificationsQuery.data?.pages.flatMap((page) => page?.items ?? []) ?? [];
  const hasUnread = notifications.some((n) => !n.is_read);

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">通知</h1>
          <p className="text-muted-foreground text-sm mt-1">
            查看所有动态
          </p>
        </div>
        {hasUnread && (
          <Button
            variant="ghost"
            size="sm"
            onClick={() => markAllReadMutation.mutate()}
            disabled={markAllReadMutation.isPending}
            className="gap-2"
          >
            {markAllReadMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <CheckCheck className="h-4 w-4" />
            )}
            全部已读
          </Button>
        )}
      </div>

      {/* Filter Tabs */}
      <div className="flex items-center gap-2 overflow-x-auto pb-2">
        {filterTabs.map((tab) => (
          <button
            key={tab.value}
            onClick={() => setActiveFilter(tab.value)}
            className={cn(
              'flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium transition-all whitespace-nowrap',
              activeFilter === tab.value
                ? 'bg-primary text-primary-foreground'
                : 'bg-muted/50 text-muted-foreground hover:bg-muted'
            )}
          >
            <tab.icon className="h-4 w-4" />
            {tab.label}
          </button>
        ))}
      </div>

      {/* Notifications List */}
      {notificationsQuery.isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : notifications.length === 0 ? (
        <Card className="shadow-soft">
          <CardContent className="py-12 text-center">
            <Bell className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
            <h3 className="text-lg font-medium mb-2">暂无通知</h3>
            <p className="text-muted-foreground text-sm">
              {activeFilter === 'all'
                ? '当有新动态时会在这里显示'
                : '该类型暂无通知'}
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          {notifications.map((notification) => (
            <NotificationCard key={notification.notification_id} notification={notification} />
          ))}

          {/* Load More */}
          {notificationsQuery.hasNextPage && (
            <div className="flex justify-center pt-4">
              <Button
                variant="outline"
                onClick={() => notificationsQuery.fetchNextPage()}
                disabled={notificationsQuery.isFetchingNextPage}
                className="rounded-full"
              >
                {notificationsQuery.isFetchingNextPage ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    加载中...
                  </>
                ) : (
                  '加载更多'
                )}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

interface NotificationCardProps {
  notification: NotificationItem;
}

function NotificationCard({ notification }: NotificationCardProps) {
  const getIcon = () => {
    switch (notification.type) {
      case 'moment_like':
        return <Heart className="h-4 w-4 text-red-500" />;
      case 'moment_comment':
      case 'comment_reply':
        return <MessageCircle className="h-4 w-4 text-blue-500" />;
      case 'new_follower':
      case 'mutual_follow':
      case 'follow_request':
      case 'follow_request_handled':
        return <UserPlus className="h-4 w-4 text-green-500" />;
      case 'group_invite':
      case 'group_invite_response':
      case 'group_application':
      case 'group_application_response':
      case 'group_announcement':
        return <Users className="h-4 w-4 text-purple-500" />;
      case 'system_announcement':
        return <Megaphone className="h-4 w-4 text-orange-500" />;
      case 'message_mention':
        return <AtSign className="h-4 w-4 text-cyan-500" />;
      default:
        return <Bell className="h-4 w-4 text-muted-foreground" />;
    }
  };

  const getLink = () => {
    if (!notification.target_type || !notification.target_id) return null;
    
    switch (notification.target_type) {
      case 'moment':
        return `/moments/${notification.target_id}`;
      case 'user':
        return `/users/${notification.target_id}`;
      case 'group':
        return `/groups/${notification.target_id}`;
      case 'web':
        return notification.target_id;
      default:
        return null;
    }
  };

  const getMessage = () => {
    switch (notification.type) {
      case 'moment_like':
        return '赞了你的瞬间';
      case 'moment_comment':
        return '评论了你的瞬间';
      case 'comment_reply':
        return '回复了你的评论';
      case 'new_follower':
        return '关注了你';
      case 'mutual_follow':
        return '你们已互相关注';
      case 'follow_request':
        return '请求关注你';
      case 'follow_request_handled':
        return notification.content || '处理了你的关注请求';
      case 'group_invite':
        return '邀请你加入群组';
      case 'group_invite_response':
        return notification.content || '处理了你的群组邀请';
      case 'group_application':
        return '申请加入群组';
      case 'group_application_response':
        return notification.content || '处理了你的群组申请';
      case 'group_announcement':
        return '发布了群公告';
      case 'system_announcement':
        return notification.title || '系统通知';
      case 'message_mention':
        return '在消息中提到了你';
      default:
        return notification.content || '新通知';
    }
  };

  const link = getLink();

  const containerClassName = cn(
    'flex items-start gap-3 p-4 rounded-xl transition-colors',
    notification.is_read
      ? 'bg-background hover:bg-muted/50'
      : 'bg-primary/5 hover:bg-primary/10',
    link && 'cursor-pointer'
  );

  const content = (
    <>
      {/* Sender Avatar or Icon */}
      {notification.sender ? (
        <Avatar className="h-10 w-10 shrink-0">
          <AvatarImage src={notification.sender.avatar_url} alt={notification.sender.nickname} />
          <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-sm">
            {notification.sender.nickname.slice(0, 2).toUpperCase()}
          </AvatarFallback>
        </Avatar>
      ) : (
        <div className="h-10 w-10 rounded-full bg-muted flex items-center justify-center shrink-0">
          {getIcon()}
        </div>
      )}

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-start justify-between gap-2">
          <div>
            {notification.sender && (
              <span className="font-medium">{notification.sender.nickname} </span>
            )}
            <span className="text-muted-foreground">{getMessage()}</span>
          </div>
          {!notification.is_read && (
            <span className="w-2 h-2 rounded-full bg-primary shrink-0 mt-2" />
          )}
        </div>
        
        {/* Preview */}
        {notification.target_text_preview && (
          <p className="text-sm text-muted-foreground mt-1 line-clamp-2">
            {notification.target_text_preview}
          </p>
        )}
        
        {/* Media Preview */}
        {notification.target_media_url && (
          <div className="mt-2 w-12 h-12 rounded-lg bg-muted overflow-hidden">
            <img
              src={notification.target_media_url}
              alt=""
              className="w-full h-full object-cover"
            />
          </div>
        )}
        
        {/* Time */}
        <p className="text-xs text-muted-foreground mt-2">
          {new Date(notification.created_at).toLocaleString('zh-CN')}
        </p>
      </div>
    </>
  );

  if (link) {
    return (
      <Link href={link} className={containerClassName}>
        {content}
      </Link>
    );
  }

  return <div className={containerClassName}>{content}</div>;
}
