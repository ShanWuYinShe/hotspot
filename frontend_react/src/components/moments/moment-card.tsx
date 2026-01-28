'use client';

import * as React from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { Heart, MessageCircle, Bookmark, MapPin, MoreHorizontal } from 'lucide-react';
import { cn, formatRelativeTime, formatDistance } from '@/lib/utils';
import { Card, CardContent, CardFooter } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import type { FeedMomentItem, MomentSearchItem, MomentCard as MomentCardType } from '@/types/api';

type MomentData = FeedMomentItem | MomentSearchItem | MomentCardType;

interface MomentCardProps {
  moment: MomentData;
  onLike?: (momentId: string) => void;
  onBookmark?: (momentId: string) => void;
  className?: string;
}

function isFeedMoment(moment: MomentData): moment is FeedMomentItem {
  return 'is_liked' in moment;
}

function hasRelativeLocation(moment: MomentData): moment is MomentSearchItem | FeedMomentItem {
  return 'distance' in (moment.location ?? {});
}

export function MomentCard({
  moment,
  onLike,
  onBookmark,
  className,
}: MomentCardProps) {
  const isLiked = isFeedMoment(moment) ? moment.is_liked : false;
  const isFavorited = isFeedMoment(moment) ? moment.is_favorited : false;

  return (
    <Card className={cn('overflow-hidden', className)}>
      {/* 用户信息 */}
      <div className="flex items-center gap-3 p-4 pb-2">
        <Link href={`/users/${moment.user.user_id}`}>
          <Avatar className="h-10 w-10">
            <AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
            <AvatarFallback>
              {moment.user.nickname.slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>
        </Link>
        <div className="flex-1 min-w-0">
          <Link
            href={`/users/${moment.user.user_id}`}
            className="font-medium hover:underline"
          >
            {moment.user.nickname}
          </Link>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>{formatRelativeTime(moment.created_at)}</span>
            {hasRelativeLocation(moment) && moment.location && (
              <>
                <span>·</span>
                <span className="flex items-center gap-1">
                  <MapPin className="h-3 w-3" />
                  {formatDistance(moment.location.distance)}
                </span>
              </>
            )}
          </div>
        </div>
        <Button variant="ghost" size="icon" className="h-8 w-8">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </div>

      {/* 内容 */}
      <CardContent className="px-4 py-2">
        <Link href={`/moments/${moment.moment_id}`}>
          <p className="whitespace-pre-wrap break-words">{moment.description}</p>
        </Link>

        {/* 标签 */}
        {moment.tags && moment.tags.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {moment.tags.map((tag) => (
              <Link
                key={tag}
                href={`/explore?tag=${encodeURIComponent(tag)}`}
                className="text-sm text-primary hover:underline"
              >
                #{tag}
              </Link>
            ))}
          </div>
        )}

        {/* 图片 */}
        {moment.images && moment.images.length > 0 && (
          <div
            className={cn(
              'mt-3 grid gap-2',
              moment.images.length === 1 && 'grid-cols-1',
              moment.images.length === 2 && 'grid-cols-2',
              moment.images.length >= 3 && 'grid-cols-3'
            )}
          >
            {moment.images.slice(0, 9).map((image, index) => (
              <Link
                key={index}
                href={`/moments/${moment.moment_id}`}
                className="relative aspect-square overflow-hidden rounded-lg bg-muted"
              >
                <Image
                  src={image.url}
                  alt={`图片 ${index + 1}`}
                  fill
                  className="object-cover"
                  sizes="(max-width: 768px) 33vw, 200px"
                />
                {index === 8 && moment.images && moment.images.length > 9 && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/50 text-white text-lg font-medium">
                    +{moment.images.length - 9}
                  </div>
                )}
              </Link>
            ))}
          </div>
        )}
      </CardContent>

      {/* 互动按钮 */}
      <CardFooter className="px-4 py-2 border-t">
        <div className="flex w-full items-center justify-between">
          <Button
            variant="ghost"
            size="sm"
            className={cn('gap-1', isLiked && 'text-red-500')}
            onClick={() => onLike?.(moment.moment_id)}
          >
            <Heart className={cn('h-4 w-4', isLiked && 'fill-current')} />
            <span>{moment.likes_count > 0 ? moment.likes_count : '赞'}</span>
          </Button>
          <Button variant="ghost" size="sm" className="gap-1" asChild>
            <Link href={`/moments/${moment.moment_id}#comments`}>
              <MessageCircle className="h-4 w-4" />
              <span>{moment.comments_count > 0 ? moment.comments_count : '评论'}</span>
            </Link>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className={cn('gap-1', isFavorited && 'text-yellow-500')}
            onClick={() => onBookmark?.(moment.moment_id)}
          >
            <Bookmark className={cn('h-4 w-4', isFavorited && 'fill-current')} />
            <span>收藏</span>
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
}
