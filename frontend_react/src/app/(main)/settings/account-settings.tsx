'use client';

/**
 * 账户设置组件 (Client Component)
 * 处理账户相关操作
 */

import { useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';
import { Trash } from '@/components/icons';
import { CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import { useRequireAuth } from '@/hooks/use-require-auth';

export function AccountSettings() {
  const t = useTranslations('settings');
  const router = useRouter();
  const { requireAuth } = useRequireAuth();

  return (
    <CardContent className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <Label>{t('changePassword')}</Label>
          <p className="text-sm text-muted-foreground">修改账户密码</p>
        </div>
        <Button
          variant="outline"
          onClick={() => {
            if (!requireAuth('修改密码')) return;
            router.push('/settings/password');
          }}
        >
          修改
        </Button>
      </div>
      <Separator />
      <div className="flex items-center justify-between">
        <div>
          <Label className="text-destructive">{t('deleteAccount')}</Label>
          <p className="text-sm text-muted-foreground">永久删除账户和所有数据</p>
        </div>
        <Button
          variant="destructive"
          onClick={() => {
            if (!requireAuth('删除账号')) return;
          }}
        >
          <Trash className="h-4 w-4 mr-2" />
          删除
        </Button>
      </div>
    </CardContent>
  );
}
