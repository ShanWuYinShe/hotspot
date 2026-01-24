/**
 * 主导航组件 (Server Component)
 * 优化的导航栏设计
 */

import { getTranslations } from 'next-intl/server';
import type { Route } from 'next';
import { NavLink } from './nav-link';
import type { LucideIcon } from '@/components/icons';

interface NavItem {
  href: Route;
  iconName: LucideIcon | string;
  labelKey: string;
  exact?: boolean;
  badge?: string;
}

const navItems: NavItem[] = [
  { href: '/' as Route, iconName: 'home', labelKey: 'nav.feed', exact: true },
  { href: '/search' as Route, iconName: 'search', labelKey: 'nav.search', exact: false },
  { href: '/moments' as Route, iconName: 'compass', labelKey: 'nav.moments', exact: false },
  { href: '/chat' as Route, iconName: 'message-circle', labelKey: 'nav.chat', exact: false },
  { href: '/notifications' as Route, iconName: 'bell', labelKey: 'nav.notifications', exact: false },
] as const;

export async function MainNav() {
  const t = await getTranslations();

  return (
    <nav className="hidden md:flex items-center gap-1">
      {navItems.map((item) => (
        <NavLink
          key={item.href}
          href={item.href}
          iconName={item.iconName}
          label={t(item.labelKey)}
          variant="desktop"
          exact={item.exact}
          badge={item.badge}
        />
      ))}
    </nav>
  );
}

export async function MobileNav() {
  const t = await getTranslations();

  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 border-t-2 border-border bg-background/95 backdrop-blur-xl supports-[backdrop-filter]:bg-background/70 md:hidden pb-safe">
      <div className="grid grid-cols-5 h-16">
        {navItems.map((item) => (
          <NavLink
            key={item.href}
            href={item.href}
            iconName={item.iconName}
            label={t(item.labelKey)}
            variant="mobile"
            exact={item.exact}
            badge={item.badge}
          />
        ))}
      </div>
    </nav>
  );
}
