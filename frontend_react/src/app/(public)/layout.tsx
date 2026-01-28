'use client';

import * as React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { useAuthStore } from '@/stores/auth';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  Flame,
  Compass,
  Home,
  MessageCircle,
  Bell,
  User,
  Settings,
  LogOut,
  LogIn,
  UserPlus,
} from 'lucide-react';

const publicNavItems = [
  { href: '/', label: '首页', icon: Home },
  { href: '/explore', label: '探索', icon: Compass },
];

const authNavItems = [
  { href: '/feed', label: '动态', icon: Home },
  { href: '/chat', label: '消息', icon: MessageCircle },
  { href: '/notifications', label: '通知', icon: Bell },
];

export default function PublicLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const { isAuthenticated, user, logout } = useAuthStore();

  const navItems = isAuthenticated
    ? [...publicNavItems, ...authNavItems]
    : publicNavItems;

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="container flex h-16 items-center justify-between px-4 mx-auto max-w-6xl">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-2.5 group">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/80 shadow-md group-hover:shadow-glow transition-all duration-300">
              <Flame className="h-5 w-5 text-primary-foreground" />
            </div>
            <span className="font-bold text-xl bg-gradient-to-r from-primary to-primary/70 bg-clip-text text-transparent">
              HotSpot
            </span>
          </Link>

          {/* Desktop Navigation */}
          <nav className="hidden md:flex items-center gap-1">
            {navItems.map((item) => {
              const isActive = pathname === item.href || 
                (item.href !== '/' && pathname.startsWith(item.href));
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    'flex items-center gap-2 rounded-full px-4 py-2 text-sm font-medium transition-all duration-200',
                    isActive
                      ? 'bg-primary text-primary-foreground shadow-md'
                      : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
                  )}
                >
                  <item.icon className="h-4 w-4" />
                  <span>{item.label}</span>
                </Link>
              );
            })}
          </nav>

          {/* User Actions */}
          <div className="flex items-center gap-2">
            {isAuthenticated ? (
              <>
                <Link href="/profile">
                  <Avatar className="h-9 w-9 ring-2 ring-background shadow-md cursor-pointer hover:ring-primary/50 transition-all">
                    <AvatarImage src={user?.avatar_url} alt={user?.nickname} />
                    <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-sm font-medium">
                      {user?.nickname?.slice(0, 2).toUpperCase() || 'U'}
                    </AvatarFallback>
                  </Avatar>
                </Link>
                <Link href="/settings" className="hidden sm:block">
                  <Button variant="ghost" size="icon" className="rounded-full">
                    <Settings className="h-4 w-4" />
                  </Button>
                </Link>
                <Button
                  variant="ghost"
                  size="icon"
                  className="rounded-full text-muted-foreground hover:text-destructive"
                  onClick={logout}
                >
                  <LogOut className="h-4 w-4" />
                </Button>
              </>
            ) : (
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" className="rounded-full" asChild>
                  <Link href="/login">
                    <LogIn className="h-4 w-4 mr-1.5" />
                    登录
                  </Link>
                </Button>
                <Button size="sm" className="rounded-full shadow-md" asChild>
                  <Link href="/register">
                    <UserPlus className="h-4 w-4 mr-1.5" />
                    注册
                  </Link>
                </Button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto max-w-6xl px-4 py-6">
        {children}
      </main>

      {/* Mobile Bottom Navigation */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 z-50 border-t bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 safe-area-inset-bottom">
        <div className="flex items-center justify-around h-16 px-2">
          {navItems.slice(0, 5).map((item) => {
            const isActive = pathname === item.href || 
              (item.href !== '/' && pathname.startsWith(item.href));
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'flex flex-col items-center justify-center gap-0.5 w-16 py-1 rounded-lg transition-colors',
                  isActive
                    ? 'text-primary'
                    : 'text-muted-foreground hover:text-foreground'
                )}
              >
                <item.icon className={cn('h-5 w-5', isActive && 'fill-primary/20')} />
                <span className="text-xs font-medium">{item.label}</span>
              </Link>
            );
          })}
          <Link
            href={isAuthenticated ? '/profile' : '/login'}
            className={cn(
              'flex flex-col items-center justify-center gap-0.5 w-16 py-1 rounded-lg transition-colors',
              pathname.startsWith('/profile')
                ? 'text-primary'
                : 'text-muted-foreground hover:text-foreground'
            )}
          >
            <User className={cn('h-5 w-5', pathname.startsWith('/profile') && 'fill-primary/20')} />
            <span className="text-xs font-medium">{isAuthenticated ? '我的' : '登录'}</span>
          </Link>
        </div>
      </nav>

      {/* Bottom padding for mobile nav */}
      <div className="md:hidden h-16" />
    </div>
  );
}
