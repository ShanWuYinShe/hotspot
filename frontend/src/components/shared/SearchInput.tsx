import React from 'react'
import { Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'

type SearchInputProps = {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  className?: string
  compact?: boolean
}

export function SearchInput({
  value,
  onChange,
  placeholder = '搜索...',
  className,
  compact = false,
}: SearchInputProps) {
  return (
    <div
      className={cn(
        'flex items-center gap-2 neu-input-surface',
        compact ? 'px-3 py-2 rounded-[14px]' : 'px-3.5 py-2.5 rounded-[18px]',
        className
      )}
    >
      <Search size={14} color="var(--neu-text-muted)" />
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="min-w-0 flex-1 bg-transparent outline-none text-[var(--neu-text)] placeholder:text-[var(--neu-text-muted)] text-[13px]"
      />
      {value && (
        <button
          type="button"
          onClick={() => onChange('')}
          className="flex items-center justify-center rounded-full hover:bg-[var(--neu-accent-soft)] w-5 h-5"
          aria-label="清除"
        >
          <X size={12} color="var(--neu-text-muted)" strokeWidth={2} />
        </button>
      )}
    </div>
  )
}
