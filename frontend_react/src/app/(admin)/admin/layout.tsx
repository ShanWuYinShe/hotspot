/**
 * Admin 布局 (Server Component)
 * 静态结构在服务端渲染
 */

import { type ReactNode } from 'react';
import Link from 'next/link';
import { getTranslations } from 'next-intl/server';
import { Shield, ArrowLeft } from '@/components/icons';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { AdminNavLink } from '@/components/layout/admin-nav-link';

export default async function AdminLayout({ children }: { children: ReactNode }) {
  const t = await getTranslations();

  const adminNavItems = [
    { href: '/admin', iconName: 'dashboard', labelKey: 'admin.dashboard', exact: true },
    { href: '/admin/users', iconName: 'users', labelKey: 'admin.users', exact: false },
    { href: '/admin/groups', iconName: 'groups', labelKey: 'admin.groups', exact: false },
    { href: '/admin/reports', iconName: 'reports', labelKey: 'admin.reports', exact: false },
    { href: '/admin/announcements', iconName: 'announcements', labelKey: 'admin.announcements', exact: false },
    { href: '/admin/audit', iconName: 'audit', labelKey: 'admin.audit', exact: false },
    { href: '/admin/rbac', iconName: 'rbac', labelKey: 'admin.rbac', exact: false },
  ] as const;

  return (
    <div className="min-h-screen bg-background">
      {/* Sidebar */}
      <aside className="fixed inset-y-0 left-0 z-50 w-64 border-r bg-card">
        <div className="flex h-14 items-center border-b px-4">
          <Link href="/admin" className="flex items-center gap-2 font-bold text-lg">
            <Shield className="h-5 w-5 text-primary" />
            {t('admin.title')}
          </Link>
        </div>
        <ScrollArea className="h-[calc(100vh-3.5rem)]">
          <nav className="p-4 space-y-1">
            {adminNavItems.map((item) => (
              <AdminNavLink
                key={item.href}
                href={item.href}
                iconName={item.iconName}
                label={t(item.labelKey)}
                exact={item.exact}
              />
            ))}
            <div className="pt-4 border-t mt-4">
              <Button asChild variant="ghost" className="w-full justify-start gap-2">
                <Link href="/">
                  <ArrowLeft className="h-4 w-4" />
                  返回主站
                </Link>
              </Button>
            </div>
          </nav>
        </ScrollArea>
      </aside>

      {/* Main Content */}
      <main className="pl-64">
        <div className="container py-6">{children}</div>
      </main>
    </div>
  );
}
