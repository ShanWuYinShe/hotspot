'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { AdminSettingsService } from '@/lib/api/services/admin/settings';

export default function AdminSettingsPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');

  const { data } = useQuery({
    queryKey: ['admin-settings'],
    queryFn: async () => {
      const res = await AdminSettingsService.fetch();
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <Card>
        <CardContent className="py-6">
          <div className="font-semibold">{tCommon('view')}</div>
          <p className="text-sm text-muted-foreground">{data?.updated_at ?? '-'}</p>
        </CardContent>
      </Card>
    </div>
  );
}
