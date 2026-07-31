import React from 'react'
import { ArrowLeft } from 'lucide-react'
import { ActionIconButton } from '@/components/shared/ActionIconButton'
import { cn } from '@/lib/utils'

type PageHeaderProps = {
  title: string
  subtitle?: string
  onBack?: () => void
  actions?: React.ReactNode
  children?: React.ReactNode
  sticky?: boolean
  className?: string
}

export function PageHeader({
  title,
  subtitle,
  onBack,
  actions,
  children,
  sticky = false,
  className,
}: PageHeaderProps) {
  return (
    <header
      className={cn(
        'flex items-center gap-3 px-5 py-3',
        sticky && 'sticky top-0 z-10 bg-[var(--neu-bg)]',
        className
      )}
      style={sticky ? { backdropFilter: 'blur(16px)', borderBottom: '1px solid var(--neu-border)' } : undefined}
    >
      <ActionIconButton
        icon={ArrowLeft}
        label="返回"
        onClick={onBack}
        size={34}
      />
      <div className={cn('min-w-0', children ? '' : 'flex-1')}>
        <h1 className="text-[var(--neu-text)] text-lg">{title}</h1>
        {subtitle && (
          <p className="text-[var(--neu-text-muted)] text-[11px] mt-0.5">{subtitle}</p>
        )}
      </div>
      {children}
      {actions}
    </header>
  )
}
