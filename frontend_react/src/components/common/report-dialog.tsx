'use client';

/**
 * 举报弹窗
 */

import { useEffect, useMemo, useState } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';

interface ReportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: { reason: string; description?: string }) => void | Promise<void>;
  isPending?: boolean;
  targetLabel?: string;
}

const REASONS = [
  { value: '垃圾信息/广告', label: '垃圾信息/广告' },
  { value: '骚扰或仇恨言论', label: '骚扰或仇恨言论' },
  { value: '违规内容', label: '违规内容' },
  { value: '侵犯隐私', label: '侵犯隐私' },
  { value: '其他', label: '其他' },
] as const;

export function ReportDialog({
  open,
  onOpenChange,
  onSubmit,
  isPending = false,
  targetLabel = '内容',
}: ReportDialogProps) {
  const defaultReason = useMemo(() => REASONS[0]?.value ?? '其他', []);
  const [reason, setReason] = useState<string>(defaultReason);
  const [description, setDescription] = useState('');

  useEffect(() => {
    if (open) {
      setReason(defaultReason);
      setDescription('');
    }
  }, [open, defaultReason]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>举报{targetLabel}</DialogTitle>
          <DialogDescription>请选择原因并补充说明，便于我们更快处理。</DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <div className="text-sm font-medium">举报原因</div>
            <Select value={reason} onValueChange={setReason}>
              <SelectTrigger>
                <SelectValue placeholder="请选择原因" />
              </SelectTrigger>
              <SelectContent>
                {REASONS.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <div className="text-sm font-medium">补充说明（可选）</div>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="请描述具体问题"
              rows={4}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={isPending}>
            取消
          </Button>
          <Button onClick={() => onSubmit({ reason, description: description.trim() || undefined })} disabled={isPending}>
            提交举报
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
