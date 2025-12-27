<template>
  <UCard
    variant="outline"
    class="group relative flex flex-col h-full overflow-hidden"
    data-hs-card="interactive"
    @click="$emit('click', user)"
  >
    <template #header>
      <div class="flex flex-col gap-3">
        <div class="flex items-start justify-between gap-3">
          <UUser
            class="min-w-0"
            :name="user.nickname"
            :description="user.bio || '暂无简介'"
            :avatar="{
              src: user.avatar_url,
              alt: user.nickname,
              size: 'xl',
              chip: isOnline ? { color: 'success', position: 'bottom-right', size: '2xs', inset: false } : false
            }"
          />

          <div class="flex gap-1 shrink-0">
            <UBadge
              v-if="isFriend"
              color="primary"
              variant="soft"
              size="xs"
            >好友</UBadge>
            <UBadge
              v-else-if="isFollowing"
              color="neutral"
              variant="soft"
              size="xs"
            >已关注</UBadge>
            <UBadge
              v-if="userTypeBadge"
              :color="userTypeBadge.color"
              variant="soft"
              size="xs"
            >
              {{ userTypeBadge.label }}
            </UBadge>
          </div>
        </div>

        <div
          v-if="user.geo"
          class="flex flex-wrap gap-2"
        >
          <UBadge
            v-if="user.geo.location_name"
            :label="user.geo.location_name"
            icon="i-lucide-map-pin"
            color="neutral"
            variant="soft"
            size="xs"
            class="max-w-full truncate"
          />
          <UBadge
            v-if="user.geo.distance !== undefined"
            :label="formatDistance(user.geo.distance)"
            icon="i-lucide-send"
            color="neutral"
            variant="soft"
            size="xs"
          />
        </div>
      </div>

    </template>

    <!-- 底部操作栏 -->
    <template #footer>
      <div class="flex items-center justify-between">
        <div
          class="flex items-center gap-1"
          @click.stop
        >
          <!-- 关注/取消关注 -->
          <UTooltip
            :text="isFollowing ? '取消关注' : '关注用户'"
            :portal="false"
          >
            <UButton
              :variant="isFollowing ? 'ghost' : 'soft'"
              :color="isFollowing ? 'neutral' : 'primary'"
              size="sm"
              :square="true"
              :icon="isFollowing ? 'i-lucide-user-minus' : 'i-lucide-user-plus'"
              :loading="followOperating"
              :disabled="followOperating"
              @click.stop="isFollowing ? handleUnfollowUser() : handleFollowUser()"
            />
          </UTooltip>

          <!-- 发送消息 -->
          <UTooltip
            text="发送消息"
            :portal="false"
          >
            <UButton
              variant="ghost"
              color="neutral"
              size="sm"
              :square="true"
              icon="i-lucide-messages-square"
              @click.stop="handleSendMessage"
            />
          </UTooltip>

          <!-- 举报用户 -->
          <UTooltip
            text="举报用户"
            :portal="false"
          >
            <UButton
              variant="ghost"
              color="error"
              size="sm"
              :square="true"
              icon="i-lucide-flag"
              @click.stop="handleReportUser"
            />
          </UTooltip>
        </div>
      </div>
    </template>
  </UCard>
</template>

<script setup lang="ts">
import type { GeoData } from '~/types/api'
import { useUsersService } from '~/services/users.service'
import { isTempUser } from '~/utils/userTypes'
import { useAuthStore } from '~/stores/auth.store'
import { useHydratedStore } from '~/composables/useHydratedStore'

// 关系状态缓存（避免页面内多个卡片重复请求；页面刷新后会重新拉取一次）
const relationshipCache = new Map<number, { is_mutual_follow: boolean; is_following: boolean; ts: number }>()
const relationshipInflight = new Map<number, Promise<void>>()
let relationshipActive = 0
const relationshipQueue: Array<() => void> = []
const REL_CACHE_TTL_MS = 30_000
const REL_MAX_CONCURRENCY = 4

async function withRelationshipLimit<T>(fn: () => Promise<T>): Promise<T> {
  if (relationshipActive >= REL_MAX_CONCURRENCY) {
    await new Promise<void>((resolve) => relationshipQueue.push(resolve))
  }
  relationshipActive += 1
  try {
    return await fn()
  } finally {
    relationshipActive -= 1
    const next = relationshipQueue.shift()
    if (next) next()
  }
}

interface UserCardData {
  user_id: number;
  nickname: string;
  avatar_url?: string;
  user_type?: string;
  bio?: string;
  tags?: string[];
  is_online?: boolean;
  geo?: GeoData;
  relationship?: {
    is_mutual_follow?: boolean;
    is_following?: boolean;
  };
}

interface Props {
  user: UserCardData
}

interface Emits {
  click: [user: UserCardData]
  message: [user: UserCardData]
  more: [user: UserCardData]
}

const props = defineProps<Props>()
const { formatDistance } = useFormatters()
const emit = defineEmits<Emits>()

const authStore = useAuthStore()
const isAuthenticated = useHydratedStore(authStore, 'isAuthenticated', false)
const { requireAuth } = useRequireAuth()

const isOnline = computed(() => props.user.is_online ?? false)
const relationshipOverride = ref<{ is_mutual_follow?: boolean; is_following?: boolean } | null>(null)
watch(
  () => props.user.relationship,
  (v) => {
    // 外部数据刷新时，同步覆盖本地 override（避免长时间不一致）
    relationshipOverride.value = v ? { ...v } : null
  },
  { immediate: true }
)

