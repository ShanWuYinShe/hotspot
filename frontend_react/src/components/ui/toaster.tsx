'use client';

import { useFeedbackStore } from '@/stores/feedback';
import { X, CheckCircle, XCircle, AlertTriangle, Info } from '@/components/icons';
import { cn } from '@/lib/utils';

const iconMap = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
};

const colorMap = {
  success: 'bg-green-50 border-green-200 text-green-800 dark:bg-green-950 dark:border-green-800 dark:text-green-200',
  error: 'bg-red-50 border-red-200 text-red-800 dark:bg-red-950 dark:border-red-800 dark:text-red-200',
  warning: 'bg-yellow-50 border-yellow-200 text-yellow-800 dark:bg-yellow-950 dark:border-yellow-800 dark:text-yellow-200',
  info: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-950 dark:border-blue-800 dark:text-blue-200',
};

export function Toaster() {
  const { toasts, removeToast } = useFeedbackStore();

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((toast, index) => {
        const Icon = iconMap[toast.type];
        return (
          <div
            key={toast.id}
            className={cn(
              'flex items-start gap-3 rounded-lg border p-4 shadow-lg animate-in slide-in-from-right-full duration-300 ease-out',
              colorMap[toast.type],
              `translate-x-0 opacity-100`
            )}
            style={{
              animationDelay: `${index * 50}ms`,
            }}
          >
            <Icon className="h-5 w-5 shrink-0" />
            <div className="flex-1 space-y-1">
              <p className="text-sm font-medium">{toast.title}</p>
              {toast.description && (
                <p className="text-sm opacity-80">{toast.description}</p>
              )}
            </div>
            <button
              onClick={() => removeToast(toast.id)}
              className="shrink-0 rounded-md p-1 transition-colors hover:bg-black/10 dark:hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-ring/60 focus:ring-offset-2"
              aria-label="关闭通知"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        );
      })}
    </div>
  );
}
