/**
 * 主导航组件 (Server Component)
 * 优化的导航栏设计
 */

import { getTranslations } from 'next-intl/server';
import { NavLink } from './nav-link';
import type { LucideIcon } from '@/components/icons';

interface NavItem {
  href: string;
  iconName: LucideIcon | string;
  labelKey: string;
  exact?: boolean;
  badge?: string;
}

const navItems: NavItem[] = [
  { href: '/', iconName: 'home', labelKey: 'nav.feed', exact: true },
  { href: '/search', iconName: 'search', labelKey: 'nav.search', exact: false },
  { href: '/moments', iconName: 'compass', labelKey: 'nav.moments', exact: false },
  { href: '/chat', iconName: 'message-circle', labelKey: 'nav.chat', exact: false },
  { href: '/notifications', iconName: 'bell', labelKey: 'nav.notifications', exact: false },
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
    <nav className="fixed bottom-0 left-0 right-0 z-40 border-t bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 md:hidden">
      <div className="grid grid-cols-5 h-16 safe-area-inset-bottom">
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
