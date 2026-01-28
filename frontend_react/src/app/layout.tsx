import type { Metadata, Viewport } from 'next';
import { Providers } from '@/components/providers';
import './globals.css';

export const metadata: Metadata = {
  title: {
    default: 'HotSpot',
    template: '%s | HotSpot',
  },
  description: '发现身边的精彩，与附近的人分享生活',
  keywords: ['社交', '附近', '位置', '分享', '瞬间'],
  authors: [{ name: 'HotSpot Team' }],
  creator: 'HotSpot',
  openGraph: {
    type: 'website',
    locale: 'zh_CN',
    url: 'https://hotspot.app',
    siteName: 'HotSpot',
    title: 'HotSpot',
    description: '发现身边的精彩，与附近的人分享生活',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'HotSpot',
    description: '发现身边的精彩，与附近的人分享生活',
  },
  robots: {
    index: true,
    follow: true,
  },
};

export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
  maximumScale: 1,
  userScalable: false,
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#ffffff' },
    { media: '(prefers-color-scheme: dark)', color: '#0a0a0a' },
  ],
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body className="min-h-screen bg-background font-sans antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
