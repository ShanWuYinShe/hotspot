'use client';

import { useFeedbackStore } from '@/stores/feedback';

/**
 * Toast Hook - 简化 Toast 使用
 */
export function useToast() {
  const { success, error, warning, info, addToast, removeToast, clearToasts, toasts } = useFeedbackStore();

  return {
    toasts,
    toast: {
      success,
      error,
      warning,
      info,
      custom: addToast,
    },
    dismiss: removeToast,
    dismissAll: clearToasts,
  };
}
