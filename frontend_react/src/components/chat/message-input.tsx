'use client';

import { useEffect, useRef, useState } from 'react';
import { Send } from 'lucide-react';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';

interface MessageInputProps {
  onSend: (content: string) => void;
  onTyping?: (isTyping: boolean) => void;
  disabled?: boolean;
}

export function MessageInput({ onSend, onTyping, disabled }: MessageInputProps) {
  const t = useTranslations('chat');
  const [value, setValue] = useState('');
  const typingTimer = useRef<number | null>(null);

  useEffect(() => {
    return () => {
      if (typingTimer.current) {
        window.clearTimeout(typingTimer.current);
      }
    };
  }, []);

  const handleTyping = () => {
    if (onTyping) {
      onTyping(true);
    }
    if (typingTimer.current) {
      window.clearTimeout(typingTimer.current);
    }
    typingTimer.current = window.setTimeout(() => {
      onTyping?.(false);
    }, 2000);
  };

  const handleSend = () => {
    const trimmed = value.trim();
    if (!trimmed) {
      return;
    }
    onSend(trimmed);
    setValue('');
    onTyping?.(false);
  };

  return (
    <div className="flex items-end gap-2 border-t pt-4">
      <Textarea
        value={value}
        placeholder={t('inputPlaceholder')}
        onChange={(event) => {
          setValue(event.target.value);
          handleTyping();
        }}
        className="min-h-[80px] resize-none"
        disabled={disabled}
      />
      <Button onClick={handleSend} disabled={disabled || !value.trim()}>
        <Send className="h-4 w-4 mr-2" />
        {t('send')}
      </Button>
    </div>
  );
}
