/**
 * LoginGate — 统一的动作级登录门禁
 *
 * 设计依据：Material Design 3「延迟注册（Lazy Registration）」模式。
 * 公开详情页里的轻互动（点赞/收藏/关注/加群/进群聊）在未登录时：
 *   1) 先弹轻量 Toast 提示（告知用户为何需要登录），
 *   2) 随即跳转登录页，并带上 restoreAction/restoreTargetId，
 *   3) 登录成功后由 LoginPage.finishAuthNavigation 自动补做原动作。
 *
 * 本类取代此前散落在 GroupPage/GroupDetailPage/SettingsPage/MomentsFeedPage/
 * SearchPage/ChatPage/MomentPage/HomePage/InteractionActions 中的重复门禁逻辑，
 * 保证「同一动作 → 同一提示文案 + 同一跳转行为 + 同一登录后补做」。
 */

import { GlobalStateManager } from './GlobalStateManager';
import { AppRouter, LoginRouteParam } from '../router/AppRouter';
import { PageNames } from '../../constants/AppConstants';
import { SystemToast } from '../../components/SystemToast';

/** restoreAction 取值（登录后自动补做的动作标识） */
export class RestoreAction {
  static readonly LIKE_MOMENT: string = 'like_moment';
  static readonly FAVORITE_MOMENT: string = 'favorite_moment';
  static readonly FOLLOW_USER: string = 'follow_user';
  static readonly JOIN_GROUP: string = 'join_group';
  static readonly ENTER_GROUP_CHAT: string = 'enter_group_chat';
  static readonly OPEN_PRIVATE_CHAT: string = 'open_private_chat';
  static readonly OPEN_CREATE_SHEET: string = 'open_create_sheet';
}

export interface LoginGateOptions {
  /** 登录后自动补做的动作标识（见 RESTORE_ACTION） */
  restoreAction?: string;
  /** 补做动作所需的目标 id（如 momentId / userId / groupId） */
  restoreTargetId?: string;
  /** 补做动作所需的展示文案（如私聊标题） */
  restoreTitle?: string;
  /** 登录后跳回的页面路由（PageNames），无 restoreAction 时使用 */
  redirectUrl?: string;
}

export class LoginGate {
  /**
   * 动作级门禁：未登录时弹 Toast + 跳登录页（带补做意图）。
   * @param action 人类可读的动作名，用于 Toast 文案，如「点赞动态」「加入群组」
   * @param uiContext 当前 UIContext（用于显示 Toast）
   * @param options 登录后补做相关参数
   * @returns true 表示已登录可继续；false 表示已拦截（调用方应 return）
   */
  static requireAction(
    action: string,
    uiContext: UIContext,
    options?: LoginGateOptions
  ): boolean {
    if (GlobalStateManager.getInstance().isAuthenticated()) {
      return true;
    }
    // 统一提示文案：先告知「为何需要登录」，再跳转，避免静默跳转带来的困惑。
    SystemToast.showWarning(`请先登录后再${action}`, uiContext);

    const params = new LoginRouteParam();
    params.reason = 'action_required';
    params.action = action;
    if (options?.restoreAction) {
      params.restoreAction = options.restoreAction;
    }
    if (options?.restoreTargetId) {
      params.restoreTargetId = options.restoreTargetId;
    }
    if (options?.restoreTitle) {
      params.restoreTitle = options.restoreTitle;
    }
    if (options?.redirectUrl) {
      params.redirectUrl = options.redirectUrl;
    }
    AppRouter.push(PageNames.LOGIN, params);
    return false;
  }
}
