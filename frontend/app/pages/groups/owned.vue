<template>
    <div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-14 sm:py-16">
        <UCard>
            <template #header>
                <h1 class="text-3xl font-bold text-neutral-900">我创建的群组</h1>
            </template>

            <USkeleton v-if="loading" class="h-32" />
            <div v-else>
                <UTable :rows="items" :columns="columns">
                        <template #name-cell="{ row }">
                            <div class="flex items-center gap-3 min-w-[220px]">
                                <UAvatar :alt="(row as any).name">{{ (row as any).name?.charAt(0) }}</UAvatar>
                                <div class="min-w-0">
                                    <div class="flex items-center gap-2">
                                        <span class="font-medium truncate">{{ (row as any).name }}</span>
                                        <UBadge color="primary" variant="soft" size="xs">群主</UBadge>
                                        <UBadge v-if="(ownedUnreadMap[String((row as any).group_id)] || 0) > 0" variant="soft" size="xs">未读 {{ ownedUnreadMap[String((row as any).group_id)] || 0 }}</UBadge>
                                    </div>
                                    <p class="text-xs text-neutral-500 truncate">{{ (row as any).description || '—' }}</p>
                                </div>
                            </div>
                        </template>

                        <template #activity-cell="{ row }">
                            <span class="text-sm text-neutral-600 dark:text-neutral-400 hidden md:inline">{{ formatTime((row as any).last_active_at) }}</span>
                        </template>

                        <template #actions-cell="{ row }">
                            <div class="flex items-center justify-end gap-2">
                                <UButton size="sm" @click="enterGroup(row as any)">
                                    <UIcon name="i-heroicons-arrow-right" class="w-4 h-4 mr-1" />进入
                                </UButton>
                                <UButton size="sm" variant="ghost" icon="i-heroicons-cog-6-tooth" @click="manageGroup(row as any)" />
                            </div>
                        </template>

                        <template #empty>
                            <div class="py-10">
                                <EmptyState
                                    icon="i-heroicons-user-group"
                                    title="还没有创建群组"
                                    description="去创建第一个群组吧"
                                    action-text="创建群组"
                                    :action-handler="() => navigateTo('/groups/create')"
                                >
                                    <template #illustration>
                                        <EmptyGroups size="lg" />
                                    </template>
                                </EmptyState>
                            </div>
                        </template>
                </UTable>
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import type { GroupInfo } from "~/types";
import { useMessageService } from '~/services/message.service';
import { formatTime } from '~/utils/formatters';

definePageMeta({ title: "我创建的群组", requiresAuth: true });
const groupsStore = useGroupsStore();
const toast = useSmartToast();
const EmptyGroups = defineAsyncComponent(() => import("~/components/illustrations/EmptyGroups.vue"));
const items = ref<GroupInfo[]>([]);
const loading = ref(true);
const columns = [
  { id: 'name', key: 'name', label: '群组' },
  { id: 'activity', key: 'activity', label: '最近活跃', class: 'hidden md:table-cell w-40' },
  { id: 'actions', key: 'actions', label: '操作', class: 'w-40 text-right' }
] as any;

onMounted(async () => {
    try {
        await groupsStore.fetchOwnedGroups();
        items.value = groupsStore.ownedGroups;
    } catch (e: any) {
        toast.add({
            title: "加载失败",
            description: e?.message,
            color: "error",
        });
    } finally {
        loading.value = false;
    }

    // 懒加载未读数（限前30项以控流）
    try {
        const ms = useMessageService();
        const list = items.value.slice(0, 30);
        await Promise.all(list.map(async (g) => {
            try {
                const r: any = await ms.getGroupUnreadCount(String(g.group_id));
                ownedUnreadMap[String(g.group_id)] = Number(r?.unread_count ?? r?.total_unread_count ?? 0);
            } catch {}
        }));
    } catch {}
});

const enterGroup = (g: GroupInfo) => {
    // 进入前标记已读并清零本地未读
    try {
        const ms = useMessageService();
        void ms.markAsRead(String(g.group_id), 'group');
        ownedUnreadMap[String(g.group_id)] = 0;
    } catch {}
    navigateTo(`/chat/groups/${g.group_id}`);
};
const manageGroup = (g: GroupInfo) => {
    navigateTo(`/groups/${g.group_id}/settings`);
};

// formatTime 函数已从统一工具函数导入

// 我创建的群未读映射
const ownedUnreadMap = reactive<Record<string, number>>({});
</script>
