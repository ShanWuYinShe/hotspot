<template>
    <div class="flex h-full min-h-0 w-full flex-col">
        <!-- Empty -->
        <UEmpty
            v-if="messages.length === 0"
            icon="i-lucide-messages-square"
            title="暂无消息"
            description="发送一条消息开始对话"
            class="h-full"
        >
            <template v-if="$slots.empty" #default>
                <slot name="empty" />
            </template>
        </UEmpty>

        <!-- Scroll container (Nuxt UI) -->
        <UScrollArea
            v-else
            ref="scrollAreaRef"
            class="flex-1 min-h-0"
        >
            <div ref="containerRef" class="space-y-6 scroll-smooth px-4 py-4">
                <!-- Message row -->
                <div
                    v-for="message in messages"
                    :key="message.id"
                    :data-message-id="message.id"
                    class="flex w-full gap-3"
                    :class="
                        message.role === 'user'
                            ? 'justify-end'
                            : 'justify-start'
                    "
                >
                    <!-- Avatar (other) - Only in Group Chat -->
                    <div
                        v-if="message.role !== 'user' && isGroupChat"
                        class="shrink-0 flex items-end"
                    >
                        <slot name="leading" :message="message">
                            <UAvatar
                                :src="
                                    message.metadata?.senderAvatar ||
                                    assistant?.avatar?.src
                                "
                                :alt="
                                    message.metadata?.senderName ||
                                    assistant?.avatar?.alt ||
                                    assistant?.avatar?.fallback
                                "
                                :fallback="
                                    message.metadata?.senderName?.charAt(0) ||
                                    assistant?.avatar?.fallback
                                "
                                size="md"
                                class="cursor-pointer"
                                @click="emit('avatarClick', message)"
                            />
                        </slot>
                    </div>

                    <!-- Content -->
                    <div
                        class="flex min-w-0 flex-col gap-1"
                        :class="
                            message.role === 'user'
                                ? 'items-end'
                                : 'items-start'
                        "
                    >
                        <!-- Group sender label -->
                        <UBadge
                            v-if="
                                isGroupChat &&
                                message.role !== 'user' &&
                                message.metadata?.senderName
                            "
                            variant="subtle"
                            color="neutral"
                            size="xs"
                        >
                            {{ message.metadata.senderName }}
                        </UBadge>

                        <!-- Bubble -->
                        <div
                            class="w-full max-w-[min(40rem,calc(100vw-6rem))] px-3.5 py-2.5 rounded-2xl ring-1 ring-neutral-200 dark:ring-neutral-700 shadow-sm"
                            :class="[
                                message.role === 'user'
                                    ? 'bg-primary-500 text-white dark:bg-primary-600'
                                    : 'bg-white dark:bg-neutral-800 text-neutral-900 dark:text-white',
                            ]"
                        >
                            <div class="text-sm leading-relaxed wrap-break-word">
                                <slot name="content" :message="message">
                                    {{ message.content }}
                                </slot>
                            </div>
                        </div>

                        <!-- Timestamp -->
                        <UBadge
                            v-if="getMessageTimestamp(message)"
                            variant="subtle"
                            color="neutral"
                            size="xs"
                            class="select-none"
                        >
                            {{ displayTime(getMessageTimestamp(message)!) }}
                        </UBadge>
                    </div>

                    <!-- Avatar (me) - Only in Group Chat -->
                    <div
                        v-if="message.role === 'user' && isGroupChat"
                        class="shrink-0 flex items-end"
                    >
                        <UAvatar
                            :src="user?.avatar?.src"
                            :alt="user?.avatar?.alt || user?.avatar?.fallback"
                            :fallback="user?.avatar?.fallback"
                            size="md"
                            class="cursor-pointer"
                            @click="emit('avatarClick', message)"
                        />
                    </div>
                </div>

                <!-- typing indicator -->
                <div v-if="$slots.indicator" class="pt-2">
                    <slot name="indicator" />
                </div>

                <!-- bottom anchor -->
                <div ref="bottomRef" class="h-px w-full" />
            </div>
        </UScrollArea>
    </div>
</template>

<script setup lang="ts">
import type { ClientMessageType } from "~/types/client";

// Chat message types
export interface ChatMessageMetadata {
    senderId?: number | string;
    senderName?: string;
    senderAvatar?: string;
    timestamp?: string | Date;
}

