'use client';

/**
 * 瞬间菜单 (Client Component)
 * 处理编辑、删除、举报等操作
 */

import Link from 'next/link';
import { MoreHorizontal } from '@/components/icons';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useRequireAuth } from '@/hooks/use-require-auth';

interface MomentMenuProps {
  momentId: string;
  isOwner: boolean;
  onReport?: (momentId: string) => void;
  onDelete?: (momentId: string) => void;
}

export function MomentMenu({ momentId, isOwner, onReport, onDelete }: MomentMenuProps) {
  const { requireAuth } = useRequireAuth();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="h-8 w-8" aria-label="更多操作">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {isOwner ? (
          <>
            <DropdownMenuItem asChild>
              <Link href={`/moments/${momentId}/edit`}>编辑</Link>
            </DropdownMenuItem>
            <DropdownMenuItem
              className="text-destructive"
              onClick={() => {
                if (!requireAuth('删除瞬间')) return;
                onDelete?.(momentId);
              }}
            >
              删除
            </DropdownMenuItem>
          </>
        ) : (
          <DropdownMenuItem
            onClick={() => {
              if (!requireAuth('举报')) return;
              onReport?.(momentId);
            }}
          >
            举报
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
