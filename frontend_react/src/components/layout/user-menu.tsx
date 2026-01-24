'use client';

/**
 * 用户菜单组件 (Client Component)
 * 需要 useAuth hook 获取当前用户状态
 */

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { User, Settings, LogOut, Plus } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { UserAvatar } from '@/components/common/user-avatar';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useAuth } from '@/hooks/use-auth';

export function UserMenu() {
  const t = useTranslations();
  const { currentAccount, accounts, switchAccount, logout } = useAuth();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="gap-2">
          {currentAccount ? (
            <UserAvatar
              user={{ nickname: currentAccount.nickname, avatar_url: currentAccount.avatar_url }}
              size="sm"
            />
          ) : (
            <User className="h-4 w-4" />
          )}
          <span className="hidden lg:inline">{currentAccount?.nickname ?? '用户'}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        {currentAccount ? (
          <>
            <DropdownMenuItem asChild>
              <Link href={`/users/${currentAccount.user_id}`} className="gap-2">
                <User className="h-4 w-4" />
                {t('nav.profile')}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/settings" className="gap-2">
                <Settings className="h-4 w-4" />
                {t('nav.settings')}
              </Link>
            </DropdownMenuItem>

            {accounts.length > 1 && (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuLabel className="text-xs text-muted-foreground">
                  账号切换
                </DropdownMenuLabel>
                {accounts.map((account) => (
                  <DropdownMenuItem
                    key={account.user_id}
                    onClick={() => switchAccount(account.user_id)}
                    disabled={account.user_id === currentAccount.user_id}
                  >
                    {account.nickname}
                  </DropdownMenuItem>
                ))}
              </>
            )}

            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href="/login" className="gap-2">
                <Plus className="h-4 w-4" />
                添加账号
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem onClick={logout} className="gap-2 text-destructive">
              <LogOut className="h-4 w-4" />
              {t('auth.logout')}
            </DropdownMenuItem>
          </>
        ) : (
          <>
            <DropdownMenuItem asChild>
              <Link href="/login" className="gap-2">
                <User className="h-4 w-4" />
                {t('auth.login.submit')}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/register" className="gap-2">
                <Plus className="h-4 w-4" />
                {t('auth.register.submit')}
              </Link>
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