export interface ChatMessage {
    id: string;
    role: "user" | "assistant" | "system";
    content?: string;
    type?: ClientMessageType | "typing";
    createdAt?: string | Date;
    timestamp?: string;
    metadata?: ChatMessageMetadata;
    [key: string]: unknown;
}

export interface ChatAvatarConfig {
    avatar?: {
        alt?: string;
        src?: string;
        fallback?: string;
    };
    variant?: "solid" | "soft" | "outline" | "ghost";
    side?: "left" | "right";
}

interface Props {
    messages: ChatMessage[];
    status?: "idle" | "streaming" | "submitted" | "ready" | "error";
    user?: ChatAvatarConfig;
    assistant?: ChatAvatarConfig;
    shouldAutoScroll?: boolean;
    shouldScrollToBottom?: boolean;
    isGroupChat?: boolean;
    highlightMessageId?: string;
}

const props = withDefaults(defineProps<Props>(), {
    status: "idle",
    shouldAutoScroll: true,
    shouldScrollToBottom: true,
    isGroupChat: false,
});

const emit = defineEmits<{
    (e: "avatarClick", message: ChatMessage): void;
}>();

// `containerRef` is used for querying message elements.
// `scrollAreaRef` keeps a reference to Nuxt UI ScrollArea instance (or root element).
const scrollAreaRef = ref<unknown>(null);
const containerRef = ref<HTMLElement | null>(null);
const bottomRef = ref<HTMLElement | null>(null);

function getScrollViewportEl(): HTMLElement | null {
    const anyRef = scrollAreaRef.value;
    
    if (!anyRef) return null;

    // Helper to check if object has $el
    const hasEl = (obj: unknown): obj is { $el: HTMLElement } => {
        return typeof obj === 'object' && obj !== null && '$el' in obj;
    };

    const root = hasEl(anyRef) ? anyRef.$el : (anyRef as HTMLElement);

    if (!root) return null;

    // Nuxt UI ScrollArea renders a viewport — prefer it if present.
    const viewport =
        (root.querySelector(
            "[data-radix-scroll-area-viewport]",
        ) as HTMLElement | null) ?? root;

    return viewport;
}

// 获取消息时间戳
const getMessageTimestamp = (
    message: ChatMessage,
): string | Date | undefined => {
    if (message.createdAt) return message.createdAt;
    if (message.timestamp) return message.timestamp;
    if (message.metadata?.timestamp) return message.metadata.timestamp;
    return undefined;
};

// 显示时间
const { formatDate } = useFormatters();
const displayTime = (date: string | Date) => formatDate(date, "time");

const scrollToBottom = (behavior: ScrollBehavior = "smooth") => {
    const viewport = getScrollViewportEl();
    if (viewport) {
        viewport.scrollTo({ top: viewport.scrollHeight, behavior });
        return;
    }

    if (!bottomRef.value) return;
    bottomRef.value.scrollIntoView({ block: "end", behavior });
};

const scrollToMessage = (id: string, behavior: ScrollBehavior = "smooth") => {
    if (!id) return;

    const container = containerRef.value;
    if (!container) return;

    const target = container.querySelector(
        `[data-message-id="${id}"]`,
    ) as HTMLElement | null;

    const doScroll = (el: HTMLElement) => {
        const viewport = getScrollViewportEl();
        if (!viewport) {
            el.scrollIntoView({ block: "center", behavior });
            return;
        }

        const viewportRect = viewport.getBoundingClientRect();
        const targetRect = el.getBoundingClientRect();

        const delta =
            targetRect.top -
            viewportRect.top -
            (viewportRect.height / 2 - targetRect.height / 2);

        viewport.scrollTo({
            top: viewport.scrollTop + delta,
            behavior,
        });
    };

    if (!target) {
        nextTick(() => {
            const retryTarget = containerRef.value?.querySelector(
                `[data-message-id="${id}"]`,
            ) as HTMLElement | null;
            if (retryTarget) doScroll(retryTarget);
        });
        return;
    }

    doScroll(target);
};

// 自动滚动逻辑
watch(
    () => props.messages.length,
    (newLen, oldLen) => {
        if (props.shouldAutoScroll && newLen > oldLen) {
            nextTick(() => scrollToBottom("smooth"));
        }
    },
);

// 初始滚动
onMounted(() => {
    if (props.shouldScrollToBottom) {
        nextTick(() => scrollToBottom("auto"));
    }
});

// 暴露方法给父组件
defineExpose({
    scrollToBottom,
    scrollToMessage,
    scrollAreaRef,
});
</script>
