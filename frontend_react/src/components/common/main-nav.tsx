'use client';

import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { Home, MessageSquare, Bell, Search, Users, Map, Settings, LogOut, LogIn, Shield, MoreHorizontal, PlusCircle } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useAuth } from '@/hooks/use-auth';
import { useAdminStore } from '@/stores/admin';
import { useNotificationsStore } from '@/stores/notifications';
import { useLoginModal } from '@/providers/login-modal-provider';

export function MainNav() {
  const pathname = usePathname();
  const router = useRouter();
  const { logout, user, isAuthenticated } = useAuth();
  const { permissions } = useAdminStore();
  const { unreadCount } = useNotificationsStore();
  const { setOpen: openLoginModal } = useLoginModal();
  const t = useTranslations('nav');
  const tCommon = useTranslations('common');

  const primaryLinks = [
    { href: '/feed', label: t('feed'), icon: Home },
    { href: '/search', label: t('search'), icon: Search },
    { href: '/chat', label: t('chat'), icon: MessageSquare },
    { href: '/groups', label: t('groups'), icon: Users },
  ];

  const secondaryLinks = [
    { href: '/map', label: t('map'), icon: Map },
    { href: '/notifications', label: t('notifications'), icon: Bell, badge: unreadCount },
  ];

  const isAdmin = permissions.includes('*') || permissions.length > 0;

  const handleLogout = async () => {
    await logout();
    router.push('/feed');
  };

  const handleLogin = () => {
    openLoginModal(true);
  };

  const isActive = (href: string) => {
    if (href === '/feed') {
      return pathname === '/feed' || pathname === '/';
    }
    return pathname.startsWith(href);
  };

  return (
    <nav className="flex flex-col p-4 w-64 border-r h-screen sticky top-0 bg-background">
      <Link href="/feed" className="font-bold text-xl mb-6 px-4 hover:opacity-80 transition-opacity">
        {tCommon('appName')}
      </Link>

      <div className="flex-1 space-y-1">
        {primaryLinks.map((link) => {
          const Icon = link.icon;
          return (
            <Link key={link.href} href={link.href}>
              <Button
                variant={isActive(link.href) ? 'secondary' : 'ghost'}
                className="w-full justify-start"
              >
                <Icon className="mr-3 h-5 w-5" />
                {link.label}
              </Button>
            </Link>
          );
        })}

        <div className="py-2" />

        {secondaryLinks.map((link) => {
          const Icon = link.icon;
          return (
            <Link key={link.href} href={link.href}>
              <Button
                variant={isActive(link.href) ? 'secondary' : 'ghost'}
                className="w-full justify-start"
              >
                <Icon className="mr-3 h-5 w-5" />
                {link.label}
                {link.badge && link.badge > 0 ? (
                  <span className="ml-auto rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">
                    {link.badge > 99 ? '99+' : link.badge}
                  </span>
                ) : null}
              </Button>
            </Link>
          );
        })}

        {isAdmin && (
          <>
            <div className="py-2" />
            <Link href="/admin">
              <Button
                variant={pathname.startsWith('/admin') ? 'secondary' : 'ghost'}
                className="w-full justify-start"
              >
                <Shield className="mr-3 h-5 w-5" />
                {t('admin')}
              </Button>
            </Link>
          </>
        )}

        {isAuthenticated && (
          <>
            <div className="py-3" />
            <Button
              className="w-full"
              onClick={() => router.push('/moments/create')}
            >
              <PlusCircle className="mr-2 h-5 w-5" />
              {t('post')}
            </Button>
          </>
        )}
      </div>

      <div className="mt-auto pt-4 border-t">
        {isAuthenticated ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="w-full justify-start h-auto py-3">
                <Avatar className="h-10 w-10 mr-3">
                  <AvatarImage src={user?.avatar_url} alt={user?.nickname} />
                  <AvatarFallback>
                    {user?.nickname?.slice(0, 2).toUpperCase() || '??'}
                  </AvatarFallback>
                </Avatar>
                <div className="flex-1 text-left">
                  <div className="font-medium truncate">{user?.nickname || t('guest')}</div>
                  <div className="text-xs text-muted-foreground">@{user?.user_id}</div>
                </div>
                <MoreHorizontal className="h-5 w-5 text-muted-foreground" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              <DropdownMenuItem onClick={() => router.push(`/users/${user?.user_id}`)}>
                {t('profile')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => router.push('/settings')}>
                <Settings className="mr-2 h-4 w-4" />
                {t('settings')}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={handleLogout}
                className="text-red-500 focus:text-red-500"
              >
                <LogOut className="mr-2 h-4 w-4" />
                {t('logout')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : (
          <Button
            variant="default"
            className="w-full"
            onClick={handleLogin}
          >
            <LogIn className="mr-2 h-5 w-5" />
            {t('login')}
          </Button>
        )}
      </div>
    </nav>
  );
}
