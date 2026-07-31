'use client'
import React, { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ROUTES } from "@/lib/routes";
import { Heart, UserPlus, MessageCircle, Bell, MapPin, CheckCheck, Filter } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useHoverConfig } from "@/hooks/useHoverConfig";
import { usePlatform } from "@/contexts/PlatformContext";
import { DesktopDetailSplit, RailSection, RailUserRow } from "@/components/layout/DetailRail";
import { PageHeader } from "@/components/shared/PageHeader";
import { TagPill } from "@/components/shared/TagPill";
import { EmptyState } from "@/components/shared/EmptyState";
import { listNotifications, getUnreadCount, markRead } from "@/api";

/* ── Types & data ───────────────────────────────────────────────── */
type NotifType = "like" | "follow" | "comment" | "nearby" | "event";

interface Notif {
  id: string; type: string; read: boolean; time: string;
  user: { name: string; avatar: string };
  text: string; postImage?: string;
}

const TYPE_CONFIG: Record<string, { icon: React.ElementType; color: string; label: string }> = {
  like:    { icon: Heart,          color: "var(--semantic-hot)", label: "点赞" },
  follow:  { icon: UserPlus,       color: "var(--neu-accent-primary)", label: "关注" },
  comment: { icon: MessageCircle,  color: "var(--map-user)", label: "评论" },
  nearby:  { icon: MapPin,         color: "var(--semantic-online)", label: "附近" },
  event:   { icon: Bell,           color: "var(--semantic-warning)", label: "活动" },
};

const FILTER_TABS = [
  { id: "all",     label: "全部" },
  { id: "like",    label: "点赞" },
  { id: "follow",  label: "关注" },
  { id: "comment", label: "评论" },
  { id: "nearby",  label: "附近" },
];

