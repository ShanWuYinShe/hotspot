import React, { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { ArrowLeft, Bell, CalendarDays, Check, MapPin, MessageCircle, Users } from "lucide-react";
import { motion } from "motion/react";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { ROUTES } from "../lib/routes";
import { getGroupDetail, getGroupMembers, getGroupAnnouncements } from "../../api";

type GroupTab = "overview" | "members" | "announcements";

interface Group {
  name: string; category: string; members: number; online: number;
  distance: string; description: string; banner: string;
}

interface Member {
  id: number; name: string; role: string; online: boolean; avatar: string;
}

interface Announcement {
  title: string; body: string; time: string;
}

export function GroupDetail() {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const [joined, setJoined] = useState(false);
  const [tab, setTab] = useState<GroupTab>("overview");
  const [group, setGroup] = useState<Group>({
    name: "", category: "", members: 0, online: 0, distance: "",
    description: "",
    banner: "",
  });
  const [members, setMembers] = useState<Member[]>([]);
  const [announcements, setAnnouncements] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    const gid = groupId ?? "";
    async function fetchGroup() {
      try {
        setLoading(true);
        const [detailRes, membersRes, announcementsRes] = await Promise.all([
          getGroupDetail(gid).catch(() => ({ code: -1, data: null })),
          getGroupMembers(gid).catch(() => ({ code: -1, data: { items: [] } })),
          getGroupAnnouncements(gid).catch(() => ({ code: -1, data: { items: [] } })),
        ]);
        if (cancelled) return;

        const detailData = (detailRes.code === 0 ? detailRes.data : null) as any;
        const membersData = (membersRes.code === 0 ? membersRes.data : null) as any;
        const announcementsData = (announcementsRes.code === 0 ? announcementsRes.data : null) as any;

        if (detailData) {
          const g = detailData.group ?? detailData;
          setGroup({
            name: g.name ?? group.name,
            category: (detailData.tags?.[0] ?? g.category ?? group.category) || "同城",
            members: g.members_count ?? group.members,
            online: g.online_count ?? group.online,
            distance: detailData.location?.distance ?? group.distance,
            description: detailData.description ?? group.description,
            banner: g.avatar_url ?? group.banner,
          });
          if (detailData.my_state?.is_joined) setJoined(true);
        }

        if (membersData) {
          const items: Member[] = (membersData.items || []).map((m: any, i: number) => ({
            id: m.user?.user_id ?? i + 1,
            name: m.user?.nickname ?? m.name ?? "",
            role: m.role ?? "成员",
            online: m.user?.presence_status === "online" || m.online || false,
            avatar: m.user?.avatar_url ?? m.avatar ?? "",
          }));
          setMembers(items);
        }

        if (announcementsData) {
          const items: Announcement[] = (announcementsData.items || []).map((a: any) => ({
            title: a.title ?? a.name ?? "",
            body: a.body ?? a.content ?? a.text ?? "",
            time: a.time ?? a.created_at ?? "",
          }));
          setAnnouncements(items);
        }
      } catch {
        // empty data is normal
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchGroup();
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [groupId]);

  return (
    <div className="content-mode-wide min-h-full bg-[var(--neu-bg)]">
      <div className="relative h-[clamp(220px,28vw,340px)] overflow-hidden">
        <img src={group.banner} alt={`${group.name} 的群组封面`} className="h-full w-full object-cover" />
        <div className="absolute inset-0 bg-[linear-gradient(to_top,rgba(0,0,0,.7),transparent_65%)]" />
        <div className="absolute left-5 top-4"><ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} variant="overlay" tone="inverse" /></div>
        <div className="absolute bottom-5 left-6 right-6"><span className="rounded-full bg-white/18 px-2.5 py-1 text-[11px] text-white backdrop-blur-sm">{group.category}</span><h1 className="mt-3 text-[clamp(24px,3vw,34px)] font-medium tracking-[-0.02em] text-white">{group.name}</h1></div>
      </div>

      <div className="grid gap-8 px-6 py-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:px-8">
        <main className="min-w-0">
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-[13px] text-[var(--neu-text-muted)]"><span className="flex items-center gap-1.5"><Users size={14} />{group.members} 位成员</span><span className="flex items-center gap-1.5 text-[var(--semantic-online)]"><span className="h-2 w-2 rounded-full bg-[var(--semantic-online)]" />{group.online} 人在线</span><span className="flex items-center gap-1.5"><MapPin size={14} />距你 {group.distance}</span></div>
          <div className="mt-6 flex gap-1 border-b border-[var(--neu-border)]">{(["overview", "members", "announcements"] as GroupTab[]).map((item) => <button key={item} type="button" onClick={() => setTab(item)} className={`relative px-3 py-2.5 text-[13px] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${tab === item ? "text-[var(--neu-accent-primary)]" : "text-[var(--neu-text-muted)]"}`}>{item === "overview" ? "简介" : item === "members" ? "成员" : "公告"}{tab === item && <span className="absolute inset-x-3 bottom-0 h-0.5 rounded-full bg-[var(--neu-accent-primary)]" />}</button>)}</div>
          {tab === "overview" && <div className="py-6"><p className="max-w-3xl text-[15px] leading-7 text-[var(--neu-text-muted)]">{group.description}</p><div className="mt-7 rounded-[18px] border border-[var(--neu-border)] p-4"><p className="flex items-center gap-2 text-[14px] font-medium text-[var(--neu-text)]"><CalendarDays size={15} className="text-[var(--neu-accent-primary)]" />群组节奏</p><p className="mt-2 text-[12px] leading-5 text-[var(--neu-text-muted)]">每周三、周末晚间有主题讨论。加入后可接收新公告与群聊提醒。</p></div></div>}
          {tab === "members" && <div className="divide-y divide-[var(--neu-border)]">{members.map((member) => <button type="button" key={member.id} onClick={() => navigate(ROUTES.user(member.id))} className="app-list-row app-list-row--standard neu-row-hover rounded-[12px] outline-none"><div className="relative"><img src={member.avatar} alt="" className="h-10 w-10 rounded-full object-cover" />{member.online && <span className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-[var(--neu-bg)] bg-[var(--semantic-online)]" />}</div><span className="min-w-0 flex-1"><span className="block text-[14px] text-[var(--neu-text)]">{member.name}</span><span className="text-[11px] text-[var(--neu-text-muted)]">{member.role}</span></span><span className="text-[11px] text-[var(--neu-text-muted)]">查看资料</span></button>)}</div>}
          {tab === "announcements" && <div className="divide-y divide-[var(--neu-border)]">{announcements.map((item) => <article key={item.title} className="py-4"><p className="text-[14px] font-medium text-[var(--neu-text)]">{item.title}</p><p className="mt-1 text-[12px] leading-5 text-[var(--neu-text-muted)]">{item.body}</p><p className="mt-2 text-[11px] text-[var(--neu-text-muted)]">{item.time}</p></article>)}</div>}
        </main>
        <aside className="flex h-fit flex-col gap-3 rounded-[20px] border border-[var(--neu-border)] bg-[var(--neu-bg)] p-4"><motion.button type="button" whileTap={{ scale: 0.98 }} onClick={() => setJoined((value) => !value)} className={`w-full rounded-full px-4 py-3 text-[13px] font-medium outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${joined ? "border border-[var(--neu-accent-primary)] text-[var(--neu-accent-primary)]" : "bg-[var(--neu-accent-primary)] text-white"}`}>{joined ? <span className="flex items-center justify-center gap-1.5"><Check size={15} />已加入 · 点击退出</span> : "加入群组"}</motion.button>{joined ? <button type="button" onClick={() => navigate(ROUTES.messages)} className="flex w-full items-center justify-center gap-2 rounded-full border border-[var(--neu-border)] px-4 py-3 text-[13px] text-[var(--neu-text)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"><MessageCircle size={15} />打开群聊</button> : <p className="px-2 text-center text-[11px] leading-5 text-[var(--neu-text-muted)]"><Bell size={13} className="mr-1 inline" />加入后可管理群组通知并进入群聊</p>}</aside>
      </div>
    </div>
  );
}
