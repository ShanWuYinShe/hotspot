import { useState, useCallback, useRef } from 'react';
import { searchService } from '@/lib/api/services/search';
import { useLocationStore } from '@/lib/stores/location';
import { isSuccess } from '@/lib/api/client';
import type { SearchSuggestItem } from '@/lib/types/api';

interface SearchData {
  results: SearchSuggestItem[];
  loading: boolean;
  search: (query: string) => void;
  clear: () => void;
}

export function useSearchData(): SearchData {
  const [results, setResults] = useState<SearchSuggestItem[]>([]);
  const [loading, setLoading] = useState(false);
  const geoPoint = useLocationStore((s) => s.geoPoint);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  const search = useCallback(
    (query: string) => {
      if (timerRef.current) clearTimeout(timerRef.current);

      if (!query.trim()) {
        setResults([]);
        setLoading(false);
        return;
      }

      setLoading(true);
      timerRef.current = setTimeout(async () => {
        try {
          const point = geoPoint ?? { latitude: 34.26, longitude: 117.18 };
          const res = await searchService.suggest({
            prefix: query.trim(),
            reference_point: point
          });

          if (isSuccess(res) && res.data) {
            setResults(res.data.items);
          } else {
            setResults([]);
          }
        } catch {
          setResults([]);
        } finally {
          setLoading(false);
        }
      }, 300);
    },
    [geoPoint]
  );

  const clear = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    setResults([]);
    setLoading(false);
  }, []);

  return { results, loading, search, clear };
}
