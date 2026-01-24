/**
 * 登录页面 (Server Component)
 * 静态结构服务端渲染，交互由客户端组件处理
 */

import { getTranslations } from 'next-intl/server';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { AuthTabs } from '@/components/forms/auth-tabs';

export default async function LoginPage() {
  const t = await getTranslations('auth');

  return (
    <Card className="w-full">
      <CardHeader className="text-center">
        <CardTitle className="text-2xl">HotSpot</CardTitle>
        <CardDescription>连接身边的人</CardDescription>
      </CardHeader>
      <CardContent>
        <AuthTabs
          loginLabel={t('login.submit')}
          registerLabel={t('register.submit')}
          guestLabel={t('guest')}
        />
      </CardContent>
    </Card>
  );
}
