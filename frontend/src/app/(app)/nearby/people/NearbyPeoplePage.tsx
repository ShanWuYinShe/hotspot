'use client'
import React, { useEffect, useMemo, useState } from "react"
import { Map, MapPin, Radar, SlidersHorizontal, UserCheck, UserPlus, Users } from "lucide-react"
import { motion, AnimatePresence } from "motion/react"
import { useRouter } from "next/navigation"
import { ROUTES } from "@/lib/routes"
import { usePlatform } from "@/contexts/PlatformContext"
import { useHoverConfig } from "@/hooks/useHoverConfig"
import { useDefaultLocation } from "@/hooks/useDefaultLocation"
import { ActionIconButton } from "@/components/shared/ActionIconButton"
import { BspCardCanvas } from "@/components/shared/BspCardCanvas"
import { PageHeader } from "@/components/shared/PageHeader"
import { SearchInput } from "@/components/shared/SearchInput"
import { OnlineDot } from "@/components/shared/OnlineDot"
import { TagPill } from "@/components/shared/TagPill"
import { EmptyState } from "@/components/shared/EmptyState"
import { getNearbyUsers } from "@/api"

interface NearbyUser {
  id: number; name: string; distance: string; avatar: string
  status: string; tag: string; mutual: number; note: string
}

const SORT_OPTIONS = ["距离", "在线", "共同好友"]
const FILTER_TAGS = ["全部", "摄影", "旅行", "音乐", "美食", "运动", "阅读", "艺术"]

function OnlineStrip({ users, onNavigate }: { users: NearbyUser[]; onNavigate: (path: string) => void }) {
  return (
    <section className="mb-5" aria-label="正在附近">
      <div className="flex items-center gap-2 mb-3">
        <motion.span animate={{ scale: [1, 1.45, 1], opacity: [1, 0.58, 1] }} transition={{ duration: 2, repeat: Infinity }} className="w-[7px] h-[7px] rounded-full" style={{ background: "var(--semantic-online)" }} />
        <span className="neu-eyebrow">正在附近</span>
        <span className="px-1.5 py-0.5 rounded-full bg-[color-mix(in_srgb,var(--semantic-online)_14%,transparent)] text-[var(--semantic-online)] text-[10px]">{users.length} 人在线</span>
      </div>
      <div className="flex gap-3 overflow-x-auto no-scrollbar pb-1">
        {users.map((user) => (
          <motion.button key={user.id} whileHover={{ y: -2 }} whileTap={{ scale: 0.93 }} onClick={() => onNavigate(`/user/${user.id}`)} className="flex-shrink-0 flex flex-col items-center gap-1.5 outline-none">
            <div className="relative">
              <img src={user.avatar} alt={user.name} className="w-14 h-14 rounded-full object-cover border-2 border-[var(--semantic-online)]" style={{ boxShadow: "0 0 0 2px var(--neu-bg)" }} />
              <OnlineDot size="lg" />
            </div>
            <span className="max-w-14 truncate text-[var(--neu-text-muted)] text-[11px]">{user.name}</span>
          </motion.button>
        ))}
      </div>
    </section>
  )
}

