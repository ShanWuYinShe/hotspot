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
  compass: Compass,
  'message-circle': MessageCircle,
  bell: Bell,
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
          'group relative flex flex-col items-center justify-center gap-1 p-2 text-muted-foreground transition-all duration-200 rounded-xl',
          isActive && 'text-primary bg-primary/10',
          !isActive && 'hover:bg-accent/50 hover:text-foreground'
        )}
      >
        <div className="relative">
          <Icon className="h-5 w-5 transition-transform duration-200 group-hover:scale-110" />
          {isActive && (
            <span className="absolute -bottom-1 left-1/2 -translate-x-1/2 h-1 w-1 rounded-full bg-primary" />
          )}
        </div>
        <span className="text-[10px] font-medium transition-colors">
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
        'relative gap-2 transition-all duration-200 rounded-xl',
        isActive && 'bg-primary text-primary-foreground hover:bg-primary/90',
        !isActive && 'hover:bg-accent hover:scale-105'
      )}
    >
      <Link href={href}>
        <Icon className="h-4 w-4 transition-transform duration-200 group-hover:scale-110" />
        <span className="hidden sm:inline font-medium">{label}</span>
        {badge && (
          <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground ring-2 ring-background animate-pulse">
            {badge}
          </span>
        )}
      </Link>
    </Button>
  );
}
