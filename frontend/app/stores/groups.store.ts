import { defineStore } from "pinia";
import type { GroupResponse } from "~/types/api";

/**
 * 群组 Store
 * 仅负责群组数据的缓存，不持久化（数据应从服务器获取最新状态）
 */
export const useGroupsStore = defineStore(
  "groups",
  () => {
    // 缓存状态 - 不持久化
    const myGroups = ref<GroupResponse[]>([]);
    const nearbyGroups = ref<GroupResponse[]>([]);
    const searchResults = ref<GroupResponse[]>([]);
    const currentGroup = ref<GroupResponse | null>(null);

    // 分页状态
    const pagination = ref({
      myGroups: { hasMore: false, nextCursor: undefined as string | undefined },
      nearbyGroups: {
        hasMore: false,
        nextCursor: undefined as string | undefined,
      },
      searchResults: {
        hasMore: false,
        nextCursor: undefined as string | undefined,
      },
    });

    // 计算属性
    const totalGroupsCount = computed(() => myGroups.value.length);
    // 拥有的群组数量：从 myGroups 中筛选用户角色为 owner 的群组
    const ownedGroupsCount = computed(() => 
      myGroups.value.filter(group => group.user_role === 'owner').length
    );

    // 业务方法 - 使用 Service 层
    async function createGroup(groupData: {
      name: string;
      description?: string;
      latitude: number;
      longitude: number;
      location_name: string;
      password?: string;
    }) {
      const groupService = useGroupService();
      const errorHandler = useErrorHandler();
      
      try {
        const newGroup = await groupService.createGroup(groupData);
        
        // CreateGroupResponse只包含group_id，需要获取完整群组信息
        const fullGroup = await groupService.getGroupDetails(newGroup.group_id);

        // 新创建的群组会包含 user_role: 'owner'，直接添加到 myGroups
        myGroups.value.unshift(fullGroup);

        return fullGroup;
      } catch (error) {
        await errorHandler.handleError(error, 'GroupStore');
        return null;
      }
    }

    async function joinGroup(groupId: string, password?: string) {
      const groupService = useGroupService();
      const joinResponse = await groupService.joinGroup(groupId, password);

      // JoinGroupResponse只包含success和role，需要获取完整群组信息
      const group = await groupService.getGroupDetails(groupId);

      myGroups.value.unshift(group);
      updateGroupMembershipStatus(groupId, true);

      return group;
    }

    async function leaveGroup(groupId: string) {
      const groupService = useGroupService();
      await groupService.leaveGroup(groupId);

      myGroups.value = myGroups.value.filter(
        (g: GroupResponse) => g.group_id !== groupId,
      );
      updateGroupMembershipStatus(groupId, false);
    }

    async function fetchMyGroups(loadMore = false) {
      const groupService = useGroupService();
      const cursor = loadMore
        ? pagination.value.myGroups.nextCursor
        : undefined;
      const response = await groupService.getMyGroups(cursor ? parseInt(cursor) : undefined);

      if (loadMore) {
        myGroups.value.push(...response.items);
      } else {
        myGroups.value = response.items;
      }

      pagination.value.myGroups = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };

      return response.items;
    }

    // 获取拥有的群组：从 myGroups 中筛选用户角色为 owner 的群组
    // 注意：不再需要单独的 fetchOwnedGroups 方法，因为 fetchMyGroups 已包含角色信息
    function getOwnedGroups() {
      return myGroups.value.filter(group => group.user_role === 'owner');
    }

    async function searchNearbyGroups(params: {
      latitude: number;
      longitude: number;
      radius?: number; // 单位：米
      loadMore?: boolean;
    }) {
      const groupService = useGroupService();
      const cursor = params.loadMore
        ? pagination.value.nearbyGroups.nextCursor
        : undefined;

      const response = await groupService.searchNearbyGroups(
        params.latitude,
        params.longitude,
        params.radius,
        cursor ? parseInt(cursor) : undefined,
      );

      if (params.loadMore) {
        nearbyGroups.value.push(...response.items);
      } else {
        nearbyGroups.value = response.items;
      }

      pagination.value.nearbyGroups = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };

      return response.items;
    }

    async function searchGroupsByName(query: string, loadMore = false) {
      const groupService = useGroupService();
      const cursor = loadMore
        ? pagination.value.searchResults.nextCursor
        : undefined;

      const response = await groupService.searchGroupsByName(query, cursor ? parseInt(cursor) : undefined);

      if (loadMore) {
        searchResults.value.push(...response.items);
      } else {
        searchResults.value = response.items;
      }

      pagination.value.searchResults = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };

      return response.items;
    }

    async function fetchGroupDetails(groupId: string) {
      const groupService = useGroupService();
      const group = await groupService.getGroupDetails(groupId);
      currentGroup.value = group;
      return group;
    }

    // 群成员分页与搜索
    const groupMembers = ref<
      Record<string, { items: any[]; hasMore: boolean; nextCursor?: string }>
    >({});
    const memberSearchKeyword = ref("");

    async function fetchGroupMembers(groupId: string, loadMore = false) {
      const groupService = useGroupService();
      const state = groupMembers.value[groupId] || {
        items: [],
        hasMore: false,
        nextCursor: undefined as string | undefined,
      };
      const cursor = loadMore ? state.nextCursor : undefined;
      const res = await groupService.getGroupMembers(groupId, cursor);
      const merged = loadMore ? state.items.concat(res.items) : res.items;
      groupMembers.value[groupId] = {
        items: merged,
        hasMore: res.has_more,
        nextCursor: res.next_cursor,
      };
      return merged;
    }

    function getGroupMembersLocal(groupId: string) {
      return groupMembers.value[groupId]?.items || [];
    }

    async function transferOwnership(groupId: string, newOwnerId: string) {
      const groupService = useGroupService();
      const response = await groupService.transferOwnership(
        groupId,
        newOwnerId,
      );

      // TransferOwnershipResponse只包含success，需要获取更新后的群组信息
      const updatedGroup = await groupService.getGroupDetails(groupId);

      // 更新本地缓存
      updateGroupInLists(groupId, updatedGroup);

      return updatedGroup;
    }

    async function kickMember(groupId: string, memberId: string) {
      const groupService = useGroupService();
      await groupService.kickMember(groupId, memberId);

      // 更新群组成员数量
      const lists = [myGroups, nearbyGroups, searchResults];
      lists.forEach((list) => {
        const group = list.value.find((g: GroupResponse) => g.group_id === groupId);
        if (group && group.member_count > 0) {
          group.member_count--;
        }
      });

      if (
        currentGroup.value?.group_id === groupId &&
        currentGroup.value.member_count > 0
      ) {
        currentGroup.value.member_count--;
      }
    }

    // 辅助函数
    function updateGroupInLists(groupId: string, updates: Partial<GroupResponse>) {
      const lists = [myGroups, nearbyGroups, searchResults];

      lists.forEach((list) => {
        const group = list.value.find((g: GroupResponse) => g.group_id === groupId);
        if (group) {
          Object.assign(group, updates);
        }
      });

      // 更新当前群组
      if (currentGroup.value?.group_id === groupId) {
        Object.assign(currentGroup.value, updates);
      }
    }

    // 辅助函数
    function updateGroupMembershipStatus(groupId: string, isMember: boolean) {
      const lists = [nearbyGroups, searchResults];

      lists.forEach((list) => {
        const group = list.value.find((g: GroupResponse) => g.group_id === groupId);
        if (group) {
          group.is_member = isMember;
        }
      });
    }

    function clearSearchResults() {
      searchResults.value = [];
      pagination.value.searchResults = {
        hasMore: false,
        nextCursor: undefined,
      };
    }

    function clearNearbyGroups() {
      nearbyGroups.value = [];
      pagination.value.nearbyGroups = { hasMore: false, nextCursor: undefined };
    }

    // 用户切换时清理本地缓存，避免跨账号数据串扰
    function resetAll() {
      myGroups.value = [];
      nearbyGroups.value = [];
      searchResults.value = [];
      currentGroup.value = null;
      pagination.value = {
        myGroups: { hasMore: false, nextCursor: undefined },
        nearbyGroups: { hasMore: false, nextCursor: undefined },
        searchResults: { hasMore: false, nextCursor: undefined },
      };
    }

    const auth = useAuthStore();
    watch(
      () => auth.user?.user_id,
      (newId, oldId) => {
        if (newId !== oldId) resetAll();
      },
    );

    return {
      // 只读选择器（返回拷贝）
      myGroups: computed(() => myGroups.value.slice()),
      nearbyGroups: computed(() => nearbyGroups.value.slice()),
      searchResults: computed(() => searchResults.value.slice()),
      currentGroup: computed(() =>
        currentGroup.value ? { ...currentGroup.value } : null,
      ),
      pagination: computed(() => ({
        myGroups: { ...pagination.value.myGroups },
        nearbyGroups: { ...pagination.value.nearbyGroups },
        searchResults: { ...pagination.value.searchResults },
      })),

      // 计算属性
      totalGroupsCount,
      ownedGroupsCount,

      // 方法
      createGroup,
      joinGroup,
      leaveGroup,
      fetchMyGroups,
      getOwnedGroups, // 替代 fetchOwnedGroups，从 myGroups 中筛选
      searchNearbyGroups,
      searchGroupsByName,
      fetchGroupDetails,
      fetchGroupMembers,
      hasMoreGroupMembers: (groupId: string) =>
        !!groupMembers.value[groupId]?.hasMore,
      getGroupMembersLocal,
      transferOwnership,
      kickMember,
      clearSearchResults,
      clearNearbyGroups,
    };
  },
  {
    persist: {
      // 群组数据不持久化，总是从服务器获取最新数据
      pick: [],
    },
  },
);
