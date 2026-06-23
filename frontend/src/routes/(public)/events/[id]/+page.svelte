<script lang="ts">
  import { page } from '$app/state';
  import ChevronLeft from 'lucide-svelte/icons/chevron-left';
  import Calendar from 'lucide-svelte/icons/calendar';
  import Clock from 'lucide-svelte/icons/clock';
  import MapPin from 'lucide-svelte/icons/map-pin';
  import Users from 'lucide-svelte/icons/users';
  import Check from 'lucide-svelte/icons/check';
  import Star from 'lucide-svelte/icons/star';
  import ChevronRight from 'lucide-svelte/icons/chevron-right';
  import Share2 from 'lucide-svelte/icons/share-2';

  /* ── Mock data ─────────────────────────────────────────────────── */
  interface MockParticipant {
    id: number;
    name: string;
    avatar: string;
  }

  interface MockEvent {
    id: number;
    title: string;
    tag: string;
    color: string;
    cover: string;
    description: string;
    location: string;
    date: string;
    time: string;
    host: { name: string; avatar: string };
    participants: MockParticipant[];
    maxParticipants: number;
    distance: string;
    highlights: string[];
  }

  const EVENTS: Record<number, MockEvent> = {
    1: {
      id: 1,
      title: '云龙湖畔晨跑',
      tag: '运动',
      color: 'var(--theme-success)',
      cover: '',
      description:
        '每周六清晨，在云龙湖畔感受微风拂面，与志同道合的跑友一起奔跑。路线全程约5公里，适合各水平跑者参加。跑后我们会在湖边咖啡馆小聚，交流跑步心得。',
      location: '云龙湖风景区·徐州市泉山区',
      date: '2026年5月16日 周六',
      time: '06:30 - 08:00',
      host: {
        name: '跑步达人小王',
        avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80'
      },
      participants: [
        {
          id: 1,
          name: '小明',
          avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80'
        },
        {
          id: 2,
          name: '小红',
          avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80'
        },
        {
          id: 3,
          name: '阿强',
          avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80'
        },
        {
          id: 4,
          name: '小丽',
          avatar: 'https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80'
        }
      ],
      maxParticipants: 30,
      distance: '0.8km',
      highlights: ['湖景跑道', '免费饮用水', '跑后小聚', '适合新手']
    },
    2: {
      id: 2,
      title: '街头音乐节',
      tag: '音乐',
      color: 'var(--brand)',
      cover: '',
      description:
        '在城市的中心，用音乐连接每一颗心。本次街头音乐节将邀请本地独立乐队和歌手进行现场表演，免费入场，欢迎所有人来感受音乐的力量。',
      location: '彭城广场·徐州市云龙区',
      date: '2026年5月17日 周日',
      time: '14:00 - 20:00',
      host: {
        name: '本地向导',
        avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80'
      },
      participants: [
        {
          id: 1,
          name: '大伟',
          avatar: 'https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80'
        },
        {
          id: 2,
          name: '小美',
          avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80'
        },
        {
          id: 3,
          name: '阿杰',
          avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80'
        }
      ],
      maxParticipants: 200,
      distance: '1.2km',
      highlights: ['免费入场', '现场表演', '多个乐队', '本地艺术家']
    },
    3: {
      id: 3,
      title: '徐州美食探店',
      tag: '美食',
      color: 'var(--theme-warning)',
      cover: '',
      description:
        '一起探访徐州地道小吃，从户部山街头的把子肉到回龙窝的老味煎饼，感受舌尖上的徐州。每站都有独家折扣，名额有限，先到先得！',
      location: '户部山步行街·徐州市鼓楼区',
      date: '2026年5月18日 周一',
      time: '11:00 - 14:00',
      host: {
        name: '美食博主小张',
        avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80'
      },
      participants: [
        {
          id: 1,
          name: '阿花',
          avatar: 'https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80'
        },
        {
          id: 2,
          name: '老刘',
          avatar: 'https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80'
        }
      ],
      maxParticipants: 15,
      distance: '0.5km',
      highlights: ['独家折扣', '地道美食', '本地向导', '小团体']
    }
  };

  const eventId = $derived(Number(page.params.id));
  const ev: MockEvent = $derived(EVENTS[eventId] ?? EVENTS[1]!);
  const fillRate = $derived(Math.round((ev.participants.length / ev.maxParticipants) * 100));
  const remaining = $derived(ev.maxParticipants - ev.participants.length);
  let joined = $state(false);
  const accentFaint = $derived(`color-mix(in srgb, ${ev.color} 8%, transparent)`);
  const accentSoft = $derived(`color-mix(in srgb, ${ev.color} 15%, transparent)`);
  const accentMedium = $derived(`color-mix(in srgb, ${ev.color} 25%, transparent)`);
  const accentRing = $derived(`color-mix(in srgb, ${ev.color} 18%, transparent)`);
  const accentShadow = $derived(`color-mix(in srgb, ${ev.color} 40%, transparent)`);
  const ctaShadow = $derived(`color-mix(in srgb, ${ev.color} 45%, transparent)`);
  const ctaStyle = $derived(
    joined
      ? `background: var(--surface-bg); border: 2px solid ${ev.color}; color: ${ev.color}; box-shadow: 0 0 0 4px ${accentRing}; font-size: 15px;`
      : `background: ${ev.color}; color: var(--text-on-accent); box-shadow: 0 8px 24px ${ctaShadow}; border: none; font-size: 15px;`
  );

  function handleBack() {
    history.back();
  }

  function toggleJoin() {
    joined = !joined;
  }
