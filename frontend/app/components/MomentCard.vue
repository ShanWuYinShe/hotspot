<template>
    <ContextMenuWrapper 
      type="moment" 
      :data="moment" 
      @action="handleContextAction"
    >
      <UCard>
        <!-- 用户信息 -->
        <div class="flex items-start gap-3 mb-4">
            <UAvatar :alt="moment.nickname" size="md">
                {{ moment.nickname?.charAt(0) }}
            </UAvatar>

            <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-1">
                    <h4
                        class="font-medium text-neutral-900 dark:text-neutral-100 truncate"
                    >
                        {{ moment.nickname }}
                    </h4>
                    <span class="text-xs text-neutral-500">
                        {{ formatTime(moment.created_at) }}
                    </span>
                </div>

                <div class="flex items-center gap-2 text-xs text-neutral-500">
                    <UIcon name="i-heroicons-map-pin" class="w-3 h-3" />
                    <span>{{ moment.location_name }}</span>
                    <span v-if="moment.distance !== undefined" class="ml-2">
                        · {{ formatDistance(moment.distance) }}
                    </span>
                </div>
            </div>

            <!-- 操作菜单 -->
            <UDropdownMenu
                v-if="menuItems.length > 0"
                :items="menuItems"
                :content="{ side: 'bottom', align: 'end' }"
            >
                <UTooltip :text="$t('moments.moreActions')">
                    <UButton
                        variant="ghost"
                        icon="i-heroicons-ellipsis-horizontal"
                        size="md"
                        :aria-label="$t('moments.moreActions')"
                    />
                </UTooltip>
            </UDropdownMenu>
        </div>

        <!-- 动态内容 -->
        <div class="mb-4">
            <p
                class="text-neutral-800 dark:text-neutral-200 whitespace-pre-wrap leading-relaxed"
            >
                {{ moment.description }}
            </p>
        </div>

        <!-- 标签 -->
        <div v-if="moment.tags && moment.tags.length > 0" class="mb-4">
            <div class="flex flex-wrap gap-2">
                <UBadge
                    v-for="tag in moment.tags"
                    :key="tag"
                    variant="soft"
                    color="primary"
                    size="xs"
                >
                    #{{ tag }}
                </UBadge>
            </div>
        </div>

        <!-- 操作按钮 -->
        <USeparator class="my-3" />
        <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
                <!-- 点赞 -->
                <UButton
                    variant="ghost"
                    size="xs"
                    @click="$emit('like', moment)"
                    :class="{
                        'text-red-500': moment.liked_by_me,
                    }"
                >
                    <UIcon
                        :name="
                            moment.liked_by_me
                                ? 'i-heroicons-heart-solid'
                                : 'i-heroicons-heart'
                        "
                        class="w-4 h-4"
                    />
                    <span class="ml-1">{{ moment.likes_count }}</span>
                </UButton>

                <!-- 评论 -->
                <UButton
                    variant="ghost"
                    size="xs"
                    @click="$emit('comment', moment)"
                >
                    <UIcon
                        name="i-heroicons-chat-bubble-left"
                        class="w-4 h-4"
                    />
                    <span class="ml-1">{{ moment.comments_count }}</span>
                </UButton>
            </div>
        </div>
      </UCard>
    </ContextMenuWrapper>
</template>

<script setup lang="ts">
import type { MomentInfo } from "~/types";

interface Props {
    moment: MomentInfo;
}

defineProps<Props>();

defineEmits<{
    like: [moment: MomentInfo];
    comment: [moment: MomentInfo];
}>();

// 计算属性
const { t } = useI18n();
const menuItems = computed(() => [
    [
        {
            label: t('moments.actions.report'),
            icon: "i-heroicons-flag",
            click: () => handleReport(),
        },
    ],
]);

// 方法
function handleReport() {
    const toast = useSmartToast();
    toast.info({
        title: t('moments.toasts.reportedTitle'),
        description: t('moments.toasts.reportedDesc')
    });
}

// 格式化时间
function formatTime(dateString: string) {
    const date = new Date(dateString);
    const now = new Date();
    const diffInMinutes = Math.floor(
        (now.getTime() - date.getTime()) / (1000 * 60),
    );

    if (diffInMinutes < 1) {
        return t('time.justNow') as string;
    } else if (diffInMinutes < 60) {
        return t('time.minutesAgo', { n: diffInMinutes }) as string;
    } else if (diffInMinutes < 1440) {
        return t('time.hoursAgo', { n: Math.floor(diffInMinutes / 60) }) as string;
    } else if (diffInMinutes < 10080) {
        return t('time.daysAgo', { n: Math.floor(diffInMinutes / 1440) }) as string;
    } else {
        return date.toLocaleDateString();
    }
}

// 格式化距离
function formatDistance(distanceKm: number) {
    if (distanceKm < 1) {
        return `${Math.round(distanceKm * 1000)}m`;
    } else if (distanceKm < 10) {
        return `${distanceKm.toFixed(1)}km`;
    } else {
        return `${Math.round(distanceKm)}km`;
    }
}

// 处理右键菜单操作
const handleContextAction = (action: string, data?: any) => {
  switch (action) {
    case 'viewMoment':
      // 查看动态详情
      navigateTo(`/moments/${moment.value.moment_id}`)
      break
    case 'likeMoment':
      // 点赞动态
      emit('like', moment.value)
      break
    case 'commentMoment':
      // 评论动态
      emit('comment', moment.value)
      break
    case 'shareMoment':
      // 分享动态
      emit('share', moment.value)
      break
    case 'editMoment':
      // 编辑动态
      emit('edit', moment.value)
      break
    case 'deleteMoment':
      // 删除动态
      emit('delete', moment.value)
      break
    case 'reportMoment':
      // 举报动态
      emit('report', moment.value)
      break
  }
}
</script>
