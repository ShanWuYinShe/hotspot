import { type ReactNode } from 'react';
import Link from 'next/link';
import { MainNav, UserMenu, MobileNav } from '@/components/layout';

export default function MainLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-b from-background via-background to-muted/30">
      <header className="sticky top-0 z-40 border-b-2 border-border bg-background/95 shadow-lg backdrop-blur-xl supports-[backdrop-filter]:bg-background/80 transition-all duration-300">
        <div className="container flex h-16 items-center justify-between px-4 sm:px-6">
          <Link href="/" className="group flex items-center gap-2 font-bold text-2xl text-primary transition-all duration-200 hover:scale-105">
            <div className="relative flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary-dark text-primary-foreground shadow-lg shadow-primary/30 transition-transform group-hover:rotate-3">
              <span className="text-lg">H</span>
            </div>
            <span className="bg-gradient-to-r from-primary to-primary-dark bg-clip-text text-transparent">
              HotSpot
            </span>
          </Link>

          <div className="flex items-center gap-2">
            <MainNav />
            <div className="h-6 w-px bg-border" />
            <UserMenu />
          </div>
        </div>
      </header>

      <main className="container min-h-[calc(100vh-4rem)] space-y-8 px-4 pb-28 pt-8 sm:px-6 lg:px-8 md:pb-10">
        {children}
      </main>

      <MobileNav />
    </div>
  );
}
