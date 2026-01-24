'use client';

import { AlertTriangle } from '@/components/icons';
import { Button } from '@/components/ui/button';

interface FeedErrorProps {
  error?: string;
}

export function FeedError({ error }: FeedErrorProps) {
  const handleRetry = () => {
    window.location.reload();
  };

  return (
    <div className="flex min-h-[400px] items-center justify-center">
      <div className="max-w-md w-full space-y-6 rounded-2xl border bg-card p-8 text-center shadow-lg">
        <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-destructive/10">
          <AlertTriangle className="h-8 w-8 text-destructive" />
        </div>
        <div className="space-y-2">
          <h2 className="text-xl font-semibold">加载动态失败</h2>
          <p className="text-sm text-muted-foreground leading-relaxed">
            {error || '无法获取动态信息，可能是网络连接问题或服务器暂时不可用。'}
          </p>
        </div>
        <Button onClick={handleRetry} className="w-full" size="lg">
          重试
        </Button>
      </div>
    </div>
  );
}