</script>

<svelte:head>
  <title>{ev.title} - HotSpot</title>
</svelte:head>

<div class="flex h-full flex-col" style="background: var(--surface-bg);">
  <!-- Scrollable area -->
  <div class="flex-1 overflow-y-auto no-scrollbar page-pad">
    <div class="page-shell page-shell--narrow">
      <!-- Cover image -->
      <div
        class="relative flex-shrink-0 overflow-hidden"
        style="height: clamp(200px, 32vw, 280px);"
      >
        {#if ev.cover}
          <img src={ev.cover} alt={ev.title} class="absolute inset-0 h-full w-full object-cover" />
        {:else}
          <!-- Gradient placeholder -->
          <div
            class="absolute inset-0"
            style="background: linear-gradient(135deg, {accentMedium}, {accentFaint}, var(--surface-bg));"
          ></div>
        {/if}
        <div
          class="absolute inset-0"
          style="background: linear-gradient(to bottom, transparent 30%, {accentSoft} 100%);"
        ></div>

        <!-- Back button -->
        <div class="absolute top-4 left-4 flex gap-2">
          <button
            type="button"
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
            onclick={handleBack}
            aria-label="返回"
          >
            <ChevronLeft size={15} strokeWidth={1.75} color="var(--text-on-overlay)" />
          </button>
        </div>

        <!-- Share button -->
        <div class="absolute top-4 right-4">
          <button
            type="button"
            class="flex items-center justify-center rounded-full outline-none"
            style="width: 38px; height: 38px; background: rgba(0,0,0,0.35); backdrop-filter: blur(10px); border: 1px solid var(--glass-border);"
            aria-label="分享"
          >
            <Share2 size={14} strokeWidth={1.75} color="var(--text-on-overlay)" />
          </button>
        </div>

        <!-- Tag pill -->
        <div class="absolute bottom-4 left-4">
          <span
            class="rounded-full px-3 py-1"
            style="font-size: 12px; background: {ev.color}; color: var(--text-on-accent); box-shadow: 0 3px 10px {accentShadow};"
          >
            {ev.tag}
          </span>
        </div>
      </div>

      <!-- Content -->
      <div class="pt-4">
        <h1
          style="font-size: 24px; font-weight: 600; color: var(--text-primary); margin-bottom: 8px;"
        >
          {ev.title}
        </h1>

        <!-- Meta info -->
        <div class="mb-5 flex flex-col gap-2">
          {#each [{ icon: Calendar, text: ev.date }, { icon: Clock, text: ev.time }, { icon: MapPin, text: ev.location }] as meta (meta.text)}
            <div class="flex items-center gap-2.5">
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 30px; height: 30px; background: {accentSoft};"
              >
                <meta.icon size={13} strokeWidth={1.75} color={ev.color} />
              </div>
              <span style="font-size: 13px; color: var(--text-secondary);">{meta.text}</span>
            </div>
          {/each}
        </div>

        <!-- Host card -->
        <div
          class="mb-4 flex items-center gap-3 rounded-[var(--radius-lg)] px-4 py-3.5"
          style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
        >
          <div
            class="flex-shrink-0 overflow-hidden rounded-full"
            style="width: 40px; height: 40px; border: 2px solid {ev.color};"
          >
            <img src={ev.host.avatar} alt={ev.host.name} class="h-full w-full object-cover" />
          </div>
          <div class="flex-1">
            <p style="font-size: 13px; color: var(--text-primary);">{ev.host.name}</p>
            <div class="flex items-center gap-1">
              <Star
                size={10}
                style="color: var(--theme-warning);"
                fill="var(--theme-warning)"
                strokeWidth={1.5}
              />
              <span style="font-size: 11px; color: var(--text-secondary);">活动发起人</span>
            </div>
          </div>
          <button
            type="button"
            class="rounded-full px-3 py-1.5 outline-none"
            style="background: {ev.color}; color: var(--text-on-accent); font-size: 12px; box-shadow: 0 3px 10px {accentShadow};"
          >
            关注
          </button>
        </div>

        <!-- Description -->
        <p
          style="font-size: 14px; color: var(--text-secondary); line-height: 1.75; margin-bottom: 20px;"
        >
          {ev.description}
        </p>

        <!-- Highlights -->
        <div
          class="mb-5 grid gap-2.5"
          style="grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));"
        >
          {#each ev.highlights as h, i (i)}
            <div
              class="flex items-center gap-2.5 rounded-[16px] px-3.5 py-3"
              style="background: var(--surface-bg); box-shadow: var(--shadow-md); border: 1px solid var(--border-theme);"
            >
              <div
                class="flex flex-shrink-0 items-center justify-center rounded-full"
                style="width: 24px; height: 24px; background: {accentSoft};"
              >
                <Check size={11} color={ev.color} strokeWidth={2.5} />
              </div>
              <span style="font-size: 12px; color: var(--text-primary);">{h}</span>
            </div>
          {/each}
        </div>

        <!-- Participants section -->
        <div class="mb-5">
          <div class="mb-3 flex items-center justify-between">
            <p style="font-size: 14px; color: var(--text-primary);">
              参与者 · {ev.participants.length}/{ev.maxParticipants}
            </p>
            <button
              type="button"
              class="flex items-center gap-0.5 outline-none"
              style="color: var(--brand);"
            >
              <span style="font-size: 12px;">查看全部</span>
              <ChevronRight size={13} strokeWidth={1.5} />
            </button>
          </div>

          <!-- Progress bar -->
          <div
            class="mb-3 overflow-hidden rounded-full"
            style="height: 6px; background: var(--border-theme);"
          >
            <div
              style="height: 100%; width: {fillRate}%; background: {ev.color}; border-radius: 3px; transition: width 0.8s ease-out;"
            ></div>
          </div>

          <!-- Avatar stack -->
          <div class="flex items-center gap-1">
            {#each ev.participants as p, i (p.id)}
              <div
                class="overflow-hidden rounded-full"
                style="width: 36px; height: 36px; border: 2px solid var(--surface-bg); margin-left: {i >
                0
                  ? '-10px'
                  : '0'}; z-index: {ev.participants.length - i};"
              >
                <img src={p.avatar} alt={p.name} class="h-full w-full object-cover" />
              </div>
            {/each}
            {#if remaining > 0}
              <div
                class="flex items-center justify-center rounded-full"
                style="width: 36px; height: 36px; border: 2px solid var(--surface-bg); margin-left: -10px; background: {accentRing};"
              >
                <span style="font-size: 11px; color: {ev.color}; font-weight: 600;"
                  >+{remaining}</span
                >
              </div>
            {/if}
            <span style="font-size: 12px; color: var(--text-secondary); margin-left: 8px;">
              共 {ev.participants.length} 人
            </span>
          </div>
        </div>

        <!-- Distance info -->
        <div
          class="flex items-center gap-2 rounded-[16px] px-4 py-3"
          style="background: {accentFaint}; border: 1px solid {accentMedium};"
        >
          <MapPin size={14} color={ev.color} strokeWidth={1.5} />
          <span style="font-size: 13px; color: var(--text-secondary);">
            距你 {ev.distance} · 步行约 {Math.ceil(parseFloat(ev.distance) * 12)} 分钟
          </span>
        </div>
      </div>
    </div>
  </div>

  <!-- Sticky bottom CTA -->
  <div
    class="page-shell page-shell--narrow flex-shrink-0 px-5 pb-6 pt-3"
    style="background-color: var(--surface-bg); border-top: 1px solid var(--border-theme);"
  >
    <button
      type="button"
      class="flex w-full items-center justify-center gap-2 rounded-[var(--radius-lg)] py-4 outline-none transition-all duration-300"
      style={ctaStyle}
      onclick={toggleJoin}
    >
      {#if joined}
        <Check size={18} strokeWidth={2.5} />
        已报名 · 点击取消
      {:else}
        <Users size={18} strokeWidth={2} />
        立即报名参加
      {/if}
    </button>
  </div>
</div>
