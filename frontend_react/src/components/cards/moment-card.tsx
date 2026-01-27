import { formatDistanceToNow } from 'date-fns';
import { Heart, MessageSquare, Share2, MapPin, MoreHorizontal } from 'lucide-react';
import { useState } from 'react';
import Link from 'next/link';
import { toast } from 'sonner';
import { useTranslations } from 'next-intl';

import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { FeedMomentItem } from '@/types/api';
import { MomentService } from '@/lib/api/services/moments';
import { cn } from '@/lib/utils';
import { useLoginModal } from '@/providers/login-modal-provider';
import { ShareDialog } from '@/components/modals/share-dialog';

interface MomentCardProps {
  moment: FeedMomentItem;
}

export function MomentCard({ moment }: MomentCardProps) {
  const [isLiked, setIsLiked] = useState(moment.is_liked);
  const [likesCount, setLikesCount] = useState(moment.likes_count);
  const [isShareOpen, setIsShareOpen] = useState(false);
  const t = useTranslations('moments');
  const tCommon = useTranslations('common');
  const { requireAuth } = useLoginModal();

  const handleLike = () => {
    requireAuth(async () => {
      const previousState = isLiked;
      const previousCount = likesCount;
      
      setIsLiked(!previousState);
      setLikesCount(previousState ? previousCount - 1 : previousCount + 1);

      try {
        const response = previousState 
          ? await MomentService.unlike(moment.moment_id)
          : await MomentService.like(moment.moment_id);

        if (response.code !== 0) {
          throw new Error(response.message);
        }
      } catch {
        setIsLiked(previousState);
        setLikesCount(previousCount);
        toast.error(tCommon('retry'));
      }
    });
  };

  return (
    <Card className="w-full max-w-2xl mb-4">
      <CardHeader className="flex flex-row items-center gap-4 pb-2">
        <Link href={`/users/${moment.user.user_id}`}>
          <Avatar>
            <AvatarImage src={moment.user.avatar_url} alt={moment.user.nickname} />
            <AvatarFallback>{moment.user.nickname.slice(0, 2).toUpperCase()}</AvatarFallback>
          </Avatar>
        </Link>
        <div className="flex flex-col">
          <Link href={`/users/${moment.user.user_id}`} className="font-semibold hover:underline">
            {moment.user.nickname}
          </Link>
          <span className="text-sm text-muted-foreground">
            {formatDistanceToNow(new Date(moment.created_at), { addSuffix: true })}
          </span>
        </div>
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="ml-auto h-8 w-8 p-0">
                    <MoreHorizontal className="h-4 w-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                <DropdownMenuItem>{t('menu.report')}</DropdownMenuItem>
                <DropdownMenuItem>{t('menu.notInterested')}</DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
      </CardHeader>
      
      <CardContent className="pb-2">
        <p className="whitespace-pre-wrap mb-4">{moment.description}</p>
        
        {moment.images && moment.images.length > 0 && (
          <div className={cn("grid gap-2", moment.images.length > 1 ? "grid-cols-2" : "grid-cols-1")}>
            {moment.images.map((image, index) => (
              <div key={index} className="relative aspect-square overflow-hidden rounded-md bg-muted">
                <div className="flex items-center justify-center w-full h-full text-muted-foreground text-xs">
                    {image.url}
                </div>
              </div>
            ))}
          </div>
        )}

        {moment.location && (
           <div className="flex items-center text-sm text-muted-foreground mt-4">
               <MapPin className="h-3 w-3 mr-1" />
               {moment.location.location_name}
               {moment.location.distance !== undefined && (
                   <span className="ml-1">
                     • {moment.location.distance < 1000 ? `${moment.location.distance}m` : `${(moment.location.distance / 1000).toFixed(1)}km`} {t('distanceAway')}
                   </span>
               )}
           </div>
        )}
      </CardContent>

      <CardFooter className="flex justify-between pt-2">
        <Button variant="ghost" size="sm" onClick={handleLike} className={cn(isLiked && "text-red-500")}>
          <Heart className={cn("mr-2 h-4 w-4", isLiked && "fill-current")} />
          {likesCount}
        </Button>
        <Link href={`/moments/${moment.moment_id}`}>
            <Button variant="ghost" size="sm">
            <MessageSquare className="mr-2 h-4 w-4" />
            {moment.comments_count}
            </Button>
        </Link>
        <Button variant="ghost" size="sm" onClick={() => setIsShareOpen(true)}>
          <Share2 className="mr-2 h-4 w-4" />
          {t('actions.share')}
        </Button>
      </CardFooter>
      <ShareDialog 
        open={isShareOpen} 
        onOpenChange={setIsShareOpen} 
        momentId={moment.moment_id} 
      />
    </Card>
  );
}
