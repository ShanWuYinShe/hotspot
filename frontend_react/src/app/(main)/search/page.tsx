/**
 * 搜索页面 (Server Component)
 * 服务端渲染结构和初始数据
 */

import { Suspense } from 'react';
import { LoadingState } from '@/components/common/loading-state';
import { SearchForm } from './search-form';
import { SearchResults } from './search-results';

interface SearchPageProps {
  searchParams: Promise<{
    q?: string;
    tab?: 'users' | 'groups' | 'moments';
    lat?: string;
    lng?: string;
  }>;
}

export default async function SearchPage({ searchParams }: SearchPageProps) {
  const params = await searchParams;
  const keyword = params.q || '';
  const tab = params.tab || 'users';
  const latitude = params.lat ? Number(params.lat) : undefined;
  const longitude = params.lng ? Number(params.lng) : undefined;
  const referencePoint =
    Number.isFinite(latitude) && Number.isFinite(longitude)
      ? { latitude: latitude as number, longitude: longitude as number }
      : undefined;

  return (
    <div className="max-w-2xl mx-auto">
      <SearchForm defaultKeyword={keyword} defaultTab={tab} />
      
      <div className="mt-6">
        <Suspense fallback={<LoadingState text="搜索中..." />}>
          <SearchResults keyword={keyword} tab={tab} referencePoint={referencePoint} />
        </Suspense>
      </div>
    </div>
  );
}
