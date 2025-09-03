import { defineStore } from "pinia";
import { useMomentService } from "../services/moment.service";
import { DEFAULTS } from "~/config/defaults";
import type {
  MomentInfo,
  PaginatedResponse,
  CreateMomentRequest,
} from "~/types";

type Moment = MomentInfo;

/**
 * 动态 Store
 * 仅负责动态数据的缓存，不持久化（数据应从服务器获取最新状态）
 */
export const useMomentsStore = defineStore(
  "moments",
  () => {
    // 缓存状态 - 不持久化
    const myMoments = ref<MomentInfo[]>([]);
    const nearbyMoments = ref<MomentInfo[]>([]);
    const searchResults = ref<MomentInfo[]>([]);
    const currentMoment = ref<MomentInfo | null>(null);

    // 分页状态
    const pagination = ref({
      myMoments: {
        hasMore: false,
        nextCursor: undefined as number | undefined,
      },
      nearbyMoments: {
        hasMore: false,
        nextCursor: undefined as number | undefined,
      },
      searchResults: {
        hasMore: false,
        nextCursor: undefined as number | undefined,
      },
    });

    // 计算属性
    const myMomentsCount = computed(() => myMoments.value.length);
    const nearbyMomentsCount = computed(() => nearbyMoments.value.length);
    const searchResultsCount = computed(() => searchResults.value.length);

    // 业务方法 - 使用 Service 层
    async function createMoment(momentData: CreateMomentRequest) {
      const momentService = useMomentService();
      const newMoment = await momentService.createMoment(
        momentData.description,
        momentData.latitude,
        momentData.longitude,
        momentData.location_name,
        momentData.tags,
      );

      // 创建响应只返回moment_id，需要获取完整的动态信息
      // 这里暂时不添加到列表，让UI刷新列表
      return newMoment;
    }

    async function fetchMyMoments(loadMore = false) {
      const momentService = useMomentService();
      const cursor =
        loadMore && pagination.value.myMoments.nextCursor
          ? pagination.value.myMoments.nextCursor
          : undefined;
      const response = await momentService.getMyMoments(cursor);

      if (loadMore) {
        myMoments.value.push(...response.items);
      } else {
        myMoments.value = response.items;
      }

      pagination.value.myMoments = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };

      return response.items;
    }

    async function fetchNearbyMoments(params: {
      latitude: number;
      longitude: number;
      radius?: number;
      cursor?: number;
      limit?: number;
      loadMore?: boolean;
    }) {
      const momentService = useMomentService();
      const cursor = params.loadMore
        ? pagination.value.nearbyMoments.nextCursor
        : undefined;

      const response = await momentService.getNearbyMoments(
        params.latitude,
        params.longitude,
        params.radius,
        cursor,
        params.limit,
      );

      if (params.loadMore) {
        nearbyMoments.value.push(...response.items);
      } else {
        nearbyMoments.value = response.items;
      }

      pagination.value.nearbyMoments = {
        hasMore: response.has_more,
        nextCursor: response.next_cursor,
      };

      return response.items;
    }

    async function searchMomentsByTags(params: {
      tags: string[];
      latitude?: number;
      longitude?: number;
      cursor?: number;
      limit?: number;
      loadMore?: boolean;
    }) {
      const momentService = useMomentService();
      const cursor = params.loadMore
        ? pagination.value.searchResults.nextCursor
        : undefined;

      const response = await momentService.searchMomentsByTags(
        params.tags,
        params.latitude || DEFAULTS.latitude,
        params.longitude || DEFAULTS.longitude,
        cursor,
        params.limit,
      );

      if (params.loadMore) {
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

    async function fetchMomentDetails(momentId: string) {
      const momentService = useMomentService();
      const moment = await momentService.getMomentById(momentId);
      currentMoment.value = moment;
      return moment;
    }

    async function likeMoment(momentId: string) {
      const momentService = useMomentService();
      try {
        const response = await momentService.likeMoment(momentId);

        // 乐观更新：增加点赞数
        updateMomentInLists(momentId, {
          liked_by_me: true,
          likes_count: response.likes_count,
        });
      } catch (error) {
        console.error("点赞失败:", error);
      }
    }

    async function unlikeMoment(momentId: string) {
      const momentService = useMomentService();
      try {
        const response = await momentService.unlikeMoment(momentId);

        // 乐观更新：减少点赞数
        updateMomentInLists(momentId, {
          liked_by_me: false,
          likes_count: response.likes_count,
        });
      } catch (error) {
        console.error("取消点赞失败:", error);
      }
    }

    async function deleteMoment(momentId: string) {
      const momentService = useMomentService();
      await momentService.deleteMoment(momentId);

      // 从所有列表中移除
      myMoments.value = myMoments.value.filter(
        (m: Moment) => m.moment_id !== momentId,
      );
      nearbyMoments.value = nearbyMoments.value.filter(
        (m: Moment) => m.moment_id !== momentId,
      );
      searchResults.value = searchResults.value.filter(
        (m: Moment) => m.moment_id !== momentId,
      );

      // 清空当前动态
      if (currentMoment.value?.moment_id === momentId) {
        currentMoment.value = null;
      }
    }

    // 辅助函数
    function updateMomentInLists(
      momentId: string,
      updates: Partial<MomentInfo>,
    ) {
      const lists = [myMoments, nearbyMoments, searchResults];

      lists.forEach((list) => {
        const moment = list.value.find((m: Moment) => m.moment_id === momentId);
        if (moment) {
          Object.assign(moment, updates);
        }
      });

      // 更新当前动态
      if (currentMoment.value?.moment_id === momentId) {
        Object.assign(currentMoment.value, updates);
      }
    }

    function getMomentFromLists(momentId: string): MomentInfo | undefined {
      const lists = [myMoments, nearbyMoments, searchResults];

      for (const list of lists) {
        const moment = list.value.find((m: Moment) => m.moment_id === momentId);
        if (moment) return moment;
      }

      return currentMoment.value?.moment_id === momentId
        ? currentMoment.value
        : undefined;
    }

    function clearMyMoments() {
      myMoments.value = [];
      pagination.value.myMoments = { hasMore: false, nextCursor: undefined };
    }

    function clearNearbyMoments() {
      nearbyMoments.value = [];
      pagination.value.nearbyMoments = {
        hasMore: false,
        nextCursor: undefined,
      };
    }

    function clearSearchResults() {
      searchResults.value = [];
      pagination.value.searchResults = {
        hasMore: false,
        nextCursor: undefined,
      };
    }

    return {
      // 只读选择器（返回拷贝）
      myMoments: computed(() => myMoments.value.slice()),
      nearbyMoments: computed(() => nearbyMoments.value.slice()),
      searchResults: computed(() => searchResults.value.slice()),
      currentMoment: computed(() =>
        currentMoment.value ? { ...currentMoment.value } : null,
      ),
      pagination: computed(() => ({
        myMoments: { ...pagination.value.myMoments },
        nearbyMoments: { ...pagination.value.nearbyMoments },
        searchResults: { ...pagination.value.searchResults },
      })),

      // 计算属性
      myMomentsCount,
      nearbyMomentsCount,
      searchResultsCount,

      // 方法
      createMoment,
      fetchMyMoments,
      fetchNearbyMoments,
      searchMomentsByTags,
      fetchMomentDetails,
      likeMoment,
      unlikeMoment,
      deleteMoment,
      clearMyMoments,
      clearNearbyMoments,
      clearSearchResults,
    };
  },
  {
    persist: {
      // 动态数据不持久化，总是从服务器获取最新数据
      pick: [],
    },
  },
);
