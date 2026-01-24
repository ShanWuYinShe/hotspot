'use client';

/**
 * 群组设置表单 (Client Component)
 */

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { Save } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { groupsService } from '@/lib/api/services/groups';
import { isSuccess } from '@/lib/api/client';
import type { GroupDetailData } from '@/types/api';

interface GroupSettingsFormProps {
  group: GroupDetailData;
}

export function GroupSettingsForm({ group }: GroupSettingsFormProps) {
  const router = useRouter();
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  const [name, setName] = useState(group.name);
  const [description, setDescription] = useState(group.description ?? '');
  const [joinMethod, setJoinMethod] = useState<'open' | 'approval' | 'invite_only'>(
    group.settings?.join_method ?? 'open'
  );
  const [isSearchable, setIsSearchable] = useState(
    group.settings?.is_searchable ?? true
  );
  const [maxMembers, setMaxMembers] = useState(
    group.settings?.max_members ?? 200
  );
  const [password, setPassword] = useState(group.settings?.password ?? '');

  const handleSubmit = () => {
    if (!requireAuth('管理群组')) return;

    startTransition(async () => {
      const response = await groupsService.update(group.group_chat_id, {
        name,
        description: description || undefined,
        settings: {
          join_method: joinMethod,
          is_searchable: isSearchable,
          max_members: maxMembers,
          password: joinMethod === 'approval' && password ? password : undefined,
        },
      });

      if (isSuccess(response)) {
        toast.success('保存成功');
        router.refresh();
      } else {
        toast.error('保存失败', response.message);
      }
    });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Label>群名称</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} />
      </div>

      <div className="space-y-2">
        <Label>群简介</Label>
        <Textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
        />
      </div>

      <div className="space-y-2">
        <Label>加群方式</Label>
        <Select value={joinMethod} onValueChange={(v) => setJoinMethod(v as 'open' | 'approval' | 'invite_only')}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="open">公开加入</SelectItem>
            <SelectItem value="approval">需要审批</SelectItem>
            <SelectItem value="invite_only">仅邀请</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {joinMethod === 'approval' && (
        <div className="space-y-2">
          <Label>入群密码（可选）</Label>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </div>
      )}

      <div className="space-y-2">
        <Label>可搜索</Label>
        <Select
          value={String(isSearchable)}
          onValueChange={(v) => setIsSearchable(v === 'true')}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="true">允许搜索</SelectItem>
            <SelectItem value="false">仅邀请</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>最大成员数</Label>
        <Select
          value={String(maxMembers)}
          onValueChange={(v) => setMaxMembers(Number(v))}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="50">50 人</SelectItem>
            <SelectItem value="100">100 人</SelectItem>
            <SelectItem value="200">200 人</SelectItem>
            <SelectItem value="500">500 人</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Button className="w-full" onClick={handleSubmit} disabled={isPending}>
        <Save className="h-4 w-4 mr-2" />
        {isPending ? '保存中...' : '保存设置'}
      </Button>
    </div>
  );
}
