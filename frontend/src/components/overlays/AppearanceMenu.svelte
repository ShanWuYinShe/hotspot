<script lang="ts">
	import { Check, Moon, Palette, Sun, Database } from '@lucide/svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ListItem from '@/components/list/ListItem.svelte';
	import { theme, toggleDark, setTheme, THEME_OPTIONS } from '@/stores/theme.svelte';
	import * as Popover from '@/components/ui/popover';

	// 唯一的外观/模式控制。选择面板保持打开，仅显式完成、外部点击或 Escape 关闭。
	// 弹层行为现由 bits-ui Popover 提供。

	let { class: className = '' }: { class?: string } = $props();

	// 主题选项来自 theme store 的 THEME_OPTIONS（与设置页共用，避免两份配置漂移）

	let open = $state(false);
	let rootEl = $state<HTMLElement | null>(null);

	const currentTheme = $derived(THEME_OPTIONS.find((item) => item.id === theme.theme)?.label ?? '星际');
	const modeLabel = $derived(theme.isDark ? '深色' : '浅色');

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
		/>
	</div>

	<Popover.Content customAnchor={rootEl} class="w-57.5 rounded-md p-2.5" role="dialog" aria-label="外观设置">
			<div class="px-2 pb-2">
				<p class="eyebrow">外观</p>
				<p class="mt-1 text-xs text-text-secondary">
					当前：{currentTheme} · {modeLabel}
				</p>
			</div>

			<section
				aria-labelledby="theme-choice-label"
				class="rounded-md ring-1 ring-border-default surface-tinted p-2"
			>
				<div class="flex items-center justify-between px-2 pb-2 pt-1">
					<p id="theme-choice-label" class="eyebrow">主题</p>
					<span class="text-xxs text-text-secondary">选择一种基调</span>
				</div>
				<div class="grid grid-cols-2 gap-1.5" role="group" aria-label="选择主题">
					{#each THEME_OPTIONS as item}
						{@const selected = theme.theme === item.id}
						<button
							type="button"
							onclick={() => setTheme(item.id)}
							aria-pressed={selected}
							class="rounded-sm p-2.5 text-left outline-none transition-colors focus-visible:ring-2 focus-visible:ring-accent-default {selected
								? 'bg-accent-soft text-accent-default hover:bg-interactive-active'
								: 'surface-ghost text-text-secondary'}"
						>
							<span
								class="mb-2 block h-5 w-5 rounded-full ring-3"
								style="background:{item.swatch};--tw-ring-color:{selected ? 'var(--cst-accent-soft)' : 'transparent'};"
							></span>
							<span class="block text-sm font-medium text-text-primary" >{item.label}</span>
							<span class="mt-0.5 block text-xxs text-text-secondary">{item.note}</span>
						</button>
					{/each}
				</div>
			</section>

			<button
				type="button"
				onclick={toggleDark}
				aria-pressed={theme.isDark}
				class="hover-btn surface-ghost mt-2 flex w-full items-center justify-between rounded-sm px-3 py-2.5 text-left outline-none focus-visible:ring-2 focus-visible:ring-accent-default text-text-primary"
				
			>
				<span class="flex items-center gap-2">
					<span class="flex h-6.5 w-6.5 items-center justify-center rounded-full bg-accent-soft">
						{#if theme.isDark}
							<Sun size={13} color="var(--cst-accent-default)" />
						{:else}
							<Moon size={13} color="var(--cst-accent-default)" />
						{/if}
					</span>
					<span>
						<span class="block text-sm font-medium">{theme.isDark ? '深色模式' : '浅色模式'}</span>
						<span class="block text-xxs text-text-secondary">点击切换</span>
					</span>
				</span>
				<span class="relative h-5 w-8.5 overflow-hidden rounded-full" style="background:{theme.isDark ? 'var(--cst-accent-default)' : 'var(--cst-border-strong)'};">
					<span
						class="absolute top-0.75 h-3.5 w-3.5 rounded-full"
						style="background:var(--cst-surface-control-thumb);box-shadow:var(--cst-elevation-control-thumb);left:{theme.isDark ? 17 : 3}px;transition:left var(--cst-motion-standard) var(--cst-ease-bounce);"
					></span>
				</span>
			</button>

			{#if import.meta.env.DEV}
				<button
					type="button"
					onclick={handleMockToggle}
					class="hover-btn mt-2 flex w-full items-center justify-between rounded-sm ring-1 ring-border-default px-3 py-2.5 text-left outline-none transition-colors hover:bg-accent-soft focus-visible:ring-2 focus-visible:ring-accent-default text-text-primary"
					
				>
					<span class="flex items-center gap-2">
						<span class="flex h-6.5 w-6.5 items-center justify-center rounded-full bg-accent-soft">
							<Database size={13} color="var(--cst-accent-default)" />
						</span>
						<span>
							<span class="block text-sm font-medium">API 数据来源</span>
							<span class="block text-xxs text-text-secondary">{mockEnabled ? '模拟数据' : '真实后端'}</span>
						</span>
					</span>
					<span class="relative h-5 w-8.5 overflow-hidden rounded-full" style="background:{mockEnabled ? 'var(--cst-accent-default)' : 'var(--cst-border-strong)'};">
						<span
							class="absolute top-0.75 h-3.5 w-3.5 rounded-full bg-content-inverse shadow-[var(--cst-elevation-control-thumb)]"
							style="left:{mockEnabled ? 17 : 3}px;transition:left var(--cst-motion-standard) var(--cst-ease-bounce);"
						></span>
					</span>
				</button>

				<div class="mt-2 rounded-sm ring-1 ring-border-default px-3 py-2.5 text-text-primary" >
					<span class="block text-sm font-medium">模拟登录态</span>
					<span class="mb-2 block text-xxs text-text-secondary">影响 login/register/guest/reset 的返回</span>
					<ListContainer>
						{#each AUTH_SCENARIOS as s}
							<ListItem
								pressed={scenario === s.id}
								onClick={() => void applyScenario(s.id)}
								ariaLabel={`切换登录态：${s.label}`}
								class="flex items-center justify-between px-2 py-1.5"
							>
								<span class="{scenario === s.id ? 'text-text-on-accent' : 'text-text-primary'} text-xs">{s.label}</span>
								{#if scenario === s.id}
									<Check size={12}  class="text-text-on-accent" />
								{/if}
							</ListItem>
						{/each}
					</ListContainer>
					<button
						type="button"
						onclick={() => void resetLoggedOut()}
						class="mt-2 w-full rounded-sm border border-border-default px-2 py-1.5 text-xs focus-ring transition-colors hover:bg-accent-soft text-text-secondary"
						
					>
						重置为未登录（匿名浏览）
					</button>
				</div>
			{/if}
	</Popover.Content>
</Popover.Root>
