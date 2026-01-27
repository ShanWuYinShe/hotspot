'use client';

import { MapPin } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { GroupMessageItem, PrivateMessageItem } from '@/types/api';
import { cn } from '@/lib/utils';

type ChatMessage = PrivateMessageItem | GroupMessageItem;

interface MessageBubbleProps {
  message: ChatMessage;
  isMine: boolean;
}

export function MessageBubble({ message, isMine }: MessageBubbleProps) {
  const t = useTranslations('chat');

  const renderContent = () => {
    if (message.is_recalled) {
      return <span className="text-xs text-muted-foreground">{t('messageRecalled')}</span>;
    }

    if (message.type === 'text') {
      return <span>{message.content}</span>;
    }

    if (message.type === 'location' && message.location?.location) {
      return (
        <div className="flex items-center gap-2">
          <MapPin className="h-4 w-4" />
          <span>{message.location.location.location_name}</span>
        </div>
      );
    }

    if (message.type === 'image') {
      return <span>{t('attachmentImage')}</span>;
    }

    if (message.type === 'video') {
      return <span>{t('attachmentVideo')}</span>;
    }

    if (message.type === 'audio') {
      return <span>{t('attachmentAudio')}</span>;
    }

    if (message.type === 'file') {
      return <span>{t('attachmentFile')}</span>;
    }

    return <span>{message.content}</span>;
  };

  return (
    <div className={cn('flex', isMine ? 'justify-end' : 'justify-start')}>
      <div
        className={cn(
          'max-w-[70%] rounded-2xl px-4 py-2 text-sm shadow-sm',
          isMine ? 'bg-primary text-primary-foreground' : 'bg-muted',
        )}
      >
        {renderContent()}
      </div>
    </div>
  );
}
