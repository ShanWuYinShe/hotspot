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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { UserService } from '@/lib/api/services/user';

const createSchema = () =>
  z.object({
    privacy_level: z.enum(['public', 'friends', 'private']),
    location_visible: z.boolean(),
    online_visible: z.boolean(),
    last_online_visible: z.boolean(),
    allow_follow: z.boolean(),
    allow_private_chat: z.enum(['everyone', 'friends', 'nobody']),
  });

export default function PrivacySettingsPage() {
  const t = useTranslations('settings');
  const tCommon = useTranslations('common');

  const { data: settings } = useQuery({
    queryKey: ['user-settings'],
    queryFn: async () => {
      const res = await UserService.mySettings();
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema()),
    defaultValues: {
      privacy_level: 'public',
      location_visible: true,
      online_visible: true,
      last_online_visible: true,
      allow_follow: true,
      allow_private_chat: 'friends',
    },
  });

  const { reset } = form;

  useEffect(() => {
    if (settings) {
      reset({
        privacy_level: settings.privacy_level ?? 'public',
        location_visible: settings.location_visible ?? true,
        online_visible: settings.online_visible ?? true,
        last_online_visible: settings.last_online_visible ?? true,
        allow_follow: settings.allow_follow ?? true,
        allow_private_chat: settings.allow_private_chat ?? 'friends',
      });
    }
  }, [reset, settings]);

  const onSubmit = async (values: z.infer<ReturnType<typeof createSchema>>) => {
    const res = await UserService.updateMySettings(values);
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('sections.privacy')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.privacyLevel')}</label>
              <Select
                value={form.watch('privacy_level')}
                onValueChange={(value) => form.setValue('privacy_level', value as 'public' | 'friends' | 'private')}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="public">{t('options.privacy.public')}</SelectItem>
                  <SelectItem value="friends">{t('options.privacy.friends')}</SelectItem>
                  <SelectItem value="private">{t('options.privacy.private')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center gap-3">
              <Checkbox
                checked={form.watch('location_visible')}
                onCheckedChange={(value) => form.setValue('location_visible', Boolean(value))}
              />
              <span className="text-sm">{t('labels.locationVisible')}</span>
            </div>

            <div className="flex items-center gap-3">
              <Checkbox
                checked={form.watch('online_visible')}
                onCheckedChange={(value) => form.setValue('online_visible', Boolean(value))}
              />
              <span className="text-sm">{t('labels.onlineVisible')}</span>
            </div>

            <div className="flex items-center gap-3">
              <Checkbox
                checked={form.watch('last_online_visible')}
                onCheckedChange={(value) => form.setValue('last_online_visible', Boolean(value))}
              />
              <span className="text-sm">{t('labels.lastOnlineVisible')}</span>
            </div>

            <div className="flex items-center gap-3">
              <Checkbox
                checked={form.watch('allow_follow')}
                onCheckedChange={(value) => form.setValue('allow_follow', Boolean(value))}
              />
              <span className="text-sm">{t('labels.allowFollow')}</span>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.allowPrivateChat')}</label>
              <Select
                value={form.watch('allow_private_chat')}
                onValueChange={(value) => form.setValue('allow_private_chat', value as 'everyone' | 'friends' | 'nobody')}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="everyone">{t('options.chat.everyone')}</SelectItem>
                  <SelectItem value="friends">{t('options.chat.friends')}</SelectItem>
                  <SelectItem value="nobody">{t('options.chat.nobody')}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Button type="submit">{tCommon('save')}</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