function UserCard({ user, followed, onFollow, onNavigate, format = "standard" }: { user: NearbyUser; followed: boolean; onFollow: () => void; onNavigate: () => void; format?: "focus" | "portrait" | "standard" }) {
  const hover = useHoverConfig()
  const [hovered, setHovered] = useState(false)
  return (
    <motion.article whileHover={hover.card as any} whileTap={hover.tap as any} onHoverStart={() => setHovered(true)} onHoverEnd={() => setHovered(false)} onClick={onNavigate} role="button" tabIndex={0} onKeyDown={(event) => { if (event.key === "Enter" || event.key === " ") onNavigate() }} className="flex h-full flex-col rounded-[20px] overflow-hidden cursor-pointer neu-surface-flat neu-card-hover neu-focus-ring" style={{ background: hovered ? hover.bgHover : "var(--neu-bg)", transition: "background 0.18s ease" }}>
      <div className="relative min-h-[160px] flex-1 overflow-hidden">
        <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
        <div className="absolute inset-x-0 bottom-0 h-3/5 bg-gradient-to-t from-black/70 to-transparent" />
        <div className="absolute top-2.5 right-2.5 flex items-center gap-1 px-1.5 py-0.5 rounded-full bg-[var(--overlay-strong)]" style={{ backdropFilter: "blur(8px)" }}>
          <MapPin className="w-[9px] h-[9px] text-white" strokeWidth={1.5} /><span className="text-white text-[10px]">{user.distance}</span>
        </div>
        <div className="absolute bottom-0 inset-x-0 flex items-end justify-between gap-2 px-3 pb-2.5">
          <div className="min-w-0"><p className="truncate text-white text-sm font-semibold">{user.name}</p><p className="truncate text-white/78 text-[10px] mt-0.5">{user.note}</p></div>
          {user.status === "在线" && <OnlineDot size="sm" className="relative static" />}
        </div>
      </div>
      <div className="px-3 pt-2.5 pb-3">
        <div className="flex items-center justify-between gap-2 mb-2.5">
          <TagPill size="sm" active>#{user.tag}</TagPill>
          <span className="truncate text-[var(--neu-text-muted)] text-[10px]">{user.mutual ? `${user.mutual} 位共同好友` : "等待认识"}</span>
        </div>
        <div className="flex items-center justify-between gap-2">
          <span className="text-[var(--neu-text-muted)] text-[10px]">{followed ? "已关注" : "感兴趣就打个招呼"}</span>
          <motion.button whileTap={{ scale: 0.9 }} onClick={(event) => { event.stopPropagation(); onFollow() }} className="flex items-center justify-center rounded-full outline-none flex-shrink-0 w-[30px] h-[30px]" aria-label={followed ? `取消关注 ${user.name}` : `关注 ${user.name}`} style={followed ? { background: "var(--neu-accent-soft)", border: "1px solid var(--neu-accent-soft-border)", color: "var(--neu-accent-primary)" } : { background: "var(--neu-accent-primary)", color: "white", boxShadow: "0 2px 8px var(--neu-glow)" }}>{followed ? <UserCheck size={15} /> : <UserPlus size={15} />}</motion.button>
        </div>
      </div>
    </motion.article>
  )
}

