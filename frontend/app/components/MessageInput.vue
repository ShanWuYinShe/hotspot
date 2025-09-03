<template>
    <UCard class="rounded-none">
        <!-- 回复预览 -->
        <div
            v-if="replyingTo"
            class="mb-3 p-3 bg-neutral-100 dark:bg-neutral-800 rounded-lg"
        >
            <div class="flex items-center justify-between">
                <div class="flex-1 min-w-0">
                    <div class="text-xs text-neutral-500 mb-1">
                        回复 {{ replyingTo.sender_name }}
                    </div>
                    <div
                        class="text-sm text-neutral-700 dark:text-neutral-300 truncate"
                    >
                        {{ replyingTo.content }}
                    </div>
                </div>
                <UTooltip text="取消回复">
                    <UButton
                        variant="ghost"
                        size="xs"
                        icon="i-heroicons-x-mark"
                        aria-label="取消回复"
                        @click="cancelReply"
                    />
                </UTooltip>
            </div>
        </div>

        <div class="flex items-end space-x-3">
            <!-- 附加功能按钮 -->
            <UDropdownMenu :items="attachmentItems">
                <UTooltip text="更多附件">
                    <UButton
                        variant="ghost"
                        size="md"
                        icon="i-heroicons-plus"
                        aria-label="更多附件"
                        class="flex-shrink-0"
                    />
                </UTooltip>
            </UDropdownMenu>

            <!-- 输入框容器 -->
            <div class="flex-1 relative">
                <UTextarea
                    v-model="message"
                    :placeholder="placeholder"
                    :rows="Math.min(Math.max(1, message.split('\n').length), 4)"
                    resize="none"
                    size="md"
                    class="resize-none text-base rounded-lg"
                    @keydown.enter.exact.prevent="sendMessage"
                    @keydown.enter.shift.prevent="addNewLine"
                    @paste="handlePaste"
                    autofocus
                />

                <!-- 输入提示 -->
                <div class="flex items-center justify-between mt-1">
                    <div class="text-xs text-neutral-500 flex items-center gap-1">
                        按 <UKbd>Enter</UKbd> 发送，<UKbd>Shift</UKbd> + <UKbd>Enter</UKbd> 换行
                    </div>
                    <div class="flex items-center space-x-2">
                        <!-- 表情按钮（使用 UPopover） -->
                        <UPopover v-model:open="showEmojiPicker">
                            <UTooltip text="选择表情">
                                <UButton
                                    variant="ghost"
                                    size="sm"
                                    icon="i-heroicons-face-smile"
                                    aria-label="选择表情"
                                />
                            </UTooltip>
                            <template #content>
                                <div class="p-3 bg-neutral-50 dark:bg-neutral-800 rounded-lg emoji-picker">
                                    <div class="grid grid-cols-8 gap-2">
                                        <UButton
                                            v-for="emoji in commonEmojis"
                                            :key="emoji"
                                            size="xs"
                                            variant="ghost"
                                            class="text-xl"
                                            :aria-label="`选择表情 ${emoji}`"
                                            @click="insertEmoji(emoji)"
                                        >
                                            {{ emoji }}
                                        </UButton>
                                    </div>
                                </div>
                            </template>
                        </UPopover>
                        <!-- 字数统计 -->
                        <span class="text-xs text-neutral-400">
                            {{ message.length }}/{{ maxLength }}
                        </span>
                    </div>
                </div>
            </div>

            <!-- 发送按钮 -->
            <UTooltip text="发送">
                <UButton
                    :disabled="!canSend"
                    @click="sendMessage"
                    :loading="sending"
                    size="lg"
                    aria-label="发送"
                    class="flex-shrink-0 transition-transform duration-150"
                >
                    <UIcon name="i-heroicons-paper-airplane" class="w-5 h-5" />
                </UButton>
            </UTooltip>
        </div>

        
        <!-- 隐藏的文件选择输入 -->
        <UInput
            ref="fileInputRef"
            type="file"
            :accept="fileAccept"
            class="hidden"
            @change="onFileSelected"
        />
    </UCard>
