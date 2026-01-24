/**
 * 登录页面 (Server Component)
 * 静态结构服务端渲染，交互由客户端组件处理
 */

import { getTranslations } from 'next-intl/server';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { AuthTabs } from '@/components/forms/auth-tabs';
import { Users, MapPin, Compass } from '@/components/icons';

export default async function LoginPage() {
  const t = await getTranslations('auth');

  return (
    <div className="flex flex-col items-center justify-center min-h-screen px-4 py-8 gap-8">
      <div className="text-center space-y-4 max-w-md">
        <div className="mx-auto mb-4 relative">
          <div className="absolute inset-0 bg-gradient-to-r from-primary/20 to-primary/40 rounded-full blur-2xl" />
          <div className="relative flex h-20 w-20 mx-auto items-center justify-center rounded-2xl bg-gradient-to-br from-primary to-primary-dark shadow-2xl shadow-primary/30">
            <span className="text-4xl font-bold text-primary-foreground">H</span>
          </div>
        </div>
        <CardTitle className="text-4xl font-bold bg-gradient-to-r from-primary to-primary-dark bg-clip-text text-transparent">
          HotSpot
        </CardTitle>
        <CardDescription className="text-base">
          连接身边的人，发现身边的精彩
        </CardDescription>
      </div>

      <Card className="w-full max-w-md border-2 shadow-xl shadow-primary/10 backdrop-blur-sm bg-card/95">
        <CardContent className="p-6">
          <AuthTabs
            loginLabel={t('login.submit')}
            registerLabel={t('register.submit')}
            guestLabel={t('guest')}
          />
        </CardContent>
      </Card>

      <div className="grid grid-cols-3 gap-8 max-w-md">
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Users className="h-6 w-6" />
          </div>
          <span className="text-xs font-medium text-muted-foreground">社交互动</span>
        </div>
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <MapPin className="h-6 w-6" />
          </div>
          <span className="text-xs font-medium text-muted-foreground">附近动态</span>
        </div>
        <div className="flex flex-col items-center gap-2 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Compass className="h-6 w-6" />
          </div>
          <span className="text-xs font-medium text-muted-foreground">精彩瞬间</span>
        </div>
      </div>
    </div>
  );
}
