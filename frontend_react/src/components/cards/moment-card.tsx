import Link from 'next/link';
import Image from 'next/image';
import { MapPin } from '@/components/icons';
import { Card, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { UserAvatar } from '@/components/common/user-avatar';
import { formatRelativeTime, formatDistance, cn } from '@/lib/utils';
import { MomentActions } from './moment-actions';
import { MomentMenu } from './moment-menu';
import type { MomentItem } from '@/types/api';

interface MomentCardProps {
  moment: MomentItem;
  isOwner?: boolean;
  onLike?: (momentId: string) => void;
  onUnlike?: (momentId: string) => void;
  onFavorite?: (momentId: string) => void;
  onUnfavorite?: (momentId: string) => void;
  onReport?: (momentId: string) => void;
  onDelete?: (momentId: string) => void;
}

export function MomentCard({
  moment,
  isOwner = false,
  onLike,
  onUnlike,
  onFavorite,
  onUnfavorite,
  onReport,
  onDelete,
}: MomentCardProps) {
  return (
    <Card className="group overflow-hidden border-2 transition-all duration-300 hover:shadow-xl hover:shadow-primary/10 hover:border-primary/30">
      <CardContent className="p-5">
        <div className="flex items-center justify-between mb-4">
          <Link
            href={`/users/${moment.user.user_id}`}
            className="flex items-center gap-3 transition-all hover:scale-[1.02]"
          >
            <div className="relative">
              <div className="absolute inset-0 bg-gradient-to-br from-primary/20 to-primary/0 rounded-full blur-sm" />
              <UserAvatar user={moment.user} size="md" />
            </div>
            <div>
              <p className="font-semibold text-base">{moment.user.nickname}</p>
              <p className="text-xs text-muted-foreground flex items-center gap-1">
                {formatRelativeTime(moment.created_at)}
                {moment.location && (
                  <>
                    <span className="mx-1">·</span>
                    <MapPin className="h-3 w-3 text-primary" />
                    <span className="text-primary">{formatDistance(moment.location.distance)}</span>
                  </>
                )}
              </p>
            </div>
          </Link>
          <MomentMenu
            momentId={moment.moment_id}
            isOwner={isOwner}
            onReport={onReport}
            onDelete={onDelete}
          />
        </div>

        <Link
          href={`/moments/${moment.moment_id}`}
          className="block transition-transform group-hover:scale-[1.01]"
        >
          <p className="text-sm whitespace-pre-wrap mb-4 leading-relaxed text-foreground/90">{moment.description}</p>

          {moment.images && moment.images.length > 0 && (
            <div
              className={cn(
                'grid gap-3 mb-4 overflow-hidden rounded-2xl',
                moment.images.length === 1 && 'grid-cols-1',
                moment.images.length === 2 && 'grid-cols-2',
                moment.images.length >= 3 && 'grid-cols-3'
              )}
            >
              {moment.images.slice(0, 9).map((image, index) => (
                <div
                  key={index}
                  className="relative aspect-square overflow-hidden bg-muted transition-all duration-300 hover:scale-105 cursor-pointer group/img"
                >
                  <div className="absolute inset-0 bg-gradient-to-t from-black/20 to-transparent opacity-0 group-hover/img:opacity-100 transition-opacity duration-300" />
                  <Image
                    src={image.url}
                    alt=""
                    fill
                    className="object-cover"
                    sizes="(max-width: 768px) 33vw, 200px"
                  />
                </div>
              ))}
            </div>
          )}

          {moment.tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-3">
              {moment.tags.map((tag) => (
                <Badge
                  key={tag}
                  variant="secondary"
                  className="text-xs font-medium transition-all hover:bg-primary/20 hover:text-primary hover:scale-105 cursor-pointer border-0"
                >
                  #{tag}
                </Badge>
              ))}
            </div>
          )}
        </Link>
      </CardContent>

      <CardFooter className="px-5 py-4 border-t-2 bg-muted/50">
        <MomentActions
          momentId={moment.moment_id}
          likesCount={moment.likes_count}
          commentsCount={moment.comments_count}
          isLiked={moment.is_liked ?? false}
          isFavorited={moment.is_favorited ?? false}
          onLike={onLike}
          onUnlike={onUnlike}
          onFavorite={onFavorite}
          onUnfavorite={onUnfavorite}
        />
      </CardFooter>
    </Card>
  );
}
