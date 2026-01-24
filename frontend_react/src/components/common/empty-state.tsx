import { cn } from '@/lib/utils';
import type { AppIcon } from '@/components/icons';

interface EmptyStateProps {
  icon?: AppIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
  size?: 'sm' | 'md' | 'lg';
}

const sizeMap = {
  sm: 'p-6 py-8',
  md: 'px-6 py-12',
  lg: 'px-8 py-16',
};

const iconSizeMap = {
  sm: 'h-6 w-6',
  md: 'h-9 w-9',
  lg: 'h-12 w-12',
};

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
  size = 'md',
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-4 rounded-2xl border border-dashed bg-muted/40 text-center transition-all duration-300',
        sizeMap[size],
        className
      )}
    >
      {Icon && (
        <div className="rounded-full bg-background p-4 shadow-sm transition-transform duration-300 hover:scale-110">
          <Icon className={cn('text-muted-foreground', iconSizeMap[size])} />
        </div>
      )}
      <div className="space-y-2">
        <h3 className="text-lg font-semibold transition-colors">{title}</h3>
        {description && (
          <p className="text-sm text-muted-foreground max-w-sm leading-relaxed">
            {description}
          </p>
        )}
      </div>
      {action && (
        <div className="mt-2 transition-all duration-300 hover:scale-105">
          {action}
        </div>
      )}
    </div>
  );
}
