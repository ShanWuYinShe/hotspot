'use client';

/**
 * 创建瞬间表单组件 (Client Component)
 * 处理表单状态、文件上传、提交逻辑
 */

import { useEffect, useState, useTransition } from 'react';
import { useRouter } from 'next/navigation';
import Image from 'next/image';
import { ImagePlus, X, MapPin, Hash, Send, ArrowLeft } from '@/components/icons';
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
import { momentsService } from '@/lib/api/services/moments';
import { uploadService } from '@/lib/api/services/upload';
import { isSuccess } from '@/lib/api/client';
import type { PrivacyLevel } from '@/types/api';

interface CreateMomentFormProps {
  translations: {
    create: string;
  };
}

export function CreateMomentForm({ translations }: CreateMomentFormProps) {
  const router = useRouter();
  const { toast } = useToast();
  const { currentLocation, refreshLocation } = useLocation();
  const { requireAuth } = useRequireAuth();
  const [isPending, startTransition] = useTransition();

  const [description, setDescription] = useState('');
  const [images, setImages] = useState<File[]>([]);
  const [imageUrls, setImageUrls] = useState<string[]>([]);
  const [tags, setTags] = useState<string[]>([]);
  const [tagInput, setTagInput] = useState('');
  const [privacy, setPrivacy] = useState<PrivacyLevel>('public');
  const [isUploading, setIsUploading] = useState(false);

  useEffect(() => {
    if (!currentLocation) {
      refreshLocation(false);
    }
  }, [currentLocation, refreshLocation]);

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    if (images.length + files.length > 9) {
      toast.error('最多上传9张图片');
      return;
    }
    
    const newUrls = files.map((file) => URL.createObjectURL(file));
    setImages((prev) => [...prev, ...files]);
    setImageUrls((prev) => [...prev, ...newUrls]);
  };

  const handleRemoveImage = (index: number) => {
    const url = imageUrls[index];
    if (url) URL.revokeObjectURL(url);
    setImages((prev) => prev.filter((_, i) => i !== index));
    setImageUrls((prev) => prev.filter((_, i) => i !== index));
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
    if (!description.trim()) {
      toast.error('请输入内容');
      return;
    }

    if (!requireAuth('发布瞬间')) {
      return;
    }

    if (!currentLocation) {
      toast.error('无法获取位置信息');
      return;
    }

    startTransition(async () => {
      try {
        setIsUploading(true);

        const fileIds: string[] = [];
        for (const file of images) {
          const uploadResult = await uploadService.uploadFile(file, 'moment_image');
          fileIds.push(uploadResult.file_id);
        }

        const result = await momentsService.create({
          description: description.trim(),
          location: {
            latitude: currentLocation.latitude,
            longitude: currentLocation.longitude,
            location_name: currentLocation.location_name || '未知位置',
          },
          image_file_ids: fileIds.length > 0 ? fileIds : undefined,
          tags: tags.length > 0 ? tags : undefined,
          privacy_level: privacy,
        });

        if (isSuccess(result)) {
          toast.success('发布成功');
          router.push('/');
        } else {
          toast.error('发布失败', result.message);
        }
      } catch (error) {
        toast.error('发布失败', error instanceof Error ? error.message : '未知错误');
      } finally {
        setIsUploading(false);
      }
    });
  };

  return (
    <div className="max-w-2xl mx-auto">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="icon" onClick={() => router.back()}>
              <ArrowLeft className="h-5 w-5" />
            </Button>
            <CardTitle>{translations.create}</CardTitle>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* 内容输入 */}
          <div className="space-y-2">
            <Label>内容</Label>
            <Textarea
              placeholder="分享你的生活..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={4}
              className="resize-none"
              maxLength={500}
            />
            <p className="text-xs text-muted-foreground text-right">
              {description.length}/500
            </p>
          </div>

          {/* 图片上传 */}
          <div className="space-y-2">
            <Label>图片 ({images.length}/9)</Label>
            <div className="grid grid-cols-3 gap-2">
              {imageUrls.map((url, index) => (
                <div key={index} className="relative aspect-square rounded-lg overflow-hidden bg-muted">
                  <Image
                    src={url}
                    alt=""
                    fill
                    className="object-cover"
                  />
                  <Button
                    variant="destructive"
                    size="icon"
                    className="absolute top-1 right-1 h-6 w-6"
                    onClick={() => handleRemoveImage(index)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
              {images.length < 9 && (
                <label className="aspect-square rounded-lg border-2 border-dashed border-muted-foreground/25 flex items-center justify-center cursor-pointer hover:border-muted-foreground/50 transition-colors">
                  <ImagePlus className="h-8 w-8 text-muted-foreground" />
                  <input
                    type="file"
                    accept="image/*"
                    multiple
                    className="hidden"
                    onChange={handleImageSelect}
                  />
                </label>
              )}
            </div>
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

          {/* 隐私设置 */}
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

          {/* 提交按钮 */}
          <Button
            className="w-full"
            size="lg"
            onClick={handleSubmit}
            disabled={isPending || isUploading || !description.trim()}
          >
            <Send className="h-4 w-4 mr-2" />
            {isPending || isUploading ? '发布中...' : '发布'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
