'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { useEffect } from 'react';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { UserService } from '@/lib/api/services/user';

const createSchema = (message: string) =>
  z.object({
    nickname: z.string().min(1, message),
    bio: z.string().optional(),
    tags: z.string().optional(),
  });

export default function AccountSettingsPage() {
  const t = useTranslations('settings');
  const tCommon = useTranslations('common');

  const { data: profile } = useQuery({
    queryKey: ['me-profile'],
    queryFn: async () => {
      const res = await UserService.getProfile();
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema(tCommon('empty'))),
    defaultValues: {
      nickname: profile?.nickname ?? '',
      bio: profile?.bio ?? '',
      tags: profile?.tags?.join(', ') ?? '',
    },
  });

  const { reset } = form;

  useEffect(() => {
    if (profile) {
      reset({
        nickname: profile.nickname ?? '',
        bio: profile.bio ?? '',
        tags: profile.tags?.join(', ') ?? '',
      });
    }
  }, [profile, reset]);

  const onSubmit = async (values: z.infer<ReturnType<typeof createSchema>>) => {
    const tags = values.tags?.split(',').map((tag) => tag.trim()).filter(Boolean);
    const res = await UserService.updateProfile({
      nickname: values.nickname,
      bio: values.bio,
      tags,
    });
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
          <CardTitle>{t('sections.account')}</CardTitle>
        </CardHeader>
        <CardContent>
          <form className="space-y-4" onSubmit={form.handleSubmit(onSubmit)}>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.nickname')}</label>
              <Input {...form.register('nickname')} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.bio')}</label>
              <Textarea {...form.register('bio')} className="min-h-[100px]" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.tags')}</label>
              <Input {...form.register('tags')} placeholder={t('labels.tagsPlaceholder')} />
            </div>
            <Button type="submit">{tCommon('save')}</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
