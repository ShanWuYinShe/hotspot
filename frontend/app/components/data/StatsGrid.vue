<template>
    <div :class="gridClass">
        <UCard
            v-for="stat in stats"
            :key="stat.key"
            class="h-full flex flex-col justify-between"
            :variant="stat.onClick ? 'interactive' : 'soft'"
            @click="stat.onClick?.()"
        >
            <div class="flex items-start gap-4">
                <UAvatar
                    :icon="stat.icon"
                    size="lg"
                    :class="[
                        getUiColor(stat.color) === 'neutral'
                            ? 'bg-neutral-100 dark:bg-neutral-800 text-neutral-500 dark:text-neutral-400'
                            : `bg-${getUiColor(stat.color)}-50 dark:bg-${getUiColor(stat.color)}-950/30 text-${getUiColor(stat.color)}-500 dark:text-${getUiColor(stat.color)}-400`,
                    ]"
                />

                <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2">
                        <p
                            class="text-2xl font-bold text-neutral-900 dark:text-white"
                        >
                            {{ transformValue(stat.value) }}
                        </p>

                        <UBadge
                            v-if="stat.change !== undefined"
                            size="sm"
                            variant="soft"
                            :color="getChangeBadgeColor(stat.change)"
                        >
                            <UIcon
                                :name="
                                    stat.change >= 0
                                        ? 'i-lucide-trending-up'
                                        : 'i-lucide-trending-down'
                                "
                                :size="14"
                            />
                            <span class="ml-1">
                                {{ transformChange(stat.change) }}
                            </span>
                        </UBadge>
                    </div>

                    <p
                        class="text-sm font-medium text-neutral-500 dark:text-neutral-400"
                    >
                        {{ stat.label }}
                    </p>
                    <UBadge
                        v-if="stat.description"
                        color="neutral"
                        variant="soft"
                        size="xs"
                        class="mt-1 w-fit max-w-full truncate"
                    >
                        {{ stat.description }}
                    </UBadge>

                    <div v-if="stat.trend || stat.extraContent" class="mt-4">
                        <USeparator class="mb-4" />
                        <div
                            v-if="stat.trend"
                            class="flex items-center justify-between gap-3"
                        >
                            <UBadge color="neutral" variant="soft" size="xs"
                                >趋势</UBadge
                            >
                            <UBadge
                                variant="soft"
                                size="sm"
                                :color="getTrendBadgeColor(stat.trend)"
                            >
                                <UIcon
                                    :name="getTrendIcon(stat.trend)"
                                    :size="14"
                                />
                                <span class="ml-1">
                                    {{ getTrendText(stat.trend) }}
                                </span>
                            </UBadge>
                        </div>

                        <div v-if="stat.extraContent" class="mt-3">
                            <component
                                :is="stat.extraContent"
                                v-bind="stat.extraProps"
                            />
                        </div>
                    </div>
                </div>

                <div v-if="stat.actions?.length" class="shrink-0 -mt-1 -mr-1">
                    <UDropdownMenu :items="stat.actions">
                        <UButton
                            variant="ghost"
                            color="neutral"
                            size="sm"
                            icon="i-lucide-ellipsis"
                            @click.stop
                        />
                    </UDropdownMenu>
                </div>
            </div>
        </UCard>
    </div>
</template>

<script setup lang="ts">
import type { Component } from "vue";

export interface StatAction {
    label: string;
    icon?: string;
    click: () => void;
}

export interface StatItem {
    key: string;
    label: string;
    value: number | string;
    icon: string;
    color?:
        | "blue"
        | "green"
        | "orange"
        | "purple"
        | "red"
        | "yellow"
        | "indigo"
        | "pink"
        | "primary";
    description?: string;
    change?: number; // 变化百分比
    trend?: "up" | "down" | "stable";
    onClick?: () => void;
    actions?: StatAction[];
    extraContent?: Component; // 组件
    extraProps?: Record<string, unknown>;
}

interface Props {
    stats: StatItem[];
    columns?: 1 | 2 | 3 | 4 | 5 | 6;
    responsive?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
    columns: 4,
    responsive: true,
});
const { formatCount } = useFormatters();

// 转换数值显示
const transformValue = (value: number | string | undefined) => {
    if (value === undefined) return "0";
    if (typeof value === "string") return value;
    return formatCount(value);
};

// 转换变化百分比
const transformChange = (change: number | undefined) => {
    if (change === undefined) return "0%";
    const sign = change > 0 ? "+" : "";
    return `${sign}${change.toFixed(1)}%`;
};

// 计算网格类名
const gridClass = computed(() => {
    const baseClass = "grid gap-6";

    if (!props.responsive) {
        return `${baseClass} grid-cols-${props.columns}`;
    }

    // 响应式网格
    const responsiveClasses = {
        1: "grid-cols-1",
        2: "grid-cols-1 md:grid-cols-2",
        3: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3",
        4: "grid-cols-1 md:grid-cols-2 lg:grid-cols-4",
        5: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5",
        6: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6",
    };

    return `${baseClass} ${responsiveClasses[props.columns]}`;
});

const getUiColor = (
    color: StatItem["color"] = "primary",
):
    | "primary"
    | "secondary"
    | "success"
    | "info"
    | "warning"
    | "error"
    | "neutral" => {
    switch (color) {
        case "green":
            return "success";
        case "orange":
        case "yellow":
            return "warning";
        case "red":
            return "error";
        case "blue":
            return "info";
        case "purple":
        case "indigo":
        case "pink":
            return "secondary";
        case "primary":
        default:
            return "primary";
    }
};

const getChangeBadgeColor = (
    change: number,
): "success" | "error" | "neutral" => {
    if (change > 0) return "success";
    if (change < 0) return "error";
    return "neutral";
};

const getTrendIcon = (trend: string) => {
    switch (trend) {
        case "up":
            return "i-lucide-trending-up";
        case "down":
            return "i-lucide-trending-down";
        default:
            return "i-lucide-minus";
    }
};

const getTrendBadgeColor = (trend: string) => {
    switch (trend) {
        case "up":
            return "success";
        case "down":
            return "error";
        default:
            return "neutral";
    }
};

const getTrendText = (trend: string) => {
    switch (trend) {
        case "up":
            return "上升";
        case "down":
            return "下降";
        default:
            return "稳定";
    }
};
</script>
