'use client';

/**
 * 搜索表单组件 (Client Component)
 * 处理搜索输入和提交
 */

import { useEffect, useState, useTransition } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Search as SearchIcon, Users, UsersRound, Compass } from '@/components/icons';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useLocation } from '@/hooks/use-location';

type SearchTab = 'users' | 'groups' | 'moments';

interface SearchFormProps {
  defaultKeyword?: string;
  defaultTab?: SearchTab;
}

export function SearchForm({ defaultKeyword = '', defaultTab = 'users' }: SearchFormProps) {
  const t = useTranslations();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { referencePoint, refreshLocation } = useLocation();
  const [isPending, startTransition] = useTransition();
  
  const [keyword, setKeyword] = useState(defaultKeyword);
  const currentTab = (searchParams.get('tab') as SearchTab) || defaultTab;

  useEffect(() => {
    if (!referencePoint) {
      refreshLocation(false);
    }
  }, [referencePoint, refreshLocation]);

  const handleSearch = () => {
    if (!keyword.trim()) return;
    startTransition(() => {
      const params = new URLSearchParams(searchParams);
      params.set('q', keyword);
      params.set('tab', currentTab);
      if (referencePoint) {
        params.set('lat', String(referencePoint.latitude));
        params.set('lng', String(referencePoint.longitude));
      }
      router.push(`/search?${params.toString()}`);
    });
  };

  const handleTabChange = (tab: string) => {
    startTransition(() => {
      const params = new URLSearchParams(searchParams);
      params.set('tab', tab);
      if (keyword) params.set('q', keyword);
      if (referencePoint) {
        params.set('lat', String(referencePoint.latitude));
        params.set('lng', String(referencePoint.longitude));
      }
      router.push(`/search?${params.toString()}`);
    });
  };

  return (
    <div className="space-y-6">
      {/* Search Input */}
      <div className="flex gap-2">
        <div className="relative flex-1">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder={t('common.search')}
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            className="pl-9"
            disabled={isPending}
          />
        </div>
        <Button onClick={handleSearch} disabled={isPending || !keyword.trim()}>
          搜索
        </Button>
      </div>

      {/* Tabs */}
      <Tabs value={currentTab} onValueChange={handleTabChange}>
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="users" className="gap-2" disabled={isPending}>
            <Users className="h-4 w-4" />
            {t('users.title')}
          </TabsTrigger>
          <TabsTrigger value="groups" className="gap-2" disabled={isPending}>
            <UsersRound className="h-4 w-4" />
            {t('groups.title')}
          </TabsTrigger>
          <TabsTrigger value="moments" className="gap-2" disabled={isPending}>
            <Compass className="h-4 w-4" />
            {t('moments.title')}
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  );
}
