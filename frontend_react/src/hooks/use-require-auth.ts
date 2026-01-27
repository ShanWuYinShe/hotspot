'use client';

import { useCallback } from 'react';
import { useRouter } from 'next/navigation';

import { useAuthStore } from '@/stores/auth';
import { useLoginModal } from '@/providers/login-modal-provider';

interface RequireAuthOptions {
  showModal?: boolean;
  redirectAfterLogin?: boolean;
}

export function useRequireAuth() {
  const router = useRouter();
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const { requireAuth: openLoginModal } = useLoginModal();

  const requireAuth = useCallback(
    (
      onSuccess?: () => void,
      options: RequireAuthOptions = {}
    ): boolean => {
      const { showModal = true } = options;

      if (isAuthenticated) {
        onSuccess?.();
        return true;
      }

      if (showModal) {
        openLoginModal(onSuccess);
      } else {
        router.push('/login');
      }

      return false;
    },
    [isAuthenticated, openLoginModal, router]
  );

  const checkPermission = useCallback(
    (action: string): boolean => {
      const guestAllowedActions: string[] = [];
      if (guestAllowedActions.includes(action)) {
        return true;
      }
      return isAuthenticated;
    },
    [isAuthenticated]
  );

  return {
    requireAuth,
    isAuthenticated,
    checkPermission,
  };
}
