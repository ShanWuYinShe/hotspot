/**
 * 瞬间详情页面 (Server Component)
 * 服务端获取瞬间详情和评论
 */

import { Suspense } from 'react';
import { notFound } from 'next/navigation';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { LoadingState } from '@/components/common/loading-state';
import { MomentDetail } from './moment-detail';
import { CommentSection } from './comment-section';
import type { MomentItem, CommentListItem, PaginatedData } from '@/types/api';

interface MomentPageProps {
  params: Promise<{ momentId: string }>;
}

async function MomentContent({ momentId }: { momentId: string }) {
  const [momentRes, commentsRes] = await Promise.all([
    serverApi.post<MomentItem>(
      ENDPOINTS.MOMENTS.DETAIL(momentId),
      undefined,
      { tags: [`moment-${momentId}`], revalidate: 60 }
    ),
    serverApi.get<PaginatedData<CommentListItem>>(
      ENDPOINTS.MOMENTS.COMMENTS(momentId),
      { limit: 20 },
      { tags: [`comments-${momentId}`], revalidate: 30 }
    ),
  ]);

  if (!isServerSuccess(momentRes)) {
    notFound();
  }

  const moment = momentRes.data;
  const comments = isServerSuccess(commentsRes) ? commentsRes.data?.items ?? [] : [];
  const commentsPagination = isServerSuccess(commentsRes) ? commentsRes.data?.pagination : undefined;

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <MomentDetail moment={moment} />
      <CommentSection 
        momentId={momentId} 
        initialComments={comments} 
        commentsCount={moment.comments_count}
        initialPagination={commentsPagination}
      />
    </div>
  );
}

export default async function MomentPage({ params }: MomentPageProps) {
  const { momentId } = await params;

  return (
    <Suspense fallback={<LoadingState text="加载中..." />}>
      <MomentContent momentId={momentId} />
    </Suspense>
  );
}
