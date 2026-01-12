<template>
    <UCard
        variant="outline"
        class="h-full flex flex-col"
        @click="$emit('click', moment)"
    >
        <!-- 主体内容 -->
        <div class="flex flex-col h-full">
            <!-- 头部：用户信息与时间 -->
            <div class="flex items-start justify-between gap-3">
                <UUser
                    class="min-w-0"
                    :name="moment.user.nickname"
                    :description="formatTime(moment.created_at)"
                    :avatar="{
                        src: moment.user.avatar_url,
                        alt: moment.user.nickname,
                        size: 'md',
                    }"
                />

                <div class="flex items-center gap-2 shrink-0">
                    <UBadge
                        v-if="isMyMoment"
                        label="我的"
                        color="primary"
                        variant="subtle"
                        size="xs"
                    />

                    <!-- 更多操作 -->
                    <UDropdownMenu :items="menuItems">
                        <UButton
                            variant="ghost"
                            color="neutral"
                            size="xs"
                            icon="i-lucide-more-horizontal"
                            @click.stop
                        />
                    </UDropdownMenu>
                </div>
            </div>

            <!-- 动态内容 -->
            <div class="space-y-3 flex-1">
                <p
                    v-if="moment.description"
                    class="text-base text-neutral-600 dark:text-neutral-300 leading-relaxed line-clamp-4 whitespace-pre-wrap"
                >
                    {{ moment.description }}
                </p>

                <!-- 图片展示 -->
                <ImageGrid
                    v-if="moment.images?.length"
                    :images="moment.images.map((i) => i.url)"
                    class="mt-2"
                    @click="
                        ({ image, index }) => handleImageClick(image, index)
                    "
                    @more="$emit('more-images', moment)"
                />

                <!-- 位置和标签 -->
                <div
                    v-if="moment.location?.location_name || moment.tags?.length"
                    class="flex flex-wrap gap-2 pt-1"
                >
                    <UBadge
                        v-if="moment.location?.location_name"
                        :label="moment.location.location_name"
                        icon="i-lucide-map-pin"
                        color="neutral"
                        variant="soft"
                        size="xs"
                        class="max-w-[150px] truncate"
                    />
                    <UBadge
                        v-for="tag in moment.tags"
                        :key="tag"
                        :label="`#${tag}`"
                        color="primary"
                        variant="subtle"
                        size="xs"
                        class="cursor-pointer"
                        @click.stop="$emit('tag-click', tag)"
                    />
                </div>
            </div>
        </div>

        <!-- 底部互动栏 -->
        <template #footer>
            <div class="grid grid-cols-3 gap-1 w-full" @click.stop>
                <UButton
                    variant="ghost"
                    size="sm"
                    :color="(moment.is_liked ?? false) ? 'error' : 'neutral'"
                    :icon="(moment.is_liked ?? false) ? 'i-lucide-heart-fill' : 'i-lucide-heart'"
                    :label="moment.likes_count ? String(moment.likes_count) : '点赞'"
                    class="justify-center"
                    :loading="likeLoading"
                    @click="handleLike"
                />

                <UButton
                    variant="ghost"
                    size="sm"
                    color="neutral"
                    icon="i-lucide-message-circle"
                    :label="
                        moment.comments_count
                            ? String(moment.comments_count)
                            : '评论'
                    "
                    class="justify-center"
                    @click="handleComment"
                />

                <UButton
                    variant="ghost"
                    size="sm"
                    color="neutral"
                    icon="i-lucide-share-2"
                    label="分享"
                    class="justify-center"
                    @click="handleShare"
                />
            </div>
        </template>
    </UCard>
</template>

<script setup lang="ts">
import { useMomentsService } from "~/services/moments.service";
import { useUsersService } from "~/services/users.service";
import type { MomentCardData } from "~/types/client";
import type { DropdownMenuItem } from "@nuxt/ui";

interface Props {
    moment: MomentCardData;
}

