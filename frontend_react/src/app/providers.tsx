'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Toaster } from '@/components/ui/sonner';
import { ThemeProvider } from '@/components/common/theme-provider';
import { AppBootstrap } from '@/components/common/app-bootstrap';
import { useState } from 'react';

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 60 * 1000,
        retry: false, // API client handles some retries, or we can configure here
      },
    },
  }));

  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <AppBootstrap />
        {children}
        <Toaster />
      </QueryClientProvider>
    </ThemeProvider>
  );
}
