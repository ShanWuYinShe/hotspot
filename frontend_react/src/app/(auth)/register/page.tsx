/**
 * 注册页面 (Server Component)
 */

import type { Metadata } from 'next';
import Link from 'next/link';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { RegisterForm } from '@/components/forms/register-form';

export const metadata: Metadata = {
  title: '注册 - HotSpot',
  description: '创建您的 HotSpot 账户',
};

interface RegisterPageProps {
  searchParams: Promise<{ redirect?: string }>;
}

export default async function RegisterPage({ searchParams }: RegisterPageProps) {
  const params = await searchParams;
  const redirectTo = params.redirect;

  return (
    <Card className="w-full">
      <CardHeader className="text-center">
        <CardTitle className="text-2xl">HotSpot</CardTitle>
        <CardDescription>创建您的账户</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <RegisterForm redirectTo={redirectTo} />

        <p className="text-center text-sm text-muted-foreground">
          已有账户？{' '}
          <Link href="/login" className="text-primary hover:underline">
            立即登录
          </Link>
        </p>
      </CardContent>
    </Card>
  );
}
