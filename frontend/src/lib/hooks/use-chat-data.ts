import { useState, useEffect, useCallback, useRef } from 'react';
import { privateChatsService, privateMessagesService } from '@/lib/api/services';
import { isSuccess } from '@/lib/api/client';
import type { PrivateChatItem, PrivateMessageItem } from '@/lib/types/api';

interface ChatData {
  chat: PrivateChatItem | null;
  messages: PrivateMessageItem[];
  loading: boolean;
  sending: boolean;
  error: string | null;
  hasMore: boolean;
  sendMessage: (content: string) => Promise<boolean>;
  loadMore: () => void;
  refresh: () => void;
}

export function useChatData(chatId: string | undefined): ChatData {
  const [chat, setChat] = useState<PrivateChatItem | null>(null);
  const [messages, setMessages] = useState<PrivateMessageItem[]>([]);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const initialised = useRef(false);

  const loadChat = useCallback(async () => {
    if (!chatId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await privateChatsService.openChat({ target_user_id: chatId });
      if (isSuccess(res) && res.data) {
        setChat(res.data.chat);
        setMessages(res.data.recent_messages.items);
        setNextCursor(res.data.recent_messages.next_cursor);
      } else {
        setError('加载失败，请重试');
      }
    } catch {
      setError('网络异常');
    } finally {
      setLoading(false);
    }
  }, [chatId]);

  useEffect(() => {
    if (!chatId) return;
    if (initialised.current) return;
    initialised.current = true;
    loadChat();
  }, [chatId, loadChat]);

  const loadMore = useCallback(async () => {
    if (!nextCursor || !chatId) return;
    try {
      const res = await privateMessagesService.getHistory({
        target_user_id: chatId,
        pagination: { cursor: nextCursor, limit: 20 }
      });
      if (isSuccess(res) && res.data) {
        setMessages(prev => [...res.data!.items, ...prev]);
        setNextCursor(res.data.next_cursor);
      }
    } catch {}
  }, [chatId, nextCursor]);

  const sendMessage = useCallback(async (content: string): Promise<boolean> => {
    if (!chatId || !content.trim()) return false;
    setSending(true);
    try {
      const res = await privateMessagesService.send({
        target_user_id: chatId,
        type: 'text',
        content: content.trim()
      });
      if (isSuccess(res) && res.data) {
        const msg: PrivateMessageItem = {
          message_id: res.data.message_id,
          type: 'text',
          content: content.trim(),
          sender: { user_id: 'me', nickname: '' },
          sent_at: res.data.sent_at,
          is_read: false,
        };
        setMessages(prev => [...prev, msg]);
        return true;
      }
      return false;
    } catch {
      return false;
    } finally {
      setSending(false);
    }
  }, [chatId]);

  return {
    chat, messages, loading, sending, error,
    hasMore: !!nextCursor,
    sendMessage, loadMore,
    refresh: loadChat,
  };
}
