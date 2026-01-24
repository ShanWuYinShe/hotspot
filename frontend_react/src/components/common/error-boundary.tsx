'use client';

/**
 * 错误边界组件 (Client Component)
 * 捕获子组件中的错误并提供友好的用户界面
 */

import * as React from 'react';
import { AlertTriangle, RefreshCw, Home } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

interface ErrorBoundaryProps {
  children: React.ReactNode;
  fallback?: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  errorInfo: {
    type: 'network' | 'timeout' | 'server' | 'unknown';
    title: string;
    description: string;
  };
}

class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: {
        type: 'unknown',
        title: '出现错误',
        description: '抱歉，页面加载时遇到了问题。请稍后再试。',
      },
    };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      error,
      errorInfo: {
        type: 'unknown',
        title: '出现错误',
        description: '抱歉，页面加载时遇到了问题。请稍后再试。',
      },
    };
  }

  componentDidCatch(error: Error, errorInfo: any): void {
    this.setState({
      error,
      errorInfo,
    });

    console.error('ErrorBoundary caught an error:', error, errorInfo);
  }

  handleReset = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: {
        type: 'unknown',
        title: '出现错误',
        description: '抱歉，页面加载时 捕获到错误。请刷新或稍后再试。',
      },
    });
  };

  handleReload = (): void => {
    window.location.reload();
  };

  render(): React.ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div className="flex min-h-[60vh] flex flex-col items-center justify-center gap-6 px-6">
          <div className="max-w-md w-full space-y-6 rounded-2xl border bg-card p-8 text-center shadow-lg">
            <div
              className={cn(
                'mx-auto flex h-16 w-16 items-center justify-center rounded-full',
                this.state.errorInfo?.type === 'network' && 'bg-orange-100 dark:bg-orange-950',
                this.state.errorInfo?.type === 'timeout' && 'bg-yellow-100 dark:bg-yellow-950',
                this.state.errorInfo?.type === 'server' && 'bg-red-100 dark:bg-red-950',
                this.state.errorInfo?.type === 'unknown' && 'bg-destructive/10'
              )}
            >
              <AlertTriangle
                className={cn(
                  'h-8 w-8',
                  this.state.errorInfo?.type === 'network' && 'text-orange-600 dark:text-orange-400',
                  this.state.errorInfo?.type === 'timeout' && 'text-yellow-600 dark:text-yellow-400',
                  this.state.errorInfo?.type === 'server' && 'text-red-600 dark:text-red-400',
                  this.state.errorInfo?.z.type === 'unknown' && 'text-destructive'
                )}
              />
            </div>

            <div className="space-y-3">
              <h2 className="text-2xl font-semibold">
                {this.state.errorInfo?.title || '出现错误'}
              </h2>
              <p className="text-sm text-muted-foreground leading-relaxed">
                {this.state.errorInfo?.description || '抱歉，页面加载时遇到了问题。请稍后再试。'}
              </p>
            </div>

            {process.env.NODE_ENV === 'development' && this.state.error?.stack && (
              <details className="rounded-lg bg-muted p-4 text-left">
                <summary className="cursor-pointer text-sm font-medium">
                  错误详情（开发模式）
                </summary>
                <pre className="mt-2 overflow-x-auto text-xs">
                  {this.state.error?.stack}
                </pre>
              </details>
            )}

            <div className="flex flex-col gap-3 sm:flex-row">
              <Button onClick={this.handleReset} variant="outline" className="flex-1">
                重试
              </Button>
              <Button onClick={this.handleReload} className="flex-1">
                <RefreshCw className="mr-2 h-4 w-4" />
                刷新页面
              </Button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export { ErrorBoundary };
