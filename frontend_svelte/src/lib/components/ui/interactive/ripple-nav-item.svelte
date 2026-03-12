<script lang="ts">
    import { rippleFeedback } from "$lib/attachments";
    import { cn } from "$lib/utils";
    import type { Snippet } from "svelte";
    import type { HTMLAnchorAttributes } from "svelte/elements";

    interface Props extends HTMLAnchorAttributes {
        active?: boolean;
        children?: Snippet;
    }

    let {
        active = false,
        class: className,
        children,
        ...restProps
    }: Props = $props();
</script>

<a
    class={cn(
        "group relative flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium",
        "transition-all duration-200 overflow-hidden",
        "min-h-11",
        // 移动端触摸目标优化
        "md:min-h-0",
        active
            ? "bg-primary text-primary-foreground shadow-md shadow-primary/25 ring-1 ring-primary/30"
            : "text-muted-foreground hover:bg-accent hover:text-foreground",
        className,
    )}
    {@attach rippleFeedback}
    {...restProps}
>
    {#if children}
        {@render children()}
    {/if}
</a>
