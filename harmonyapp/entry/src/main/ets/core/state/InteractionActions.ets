import { ApiClient } from '../api/ApiClient';
import { GlobalStateManager } from './GlobalStateManager';
import { InteractionStateStore, MomentSocialState } from './InteractionStateStore';
import { AppRouter, LoginRouteParam } from '../router/AppRouter';
import { PageNames } from '../../constants/AppConstants';
import { FollowRequest, UnfollowRequest, FavoriteMomentRequest, UnfavoriteMomentRequest, LikeMomentRequest, UnlikeMomentRequest, JoinGroupRequest, LeaveGroupRequest } from '../types/api';

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
      my_state: { is_liked: false, is_favorited: false }
    };
    this.store.setMomentState(momentId, emptyState);
    return emptyState;
  }

  async toggleMomentLike(momentId: string): Promise<boolean> {
    if (!this.ensureLoggedIn('点赞动态')) {
      return false;
    }
    const snapshot = this.store.cloneMomentState(momentId);
    const current = snapshot ?? this.ensureMomentState(momentId);
    const isLiked = current?.my_state?.is_liked ?? false;
    const nextLiked = !isLiked;
    this.store.updateMomentMyState(momentId, { is_liked: nextLiked });
    this.store.updateMomentCounts(momentId, nextLiked ? 1 : -1);

    try {
      const request: LikeMomentRequest | UnlikeMomentRequest = { moment_id: momentId };
      const response = nextLiked ? await ApiClient.likeMoment(request as LikeMomentRequest)
        : await ApiClient.unlikeMoment(request as UnlikeMomentRequest);
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
