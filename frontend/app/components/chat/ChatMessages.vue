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
                                    <!-- Text -->
                                    <template v-if="message.type === 'text' || !message.type">
                                        {{ message.content }}
                                    </template>

                                    <!-- Image -->
                                    <template v-else-if="message.type === 'image' && message.media">
                                        <div class="relative group">
                                            <img 
                                                :src="message.media.url" 
                                                :alt="message.content || 'Image'" 
                                                class="rounded-lg max-w-full max-h-80 object-cover cursor-pointer hover:opacity-90 transition-opacity"
                                                @click="openPreview(message.media.url)"
                                            />
                                        </div>
                                    </template>

                                    <!-- Video -->
                                    <template v-else-if="message.type === 'video' && message.media">
                                        <video 
                                            :src="message.media.url" 
                                            controls 
                                            class="rounded-lg max-w-full max-h-80 w-full bg-black"
                                        ></video>
                                    </template>

                                    <!-- Audio -->
                                    <template v-else-if="message.type === 'audio' && message.media">
                                        <audio 
                                            :src="message.media.url" 
                                            controls 
                                            class="w-full min-w-[240px]"
                                        ></audio>
                                    </template>

                                    <!-- File -->
                                    <template v-else-if="message.type === 'file' && message.media">
                                        <a 
                                            :href="message.media.url" 
                                            target="_blank"
                                            download
                                            class="flex items-center gap-3 p-3 rounded-lg bg-neutral-100/50 dark:bg-neutral-900/50 hover:bg-neutral-100 dark:hover:bg-neutral-900 border border-neutral-200 dark:border-neutral-700 transition-colors no-underline group"
                                        >
                                            <div class="p-2 bg-white dark:bg-neutral-800 rounded-md shadow-sm group-hover:scale-105 transition-transform">
                                                <UIcon name="i-lucide-file" class="w-5 h-5 text-primary-500" />
                                            </div>
                                            <div class="flex-1 min-w-0">
                                                <div class="truncate font-medium text-sm" :class="message.role === 'user' ? 'text-white' : 'text-neutral-900 dark:text-white'">
                                                    {{ message.media.file_name || message.content || '未命名文件' }}
                                                </div>
                                                <div class="text-xs opacity-70 mt-0.5">
                                                    {{ formatFileSize(message.media.file_size) }}
                                                </div>
                                            </div>
                                            <UIcon name="i-lucide-download" class="w-4 h-4 opacity-70" />
                                        </a>
                                    </template>

                                    <!-- Location -->
                                    <template v-else-if="message.type === 'location' && message.location">
                                         <div class="flex flex-col gap-2 min-w-[200px]">
                                            <div class="flex items-start gap-2">
                                                <div class="mt-0.5 p-1.5 bg-primary-100 dark:bg-primary-900/30 rounded-md">
                                                    <UIcon name="i-lucide-map-pin" class="w-4 h-4 text-primary-600 dark:text-primary-400" />
                                                </div>
                                                <div>
                                                    <div class="font-medium text-sm">{{ message.location.location_name }}</div>
                                                    <div v-if="message.location_address" class="text-xs opacity-70 mt-0.5 break-all">
                                                        {{ message.location_address }}
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="text-xs opacity-50 pl-8">
                                                {{ message.location.latitude.toFixed(6) }}, {{ message.location.longitude.toFixed(6) }}
                                            </div>
                                         </div>
                                    </template>

                                    <!-- Fallback -->
                                    <template v-else>
                                         {{ message.content }}
                                    </template>
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

        <!-- Image Preview Modal -->
        <UModal v-model="isOpen">
            <div class="p-1">
                <img :src="previewImage || ''" class="w-full h-auto rounded-lg" alt="Preview" />
            </div>
        </UModal>
    </div>
</template>

<script setup lang="ts">
import type { ClientMessageType } from "~/types/client";
import type { MessageMedia, Location } from "~/types/api";

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
    media?: MessageMedia;
    location?: Location;
    location_address?: string;
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
const { formatDate, formatFileSize } = useFormatters();
const displayTime = (date: string | Date) => formatDate(date, "time");

// Image preview
const previewImage = ref<string | null>(null);
const isOpen = computed({
    get: () => !!previewImage.value,
    set: (val) => { if (!val) previewImage.value = null; }
});

const openPreview = (url: string) => {
    previewImage.value = url;
};

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
