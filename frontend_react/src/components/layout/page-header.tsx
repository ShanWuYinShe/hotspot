import type { ReactNode } from 'react';
import { cn } from '@/lib/utils';

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  align?: 'start' | 'center';
  className?: string;
  children?: ReactNode;
}

export function PageHeader({
  title,
  description,
  actions,
  align = 'start',
  className,
  children,
}: PageHeaderProps) {
  return (
    <div
      className={cn(
        'flex flex-col gap-4',
        align === 'center' ? 'items-center text-center' : 'items-start',
        className
      )}
    >
      <div
        className={cn(
          'flex w-full flex-col gap-3 sm:flex-row sm:items-end sm:justify-between',
          align === 'center' && 'sm:items-center'
        )}
      >
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">{title}</h1>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
      {children && <div className="w-full">{children}</div>}
    </div>
  );
}
