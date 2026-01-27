import { formatDistanceToNow } from 'date-fns';
import { Heart, MessageSquare, MoreHorizontal, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslations } from 'next-intl';

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { CommentListItem } from '@/types/api';
import { CommentService } from '@/lib/api/services/comments';
import { toast } from 'sonner';

interface CommentItemProps {
  comment: CommentListItem;
  onReply?: (commentId: string, userNickname: string) => void;
  onDelete?: () => void;
  canDelete?: boolean;
}

export function CommentItem({ comment, onReply, onDelete, canDelete = false }: CommentItemProps) {
  const [isLiked, setIsLiked] = useState(false);
  const [likesCount, setLikesCount] = useState(comment.likes_count);
  const t = useTranslations('comments');
  const tCommon = useTranslations('common');

  const handleLike = async () => {
    const previousState = isLiked;
    const previousCount = likesCount;
    
    setIsLiked(!previousState);
    setLikesCount(previousState ? previousCount - 1 : previousCount + 1);

    try {
      const response = previousState 
        ? await CommentService.unlike(comment.comment_id)
        : await CommentService.like(comment.comment_id);

      if (response.code !== 0) {
        throw new Error(response.message);
      }
     } catch {
      setIsLiked(previousState);
      setLikesCount(previousCount);
      toast.error(tCommon('retry'));
    }
  };

  const handleDelete = async () => {
    try {
      const response = await CommentService.delete(comment.comment_id);
      if (response.code === 0) {
        toast.success(t('deleted'));
        onDelete?.();
      } else {
        toast.error(response.message || tCommon('retry'));
      }
    } catch {
      toast.error(tCommon('retry'));
    }

  };

  return (
    <div className="flex gap-3 py-4">
      <Avatar className="h-8 w-8">
        <AvatarImage src={comment.user.avatar_url} alt={comment.user.nickname} />
        <AvatarFallback>{comment.user.nickname.slice(0, 2).toUpperCase()}</AvatarFallback>
      </Avatar>
      
      <div className="flex-1 space-y-2">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-sm">{comment.user.nickname}</span>
            <span className="text-xs text-muted-foreground">
              {formatDistanceToNow(new Date(comment.created_at), { addSuffix: true })}
            </span>
          </div>
          
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-6 w-6 p-0">
                <MoreHorizontal className="h-3 w-3" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={() => onReply?.(comment.comment_id, comment.user.nickname)}>
                {t('reply')}
              </DropdownMenuItem>
              {canDelete && (
                <DropdownMenuItem 
                  onClick={handleDelete}
                  className="text-red-600"
                >
                  <Trash2 className="h-3 w-3 mr-2" />
                  {tCommon('delete')}
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
        
        <p className="text-sm break-words">{comment.content}</p>
        
        <div className="flex items-center gap-4">
          <Button 
            variant="ghost" 
            size="sm" 
            onClick={handleLike}
            className={`h-6 px-2 text-xs ${isLiked ? 'text-red-500' : ''}`}
          >
            <Heart className={`mr-1 h-3 w-3 ${isLiked ? 'fill-current' : ''}`} />
            {likesCount}
          </Button>
          
          {comment.replies_count > 0 && (
            <Button 
              variant="ghost" 
              size="sm"
              className="h-6 px-2 text-xs text-muted-foreground"
            >
              <MessageSquare className="mr-1 h-3 w-3" />
              {comment.replies_count} {t('replies')}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
