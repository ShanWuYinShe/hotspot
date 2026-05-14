import React, { useState } from "react";
import { useParams, useNavigate } from "react-router";
import {
  ArrowLeft, MapPin, Heart, MessageCircle, Share2, Bookmark,
  MoreHorizontal, Send, ChevronRight,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";

/* ── Mock data ─────────────────────────────────────────────────── */
const POSTS: Record<number, {
  id: number; user: string; handle: string; avatar: string;
  location: string; image: string; content: string;
  time: string; likes: number; tag: string;
  comments: Array<{ id: number; user: string; avatar: string; text: string; time: string; likes: number }>;
}> = {
  40: {
    id: 40, user: "Aria Sol", handle: "@aria.sol", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
    location: "植物园", image: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=1080",
    content: "清晨在绿意中找到了一片宁静。黎明时分阳光穿过树叶，滤出完美的光影。推荐大家周末早起来这里散步，绝对值得！这种时刻让人感觉整个世界都属于自己。",
    time: "2小时前", likes: 124, tag: "自然",
    comments: [
      { id: 1, user: "Marco",   avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",    text: "太美了！我也想去",              time: "1小时前",  likes: 8 },
      { id: 2, user: "Noah",    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",   text: "这光线真的绝了，什么时候去的？", time: "45分钟前", likes: 3 },
      { id: 3, user: "Isabella",avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",   text: "我们下次一起去吧 🌿",           time: "20分钟前", likes: 12 },
    ],
  },
  41: {
    id: 41, user: "本地向导", handle: "@local.guide", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",
    location: "市中心核心区", image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=1080",
    content: "城市里那些安静的角落已经被我全部标注出来了。每一条小巷都有属于自己的故事，等待有心人去发现。\n\n这次找到了一条被遗忘的石板路，通向一家隐藏了30年的老茶馆。",
    time: "5小时前", likes: 89, tag: "城市",
    comments: [
      { id: 1, user: "Kai",    avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80", text: "请问地址方便分享吗？",  time: "3小时前",  likes: 5 },
      { id: 2, user: "Sophia", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80", text: "老茶馆！我一定要去！", time: "2小时前",  likes: 14 },
    ],
  },
};

const DEFAULT_POST = POSTS[40];

export function PostDetail() {
  const { postId } = useParams<{ postId: string }>();
  const navigate = useNavigate();
  const post = POSTS[Number(postId)] ?? DEFAULT_POST;

  const [liked, setLiked]   = useState(false);
  const [saved, setSaved]   = useState(false);
  const [input, setInput]   = useState("");
  const [comments, setComments] = useState(post.comments);

  const handleComment = () => {
    if (!input.trim()) return;
    setComments(prev => [...prev, {
      id: prev.length + 1, user: "Elara", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
      text: input.trim(), time: "刚刚", likes: 0,
    }]);
    setInput("");
  };

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      {/* Header */}
      <div className="flex-shrink-0 flex items-center justify-between px-4 py-3"
        style={{ borderBottom: "1px solid var(--neu-border)", backgroundColor: "var(--neu-bg)" }}>
        <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }}
          whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
          style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </motion.button>
        <h1 style={{ fontSize: 16, color: "var(--neu-text)" }}>动态详情</h1>
        <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }}
          whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }}
          className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
          style={{ width: 36, height: 36, background: "var(--neu-bg)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
          <MoreHorizontal style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </motion.button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto no-scrollbar">
        {/* Post image */}
        <div className="relative overflow-hidden" style={{ height: "clamp(220px, 38vw, 380px)" }}>
          <img src={post.image} alt="post" className="w-full h-full object-cover" />
          <div className="absolute top-3 left-3 px-2.5 py-0.5 rounded-full"
            style={{ background: "var(--neu-accent-primary)", boxShadow: "0 2px 8px var(--neu-glow)" }}>
            <span style={{ fontSize: 10, color: "white" }}>#{post.tag}</span>
          </div>
        </div>

        <div className="px-5 pt-4 pb-3"
          style={{ maxWidth: 680, marginLeft: "auto", marginRight: "auto", width: "100%" }}>
          {/* Author */}
          <div className="flex items-center gap-3 mb-3">
            <motion.div whileTap={{ scale: 0.95 }} onClick={() => navigate(`/user/${post.id}`)}
              className="rounded-full overflow-hidden flex-shrink-0 cursor-pointer"
              style={{ width: 44, height: 44, border: "2px solid var(--neu-accent-primary)" }}>
              <img src={post.avatar} alt={post.user} className="w-full h-full object-cover" />
            </motion.div>
            <div className="flex-1">
              <p style={{ fontSize: 15, color: "var(--neu-text)" }}>{post.user}</p>
              <div className="flex items-center gap-1.5">
                <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{post.handle}</span>
                <span style={{ fontSize: 10, color: "var(--neu-text-muted)", opacity: 0.5 }}>·</span>
                <MapPin style={{ width: 10, height: 10, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{post.location}</span>
              </div>
            </div>
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{post.time}</span>
          </div>

          {/* Content */}
          <p style={{ fontSize: 14, color: "var(--neu-text)", lineHeight: 1.75 }}>{post.content}</p>

          {/* Actions */}
          <div className="flex items-center justify-between mt-4 py-3"
            style={{ borderTop: "1px solid var(--neu-border)", borderBottom: "1px solid var(--neu-border)" }}>
            <div className="flex gap-4">
              <motion.button whileTap={{ scale: 0.85 }}
                onClick={() => setLiked(v => !v)}
                className="flex items-center gap-1.5 outline-none"
                style={{ color: liked ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}>
                <Heart style={{ width: 19, height: 19 }} strokeWidth={1.5} fill={liked ? "currentColor" : "none"} />
                <span style={{ fontSize: 13 }}>{post.likes + (liked ? 1 : 0)}</span>
              </motion.button>
              <div className="flex items-center gap-1.5" style={{ color: "var(--neu-text-muted)" }}>
                <MessageCircle style={{ width: 19, height: 19 }} strokeWidth={1.5} />
                <span style={{ fontSize: 13 }}>{comments.length}</span>
              </div>
            </div>
            <div className="flex gap-3">
              <motion.button whileTap={{ scale: 0.85 }} onClick={() => setSaved(v => !v)} className="outline-none"
                style={{ color: saved ? "var(--neu-accent-primary)" : "var(--neu-text-muted)" }}>
                <Bookmark style={{ width: 19, height: 19 }} strokeWidth={1.5} fill={saved ? "currentColor" : "none"} />
              </motion.button>
              <motion.button whileTap={{ scale: 0.85 }} className="outline-none" style={{ color: "var(--neu-text-muted)" }}>
                <Share2 style={{ width: 19, height: 19 }} strokeWidth={1.5} />
              </motion.button>
            </div>
          </div>

          {/* Comments */}
          <p className="uppercase mt-4 mb-3" style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--neu-text-muted)" }}>
            评论 · {comments.length}
          </p>
          <div className="flex flex-col gap-4">
            {comments.map(c => (
              <div key={c.id} className="flex gap-3">
                <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 34, height: 34 }}>
                  <img src={c.avatar} alt={c.user} className="w-full h-full object-cover" />
                </div>
                <div className="flex-1">
                  <div className="flex items-baseline gap-2">
                    <span style={{ fontSize: 13, color: "var(--neu-text)" }}>{c.user}</span>
                    <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{c.time}</span>
                  </div>
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)", lineHeight: 1.6, marginTop: 2 }}>{c.text}</p>
                  <div className="flex items-center gap-1 mt-1.5">
                    <motion.button whileTap={{ scale: 0.85 }} className="flex items-center gap-1 outline-none" style={{ color: "var(--neu-text-muted)" }}>
                      <Heart style={{ width: 12, height: 12 }} strokeWidth={1.5} />
                      <span style={{ fontSize: 11 }}>{c.likes}</span>
                    </motion.button>
                    <span style={{ fontSize: 11, color: "var(--neu-text-muted)", marginLeft: 8, cursor: "pointer" }}>回复</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
          <div style={{ height: 80 }} />
        </div>
      </div>

      {/* Input bar */}
      <div className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderTop: "1px solid var(--neu-border)" }}>
        <div className="rounded-full overflow-hidden flex-shrink-0" style={{ width: 30, height: 30 }}>
          <img src="https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" alt="me" className="w-full h-full object-cover" />
        </div>
        <div className="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}>
          <input value={input} onChange={e => setInput(e.target.value)} onKeyDown={e => e.key === "Enter" && handleComment()}
            placeholder="写下你的评论…" className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }} />
        </div>
        <motion.button whileTap={{ scale: 0.88 }} onClick={handleComment}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{ width: 36, height: 36, background: input.trim() ? "var(--neu-accent-primary)" : "var(--neu-bg)", boxShadow: input.trim() ? "0 4px 14px var(--neu-glow)" : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", transition: "all 0.2s" }}>
          <Send style={{ width: 14, height: 14, color: input.trim() ? "white" : "var(--neu-text-muted)", transform: "translateX(1px)" }} strokeWidth={2} />
        </motion.button>
      </div>
    </div>
  );
}