'use client';

import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';

export default function AdminHomePage() {
  const t = useTranslations('admin');

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">{t('users')}</CardContent>
        </Card>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">{t('groups')}</CardContent>
        </Card>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">{t('reports')}</CardContent>
        </Card>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">{t('audit')}</CardContent>
        </Card>
      </div>
    </div>
  );
}
