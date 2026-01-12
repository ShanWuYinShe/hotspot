<template>
    <div class="flex w-full items-center gap-2 p-3">
        <!-- Native file input (hidden) -->
        <input
            ref="fileInput"
            type="file"
            class="hidden"
            :accept="CHAT_CONSTANTS.ACCEPTED_ATTACHMENTS"
            @change="onNativeFileChange"
        />

        <!-- Attachment -->
        <UDropdownMenu :items="attachmentMenuItems">
            <UButton
                variant="ghost"
                color="neutral"
                icon="i-lucide-plus-circle"
                size="md"
                square
                :disabled="isLoading"
                aria-label="添加附件"
            />
        </UDropdownMenu>

        <!-- Composer -->
        <div class="relative flex-1 min-w-0">
            <UTextarea
                v-model="messageInput"
                variant="outline"
                placeholder="发送消息…"
                :rows="1"
                :maxrows="5"
                autoresize
                :disabled="isLoading"
                @keydown="onKeydown"
            />

            <!-- Emoji trigger (inside textarea, centered on same axis as other controls) -->
            <div class="absolute inset-y-0 right-2 flex items-center">
                <UPopover v-model:open="showEmoji">
                    <UButton
                        variant="ghost"
                        color="neutral"
                        icon="i-lucide-smile"
                        size="md"
                        square
                        :disabled="isLoading"
                        aria-label="选择表情"
                    />
                    <template #content>
                        <div
                            class="grid w-72 grid-cols-8 gap-1 p-2 max-h-64 overflow-y-auto"
                        >
                            <UButton
                                v-for="emoji in CHAT_CONSTANTS.COMMON_EMOJIS"
                                :key="emoji"
                                variant="ghost"
                                color="neutral"
                                size="md"
                                square
                                :label="emoji"
                                :aria-label="`插入表情 ${emoji}`"
                                class="text-lg"
                                @click="insertEmoji(emoji)"
                            />
                        </div>
                    </template>
                </UPopover>
            </div>
        </div>

        <!-- Send -->
        <UButton
            color="primary"
            variant="solid"
            icon="i-lucide-send-horizontal"
            size="md"
            square
            :disabled="!canSubmit"
            :loading="isLoading"
            aria-label="发送消息"
            @click="sendMessage"
        />
    </div>
</template>

<script setup lang="ts">
import type { DropdownMenuItem } from "@nuxt/ui";
import { CHAT_CONSTANTS } from "~/config/defaults";

interface Props {
    loading?: boolean;
}
const props = withDefaults(defineProps<Props>(), { loading: false });

interface Emits {
    "send": [payload: { content: string; type: "text" | "image" | "file" | "location"; file?: File }];
    "typing": [isTyping: boolean];
}
const emit = defineEmits<Emits>();

const messageInput = ref("");
const showEmoji = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

const isLoading = computed(() => props.loading);
const trimmedMessage = computed(() => messageInput.value.trim());
const canSubmit = computed(
    () => trimmedMessage.value.length > 0 && !isLoading.value,
);

const triggerFileSelect = () => {
    fileInput.value?.click();
};

const attachmentMenuItems: DropdownMenuItem[][] = [
    [
        {
            label: "图片",
            icon: "i-lucide-image",
            onSelect: triggerFileSelect,
        },
        {
            label: "文件",
            icon: "i-lucide-file",
            onSelect: triggerFileSelect,
        },
    ],
    [
        {
            label: "位置",
            icon: "i-lucide-map-pin",
            onSelect: () => emit("send", { content: "[位置]", type: "location" }),
        },
    ],
];

watch(
    () => messageInput.value,
    (v) => emit("typing", v.length > 0),
);

function sendMessage() {
    if (!canSubmit.value) return;
    emit("send", { content: trimmedMessage.value, type: "text" });
    messageInput.value = "";
}

function insertEmoji(emoji: string) {
    messageInput.value += emoji;
    showEmoji.value = false;
}

function onNativeFileChange(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    const type = file.type?.startsWith("image/") ? "image" : "file";
    emit("send", { content: "[文件]", type, file });

    // Reset input so same file can be selected again
    input.value = "";
}

function onKeydown(e: KeyboardEvent) {
    if (e.key !== "Enter") return;
    if (e.shiftKey) return;
    e.preventDefault();
    sendMessage();
}
</script>
