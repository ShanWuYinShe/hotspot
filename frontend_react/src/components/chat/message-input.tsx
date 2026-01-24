'use client';

/**
 * 消息输入框组件
 */

import { useState, useRef, useCallback } from 'react';
import { Send, ImagePlus, Paperclip, MapPin, X } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import type { SendMessageType, MessageReplyTo } from '@/types/api';

interface MessageInputProps {
  onSend: (type: SendMessageType, content?: string, file?: File) => void;
  onTyping?: (isTyping: boolean) => void;
  replyTo?: MessageReplyTo | null;
  onCancelReply?: () => void;
  disabled?: boolean;
  placeholder?: string;
}

export function MessageInput({
  onSend,
  onTyping,
  replyTo,
  onCancelReply,
  disabled = false,
  placeholder = '输入消息...',
}: MessageInputProps) {
  const [content, setContent] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const typingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const imageInputRef = useRef<HTMLInputElement>(null);

  const handleTyping = useCallback(() => {
    if (!isTyping) {
      setIsTyping(true);
      onTyping?.(true);
    }

    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }

    typingTimeoutRef.current = setTimeout(() => {
      setIsTyping(false);
      onTyping?.(false);
    }, 3000);
  }, [isTyping, onTyping]);

  const handleSend = () => {
    if (!content.trim()) return;
    
    onSend('text', content.trim());
    setContent('');
    setIsTyping(false);
    onTyping?.(false);
    
    if (typingTimeoutRef.current) {
      clearTimeout(typingTimeoutRef.current);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      onSend('image', undefined, file);
      e.target.value = '';
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      const type = file.type.startsWith('video/') ? 'video' :
                   file.type.startsWith('audio/') ? 'audio' : 'file';
      onSend(type as SendMessageType, undefined, file);
      e.target.value = '';
    }
  };

  return (
    <div className="border-t bg-background p-4">
      {/* 回复引用 */}
      {replyTo && (
        <div className="flex items-center justify-between mb-2 p-2 bg-muted rounded-lg text-sm">
          <div className="truncate">
            <span className="text-muted-foreground">回复 </span>
            <span className="font-medium">@{replyTo.nickname}</span>
            <span className="text-muted-foreground">: {replyTo.content}</span>
          </div>
          <Button variant="ghost" size="icon" className="h-6 w-6" onClick={onCancelReply}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      )}

      <div className="flex items-end gap-2">
        {/* 附件按钮 */}
        <div className="flex gap-1">
          <Button
            variant="ghost"
            size="icon"
            className="h-9 w-9"
            onClick={() => imageInputRef.current?.click()}
            disabled={disabled}
          >
            <ImagePlus className="h-5 w-5" />
          </Button>
          <input
            ref={imageInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleImageSelect}
          />

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-9 w-9" disabled={disabled}>
                <Paperclip className="h-5 w-5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent>
              <DropdownMenuItem onClick={() => fileInputRef.current?.click()}>
                文件
              </DropdownMenuItem>
              <DropdownMenuItem>
                <MapPin className="h-4 w-4 mr-2" />
                位置
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <input
            ref={fileInputRef}
            type="file"
            className="hidden"
            onChange={handleFileSelect}
          />
        </div>

        {/* 输入框 */}
        <Textarea
          placeholder={placeholder}
          value={content}
          onChange={(e) => {
            setContent(e.target.value);
            handleTyping();
          }}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          rows={1}
          className="min-h-[40px] max-h-[120px] resize-none"
        />

        {/* 发送按钮 */}
        <Button
          size="icon"
          className="h-10 w-10"
          onClick={handleSend}
          disabled={disabled || !content.trim()}
        >
          <Send className="h-5 w-5" />
        </Button>
      </div>
    </div>
  );
}
