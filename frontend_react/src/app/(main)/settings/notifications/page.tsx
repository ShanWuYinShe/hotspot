'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { useEffect } from 'react';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Checkbox } from '@/components/ui/checkbox';
import { Button } from '@/components/ui/button';
import { SubscriptionService } from '@/lib/api/services/notifications';

const createSchema = () =>
  z.object({
    new_follower_enabled: z.boolean(),
    mutual_follow_enabled: z.boolean(),
    follow_request_enabled: z.boolean(),
    follow_request_handled_enabled: z.boolean(),
    group_invite_enabled: z.boolean(),
    group_invite_response_enabled: z.boolean(),
    group_application_enabled: z.boolean(),
    group_application_response_enabled: z.boolean(),
    group_announcement_enabled: z.boolean(),
    message_mention_enabled: z.boolean(),
    moment_like_enabled: z.boolean(),
    moment_comment_enabled: z.boolean(),
    comment_reply_enabled: z.boolean(),
    system_announcement_enabled: z.boolean(),
  });

export default function NotificationSettingsPage() {
  const t = useTranslations('settings');
  const tCommon = useTranslations('common');

  const { data: subscriptions } = useQuery({
    queryKey: ['subscriptions'],
    queryFn: async () => {
      const res = await SubscriptionService.get();
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema()),
    defaultValues: {
      new_follower_enabled: true,
      mutual_follow_enabled: true,
      follow_request_enabled: true,
      follow_request_handled_enabled: true,
      group_invite_enabled: true,
      group_invite_response_enabled: true,
      group_application_enabled: true,
      group_application_response_enabled: true,
      group_announcement_enabled: true,
      message_mention_enabled: true,
      moment_like_enabled: true,
      moment_comment_enabled: true,
      comment_reply_enabled: true,
      system_announcement_enabled: true,
    },
  });

  const { reset } = form;

  useEffect(() => {
    if (subscriptions) {
      reset({
        new_follower_enabled: subscriptions.new_follower_enabled ?? true,
        mutual_follow_enabled: subscriptions.mutual_follow_enabled ?? true,
        follow_request_enabled: subscriptions.follow_request_enabled ?? true,
        follow_request_handled_enabled: subscriptions.follow_request_handled_enabled ?? true,
        group_invite_enabled: subscriptions.group_invite_enabled ?? true,
        group_invite_response_enabled: subscriptions.group_invite_response_enabled ?? true,
        group_application_enabled: subscriptions.group_application_enabled ?? true,
        group_application_response_enabled: subscriptions.group_application_response_enabled ?? true,
        group_announcement_enabled: subscriptions.group_announcement_enabled ?? true,
        message_mention_enabled: subscriptions.message_mention_enabled ?? true,
        moment_like_enabled: subscriptions.moment_like_enabled ?? true,
        moment_comment_enabled: subscriptions.moment_comment_enabled ?? true,
        comment_reply_enabled: subscriptions.comment_reply_enabled ?? true,
        system_announcement_enabled: subscriptions.system_announcement_enabled ?? true,
      });
    }
  }, [reset, subscriptions]);

  const onSubmit = async (values: z.infer<ReturnType<typeof createSchema>>) => {
    const res = await SubscriptionService.update(values);
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const renderOption = (name: keyof z.infer<ReturnType<typeof createSchema>>, label: string) => (
    <div className="flex items-center gap-3">
      <Checkbox
        checked={form.watch(name)}
        onCheckedChange={(value) => form.setValue(name, Boolean(value))}
      />
      <span className="text-sm">{label}</span>
    </div>
  );

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('sections.notifications')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-3">
            {renderOption('new_follower_enabled', t('notificationOptions.newFollower'))}
            {renderOption('mutual_follow_enabled', t('notificationOptions.mutualFollow'))}
            {renderOption('follow_request_enabled', t('notificationOptions.followRequest'))}
            {renderOption('follow_request_handled_enabled', t('notificationOptions.followRequestHandled'))}
            {renderOption('group_invite_enabled', t('notificationOptions.groupInvite'))}
            {renderOption('group_invite_response_enabled', t('notificationOptions.groupInviteResponse'))}
            {renderOption('group_application_enabled', t('notificationOptions.groupApplication'))}
            {renderOption('group_application_response_enabled', t('notificationOptions.groupApplicationResponse'))}
            {renderOption('group_announcement_enabled', t('notificationOptions.groupAnnouncement'))}
            {renderOption('message_mention_enabled', t('notificationOptions.messageMention'))}
            {renderOption('moment_like_enabled', t('notificationOptions.momentLike'))}
            {renderOption('moment_comment_enabled', t('notificationOptions.momentComment'))}
            {renderOption('comment_reply_enabled', t('notificationOptions.commentReply'))}
            {renderOption('system_announcement_enabled', t('notificationOptions.systemAnnouncement'))}
            <Button type="submit" className="mt-4">
              {tCommon('save')}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
