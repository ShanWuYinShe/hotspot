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
    <Card className="group transition-all duration-300 hover:shadow-lg hover:shadow-primary/10">
      <CardContent className="p-4">
        <div className="flex items-center justify-between mb-3">
          <Link
            href={`/users/${moment.user.user_id}`}
            className="flex items-center gap-3 transition-colors hover:text-primary"
          >
            <UserAvatar user={moment.user} size="md" />
            <div>
              <p className="font-medium text-sm">{moment.user.nickname}</p>
              <p className="text-xs text-muted-foreground">
                {formatRelativeTime(moment.created_at)}
                {moment.location && (
                  <>
                    {' · '}
                    <MapPin className="h-3 w-3 inline" /> {formatDistance(moment.location.distance)}
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
          <p className="text-sm whitespace-pre-wrap mb-3 leading-relaxed">{moment.description}</p>

          {moment.images && moment.images.length > 0 && (
            <div
              className={cn(
                'grid gap-2 mb-3 overflow-hidden rounded-lg',
                moment.images.length === 1 && 'grid-cols-1',
                moment.images.length === 2 && 'grid-cols-2',
                moment.images.length >= 3 && 'grid-cols-3'
              )}
            >
              {moment.images.slice(0, 9).map((image, index) => (
                <div
                  key={index}
                  className="relative aspect-square overflow-hidden bg-muted transition-transform duration-300 hover:scale-105"
                >
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
            <div className="flex flex-wrap gap-1 mb-3">
              {moment.tags.map((tag) => (
                <Badge
                  key={tag}
                  variant="secondary"
                  className="text-xs transition-colors hover:bg-primary/20 hover:text-primary"
                >
                  #{tag}
                </Badge>
              ))}
            </div>
          )}
        </Link>
      </CardContent>

      <CardFooter className="px-4 py-3 border-t bg-muted/30">
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
