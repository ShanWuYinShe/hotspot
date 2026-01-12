<template>
    <UModal
        :title="isSetup ? '设置设备PIN' : '输入设备PIN'"
        :dismissible="false"
    >
        <template #body>
            <UAlert
                class="mb-6"
                color="neutral"
                variant="soft"
                icon="i-lucide-lock-keyhole"
                :title="
                    isSetup
                        ? '请设置4位数字PIN以保护账户切换功能'
                        : '请输入4位数字PIN以解锁账户切换功能'
                "
            />

            <section class="flex justify-center gap-4 mb-6">
                <template v-for="(digit, index) in 4" :key="index">
                    <!-- 使用 UInput 替换原生 input，保持样式一致性 -->
                    <div class="w-12 h-12">
                        <UInput
                        :ref="(el) => (inputRefs[index] = el)"
                        v-model="pinValue[index]"
                        type="text"
                        inputmode="numeric"
                        maxlength="1"
                            variant="soft"
                            size="xl"
                        placeholder="●"
                            class="h-full"
                        @input="handleInput($event, index)"
                        @keydown="handleKeydown($event, index)"
                        @focus="handleFocus(index)"
                    />
                    </div>
                </template>
            </section>

            <UAlert
                v-if="hasError"
                color="error"
                variant="subtle"
                :title="errorMessage"
                class="mb-4"
            />

            <p v-if="isSetup" class="text-xs text-center">
                <UBadge color="neutral" variant="soft" size="xs">
                    建议使用好记的数字组合，如生日或纪念日
                </UBadge>
            </p>
        </template>

        <template #footer>
            <UButton
                variant="ghost"
                color="neutral"
                :disabled="isProcessing"
                @click="handleCancel"
            >
                取消
            </UButton>
            <UButton
                :disabled="!isComplete || isProcessing"
                :loading="isProcessing"
                @click="handleConfirm"
            >
                {{ isSetup ? "设置" : "确认" }}
            </UButton>
        </template>
    </UModal>
</template>

<script setup lang="ts">
interface Props {
    isSetup?: boolean;
}

interface Emits {
    (e: "confirm", pin: string): void;
    (e: "cancel"): void;
    (e: "close", value?: string): void;
}

const props = withDefaults(defineProps<Props>(), {
    isSetup: false,
});

const emit = defineEmits<Emits>();

const pinValue = ref<string[]>(["", "", "", ""]);
// inputRefs will hold the UInput component instance or element
const inputRefs = ref<any[]>([]);
const hasError = ref(false);
const errorMessage = ref("");
const isProcessing = ref(false);

const isComplete = computed(
    () =>
        pinValue.value.length === 4 &&
        pinValue.value.every((digit) => /^\d$/.test(digit)),
);

const pinString = computed(() => pinValue.value.join(""));

onMounted(() => {
    resetForm();
    // Focus first input on mount
    nextTick(() => {
        focusInput(0);
    });
});

watch(
    pinValue,
    () => {
        if (hasError.value) {
            hasError.value = false;
            errorMessage.value = "";
        }
    },
    { deep: true },
);

// Helper to focus input (handles UInput component wrapper)
const focusInput = (index: number) => {
    const el = inputRefs.value[index];
    if (el?.input) {
        el.input.focus();
    } else if (el?.focus) {
        el.focus();
    }
};

// Helper to select input content
const selectInput = (index: number) => {
    const el = inputRefs.value[index];
    if (el?.input) {
        el.input.select();
    } else if (el?.select) {
        el.select();
    }
};

const handleInput = (event: Event, index: number) => {
    // With UInput v-model, event might be the value string if @update:modelValue,
    // but @input emits Event. However UInput wraps standard input.
    // If using v-model, we check pinValue directly or the event.
    // Let's rely on v-model update which happens before or synced.
    // But we need to check the value typed.

    // Allow single digit logic
    const val = pinValue.value[index];

    // Ensure only numbers
    if (val && !/^\d*$/.test(val)) {
        pinValue.value[index] = "";
        return;
    }

    // Handle auto-focus next
    if (val && index < 3) {
        nextTick(() => {
            focusInput(index + 1);
        });
    }
};

const handleKeydown = (event: KeyboardEvent, index: number) => {
    if (event.key === "Backspace") {
        if (!pinValue.value[index] && index > 0) {
            // If empty and backspace, move to prev and clear
            pinValue.value[index - 1] = "";
            nextTick(() => {
                focusInput(index - 1);
            });
        }
    }
};

const handleFocus = (index: number) => {
    // Select text on focus
    selectInput(index);
};

function resetForm() {
    pinValue.value = ["", "", "", ""];
    hasError.value = false;
    errorMessage.value = "";
    isProcessing.value = false;
    nextTick(() => {
        focusInput(0);
    });
}

async function handleConfirm() {
    if (!isComplete.value || isProcessing.value) return;

    isProcessing.value = true;

    try {
        emit("confirm", pinString.value);
        emit("close", pinString.value);
    } catch (error) {
        hasError.value = true;
        errorMessage.value = "处理失败，请重试";
        isProcessing.value = false;
    }
}

function handleCancel() {
    if (isProcessing.value) return;
    emit("cancel");
    emit("close");
}

function showError(message: string) {
    hasError.value = true;
    errorMessage.value = message;
    isProcessing.value = false;
    resetForm();
}

defineExpose({ showError, resetForm });
</script>
