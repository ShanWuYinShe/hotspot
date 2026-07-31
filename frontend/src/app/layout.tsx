import type { Metadata } from 'next'
import { Providers } from '@/app/Providers'
import '@/styles/index.css'

export const metadata: Metadata = {
  title: 'HotSpot',
  description: '发现附近，连接彼此',
}

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <head>
        <style>{`html, body { height: 100%; margin: 0; }`}</style>
      </head>
      <body>
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
