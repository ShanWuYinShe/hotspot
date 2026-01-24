'use client';

/**
 * 新建瞬间按钮 (Client Component)
 */

import { useRouter } from 'next/navigation';
import { Plus } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { useRequireAuth } from '@/hooks/use-require-auth';

interface CreateMomentButtonProps {
  label: string;
}

export function CreateMomentButton({ label }: CreateMomentButtonProps) {
  const router = useRouter();
  const { requireAuth } = useRequireAuth();

  return (
    <Button
      onClick={() => {
        if (!requireAuth('发布瞬间')) return;
        router.push('/moments/create');
      }}
    >
      <Plus className="h-4 w-4 mr-2" />
      {label}
    </Button>
  );
}