async function ensureRelationshipLoaded(userId: number) {
  // 只有已登录场景才有 relationship 的意义；未登录直接跳过
  if (!isAuthenticated.value) return
  // 如果外部已经给了 relationship，就不需要拉
  if (props.user.relationship) return

  const cached = relationshipCache.get(userId)
  if (cached && Date.now() - cached.ts < REL_CACHE_TTL_MS) {
    relationshipOverride.value = { ...cached }
    return
  }

  const inflight = relationshipInflight.get(userId)
  if (inflight) {
    await inflight
    const again = relationshipCache.get(userId)
    if (again) relationshipOverride.value = { ...again }
    return
  }

  const userService = useUsersService()
  const p = withRelationshipLimit(async () => {
    try {
      const rel = await userService.getRelationship({ id: userId })
      const value = {
        is_mutual_follow: !!rel.is_mutual_follow,
        is_following: !!rel.is_following,
        ts: Date.now()
      }
      relationshipCache.set(userId, value)
      relationshipOverride.value = { ...value }
    } catch {
      // 静默失败：可能未登录、token 过期、网络问题等；不影响页面展示与点击 follow 的主流程
    }
  }).finally(() => {
    relationshipInflight.delete(userId)
  })

  relationshipInflight.set(userId, p)
  await p
}

watch(
  () => [props.user.user_id, isAuthenticated.value] as const,
  ([userId]) => {
    // relationship 缺失时补拉（页面刷新后恢复“已关注/好友”状态）
    if (!props.user.relationship) {
      ensureRelationshipLoaded(userId)
    }
  },
  { immediate: true }
)

const isFriend = computed(() => relationshipOverride.value?.is_mutual_follow ?? false)
const isFollowing = computed(() => relationshipOverride.value?.is_following ?? false)

const userTypeBadge = computed<{ label: string; color: 'warning' | 'neutral' } | null>(() => {
  const t = props.user.user_type
  if (!t) return null
  if (isTempUser(t)) return { label: '临时', color: 'warning' }
  // 非 normal 一律视为“访客/特殊用户”，避免重复展示 TEMP + Guest
  if (t !== 'normal') return { label: '访客', color: 'neutral' }
  return null
})

const followOperating = ref(false)

const handleFollowUser = async () => {
  if (followOperating.value) return
  if (!requireAuth('关注用户')) return
  const feedback = useFeedbackStore()
  const userService = useUsersService()
  try {
    followOperating.value = true
    await userService.follow({ id: props.user.user_id })
    // 立即更新 UI（避免用户重复点击）
    relationshipOverride.value = {
      ...(relationshipOverride.value ?? {}),
      is_following: true
    }

    // 同步缓存（避免同页其它卡片/后续重新渲染出现“又可关注”）
    relationshipCache.set(props.user.user_id, {
      is_mutual_follow: !!relationshipOverride.value?.is_mutual_follow,
      is_following: true,
      ts: Date.now()
    })

    // 尝试拉一次 relationship，补齐互相关注状态（失败不影响主流程）
    userService
      .getRelationship({ id: props.user.user_id })
      .then((rel) => {
        relationshipOverride.value = { ...rel }
        relationshipCache.set(props.user.user_id, {
          is_mutual_follow: !!rel.is_mutual_follow,
          is_following: !!rel.is_following,
          ts: Date.now()
        })
      })
      .catch(() => { })
    feedback.success('关注成功', `已关注 ${props.user.nickname}`)
  } catch (error: unknown) {
    feedback.error('关注失败', getErrorMessage(error, '关注用户时发生错误'))
  } finally {
    followOperating.value = false
  }
}

const handleUnfollowUser = async () => {
  if (followOperating.value) return
  if (!requireAuth('取消关注')) return
  const feedback = useFeedbackStore()
  const userService = useUsersService()
  try {
    followOperating.value = true
    await userService.unfollow({ id: props.user.user_id })
    relationshipOverride.value = {
      ...(relationshipOverride.value ?? {}),
      is_following: false,
      is_mutual_follow: false
    }

    relationshipCache.set(props.user.user_id, {
      is_mutual_follow: false,
      is_following: false,
      ts: Date.now()
    })

    userService
      .getRelationship({ id: props.user.user_id })
      .then((rel) => {
        relationshipOverride.value = { ...rel }
        relationshipCache.set(props.user.user_id, {
          is_mutual_follow: !!rel.is_mutual_follow,
          is_following: !!rel.is_following,
          ts: Date.now()
        })
      })
      .catch(() => { })
    feedback.success('已取消关注', `已取消关注 ${props.user.nickname}`)
  } catch (error: unknown) {
    feedback.error('取消关注失败', getErrorMessage(error, '取消关注时发生错误'))
  } finally {
    followOperating.value = false
  }
}

const handleSendMessage = () => {
  if (!requireAuth('发送消息')) return
  emit('message', props.user)
}

const handleReportUser = async () => {
  if (!requireAuth('提交举报')) return
  const modal = useModal()
  const feedback = useFeedbackStore()
  await modal.showReport('user', String(props.user.user_id))
  feedback.success('举报已提交', '感谢您的举报，我们会尽快处理')
}
</script>
