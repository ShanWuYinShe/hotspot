'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { SystemService } from '@/lib/api/services/system';

export default function HealthPage() {
  const t = useTranslations('health');
  const tCommon = useTranslations('common');

  const { data, isLoading } = useQuery({
    queryKey: ['system-health'],
    queryFn: async () => {
      const res = await SystemService.health();
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <Card>
        <CardContent className="py-12 text-center text-muted-foreground">
          {isLoading ? tCommon('loading') : `${t('status')}: ${data?.status ?? '-'}`}
        </CardContent>
      </Card>
    </div>
  );
}
