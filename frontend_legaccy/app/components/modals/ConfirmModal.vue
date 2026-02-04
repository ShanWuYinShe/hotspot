<template>
    <UModal :title="title" :dismissible="false" size="sm">
        <template #body>
            <p class="text-base text-neutral-600 dark:text-neutral-300">{{ message }}</p>
        </template>

        <template #footer>
            <div class="flex justify-end gap-3 w-full">
                <UButton
                    v-if="cancelText"
                    variant="ghost"
                    color="neutral"
                    :label="cancelText"
                    @click="emit('close', false)"
                />
                <UButton
                    :color="confirmColor"
                    :label="confirmText"
                    @click="emit('close', true)"
                />
            </div>
        </template>
    </UModal>
</template>

<script setup lang="ts">
interface Props {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    confirmColor?: "primary" | "error" | "warning" | "success" | "neutral";
}

withDefaults(defineProps<Props>(), {
    confirmText: "确认",
    cancelText: "取消",
    confirmColor: "primary",
});

const emit = defineEmits<{
    close: [value: boolean];
}>();
</script>