interface Emits {
    click: [moment: MomentCardData];
    like: [moment: MomentCardData];
    comment: [moment: MomentCardData];
    share: [moment: MomentCardData];
    more: [moment: MomentCardData];
    deleted: [moment: MomentCardData];
    updated: [momentId: string];
    "tag-click": [tag: string];
    "image-click": [data: { image: string; index: number }];
    "more-images": [moment: MomentCardData];
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const { formatTime } = useFormatters();
const { requireAuth } = useRequireAuth();
const authStore = useAuthStore();
const hydratedUser = useHydratedStore(authStore, "user", null);
const feedback = useFeedbackStore();
const modal = useModal();
const momentService = useMomentsService();
const userService = useUsersService();

const isMyMoment = computed(
    () => props.moment.user.user_id === hydratedUser.value?.user_id,
);

const likeLoading = ref(false);
const handleLike = async () => {
    // 未登录时 is_liked 不存在（undefined），需要先登录
    if (props.moment.is_liked === undefined) {
        if (!requireAuth("点赞")) return;
        return;
    }

    if (likeLoading.value) return;

    const previousLiked = props.moment.is_liked;
    const previousCount = props.moment.likes_count;

    // Optimistic update
    props.moment.is_liked = !previousLiked;
    props.moment.likes_count += props.moment.is_liked ? 1 : -1;
    emit("like", props.moment);

    likeLoading.value = true;
    try {
        if (previousLiked) {
            await momentService.unlikeMoment({ moment_id: props.moment.moment_id });
        } else {
            await momentService.likeMoment({ moment_id: props.moment.moment_id });
        }
    } catch (e) {
        // Revert
        props.moment.is_liked = previousLiked;
        props.moment.likes_count = previousCount;
        feedback.error("操作失败", getErrorMessage(e));
    } finally {
        likeLoading.value = false;
    }
};

// Actions
const handleComment = () => {
    if (!requireAuth("发表评论")) return;
    emit("comment", props.moment);
};
const handleShare = () => {
    if (!requireAuth("分享动态")) return;
    emit("share", props.moment);
};
const handleImageClick = (image: string, index: number) =>
    emit("image-click", { image, index });

const handleEditMoment = async () => {
    const updatedId = await modal.showEditMoment({ ...props.moment });
    if (updatedId) emit("updated", updatedId);
};

const handleDeleteMoment = async () => {
    if (
        !(await modal.showConfirm({
            title: "删除动态",
            message: "确定要删除这条动态吗？此操作无法撤销。",
            confirmText: "删除",
            confirmColor: "error",
        }))
    )
        return;

    try {
        await momentService.deleteMoment({ moment_id: props.moment.moment_id });
        feedback.success("删除成功");
        emit("deleted", props.moment);
    } catch (e) {
        feedback.error("删除失败", getErrorMessage(e));
    }
};

const handleFollowUser = async () => {
    // 符合设计理念：只在执行操作时才要求登录，提供友好的登录引导
    if (!requireAuth("关注用户")) return;
    
    try {
        await userService.follow({ user_id: props.moment.user.user_id });
        feedback.success("关注成功");
    } catch (e) {
        feedback.error("关注失败", getErrorMessage(e));
    }
};

const handleSendMessage = () => {
    // 符合设计理念：只在执行操作时才要求登录，提供友好的登录引导
    if (!requireAuth("发送私信")) return;
    
    if (isMyMoment.value) {
        void navigateTo("/profile");
        return;
    }
    void navigateTo(`/chat/users/${props.moment.user.user_id}`);
};

const handleReportMoment = async () => {
    await modal.showReport("moment", props.moment.moment_id);
    feedback.success("举报已提交");
};

// 菜单项
const menuItems = computed((): DropdownMenuItem[][] => {
    // 符合设计理念：游客可以浏览所有公开内容，只在执行操作时才要求登录
    // 未登录用户也应该能看到完整菜单，点击时通过 requireAuth 引导登录
    // 这样可以让未登录用户了解"登录后能做什么"，引导他们注册/登录
    
    if (isMyMoment.value) {
        // 我的动态：编辑/删除
        return [
            [
                {
                    label: "编辑",
                    icon: "i-lucide-pencil",
                    onSelect: handleEditMoment,
                },
                {
                    label: "删除",
                    icon: "i-lucide-trash-2",
                    color: "error",
                    onSelect: handleDeleteMoment,
                },
            ],
        ];
    }

    // 他人动态：所有用户都能看到完整菜单（关注/私信/举报）
    // 未登录用户点击时会通过 requireAuth 引导登录
    return [
        [
            {
                label: props.moment.relation?.is_following
                    ? "已关注"
                    : "关注用户",
                icon: props.moment.relation?.is_following
                    ? "i-lucide-user-check"
                    : "i-lucide-user-plus",
                disabled: !!props.moment.relation?.is_following,
                onSelect: handleFollowUser,
            },
            {
                label: "私信",
                icon: "i-lucide-message-square",
                onSelect: handleSendMessage,
            },
            {
                label: "举报",
                icon: "i-lucide-flag",
                color: "error",
                onSelect: handleReportMoment,
            },
        ],
    ];
});
</script>
