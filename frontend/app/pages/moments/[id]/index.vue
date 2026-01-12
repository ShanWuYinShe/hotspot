<template>
    <div class="flex flex-col gap-6 pb-20">
        <!-- Header -->
        <div class="flex items-center gap-4 shrink-0">
            <UButton
                icon="i-lucide-arrow-left"
                variant="ghost"
                color="neutral"
                to="/moments/nearby"
            />
            <h1 class="text-2xl font-bold text-neutral-900 dark:text-white">
                动态详情
            </h1>
        </div>

        <div class="max-w-2xl mx-auto w-full space-y-6">
            <!-- 加载中 -->
            <div v-if="loading" class="space-y-4">
                <USkeleton shape="rectangle" class="h-16 w-full" />
                <USkeleton shape="rectangle" class="h-64 w-full" />
            </div>

            <template v-else-if="momentCardData">
                <MomentCard
                    :moment="momentCardData"
                    @like="toggleLike"
                    @comment="scrollToComments"
                    @share="shareMoment"
                    @deleted="$router.back()"
                    @updated="fetchMomentDetail"
                />

                <div id="comments" class="mt-6">
                    <CommentList
                        :moment-id="momentId"
                        :initial-total="momentCardData.comments_count"
                        @update:total="updateCommentsCount"
                    />
                </div>
            </template>

            <UEmpty
                v-else
                icon="i-lucide-file-x"
                title="动态不存在"
                description="该动态可能已被删除"
                class="py-20"
            >
                <template #actions>
                    <UButton to="/moments/nearby">返回列表</UButton>
                </template>
            </UEmpty>
        </div>
    </div>
</template>

<script setup lang="ts">
import type { MomentCardData } from "~/types/client";
import { useMomentsService } from "~/services/moments.service";
import { useUsersService } from "~/services/users.service";
import CommentList from "~/components/comments/CommentList.vue";
import type { UserRelation } from "~/types/api";
import MomentCard from "~/components/cards/MomentCard.vue";

definePageMeta({
    title: "动态详情",
    layout: "dashboard",
    requiresAuth: false  // 动态详情页允许未登录用户浏览
});

const route = useRoute();
const momentId = route.params.id as string;
const momentsService = useMomentsService();
const userService = useUsersService();
const momentsStore = useMomentsStore();
const authStore = useAuthStore();
const feedback = useFeedbackStore();
const geolocation = useGeolocation();
const { requireAuth } = useRequireAuth();
const { share, isSupported: isShareSupported } = useShare();
const { copy } = useClipboard();

const loading = ref(true);
const momentData = ref<MomentCardData | null>(null);
const relationship = ref<UserRelation | null>(null);
const isLiked = ref(false);

// 合并数据
const momentCardData = computed((): MomentCardData | null => {
    const base = momentsStore.getMoment(momentId) || momentData.value;
    if (!base) return null;
    
    // 计算点赞数变化：base.is_liked 可能是 undefined（未登录），需要正确处理
    let likesCountDelta = 0;
    if (base.is_liked !== undefined) {
        // 有初始点赞状态（已登录）
        if (isLiked.value && !base.is_liked) {
            likesCountDelta = 1; // 未点赞 -> 已点赞
        } else if (!isLiked.value && base.is_liked) {
            likesCountDelta = -1; // 已点赞 -> 未点赞
        }
    }
    
    return {
        ...base,
        relation: relationship.value,
        is_liked: isLiked.value,
        likes_count: base.likes_count + likesCountDelta,
    } as MomentCardData;
});

onMounted(fetchMomentDetail);

async function fetchMomentDetail() {
    loading.value = true;
    try {
        const coords = await geolocation.getCoordinates();
        const detail = await momentsService.openTheDoor({
            moment_id: momentId,
            reference_point: {
                latitude: coords.latitude,
                longitude: coords.longitude,
            },
        });
        momentData.value = detail as unknown as MomentCardData; // 类型兼容处理

        // 补充信息
        const [likeRes, relRes] = await Promise.allSettled([
            momentsService.checkLiked({ moment_id: momentId }),
            authStore.user?.user_id !== detail.user.user_id
                ? userService.getRelationship({ user_id: detail.user.user_id })
                : Promise.resolve(null),
        ]);

        if (likeRes.status === "fulfilled") {
            isLiked.value = likeRes.value.is_liked;
        }
        if (relRes.status === "fulfilled") {
            relationship.value = relRes.value;
        }
    } catch (e) {
        // error
    } finally {
        loading.value = false;
    }
}

import { useRequireAuth } from "~/composables/useRequireAuth";

async function toggleLike() {
    if (!momentCardData.value) return;
    if (!requireAuth("点赞")) return;
    const wasLiked = isLiked.value;
    isLiked.value = !wasLiked;
    try {
        if (wasLiked)
            await momentsService.unlikeMoment({ moment_id: momentId });
        else await momentsService.likeMoment({ moment_id: momentId });
    } catch {
        isLiked.value = wasLiked;
        feedback.error("操作失败");
    }
}

async function shareMoment() {
    const url = window.location.href;
    if (isShareSupported.value) await share({ title: "分享动态", url });
    else {
        await copy(url);
        feedback.success("链接已复制");
    }
}

function scrollToComments() {
    document.getElementById("comments")?.scrollIntoView({ behavior: "smooth" });
}

function updateCommentsCount(count: number) {
    if (momentData.value) momentData.value.comments_count = count;
}
</script>
