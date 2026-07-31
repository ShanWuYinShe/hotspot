import React from 'react'
import { cn } from '@/lib/utils'

type EmptyStateProps = {
  icon?: React.ReactNode
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}

export function EmptyState({
  icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-20 gap-3', className)}>
      {icon && (
        <div
          className="flex items-center justify-center rounded-full w-16 h-16 border border-[var(--neu-border)]"
          style={{ boxShadow: '6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)' }}
        >
          {icon}
        </div>
      )}
      <p className="text-[var(--neu-text)] text-[15px] font-medium">{title}</p>
      {description && (
        <p className="text-[var(--neu-text-muted)] text-center max-w-xs text-[13px] leading-relaxed">
          {description}
        </p>
      )}
      {action}
    </div>
  )
}
