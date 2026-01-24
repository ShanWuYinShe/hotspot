'use client';

import type { Route } from 'next';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import type { LucideIcon } from '@/components/icons';
import {
  Home,
  Search,
  Compass,
  MessageCircle,
  Bell,
} from '@/components/icons';

interface NavLinkProps {
  href: Route;
  iconName: LucideIcon | string;
  label: string;
  variant?: 'desktop' | 'mobile';
  exact?: boolean;
  badge?: string;
}

const iconMap: Record<string, LucideIcon> = {
  home: Home,
  search: Search,
  moments: Compass,
  chat: MessageCircle,
  notifications: Bell,
} as const;

export function NavLink({
  href,
  iconName,
  label,
  variant = 'desktop',
  exact = false,
  badge,
}: NavLinkProps) {
  const pathname = usePathname();
  const isActive = exact ? pathname === href : pathname.startsWith(href);
  const Icon = iconMap[iconName as keyof typeof iconMap];

  if (variant === 'mobile') {
    return (
      <Link
        href={href}
        className={cn(
          'group relative flex flex-col items-center justify-center gap-1 p-2 text-muted-foreground transition-all duration-200',
          isActive && 'text-primary bg-primary/10'
        )}
      >
        <Icon className="h-5 w-5 transition-transform duration-200 group-hover:scale-110" />
        <span className="text-xs font-medium transition-colors group-hover:text-foreground">
          {label}
        </span>
        {badge && (
          <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground ring-2 ring-background">
            {badge}
          </span>
        )}
      </Link>
    );
  }

  return (
    <Button
      asChild
      variant={isActive ? 'secondary' : 'ghost'}
      size="sm"
      className={cn(
        'relative gap-2 transition-all duration-200 hover:scale-105',
        isActive && 'bg-accent ring-2 ring-primary/20'
      )}
    >
      <Link href={href}>
        <Icon className="h-4 w-4 transition-transform duration-200 group-hover:scale-110" />
        <span className="hidden lg:inline font-medium">{label}</span>
        {badge && (
          <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground ring-2 ring-background animate-bounce">
            {badge}
          </span>
        )}
      </Link>
    </Button>
  );
}