export function NotificationsPage() {
  const router = useRouter();
  const { platform } = usePlatform();
  const [notifs, setNotifs] = useState<Notif[]>([]);
  const [filter, setFilter] = useState("all");
  const [loading, setLoading] = useState(true);

  const unread = notifs.filter(n => !n.read).length;
  const isDesktop = platform === "desktop";

  useEffect(() => {
    let cancelled = false;
    async function fetchNotifs() {
      try {
        setLoading(true);
        const [notifRes, unreadRes] = await Promise.all([
          listNotifications().catch(() => ({ code: -1, data: { items: [] } })),
          getUnreadCount().catch(() => ({ code: -1, data: { total: 0 } })),
        ]);
        if (cancelled) return;

        const notifData = (notifRes.code === 0 ? notifRes.data : null) as any;
        const unreadData = (unreadRes.code === 0 ? unreadRes.data : null) as any;

        if (notifData) {
          const items: Notif[] = (notifData.items || []).map((n: any, i: number) => {
            const actor = n.payload?.actor ?? n.actor ?? {};
            let text = "";
            if (n.type === "moment_liked") text = `${actor.nickname ?? ""} 赞了你的动态`;
            else if (n.type === "moment_commented") text = n.payload?.comment_preview ?? `${actor.nickname ?? ""} 评论了你的动态`;
            else if (n.type === "new_follower") text = `${actor.nickname ?? ""} 关注了你`;
            else if (n.type === "group_invite_received") text = `${actor.nickname ?? ""} 邀请你加入 ${n.payload?.group_name ?? "群组"}`;
            else if (n.type === "reaction_added") text = `${actor.nickname ?? ""} 做出了反应 ${n.payload?.emoji ?? ""}`;
            else if (n.type === "system_announcement") text = "系统公告";
            else text = n.payload?.title ?? n.message ?? "";
            return {
              id: n.notification_id ?? String(i + 1),
              type: n.type ?? "like",
              read: n.is_read ?? false,
              time: n.created_at ?? "",
              user: { name: actor.nickname ?? "", avatar: actor.avatar_url ?? "" },
              text,
              postImage: n.payload?.moment_summary?.cover_url ?? n.payload?.cover_url ?? undefined,
            };
          });
          setNotifs(items);
        }
      } catch {
        if (!cancelled) setNotifs([]);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchNotifs();
    return () => { cancelled = true; };
  }, []);

  const markAllRead = () => {
    // spec: scope='all' clears all unread. Update local state optimistically.
    setNotifs(prev => prev.map(n => ({ ...n, read: true })));
    markRead({ scope: "all" }).catch(() => {});
  };
  const markSingleRead = (id: string) => {
    setNotifs(prev => prev.map(n => n.id === id ? { ...n, read: true } : n));
    // spec: scope='selected' with notification_ids (one of notification_ids|type).
    markRead({ scope: "selected", notification_ids: [id] }).catch(() => {});
  };
  const remove = (id: string) => setNotifs(prev => prev.filter(n => n.id !== id));

  const filtered = notifs.filter(n => filter === "all" || n.type === filter);
  const unreadGroup = filtered.filter(n => !n.read);
  const readGroup   = filtered.filter(n => n.read);

  const byType = (t: NotifType) => notifs.filter(n => n.type === t).length;
  const typeStats: { type: NotifType; count: number }[] = (Object.keys(TYPE_CONFIG) as NotifType[])
    .map(t => ({ type: t, count: byType(t) }));

  const main = (
    /* main scroll container */
    <div className="h-full overflow-y-auto no-scrollbar">
      <PageHeader
        title="通知"
        sticky
        onBack={() => router.back()}
        subtitle={unread > 0 ? `${unread} 条未读` : undefined}
        actions={unread > 0 ? (
          <motion.button whileTap={{ scale: 0.93 }} onClick={markAllRead}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none bg-[var(--neu-bg)] text-[var(--neu-accent-primary)] text-[12px]"
            style={{ border: "1px solid var(--neu-border)", boxShadow: "3px 3px 8px var(--neu-shadow-dark), -3px -3px 8px var(--neu-shadow-light)" }}>
            <CheckCheck style={{ width: 13, height: 13 }} strokeWidth={2} />
            全部已读
          </motion.button>
        ) : undefined}
      />

      {/* Filter tabs */}
      <div className="flex overflow-x-auto no-scrollbar pb-3 gap-2 px-5"
        style={{ maxWidth: isDesktop ? "var(--content-reading-max)" : 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        {FILTER_TABS.map(t => (
          <TagPill key={t.id} active={filter === t.id} onClick={() => setFilter(t.id)} size="sm">
            {t.label}
          </TagPill>
        ))}
      </div>

      {/* List */}
      <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} className="flex flex-col px-5 pt-3 pb-6 gap-0"
        style={{ maxWidth: isDesktop ? "var(--content-reading-max)" : 640, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        {/* Unread group */}
        {unreadGroup.length > 0 && (
          <>
            <p className="uppercase mb-2 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">新消息</p>
            {unreadGroup.map(n => <NotifRow key={n.id} notif={n} onRead={markSingleRead} onRemove={remove} navigate={router.push} />)}
            {readGroup.length > 0 && <p className="uppercase mt-4 mb-2 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">较早</p>}
          </>
        )}

        {/* Read group */}
        <AnimatePresence>
          {readGroup.map(n => <NotifRow key={n.id} notif={n} onRead={markSingleRead} onRemove={remove} navigate={router.push} />)}
        </AnimatePresence>

        {filtered.length === 0 && (
          <EmptyState
            icon={<Bell style={{ width: 24, height: 24, color: "var(--neu-text-muted)", opacity: 0.5 }} strokeWidth={1.25} />}
            title="暂无通知"
          />
        )}
      </motion.div>
    </div>
  );

  const rail = (
    <>
      <RailSection title="今日概览">
        <div className="flex items-baseline gap-1.5 mb-3">
          <span className="text-[var(--neu-text)] text-[30px] font-semibold leading-none">{unread}</span>
          <span className="text-[var(--neu-text-muted)] text-[11px]">条未读 · 共 {notifs.length} 条</span>
        </div>
        <div className="flex flex-col gap-2">
          {typeStats.map(({ type, count }) => {
            const cfg = TYPE_CONFIG[type];
            const Icon = cfg.icon;
            return (
              <motion.button
                key={type}
                whileTap={{ scale: 0.98 }}
                onClick={() => setFilter(type)}
                className="flex items-center gap-2.5 rounded-[10px] px-2 py-1.5 outline-none text-left"
                style={{
                  background: filter === type ? `color-mix(in srgb, ${cfg.color} 10%, transparent)` : "transparent",
                  border: `1px solid ${filter === type ? `color-mix(in srgb, ${cfg.color} 22%, transparent)` : "transparent"}`,
                  transition: "background 0.15s",
                }}
              >
                <div
                  className="flex items-center justify-center rounded-full flex-shrink-0"
                  style={{ width: 24, height: 24, background: `color-mix(in srgb, ${cfg.color} 16%, transparent)` }}
                >
                  <Icon style={{ width: 11, height: 11, color: cfg.color }} strokeWidth={1.75} />
                </div>
                <span className="text-[var(--neu-text)] text-[12px] flex-1">{cfg.label}</span>
                <span className="text-[var(--neu-text-muted)] text-[11px]">{count}</span>
              </motion.button>
            );
          })}
        </div>
      </RailSection>

      <RailSection title="新关注你的人" action="全部" onAction={() => router.push(`${ROUTES.meSection("relationships")}?tab=followers`)}>
        <div className="flex flex-col gap-3">
          {notifs.filter(n => n.type === "follow").slice(0, 3).map(n => (
            <RailUserRow
              key={n.id}
              avatar={n.user.avatar}
              name={n.user.name}
              sub={n.time}
              trailing={
                <motion.button
                  whileTap={{ scale: 0.94 }}
                  className="px-2.5 py-1 rounded-full outline-none flex-shrink-0 bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] text-[11px]"
                  style={{ border: "1px solid var(--neu-accent-soft-border)" }}
                >
                  回关
                </motion.button>
              }
            />
          ))}
        </div>
      </RailSection>

      <RailSection title="此刻附近" action="查看" onAction={() => router.push(ROUTES.nearbyPeople)}>
        <div className="flex flex-col gap-3">
          {notifs.filter(n => n.type === "nearby").slice(0, 3).map(n => (
            <RailUserRow
              key={n.id}
              avatar={n.user.avatar}
              name={n.user.name}
              sub={n.text}
            />
          ))}
        </div>
      </RailSection>
    </>
  );

  return <DesktopDetailSplit main={main} rail={rail} />;
}

function NotifRow({ notif, onRead, onRemove, navigate }: {
  notif: Notif;
  onRead: (id: string) => void;
  onRemove: (id: string) => void;
  navigate: (path: string) => void;
}) {
  const cfg = TYPE_CONFIG[notif.type];
  const Icon = cfg.icon;
  const cfgTone = (amount: number) => `color-mix(in srgb, ${cfg.color} ${amount}%, transparent)`;
  const hover = useHoverConfig();
  return (
    <motion.div
      layout
      initial={{ opacity: 0, x: -12 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 12, height: 0, marginBottom: 0 }}
      transition={{ duration: 0.22 }}
      onClick={() => onRead(notif.id)}
      whileTap={hover.tap as any}
      whileHover={hover.row as any}
      className="app-list-row app-list-row--rich cursor-pointer rounded-[18px] mb-1.5 px-3 neu-row-hover"
      style={{
        background: notif.read ? "transparent" : cfgTone(7),
        border: notif.read ? "1px solid transparent" : `1px solid ${cfgTone(18)}`,
      }}
    >
      {/* Avatar + icon badge */}
      <div className="relative flex-shrink-0">
        {notif.user.avatar ? (
          <div className="rounded-full overflow-hidden" style={{ width: 44, height: 44 }}>
            <img src={notif.user.avatar} alt={notif.user.name} className="w-full h-full object-cover" />
          </div>
        ) : (
          <div className="rounded-full flex items-center justify-center" style={{ width: 44, height: 44, background: cfgTone(18) }}>
            <Icon style={{ width: 18, height: 18, color: cfg.color }} strokeWidth={1.75} />
          </div>
        )}
        {notif.user.avatar && (
          <div className="absolute -bottom-0.5 -right-0.5 flex items-center justify-center rounded-full"
            style={{ width: 18, height: 18, background: cfg.color, border: "2px solid var(--neu-bg)" }}>
            <Icon style={{ width: 9, height: 9, color: "white" }} strokeWidth={2.5} />
          </div>
        )}
      </div>

      {/* Text */}
      <div className="flex-1 min-w-0">
        <p className="truncate text-[var(--neu-text)] text-[13px]">
          <span style={{ fontWeight: notif.read ? 400 : 500 }}>{notif.user.name}</span>
          {" "}{notif.text}
        </p>
        <p className="text-[var(--neu-text-muted)] text-[11px] mt-0.5">{notif.time}</p>
      </div>

      {/* Post thumb or unread dot */}
      <div className="flex items-center gap-2 flex-shrink-0">
        {notif.postImage && (
          <div className="rounded-[10px] overflow-hidden" style={{ width: 38, height: 38 }}>
            <img src={notif.postImage} alt="" className="w-full h-full object-cover" />
          </div>
        )}
        {!notif.read && <div className="rounded-full" style={{ width: 8, height: 8, background: cfg.color }} />}
      </div>
    </motion.div>
  );
}