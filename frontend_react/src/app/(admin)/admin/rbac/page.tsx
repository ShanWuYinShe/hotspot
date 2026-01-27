'use client';

import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { AdminRbacService } from '@/lib/api/services/admin/rbac';

export default function AdminRbacPage() {
  const t = useTranslations('admin');
  const tCommon = useTranslations('common');

  const { data: roles } = useQuery({
    queryKey: ['admin-roles'],
    queryFn: async () => {
      const res = await AdminRbacService.roles();
      if (res.code !== 0) throw new Error(res.message);
      return res.data?.items ?? [];
    },
  });

  const { data: permissions } = useQuery({
    queryKey: ['admin-permissions'],
    queryFn: async () => {
      const res = await AdminRbacService.permissions();
      if (res.code !== 0) throw new Error(res.message);
      return res.data?.items ?? [];
    },
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('rbac')}</h1>
      <Card>
        <CardContent className="py-6">
          <div className="font-semibold">{t('rbac')}</div>
          <p className="text-sm text-muted-foreground">{roles?.length ?? 0} {tCommon('view')}</p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="py-6">
          <div className="font-semibold">{t('title')}</div>
          <p className="text-sm text-muted-foreground">{permissions?.length ?? 0} {tCommon('view')}</p>
        </CardContent>
      </Card>
    </div>
  );
}
