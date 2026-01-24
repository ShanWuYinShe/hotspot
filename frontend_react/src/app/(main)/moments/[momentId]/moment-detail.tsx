/**
 * 瞬间详情 (Server Component)
 * 展示瞬间内容，交互由客户端组件处理
 */

import Link from 'next/link';
import Image from 'next/image';
import { MapPin } from '@/components/icons';
import { Card, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/common/user-avatar';
import { formatRelativeTime, formatDistance, cn } from '@/lib/utils';
import { MomentDetailActions, MomentDetailMenu } from '@/components/moments';
import type { MomentItem } from '@/types/api';

interface MomentDetailProps {
  moment: MomentItem;
}

export function MomentDetail({ moment }: MomentDetailProps) {
  const isLiked = moment.is_liked ?? false;
  const isFavorited = moment.is_favorited ?? false;
  const shareTitle = `${moment.user.nickname}的瞬间`;
  const shareText = moment.description.slice(0, 100);

  return (
    <Card>
      <CardContent className="p-6">
        {/* 用户信息 */}
        <div className="flex items-center justify-between mb-4">
          <Link 
            href={`/users/${moment.user.user_id}`}
            className="flex items-center gap-3"
          >
            <UserAvatar user={moment.user} size="lg" />
            <div>
              <p className="font-semibold">{moment.user.nickname}</p>
              <p className="text-sm text-muted-foreground">
                {formatRelativeTime(moment.created_at)}
                {moment.location && (
                  <>
                    {' · '}
                    <MapPin className="h-3 w-3 inline" />
                    {' '}
                    {formatDistance(moment.location.distance)}
                  </>
                )}
              </p>
            </div>
          </Link>

          <MomentDetailMenu momentId={moment.moment_id} ownerId={moment.user.user_id} />
        </div>

        {/* 内容 */}
        <div className="space-y-4">
          <p className="text-base whitespace-pre-wrap">{moment.description}</p>

          {/* 图片网格 */}
          {moment.images && moment.images.length > 0 && (
            <div
              className={cn(
                'grid gap-2',
                moment.images.length === 1 && 'grid-cols-1',
                moment.images.length === 2 && 'grid-cols-2',
                moment.images.length >= 3 && 'grid-cols-3'
              )}
            >
              {moment.images.map((image, index) => (
                <div
                  key={index}
                  className="relative aspect-square rounded-lg overflow-hidden bg-muted cursor-pointer"
                >
                  <Image
                    src={image.url}
                    alt=""
                    fill
                    className="object-cover hover:scale-105 transition-transform"
                    sizes="(max-width: 768px) 50vw, 300px"
                  />
                </div>
              ))}
            </div>
          )}

          {/* 标签 */}
          {moment.tags.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {moment.tags.map((tag) => (
                <Badge key={tag} variant="secondary">
                  #{tag}
                </Badge>
              ))}
            </div>
          )}
        </div>
      </CardContent>

      {/* 操作栏 */}
      <CardFooter className="px-6 py-4 border-t">
        <MomentDetailActions
          momentId={moment.moment_id}
          likesCount={moment.likes_count}
          commentsCount={moment.comments_count}
          isLiked={isLiked}
          isFavorited={isFavorited}
          shareTitle={shareTitle}
          shareText={shareText}
        />
      </CardFooter>
    </Card>
  );
}
