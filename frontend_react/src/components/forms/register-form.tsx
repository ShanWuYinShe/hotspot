'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useTranslations } from 'next-intl';
import { Loader } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuth } from '@/hooks/use-auth';

const registerSchema = z
  .object({
    login_id: z.string().min(3, '用户名至少3个字符').max(20, '用户名最多20个字符'),
    nickname: z.string().min(1, '请输入昵称').max(20, '昵称最多20个字符'),
    password: z.string().min(6, '密码至少6个字符'),
    confirmPassword: z.string(),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: '两次输入的密码不一致',
    path: ['confirmPassword'],
  });

type RegisterFormData = z.infer<typeof registerSchema>;

interface RegisterFormProps {
  redirectTo?: string;
}

export function RegisterForm({ redirectTo }: RegisterFormProps) {
  const t = useTranslations('auth.register');
  const { register: registerUser } = useAuth();
  const [isLoading, setIsLoading] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
  });

  const onSubmit = async (data: RegisterFormData) => {
    setIsLoading(true);
    try {
      await registerUser(data.login_id, data.nickname, data.password, redirectTo);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="login_id">{t('loginId')}</Label>
        <Input
          id="login_id"
          type="text"
          placeholder={t('loginId')}
          disabled={isLoading}
          {...register('login_id')}
        />
        {errors.login_id && (
          <p className="text-sm text-destructive">{errors.login_id.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="nickname">昵称</Label>
        <Input
          id="nickname"
          type="text"
          placeholder="昵称"
          disabled={isLoading}
          {...register('nickname')}
        />
        {errors.nickname && (
          <p className="text-sm text-destructive">{errors.nickname.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="password">{t('password')}</Label>
        <Input
          id="password"
          type="password"
          placeholder={t('password')}
          disabled={isLoading}
          {...register('password')}
        />
        {errors.password && (
          <p className="text-sm text-destructive">{errors.password.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="confirmPassword">{t('confirmPassword')}</Label>
        <Input
          id="confirmPassword"
          type="password"
          placeholder={t('confirmPassword')}
          disabled={isLoading}
          {...register('confirmPassword')}
        />
        {errors.confirmPassword && (
          <p className="text-sm text-destructive">{errors.confirmPassword.message}</p>
        )}
      </div>

      <Button type="submit" className="w-full" disabled={isLoading}>
        {isLoading && <Loader className="mr-2 h-4 w-4 animate-spin" />}
        {t('submit')}
      </Button>
    </form>
  );
}
