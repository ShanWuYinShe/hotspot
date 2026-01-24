'use client';

/**
 * 创建群组表单组件 (Client Component)
 * 处理表单状态、文件上传、提交逻辑
 */

import { useEffect, useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { ImagePlus, X, MapPin, Hash, Users, Lock, ArrowLeft } from '@/components/icons';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useToast } from '@/hooks/use-toast';
import { useLocation } from '@/hooks/use-location';
import { useRequireAuth } from '@/hooks/use-require-auth';
import { groupsService } from '@/lib/api/services/groups';
import { uploadService } from '@/lib/api/services/upload';
import { isSuccess } from '@/lib/api/client';

type JoinMethod = 'open' | 'approval' | 'invite_only';

export function CreateGroupForm() {
  const router = useRouter();
  const { toast } = useToast();
  const { currentLocation, refreshLocation } = useLocation();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [avatar, setAvatar] = useState<File | null>(null);
  const [avatarUrl, setAvatarUrl] = useState<string>('');
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  const [joinMethod, setJoinMethod] = useState<JoinMethod>('open');
  const [password, setPassword] = useState('');
  const [maxMembers, setMaxMembers] = useState(200);
  const [isSearchable, setIsSearchable] = useState(true);

  useEffect(() => {
    if (!currentLocation) {
      refreshLocation(false);
    }
  }, [currentLocation, refreshLocation]);

  const handleAvatarSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setAvatar(file);
      setAvatarUrl(URL.createObjectURL(file));
    }
  };

  const handleRemoveAvatar = () => {
    if (avatarUrl) {
      URL.revokeObjectURL(avatarUrl);
    }
    setAvatar(null);
    setAvatarUrl('');
  };

  const handleAddTag = () => {
    const tag = tagInput.trim().replace(/^#/, '');
    if (tag && !tags.includes(tag) && tags.length < 10) {
      setTags((prev) => [...prev, tag]);
      setTagInput('');
    }
  };

  const handleRemoveTag = (tag: string) => {
    setTags((prev) => prev.filter((t) => t !== tag));
  };

  const handleSubmit = async () => {
    if (!name.trim()) {
      toast.error('请输入群组名称');
      return;
    }

    if (!requireAuth('创建群组')) {
      return;
    }

    startTransition(async () => {
      try {
        let avatarFileId: string | undefined;
        if (avatar) {
          const uploadResult = await uploadService.uploadFile(avatar, 'group_avatar');
          avatarFileId = uploadResult.file_id;
        }

        const result = await groupsService.create({
          name: name.trim(),
          description: description.trim() || undefined,
          avatar_file_id: avatarFileId,
          tags: tags.length > 0 ? tags : undefined,
          settings: {
            join_method: joinMethod,
            is_searchable: isSearchable,
            max_members: maxMembers,
            password: joinMethod === 'approval' && password ? password : undefined,
            longitude: currentLocation?.longitude,
            latitude: currentLocation?.latitude,
            location_name: currentLocation?.location_name,
          },
        });

        if (isSuccess(result)) {
          toast.success('创建成功');
          router.push(`/groups/${result.data.group_chat_id}`);
        } else {
          toast.error('创建失败', result.message);
        }
      } catch (error) {
        toast.error('创建失败', error instanceof Error ? error.message : '未知错误');
      }
    });
  };

  return (
    <div className="max-w-4xl mx-auto">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.back()}>
              <ArrowLeft className="h-5 w-5" />
            </Button>
            <CardTitle>创建群组</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* 群头像 */}
          <div className="space-y-2">
            <Label>群头像</Label>
            <div className="flex items-center gap-4">
              {avatarUrl ? (
                <div className="relative h-20 w-20 rounded-full overflow-hidden">
                  <Image src={avatarUrl} alt="" fill className="object-cover" />
                  <Button
                    variant="destructive"
                    size="icon"
                    className="absolute top-0 right-0 h-6 w-6"
                    onClick={handleRemoveAvatar}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ) : (
                <label className="h-20 w-20 rounded-full border-2 border-dashed border-muted-foreground/25 flex items-center justify-center cursor-pointer hover:border-muted-foreground/50 transition-colors">
                  <ImagePlus className="h-8 w-8 text-muted-foreground" />
                  <input
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={handleAvatarSelect}
                  />
                </label>
              )}
            </div>
          </div>

          {/* 群名称 */}
          <div className="space-y-2">
            <Label>群名称 *</Label>
            <Input
              placeholder="请输入群名称"
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={50}
            />
          </div>

          {/* 群简介 */}
          <div className="space-y-2">
            <Label>群简介</Label>
            <Textarea
              placeholder="介绍一下这个群组..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="resize-none"
              maxLength={200}
            />
          </div>

          {/* 标签 */}
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

          {/* 加群方式 */}
          <div className="space-y-2">
            <Label>加群方式</Label>
            <Select value={joinMethod} onValueChange={(v) => setJoinMethod(v as JoinMethod)}>
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

          {/* 入群密码 */}
          {joinMethod === 'approval' && (
            <div className="space-y-2">
              <Label>入群密码（可选）</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  type="password"
                  placeholder="设置入群密码"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="pl-9"
                />
              </div>
            </div>
          )}

          {/* 搜索可见性 */}
          <div className="space-y-2">
            <Label>搜索可见性</Label>
            <Select
              value={isSearchable ? 'true' : 'false'}
              onValueChange={(value) => setIsSearchable(value === 'true')}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="true">可被搜索</SelectItem>
                <SelectItem value="false">不可搜索</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* 最大成员数 */}
          <div className="space-y-2">
            <Label>最大成员数</Label>
            <Select value={String(maxMembers)} onValueChange={(v) => setMaxMembers(Number(v))}>
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

          {/* 位置 */}
          <div className="space-y-2">
            <Label>位置</Label>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <MapPin className="h-4 w-4" />
              <span>
                {currentLocation
                  ? currentLocation.location_name || '当前位置'
                  : '正在获取位置...'}
              </span>
            </div>
          </div>

          {/* 提交按钮 */}
          <Button
            className="w-full"
            size="lg"
            onClick={handleSubmit}
            disabled={isPending || !name.trim()}
          >
            <Users className="h-4 w-4 mr-2" />
            {isPending ? '创建中...' : '创建群组'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
