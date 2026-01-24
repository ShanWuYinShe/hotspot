'use client';

import { useEffect, useState } from 'react';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

interface GroupPasswordDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (password: string) => void;
  isPending?: boolean;
  title?: string;
  description?: string;
  confirmText?: string;
}

export function GroupPasswordDialog({
  open,
  onOpenChange,
  onSubmit,
  isPending = false,
  title = '需要密码',
  description = '该群组开启了加入密码，请输入后继续。',
  confirmText = '确认加入',
}: GroupPasswordDialogProps) {
  const [password, setPassword] = useState('');

  useEffect(() => {
    if (!open) {
      setPassword('');
    }
  }, [open]);

  const handleSubmit = () => {
    const value = password.trim();
    if (!value) return;
    onSubmit(value);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <form
          onSubmit={(event) => {
            event.preventDefault();
            handleSubmit();
          }}
          className="space-y-4"
        >
          <Input
            type="password"
            placeholder="输入群组密码"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoFocus
          />
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" disabled={isPending || !password.trim()}>
              {isPending ? '处理中...' : confirmText}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
