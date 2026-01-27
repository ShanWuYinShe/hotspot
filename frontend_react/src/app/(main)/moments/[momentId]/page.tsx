'use client';

import { useParams, useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { Loader2, ArrowLeft } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { MomentService } from '@/lib/api/services/moments';
import { Button } from '@/components/ui/button';
import { MomentCard } from '@/components/cards/moment-card';
import { FeedMomentItem } from '@/types/api';
import { CommentList } from '@/components/comments/comment-list';

export default function MomentDetailPage() {
  const params = useParams();
  const router = useRouter();
  const t = useTranslations('moments.detail');
  const tCommon = useTranslations('common');
  const momentId = params.momentId as string;

  const { data: moment, isLoading, error } = useQuery({
    queryKey: ['moment', momentId],
    queryFn: async () => {
      const res = await MomentService.getDetail(momentId);
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
  });

  if (isLoading) {
    return (
      <div className="flex h-[50vh] items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (error || !moment) {
    return (
      <div className="flex flex-col items-center justify-center py-12">
        <h2 className="text-xl font-semibold mb-2">{t('notFound')}</h2>
        <p className="text-muted-foreground mb-4">{tCommon('retry')}</p>
        <Button onClick={() => router.back()}>{tCommon('back')}</Button>
      </div>
    );
  }

  // Convert MomentCard to FeedMomentItem for reuse of MomentCard component
  // (In real app, might want separate components or unified type usage)
  const feedItem: FeedMomentItem = {
      ...moment,
      // Map MomentCard specific fields if necessary or ensuring types align
      is_liked: moment.my_state?.is_liked ?? false,
      is_favorited: moment.my_state?.is_favorited ?? false,
      // MomentCard has absolute Location, FeedMomentItem has RelativeLocation
      // We can adapt or just cast if the UI component handles it.
      // MomentCard component expects FeedMomentItem.
      // Location is compatible (RelativeLocation extends Location).
      // But we need to ensure 'distance' and 'bearing_degrees' are handled gracefully if missing.
      location: {
          ...moment.location,
          distance: 0, // Detail view usually implies we are "there" or distance is less relevant/unknown without reference
          bearing_degrees: 0
      }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="mb-4">
        <Button variant="ghost" size="sm" onClick={() => router.back()}>
          <ArrowLeft className="h-4 w-4 mr-2" />
          {tCommon('back')}
        </Button>
      </div>
      
      <MomentCard moment={feedItem} />
      
      <div className="mt-8">
        <CommentList momentId={momentId} />
      </div>
    </div>
  );
}
