<script lang="ts">
	import { Wifi } from '@lucide/svelte';

	// 模拟 iOS 风格状态栏：真实时钟 + 装饰性信号/电池。

	let now = $state(new Date());

	$effect(() => {
		const timer = setInterval(() => {
			now = new Date();
		}, 1000);
		return () => clearInterval(timer);
	});

	const h = $derived(now.getHours().toString().padStart(2, '0'));
	const m = $derived(now.getMinutes().toString().padStart(2, '0'));

	const signalHeights = [4, 7, 10, 13];
	const batteryLevel = 0.72;
	const batteryLow = batteryLevel < 0.2;
</script>

<div
	class="flex-shrink-0 flex items-center justify-between px-5 select-none"
	style="height:44px;background-color:var(--neu-bg);"
>
	<!-- Time -->
	<span
		style="font-size:15px;font-weight:600;color:var(--neu-text);letter-spacing:-0.3px;font-variant-numeric:tabular-nums;"
	>
		{h}:{m}
	</span>

	<!-- System indicators -->
	<div class="flex items-center gap-[5px]">
		<div class="flex items-end gap-[2px]">
			{#each signalHeights as hgt, i}
				<div
					style="width:3px;height:{hgt}px;border-radius:1px;background-color:var(--neu-text);opacity:{i < 3 ? 1 : 0.28};"
				></div>
			{/each}
		</div>
		<Wifi style="width:14px;height:14px;color:var(--neu-text);" strokeWidth={2.5} />
		<!-- Battery -->
		<div style="display:flex;align-items:center;">
			<div
				style="width:23px;height:11px;border-radius:3px;border:1.5px solid var(--neu-text);position:relative;overflow:hidden;box-sizing:border-box;"
			>
				<div
					style="position:absolute;inset:1px;right:auto;width:calc({batteryLevel * 100}% - 2px);border-radius:1.5px;background-color:{batteryLow ? 'var(--destructive)' : 'var(--neu-text)'};"
				></div>
			</div>
			<div
				style="width:2px;height:5px;border-radius:0 1px 1px 0;background-color:var(--neu-text);opacity:0.45;margin-left:1px;"
			></div>
		</div>
	</div>
</div>
