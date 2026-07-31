type OnlineDotSize = 'sm' | 'md' | 'lg'

const sizeMap: Record<OnlineDotSize, number> = { sm: 8, md: 10, lg: 12 }

type OnlineDotProps = {
  size?: OnlineDotSize
  className?: string
}

export function OnlineDot({ size = 'md', className }: OnlineDotProps) {
  const px = sizeMap[size]
  return (
    <span
      className={className}
      style={{
        width: px,
        height: px,
        borderRadius: '9999px',
        background: 'var(--semantic-online)',
        border: '2px solid var(--neu-bg)',
        position: 'absolute',
        bottom: 0,
        right: 0,
      }}
    />
  )
}
