'use client';

import { useTranslations } from 'next-intl';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { toast } from 'sonner';

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { AuthService } from '@/lib/api/services/auth';

const createSchema = (message: string) =>
  z.object({
    old_password: z.string().min(1, message),
    new_password: z.string().min(6, message),
  });

export default function PasswordSettingsPage() {
  const t = useTranslations('settings');
  const tCommon = useTranslations('common');

  const form = useForm<z.infer<ReturnType<typeof createSchema>>>({
    resolver: zodResolver(createSchema(tCommon('empty'))),
    defaultValues: {
      old_password: '',
      new_password: '',
    },
  });

  const onSubmit = async (values: z.infer<ReturnType<typeof createSchema>>) => {
    const res = await AuthService.changePassword(values);
    if (res.code === 0) {
      toast.success(tCommon('saved'));
      form.reset();
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>{t('sections.password')}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.currentPassword')}</label>
              <Input type="password" {...form.register('old_password')} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('labels.newPassword')}</label>
              <Input type="password" {...form.register('new_password')} />
            </div>
            <Button type="submit">{tCommon('save')}</Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
