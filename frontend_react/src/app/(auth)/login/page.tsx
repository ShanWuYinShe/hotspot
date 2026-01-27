'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';
import Link from 'next/link';

import { Button } from '@/components/ui/button';
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { useAuth } from '@/hooks/use-auth';

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuth();
  const [isLoading, setIsLoading] = useState(false);
  const t = useTranslations('auth.login');
  const tCommon = useTranslations('common');
  const formSchema = z.object({
    login_id: z.string().min(1, tCommon('empty')),
    password: z.string().min(1, tCommon('empty')),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      login_id: '',
      password: '',
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    setIsLoading(true);
    try {
      const response = await login(values);
      if (response.code === 0) {
        toast.success(tCommon('saved'));
        router.push('/feed');
       } else {
        toast.error(response.message || tCommon('retry'));
      }
    } finally {
      setIsLoading(false);
    }

  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('subtitle')}</CardDescription>
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="login_id"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('loginId')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('loginId')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="password"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('password')}</FormLabel>
                  <FormControl>
                    <Input type="password" placeholder={t('password')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" className="w-full" disabled={isLoading}>
              {isLoading ? t('loading') : t('submit')}
            </Button>
          </form>
        </Form>
        <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
          <span>
            {t('noAccount')}{' '}
            <Link href="/register" className="text-primary hover:underline">
              {t('register')}
            </Link>
          </span>
          <Link href="/reset-password" className="text-primary hover:underline">
            {t('reset')}
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}
