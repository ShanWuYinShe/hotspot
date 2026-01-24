import { Suspense } from 'react';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { FeedTabs } from './feed/feed-tabs';
import { FeedList } from './feed/feed-list';
import { FeedSkeleton } from '@/components/skeletons/feed-skeleton';
import { FeedError } from './feed-error';
import { PageHeader } from '@/components/layout';
import { CreateMomentButton } from '@/components/moments';
import type { FeedSource, MomentItem, PaginatedData } from '@/types/api';

interface HomePageProps {
  searchParams: Promise<{ source?: FeedSource }>;
}



async function FeedContent({ source }: { source: FeedSource }) {
  const response = await serverApi.post<PaginatedData<MomentItem>>(
    ENDPOINTS.FEED.MOMENTS,
    { source, limit: 20 },
    {
      tags: ['feed'],
      revalidate: 60,
    }
  );

  if (!isServerSuccess(response)) {
    return <FeedError error={response.message} />;
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
    <div className="mx-auto max-w-2xl space-y-6">
      <PageHeader
        title="动态"
        description="发现身边和你关注的精彩内容"
        actions={<CreateMomentButton label="发布瞬间" />}
      >
        <FeedTabs defaultSource={source} />
      </PageHeader>

      <Suspense fallback={<FeedSkeleton />}>
        <FeedContent source={source} />
      </Suspense>
    </div>
  );
}
