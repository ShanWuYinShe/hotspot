'use client';

/**
 * 登录提示弹窗
 */

import { useMemo } from 'react';
import { usePathname, useSearchParams, useRouter } from 'next/navigation';
import { LogIn, UserPlus } from '@/components/icons';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useAuthPromptStore } from '@/stores/auth-prompt';

export function LoginPromptDialog() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { isOpen, actionName, description, redirectPath, closePrompt } =
    useAuthPromptStore();

  const currentPath = useMemo(() => {
    const query = searchParams?.toString();
    return query ? `${pathname}?${query}` : pathname;
  }, [pathname, searchParams]);

  const redirectTo = redirectPath ?? currentPath;

  const handleLogin = () => {
    closePrompt();
    router.push(`/login?redirect=${encodeURIComponent(redirectTo)}`);
  };

  const handleRegister = () => {
    closePrompt();
    router.push(`/register?redirect=${encodeURIComponent(redirectTo)}`);
  };

  return (
    <Dialog open={isOpen} onOpenChange={(open) => !open && closePrompt()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>需要登录</DialogTitle>
          <DialogDescription>
            {description ?? `登录后即可${actionName ?? '继续操作'}。`}
          </DialogDescription>
        </DialogHeader>
        <div className="rounded-lg bg-muted p-3 text-sm text-muted-foreground">
          登录后可进行互动、同步消息与个性化设置。
        </div>
        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={handleRegister}>
            <UserPlus className="h-4 w-4 mr-2" />
            立即注册
          </Button>
          <Button onClick={handleLogin}>
            <LogIn className="h-4 w-4 mr-2" />
            立即登录
          </Button>
          <Button variant="ghost" onClick={closePrompt}>
            继续浏览
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
