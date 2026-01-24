'use client';

/**
 * 登录提示卡片
 */

import { LogIn } from '@/components/icons';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { useAuthPromptStore } from '@/stores/auth-prompt';

interface LoginRequiredCardProps {
  actionName?: string;
  description?: string;
}

export function LoginRequiredCard({ actionName, description }: LoginRequiredCardProps) {
  const { openPrompt } = useAuthPromptStore();

  return (
    <Card>
      <CardContent className="p-6 text-center space-y-3">
        <p className="text-sm text-muted-foreground">
          {description ?? `登录后即可${actionName ?? '查看内容'}。`}
        </p>
        <Button onClick={() => openPrompt({ actionName, description })}>
          <LogIn className="h-4 w-4 mr-2" />
          立即登录
        </Button>
      </CardContent>
    </Card>
  );
}
