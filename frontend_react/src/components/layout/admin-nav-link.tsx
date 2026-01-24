'use client';

/**
 * Admin 导航链接组件 (Client Component)
 * 需要 usePathname 判断激活状态
 */

import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { Shield, Users, UsersRound, Flag, Megaphone, FileText, Lock } from '@/components/icons';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';

type AdminIconName = 'dashboard' | 'users' | 'groups' | 'reports' | 'announcements' | 'audit' | 'rbac';

const iconMap = {
  dashboard: Shield,
  users: Users,
  groups: UsersRound,
  reports: Flag,
  announcements: Megaphone,
  audit: FileText,
  rbac: Lock,
} as const;

interface AdminNavLinkProps {
  href: Route;
  iconName: AdminIconName;
  label: string;
  exact?: boolean;
}

export function AdminNavLink({ href, iconName, label, exact = false }: AdminNavLinkProps) {
  const pathname = usePathname();
  const isActive = exact ? pathname === href : pathname.startsWith(href);
  const Icon = iconMap[iconName];

  return (
    <Button
      asChild
      variant={isActive ? 'secondary' : 'ghost'}
      className={cn('w-full justify-start gap-2', isActive && 'bg-accent')}
    >
      <Link href={href}>
        <Icon className="h-4 w-4" />
        {label}
      </Link>
    </Button>
  );
}
