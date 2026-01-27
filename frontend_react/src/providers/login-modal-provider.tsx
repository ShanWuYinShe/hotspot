'use client';

import { createContext, useContext, useState, ReactNode, useCallback } from 'react';
import { useAuth } from '@/hooks/use-auth';
import { LoginDialog } from '@/components/modals/login-dialog';

interface LoginModalContextType {
  open: boolean;
  setOpen: (open: boolean) => void;
  requireAuth: (onSuccess?: () => void) => void;
}

const LoginModalContext = createContext<LoginModalContextType | undefined>(undefined);

export function LoginModalProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const [pendingAction, setPendingAction] = useState<(() => void) | null>(null);
  const { isAuthenticated } = useAuth();

  const handleLoginSuccess = useCallback(() => {
    if (pendingAction) {
      pendingAction();
      setPendingAction(null);
    }
  }, [pendingAction]);

  const requireAuth = useCallback((onSuccess?: () => void) => {
    if (isAuthenticated) {
      onSuccess?.();
    } else {
      if (onSuccess) {
        setPendingAction(() => onSuccess);
      }
      setOpen(true);
    }
  }, [isAuthenticated]);

  const handleOpenChange = useCallback((val: boolean) => {
    setOpen(val);
    if (!val) {
      setPendingAction(null);
    }
  }, []);

  return (
    <LoginModalContext.Provider value={{ open, setOpen, requireAuth }}>
      {children}
      <LoginDialog 
        open={open} 
        onOpenChange={handleOpenChange} 
        onSuccess={handleLoginSuccess} 
      />
    </LoginModalContext.Provider>
  );
}

export function useLoginModal() {
  const context = useContext(LoginModalContext);
  if (context === undefined) {
    throw new Error('useLoginModal must be used within a LoginModalProvider');
  }
  return context;
}
