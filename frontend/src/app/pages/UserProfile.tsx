import React, { useState } from "react";
import { useParams, useNavigate } from "react-router";
import { ArrowLeft, MapPin, MessageCircle, UserPlus, UserCheck, Share2, Heart, Grid3X3, Bookmark, MoreHorizontal } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";

/* ── Mock data ────────────────────────────────────────────────────── */
const USERS: Record<number, {
  name: string; handle: string; bio: string; location: string;
  avatar: string; banner: string; followers: number; following: number; posts: number;
  gallery: string[]; tags: string[];
}> = {
  1: {
    name: "Liam",          handle: "@liam.photos",
    bio: "自由摄影师 · 用镜头记录城市的呼吸和光影。",
    location: "徐州·云龙区",
    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400",
    banner: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=1080",
    followers: 1240, following: 386, posts: 87,
    gallery: [
      "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=400",
      "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=400",
      "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=400",
      "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=400",
      "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=400",
      "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=400",
    ],
    tags: ["摄影", "城市", "光影"],
  },
  2: {
    name: "Sophia",        handle: "@sophia.wander",
    bio: "世界旅行者 · 每座城市都是一首诗。",
    location: "徐州·泉山区",
    avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=400",
    banner: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=1080",
    followers: 3820, following: 210, posts: 156,
    gallery: [
      "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?w=400",
      "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=400",
      "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=400",
      "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=400",
      "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=400",
      "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400",
    ],
    tags: ["旅行", "探索", "文化"],
  },
};

const DEFAULT_USER = USERS[1];

type Tab = "posts" | "liked" | "saved";

export function UserProfile() {
  const { userId } = useParams<{ userId: string }>();
  const navigate = useNavigate();
  const user = USERS[Number(userId)] ?? DEFAULT_USER;
  const [followed, setFollowed] = useState(false);
  const [activeTab, setActiveTab] = useState<Tab>("posts");

  const fmtNum = (n: number) => n >= 1000 ? `${(n / 1000).toFixed(1)}k` : String(n);

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)", overflowY: "auto" }}>
      {/* Banner */}
      <div className="relative overflow-hidden" style={{ height: "clamp(150px, 25vw, 210px)", flexShrink: 0 }}>
        <img src={user.banner} alt="banner" className="absolute inset-0 w-full h-full object-cover" />
        <div className="absolute inset-0" style={{ background: "linear-gradient(to bottom, rgba(0,0,0,0.15) 0%, rgba(0,0,0,0.55) 100%)" }} />

        {/* Top buttons */}
        <div className="absolute top-0 left-0 right-0 flex items-center justify-between px-4 pt-4">
          <motion.button whileTap={{ scale: 0.88 }} onClick={() => navigate(-1)}
            className="flex items-center justify-center rounded-full outline-none"
            style={{ width: 36, height: 36, background: "rgba(0,0,0,0.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)" }}>
            <ArrowLeft style={{ width: 15, height: 15, color: "white" }} strokeWidth={1.75} />
          </motion.button>
          <div className="flex gap-2">
            <motion.button whileTap={{ scale: 0.88 }}
              className="flex items-center justify-center rounded-full outline-none"
              style={{ width: 36, height: 36, background: "rgba(0,0,0,0.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)" }}>
              <Share2 style={{ width: 14, height: 14, color: "white" }} strokeWidth={1.75} />
            </motion.button>
            <motion.button whileTap={{ scale: 0.88 }}
              className="flex items-center justify-center rounded-full outline-none"
              style={{ width: 36, height: 36, background: "rgba(0,0,0,0.35)", backdropFilter: "blur(10px)", border: "1px solid rgba(255,255,255,0.25)" }}>
              <MoreHorizontal style={{ width: 14, height: 14, color: "white" }} strokeWidth={1.75} />
            </motion.button>
          </div>
        </div>
      </div>

      {/* Avatar + info — constrained width, centered */}
      <div style={{ maxWidth: 560, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
        <div className="px-5 -mt-10 pb-4 flex flex-col">
          <div className="flex items-end justify-between mb-3">
            <div className="relative">
              <div className="rounded-[22px] overflow-hidden"
                style={{ width: 76, height: 76, border: "3px solid var(--neu-bg)", boxShadow: "0 6px 20px rgba(0,0,0,0.22)" }}>
                <img src={user.avatar} alt={user.name} className="w-full h-full object-cover" />
              </div>
              <span className="absolute bottom-1 right-1 rounded-full"
                style={{ width: 12, height: 12, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />
            </div>

            {/* Action buttons */}
            <div className="flex gap-2 pb-1">
              <motion.button whileTap={{ scale: 0.93 }}
                onClick={() => navigate(`/messages/1`)}
                className="flex items-center gap-1.5 px-3.5 py-2 rounded-full outline-none"
                style={{ background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)", fontSize: 12, color: "var(--neu-text-muted)" }}>
                <MessageCircle style={{ width: 14, height: 14 }} strokeWidth={1.5} />
                私信
              </motion.button>
              <motion.button whileTap={{ scale: 0.93 }}
                onClick={() => setFollowed(f => !f)}
                className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none transition-all"
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
          <p style={{ fontSize: 13, color: "var(--neu-text-muted)", marginTop: 1 }}>{user.handle}</p>
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
          <div className="flex gap-0 mt-5 rounded-[20px] overflow-hidden"
            style={{ background: "var(--neu-bg)", boxShadow: "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
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
                  style={{ background: "rgba(0,0,0,0.08)" }}>
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
}