'use client';

import { useEffect } from 'react';

import { useAuth } from '@/hooks/use-auth';
import { useWebSocket } from '@/hooks/use-websocket';

export function AppBootstrap() {
  const { bootstrap } = useAuth();
  useWebSocket();

  useEffect(() => {
    bootstrap();
  }, [bootstrap]);

  return null;
}
