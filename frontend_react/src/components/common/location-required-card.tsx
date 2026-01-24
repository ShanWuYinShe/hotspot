'use client';

/**
 * 需要位置信息的提示卡片
 */

import { MapPin } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/common/empty-state';
import { useLocation } from '@/hooks/use-location';

interface LocationRequiredCardProps {
  title?: string;
  description?: string;
  actionLabel?: string;
}

export function LocationRequiredCard({
  title = '需要位置信息',
  description = '请允许定位以获得更准确的内容',
  actionLabel = '开启定位',
}: LocationRequiredCardProps) {
  const { refreshLocation, isLoading, error } = useLocation();
  const message = error ? `${description}（${error}）` : description;

  return (
    <EmptyState
      icon={MapPin}
      title={title}
      description={message}
      action={
        <Button
          variant="outline"
          size="sm"
          onClick={() => refreshLocation(false)}
          disabled={isLoading}
        >
          {actionLabel}
        </Button>
      }
    />
  );
}
