/**
 * RDB 数据库管理器
 * 负责 SQLite 数据库的创建、升级和基础 CRUD 操作
 */
import { relationalStore } from '@kit.ArkData';
import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { BusinessError } from '@kit.BasicServicesKit';

const DOMAIN_NUMBER: number = 0xFF09;
const TAG: string = 'RdbManager';

export class RdbManager {
  private static instance: RdbManager;
  private rdbStore: relationalStore.RdbStore | null = null;
  private readonly DB_NAME: string = 'hotspot.db';
  private readonly DB_VERSION: number = 1;

  // SQL 建表语句
  private readonly SQL_CREATE_TABLE_MOMENTS = `
    CREATE TABLE IF NOT EXISTS moments (
      id TEXT PRIMARY KEY,
      user_id INTEGER,
      nickname TEXT,
      avatar_url TEXT,
      description TEXT,
      images TEXT,
      tags TEXT,
      latitude REAL,
      longitude REAL,
      location_name TEXT,
      distance REAL,
      likes_count INTEGER,
      comments_count INTEGER,
      created_at TEXT,
      json_data TEXT
    )
  `;

  private readonly SQL_CREATE_TABLE_MESSAGES = `
    CREATE TABLE IF NOT EXISTS messages (
      id TEXT PRIMARY KEY,
      chat_id TEXT,
      type TEXT, 
      sender_id INTEGER,
      content TEXT,
      sent_at TEXT,
      json_data TEXT
    )
  `;
  
  // 简单缓存聊天列表
  private readonly SQL_CREATE_TABLE_CHATS = `
    CREATE TABLE IF NOT EXISTS chats (
      id TEXT PRIMARY KEY,
      type TEXT, -- 'private' or 'group'
      name TEXT,
      avatar_url TEXT,
      last_message TEXT,
      last_time TEXT,
      unread_count INTEGER,
      json_data TEXT
    )
  `;

  private constructor() {}

  public static getInstance(): RdbManager {
    if (!RdbManager.instance) {
      RdbManager.instance = new RdbManager();
    }
    return RdbManager.instance;
  }

  /**
   * 初始化数据库
   */
  public async init(context: common.Context): Promise<void> {
    const config: relationalStore.StoreConfig = {
      name: this.DB_NAME,
      securityLevel: relationalStore.SecurityLevel.S1
    };

    try {
      this.rdbStore = await relationalStore.getRdbStore(context, config);
      hilog.info(DOMAIN_NUMBER, TAG, 'RdbStore initialized');
      
      // 检查版本并建表（简化版，实际应在 upgrade 回调中处理）
      // 这里为了简单直接执行建表
      await this.createTables();
    } catch (err) {
      const error = err as BusinessError;
      hilog.error(DOMAIN_NUMBER, TAG, `Failed to get RdbStore. Code:${error.code}, message:${error.message}`);
    }
  }

  private async createTables() {
    if (!this.rdbStore) return;
    try {
      await this.rdbStore.executeSql(this.SQL_CREATE_TABLE_MOMENTS);
      await this.rdbStore.executeSql(this.SQL_CREATE_TABLE_MESSAGES);
      await this.rdbStore.executeSql(this.SQL_CREATE_TABLE_CHATS);
      hilog.info(DOMAIN_NUMBER, TAG, 'Tables created');
    } catch (err) {
      hilog.error(DOMAIN_NUMBER, TAG, `Create tables failed: ${JSON.stringify(err)}`);
    }
  }

  public getStore(): relationalStore.RdbStore | null {
    return this.rdbStore;
  }
}










