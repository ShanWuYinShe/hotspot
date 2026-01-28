'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Card } from '@/components/ui/card';
import { useAuthStore } from '@/stores/auth';
import { Flame } from 'lucide-react';

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const { isAuthenticated, isHydrated } = useAuthStore();

  // 已登录用户重定向到首页
  React.useEffect(() => {
    if (isHydrated && isAuthenticated) {
      router.replace('/');
    }
  }, [isHydrated, isAuthenticated, router]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background via-background to-primary/5 p-4">
      <Card className="w-full max-w-md p-8 shadow-soft">
        {/* Logo */}
        <Link href="/" className="flex items-center justify-center gap-2 mb-8 group">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/80 shadow-md group-hover:shadow-glow transition-all duration-300">
            <Flame className="h-6 w-6 text-primary-foreground" />
          </div>
          <span className="font-bold text-2xl bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent">
            HotSpot
          </span>
        </Link>
        
        {children}
      </Card>
    </div>
  );
}