</template>

<script setup lang="ts">
interface Props {
    placeholder?: string;
    maxLength?: number;
    sending?: boolean;
    replyingTo?: any;
}

interface Emits {
    (e: "send", message: string): void;
    (e: "upload-file", file: File): void;
    (e: "send-location"): void;
    (e: "cancel-reply"): void;
}

const props = withDefaults(defineProps<Props>(), {
    placeholder: "输入消息...",
    maxLength: 2000,
});

const emit = defineEmits<Emits>();

const toast = useSmartToast();

// 响应式状态
const message = ref("");
const showEmojiPicker = ref(false);

// 计算属性
const canSend = computed(() => {
    return (
        message.value.trim() &&
        !props.sending &&
        message.value.length <= props.maxLength
    );
});

// 常用表情
const commonEmojis = [
    "😀",
    "😃",
    "😄",
    "😁",
    "😆",
    "😅",
    "😂",
    "🤣",
    "😊",
    "😇",
    "🙂",
    "🙃",
    "😉",
    "😌",
    "😍",
    "🥰",
    "😘",
    "😗",
    "😙",
    "😚",
    "😋",
    "😛",
    "😝",
    "😜",
    "🤪",
    "🤨",
    "🧐",
    "🤓",
    "😎",
    "🤩",
    "🥳",
    "😏",
    "😔",
    "😟",
    "😕",
    "🙁",
    "☹️",
    "😣",
    "😖",
    "😫",
    "😩",
    "🥺",
    "😢",
    "😭",
    "😤",
    "😠",
    "😡",
    "🤬",
    "🤯",
    "😳",
    "🥵",
    "🥶",
    "😱",
    "😨",
    "😰",
    "😥",
    "😓",
    "🤗",
    "🤔",
    "🤭",
    "🤫",
    "🤥",
    "😶",
    "😐",
];

// 附件菜单项
const attachmentItems = computed(() => [
    [
        {
            label: "图片",
            icon: "i-heroicons-photo",
            click: () => selectFile("image/*"),
        },
        {
            label: "文件",
            icon: "i-heroicons-document",
            click: () => selectFile(""),
        },
        {
            label: "位置",
            icon: "i-heroicons-map-pin",
            click: () => sendLocation(),
        },
    ],
]);

// 方法
const addNewLine = () => {
    message.value += "\n";
};

const sendMessage = () => {
    if (!canSend.value) return;

    const content = message.value.trim();
    message.value = "";
    showEmojiPicker.value = false;

    emit("send", content);
};

// 使用 Nuxt UI 的 UInput[type=file] 代替原生创建
const fileAccept = ref("");
const fileInputRef = ref<HTMLInputElement | null>(null);
const onFileSelected = (e: Event) => {
    const file = (e.target as HTMLInputElement)?.files?.[0];
    if (file) emit("upload-file", file);
    // 重置，以便选择同一文件也能触发 change
    if (fileInputRef.value) fileInputRef.value.value = "";
};
const selectFile = (accept: string) => {
    fileAccept.value = accept;
    // 等下一帧确保属性已更新
    requestAnimationFrame(() => fileInputRef.value?.click());
};

const sendLocation = () => {
    emit("send-location");
};

const insertEmoji = (emoji: string) => {
    message.value += emoji;
    showEmojiPicker.value = false;
};

const handlePaste = (e: ClipboardEvent) => {
    const items = e.clipboardData?.items;
    if (!items) return;

    for (const item of items) {
        if (item.type.indexOf("image") !== -1) {
            const file = item.getAsFile();
            if (file) {
                e.preventDefault();
                emit("upload-file", file);
                break;
            }
        }
    }
};

const cancelReply = () => {
    emit("cancel-reply");
};

// 使用 UPopover 管理显示/隐藏，无需手动监听外部点击
</script>
