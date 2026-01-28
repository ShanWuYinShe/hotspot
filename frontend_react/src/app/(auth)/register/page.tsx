'use client';

import * as React from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Loader2, Eye, EyeOff, MapPin, Compass } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { useAuth } from '@/hooks/use-auth';

const registerSchema = z.object({
  loginId: z
    .string()
    .min(4, '用户名至少 4 个字符')
    .max(32, '用户名最多 32 个字符')
    .regex(/^[a-zA-Z0-9_@.]+$/, '用户名只能包含字母、数字、下划线、@和.'),
  nickname: z
    .string()
    .min(1, '请输入昵称')
    .max(32, '昵称最多 32 个字符'),
  password: z
    .string()
    .min(6, '密码至少 6 位')
    .max(128, '密码最多 128 位'),
  confirmPassword: z
    .string()
    .min(1, '请确认密码'),
}).refine((data) => data.password === data.confirmPassword, {
  message: '两次输入的密码不一致',
  path: ['confirmPassword'],
});

type RegisterFormData = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const router = useRouter();
  const { register: registerUser, isLoading } = useAuth();
  const [error, setError] = React.useState<string | null>(null);
  const [locationError, setLocationError] = React.useState<string | null>(null);
  const [showPassword, setShowPassword] = React.useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = React.useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      loginId: '',
      nickname: '',
      password: '',
      confirmPassword: '',
    },
  });

  const onSubmit = async (data: RegisterFormData) => {
    setError(null);
    setLocationError(null);

    const result = await registerUser(
      data.loginId,
      data.password,
      data.nickname
    );

    if (result.code === 0) {
      router.push('/');
    } else {
      setError(result.message || '注册失败');
    }
  };

  return (
    <div className="space-y-6">
      <div className="space-y-2 text-center">
        <h2 className="text-2xl font-bold">创建账号</h2>
        <p className="text-sm text-muted-foreground">
          加入 HotSpot，发现身边的精彩
        </p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="loginId">用户名 / 邮箱 / 手机号</Label>
          <Input
            id="loginId"
            type="text"
            placeholder="用于登录的账号"
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
          <Label htmlFor="nickname">昵称</Label>
          <Input
            id="nickname"
            type="text"
            placeholder="显示给其他用户的名称"
            autoComplete="nickname"
            disabled={isLoading}
            className="h-11 rounded-lg"
            {...register('nickname')}
          />
          {errors.nickname && (
            <p className="text-sm text-destructive">{errors.nickname.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="password">密码</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              placeholder="至少 6 位"
              autoComplete="new-password"
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

        <div className="space-y-2">
          <Label htmlFor="confirmPassword">确认密码</Label>
          <div className="relative">
            <Input
              id="confirmPassword"
              type={showConfirmPassword ? 'text' : 'password'}
              placeholder="再次输入密码"
              autoComplete="new-password"
              disabled={isLoading}
              className="h-11 rounded-lg pr-10"
              {...register('confirmPassword')}
            />
            <button
              type="button"
              onClick={() => setShowConfirmPassword(!showConfirmPassword)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
            >
              {showConfirmPassword ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          </div>
          {errors.confirmPassword && (
            <p className="text-sm text-destructive">{errors.confirmPassword.message}</p>
          )}
        </div>

        {/* Location Notice */}
        <div className="flex items-start gap-2 p-3 rounded-lg bg-muted/50">
          <MapPin className="h-4 w-4 text-primary mt-0.5 shrink-0" />
          <p className="text-xs text-muted-foreground">
            注册需要获取您的位置信息，以便发现附近的内容和用户。
          </p>
        </div>

        {locationError && (
          <div className="p-3 rounded-lg bg-destructive/10 border border-destructive/20">
            <p className="text-sm text-destructive text-center">{locationError}</p>
          </div>
        )}

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
              注册中...
            </>
          ) : (
            '注册'
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
          <Link href="/login">已有账号？登录</Link>
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
