<script lang="ts">
	import { Check, Palette, Database } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import { theme, setColorScheme, setTheme, THEME_OPTIONS, COLOR_SCHEME_OPTIONS, type ColorSchemeMode } from '@/stores/theme.svelte';
	import * as Popover from '@/components/ui/popover';

	// 唯一的外观/模式控制。选择面板保持打开，仅显式完成、外部点击或 Escape 关闭。
	// 弹层行为现由 bits-ui Popover 提供。

	let { class: className = '' }: { class?: string } = $props();

	// 主题选项来自 theme store 的 THEME_OPTIONS（与设置页共用，避免两份配置漂移）

	let open = $state(false);
	let rootEl = $state<HTMLElement | null>(null);

	const currentTheme = $derived(THEME_OPTIONS.find((item) => item.id === theme.theme)?.label ?? '星际');
	const modeLabel = $derived(
		theme.colorScheme === 'auto' ? '跟随系统' : theme.colorScheme === 'dark' ? '深色' : '浅色'
	);

	// ── Dev API source toggle（仅 dev）───────────────────────────
	let mockEnabled = $state(true);

	$effect(() => {
		if (!import.meta.env.DEV) return;
		let unsub: (() => void) | undefined;
		void import('@/mocks/toggle').then(({ isMockEnabled, subscribe }) => {
			mockEnabled = isMockEnabled();
			unsub = subscribe(() => {
				mockEnabled = isMockEnabled();
			});
		});
		return () => unsub?.();
	});

	const handleMockToggle = async () => {
		const { toggleMock } = await import('@/mocks/toggle');
		await toggleMock();
	};

	// ── Dev auth scenario（仅 dev）──────────────────────────────
	const AUTH_SCENARIOS = [
		{ id: 'ok', label: '正常登录' },
		{ id: 'invalid-creds', label: '账号/密码错误 (10100)' },
		{ id: 'banned', label: '账号封禁 (10201)' },
		{ id: 'param-error', label: '参数错误 (10000)' },
		{ id: 'rate-limit', label: '限流 (10500)' }
	] as const;

	let scenario = $state<string>('ok');

	$effect(() => {
		if (!import.meta.env.DEV) return;
		void import('@/mocks/data').then(({ getMockAuthScenario }) => {
			scenario = getMockAuthScenario();
		});
	});

	const applyScenario = async (next: string) => {
		const { setMockAuthScenario } = await import('@/mocks/data');
		setMockAuthScenario(next as any);
		scenario = next;
	};

	const resetLoggedOut = async () => {
		const { setMockLoggedIn } = await import('@/mocks/data');
		// Clear the mock session cookies + state, then reload so auth store
		// re-checks status and renders the anonymous UI.
		document.cookie = 'access_token=; Path=/; Max-Age=0';
		document.cookie = 'refresh_token=; Path=/; Max-Age=0';
		setMockLoggedIn(false);
		window.location.reload();
	};
</script>

