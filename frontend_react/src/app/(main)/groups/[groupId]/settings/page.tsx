'use client';

import { useParams } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { useEffect } from 'react';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { GroupService } from '@/lib/api/services/groups';

const createSchema = (message: string) =>
  z.object({
    name: z.string().min(1, message),
    description: z.string().optional(),
    tags: z.string().optional(),
    join_method: z.enum(['open', 'approval', 'invite_only']),
  });

export default function GroupSettingsPage() {
  const params = useParams();
  const groupId = params.groupId as string;
  const t = useTranslations('groups');
  const tCommon = useTranslations('common');

  const { data: group } = useQuery({
    queryKey: ['group-detail', groupId],
    queryFn: async () => {
      const res = await GroupService.openTheDoor(groupId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!groupId,
  });

  const { data: settings } = useQuery({
    queryKey: ['group-settings', groupId],
    queryFn: async () => {
      const res = await GroupService.settings(groupId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!groupId,
  });

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema(tCommon('empty'))),
    defaultValues: {
      name: '',
      description: '',
      tags: '',
      join_method: 'open',
    },
  });

  const { reset } = form;

  useEffect(() => {
    if (group || settings) {
      reset({
        name: group?.name ?? '',
        description: group?.description ?? '',
        tags: group?.tags?.join(', ') ?? '',
        join_method: settings?.settings.join_method ?? 'open',
      });
    }
  }, [group, reset, settings]);

  const onSubmit = async (values: z.infer<ReturnType<typeof createSchema>>) => {
    const tags = values.tags?.split(',').map((tag) => tag.trim()).filter(Boolean);
    const res = await GroupService.update({
      group_chat_id: groupId,
      name: values.name,
      description: values.description,
      tags,
      join_method: values.join_method,
    });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('settings')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('name')}</label>
              <Input {...form.register('name')} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('description')}</label>
              <Textarea {...form.register('description')} className="min-h-[100px]" />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('tags')}</label>
              <Input {...form.register('tags')} placeholder={t('tagsPlaceholder')} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('joinMethod')}</label>
              <Select
                value={form.watch('join_method')}
                onValueChange={(value) =>
                  form.setValue('join_method', value as 'open' | 'approval' | 'invite_only')
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder={t('joinMethod')} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="open">{t('joinMethodOpen')}</SelectItem>
                  <SelectItem value="approval">{t('joinMethodApproval')}</SelectItem>
                  <SelectItem value="invite_only">{t('joinMethodInvite')}</SelectItem>
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
