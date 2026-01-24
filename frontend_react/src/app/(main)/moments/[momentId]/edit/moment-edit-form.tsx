'use client';

/**
 * 瞬间编辑表单 (Client Component)
 */

import { useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import { Save, Hash, X } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { momentsService } from '@/lib/api/services/moments';
import { isSuccess } from '@/lib/api/client';
import type { MomentItem, PrivacyLevel } from '@/types/api';

interface MomentEditFormProps {
  moment: MomentItem;
}

export function MomentEditForm({ moment }: MomentEditFormProps) {
  const router = useRouter();
  const { toast } = useToast();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  const [description, setDescription] = useState(moment.description);
  const [tags, setTags] = useState(moment.tags ?? []);
  const [tagInput, setTagInput] = useState('');
  const [privacy, setPrivacy] = useState<PrivacyLevel>(moment.privacy_level);

  const handleAddTag = () => {
    const tag = tagInput.trim().replace(/^#/, '');
    if (tag && !tags.includes(tag)) {
      setTags((prev) => [...prev, tag]);
      setTagInput('');
    }
  };

  const handleRemoveTag = (tag: string) => {
    setTags((prev) => prev.filter((t) => t !== tag));
  };

  const handleSubmit = () => {
    if (!requireAuth('编辑瞬间')) return;
    if (!description.trim()) {
      toast.error('请输入内容');
      return;
    }

    startTransition(async () => {
      const response = await momentsService.update(moment.moment_id, {
        description: description.trim(),
        tags: tags.length > 0 ? tags : undefined,
        privacy_level: privacy,
      });

      if (isSuccess(response)) {
        toast.success('保存成功');
        router.push(`/moments/${moment.moment_id}`);
      } else {
        toast.error('保存失败', response.message);
      }
    });
  };

  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Label>内容</Label>
        <Textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={4}
          className="resize-none"
          maxLength={500}
        />
      </div>

      <div className="space-y-2">
        <Label>标签</Label>
        <div className="flex gap-2">
          <Input
            placeholder="添加标签"
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddTag())}
          />
          <Button variant="outline" onClick={handleAddTag}>
            <Hash className="h-4 w-4" />
          </Button>
        </div>
        {tags.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-2">
            {tags.map((tag) => (
              <Badge key={tag} variant="secondary" className="gap-1">
                #{tag}
                <button onClick={() => handleRemoveTag(tag)}>
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            ))}
          </div>
        )}
      </div>

      <div className="space-y-2">
        <Label>可见范围</Label>
        <Select value={privacy} onValueChange={(v) => setPrivacy(v as PrivacyLevel)}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="public">公开</SelectItem>
            <SelectItem value="friends">仅好友</SelectItem>
            <SelectItem value="private">仅自己</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Button className="w-full" onClick={handleSubmit} disabled={isPending}>
        <Save className="h-4 w-4 mr-2" />
        {isPending ? '保存中...' : '保存修改'}
      </Button>
    </div>
  );
}
