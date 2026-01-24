import { cn } from '@/lib/utils';

interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: 'default' | 'text' | 'circular' | 'rectangular';
  size?: 'sm' | 'md' | 'lg';
}

const sizeMap = {
  sm: 'h-4 w-4',
  md: 'h-6 w-6',
  lg: 'h-8 w-8',
};

const variantMap = {
  default: 'rounded-md',
  text: 'h-4 w-full rounded-sm',
  circular: 'rounded-full',
  rectangular: 'rounded-md',
};

export function Skeleton({ className, variant = 'default', size, ...props }: SkeletonProps) {
  return (
    <div
      className={cn(
        'animate-pulse bg-muted',
        variantMap[variant],
        size && sizeMap[size],
        className
      )}
      {...props}
    />
  );
}
