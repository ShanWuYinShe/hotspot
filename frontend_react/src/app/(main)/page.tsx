import { Suspense } from 'react';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { FeedTabs } from './feed/feed-tabs';
import { FeedList } from './feed/feed-list';
import { FeedSkeleton } from '@/components/skeletons/feed-skeleton';
import { AlertTriangle } from '@/components/icons';
import { Button } from '@/components/ui/button';
import type { FeedSource, MomentItem, PaginatedData } from '@/types/api';

interface HomePageProps {
  searchParams: Promise<{ source?: FeedSource }>;
}

interface FeedErrorProps {
  error?: string;
  onRetry: () => void;
}

function FeedError({ error, onRetry }: FeedErrorProps) {
  return (
    <div className="flex min-h-[400px] items-center justify-center">
      <div className="max-w-md w-full space-y-6 rounded-2xl border bg-card p-8 text-center shadow-lg">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
          <AlertTriangle className="h-8 w-8 text-destructive" />
        </div>
        <div className="space-y-2">
          <h2 className="text-xl font-semibold">加载动态失败</h2>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {error || '无法获取动态信息，可能是网络连接问题或服务器暂时不可用。'}
          </p>
        </div>
        <Button onClick={onRetry} className="w-full" size="lg">
          重试
        </Button>
      </div>
    </div>
  );
}

async function FeedContent({ source }: { source: FeedSource }) {
  const response = await serverApi.get<PaginatedData<MomentItem>>(
    ENDPOINTS.FEED.MOMENTS,
    { source, limit: 20 },
    {
      tags: ['feed'],
      revalidate: 60,
    }
  );

  if (!isServerSuccess(response)) {
    return <FeedError error={response.message} onRetry={() => window.location.reload()} />;
  }

  const moments = response.data?.items ?? [];
  const pagination = response.data?.pagination ?? null;

  return (
    <FeedList
      initialMoments={moments}
      hasMore={pagination?.has_more ?? false}
      nextCursor={pagination?.next_cursor}
      source={source}
    />
  );
}

export default async function HomePage({ searchParams }: HomePageProps) {
  const params = await searchParams;
  const source = params.source || 'all';

  return (
    <div className="min-h-screen">
      <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <FeedTabs defaultSource={source} />

        <Suspense fallback={<FeedSkeleton />}>
          <FeedContent source={source} />
        </Suspense>
      </div>
    </div>
  );
}
