import { type ReactNode } from 'react';
import Link from 'next/link';
import { MainNav, MobileNav, UserMenu } from '@/components/layout';

export default function MainLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 transition-all duration-300">
        <div className="container flex h-16 items-center justify-between px-4 sm:px-6">
          <Link href="/" className="group flex items-center gap-2 font-semibold text-2xl text-primary transition-transform duration-200 hover:scale-105">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <span className="text-lg font-bold">H</span>
            </div>
            <span>HotSpot</span>
          </Link>

          <div className="flex items-center gap-2">
            <MainNav />
            <div className="h-6 w-px bg-border" />
            <UserMenu />
          </div>
        </div>
      </header>

      <main className="container min-h-[calc(100vh-4rem)] px-4 py-8 sm:px-6 lg:px-8 pb-28 md:pb-8">
        {children}
      </main>

      <MobileNav />
    </div>
  );
}
