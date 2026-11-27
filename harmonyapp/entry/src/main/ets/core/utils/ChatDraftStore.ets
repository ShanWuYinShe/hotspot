/**

 * 聊天输入草稿工具（鸿蒙端）
 * 按会话 key 持久化未发送文本：切回会话 / 冷启动都能恢复。
 * 与 Web `lib/chat-draft.ts`、Android `ChatDrafts` 同语义——发送成功、
 * 删除会话、退出登录时清理，避免把上一位登录用户的内容泄露给下一位。
 *
 * 存储形态：单键 JSON map（不是每会话一个 key）。HarmonyOS Preferences 没有
 * 键枚举 API，退出登录需要「清空全部草稿」时无法按前缀删除，集中一个键可整体清除。
 */
import { preferences } from '@kit.ArkData';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { PreferenceStores } from '../../constants/AppConstants';

const DOMAIN_NUMBER: number = 0xFF10;
const TAG: string = 'ChatDraftStore';
const DRAFTS_KEY: string = 'drafts_v1';

export class ChatDraftStore {
  private static prefInstance: preferences.Preferences | null = null;
  private static cache: Map<string, string> | null = null;

  /** 初始化 Preferences（EntryAbility 启动时调用一次） */
  static init(context: Context): void {
    if (ChatDraftStore.prefInstance) {
      return;
    }
    try {
      ChatDraftStore.prefInstance = preferences.getPreferencesSync(context, { name: PreferenceStores.CHAT_DRAFTS });
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '初始化草稿存储失败');
    }
  }

  /** 会话草稿键：私聊按对端 user_id、群聊按 group_id 分键，避免同 id 串档 */
  static key(scope: string, id: string): string {
    return `${scope}:${id}`;
  }

  private static readAll(): Map<string, string> {
    if (ChatDraftStore.cache) {
      return ChatDraftStore.cache;
    }
    const map: Map<string, string> = new Map<string, string>();
    try {
      const pref = ChatDraftStore.prefInstance;
      if (pref) {
        const json = pref.getSync(DRAFTS_KEY, '{}') as string;
        const parsed = JSON.parse(json) as Record<string, string>;
        Object.keys(parsed).forEach((key: string) => {
          map.set(key, parsed[key]);
        });
      }
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '读取草稿失败');
    }
    ChatDraftStore.cache = map;
    return map;
  }

  private static writeAll(map: Map<string, string>): void {
    try {
      const pref = ChatDraftStore.prefInstance;
      if (!pref) {
        return;
      }
      const plain: Record<string, string> = {};
      map.forEach((value: string, key: string) => {
        plain[key] = value;
      });
      pref.putSync(DRAFTS_KEY, JSON.stringify(plain));
      pref.flush();
    } catch {
      hilog.warn(DOMAIN_NUMBER, TAG, '写入草稿失败');
    }
  }

  /** 读取草稿；无草稿 / 存储不可用时返回空串 */
  static get(key: string): string {
    if (!key) {
      return '';
    }
    return ChatDraftStore.readAll().get(key) ?? '';
  }

  /** 保存草稿；空白文本等同清空（不落盘），避免只有空格的草稿被恢复 */
  static set(key: string, text: string): void {
    if (!key) {
      return;
    }
    const map = ChatDraftStore.readAll();
    if (text.trim().length === 0) {
      map.delete(key);
    } else {
      map.set(key, text);
    }
    ChatDraftStore.writeAll(map);
  }

  /** 删除指定会话草稿（发送成功 / 删除会话时调用） */
  static remove(key: string): void {
    if (!key) {
      return;
    }
    const map = ChatDraftStore.readAll();
    if (!map.has(key)) {
      return;
    }
    map.delete(key);
    ChatDraftStore.writeAll(map);
  }

  /** 退出登录 / 切换账号时清空本机全部聊天草稿 */
  static clearAll(): void {
    ChatDraftStore.cache = new Map<string, string>();
    ChatDraftStore.writeAll(ChatDraftStore.cache);
  }
}
