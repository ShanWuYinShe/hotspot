'use client';

import * as React from 'react';
import { useTransition } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useRequireAuth } from '@/hooks/use-require-auth';
import type { FeedSource } from '@/types/api';

interface FeedTabsProps {
  defaultSource?: FeedSource;
}

export function FeedTabs({ defaultSource = 'all' }: FeedTabsProps) {
  const t = useTranslations('feed');
  const router = useRouter();
  const searchParams = useSearchParams();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  const source = (searchParams.get('source') as FeedSource) || defaultSource;

  const handleSourceChange = (newSource: string) => {
    if (newSource === 'following' && !requireAuth('查看关注动态')) {
      return;
    }
    startTransition(() => {
      const params = new URLSearchParams(searchParams);
      params.set('source', newSource);
      router.push(`/feed?${params.toString()}`);
    });
  };

  return (
    <Tabs value={source} onValueChange={handleSourceChange}>
      <TabsList className="mb-4 rounded-2xl bg-muted/60 p-1.5 border-2 border-border">
        <TabsTrigger value="all" disabled={isPending} className="relative rounded-xl px-5 data-[state=active]:bg-background data-[state=active]:shadow-md transition-all duration-200">
          <span className="flex items-center gap-2">
            <span className="text-sm font-semibold">{t('trending')}</span>
          </span>
        </TabsTrigger>
        <TabsTrigger value="following" disabled={isPending} className="relative rounded-xl px-5 data-[state=active]:bg-background data-[state=active]:shadow-md transition-all duration-200">
          <span className="flex items-center gap-2">
            <span className="text-sm font-semibold">{t('following')}</span>
          </span>
        </TabsTrigger>
        <TabsTrigger value="nearby" disabled={isPending} className="relative rounded-xl px-5 data-[state=active]:bg-background data-[state=active]:shadow-md transition-all duration-200">
          <span className="flex items-center gap-2">
            <span className="text-sm font-semibold">{t('nearby')}</span>
          </span>
        </TabsTrigger>
      </TabsList>
    </Tabs>
  );
}
