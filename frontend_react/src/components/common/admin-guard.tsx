'use client';

import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { useAdminStore } from '@/stores/admin';

interface AdminGuardProps {
  children: React.ReactNode;
}

export function AdminGuard({ children }: AdminGuardProps) {
  const permissions = useAdminStore((state) => state.permissions);
  const t = useTranslations('admin');

  const isAdmin = permissions.includes('*') || permissions.length > 0;

  if (!isAdmin) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-muted-foreground">
          {t('noAccess')}
        </CardContent>
      </Card>
    );
  }

  return <>{children}</>;
}
