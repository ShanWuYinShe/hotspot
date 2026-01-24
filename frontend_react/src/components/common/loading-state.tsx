/**
 * 加载状态组件 (Server Component)
 * 纯展示组件，无客户端逻辑
 */

import { Loader } from '@/components/icons';
import { cn } from '@/lib/utils';

interface LoadingStateProps {
  className?: string;
  text?: string;
  size?: 'sm' | 'md' | 'lg';
}

const sizeMap = {
  sm: 'h-4 w-4',
  md: 'h-6 w-6',
  lg: 'h-8 w-8',
};

export function LoadingState({ className, text, size = 'md' }: LoadingStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center gap-2 py-8', className)}>
      <Loader className={cn('animate-spin text-muted-foreground', sizeMap[size])} />
      {text && <p className="text-sm text-muted-foreground">{text}</p>}
    </div>
  );
}
