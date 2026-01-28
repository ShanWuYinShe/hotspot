'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, Eye, EyeOff, Compass } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { useAuth } from '@/hooks/use-auth';

const loginSchema = z.object({
  loginId: z
    .string()
    .min(1, '请输入用户名/邮箱/手机号'),
  password: z
    .string()
    .min(6, '密码至少 6 位'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = searchParams.get('redirect') || '/';
  const { login, isLoading } = useAuth();
  const [error, setError] = React.useState<string | null>(null);
  const [showPassword, setShowPassword] = React.useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      loginId: '',
      password: '',
    },
  });

  const onSubmit = async (data: LoginFormData) => {
    setError(null);
    const result = await login(data.loginId, data.password);
    if (result.code === 0) {
      router.push(redirect);
    } else {
      setError(result.message || '登录失败');
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-2 text-center">
        <h2 className="text-2xl font-bold">欢迎回来</h2>
        <p className="text-sm text-muted-foreground">
          输入您的账号信息登录
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="loginId">用户名 / 邮箱 / 手机号</Label>
          <Input
            id="loginId"
            type="text"
            placeholder="请输入用户名、邮箱或手机号"
            autoComplete="username"
            disabled={isLoading}
            className="h-11 rounded-lg"
            {...register('loginId')}
          />
          {errors.loginId && (
            <p className="text-sm text-destructive">{errors.loginId.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="password">密码</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              placeholder="请输入密码"
              autoComplete="current-password"
              disabled={isLoading}
              className="h-11 rounded-lg pr-10"
              {...register('password')}
            />
            <button
              type="button"
              onClick={() => setShowPassword(!showPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              {showPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
          {errors.password && (
            <p className="text-sm text-destructive">{errors.password.message}</p>
          )}
        </div>

        {error && (
          <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20">
            <p className="text-sm text-destructive text-center">{error}</p>
          </div>
        )}

        <Button 
          type="submit" 
          className="w-full h-11 rounded-lg font-medium" 
          disabled={isLoading}
        >
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              登录中...
            </>
          ) : (
            '登录'
          )}
        </Button>
      </form>

      <div className="relative">
        <div className="absolute inset-0 flex items-center">
          <Separator />
        </div>
        <div className="relative flex justify-center text-xs uppercase">
          <span className="bg-card px-2 text-muted-foreground">或</span>
        </div>
      </div>

      <div className="space-y-3">
        <Button asChild variant="outline" className="w-full h-11 rounded-lg">
          <Link href="/register">创建新账号</Link>
        </Button>
        <Button asChild variant="ghost" className="w-full">
          <Link href="/explore" className="flex items-center gap-2">
            <Compass className="h-4 w-4" />
            以访客身份浏览
          </Link>
        </Button>
      </div>
    </div>
  );
}
