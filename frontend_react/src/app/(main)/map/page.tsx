'use client';

import { useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { MapService } from '@/lib/api/services/map';
import { useLocation } from '@/hooks/use-location';

export default function MapPage() {
  const t = useTranslations('map');
  const tCommon = useTranslations('common');
  const { referencePoint, requestLocation } = useLocation();

  const viewport = useMemo(() => {
    if (!referencePoint) {
      return null;
    }
    const delta = 0.02;
    return {
      northeast: {
        latitude: referencePoint.latitude + delta,
        longitude: referencePoint.longitude + delta,
      },
      southwest: {
        latitude: referencePoint.latitude - delta,
        longitude: referencePoint.longitude - delta,
      },
      zoom: 14,
    };
  }, [referencePoint]);

  const { data } = useQuery({
    queryKey: ['map-render', viewport],
    queryFn: async () => {
      if (!viewport) {
        return null;
      }
      const res = await MapService.render({ viewport });
      if (res.code !== 0) throw new Error(res.message);
      return res.data!;
    },
    enabled: !!viewport,
  });

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      {!referencePoint ? (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground space-y-4">
            <p>{t('loading')}</p>
            <Button onClick={requestLocation}>{tCommon('confirm')}</Button>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="space-y-4 py-6">
            {data?.items?.length ? (
              <div className="space-y-3">
                {data.items.map((item) => (
                  <div key={item.id} className="rounded-lg border p-3 text-sm">
                    <div className="font-semibold">{item.entity?.title ?? item.cluster?.label ?? item.id}</div>
                    {item.entity?.subtitle ? (
                      <div className="text-muted-foreground">{item.entity.subtitle}</div>
                    ) : null}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-center text-muted-foreground">{tCommon('empty')}</p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
