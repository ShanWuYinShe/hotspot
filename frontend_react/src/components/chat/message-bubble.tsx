'use client';

/**
 * 消息气泡组件
 */

import Image from 'next/image';
import Link from 'next/link';
import { CheckCheck, MoreHorizontal, MapPin, File as FileIcon } from '@/components/icons';
import { UserAvatar } from '@/components/common/user-avatar';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { formatRelativeTime, cn } from '@/lib/utils';
import type { PrivateMessageItem, GroupMessageItem } from '@/types/api';

type MessageItem = PrivateMessageItem | GroupMessageItem;

interface MessageBubbleProps {
  message: MessageItem;
  isOwn: boolean;
  showAvatar?: boolean;
  onRecall?: () => void;
  onDelete?: () => void;
  onReply?: () => void;
}

export function MessageBubble({
  message,
  isOwn,
  showAvatar = true,
  onRecall,
  onDelete,
  onReply,
}: MessageBubbleProps) {
  if (message.is_recalled) {
    return (
      <div className={cn('flex items-center gap-2', isOwn && 'flex-row-reverse')}>
        {showAvatar && !isOwn && (
          <Link href={`/users/${message.sender.user_id}`}>
            <UserAvatar user={message.sender} size="sm" />
          </Link>
        )}
        <div className="text-sm text-muted-foreground italic">
          消息已撤回
        </div>
      </div>
    );
  }

  return (
    <div className={cn('flex gap-2 group', isOwn && 'flex-row-reverse')}>
      {showAvatar && !isOwn && (
        <Link href={`/users/${message.sender.user_id}`}>
          <UserAvatar user={message.sender} size="sm" />
        </Link>
      )}

      <div className={cn('flex flex-col gap-1 max-w-[70%]', isOwn && 'items-end')}>
        {/* 发送者名称（群聊） */}
        {!isOwn && showAvatar && (
          <span className="text-xs text-muted-foreground ml-2">
            {message.sender.nickname}
          </span>
        )}

        {/* 回复引用 */}
        {message.reply_to && (
          <div className="text-xs text-muted-foreground bg-muted/50 px-2 py-1 rounded ml-2 max-w-xs truncate">
            回复 @{message.reply_to.nickname}: {message.reply_to.content}
          </div>
        )}

        {/* 消息内容 */}
        <div className="flex items-end gap-2">
          <div
            className={cn(
              'rounded-2xl px-4 py-2',
              isOwn
                ? 'bg-primary text-primary-foreground rounded-br-sm'
                : 'bg-muted rounded-bl-sm'
            )}
          >
            <MessageContent message={message} />
          </div>

          {/* 操作菜单 */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6 opacity-0 group-hover:opacity-100 transition-opacity"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align={isOwn ? 'end' : 'start'}>
              <DropdownMenuItem onClick={onReply}>回复</DropdownMenuItem>
              {isOwn && (
                <>
                  <DropdownMenuItem onClick={onRecall}>撤回</DropdownMenuItem>
                  <DropdownMenuItem onClick={onDelete} className="text-destructive">
                    删除
                  </DropdownMenuItem>
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>

        {/* 时间和状态 */}
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <span>{formatRelativeTime(message.sent_at)}</span>
          {message.edited_at && <span>(已编辑)</span>}
          {isOwn && (
            <span className="text-primary">
              <CheckCheck className="h-3 w-3" />
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

interface MessageContentProps {
  message: MessageItem;
}

function MessageContent({ message }: MessageContentProps) {
  switch (message.type) {
    case 'text':
      return (
        <p className="text-sm whitespace-pre-wrap break-words">
          {message.content}
        </p>
      );

    case 'image':
      return (
        <div className="relative w-48 h-48 rounded-lg overflow-hidden">
          <Image
            src={message.media?.url ?? ''}
            alt=""
            fill
            className="object-cover"
          />
        </div>
      );

    case 'video':
      return (
        <video
          src={message.media?.url}
          controls
          className="w-48 rounded-lg"
          poster={message.media?.thumbnail_url}
        />
      );

    case 'audio':
      return (
        <audio src={message.media?.url} controls className="w-48" />
      );

    case 'file':
      return (
        <a
          href={message.media?.url}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-2 text-sm underline"
        >
          <FileIcon className="h-4 w-4" />
          <span>{message.media?.file_name ?? '文件'}</span>
        </a>
      );

    case 'location':
      return (
        <div className="flex items-center gap-2 text-sm">
          <MapPin className="h-4 w-4" />
          <span>{message.location_address ?? '位置'}</span>
        </div>
      );

    case 'system':
      return (
        <p className="text-sm text-muted-foreground text-center">
          {message.content}
        </p>
      );

    default:
      return <p className="text-sm">[不支持的消息类型]</p>;
  }
}
