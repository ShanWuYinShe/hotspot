<script lang="ts">
	import { Wifi } from '@lucide/svelte';

	// 模拟 iOS 风格状态栏：真实时钟 + 装饰性信号/电池。
	// 注：信号/电池图标的 1-3px 微圆角属于模拟系统 chrome 的几何微调，
	// 不在 shape token 刻度内（登记例外，非主题风格值）。

	let now = $state(new Date());

	$effect(() => {
		// 时钟显示分钟精度，按分钟边界对齐刷新（+50ms 容差），
		// 避免每秒 set 触发 60 倍的无谓重渲染
		let timer: ReturnType<typeof setTimeout>;
		const tick = () => {
			now = new Date();
			timer = setTimeout(tick, 60_000 - (Date.now() % 60_000) + 50);
		};
		tick();
		return () => clearTimeout(timer);
	});

	const h = $derived(now.getHours().toString().padStart(2, '0'));
	const m = $derived(now.getMinutes().toString().padStart(2, '0'));

	const signalHeights = [4, 7, 10, 13];
	const batteryLevel = 0.72;
</script>

<div
	class="shrink-0 flex items-center justify-between px-5 select-none"
	style="height:44px;background-color:var(--cst-surface-primary);"
>
	<!-- Time -->
		<span
			class="text-md font-semibold tracking-tight text-text-primary"
			style="font-variant-numeric:tabular-nums;"
		>
			{h}:{m}
		</span>

	<!-- System indicators -->
	<div class="flex items-center gap-1.25">
		<div class="flex items-end gap-0.5">
			{#each signalHeights as hgt, i}
				<div
					class="{i < 3 ? 'opacity-full' : 'opacity-faint'}"
					style="width:3px;height:{hgt}px;border-radius:1px;background-color:var(--cst-text-primary);"
				></div>
			{/each}
		</div>
		<Wifi class="size-3.5 text-text-primary"   strokeWidth={2.5} />
		<!-- Battery -->
		<div style="display:flex;align-items:center;">
			<div
				style="width:23px;height:11px;border-radius:3px;border:var(--cst-border-width-regular) solid var(--cst-text-primary);position:relative;overflow:hidden;box-sizing:border-box;"
			>
				<div
					style="position:absolute;inset:1px;right:auto;width:calc({batteryLevel * 100}% - 2px);border-radius:1.5px;background-color:var(--cst-text-primary);"
				></div>
			</div>
			<div
				class="opacity-muted"
				style="width:2px;height:5px;border-radius:0 1px 1px 0;background-color:var(--cst-text-primary);margin-left:1px;"
			></div>
		</div>
	</div>
</div>
