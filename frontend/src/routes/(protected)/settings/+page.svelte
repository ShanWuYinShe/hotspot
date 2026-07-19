<script lang="ts">
  import { authStore } from '$lib/stores/auth.svelte';
  import { authService } from '$lib/api/services/auth';
  import { THEME_FAMILY_OPTIONS, themeStore, type ThemeFamily } from '$lib/stores/theme.svelte';
  import { toast } from '$lib/stores/toast.svelte';
  import { platformStore } from '$lib/stores/platform.svelte';
  import { formatHandle } from '$lib/utils/format';
  import { t } from '$lib/i18n';
  import ThemeToggle from '$lib/components/common/theme-toggle.svelte';
  import { Switch } from '$lib/components/ui/switch';
  import Avatar from '$lib/components/common/avatar.svelte';
  import User from 'lucide-svelte/icons/user';
  import Bell from 'lucide-svelte/icons/bell';
  import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';
  import Shield from 'lucide-svelte/icons/shield';
  import Lock from 'lucide-svelte/icons/lock';
  import LogOut from 'lucide-svelte/icons/log-out';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Share2 from 'lucide-svelte/icons/share-2';
  import MonitorSmartphone from 'lucide-svelte/icons/monitor-smartphone';
  import EyeOff from 'lucide-svelte/icons/eye-off';
  import Flag from 'lucide-svelte/icons/flag';
  import Palette from 'lucide-svelte/icons/palette';
  import Check from 'lucide-svelte/icons/check';
  import BackButton from '$lib/components/common/back-button.svelte';
  import Moon from 'lucide-svelte/icons/moon';

  const PREVIEW_ACCENT: Record<ThemeFamily, string> = {
    mist: '#6366f1',
    grove: '#3D7A55'
  };

  const THEME_NAMES = $derived<Record<ThemeFamily, string>>({
    mist: t('settings.themeNameMist'),
    grove: t('settings.themeNameGrove')
  });

  const THEME_DESC = $derived<Record<ThemeFamily, string>>({
    mist: t('settings.themeDescMist'),
    grove: t('settings.themeDescGrove')
  });

  const settingsSections = $derived([
    {
      id: 'account',
      title: t('settings.sectionAccount'),
      icon: User,
      items: [
        {
          icon: User,
          label: t('settings.accountProfile'),
          href: '/profile',
          description: t('settings.accountProfileDesc')
        },
        {
          icon: Lock,
          label: t('settings.changePassword'),
          href: '/settings/password',
          description: t('settings.changePasswordDesc')
        },
        {
          icon: MonitorSmartphone,
          label: t('settings.sessionsManagement'),
          href: '/settings/sessions',
          description: t('settings.sessionsManagementDesc')
        }
      ]
    },
    {
      id: 'privacy',
      title: t('settings.sectionPrivacy'),
      icon: Shield,
      items: [
        {
          icon: EyeOff,
          label: t('settings.privacySettings'),
          href: '/settings/privacy',
          description: t('settings.privacySettingsDesc')
        }
      ]
    },
    {
      id: 'notifications',
      title: t('settings.sectionNotifications'),
      icon: Bell,
      items: [
        {
          icon: Bell,
          label: t('settings.notificationSettings'),
          href: '/settings/notifications',
          description: t('settings.notificationSettingsDesc')
        }
      ]
    },
    {
      id: 'records',
      title: t('settings.sectionRecordsAndSharing'),
      icon: Share2,
      items: [
        {
          icon: Share2,
          label: t('settings.myShares'),
          href: '/settings/shares',
          description: t('settings.mySharesDesc')
        },
        {
          icon: Flag,
          label: t('settings.myReports'),
          href: '/settings/reports',
          description: t('settings.myReportsDesc')
        }
      ]
    }
  ]);

  const notificationItems = $derived([
    { key: 'like', label: t('settings.notificationLike') },
    { key: 'follow', label: t('settings.notificationFollow') },
    { key: 'comment', label: t('settings.notificationComment') },
    { key: 'nearby', label: t('settings.notificationNearby') }
  ]);

  let notifications = $state<Record<string, boolean>>({
    like: true,
    follow: true,
    comment: true,
    nearby: false
  });

  let activeSection = $state<string>('appearance');

  const currentThemeFamily = $derived(themeStore.themeFamily);

  async function handleLogout() {
    // 先调后端 logout 清除 HttpOnly token Cookie（Set-Cookie Max-Age=0），
    // 再清前端状态。否则 Cookie 仍在，刷新后会自动登录回来。
    try {
      await authService.logout();
    } catch {
      // 网络失败也继续清本地，避免用户被困在已失效会话里
    }
    authStore.logout();
    toast.success(t('settings.logoutSuccess'));
  }

  function navStyle(active: boolean): string {
    return active
      ? 'background: var(--brand); color: var(--text-on-accent);'
      : 'color: var(--text-primary);';
  }

  function setThemeFamily(themeFamily: ThemeFamily) {
    themeStore.setThemeFamily(themeFamily);
  }
