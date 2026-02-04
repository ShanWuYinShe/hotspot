<template>
    <UCard
        class="group relative"
        :variant="notification.is_read ? 'outline' : 'subtle'"
        @click="$emit('click')"
    >
        <!-- 选择框 -->
        <div
            class="absolute top-2 left-2 z-10 opacity-0 group-hover:opacity-100 transition-opacity"
            :class="{ 'opacity-100': selected }"
        >
            <UCheckbox
                :model-value="selected"
                @update:model-value="$emit('select')"
                @click.stop
            />
        </div>

        <!-- 主要内容 -->
        <div class="flex items-start gap-4 ml-6">
            <!-- 头像 -->
            <div class="shrink-0 relative">
                <UAvatar
                    :src="notification.avatar"
                    :alt="notification.sender_name || notification.title"
                    size="md"
                />

                <!-- 类型图标 -->
                <UBadge
                    class="absolute -bottom-1 -right-1"
                    :color="typeColor"
                    variant="solid"
                    size="xs"
                    square
                    :icon="typeIcon"
                />
            </div>

            <!-- 通知内容 -->
            <div class="flex-1 min-w-0">
                <div class="flex items-start justify-between gap-3">
                    <div class="flex-1 min-w-0">
                        <h4 class="font-semibold text-neutral-900 dark:text-white text-sm truncate">
                            {{ notification.title }}
                        </h4>
                        <p class="mt-1 line-clamp-2 text-sm text-neutral-600 dark:text-neutral-300">
                            {{ notification.content }}
                        </p>
                    </div>

                    <!-- 时间和状态 -->
                    <div class="flex flex-col items-end gap-2 shrink-0">
                        <UBadge
                            :color="
                                notification.is_read ? 'neutral' : 'primary'
                            "
                            class="flex items-center gap-1"
                        >
                            <UIcon name="i-lucide-clock" :size="14" />
                            <span class="font-medium">{{
                                formatTime(notification.created_at)
                            }}</span>
                        </UBadge>
                        <UChip
                            v-if="!notification.is_read"
                            :standalone="true"
                            size="sm"
                            color="primary"
                        />
                    </div>
                </div>

                <!-- 发送者信息 -->
                <div
                    v-if="notification.sender_name"
                    class="flex items-center gap-2 mt-3"
                >
                    <UBadge class="flex items-center gap-1">
                        <UIcon name="i-lucide-user" :size="14" />
                        <span class="font-medium">{{
                            notification.sender_name
                        }}</span>
                    </UBadge>
                </div>
            </div>
        </div>

        <!-- 操作按钮 -->
        <div
            class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity"
        >
            <div class="flex items-center gap-1">
                <UButton
                    v-if="!notification.is_read"
                    icon="i-lucide-check"
                    variant="ghost"
                    size="sm"
                    @click.stop="$emit('mark-read')"
                />
                <UDropdownMenu :items="actionItems">
                    <UButton
                        icon="i-lucide-ellipsis-vertical"
                        variant="ghost"
                        size="sm"
                        @click.stop
                    />
                </UDropdownMenu>
            </div>
        </div>
    </UCard>
</template>

<script setup lang="ts">
interface NotificationItem {
    notification_id: string;
    type: "system" | "friend" | "group" | "moment" | "message";
    title: string;
    content: string;
    is_read: boolean;
    created_at: string;
    data?: Record<string, unknown>;
    avatar?: string;
    sender_name?: string;
}

interface Props {
    notification: NotificationItem;
    selected?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    selected: false,
});
const { formatTime } = useFormatters();

const emit = defineEmits<{
    click: [];
    select: [];
    "mark-read": [];
    delete: [];
}>();

// 计算属性
const typeIcon = computed(() => {
    switch (props.notification.type) {
        case "friend":
            return "i-lucide-users";
        case "group":
            return "i-lucide-users-round";
        case "moment":
            return "i-lucide-image";
        case "message":
            return "i-lucide-messages-square";
        case "system":
        default:
            return "i-lucide-settings";
    }
});

const typeColor = computed(() => {
    switch (props.notification.type) {
        case "friend":
            return "info";
        case "group":
            return "success";
        case "moment":
            return "warning";
        case "message":
            return "primary";
        case "system":
        default:
            return "neutral";
    }
});

const actionItems = computed(() => [
    [
        {
            label: props.notification.is_read ? "标记未读" : "标记已读",
            icon: props.notification.is_read
                ? "i-lucide-bell"
                : "i-lucide-check",
            click: () => emit("mark-read"),
        },
    ],
    [
        {
            label: "删除",
            icon: "i-lucide-trash-2",
            click: () => emit("delete"),
        },
    ],
]);
</script>
