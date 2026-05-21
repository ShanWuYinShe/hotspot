import { useState, useEffect, useCallback } from 'react';
import { feedService } from '@/lib/api/services/feed';
import { useLocationStore } from '@/lib/stores/location';
import { isSuccess } from '@/lib/api/client';
import type { MomentItem, FeedSource } from '@/lib/types/api';

interface DiscoverData {
  moments: MomentItem[];
  loading: boolean;
  error: string | null;
  hasMore: boolean;
  loadMore: () => void;
  refresh: () => void;
}

export function useDiscoverData(source: FeedSource): DiscoverData {
  const [moments, setMoments] = useState<MomentItem[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const geoPoint = useLocationStore((s) => s.geoPoint);

  const load = useCallback(async (cursor?: string) => {
    setLoading(true);
    setError(null);

    try {
      const point = geoPoint ?? { latitude: 34.26, longitude: 117.18 };
      const res = await feedService.getMoments({
        reference_point: point,
        source,
        radius: 50000,
        pagination: { limit: 20, cursor }
      });

      if (isSuccess(res) && res.data) {
        if (cursor) {
          setMoments(prev => [...prev, ...res.data!.items]);
        } else {
          setMoments(res.data.items);
        }
        setNextCursor(res.data.next_cursor);
      } else {
        setError('加载失败，请重试');
      }
    } catch (e) {
      setError('网络异常，请重试');
    } finally {
      setLoading(false);
    }
  }, [geoPoint, source]);

  useEffect(() => {
    load();
  }, [load]);

  return {
    moments,
    loading,
    error,
    hasMore: !!nextCursor,
    loadMore: () => nextCursor && load(nextCursor),
    refresh: () => load()
  };
}
