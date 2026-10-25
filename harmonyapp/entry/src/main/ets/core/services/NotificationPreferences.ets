/**
 * 通知偏好（客户端本地生效）
 *
 * 背景：后端 spec 目前没有通知订阅接口，服务端不做按类型投递过滤；
 * 因此以下两组本地偏好由 App 在展示「应用内通知横幅」前自行判断：
 * - 推送通知总开关（SettingsPage）：关闭后不再弹出任何应用内通知横幅；
 *   通知中心历史与未读数仍由服务端维护，不在此处隐藏（与系统推送设置语义一致）。
 * - 通知类型订阅（SubscriptionsPage）：按后端 NotifyType 细分，映射表见
 *   NOTIFY_TYPE_TO_PREF_KEY。
 *
 * 若将来 spec 增加订阅 API，只需把本模块改为服务端读取/写入，调用方不变。
 */
import { PreferencesUtil } from '../storage/PreferencesUtil';

/** 「推送通知」总开关的偏好键（与 SettingsPage 保持一致） */
export const PUSH_NOTIFY_PREF_KEY: string = 'settings_push_notify_enabled';
/** 通知类型订阅偏好的偏好键（与 SubscriptionsPage 保持一致） */
export const SUBSCRIPTIONS_PREF_KEY: string = 'notification_subscriptions';

/** 通知类型订阅偏好（与服务端 NotifyType 的子集一一对应） */
export interface NotificationSubscriptionPrefs {
  new_follower_enabled?: boolean;
  mutual_follow_enabled?: boolean;
  group_invite_enabled?: boolean;
  group_invite_response_enabled?: boolean;
  group_application_enabled?: boolean;
  group_application_response_enabled?: boolean;
  message_mention_enabled?: boolean;
  moment_like_enabled?: boolean;
  moment_comment_enabled?: boolean;
  comment_reply_enabled?: boolean;
  system_announcement_enabled?: boolean;
}

/**
 * 应用内通知的 metadata.type（见 Index.ets 的通知桥）→ 订阅偏好键。
 * 未列出的类型默认允许（如举报处理结果、附近新瞬间——后者由服务端偏好控制）。
 */
const NOTIFY_TYPE_TO_PREF_KEY: Record<string, string> = {
  'new_follower': 'new_follower_enabled',
  'mutual_follow': 'mutual_follow_enabled',
  'moment_comment': 'moment_comment_enabled',
  'comment_reply': 'comment_reply_enabled',
  'message_mentioned': 'message_mention_enabled',
  'group_invite': 'group_invite_enabled',
  'group_invite_response': 'group_invite_response_enabled',
  'group_application': 'group_application_enabled',
  'group_application_response': 'group_application_response_enabled',
  'reaction_added': 'moment_like_enabled',
  'system_announcement': 'system_announcement_enabled'
};

export class NotificationPreferences {
  /** 应用内通知总开关；读取失败按开启处理（不因偏好异常静默吞通知） */
  static async isPushEnabled(): Promise<boolean> {
    try {
      return await PreferencesUtil.get<boolean>(PUSH_NOTIFY_PREF_KEY, true);
    } catch {
      return true;
    }
  }

  /**
   * 按应用内通知类型判断是否允许弹出。
   * 默认允许：只有用户显式关闭（存 false）才拦截，避免旧数据缺键时误吞通知。
   */
  static async isTypeEnabled(metaType: string): Promise<boolean> {
    const prefKey: string | undefined = NOTIFY_TYPE_TO_PREF_KEY[metaType];
    if (prefKey === undefined) {
      return true;
    }
    try {
      const stored = await PreferencesUtil.getObject<NotificationSubscriptionPrefs>(SUBSCRIPTIONS_PREF_KEY);
      if (!stored) {
        return true;
      }
      const record = (stored as Object) as Record<string, boolean>;
      return record[prefKey] !== false;
    } catch {
      return true;
    }
  }

  /** 是否允许弹出该类型的应用内通知（总开关 + 类型开关） */
  static async isNotificationAllowed(metaType: string): Promise<boolean> {
    const pushEnabled = await NotificationPreferences.isPushEnabled();
    if (!pushEnabled) {
      return false;
    }
    return await NotificationPreferences.isTypeEnabled(metaType);
  }
}