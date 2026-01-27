'use client';

import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';
import { useRouter } from 'next/navigation';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { GroupService } from '@/lib/api/services/groups';
import { useLocation } from '@/hooks/use-location';

const createSchema = (message: string) =>
  z.object({
    name: z.string().min(1, message),
    description: z.string().optional(),
    tags: z.string().optional(),
    join_method: z.enum(['open', 'approval', 'invite_only']),
  });

export default function CreateGroupPage() {
  const t = useTranslations('groups');
  const tCommon = useTranslations('common');
  const router = useRouter();
  const { currentLocation, requestLocation } = useLocation();

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema(tCommon('empty'))),
    defaultValues: {
      name: '',
      description: '',
      tags: '',
      join_method: 'open',
    },
  });

  const onSubmit = async (values: z.infer<ReturnType<typeof createSchema>>) => {
    if (!currentLocation) {
      toast.error(tCommon('retry'));
      return;
    }
    const tags = values.tags?.split(',').map((tag) => tag.trim()).filter(Boolean);
    const res = await GroupService.create({
      name: values.name,
      description: values.description,
      tags,
      location: currentLocation,
      join_method: values.join_method,
    });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
      router.push('/groups');
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('create')}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('name')}</label>
              <Input {...form.register('name')} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('description')}</label>
              <Textarea {...form.register('description')} className="min-h-[120px]" />
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
            {!currentLocation ? (
              <Button type="button" variant="outline" onClick={requestLocation}>
                {tCommon('confirm')}
              </Button>
            ) : (
              <p className="text-sm text-muted-foreground">
                {currentLocation.location_name || tCommon('loading')}
              </p>
            )}
            <Button type="submit">{tCommon('save')}</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
