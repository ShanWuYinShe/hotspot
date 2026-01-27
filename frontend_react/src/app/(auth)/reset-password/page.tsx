'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import * as z from 'zod';
import { useRouter } from 'next/navigation';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { AuthService } from '@/lib/api/services/auth';

export default function ResetPasswordPage() {
  const router = useRouter();
  const t = useTranslations('auth.forgot');
  const tCommon = useTranslations('common');

  const [codeId, setCodeId] = useState<string | null>(null);
  const [verifyToken, setVerifyToken] = useState<string | null>(null);

  const requestSchema = z.object({
    login_id: z.string().min(1, tCommon('empty')),
  });

  const verifySchema = z.object({
    code: z.string().min(1, tCommon('empty')),
  });

  const resetSchema = z.object({
    new_password: z.string().min(6, tCommon('empty')),
  });

  const requestForm = useForm<z.infer<typeof requestSchema>>({
    resolver: zodResolver(requestSchema),
    defaultValues: { login_id: '' },
  });

  const verifyForm = useForm<z.infer<typeof verifySchema>>({
    resolver: zodResolver(verifySchema),
    defaultValues: { code: '' },
  });

  const resetForm = useForm<z.infer<typeof resetSchema>>({
    resolver: zodResolver(resetSchema),
    defaultValues: { new_password: '' },
  });

  const handleRequest = async (values: z.infer<typeof requestSchema>) => {
    const res = await AuthService.requestPasswordReset(values);
    if (res.code === 0 && res.data) {
      setCodeId(res.data.code_id);
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const handleVerify = async (values: z.infer<typeof verifySchema>) => {
    if (!codeId) {
      return;
    }
    const res = await AuthService.verifyResetCode({ code_id: codeId, code: values.code });
    if (res.code === 0 && res.data) {
      setVerifyToken(res.data.verify_token);
      toast.success(tCommon('saved'));
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  const handleReset = async (values: z.infer<typeof resetSchema>) => {
    if (!verifyToken) {
      return;
    }
    const res = await AuthService.resetPassword({ verify_token: verifyToken, new_password: values.new_password });
    if (res.code === 0) {
      toast.success(tCommon('saved'));
      router.push('/login');
    } else {
      toast.error(res.message || tCommon('retry'));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('title')}</CardTitle>
        <CardDescription>{t('submit')}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {!codeId ? (
          <form onSubmit={requestForm.handleSubmit(handleRequest)} className="space-y-3">
            <Input placeholder={t('loginId')} {...requestForm.register('login_id')} />
            <Button type="submit" className="w-full">
              {t('submit')}
            </Button>
          </form>
        ) : !verifyToken ? (
          <form onSubmit={verifyForm.handleSubmit(handleVerify)} className="space-y-3">
            <Input placeholder={t('code')} {...verifyForm.register('code')} />
            <Button type="submit" className="w-full">
              {t('verify')}
            </Button>
          </form>
        ) : (
          <form onSubmit={resetForm.handleSubmit(handleReset)} className="space-y-3">
            <Input type="password" placeholder={t('newPassword')} {...resetForm.register('new_password')} />
            <Button type="submit" className="w-full">
              {tCommon('save')}
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
