/**
 * 反馈状态管理（Toast 等）
 */

import { create } from 'zustand';
import type { FeedbackState, ToastMessage } from '@/types/client';

interface FeedbackStore extends FeedbackState {
  // Actions
  addToast: (toast: Omit<ToastMessage, 'id'>) => void;
  removeToast: (id: string) => void;
  clearToasts: () => void;
  // Convenience methods
  success: (title: string, description?: string) => void;
  error: (title: string, description?: string) => void;
  warning: (title: string, description?: string) => void;
  info: (title: string, description?: string) => void;
}

let toastIdCounter = 0;

function generateToastId(): string {
  return `toast-${Date.now()}-${++toastIdCounter}`;
}

export const useFeedbackStore = create<FeedbackStore>((set) => ({
  // State
  toasts: [],

  // Actions
  addToast: (toast) => {
    const id = generateToastId();
    const newToast: ToastMessage = {
      id,
      duration: 5000,
      ...toast,
    };

    set((state) => ({
      toasts: [...state.toasts, newToast],
    }));

    // 自动移除
    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }));
      }, newToast.duration);
    }
  },

  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    }));
  },

  clearToasts: () => set({ toasts: [] }),

  // Convenience methods
  success: (title, description) => {
    set((state) => {
      const id = generateToastId();
      const toast: ToastMessage = {
        id,
        type: 'success',
        title,
        description,
        duration: 5000,
      };

      setTimeout(() => {
        set((s) => ({
          toasts: s.toasts.filter((t) => t.id !== id),
        }));
      }, 5000);

      return { toasts: [...state.toasts, toast] };
    });
  },

  error: (title, description) => {
    set((state) => {
      const id = generateToastId();
      const toast: ToastMessage = {
        id,
        type: 'error',
        title,
        description,
        duration: 8000, // 错误消息显示更长时间
      };

      setTimeout(() => {
        set((s) => ({
          toasts: s.toasts.filter((t) => t.id !== id),
        }));
      }, 8000);

      return { toasts: [...state.toasts, toast] };
    });
  },

  warning: (title, description) => {
    set((state) => {
      const id = generateToastId();
      const toast: ToastMessage = {
        id,
        type: 'warning',
        title,
        description,
        duration: 6000,
      };

      setTimeout(() => {
        set((s) => ({
          toasts: s.toasts.filter((t) => t.id !== id),
        }));
      }, 6000);

      return { toasts: [...state.toasts, toast] };
    });
  },

  info: (title, description) => {
    set((state) => {
      const id = generateToastId();
      const toast: ToastMessage = {
        id,
        type: 'info',
        title,
        description,
        duration: 5000,
      };

      setTimeout(() => {
        set((s) => ({
          toasts: s.toasts.filter((t) => t.id !== id),
        }));
      }, 5000);

      return { toasts: [...state.toasts, toast] };
    });
  },
}));
