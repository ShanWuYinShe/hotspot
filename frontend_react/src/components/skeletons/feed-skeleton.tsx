import { Skeleton } from '@/components/ui/skeleton';

interface FeedSkeletonProps {
  count?: number;
}

export function FeedSkeleton({ count = 3 }: FeedSkeletonProps) {
  return (
    <div className="space-y-6">
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} className="space-y-4 rounded-lg border bg-card p-4">
          <div className="flex items-center justify-between gap-3 mb-3">
            <Skeleton variant="circular" size="md" />
            <Skeleton className="h-8 w-16" />
          </div>

          <div className="space-y-3">
            <Skeleton className="h-20 w-full" />
          </div>

          <div className="grid grid-cols-3 gap-2">
            <Skeleton className="aspect-square" />
            <Skeleton className="aspect-square" />
            <Skeleton className="aspect-square" />
          </div>

          <div className="mt-4 flex gap-2">
            <Skeleton className="h-6 w-16" />
            <Skeleton className="h-6 w-16" />
          </div>
        </div>
      ))}
    </div>
  );
}
