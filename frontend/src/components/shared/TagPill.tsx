import React from 'react'
import { cn } from '@/lib/utils'

type TagPillProps = {
  children: React.ReactNode
  active?: boolean
  onClick?: () => void
  size?: 'sm' | 'md'
  className?: string
}

const sizeClasses = {
  sm: 'text-[10px]',
  md: 'text-xs',
}

export function TagPill({ children, active = false, onClick, size = 'md', className }: TagPillProps) {
  const Component = onClick ? 'button' : 'span'

  return (
    <Component
      type={onClick ? 'button' : undefined}
      onClick={onClick}
      aria-pressed={onClick ? active : undefined}
      className={cn(
        'rounded-full outline-none transition-colors px-3 py-1',
        sizeClasses[size],
        active
          ? 'bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] border border-[var(--neu-accent-soft-border)]'
          : 'bg-transparent text-[var(--neu-text-muted)] border border-[var(--neu-border)]',
        onClick && 'cursor-pointer hover:border-[var(--neu-border-pressed)] hover:text-[var(--neu-text)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]',
        className
      )}
    >
      {children}
    </Component>
  )
}
