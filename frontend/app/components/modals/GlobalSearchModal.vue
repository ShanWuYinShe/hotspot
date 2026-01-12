<template>
    <UModal
        :title="t('globalSearch.title')"
        :description="t('globalSearch.description')"
    >
        <template #body="{ close }">
            <UCommandPalette
                v-model:search-term="searchTerm"
                :loading="isSearching"
                :groups="groups"
                :placeholder="t('globalSearch.placeholder')"
                @update:model-value="(item) => handleSelect(item, close)"
            >
                <template #empty>
                    <UEmpty
                        v-if="!searchTerm.trim()"
                        icon="i-lucide-search"
                        title="全局搜索"
                        description="输入关键词，搜索用户、群组与瞬间"
                    >
                        <div
                            class="mt-8 flex gap-4 text-neutral-400 dark:text-neutral-500"
                        >
                                <div class="flex items-center gap-1.5">
                                    <UKbd>Enter</UKbd> 搜索
                                </div>
                                <div class="flex items-center gap-1.5">
                                    <UKbd>Esc</UKbd> 关闭
                                </div>
                            </div>
                    </UEmpty>
                    <UEmpty
                        v-else
                        icon="i-lucide-frown"
                        :title="
                                    hasSearched
                                        ? `未找到「${searchTerm.trim()}」相关结果`
                                        : '正在准备搜索…'
                        "
                        :description="
                                    hasSearched
                                ? '尝试更换关键词'
                                : '继续输入关键词以开始搜索'
                        "
                    />
                </template>
            </UCommandPalette>
        </template>
    </UModal>
</template>

<script setup lang="ts">
import type {
    AvatarProps,
    ChipProps,
    CommandPaletteGroup,
    CommandPaletteItem,
} from "@nuxt/ui";
import type {
    UserItem,
    GroupItem,
    MomentItem,
} from "~/types/api";
import { useUsersService } from "~/services/users.service";
import { useGroupsService } from "~/services/groups.service";
import { useMomentsService } from "~/services/moments.service";
import { useFeedbackStore } from "~/stores/feedback.store";

const { t } = useI18n();
const feedback = useFeedbackStore();

type GlobalSearchEntity = "user" | "group" | "moment";
type GlobalSearchPaletteItem = CommandPaletteItem & {
    _type: GlobalSearchEntity;
    _id: string;
};

const searchTerm = ref("");
const isSearching = ref(false);
const hasSearched = ref(false);

const userResults = ref<UserItem[]>([]);
const groupResults = ref<GroupItem[]>([]);
const momentResults = ref<MomentItem[]>([]);

const usersService = useUsersService();
const groupsService = useGroupsService();
const momentsService = useMomentsService();
const geolocation = useGeolocation();
const authStore = useAuthStore();

const joinedGroupIds = ref<Set<string>>(new Set());

const loadJoinedGroups = async () => {
    if (!authStore.isAuthenticated) return;
    try {
        const response = await groupsService.getJoined({ limit: 100 });
        joinedGroupIds.value = new Set(response.items.map((g) => g.group_chat_id));
    } catch {
        // ignore
    }
};

const isGroupMember = (groupId: string): boolean => {
    return joinedGroupIds.value.has(groupId);
};

onMounted(() => {
    loadJoinedGroups();
});

const searchSeq = ref(0);
const coords = shallowRef<{ latitude: number; longitude: number } | null>(null);

