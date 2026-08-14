<script lang="ts">
	// 多图选择器：隐藏 input[file multiple] + 缩略图网格 + 单图删除 + 上限校验。
	// 纯 UI（选择/预览/删除），上传由调用方在提交时用 uploadMediaFiles 执行。
	import { ImagePlus, X } from '@lucide/svelte';
	import { showToast } from '@/stores/toast.svelte';

	let {
		files,
		onFilesChange,
		max = 9
	}: {
		files: File[];
		onFilesChange: (files: File[]) => void;
		max?: number;
	} = $props();

	let inputEl = $state<HTMLInputElement | null>(null);

	// 缩略图预览 URL：随 files 变化重建，cleanup 时 revoke 避免泄漏
	let previews = $state<string[]>([]);
	$effect(() => {
		const next = files.map((f) => URL.createObjectURL(f));
		previews = next;
		return () => next.forEach((url) => URL.revokeObjectURL(url));
	});

	const handleSelect = (e: Event) => {
		const target = e.currentTarget as HTMLInputElement;
		const picked = Array.from(target.files ?? []);
		target.value = ''; // 允许重复选同一文件
		if (picked.length === 0) return;
		const remaining = max - files.length;
		if (picked.length > remaining) {
			showToast(`最多选择 ${max} 张图片`, 'danger');
		}
		const accepted = picked.slice(0, Math.max(0, remaining));
		if (accepted.length === 0) return;
		onFilesChange([...files, ...accepted]);
	};

	const removeAt = (index: number) => {
		onFilesChange(files.filter((_, i) => i !== index));
	};
</script>

<div class="flex flex-wrap gap-2">
	{#each previews as src, i (i)}
		<div class="relative h-20 w-20 overflow-hidden rounded-md ring-1 ring-border-default">
			<img src={src} alt={`照片 ${i + 1}`} class="h-full w-full object-cover" />
			<button
				type="button"
				onclick={() => removeAt(i)}
				class="hit-slop absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-surface-media-scrim-strong text-content-inverse outline-none backdrop-blur-sm hover-btn scale-btn focus-visible:ring-2 focus-visible:ring-content-inverse"
				aria-label={`移除照片 ${i + 1}`}
			>
				<X size={12} strokeWidth={2} />
			</button>
		</div>
	{/each}
	{#if files.length < max}
		<button
			type="button"
			onclick={() => inputEl?.click()}
			class="flex h-20 w-20 flex-col items-center justify-center gap-1 rounded-md border border-dashed border-border-strong text-text-secondary outline-none transition-colors hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default"
			aria-label="添加照片"
		>
			<ImagePlus size={18} strokeWidth={1.5} />
			<span class="text-xxs tabular-nums">{files.length}/{max}</span>
		</button>
	{/if}
	<input
		bind:this={inputEl}
		type="file"
		accept="image/*"
		multiple
		class="hidden"
		onchange={handleSelect}
	/>
</div>
