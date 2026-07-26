import React, { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { ArrowLeft, Bell, CalendarDays, Check, MapPin, MessageCircle, Users } from "lucide-react";
import { motion } from "motion/react";
import { ActionIconButton } from "../components/shared/ActionIconButton";
import { ROUTES } from "../lib/routes";

type GroupTab = "overview" | "members" | "announcements";
type Group = { name: string; category: string; members: number; online: number; distance: string; description: string; banner: string };

const GROUP_DETAILS: Record<string, Group> = {
  "1": { name: "徐州街头摄影师联盟", category: "摄影", members: 143, online: 12, distance: "0.4km", description: "记录回龙窝、老街与云龙湖边的光影。每个人都能分享自己的城市视角。", banner: "https://images.unsplash.com/photo-1637354098670-9fb8b3d765fc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1200" },
  "2": { name: "云龙湖晨跑团", category: "户外", members: 89, online: 6, distance: "0.8km", description: "每天清晨从云龙湖西门出发，用稳定的步伐认识这座城市。", banner: "https://images.unsplash.com/photo-1758599669009-5a9002c09487?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1200" },
};

const MEMBERS = [
  { id: 1, name: "鹿鸣", role: "群主", online: true, avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=120" },
  { id: 2, name: "阿禾", role: "管理员", online: true, avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=120" },
  { id: 3, name: "野原", role: "成员", online: false, avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=120" },
  { id: 4, name: "Mika", role: "成员", online: true, avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=120" },
];

const ANNOUNCEMENTS = [
  { title: "本周主题：傍晚的路口", body: "周末请带着一张你在黄昏时拍下的城市照片，一起聊聊光线与等待。", time: "2 小时前" },
  { title: "群内发布规范", body: "请尊重被拍摄者，发布街拍作品前优先确认肖像使用范围。", time: "周一" },
];

export function GroupDetail() {
  const { groupId } = useParams<{ groupId: string }>();
  const navigate = useNavigate();
  const [joined, setJoined] = useState(false);
  const [tab, setTab] = useState<GroupTab>("overview");
  const group = GROUP_DETAILS[groupId ?? ""] ?? {
    name: "附近兴趣群组", category: "同城", members: 64, online: 8, distance: "附近",
    description: "这里汇聚了正在探索城市的人。加入后可以查看群公告、成员与群聊。",
    banner: "https://images.unsplash.com/photo-1543269865-cbf427effbad?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=1200",
  };

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
          {tab === "members" && <div className="divide-y divide-[var(--neu-border)]">{MEMBERS.map((member) => <button type="button" key={member.id} onClick={() => navigate(ROUTES.user(member.id))} className="app-list-row app-list-row--standard neu-row-hover rounded-[12px] outline-none"><div className="relative"><img src={member.avatar} alt="" className="h-10 w-10 rounded-full object-cover" />{member.online && <span className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-[var(--neu-bg)] bg-[var(--semantic-online)]" />}</div><span className="min-w-0 flex-1"><span className="block text-[14px] text-[var(--neu-text)]">{member.name}</span><span className="text-[11px] text-[var(--neu-text-muted)]">{member.role}</span></span><span className="text-[11px] text-[var(--neu-text-muted)]">查看资料</span></button>)}</div>}
          {tab === "announcements" && <div className="divide-y divide-[var(--neu-border)]">{ANNOUNCEMENTS.map((item) => <article key={item.title} className="py-4"><p className="text-[14px] font-medium text-[var(--neu-text)]">{item.title}</p><p className="mt-1 text-[12px] leading-5 text-[var(--neu-text-muted)]">{item.body}</p><p className="mt-2 text-[11px] text-[var(--neu-text-muted)]">{item.time}</p></article>)}</div>}
        </main>
        <aside className="flex h-fit flex-col gap-3 rounded-[20px] border border-[var(--neu-border)] bg-[var(--neu-bg)] p-4"><motion.button type="button" whileTap={{ scale: 0.98 }} onClick={() => setJoined((value) => !value)} className={`w-full rounded-full px-4 py-3 text-[13px] font-medium outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${joined ? "border border-[var(--neu-accent-primary)] text-[var(--neu-accent-primary)]" : "bg-[var(--neu-accent-primary)] text-white"}`}>{joined ? <span className="flex items-center justify-center gap-1.5"><Check size={15} />已加入 · 点击退出</span> : "加入群组"}</motion.button>{joined ? <button type="button" onClick={() => navigate(ROUTES.messages)} className="flex w-full items-center justify-center gap-2 rounded-full border border-[var(--neu-border)] px-4 py-3 text-[13px] text-[var(--neu-text)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"><MessageCircle size={15} />打开群聊</button> : <p className="px-2 text-center text-[11px] leading-5 text-[var(--neu-text-muted)]"><Bell size={13} className="mr-1 inline" />加入后可管理群组通知并进入群聊</p>}</aside>
      </div>
    </div>
  );
}
