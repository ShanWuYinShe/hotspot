'use client';

import { useEffect, useState, useCallback } from 'react';
import { AlertTriangle, RefreshCw, Home } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface ErrorProps {
  error: Error & { digest?: string };
  reset: () => void;
}

interface ErrorInfo {
  type: 'network' | 'timeout' | 'server' | 'unknown';
  title: string;
  description: string;
  showRetry: boolean;
}

export default function Error({ error, reset }: ErrorProps) {
  const [errorInfo, setErrorInfo] = useState<ErrorInfo>({
    type: 'unknown',
    title: '页面出错了',
    description: '抱歉，页面加载时遇到了问题。请稍后再试。',
    showRetry: true,
  });

  useEffect(() => {
    const message = error.message || error.toString();
    let info: ErrorInfo;

    if (message.includes('timeout') || message.includes('AbortError')) {
      info = {
        type: 'timeout',
        title: '请求超时',
        description: '服务器响应时间过长，请检查网络连接或稍后再试。',
        showRetry: true,
      };
    } else if (message.includes('fetch') || message.includes('Network')) {
      info = {
        type: 'network',
        title: '网络连接失败',
        description: '无法连接到服务器，请检查网络连接后重试。',
        showRetry: true,
      };
    } else if (message.includes('500') || message.includes('502') || message.includes('503')) {
      info = {
        type: 'server',
        title: '服务器错误',
        description: '服务器暂时无法处理请求，请稍后再试。',
        showRetry: true,
      };
    } else {
      info = {
        type: 'unknown',
        title: '页面出错了',
        description: message || '抱歉，页面加载时遇到了问题。请稍后再试。',
        showRetry: true,
      };
    }

    setErrorInfo(info);
    console.error('Error caught:', error, info);
  }, [error]);

  const handleRetry = useCallback(() => {
    reset();
  }, [reset]);

  const handleGoHome = useCallback(() => {
    window.location.href = '/';
  }, []);

  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-6 px-6">
      <div className="max-w-md w-full space-y-6 rounded-2xl border bg-card p-8 text-center shadow-lg">
        <div
          className={cn(
            'mx-auto flex h-16 w-16 items-center justify-center rounded-full',
            errorInfo.type === 'network' && 'bg-orange-100 dark:bg-orange-950',
            errorInfo.type === 'timeout' && 'bg-yellow-100 dark:bg-yellow-950',
            errorInfo.type === 'server' && 'bg-red-100 dark:bg-red-950',
            errorInfo.type === 'unknown' && 'bg-destructive/10'
          )}
        >
          <AlertTriangle
            className={cn(
              'h-8 w-8',
              errorInfo.type === 'network' && 'text-orange-600 dark:text-orange-400',
              errorInfo.type === 'timeout' && 'text-yellow-600 dark:text-yellow-400',
              errorInfo.type === 'server' && 'text-red-600 dark:text-red-400',
              errorInfo.type === 'unknown' && 'text-destructive'
            )}
          />
        </div>

        <div className="space-y-3">
          <h2 className="text-2xl font-semibold">{errorInfo.title}</h2>
          <p className="text-sm text-muted-foreground leading-relaxed">{errorInfo.description}</p>
        </div>

        {process.env.NODE_ENV === 'development' && error.stack && (
          <details className="rounded-lg bg-muted p-4 text-left">
            <summary className="cursor-pointer text-sm font-medium">错误详情（开发模式）</summary>
            <pre className="mt-2 overflow-x-auto text-xs">{error.stack}</pre>
          </details>
        )}

        <div className="flex flex-col gap-3 sm:flex-row">
          {errorInfo.showRetry && (
            <Button onClick={handleRetry} className="flex-1" size="lg">
              <RefreshCw className="mr-2 h-4 w-4" />
              重试
            </Button>
          )}
          <Button onClick={handleGoHome} variant="outline" className="flex-1" size="lg">
            <Home className="mr-2 h-4 w-4" />
            返回首页
          </Button>
        </div>
      </div>
    </div>
  );
}
