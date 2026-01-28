'use client';

import * as React from 'react';
import { useRouter } from 'next/navigation';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useAuthStore } from '@/stores/auth';
import * as usersService from '@/lib/api/services/users';
import { isSuccess } from '@/lib/api/client';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { useToast } from '@/hooks/use-toast';
import {
  ArrowLeft,
  Loader2,
  Camera,
  X,
} from 'lucide-react';

const profileSchema = z.object({
  nickname: z
    .string()
    .min(1, '请输入昵称')
    .max(32, '昵称最多 32 个字符'),
  bio: z
    .string()
    .max(200, '简介最多 200 个字符')
    .optional(),
  tags: z.array(z.string()).max(5, '最多 5 个标签').optional(),
});

type ProfileFormData = z.infer<typeof profileSchema>;

export default function EditProfilePage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const { user, setUser } = useAuthStore();
  const [tagInput, setTagInput] = React.useState('');

  // Fetch current profile
  const profileQuery = useQuery({
    queryKey: ['my-profile'],
    queryFn: async () => {
      const res = await usersService.getMyProfile();
      if (!isSuccess(res)) throw new Error(res.message);
      return res.data;
    },
  });

  const profile = profileQuery.data;

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isDirty },
  } = useForm<ProfileFormData>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      nickname: profile?.nickname || user?.nickname || '',
      bio: profile?.bio || '',
      tags: profile?.tags || [],
    },
  });

  // Update form when profile loads
  React.useEffect(() => {
    if (profile) {
      setValue('nickname', profile.nickname);
      setValue('bio', profile.bio || '');
      setValue('tags', profile.tags || []);
    }
  }, [profile, setValue]);

  const tags = watch('tags') || [];

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: async (data: ProfileFormData) => {
      const res = await usersService.updateProfile({
        nickname: data.nickname,
        bio: data.bio,
        tags: data.tags,
      });
      if (!isSuccess(res)) throw new Error(res.message);
      return res;
    },
    onSuccess: () => {
      toast({
        title: '保存成功',
        description: '个人资料已更新',
      });
      queryClient.invalidateQueries({ queryKey: ['my-profile'] });
      // Update local user state
      if (user) {
        setUser({
          ...user,
          nickname: watch('nickname'),
          bio: watch('bio'),
          tags: watch('tags'),
        });
      }
    },
    onError: (error: Error) => {
      toast({
        title: '保存失败',
        description: error.message,
        variant: 'destructive',
      });
    },
  });

  const handleAddTag = () => {
    const tag = tagInput.trim().replace(/^#/, '');
    if (tag && tags.length < 5 && !tags.includes(tag)) {
      setValue('tags', [...tags, tag], { shouldDirty: true });
      setTagInput('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setValue('tags', tags.filter((t) => t !== tagToRemove), { shouldDirty: true });
  };

  const handleTagKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddTag();
    }
  };

  const onSubmit = (data: ProfileFormData) => {
    updateMutation.mutate(data);
  };

  if (profileQuery.isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Loader2 className="h-10 w-10 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="max-w-xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => router.back()}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-2xl font-bold">编辑资料</h1>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Avatar */}
        <Card className="shadow-soft">
          <CardHeader>
            <CardTitle className="text-lg">头像</CardTitle>
          </CardHeader>
          <CardContent className="flex items-center gap-6">
            <div className="relative">
              <Avatar className="h-24 w-24">
                <AvatarImage src={profile?.avatar_url} alt={profile?.nickname} />
                <AvatarFallback className="bg-gradient-to-br from-primary/80 to-primary text-primary-foreground text-2xl">
                  {profile?.nickname?.slice(0, 2).toUpperCase() || 'U'}
                </AvatarFallback>
              </Avatar>
              <button
                type="button"
                className="absolute bottom-0 right-0 w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center shadow-md hover:bg-primary/90 transition-colors"
              >
                <Camera className="h-4 w-4" />
              </button>
            </div>
            <div className="text-sm text-muted-foreground">
              <p>点击更换头像</p>
              <p>支持 JPG、PNG 格式</p>
            </div>
          </CardContent>
        </Card>

        {/* Basic Info */}
        <Card className="shadow-soft">
          <CardHeader>
            <CardTitle className="text-lg">基本信息</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="nickname">昵称</Label>
              <Input
                id="nickname"
                placeholder="输入昵称"
                className="rounded-lg"
                {...register('nickname')}
              />
              {errors.nickname && (
                <p className="text-sm text-destructive">{errors.nickname.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="bio">简介</Label>
              <Input
                id="bio"
                placeholder="介绍一下自己"
                className="rounded-lg"
                {...register('bio')}
              />
              {errors.bio && (
                <p className="text-sm text-destructive">{errors.bio.message}</p>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Tags */}
        <Card className="shadow-soft">
          <CardHeader>
            <CardTitle className="text-lg">兴趣标签</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Input
                placeholder="添加标签（最多5个）"
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={handleTagKeyDown}
                className="rounded-lg"
                disabled={tags.length >= 5}
              />
              <Button
                type="button"
                variant="outline"
                onClick={handleAddTag}
                disabled={!tagInput.trim() || tags.length >= 5}
                className="shrink-0"
              >
                添加
              </Button>
            </div>
            {tags.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="inline-flex items-center gap-1 px-3 py-1 rounded-full bg-primary/10 text-primary text-sm"
                  >
                    #{tag}
                    <button
                      type="button"
                      onClick={() => handleRemoveTag(tag)}
                      className="hover:text-destructive transition-colors"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
            {errors.tags && (
              <p className="text-sm text-destructive">{errors.tags.message}</p>
            )}
          </CardContent>
        </Card>

        {/* Submit */}
        <Button
          type="submit"
          className="w-full rounded-lg"
          disabled={!isDirty || updateMutation.isPending}
        >
          {updateMutation.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              保存中...
            </>
          ) : (
            '保存修改'
          )}
        </Button>
      </form>
    </div>
  );
}
