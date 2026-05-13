/**
 * 消息缓存服务
 * 封装对 chats 和 messages 表的操作
 */
import { RdbManager } from './RdbManager';
import { relationalStore, ValuesBucket } from '@kit.ArkData';
import { hilog } from '@kit.PerformanceAnalysisKit';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG = 'MessageCacheService';

export interface ChatCacheItem {
  id: string;
  type: 'private' | 'group';
  name: string;
  avatarUrl?: string;
  lastMessage?: string;
  lastTime: string;
  unreadCount: number;
  data: string; // JSON string of original data
}

export interface MessageCacheItem {
  id: string;
  chatId: string;
  type: string;
  senderId: string;
  content: string;
  sentAt: string;
  data: string; // JSON string of original data
}

export class MessageCacheService {
  private static instance: MessageCacheService;
  private rdbManager = RdbManager.getInstance();

  private constructor() {}

  public static getInstance(): MessageCacheService {
    if (!MessageCacheService.instance) {
      MessageCacheService.instance = new MessageCacheService();
    }
    return MessageCacheService.instance;
  }

  /**
   * 缓存聊天列表
   */
  public async cacheChatList(items: ChatCacheItem[]): Promise<void> {
    const store = this.rdbManager.getStore();
    if (!store) return;

    try {
      store.beginTransaction();
      
      // 简单策略：先清空旧数据，再插入新数据
      await store.executeSql('DELETE FROM chats');

      for (const item of items) {
        const valueBucket: ValuesBucket = {
          id: item.id,
          type: item.type,
          name: item.name,
          avatar_url: item.avatarUrl || '',
          last_message: item.lastMessage || '',
          last_time: item.lastTime,
          unread_count: item.unreadCount,
          json_data: item.data
        };
        await store.insert('chats', valueBucket);
      }
      
      store.commit();
      hilog.info(DOMAIN_NUMBER, TAG, `Cached ${items.length} chats`);
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Cache chat list failed: ${JSON.stringify(err)}`);
      try {
        store.rollBack();
      } catch (e) {
        hilog.error(DOMAIN_NUMBER, TAG, `Rollback failed: ${JSON.stringify(e)}`);
      }
    }
  }

  /**
   * 获取缓存的聊天列表
   */
  public async getCachedChatList(): Promise<ChatCacheItem[]> {
    const store = this.rdbManager.getStore();
    if (!store) return [];

    try {
      const predicates = new relationalStore.RdbPredicates('chats');
      predicates.orderByDesc('last_time'); // 按时间倒序
      
      const resultSet = await store.query(predicates);
      const items: ChatCacheItem[] = [];

      while (resultSet.goToNextRow()) {
        items.push({
          id: resultSet.getString(resultSet.getColumnIndex('id')),
          type: resultSet.getString(resultSet.getColumnIndex('type')) as 'private' | 'group',
          name: resultSet.getString(resultSet.getColumnIndex('name')),
          avatarUrl: resultSet.getString(resultSet.getColumnIndex('avatar_url')),
          lastMessage: resultSet.getString(resultSet.getColumnIndex('last_message')),
          lastTime: resultSet.getString(resultSet.getColumnIndex('last_time')),
          unreadCount: resultSet.getDouble(resultSet.getColumnIndex('unread_count')),
          data: resultSet.getString(resultSet.getColumnIndex('json_data'))
        });
      }
      
      resultSet.close();
      return items;
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Get cached chat list failed: ${JSON.stringify(err)}`);
      return [];
    }
  }

  /**
   * 缓存聊天消息记录
   */
  public async cacheMessages(chatId: string, messages: MessageCacheItem[]): Promise<void> {
    const store = this.rdbManager.getStore();
    if (!store) return;

    try {
      store.beginTransaction();
      
      // 删除该会话的旧消息（生产环境可能需要更精细的增量策略）
      const predicates = new relationalStore.RdbPredicates('messages');
      predicates.equalTo('chat_id', chatId);
      await store.delete(predicates);

      for (const msg of messages) {
        const valueBucket: ValuesBucket = {
          id: msg.id,
          chat_id: msg.chatId,
          type: msg.type,
          sender_id: msg.senderId,
          content: msg.content,
          sent_at: msg.sentAt,
          json_data: msg.data
        };
        await store.insert('messages', valueBucket);
      }
      
      store.commit();
      hilog.info(DOMAIN_NUMBER, TAG, `Cached ${messages.length} messages for chat ${chatId}`);
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Cache messages failed: ${JSON.stringify(err)}`);
      try {
        store.rollBack();
      } catch (e) {
        hilog.error(DOMAIN_NUMBER, TAG, `Rollback failed: ${JSON.stringify(e)}`);
      }
    }
  }

  /**
   * 追加单条消息缓存
   */
  public async addMessage(message: MessageCacheItem): Promise<void> {
    const store = this.rdbManager.getStore();
    if (!store) return;

    try {
      const valueBucket: ValuesBucket = {
        id: message.id,
        chat_id: message.chatId,
        type: message.type,
        sender_id: message.senderId,
        content: message.content,
        sent_at: message.sentAt,
        json_data: message.data
      };
      await store.insert('messages', valueBucket);
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Add message failed: ${JSON.stringify(err)}`);
    }
  }

  /**
   * 获取缓存的消息记录
   */
  public async getCachedMessages(chatId: string): Promise<MessageCacheItem[]> {
    const store = this.rdbManager.getStore();
    if (!store) return [];

    try {
      const predicates = new relationalStore.RdbPredicates('messages');
      predicates.equalTo('chat_id', chatId);
      predicates.orderByDesc('sent_at'); // 倒序取出，方便UI显示时反转
      
      const resultSet = await store.query(predicates);
      const items: MessageCacheItem[] = [];

      while (resultSet.goToNextRow()) {
        items.push({
          id: resultSet.getString(resultSet.getColumnIndex('id')),
          chatId: resultSet.getString(resultSet.getColumnIndex('chat_id')),
          type: resultSet.getString(resultSet.getColumnIndex('type')),
          senderId: resultSet.getString(resultSet.getColumnIndex('sender_id')),
          content: resultSet.getString(resultSet.getColumnIndex('content')),
          sentAt: resultSet.getString(resultSet.getColumnIndex('sent_at')),
          data: resultSet.getString(resultSet.getColumnIndex('json_data'))
        });
      }
      
      resultSet.close();
      return items;
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Get cached messages failed: ${JSON.stringify(err)}`);
      return [];
    }
  }

  /**
   * 清空所有聊天/消息缓存（用于登出/切换账号，避免串数据）
   */
  public async clearAll(): Promise<void> {
    const store = this.rdbManager.getStore();
    if (!store) return;
    try {
      store.beginTransaction();
      await store.executeSql('DELETE FROM messages');
      await store.executeSql('DELETE FROM chats');
      store.commit();
      hilog.info(DOMAIN_NUMBER, TAG, 'Cleared all chat/message cache');
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Clear cache failed: ${JSON.stringify(err)}`);
      try {
        store.rollBack();
      } catch (e) {
        hilog.error(DOMAIN_NUMBER, TAG, `Rollback failed: ${JSON.stringify(e)}`);
      }
    }
  }
}
