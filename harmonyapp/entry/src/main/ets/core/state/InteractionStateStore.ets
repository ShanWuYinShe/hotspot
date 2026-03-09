import { hilog } from '@kit.PerformanceAnalysisKit';
import { MomentItem, UserItem, GroupItem, MomentMyState, Relationship } from '../types/api';

const DOMAIN_NUMBER: number = 0xFF32;
const TAG: string = 'InteractionStateStore';

const KEY_MOMENT_STATE: string = 'interaction_moment_state';
const KEY_USER_STATE: string = 'interaction_user_state';
const KEY_GROUP_STATE: string = 'interaction_group_state';

export interface MomentSocialState {
  moment_id: string;
  likes_count: number;
  comments_count: number;
  shares_count: number;
  my_state?: MomentMyState;
}

export interface MomentMyStatePartial {
  is_liked?: boolean;
  is_favorited?: boolean;
}

export interface UserRelationState {
  user_id: string;
  relation?: Relationship;
}

export interface RelationshipPartial {
  is_following?: boolean;
  is_followed_by?: boolean;
  is_blocked?: boolean;
}

export interface GroupMemberState {
  is_member?: boolean;
  is_admin?: boolean;
  unread_count?: number;
}

export interface GroupMembershipState {
  group_id: string;
  my_state?: GroupMemberState;
}

export type InteractionEventType = 'moment_updated' | 'user_relation_updated' | 'group_membership_updated';

type MomentStateMap = Record<string, MomentSocialState>;
type UserStateMap = Record<string, UserRelationState>;
type GroupStateMap = Record<string, GroupMembershipState>;

type Listener = (payload: Object) => void;

class InteractionEventHub {
  private listeners: Map<InteractionEventType, Set<Listener>> = new Map();

  on(event: InteractionEventType, handler: Listener): void {
    const existing = this.listeners.get(event) || new Set<Listener>();
    existing.add(handler);
    this.listeners.set(event, existing);
  }

  off(event: InteractionEventType, handler: Listener): void {
    const set = this.listeners.get(event);
    if (!set) {
      return;
    }
    set.delete(handler);
    if (set.size === 0) {
      this.listeners.delete(event);
    }
  }

  emit(event: InteractionEventType, payload: Object): void {
    const set = this.listeners.get(event);
    if (!set) {
      return;
    }
    set.forEach((handler: Listener) => {
      try {
        handler(payload);
      } catch (error) {
        hilog.warn(DOMAIN_NUMBER, TAG, `EventHub handler error: ${JSON.stringify(error)}`);
      }
    });
  }
}

export class InteractionStateStore {
  private static instance: InteractionStateStore = new InteractionStateStore();
  private eventHub: InteractionEventHub = new InteractionEventHub();

  private constructor() {
    if (!AppStorage.has(KEY_MOMENT_STATE)) {
      AppStorage.setOrCreate(KEY_MOMENT_STATE, {} as MomentStateMap);
    }
    if (!AppStorage.has(KEY_USER_STATE)) {
      AppStorage.setOrCreate(KEY_USER_STATE, {} as UserStateMap);
    }
    if (!AppStorage.has(KEY_GROUP_STATE)) {
      AppStorage.setOrCreate(KEY_GROUP_STATE, {} as GroupStateMap);
    }
  }

  static getInstance(): InteractionStateStore {
    return InteractionStateStore.instance;
  }

  on(event: InteractionEventType, handler: Listener): void {
    this.eventHub.on(event, handler);
  }

  off(event: InteractionEventType, handler: Listener): void {
    this.eventHub.off(event, handler);
  }

  patchMomentFromItem(moment: MomentItem): void {
    const state: MomentSocialState = {
      moment_id: moment.moment_id,
      likes_count: moment.likes_count,
      comments_count: moment.comments_count,
      shares_count: moment.shares_count ?? 0,
      my_state: moment.my_state
    };
    this.setMomentState(moment.moment_id, state);
  }

  patchUserFromItem(user: UserItem): void {
    if (!user.user?.user_id) {
      return;
    }
    const state: UserRelationState = {
      user_id: user.user.user_id,
      relation: user.relation
    };
    this.setUserState(user.user.user_id, state);
  }

  patchGroupFromItem(group: GroupItem): void {
    if (!group.group?.group_id) {
      return;
    }
    const state: GroupMembershipState = {
      group_id: group.group.group_id,
      my_state: group.my_state
        ? { is_member: group.my_state.is_joined, is_admin: false, unread_count: group.my_state.unread_count }
        : undefined
    };
    this.setGroupState(group.group.group_id, state);
  }

