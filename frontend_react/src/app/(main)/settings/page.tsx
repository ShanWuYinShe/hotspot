'use client';

import { User, Shield, Bell, Map, LogOut, Lock, Laptop } from 'lucide-react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { useAuth } from '@/hooks/use-auth';

import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

export default function SettingsPage() {
  const { user, logout } = useAuth();
  const t = useTranslations('settings');
  const tNav = useTranslations('nav');
  const tCommon = useTranslations('common');

  const settingsSections = [
    { title: t('sections.account'), icon: User, href: '/settings/account', description: t('sections.account') },
    { title: t('sections.privacy'), icon: Shield, href: '/settings/privacy', description: t('sections.privacy') },
    { title: t('sections.notifications'), icon: Bell, href: '/settings/notifications', description: t('sections.notifications') },
    { title: t('sections.location'), icon: Map, href: '/settings/location', description: t('sections.location') },
    { title: t('sections.password'), icon: Lock, href: '/settings/password', description: t('sections.password') },
    { title: t('sections.sessions'), icon: Laptop, href: '/settings/sessions', description: t('sections.sessions') },
  ];

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>

      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <Avatar className="h-16 w-16">
              <AvatarImage src={user?.avatar_url} alt={user?.nickname} />
              <AvatarFallback className="text-xl">
                {user?.nickname?.slice(0, 2).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <div>
              <h2 className="text-lg font-semibold">{user?.nickname}</h2>
              <p className="text-sm text-muted-foreground">
                {user?.user_id ? `ID: ${user.user_id}` : tCommon('loading')}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="space-y-2">
        {settingsSections.map((section) => {
          const Icon = section.icon;
          return (
            <Link key={section.href} href={section.href}>
              <Card className="hover:bg-accent/50 transition-colors cursor-pointer">
                <CardContent className="flex items-center gap-4 p-4">
                  <div className="p-2 bg-primary/10 rounded-md">
                    <Icon className="h-5 w-5 text-primary" />
                  </div>
                  <div className="flex-1">
                    <h3 className="font-medium">{section.title}</h3>
                    <p className="text-sm text-muted-foreground">{section.description}</p>
                  </div>
                </CardContent>
              </Card>
            </Link>
          );
        })}
      </div>

      <Card className="border-red-200">
        <CardContent className="pt-6">
          <Button
            variant="ghost"
            className="w-full justify-start text-red-600 hover:text-red-700 hover:bg-red-50"
            onClick={logout}
          >
            <LogOut className="mr-2 h-4 w-4" />
            {tNav('logout')}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