export function NearbyPeoplePage() {
  const router = useRouter()
  const { platform } = usePlatform()
  const { reference_point } = useDefaultLocation()
  const [search, setSearch] = useState("")
  const [sort, setSort] = useState("距离")
  const [tagFilter, setTagFilter] = useState("全部")
  const [followedIds, setFollowedIds] = useState<Set<number>>(new Set())
  const [visibleCount, setVisibleCount] = useState(12)
  const [allUsers, setAllUsers] = useState<NearbyUser[]>([])
  const [loading, setLoading] = useState(true)
  const isDesktop = platform === "desktop"
  const isTablet = platform === "tablet"

  useEffect(() => {
    if (!reference_point) return
    let cancelled = false
    async function fetchUsers() {
      try {
        setLoading(true)
        const res = await getNearbyUsers({ reference_point }).catch(() => ({ code: -1, data: { items: [] } }))
        if (cancelled) return
        const data = (res.code === 0 ? res.data : null) as any
        if (data) {
          const items: NearbyUser[] = (data.items || []).map((u: any, i: number) => ({
            id: u.user_id ?? i + 1,
            name: u.nickname ?? "",
            distance: u.distance ?? "",
            avatar: u.avatar_url ?? "",
            status: u.presence_status === "online" ? "在线" : "离线",
            tag: u.tags?.[0] ?? "",
            mutual: u.social_context?.mutual_friends_count ?? 0,
            note: u.bio ?? "",
          }))
          setAllUsers(items)
        }
      } catch {
        if (!cancelled) setAllUsers([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchUsers()
    return () => { cancelled = true }
  }, [reference_point])

  const list = useMemo(() => {
    const next = allUsers.filter((user) => (tagFilter === "全部" || user.tag === tagFilter) && (!search || `${user.name}${user.tag}${user.note}`.toLowerCase().includes(search.toLowerCase())))
    if (sort === "在线") return [...next].sort((a, b) => Number(b.status === "在线") - Number(a.status === "在线"))
    if (sort === "共同好友") return [...next].sort((a, b) => b.mutual - a.mutual)
    return next
  }, [search, sort, tagFilter, allUsers])

  const onlineUsers = allUsers.filter((user) => user.status === "在线")
  const onlineCount = list.filter((user) => user.status === "在线").length
  const contentStyle = isDesktop ? { width: "100%" } : isTablet ? { maxWidth: 964, width: "100%" } : { maxWidth: 680, width: "100%" }
  const gridColumns = platform === "mobile" ? 2 : isDesktop ? 3 : 2
  const toggleFollow = (id: number) => setFollowedIds((previous) => { const next = new Set(previous); next.has(id) ? next.delete(id) : next.add(id); return next })

  return <>
    <PageHeader
      title="附近的人"
      subtitle="徐州 · 5km 内 · 每 3 分钟刷新"
      sticky
      onBack={() => router.back()}
      actions={
        <ActionIconButton icon={Map} label="在地图中查看" onClick={() => router.push(ROUTES.map)} variant="ghost" tone="accent" size={34} />
      }
    >
      <div className="hidden sm:flex flex-1 max-w-sm ml-3">
        <SearchInput value={search} onChange={setSearch} placeholder="搜索昵称、兴趣或此刻状态" compact />
      </div>
    </PageHeader>

    {!isDesktop && (
      <div className="sm:hidden px-5 mt-0 -mt-2 pb-2">
        <SearchInput value={search} onChange={setSearch} placeholder="搜索昵称、兴趣或状态" compact />
      </div>
    )}

    <div className={isDesktop ? "px-7 py-6" : "px-5 py-5 mx-auto"} style={contentStyle}>
      <div className={isTablet ? "grid gap-6" : ""}>
        <div className="min-w-0">
          <OnlineStrip users={onlineUsers} onNavigate={(path) => router.push(path)} />
          <section className="neu-surface-hero rounded-[22px] p-4 mb-5 flex items-center gap-4">
            <div className="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-accent-primary)] w-[46px] h-[46px]" style={{ boxShadow: "0 4px 14px var(--neu-glow)" }}><Radar size={20} color="white" /></div>
            <div className="min-w-0 flex-1"><p className="text-[var(--neu-text)] text-sm font-semibold">此刻的徐州，正有 {onlineCount} 位新朋友在附近</p><p className="text-[var(--neu-text-muted)] text-[11px] mt-1">已按你的位置与兴趣筛出 {list.length} 人；打个招呼，从共同话题开始。</p></div>
            <button type="button" onClick={() => router.push(ROUTES.map)} className="hidden md:flex flex-shrink-0 items-center gap-1.5 px-3 py-2 rounded-full neu-surface-ghost neu-focus-ring text-[var(--neu-accent-primary)] text-[11px]"><Map size={13} />查看地图</button>
          </section>
          <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-3 mb-5">
            <div className="flex items-center gap-1 p-1 rounded-full neu-input-surface w-fit">
              {SORT_OPTIONS.map((option) => (
                <button key={option} type="button" onClick={() => setSort(option)} aria-pressed={sort === option} className={`px-3.5 py-1 rounded-full outline-none transition-colors text-xs focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${sort === option ? "bg-[var(--neu-accent-primary)] text-white font-semibold" : "neu-surface-ghost text-[var(--neu-text-muted)]"}`}>{option}</button>
              ))}
            </div>
            <div className="flex gap-1.5 overflow-x-auto no-scrollbar">
              {FILTER_TAGS.map((tag) => (
                <TagPill key={tag} active={tagFilter === tag} onClick={() => setTagFilter(tag)} size="md">{tag}</TagPill>
              ))}
            </div>
          </div>
          <div className="flex items-center justify-between mb-3"><span className="neu-eyebrow">为你排序</span><span className="flex items-center gap-1 text-[var(--neu-text-muted)] text-[11px]"><SlidersHorizontal size={12} />{list.length} 位可认识</span></div>
          <AnimatePresence mode="wait">
            {list.length ? (
              <motion.div key={`${sort}-${tagFilter}-${search}`} initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                {isDesktop ? (
                  <BspCardCanvas
                    items={list.slice(0, visibleCount)}
                    gap={24}
                    minRatio={0.68}
                    maxRatio={1.48}
                    minCardWidth={160}
                    minCardHeight={250}
                    minCanvasHeight={560}
                    ariaLabel="附近用户的不规则满铺布局"
                    hasMore={visibleCount < list.length}
                    onLoadMore={() => setVisibleCount((c) => Math.min(c + 8, list.length))}
                    renderItem={(user: NearbyUser, index: number) => <UserCard user={user} format={index % 4 === 1 ? "portrait" : "standard"} followed={followedIds.has(user.id)} onFollow={() => toggleFollow(user.id)} onNavigate={() => router.push(ROUTES.user(user.id))} />}
                  />
                ) : (
                  <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(${gridColumns}, minmax(0, 1fr))` }}>
                    {list.map((user, index) => <UserCard key={user.id} user={user} format={index === 0 ? "focus" : index % 5 === 2 ? "portrait" : "standard"} followed={followedIds.has(user.id)} onFollow={() => toggleFollow(user.id)} onNavigate={() => router.push(ROUTES.user(user.id))} />)}
                  </div>
                )}
              </motion.div>
            ) : (
              <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                <EmptyState icon={<Users size={24} color="var(--neu-text-muted)" opacity={0.4} />} title="没有找到匹配的附近用户" />
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </div>
    </div>
  </>
}
