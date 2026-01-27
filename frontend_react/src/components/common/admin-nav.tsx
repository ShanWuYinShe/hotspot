'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { ShieldCheck, Users, AlertTriangle, Bell, FileText, KeyRound, Settings } from 'lucide-react';

import { Button } from '@/components/ui/button';

export function AdminNav() {
  const pathname = usePathname();
  const t = useTranslations('admin');

  const links = [
    { href: '/admin', label: t('title'), icon: ShieldCheck },
    { href: '/admin/users', label: t('users'), icon: Users },
    { href: '/admin/groups', label: t('groups'), icon: Users },
    { href: '/admin/reports', label: t('reports'), icon: AlertTriangle },
    { href: '/admin/announcements', label: t('announcements'), icon: Bell },
    { href: '/admin/audit', label: t('audit'), icon: FileText },
    { href: '/admin/rbac', label: t('rbac'), icon: KeyRound },
    { href: '/admin/settings', label: t('settings'), icon: Settings },
  ];

  return (
    <nav className="flex flex-col space-y-2 p-4 w-64 border-r h-screen sticky top-0 bg-background/95">
      <div className="font-bold text-xl mb-6 px-4">{t('title')}</div>
      {links.map((link) => {
        const Icon = link.icon;
        const isActive = pathname === link.href || (link.href !== '/admin' && pathname.startsWith(link.href));
        return (
          <Link key={link.href} href={link.href}>
            <Button variant={isActive ? 'secondary' : 'ghost'} className="w-full justify-start">
              <Icon className="mr-2 h-4 w-4" />
              {link.label}
            </Button>
          </Link>
        );
      })}
    </nav>
  );
}
