<template>
    <div class="flex flex-col gap-6">
        <!-- Header -->
        <div class="flex items-center gap-4">
            <UButton
                variant="ghost"
                color="neutral"
                icon="i-lucide-arrow-left"
                @click="$router.back()"
            />
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                群组详情
            </h1>
        </div>

        <div class="max-w-2xl mx-auto w-full space-y-6 pb-20">
            <!-- 加载中 -->
            <div v-if="loading" class="space-y-4">
                <USkeleton shape="rectangle" class="h-32 w-full" />
                <USkeleton shape="rectangle" class="h-12 w-full" />
                <USkeleton shape="rectangle" class="h-64 w-full" />
            </div>

            <template v-else-if="group">
                <!-- 群组基本信息卡片 -->
                <UCard variant="solid">
                    <div class="flex flex-col sm:flex-row gap-6">
                        <UAvatar
                            :src="group.avatar_url"
                            :alt="group.name"
                            size="3xl"
                            class="mx-auto sm:mx-0"
                        />

                        <div class="flex-1 text-center sm:text-left space-y-3">
                            <div>
                                <h1
                                    class="text-2xl font-bold text-neutral-900 dark:text-white"
                                >
                                    {{ group.name }}
                                </h1>
                                <p
                                    class="text-neutral-600 dark:text-neutral-400 mt-2"
                                >
                                    {{ group.description || "暂无介绍" }}
                                </p>
                            </div>

                            <div
                                class="flex flex-wrap justify-center sm:justify-start gap-2"
                            >
                                <UBadge
                                    color="neutral"
                                    variant="soft"
                                    icon="i-lucide-users"
                                >
                                    {{ group.members_count }} 成员
                                </UBadge>
                                <UBadge
                                    v-if="group.has_password"
                                    color="warning"
                                    variant="soft"
                                    icon="i-lucide-lock"
                                >
                                    私密群组
                                </UBadge>
                                <UBadge
                                    v-if="group.location?.location_name"
                                    color="neutral"
                                    variant="soft"
                                    icon="i-lucide-map-pin"
                                >
                                    {{ group.location.location_name }}
                                    <template v-if="group.location.distance">
                                        ·
                                        {{
                                            formatDistance(
                                                group.location.distance,
                                            )
                                        }}</template
                                    >
                                </UBadge>
                            </div>
                        </div>
                    </div>

                    <template #footer>
                        <div class="flex justify-end gap-3">
                            <UButton
                                v-if="isMember"
                                block
                                size="lg"
                                color="primary"
                                icon="i-lucide-message-circle"
                                @click="enterChat"
                            >
                                进入聊天
                            </UButton>
                            <UButton
                                v-else
                                block
                                size="lg"
                                color="primary"
                                icon="i-lucide-user-plus"
                                :loading="joining"
                                @click="handleJoin"
                            >
                                申请加入
                            </UButton>
                        </div>
                    </template>
                </UCard>

                <!-- 群组详细信息 -->
                <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
                    <UCard variant="subtle">
                        <template #header>
                            <h3 class="font-semibold flex items-center gap-2">
                                <UIcon
                                    name="i-lucide-info"
                                    class="text-primary-500"
                                />
                                群组公告
                            </h3>
                        </template>
                        <p class="text-neutral-500 italic">暂无公告</p>
                    </UCard>

                    <UCard variant="subtle">
                        <template #header>
                            <h3 class="font-semibold flex items-center gap-2">
                                <UIcon
                                    name="i-lucide-users"
                                    class="text-primary-500"
                                />
                                群组成员
                            </h3>
                        </template>
                        <div
                            class="flex items-center -space-x-2 overflow-hidden py-2"
                        >
                            <UAvatar
                                v-for="i in Math.min(5, group.members_count)"
                                :key="i"
                                :src="undefined"
                                :alt="`Member ${i}`"
                            />
                            <div
                                v-if="group.members_count > 5"
                                class="flex items-center justify-center w-8 h-8 rounded-full bg-neutral-100 dark:bg-neutral-800 ring-2 ring-white dark:ring-neutral-900 text-xs font-medium text-neutral-500"
                            >
                                +{{ group.members_count - 5 }}
                            </div>
                        </div>
                    </UCard>
                </div>
            </template>

            <UEmpty
                v-else
                icon="i-lucide-users"
                title="群组不存在"
                description="该群组可能已被解散"
                class="py-20"
            >
                <template #actions>
                    <UButton to="/groups">返回列表</UButton>
                </template>
            </UEmpty>
        </div>
    </div>
</template>

<script setup lang="ts">
import { useGroupsService } from "~/services/groups.service";
import type { GroupItem } from "~/types/api";

definePageMeta({
    layout: "dashboard",
    requiresAuth: false  // 群组详情页允许未登录用户浏览（后端会控制权限）
});

const { formatDistance } = useFormatters();
const { requireAuth } = useRequireAuth();
const route = useRoute();
const router = useRouter();
const groupsService = useGroupsService();
const geolocation = useGeolocation();
const feedback = useFeedbackStore();
const modal = useModal();

const groupId = route.params.id as string;
const loading = ref(true);
const group = ref<GroupItem | null>(null);
const joining = ref(false);
const isMember = ref(false);

onMounted(async () => {
    await fetchGroupDetail();
});

async function fetchGroupDetail() {
    loading.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        const res = await groupsService.openTheDoor({
            group_chat_id: groupId,
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
        });
        
        // 调试：打印返回的数据
        console.log('[GroupDetail] API Response:', {
            group_chat_id: res.group_chat_id,
            name: res.name,
            is_joined: res.is_joined,
            my_role: res.my_role,
            has_is_joined_field: 'is_joined' in res,
            full_response: res
        });
        
        group.value = res;
        // 使用 API 返回的 is_joined 字段判断是否已加入
        // 注意：未登录时 is_joined 为 undefined，此时应显示"加入群组"
        isMember.value = res.is_joined ?? false;
        
        console.log('[GroupDetail] isMember set to:', isMember.value);
    } catch (e) {
        console.error('[GroupDetail] fetchGroupDetail error:', e);
        // error
    } finally {
        loading.value = false;
    }
}

async function handleJoin() {
    if (!requireAuth("加入群组")) return;
    if (group.value?.has_password) {
        const password = await modal.showGroupPassword(group.value.name);
        if (password) {
            await confirmJoin(password);
        }
    } else {
        await confirmJoin();
    }
}

async function confirmJoin(password?: string) {
    // 符合设计理念：只在执行操作时才要求登录，提供友好的登录引导
    // 注意：这里不需要再次检查 requireAuth，因为 handleJoin 已经检查过了
    // 但为了防御性编程，我们仍然检查一次（防止直接调用 confirmJoin）
    if (!requireAuth("加入群组")) return;
    
    if (!group.value) return;
    joining.value = true;
    try {
        await groupsService.joinGroup({
            group_chat_id: group.value.group_chat_id,
            password,
        });
        feedback.success("加入成功");
        isMember.value = true;
    } catch (e) {
        // error handled by service interceptor
    } finally {
        joining.value = false;
    }
}

function enterChat() {
    if (!requireAuth("进入聊天")) return;
    router.push(`/chat/groups/${groupId}`);
}
</script>