  getMomentState(momentId: string): MomentSocialState | undefined {
    const map = this.getMomentMap();
    return map[momentId];
  }

  setMomentState(momentId: string, state: MomentSocialState): void {
    const map = this.getMomentMap();
    map[momentId] = state;
    AppStorage.setOrCreate(KEY_MOMENT_STATE, map);
    this.eventHub.emit('moment_updated', state);
  }

  updateMomentCounts(momentId: string, likesDelta: number): void {
    const current = this.getMomentState(momentId);
    if (!current) {
      return;
    }
    const nextLikes = Math.max(0, (current.likes_count || 0) + likesDelta);
    const nextState: MomentSocialState = {
      moment_id: current.moment_id,
      likes_count: nextLikes,
      comments_count: current.comments_count,
      shares_count: current.shares_count,
      my_state: current.my_state
    };
    this.setMomentState(momentId, nextState);
  }

  updateMomentMyState(momentId: string, myState: MomentMyStatePartial | undefined): void {
    const current = this.getMomentState(momentId);
    if (!current) {
      return;
    }
    const merged: MomentMyState = {
      is_liked: myState?.is_liked ?? current.my_state?.is_liked ?? false,
      is_favorited: myState?.is_favorited ?? current.my_state?.is_favorited ?? false
    };
    const nextState: MomentSocialState = {
      moment_id: current.moment_id,
      likes_count: current.likes_count,
      comments_count: current.comments_count,
      shares_count: current.shares_count,
      my_state: merged
    };
    this.setMomentState(momentId, nextState);
  }

  cloneMomentState(momentId: string): MomentSocialState | undefined {
    const state = this.getMomentState(momentId);
    if (!state) {
      return undefined;
    }
    const cloned: MomentSocialState = {
      moment_id: state.moment_id,
      likes_count: state.likes_count,
      comments_count: state.comments_count,
      shares_count: state.shares_count,
      my_state: state.my_state
        ? { is_liked: state.my_state.is_liked, is_favorited: state.my_state.is_favorited }
        : undefined
    };
    return cloned;
  }

  private getMomentMap(): MomentStateMap {
    return AppStorage.get<MomentStateMap>(KEY_MOMENT_STATE) ?? {};
  }

  private setUserState(userId: string, state: UserRelationState): void {
    const map = this.getUserMap();
    map[userId] = state;
    AppStorage.setOrCreate(KEY_USER_STATE, map);
    this.eventHub.emit('user_relation_updated', state);
  }

  getUserState(userId: string): UserRelationState | undefined {
    return this.getUserMap()[userId];
  }

  updateUserRelation(userId: string, relation: RelationshipPartial | undefined): void {
    const current = this.getUserState(userId) || { user_id: userId };
    const merged: Relationship = {
      is_following: relation?.is_following ?? current.relation?.is_following ?? false,
      is_followed_by: relation?.is_followed_by ?? current.relation?.is_followed_by ?? false,
      is_blocked: relation?.is_blocked ?? current.relation?.is_blocked ?? false
    };
    const nextState: UserRelationState = {
      user_id: current.user_id,
      relation: merged
    };
    this.setUserState(userId, nextState);
  }

  private getUserMap(): UserStateMap {
    return AppStorage.get<UserStateMap>(KEY_USER_STATE) ?? {};
  }

  private setGroupState(groupId: string, state: GroupMembershipState): void {
    const map = this.getGroupMap();
    map[groupId] = state;
    AppStorage.setOrCreate(KEY_GROUP_STATE, map);
    this.eventHub.emit('group_membership_updated', state);
  }

  getGroupState(groupId: string): GroupMembershipState | undefined {
    return this.getGroupMap()[groupId];
  }

  updateGroupMembership(groupId: string, myState: GroupMemberState | undefined): void {
    const current = this.getGroupState(groupId) || { group_id: groupId };
    const merged: GroupMemberState = {
      is_member: myState?.is_member ?? current.my_state?.is_member ?? false,
      is_admin: myState?.is_admin ?? current.my_state?.is_admin ?? false,
      unread_count: myState?.unread_count ?? current.my_state?.unread_count ?? 0
    };
    const nextState: GroupMembershipState = {
      group_id: current.group_id,
      my_state: merged
    };
    this.setGroupState(groupId, nextState);
  }

  private getGroupMap(): GroupStateMap {
    return AppStorage.get<GroupStateMap>(KEY_GROUP_STATE) ?? {};
  }
}

export const interactionEventHub = InteractionStateStore.getInstance();
