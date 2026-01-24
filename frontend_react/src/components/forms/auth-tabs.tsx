'use client';

/**
 * 认证标签页组件 (Client Component)
 * 处理登录/注册标签切换和访客登录
 */

import { useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import { LoginForm } from './login-form';
import { RegisterForm } from './register-form';
import { useAuth } from '@/hooks/use-auth';

interface AuthTabsProps {
  loginLabel: string;
  registerLabel: string;
  guestLabel: string;
}

export function AuthTabs({ loginLabel, registerLabel, guestLabel }: AuthTabsProps) {
  const { guestLogin } = useAuth();
  const searchParams = useSearchParams();
  const redirectTo = searchParams.get('redirect') ?? undefined;
  const [guestNickname, setGuestNickname] = useState('');
  const [isGuestLoading, setIsGuestLoading] = useState(false);

  const handleGuestLogin = async () => {
    if (!guestNickname.trim()) return;
    setIsGuestLoading(true);
    try {
      await guestLogin(guestNickname, redirectTo);
    } finally {
      setIsGuestLoading(false);
    }
  };

  return (
    <>
      <Tabs defaultValue="login" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="login">{loginLabel}</TabsTrigger>
          <TabsTrigger value="register">{registerLabel}</TabsTrigger>
        </TabsList>
        <TabsContent value="login" className="mt-4">
          <LoginForm redirectTo={redirectTo} />
        </TabsContent>
        <TabsContent value="register" className="mt-4">
          <RegisterForm redirectTo={redirectTo} />
        </TabsContent>
      </Tabs>

      <div className="relative my-6">
        <Separator />
        <span className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-card px-2 text-xs text-muted-foreground">
          或
        </span>
      </div>

      <div className="space-y-3">
        <div className="flex gap-2">
          <Input
            placeholder="输入昵称"
            value={guestNickname}
            onChange={(e) => setGuestNickname(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleGuestLogin()}
          />
          <Button
            variant="outline"
            onClick={handleGuestLogin}
            disabled={!guestNickname.trim() || isGuestLoading}
          >
            {guestLabel}
          </Button>
        </div>
      </div>
    </>
  );
}