</script>

<svelte:head>
  <title>{t('settings.title')} - HotSpot</title>
</svelte:head>

{#if platformStore.useMasterDetail}
  <!-- Wide (landscape/large tablet + desktop): master-detail 两栏
       左栏导航 + 右栏按 section 切换。窄屏改用长滚动（见 {:else}），
       这是信息架构级差异，非纯布局——容器查询无法消解。 -->
  <div class="flex h-full">
    <!-- Left navigation panel -->
    <div
      class="flex flex-shrink-0 flex-col overflow-y-auto no-scrollbar"
      style="width: 260px; border-right: 1px solid var(--border-theme); padding: 20px 16px;"
    >
      {#if !platformStore.hideRedundantEntry}
        <!-- 非桌面（无常驻侧栏）：设置是子页，自带返回头部，避免被困。 -->
        <div class="mb-3 flex items-center gap-2.5">
          <BackButton onclick={() => history.back()} size={40} />
          <h1 class="page-title" style="color: var(--text-primary);">{t('settings.title')}</h1>
        </div>
      {/if}
      <!-- User card -->
      {#if authStore.currentAccount}
        <div
          class="flex items-center gap-3 rounded-[var(--radius-md)] px-3.5 py-3"
          style="background: var(--surface-bg); border: 1px solid var(--border-theme); margin-bottom: 16px;"
        >
          <div
            class="flex-shrink-0 overflow-hidden rounded-full"
            style="width: 42px; height: 42px; padding: 2px; border: 1px solid var(--border-theme);"
          >
            <Avatar
              src={authStore.currentAccount.avatar_url}
              name={authStore.currentAccount.nickname}
              alt={authStore.currentAccount.nickname}
              size={40}
              border={false}
            />
          </div>
          <div class="min-w-0">
            <p class="text-body-lg" style="color: var(--text-primary); font-weight: 500">
              {authStore.currentAccount.nickname || t('settings.guestFallback')}
            </p>
            <p class="text-caption" style="color: var(--text-secondary)">
              @{formatHandle(authStore.currentAccount)}
            </p>
          </div>
        </div>
      {/if}

      <!-- Navigation items -->
      <div class="flex flex-col gap-1">
        <button
          type="button"
          class="flex items-center gap-2.5 rounded-[var(--radius-md)] px-3 py-2.5 text-left transition-all hover:bg-[var(--theme-accent-soft)]"
          style={navStyle(activeSection === 'appearance')}
          onclick={() => (activeSection = 'appearance')}
        >
          <Palette size={16} strokeWidth={1.5} />
          <span class="text-body" style="font-weight: 500">{t('settings.appearanceAndTheme')}</span>
        </button>

        {#each settingsSections as section (section.id)}
          <button
            type="button"
            class="flex items-center gap-2.5 rounded-[var(--radius-md)] px-3 py-2.5 text-left transition-all hover:bg-[var(--theme-accent-soft)]"
            style={navStyle(activeSection === section.id)}
            onclick={() => (activeSection = section.id)}
          >
            <section.icon size={16} strokeWidth={1.5} />
            <span class="text-body" style="font-weight: 500">{section.title}</span>
          </button>
        {/each}

        <button
          type="button"
          class="flex items-center gap-2.5 rounded-[var(--radius-md)] px-3 py-2.5 text-left transition-all hover:bg-[var(--theme-accent-soft)]"
          style={navStyle(activeSection === 'notifications')}
          onclick={() => (activeSection = 'notifications')}
        >
          <SlidersHorizontal size={16} strokeWidth={1.5} />
          <span class="text-body" style="font-weight: 500">{t('settings.notificationsPreference')}</span>
        </button>

        <!-- Logout -->
        <div style="margin-top: 12px; height: 1px; background: var(--border-theme);"></div>
        <button
          type="button"
          class="flex items-center gap-2.5 rounded-[var(--radius-md)] px-3 py-2.5 text-left transition-all hover:bg-[var(--theme-accent-soft)]"
          style="color: var(--brand);"
          onclick={handleLogout}
        >
          <LogOut size={16} strokeWidth={1.5} />
          <span class="text-body" style="font-weight: 500">{t('settings.logout')}</span>
        </button>
      </div>
    </div>

    <!-- Right content panel -->
    <div class="flex-1 overflow-y-auto no-scrollbar page-pad">
      <div class="page-shell page-shell--narrow">
        {#if activeSection === 'appearance'}
          <!-- Appearance section -->
          <div class="mb-6">
            <h2
              class="text-h2"
              style="font-weight: 600; color: var(--text-primary); margin-bottom: 4px"
            >
              {t('settings.appearanceAndTheme')}
            </h2>
            <p class="text-meta" style="color: var(--text-secondary)">{t('settings.appearanceAndThemeDesc')}</p>
          </div>

          <!-- Theme picker -->
          <div class="mb-6">
            <p
              class="text-xs"
              style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 10px"
            >
              {t('settings.themeStyle')}
            </p>
            <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px;">
              {#each THEME_FAMILY_OPTIONS as option (option.family)}
                {@const accent = PREVIEW_ACCENT[option.family]}
                {@const isSelected = currentThemeFamily === option.family}
                <button
                  type="button"
                  class="relative flex flex-col items-center gap-3 rounded-[var(--radius-lg)] py-4 text-center transition-all"
                  class:surface-flat={!isSelected}
                  style="background: var(--surface-bg); {isSelected
                    ? 'box-shadow: 0 0 0 2.5px ' +
                      accent +
                      ', 5px 5px 12px rgba(0,0,0,0.06); border: 1.5px solid ' +
                      accent +
                      ';'
                    : ''}"
                  onclick={() => setThemeFamily(option.family)}
                >
                  <div
                    style="width: 60px; height: 80px; border-radius: var(--radius-sm); background: var(--surface-bg); border: 1px solid var(--border-theme); position: relative; overflow: hidden;"
                  >
                    <div
                      style="position: absolute; bottom: 0; left: 0; right: 0; display: flex; justify-content: space-around; align-items: center; height: 20px; background: var(--surface-bg); border-top: 1px solid {accent}30;"
                    >
                      {#each [0, 1, 2] as i (i)}
                        <div
                          style="width: {i === 1 ? '14px' : '10px'}; height: {i === 1
                            ? '14px'
                            : '8px'}; border-radius: {i === 1 ? '7px' : '4px'}; background: {i === 1
                            ? accent
                            : `${accent}55`};"
                        ></div>
                      {/each}
                    </div>
                    <div
                      style="display: flex; flex-direction: column; gap: 6px; padding: 12px 8px 0 8px;"
                    >
                      {#each [80, 60, 70] as w, i (i)}
                        <div
                          style="width: {w}%; height: 5px; border-radius: var(--radius-xs); background: {i === 0
                            ? `${accent}90`
                            : `${accent}35`};"
                        ></div>
                      {/each}
                    </div>
                  </div>
                  <div
                    style="display: flex; flex-direction: column; align-items: center; gap: 2px;"
                  >
                    <span class="text-meta" style="color: {accent}"
                      >{THEME_NAMES[option.family]}</span
                    >
                    <span
                      class="text-xs"
                      style="text-align: center; line-height: 1.4; color: {accent}99"
                    >
                      {THEME_DESC[option.family]}
                    </span>
                  </div>
                  {#if isSelected}
                    <div
                      style="display: flex; align-items: center; justify-content: center; border-radius: 50%; width: 20px; height: 20px; background: {accent};"
                    >
                      <Check size={12} strokeWidth={3} color="var(--text-on-accent)" />
                    </div>
                  {/if}
                </button>
              {/each}
            </div>
          </div>

          <!-- Dark mode toggle -->
          <div>
            <p
              class="text-xs"
              style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-bottom: 10px"
            >
              {t('settings.darkMode')}
            </p>
            <div
              class="flex items-center justify-between rounded-[var(--radius-md)] px-4 py-3.5"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              <div class="flex items-center gap-3">
                <div
                  class="flex flex-shrink-0 items-center justify-center rounded-full"
                  style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
                >
                  <Moon size={15} strokeWidth={1.5} color="var(--brand)" />
                </div>
                <div>
                  <p class="text-body-lg" style="color: var(--text-primary)">{t('settings.darkMode')}</p>
                  <p class="text-caption" style="color: var(--text-secondary)">
                    {t('settings.darkModeDesc')}
                  </p>
                </div>
              </div>
              <ThemeToggle />
            </div>
          </div>
        {:else if activeSection === 'notifications'}
          <!-- Notifications section -->
          <div class="mb-6">
            <h2
              class="text-h2"
              style="font-weight: 600; color: var(--text-primary); margin-bottom: 4px"
            >
              {t('settings.notificationsPreference')}
            </h2>
            <p class="text-meta" style="color: var(--text-secondary)">{t('settings.notificationsPreferenceDesc')}</p>
          </div>
          <div
            class="overflow-hidden rounded-[var(--radius-md)]"
            style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
          >
            {#each notificationItems as item, i (item.key)}
              {#if i > 0}
                <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
              {/if}
              <div class="flex items-center justify-between px-4 py-3.5">
                <span class="text-body-lg" style="color: var(--text-primary)">{item.label}</span>
                <Switch
                  bind:checked={notifications[item.key]}
                  aria-label={item.label}
                />
              </div>
            {/each}
          </div>
        {:else}
          <!-- Dynamic section based on activeSection -->
          {@const section = settingsSections.find((s) => s.id === activeSection)}
          {#if section}
            <div class="mb-6">
              <h2
                class="text-h2"
                style="font-weight: 600; color: var(--text-primary); margin-bottom: 4px"
              >
                {section.title}
              </h2>
            </div>
            <div
              class="overflow-hidden rounded-[var(--radius-md)]"
              style="background: var(--surface-bg); border: 1px solid var(--border-theme);"
            >
              {#each section.items as item, i (item.href)}
                {#if i > 0}
                  <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
                {/if}
                <a href={item.href} class="list-row flex items-center gap-3 px-4 py-3.5">
                  <div
                    class="flex flex-shrink-0 items-center justify-center rounded-full"
                    style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
                  >
                    <item.icon size={15} strokeWidth={1.5} color="var(--text-secondary)" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="text-body-lg" style="color: var(--text-primary)">{item.label}</p>
                    <p class="text-caption" style="color: var(--text-secondary)">
                      {item.description}
                    </p>
                  </div>
                  <ChevronRight size={15} strokeWidth={1.5} color="var(--text-secondary)" />
                </a>
              {/each}
            </div>
          {/if}
        {/if}
      </div>
    </div>
  </div>
{:else}
  <!-- Narrow (mobile + portrait/small tablet): single column layout -->
  <div class="page-pad">
    <div class="page-shell page-shell--form">
      <!-- Header with back button -->
      <div
        class="sticky top-0 z-10 page-pad-x pt-3 pb-3"
        style="background-color: var(--surface-bg); border-bottom: 1px solid var(--border-theme);"
      >
        <div class="page-shell page-shell--form flex items-center gap-3">
          <BackButton onclick={() => history.back()} size={40} />
          <h1 class="page-title" style="color: var(--text-primary);">{t('settings.title')}</h1>
        </div>
      </div>

      <div class="space-y-3">
        <!-- User card -->
        {#if authStore.currentAccount}
          <div class="surface-flat flex items-center gap-3 rounded-[var(--radius-lg)] px-4 py-3.5">
            <div
              class="flex-shrink-0 overflow-hidden rounded-full"
              style="width: 50px; height: 50px; padding: 2px; border: 1px solid var(--border-theme);"
            >
              <Avatar
                src={authStore.currentAccount.avatar_url}
                name={authStore.currentAccount.nickname}
                alt={authStore.currentAccount.nickname}
                size={40}
                border={false}
              />
            </div>
            <div class="min-w-0">
              <p class="text-body-lg" style="color: var(--text-primary); font-weight: 500">
                {authStore.currentAccount.nickname || t('settings.guestFallback')}
              </p>
              <p class="text-caption" style="color: var(--text-secondary)">
                @{formatHandle(authStore.currentAccount)}
              </p>
            </div>
          </div>
        {/if}

        <!-- Theme picker (standalone section) -->
        <p
          class="text-xs px-1"
          style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px"
        >
          {t('settings.themeStyle')}
        </p>
        <div class="flex gap-2.5">
          {#each THEME_FAMILY_OPTIONS as option (option.family)}
            {@const accent = PREVIEW_ACCENT[option.family]}
            {@const isSelected = currentThemeFamily === option.family}
            <button
              type="button"
              class="relative flex flex-1 flex-col items-center gap-3 rounded-[var(--radius-lg)] py-4 text-center transition-all"
              class:surface-flat={!isSelected}
              style="background: var(--surface-bg); {isSelected
                ? 'box-shadow: 0 0 0 2.5px ' +
                  accent +
                  ', 5px 5px 12px rgba(0,0,0,0.06); border: 1.5px solid ' +
                  accent +
                  ';'
                : ''}"
              onclick={() => setThemeFamily(option.family)}
            >
              <!-- Mini phone frame preview -->
              <div
                style="width: 60px; height: 80px; border-radius: var(--radius-sm); background: var(--surface-bg); border: 1px solid var(--border-theme); position: relative; overflow: hidden;"
              >
                <div
                  style="position: absolute; bottom: 0; left: 0; right: 0; display: flex; justify-content: space-around; align-items: center; height: 20px; background: var(--surface-bg); border-top: 1px solid {accent}30;"
                >
                  {#each [0, 1, 2] as i (i)}
                    <div
                      style="width: {i === 1 ? '14px' : '10px'}; height: {i === 1
                        ? '14px'
                        : '8px'}; border-radius: {i === 1 ? '7px' : '4px'}; background: {i === 1
                        ? accent
                        : `${accent}55`};"
                    ></div>
                  {/each}
                </div>
                <div
                  style="display: flex; flex-direction: column; gap: 6px; padding: 12px 8px 0 8px;"
                >
                  {#each [80, 60, 70] as w, i (i)}
                    <div
                      style="width: {w}%; height: 5px; border-radius: var(--radius-xs); background: {i === 0
                        ? `${accent}90`
                        : `${accent}35`};"
                    ></div>
                  {/each}
                </div>
              </div>
              <div style="display: flex; flex-direction: column; align-items: center; gap: 2px;">
                <span class="text-meta" style="color: {accent}">{THEME_NAMES[option.family]}</span>
                <span
                  class="text-xs"
                  style="text-align: center; line-height: 1.4; color: {accent}99"
                >
                  {THEME_DESC[option.family]}
                </span>
              </div>
              {#if isSelected}
                <div
                  style="display: flex; align-items: center; justify-content: center; border-radius: 50%; width: 20px; height: 20px; background: {accent};"
                >
                  <Check size={12} strokeWidth={3} color="var(--text-on-accent)" />
                </div>
              {/if}
            </button>
          {/each}
        </div>

        <!-- Appearance -->
        <p
          class="text-xs px-1"
          style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px"
        >
          {t('settings.appearanceSection')}
        </p>
        <div class="surface-flat overflow-hidden rounded-[var(--radius-lg)]">
          <div class="flex items-center justify-between px-4 py-3.5">
            <div class="flex items-center gap-3">
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
              >
                <Palette size={15} strokeWidth={1.5} color="var(--brand)" />
              </div>
              <div>
                <p class="text-body-lg" style="color: var(--text-primary)">{t('settings.darkMode')}</p>
                <p class="text-caption" style="color: var(--text-secondary)">
                  {t('settings.darkModeDesc')}
                </p>
              </div>
            </div>
            <ThemeToggle />
          </div>
        </div>

        {#each settingsSections as section (section.title)}
          <p
            class="text-xs px-1"
            style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px"
          >
            {section.title}
          </p>
          <div class="surface-flat overflow-hidden rounded-[var(--radius-lg)]">
            {#each section.items as item, i (item.href)}
              {#if i > 0}
                <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
              {/if}
              <a href={item.href} class="flex items-center gap-3 px-4 py-3.5 transition-colors">
                <div
                  class="flex flex-shrink-0 items-center justify-center rounded-full"
                  style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
                >
                  <item.icon size={15} strokeWidth={1.5} color="var(--text-secondary)" />
                </div>
                <div class="min-w-0 flex-1">
                  <p class="text-body-lg" style="color: var(--text-primary)">{item.label}</p>
                  <p class="text-caption" style="color: var(--text-secondary)">
                    {item.description}
                  </p>
                </div>
                <ChevronRight size={15} strokeWidth={1.5} color="var(--text-secondary)" />
              </a>
            {/each}
          </div>
        {/each}

        <!-- Notification toggles -->
        <p
          class="text-xs px-1"
          style="letter-spacing: 0.14em; text-transform: uppercase; color: var(--text-secondary); margin-top: 8px"
        >
          {t('settings.notificationsPreference')}
        </p>
        <div class="surface-flat overflow-hidden rounded-[var(--radius-lg)]">
          {#each notificationItems as item, i (item.key)}
            {#if i > 0}
              <div class="mx-4" style="height: 1px; background: var(--border-theme);"></div>
            {/if}
            <div class="flex items-center justify-between px-4 py-3.5">
              <span class="text-body-lg" style="color: var(--text-primary)">{item.label}</span>
              <Switch
                checked={notifications[item.key]}
                onchange={(val) => (notifications[item.key] = val)}
                aria-label={item.label}
              />
            </div>
          {/each}
        </div>

        <!-- Log Out -->
        <div class="mt-5">
          <button
            class="surface-flat flex w-full items-center gap-3.5 rounded-[var(--radius-lg)] px-4 py-3.5 text-left transition-colors"
            onclick={handleLogout}
          >
            <div
              class="flex flex-shrink-0 items-center justify-center rounded-full"
              style="width: 36px; height: 36px; border: 1px solid var(--border-theme);"
            >
              <LogOut size={15} strokeWidth={1.5} color="var(--brand)" />
            </div>
            <span class="text-body-lg" style="color: var(--brand)">{t('settings.logout')}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