const doSearch = async (keyword: string, opts?: { silent?: boolean }) => {
    const kw = keyword.trim();
    if (!kw) {
        hasSearched.value = false;
        userResults.value = [];
        groupResults.value = [];
        momentResults.value = [];
        return;
    }

    const seq = ++searchSeq.value;
    isSearching.value = true;
    hasSearched.value = true;

    try {
        if (!coords.value) coords.value = await geolocation.getCoordinates();
        const location = coords.value;
        if (!location) return;

        const [users, groups, moments] = await Promise.all([
            usersService
                .searchByKeyword({
                    kw,
                    reference_point: {
                        latitude: location.latitude,
                        longitude: location.longitude,
                    },
                    limit: 20,
                })
                .catch(() => ({ items: [], pagination: { has_more: false } })),
            groupsService
                .searchByKeyword({
                    kw,
                    reference_point: {
                        latitude: location.latitude,
                        longitude: location.longitude,
                    },
                    limit: 20,
                })
                .catch(() => ({ items: [], pagination: { has_more: false } })),
            momentsService
                .searchByKeyword({
                    kw,
                    reference_point: {
                        latitude: location.latitude,
                        longitude: location.longitude,
                    },
                    limit: 20,
                })
                .catch(() => ({ items: [], pagination: { has_more: false } })),
        ]);

        if (seq !== searchSeq.value) return;
        userResults.value = users.items || [];
        groupResults.value = groups.items || [];
        momentResults.value = moments.items || [];
    } catch (err: unknown) {
        if (!opts?.silent) {
            feedback.error("搜索失败", getErrorMessage(err, "请稍后重试"));
        }
        if (seq === searchSeq.value) {
            userResults.value = [];
            groupResults.value = [];
            momentResults.value = [];
        }
    } finally {
        if (seq === searchSeq.value) isSearching.value = false;
    }
};

const debouncedSearch = useDebounceFn(() => {
    void doSearch(searchTerm.value, { silent: true });
}, 250);

watch(searchTerm, () => {
    debouncedSearch();
});

const memberChip: ChipProps = { color: "success", size: "sm" };

const groups = computed<CommandPaletteGroup<CommandPaletteItem>[]>(() => {
    const kw = searchTerm.value.trim();
    const users: CommandPaletteItem[] = userResults.value.map((u) => {
        const avatar: AvatarProps = {
            src: u.avatar_url || undefined,
            alt: u.nickname || undefined,
        };
        const chip: ChipProps | undefined = u.is_online
            ? { color: "success", size: "2xs", position: "bottom-right" }
            : undefined;
        return {
            _type: "user",
            _id: String(u.user_id),
            label: u.nickname || `用户 ${u.user_id}`,
            suffix: u.location?.location_name || "",
            avatar,
            chip,
        } satisfies GlobalSearchPaletteItem;
    });

    const groupsItems: CommandPaletteItem[] = groupResults.value.map((g) => {
        const avatar: AvatarProps = {
            src: g.avatar_url || undefined,
            alt: g.name || undefined,
        };
        return {
            _type: "group",
            _id: g.group_chat_id,
            label: g.name || "群组",
            suffix: `${g.members_count ?? 0} 成员`,
            avatar,
            chip: isGroupMember(g.group_chat_id) ? memberChip : undefined,
        } satisfies GlobalSearchPaletteItem;
    });

    const moments: CommandPaletteItem[] = momentResults.value.map((m) => {
        const avatar: AvatarProps = {
            src: m.user?.avatar_url || undefined,
            alt: m.user?.nickname || undefined,
        };
        return {
            _type: "moment",
            _id: m.moment_id,
            label: m.description || "瞬间",
            prefix: m.user?.nickname || "",
            suffix: m.location?.location_name || "",
            avatar,
        } satisfies GlobalSearchPaletteItem;
    });

    return [
        {
            id: "users",
            label: kw ? `用户 (${users.length})` : "用户",
            items: users,
        },
        {
            id: "groups",
            label: kw ? `群组 (${groupsItems.length})` : "群组",
            items: groupsItems,
        },
        {
            id: "moments",
            label: kw ? `瞬间 (${moments.length})` : "瞬间",
            items: moments,
        },
    ];
});

const handleSelect = (item: CommandPaletteItem, close: () => void) => {
    const it = item as Partial<GlobalSearchPaletteItem>;
    if (!it._type || !it._id) return;
    close();

    if (it._type === "user") {
        const uid = Number(it._id);
        if (!Number.isNaN(uid) && uid === authStore.user?.user_id)
            return void navigateTo("/profile");
        return void navigateTo(`/users/${it._id}`);
    }
    if (it._type === "group") return void navigateTo(`/groups/${it._id}`);
    return void navigateTo(`/moments/${it._id}`);
};
</script>
