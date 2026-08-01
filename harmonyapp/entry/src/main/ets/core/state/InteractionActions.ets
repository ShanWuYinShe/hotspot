import { ApiClient } from '../api/ApiClient';
import { GlobalStateManager } from './GlobalStateManager';
import { InteractionStateStore, MomentSocialState, isMomentLikedByReactions } from './InteractionStateStore';
import { AppRouter, LoginRouteParam } from '../router/AppRouter';
import { PageNames } from '../../constants/AppConstants';
import { FollowRequest, UnfollowRequest, FavoriteMomentRequest, UnfavoriteMomentRequest, JoinGroupRequest, LeaveGroupRequest, Reaction } from '../types/api';

const ACTION_LOGIN_REASON = 'action_required';

export class InteractionActions {
  private static instance: InteractionActions = new InteractionActions();
  private store = InteractionStateStore.getInstance();
  private globalState = GlobalStateManager.getInstance();

  private constructor() {}

  static getInstance(): InteractionActions {
    return InteractionActions.instance;
  }

  private ensureLoggedIn(action: string): boolean {
    if (this.globalState.isAuthenticated()) {
      return true;
    }
    const params = new LoginRouteParam();
    params.reason = ACTION_LOGIN_REASON;
    params.action = action;
    AppRouter.push(PageNames.LOGIN, params);
    return false;
  }

  private ensureMomentState(momentId: string): MomentSocialState {
    const existing = this.store.getMomentState(momentId);
    if (existing) {
      return existing;
    }
    const emptyState: MomentSocialState = {
      moment_id: momentId,
      likes_count: 0,
      comments_count: 0,
      shares_count: 0,
      my_state: { is_favorited: false }
    };
    this.store.setMomentState(momentId, emptyState);
    return emptyState;
  }

  // 乐观更新 reactions 聚合：对指定 emoji 翻转 reacted_by_me 并调整 count
  private applyReactionToggle(reactions: Reaction[] | undefined, emoji: string, liked: boolean): Reaction[] {
    const list: Reaction[] = reactions ? [...reactions] : [];
    const index = list.findIndex((r: Reaction) => r.emoji === emoji);
    if (index >= 0) {
      const existing = list[index];
      const nextCount = Math.max(0, existing.count + (liked ? 1 : -1));
      list[index] = { emoji: existing.emoji, count: nextCount, reacted_by_me: liked };
      return list;
    }
    if (liked) {
      list.push({ emoji: emoji, count: 1, reacted_by_me: true });
    }
    return list;
  }

  async toggleMomentLike(momentId: string): Promise<boolean> {
    if (!this.ensureLoggedIn('点赞动态')) {
      return false;
    }
    const snapshot = this.store.cloneMomentState(momentId);
    const current = snapshot ?? this.ensureMomentState(momentId);
    // 点赞态由 reactions[].reacted_by_me 推导（响应数据驱动），不再依赖 is_liked 字段
    const isLiked = isMomentLikedByReactions(current.reactions);
    const nextLiked = !isLiked;
    const emoji = ApiClient.getDefaultLikeEmoji();
    this.store.updateMomentReactions(momentId, this.applyReactionToggle(current.reactions, emoji, nextLiked));
    this.store.updateMomentCounts(momentId, nextLiked ? 1 : -1);

    try {
      // 契约：点赞统一走 reactions/set（App 已有 setMomentReaction 封装）
      const response = await ApiClient.setMomentReaction(momentId, emoji, nextLiked);
      if (response.code !== 0) {
        throw new Error(response.message || 'like failed');
      }
      return true;
    } catch (error) {
      if (snapshot) {
        this.store.setMomentState(momentId, snapshot);
      }
      return false;
    }
  }

  async toggleMomentFavorite(momentId: string): Promise<boolean> {
    if (!this.ensureLoggedIn('收藏动态')) {
      return false;
    }
    const snapshot = this.store.cloneMomentState(momentId);
    const current = snapshot ?? this.ensureMomentState(momentId);
    const isFavorited = current?.my_state?.is_favorited ?? false;
    const nextFavorited = !isFavorited;
    this.store.updateMomentMyState(momentId, { is_favorited: nextFavorited });
    try {
      const request: FavoriteMomentRequest | UnfavoriteMomentRequest = { moment_id: momentId };
      const response = nextFavorited
        ? await ApiClient.favoriteMoment(request as FavoriteMomentRequest)
        : await ApiClient.unfavoriteMoment(request as UnfavoriteMomentRequest);
      if (response.code !== 0) {
        throw new Error(response.message || 'favorite failed');
      }
      return true;
    } catch (error) {
      if (snapshot) {
        this.store.setMomentState(momentId, snapshot);
      }
      return false;
    }
  }

  async setFollowUser(userId: string, shouldFollow: boolean, contextAction: string = '关注用户'): Promise<boolean> {
    if (!this.ensureLoggedIn(contextAction)) {
      return false;
    }
    const snapshot = this.store.getUserState(userId);
    this.store.updateUserRelation(userId, { is_following: shouldFollow });
    try {
      const request: FollowRequest | UnfollowRequest = { user_id: userId };
      const response = shouldFollow
        ? await ApiClient.followUser(request as FollowRequest)
        : await ApiClient.unfollowUser(request as UnfollowRequest);
      if (response.code !== 0) {
        throw new Error(response.message || 'follow failed');
      }
      return true;
    } catch (error) {
      if (snapshot?.relation) {
        this.store.updateUserRelation(userId, {
          is_following: snapshot.relation.is_following,
          is_followed_by: snapshot.relation.is_followed_by,
          is_blocked: snapshot.relation.is_blocked
        });
      }
      return false;
    }
  }

  async joinGroup(groupId: string): Promise<boolean> {
    if (!this.ensureLoggedIn('加入群组')) {
      return false;
    }
    const snapshot = this.store.getGroupState(groupId);
    this.store.updateGroupMembership(groupId, { is_member: true });
    try {
      const response = await ApiClient.joinGroup({ group_id: groupId } as JoinGroupRequest);
      if (response.code !== 0) {
        throw new Error(response.message || 'join failed');
      }
      return true;
    } catch (error) {
      if (snapshot) {
        this.store.updateGroupMembership(groupId, snapshot.my_state);
      } else {
        this.store.updateGroupMembership(groupId, { is_member: false });
      }
      return false;
    }
  }

  async leaveGroup(groupId: string): Promise<boolean> {
    if (!this.ensureLoggedIn('退出群组')) {
      return false;
    }
    const snapshot = this.store.getGroupState(groupId);
    this.store.updateGroupMembership(groupId, { is_member: false });
    try {
      const response = await ApiClient.leaveGroup({ group_id: groupId } as LeaveGroupRequest);
      if (response.code !== 0) {
        throw new Error(response.message || 'leave failed');
      }
      return true;
    } catch (error) {
      if (snapshot) {
        this.store.updateGroupMembership(groupId, snapshot.my_state);
      } else {
        this.store.updateGroupMembership(groupId, { is_member: true });
      }
      return false;
    }
  }
}

export const interactionActions = InteractionActions.getInstance();
