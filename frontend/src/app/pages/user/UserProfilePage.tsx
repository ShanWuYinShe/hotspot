import React, { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router";
import { ROUTES } from "../../lib/routes";
import { ActionIconButton } from "../../components/shared/ActionIconButton";
import { ArrowLeft, MapPin, MessageCircle, UserPlus, UserCheck, Share2, Heart, Grid3X3, Bookmark, MoreHorizontal, Flag, PencilLine, UserX, X, Check } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import {
  DesktopDetailSplit, RailSection, RailUserRow, RailPostThumb, RailLocationMini,
} from "../../components/layout/DetailRail";
import { getProfile, follow, block, createReport } from "../../../api";

const MUTUAL_FRIENDS: { id: number; name: string; sub: string; avatar: string }[] = [];

const SIMILAR_USERS: { id: number; name: string; sub: string; avatar: string }[] = [];

interface UserData {
  name: string; handle: string; bio: string; location: string;
  avatar: string; banner: string; followers: number; following: number; posts: number;
  gallery: string[]; tags: string[];
}

const DEFAULT_USER: UserData = {
  name: "", handle: "", bio: "", location: "",
  avatar: "", banner: "", followers: 0, following: 0, posts: 0,
  gallery: [], tags: [],
};

type Tab = "posts" | "liked" | "saved";

export function UserProfilePage() {
  const { userId } = useParams<{ userId: string }>();
  const navigate = useNavigate();
  const [user, setUser] = useState<UserData>(DEFAULT_USER);
  const [loading, setLoading] = useState(true);
  const [mutualFriends, setMutualFriends] = useState<typeof MUTUAL_FRIENDS>([]);
  const [similarUsers, setSimilarUsers] = useState<typeof SIMILAR_USERS>([]);
  const [followed, setFollowed] = useState(false);
  const [activeTab, setActiveTab] = useState<Tab>("posts");
  const [moreOpen, setMoreOpen] = useState(false);
  const [blocked, setBlocked] = useState(false);
  const [remark, setRemark] = useState("");
  const [remarkDraft, setRemarkDraft] = useState("");
  const [reportSent, setReportSent] = useState(false);
  const [shareCopied, setShareCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const uid = userId ?? "";
    async function fetchProfile() {
      try {
        setLoading(true);
        const res = await getProfile(uid).catch(() => ({ code: -1, data: null }));
        if (cancelled) return;
        const data = (res.code === 0 ? res.data : null) as any;
        if (data) {
          const u = data.user ?? data;
          const s = data.stats ?? {};
          const r = data.relation ?? {};
          setUser({
            name: u.nickname ?? data.nickname ?? "",
            handle: u.user_id ? `@${(u.nickname ?? "").replace(/\s/g, "_")}` : "",
            bio: u.bio ?? data.bio ?? "",
            location: u.location?.location_name ?? data.location?.location_name ?? "",
            avatar: u.avatar_url ?? data.avatar ?? "",
            banner: u.cover_url ?? data.banner ?? "",
            followers: s.followers_count ?? 0,
            following: s.following_count ?? 0,
            posts: s.moments_count ?? 0,
            gallery: data.recent_moments?.items?.map((m: any) => m.images?.[0]?.url ?? "") ?? [],
            tags: u.tags ?? data.tags ?? [],
          });
          if (r.is_following !== undefined) setFollowed(r.is_following);
          if (r.is_blocked !== undefined) setBlocked(r.is_blocked);
          if (data.remark) setRemark(data.remark);

          const mutual = u.social_context?.mutual_friends_preview ?? data.mutual_friends;
          if (mutual) {
            setMutualFriends(mutual.map((f: any) => ({
              id: f.user_id ?? f.id ?? 0, name: f.nickname ?? f.name ?? "", sub: "", avatar: f.avatar_url ?? f.avatar ?? "",
            })));
          }
        }
      } catch {
        // empty data is normal
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    fetchProfile();
    return () => { cancelled = true; };
  }, [userId]);

  const fmtNum = (n: number) => n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);

  const main = (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)", overflowY: "auto" }}>
      <AnimatePresence>
        {moreOpen && <>
          <motion.button type="button" aria-label="关闭更多操作" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} onClick={() => setMoreOpen(false)} className="fixed inset-0 z-40 cursor-default bg-black/40" />
          <motion.section initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: 20 }} role="dialog" aria-modal="true" aria-label={`${user.name} 的更多操作`} className="fixed inset-x-4 bottom-4 z-50 mx-auto max-w-md rounded-[24px] ring-1 ring-[var(--neu-border)] bg-[var(--neu-bg)] p-5 shadow-[0_24px_80px_rgba(0,0,0,.28)]">
            <div className="flex items-center justify-between"><div><p className="neu-eyebrow">关系与安全</p><p className="mt-1 text-[16px] text-[var(--neu-text)]">{user.name}</p></div><ActionIconButton icon={X} label="关闭" onClick={() => setMoreOpen(false)} size={32} /></div>
            <label className="mt-5 block text-[12px] text-[var(--neu-text-muted)]">备注名称<input value={remarkDraft} onChange={(event) => setRemarkDraft(event.target.value.slice(0, 24))} placeholder="只对你可见" className="mt-2 w-full rounded-[14px] ring-1 ring-[var(--neu-border)] bg-transparent px-3 py-2.5 text-[13px] text-[var(--neu-text)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]" /></label>
            <button type="button" onClick={() => { setRemark(remarkDraft.trim()); setMoreOpen(false); }} className="mt-2 flex w-full items-center justify-center gap-2 rounded-[14px] ring-1 ring-[var(--neu-border)] py-2.5 text-[13px] text-[var(--neu-text)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"><PencilLine size={14} />保存备注</button>
                <button type="button" onClick={() => {
                  const next = !blocked;
                  setBlocked(next);
                  setFollowed(false);
                  setMoreOpen(false);
                  // spec: block/unblock uses user_id + active boolean.
                  block({ user_id: userId ?? "", active: next }).catch(() => {
                    setBlocked(!next);
                    setFollowed(next);
                  });
                }} className="mt-2 flex w-full items-center justify-center gap-2 rounded-[14px] py-2.5 text-[13px] text-[var(--semantic-hot)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--semantic-hot)]"><UserX size={14} />{blocked ? "解除屏蔽" : "屏蔽用户"}</button>
                <button type="button" onClick={() => {
                  createReport({ target_type: "user", target_id: userId ?? "", reason: "other" }).then(() => setReportSent(true)).catch(() => {});
                  setMoreOpen(false);
                }} className="mt-1 flex w-full items-center justify-center gap-2 rounded-[14px] py-2.5 text-[13px] text-[var(--neu-text-muted)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]">{reportSent ? <><Check size={14} />举报已记录</> : <><Flag size={14} />举报用户</>}</button>
          </motion.section>
        </>}
      </AnimatePresence>
      {/* Banner */}
      <div className="relative overflow-hidden" style={{ height: "clamp(150px, 25vw, 210px)", flexShrink: 0 }}>
        <img src={user.banner} alt="banner" className="absolute inset-0 w-full h-full object-cover" />
        <div className="absolute inset-0" style={{ background: "linear-gradient(to bottom, rgba(0,0,0,0.15) 0%, rgba(0,0,0,0.55) 100%)" }} />

        {/* Top buttons */}
        <div className="absolute top-0 left-0 right-0 flex items-center justify-between px-4 pt-4">
          <ActionIconButton icon={ArrowLeft} label="返回" onClick={() => navigate(-1)} variant="overlay" tone="inverse" />
          <div className="flex gap-2">
            <ActionIconButton icon={Share2} label={shareCopied ? "链接已复制" : "分享资料"} onClick={() => { navigator.clipboard?.writeText(`${window.location.origin}${ROUTES.user(userId ?? "")}`); setShareCopied(true); window.setTimeout(() => setShareCopied(false), 1600); }} variant="overlay" tone="inverse" />
            <ActionIconButton icon={MoreHorizontal} label="更多操作" onClick={() => { setRemarkDraft(remark); setMoreOpen(true); }} variant="overlay" tone="inverse" />
          </div>
        </div>
      </div>

      {/* Avatar + info — constrained width, centered */}
      <div style={{ maxWidth: "var(--content-reading-max)", marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        <div className="px-5 -mt-10 pb-4 flex flex-col">
          <div className="flex items-end justify-between mb-3">
            <div className="relative">
              <div className="rounded-[22px] overflow-hidden"
                style={{ width: 76, height: 76, border: "3px solid var(--neu-bg)", boxShadow: "0 6px 20px rgba(0,0,0,0.22)" }}>
                <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
              </div>
              <span className="absolute bottom-1 right-1 rounded-full"
                style={{ width: 12, height: 12, background: "var(--semantic-online)", border: "2px solid var(--neu-bg)" }} />
            </div>

            {/* Action buttons */}
            <div className="flex gap-2 pb-1">
              <motion.button whileTap={{ scale: 0.93 }}
                onClick={() => !blocked && navigate(ROUTES.message(1))}
                disabled={blocked}
                className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none disabled:cursor-not-allowed disabled:opacity-45"
                style={{ background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)", fontSize: 12, color: "var(--neu-text-muted)" }}>
                <MessageCircle style={{ width: 14, height: 14 }} strokeWidth={1.5} />
                私信
              </motion.button>
              <motion.button whileTap={{ scale: 0.93 }}
                onClick={() => {
                  if (blocked) return;
                  const next = !followed;
                  setFollowed(next);
                  // spec: follow/unfollow uses user_id + active boolean.
                  follow({ user_id: userId ?? "", active: next }).catch(() => setFollowed(!next));
                }}
                disabled={blocked}
                className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all disabled:cursor-not-allowed disabled:opacity-45"
                style={followed
                  ? { background: "var(--neu-bg)", border: "1.5px solid var(--neu-accent-primary)", color: "var(--neu-accent-primary)", fontSize: 12, boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)" }
                  : { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 4px 14px var(--neu-glow)", border: "1px solid transparent" }}>
                {followed
                  ? <><UserCheck style={{ width: 14, height: 14 }} strokeWidth={1.5} />已关注</>
                  : <><UserPlus style={{ width: 14, height: 14 }} strokeWidth={1.5} />关注</>}
              </motion.button>
            </div>
          </div>

          <h1 style={{ fontSize: 20, color: "var(--neu-text)" }}>{user.name}</h1>
          <p style={{ fontSize: 13, color: "var(--neu-text-muted)", marginTop: 1 }}>{user.handle}{remark ? ` · ${remark}` : ""}</p>
          {blocked && <div className="mt-3 flex items-center gap-2 rounded-[12px] ring-1 ring-[var(--semantic-hot)]/30 bg-[color-mix(in_srgb,var(--semantic-hot)_8%,transparent)] px-3 py-2 text-[12px] text-[var(--semantic-hot)]"><UserX size={14} />你已屏蔽此用户；关注和私信已停用。</div>}
          <p style={{ fontSize: 13, color: "var(--neu-text-muted)", lineHeight: 1.65, marginTop: 8 }}>{user.bio}</p>

          <div className="flex items-center gap-1.5 mt-2">
            <MapPin style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{user.location}</span>
          </div>

          {/* Tags */}
          <div className="flex gap-2 mt-3 flex-wrap">
            {user.tags.map(t => (
              <span key={t} className="px-2.5 py-0.5 rounded-full"
                style={{ fontSize: 11, background: "var(--neu-accent-soft)", color: "var(--neu-accent-primary)", border: "1px solid var(--neu-accent-soft-border)" }}>
                #{t}
              </span>
            ))}
          </div>

          {/* Stats */}
          <div className="flex gap-0 mt-5 rounded-[20px]"
            style={{ background: "var(--neu-bg)", boxShadow: "0 0 0 1px var(--neu-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)" }}>
            <div style={{ overflow: "hidden", borderRadius: 19 }}>
            {[
              { label: "帖子",  v: fmtNum(user.posts) },
              { label: "粉丝",  v: fmtNum(user.followers) },
              { label: "关注",  v: fmtNum(user.following) },
            ].map((s, i) => (
              <div key={s.label} className="flex-1 flex flex-col items-center py-4 relative">
                {i > 0 && <div className="absolute left-0 top-1/2 -translate-y-1/2" style={{ width: 1, height: 28, background: "var(--neu-border)" }} />}
                <span style={{ fontSize: 22, color: "var(--neu-text)" }}>{s.v}</span>
                <span className="uppercase mt-1" style={{ fontSize: 9, letterSpacing: "0.1em", color: "var(--neu-text-muted)" }}>{s.label}</span>
              </div>
            ))}
            </div>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex" style={{ borderBottom: "1px solid var(--neu-border)", paddingLeft: 16, paddingRight: 16, flexShrink: 0 }}>
          {([["posts", Grid3X3, "帖子"], ["liked", Heart, "喜欢"], ["saved", Bookmark, "收藏"]] as const).map(([id, Icon, label]) => {
            const active = activeTab === id;
            return (
              <motion.button key={id} whileTap={{ scale: 0.94 }} onClick={() => setActiveTab(id)}
                className="relative flex items-center justify-center gap-1.5 pb-3 pt-1.5 outline-none"
                style={{ flex: 1, color: active ? "var(--neu-accent-primary)" : "var(--neu-text-muted)", fontSize: 13, fontWeight: active ? 500 : 400 }}>
                <Icon style={{ width: 13, height: 13 }} strokeWidth={active ? 2 : 1.5} />
                {label}
                {active && <motion.div layoutId="user-profile-tab"
                  style={{ position: "absolute", bottom: -1, left: "18%", right: "18%", height: 2, borderRadius: 1, background: "var(--neu-accent-primary)" }}
                  transition={{ type: "spring", stiffness: 420, damping: 34 }} />}
              </motion.button>
            );
          })}
        </div>

        {/* Gallery */}
        <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 3, padding: "3px" }}>
          {user.gallery.map((src, i) => (
            <motion.div key={i} whileTap={{ scale: 0.97 }} className="relative cursor-pointer"
              style={{ aspectRatio: "1", overflow: "hidden" }}>
              <img src={src} alt={`photo ${i}`} className="w-full h-full object-cover" />
              {activeTab === "liked" && (
                <div className="absolute inset-0 flex items-end justify-end p-1.5"
                  style={{ background: "var(--overlay-soft)" }}>
                  <div className="flex items-center justify-center rounded-full"
                    style={{ width: 20, height: 20, background: "var(--neu-accent-primary)" }}>
                    <Heart style={{ width: 10, height: 10, color: "white" }} strokeWidth={2} fill="white" />
                  </div>
                </div>
              )}
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );

  const rail = (
    <>
      <RailSection title="所在位置">
        <RailLocationMini name={user.location} desc="附近 38 位活跃用户" />
      </RailSection>

      <RailSection title="兴趣标签">
        <div className="flex gap-2 flex-wrap">
          {user.tags.map(t => (
            <span
              key={t}
              className="px-2.5 py-0.5 rounded-full"
              style={{
                fontSize: 11,
                background: "var(--neu-accent-soft)",
                color: "var(--neu-accent-primary)",
                border: "1px solid var(--neu-accent-soft-border)",
              }}
            >
              #{t}
            </span>
          ))}
        </div>
      </RailSection>

      {mutualFriends.length > 0 && (
        <RailSection title="共同好友">
          <div className="flex flex-col gap-3">
            {mutualFriends.map(f => (
              <RailUserRow
                key={f.id}
                avatar={f.avatar}
                name={f.name}
                sub={f.sub}
                onClick={() => navigate(ROUTES.user(f.id))}
              />
            ))}
          </div>
        </RailSection>
      )}

      <RailSection title="精选作品" action="全部" onAction={() => setActiveTab("posts")}>
        <div className="grid grid-cols-3" style={{ gap: 3 }}>
          {user.gallery.slice(0, 6).map((src, i) => (
            <div key={i} className="overflow-hidden rounded-[6px]" style={{ aspectRatio: "1" }}>
              <img src={src} alt={`thumb ${i}`} className="w-full h-full object-cover" />
            </div>
          ))}
        </div>
      </RailSection>

      <RailSection title="你可能也关注" action="更多" onAction={() => navigate(ROUTES.nearbyPeople)}>
        <div className="flex flex-col gap-3">
            {similarUsers.map(u => (
            <RailUserRow
              key={u.id}
              avatar={u.avatar}
              name={u.name}
              sub={u.sub}
              onClick={() => navigate(ROUTES.user(u.id))}
            />
          ))}
        </div>
      </RailSection>
    </>
  );

  return <DesktopDetailSplit main={main} rail={rail} />;
}