'use client';

/**
 * 通知订阅设置
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { BellRing } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { notificationsService } from '@/lib/api/services/notifications';
import { isSuccess } from '@/lib/api/client';
import type { SubscriptionsData } from '@/types/api';

const settingItems: Array<{
  key: keyof SubscriptionsData;
  label: string;
  description: string;
}> = [
  { key: 'new_follower_enabled', label: '新关注者通知', description: '有人关注你时通知' },
  { key: 'mutual_follow_enabled', label: '互相关注通知', description: '互相关注时通知' },
  { key: 'group_invite_enabled', label: '群邀请通知', description: '收到群组邀请时通知' },
  { key: 'group_invite_response_enabled', label: '群邀请响应', description: '群邀请被处理时通知' },
  { key: 'group_application_enabled', label: '入群申请通知', description: '收到入群申请时通知' },
  { key: 'group_application_response_enabled', label: '入群申请响应', description: '入群申请被处理时通知' },
  { key: 'message_mention_enabled', label: '消息提及通知', description: '聊天中被提及时通知' },
  { key: 'moment_like_enabled', label: '瞬间点赞通知', description: '瞬间被点赞时通知' },
  { key: 'moment_comment_enabled', label: '瞬间评论通知', description: '瞬间被评论时通知' },
  { key: 'comment_reply_enabled', label: '评论回复通知', description: '评论被回复时通知' },
  { key: 'system_announcement_enabled', label: '系统公告通知', description: '系统公告与提醒' },
];

export function NotificationSettings() {
  const { toast } = useToast();
  const { requireAuth, isAuthenticated } = useRequireAuth();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ['subscriptions'],
    queryFn: async () => {
      const response = await notificationsService.getSubscriptions();
      if (isSuccess(response)) {
        return response.data;
      }
      throw new Error(response.message);
    },
    enabled: isAuthenticated,
  });

  const mutation = useMutation({
    mutationFn: async (payload: Partial<SubscriptionsData>) => {
      const response = await notificationsService.updateSubscriptions(payload);
      if (!isSuccess(response)) {
        throw new Error(response.message);
      }
      return payload;
    },
    onSuccess: (payload) => {
      queryClient.setQueryData<SubscriptionsData>(['subscriptions'], (prev) =>
        prev ? { ...prev, ...payload } : prev
      );
      toast.success('设置已更新');
    },
    onError: (error) => {
      toast.error('更新失败', error instanceof Error ? error.message : '请稍后重试');
    },
  });

  if (!isAuthenticated) {
    return null;
  }

  return (
    <Card className="mt-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BellRing className="h-5 w-5" />
          通知订阅
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {isLoading && <div className="text-sm text-muted-foreground">加载中...</div>}
        {!isLoading &&
          data &&
          settingItems.map((item) => {
            const enabled = data[item.key];
            return (
              <div key={item.key} className="flex items-center justify-between gap-4">
                <div>
                  <p className="font-medium text-sm">{item.label}</p>
                  <p className="text-xs text-muted-foreground">{item.description}</p>
                </div>
                <Button
                  size="sm"
                  variant={enabled ? 'default' : 'outline'}
                  aria-pressed={enabled}
                  onClick={() => {
                    if (!requireAuth('修改通知设置')) return;
                    mutation.mutate({ [item.key]: !enabled });
                  }}
                >
                  {enabled ? '已开启' : '已关闭'}
                </Button>
              </div>
            );
          })}
      </CardContent>
    </Card>
  );
}