<Popover.Root bind:open>
	<div bind:this={rootEl} class="relative {className}">
		<ActionIconButton
			icon={Palette}
			label="外观设置"
			onClick={() => (open = !open)}
			variant="ghost"
			size={34}
			class="header-action"
		/>
	</div>

	<Popover.Content customAnchor={rootEl} class="w-76 rounded-xl p-3 shadow-raised-lg border border-border-default bg-surface-primary" role="dialog" aria-label="外观设置">
			<div class="px-1 pb-2.5">
				<p class="eyebrow text-text-secondary">外观</p>
				<p class="mt-0.5 text-xs text-text-secondary">
					当前：<span class="font-medium text-text-primary">{currentTheme}</span> · <span class="font-medium text-text-primary">{modeLabel}</span>
				</p>
			</div>

			<section
				aria-labelledby="theme-choice-label"
				class="rounded-lg ring-1 ring-border-default surface-tinted p-2.5"
			>
				<div class="flex items-center justify-between pb-2 pt-0.5 px-0.5">
					<p id="theme-choice-label" class="eyebrow">主题</p>
					<span class="text-xxs text-text-secondary">选择一种基调</span>
				</div>
				<div class="grid grid-cols-2 gap-2" role="group" aria-label="选择主题">
					{#each THEME_OPTIONS as item}
						{@const selected = theme.theme === item.id}
						<button
							type="button"
							onclick={() => setTheme(item.id)}
							aria-pressed={selected}
							class="rounded-md border p-2.5 text-left outline-none transition-all duration-standard focus-visible:ring-2 focus-visible:ring-accent-default {selected
								? 'border-accent-border bg-accent-soft text-accent-default shadow-raised-xs'
								: 'border-border-default bg-surface-primary text-text-secondary hover:bg-interactive-hover hover:text-text-primary'}"
						>
							<div class="flex items-center justify-between mb-2">
								<span
									class="block h-5 w-5 rounded-full ring-2 ring-offset-1 ring-offset-surface-primary"
									style="background:{item.swatch};--tw-ring-color:{selected ? 'var(--cst-accent-default)' : 'var(--cst-border-default)'};"
								></span>
								{#if selected}
									<Check size={13} class="text-accent-default" />
								{/if}
							</div>
							<span class="block text-sm font-medium text-text-primary">{item.label}</span>
							<span class="mt-0.5 block text-xxs text-text-secondary">{item.note}</span>
						</button>
					{/each}
				</div>
			</section>

			<!-- 明暗三档（跟随系统/浅色/深色）：接入 TabBar segmented 形态，等宽底轨无粘连 -->
			<section
				aria-labelledby="color-scheme-label"
				class="rounded-lg ring-1 ring-border-default surface-tinted p-2.5 mt-2.5"
			>
				<div class="flex items-center justify-between pb-2 pt-0.5 px-0.5">
					<p id="color-scheme-label" class="eyebrow">明暗</p>
					<span class="text-xxs text-text-secondary">当前：{modeLabel}</span>
				</div>
				<TabBar
					variant="segmented"
					items={COLOR_SCHEME_OPTIONS.map((o) => ({ id: o.id, label: o.label }))}
					value={theme.colorScheme}
					onChange={(id) => setColorScheme(id as ColorSchemeMode)}
				/>
			</section>

			{#if import.meta.env.DEV}
				<button
					type="button"
					onclick={handleMockToggle}
					class="hover-btn mt-2.5 flex w-full items-center justify-between rounded-lg ring-1 ring-border-default bg-surface-primary p-2.5 text-left outline-none transition-colors hover:bg-interactive-hover focus-visible:ring-2 focus-visible:ring-accent-default text-text-primary"
				>
					<span class="flex items-center gap-2.5">
						<span class="flex h-7 w-7 items-center justify-center rounded-full bg-accent-soft shrink-0">
							<Database size={14} color="var(--cst-accent-default)" />
						</span>
						<span>
							<span class="block text-xs font-medium">API 数据来源</span>
							<span class="block text-xxs text-text-secondary">{mockEnabled ? '模拟数据' : '真实后端'}</span>
						</span>
					</span>
					<span class="relative h-5 w-8.5 overflow-hidden rounded-full {mockEnabled ? 'bg-accent-default' : 'bg-border-strong'}">
						<span
							class="absolute top-0.75 h-3.5 w-3.5 rounded-full bg-content-inverse shadow-[var(--cst-elevation-control-thumb)]"
							style="left:{mockEnabled ? 17 : 3}px;transition:left var(--cst-motion-standard) var(--cst-ease-bounce);"
						></span>
					</span>
				</button>

				<div class="mt-2.5 rounded-lg ring-1 ring-border-default bg-surface-primary p-2.5 text-text-primary">
					<div class="px-0.5 pb-2">
						<span class="block text-xs font-medium">模拟登录态</span>
						<span class="mt-0.5 block text-xxs text-text-secondary">影响 login/register/guest/reset 的返回</span>
					</div>
					<div class="space-y-1">
						{#each AUTH_SCENARIOS as s}
							{@const isCur = scenario === s.id}
							<button
								type="button"
								onclick={() => void applyScenario(s.id)}
								aria-pressed={isCur}
								class="flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-xs transition-colors duration-fast outline-none focus-visible:ring-2 focus-visible:ring-accent-default {isCur
									? 'bg-accent-soft font-medium text-accent-default'
									: 'text-text-primary hover:bg-interactive-hover'}"
							>
								<span class="truncate">{s.label}</span>
								{#if isCur}
									<Check size={13} class="shrink-0 text-accent-default" />
								{/if}
							</button>
						{/each}
					</div>
					<button
						type="button"
						onclick={() => void resetLoggedOut()}
						class="mt-2.5 w-full rounded-md border border-border-default bg-surface-subtle py-1.5 text-xs font-medium text-text-secondary transition-colors hover:bg-interactive-hover hover:text-text-primary focus-ring"
					>
						重置为未登录（匿名浏览）
					</button>
				</div>
			{/if}
	</Popover.Content>
</Popover.Root>
