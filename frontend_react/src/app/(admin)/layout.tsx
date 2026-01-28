'use client';

import * as React from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  LayoutDashboard,
  Users,
  FileText,
  Flag,
  Settings,
  LogOut,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { useAuthStore } from '@/stores/auth';
import { useAuth } from '@/hooks/use-auth';

const navItems = [
  { href: '/admin', icon: LayoutDashboard, label: '仪表盘' },
  { href: '/admin/users', icon: Users, label: '用户管理' },
  { href: '/admin/moments', icon: FileText, label: '内容管理' },
  { href: '/admin/reports', icon: Flag, label: '举报处理' },
  { href: '/admin/settings', icon: Settings, label: '系统设置' },
];

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, isAuthenticated } = useAuthStore();
  const { logout } = useAuth();

  // 检查管理员权限
  React.useEffect(() => {
    if (!isAuthenticated || user?.user_type !== 'admin') {
      router.replace('/');
    }
  }, [isAuthenticated, user, router]);

  if (!isAuthenticated || user?.user_type !== 'admin') {
    return null;
  }

  return (
    <div className="flex min-h-screen">
      {/* 侧边栏 */}
      <aside className="w-64 border-r bg-card">
        <div className="flex h-14 items-center border-b px-4">
          <Link href="/admin" className="font-bold">
            HotSpot Admin
          </Link>
        </div>
        <nav className="flex flex-col gap-1 p-2">
          {navItems.map((item) => {
            const isActive = pathname === item.href ||
              (item.href !== '/admin' && pathname.startsWith(item.href));
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                )}
              >
                <item.icon className="h-4 w-4" />
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="absolute bottom-0 left-0 w-64 border-t p-2">
          <Button
            variant="ghost"
            className="w-full justify-start"
            onClick={async () => {
              await logout();
              router.push('/');
            }}
          >
            <LogOut className="mr-2 h-4 w-4" />
            退出登录
          </Button>
        </div>
      </aside>

      {/* 主内容区 */}
      <main className="flex-1">
        <div className="h-14 border-b px-6 flex items-center justify-between">
          <h1 className="font-semibold">管理后台</h1>
          <span className="text-sm text-muted-foreground">
            {user?.nickname} (管理员)
          </span>
        </div>
        <div className="p-6">{children}</div>
      </main>
    </div>
  );
}
