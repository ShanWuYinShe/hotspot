'use client';

/**
 * 瞬间详情菜单 (Client Component)
 * 处理编辑、删除、举报
 */

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { MomentMenu } from '@/components/cards/moment-menu';
import { ReportDialog } from '@/components/common/report-dialog';
import { useAuthStore } from '@/stores/auth';
import { useToast } from '@/hooks/use-toast';
import { deleteMoment } from '@/lib/actions';
import { reportsService } from '@/lib/api/services/reports';
import { isSuccess } from '@/lib/api/client';

interface MomentDetailMenuProps {
  momentId: string;
  ownerId: number;
}

export function MomentDetailMenu({ momentId, ownerId }: MomentDetailMenuProps) {
  const router = useRouter();
  const { toast } = useToast();
  const [isPending, startTransition] = useTransition();
  const [reportOpen, setReportOpen] = useState(false);
  const currentUserId = useAuthStore((state) => state.currentAccount?.user_id);
  const isOwner = currentUserId === ownerId;

  const handleDelete = (id: string) => {
    if (isPending) return;
    startTransition(async () => {
      const result = await deleteMoment(id);
      if (result.code === 0) {
        toast.success('删除成功');
        router.push('/');
      } else {
        toast.error('删除失败', result.message);
      }
    });
  };

  const handleReport = (_id: string) => {
    if (isPending) return;
    setReportOpen(true);
  };

  const handleReportSubmit = (payload: { reason: string; description?: string }) => {
    startTransition(async () => {
      const response = await reportsService.create({
        target_type: 'moment',
        target_id: momentId,
        reason: payload.reason,
        description: payload.description,
      });
      if (isSuccess(response)) {
        toast.success('举报已提交');
        setReportOpen(false);
      } else {
        toast.error('举报失败', response.message);
      }
    });
  };

  return (
    <>
      <MomentMenu
        momentId={momentId}
        isOwner={isOwner}
        onDelete={handleDelete}
        onReport={handleReport}
      />
      <ReportDialog
        open={reportOpen}
        onOpenChange={setReportOpen}
        onSubmit={handleReportSubmit}
        isPending={isPending}
        targetLabel="瞬间"
      />
    </>
  );
}
