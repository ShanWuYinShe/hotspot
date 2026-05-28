import { useState } from "react";
import { useParams, useNavigate } from "react-router";
import {
  ArrowLeft,
  MapPin,
  Heart,
  MessageCircle,
  Share2,
  Bookmark,
  MoreHorizontal,
  Send,
} from "lucide-react";
import { motion } from "motion/react";

import { usePostDetailData } from "@/lib/hooks/use-post-detail-data";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { useAuthStore } from "@/lib/stores/auth";
import { toast } from "sonner";

export function PostDetail() {
  const { postId } = useParams<{ postId: string }>();
  const navigate = useNavigate();
  const {
    post,
    comments,
    loading,
    error,
    isLiked,
    isFavorited,
    likesCount,
    commentsCount,
    toggleLike,
    toggleFavorite,
    addComment,
    toggleCommentLike,
    refresh,
  } = usePostDetailData(postId);
  const currentAccount = useAuthStore((s) => s.currentAccount);
  useDocumentTitle("动态详情");

  const [input, setInput] = useState("");
  const [replyTo, setReplyTo] = useState<{ id: string; name: string } | null>(
    null,
  );

  const handleShare = async () => {
    const url = `${window.location.origin}/post/${postId}`;
    if (navigator.share) {
      try {
        await navigator.share({ title: post?.description?.slice(0, 50), url });
      } catch {}
    } else {
      await navigator.clipboard.writeText(url);
      toast.success("链接已复制");
    }
  };

  const handleComment = async () => {
    const text = input.trim();
    if (!text) return;
    const ok = await addComment(replyTo ? `@${replyTo.name} ${text}` : text);
    if (ok) {
      setInput("");
      setReplyTo(null);
    }
  };

  if (loading || !post) {
    return (
      <div
        className="flex flex-col items-center justify-center gap-3"
        role="status"
        aria-busy={!error}
        style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}
      >
        <p style={{ fontSize: 14, color: "var(--neu-text-muted)" }}>
          {error || "加载中…"}
        </p>
        {error && (
          <motion.button
            whileTap={{ scale: 0.94 }}
            onClick={refresh}
            className="px-4 py-2 rounded-full outline-none"
            style={{
              background: "var(--neu-accent-primary)",
              color: "var(--text-on-accent, white)",
              fontSize: 13,
            }}
          >
            重试
          </motion.button>
        )}
      </div>
    );
  }

  return (
    <div
      className="flex flex-col"
      style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}
    >
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center justify-between px-4 py-3"
        style={{
          borderBottom: "1px solid var(--neu-border)",
          backgroundColor: "var(--neu-bg)",
        }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => goBack(navigate)}
          aria-label="返回"
          className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <ArrowLeft
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
        <h1 style={{ fontSize: 16, color: "var(--neu-text)" }}>动态详情</h1>
        <motion.button
          whileTap={{ scale: 0.88 }}
          aria-label="更多操作"
          className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <MoreHorizontal
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>
      </div>

      {/* Scrollable content */}
      <div className="flex-1 overflow-y-auto no-scrollbar">
        {/* Post images */}
        {post.images && post.images.length > 0 && (
          <div
            className="relative overflow-hidden"
            style={{ height: "clamp(220px, 38vw, 380px)" }}
          >
            <img
              src={post.images[0].url}
              alt="动态图片"
              className="w-full h-full object-cover"
              loading="lazy"
            />
            {(post.tags?.length ?? 0) > 0 && (
              <div
                className="absolute top-3 left-3 px-2.5 py-0.5 rounded-full"
                style={{
                  background: "var(--neu-accent-primary)",
                  boxShadow: "0 2px 8px var(--neu-glow)",
                }}
              >
                <span
                  style={{
                    fontSize: 10,
                    color: "var(--text-on-accent, white)",
                  }}
                >
                  #{post.tags![0]}
                </span>
              </div>
            )}
          </div>
        )}

        <div
          className="px-5 pt-4 pb-3"
          style={{
            maxWidth: 680,
            marginLeft: "auto",
            marginRight: "auto",
            width: "100%",
          }}
        >
          {/* Author */}
          <div className="flex items-center gap-3 mb-3">
            <motion.div
              whileTap={{ scale: 0.93 }}
              onClick={() => navigate(`/user/${post.user.user_id}`)}
              role="button"
              tabIndex={0}
              aria-label={post.user.nickname}
              onKeyDown={(e) =>
                e.key === "Enter" && navigate(`/user/${post.user.user_id}`)
              }
              className="rounded-full overflow-hidden flex-shrink-0 cursor-pointer"
              style={{
                width: 44,
                height: 44,
                border: "2px solid var(--neu-accent-primary)",
              }}
            >
              {post.user.avatar_url ? (
                <img
                  src={post.user.avatar_url}
                  alt={post.user.nickname}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div
                  className="w-full h-full flex items-center justify-center"
                  style={{ background: "var(--neu-accent-primary)" }}
                >
                  <span
                    style={{
                      fontSize: 18,
                      color: "var(--text-on-accent, white)",
                    }}
                  >
                    {post.user.nickname.charAt(0)}
                  </span>
                </div>
              )}
            </motion.div>
            <div className="flex-1 min-w-0">
              <p
                className="truncate"
                style={{ fontSize: 15, color: "var(--neu-text)" }}
              >
                {post.user.nickname}
              </p>
              <div className="flex items-center gap-1.5 min-w-0">
                {post.location && (
                  <>
                    <MapPin
                      style={{
                        width: 10,
                        height: 10,
                        color: "var(--neu-accent-primary)",
                      }}
                      strokeWidth={1.5}
                      className="flex-shrink-0"
                    />
                    <span
                      className="truncate"
                      style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                    >
                      {post.location.location_name}
                    </span>
                  </>
                )}
              </div>
            </div>
            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>
              {new Date(post.created_at).toLocaleDateString("zh-CN", {
                month: "short",
                day: "numeric",
              })}
            </span>
          </div>

          {/* Content */}
          <p
            style={{ fontSize: 14, color: "var(--neu-text)", lineHeight: 1.75 }}
          >
            {post.description}
          </p>

          {/* Tags */}
          {(post.tags?.length ?? 0) > 1 && (
            <div className="flex gap-2 mt-2 flex-wrap">
              {post.tags!.map((t) => (
                <span
                  key={t}
                  className="px-2.5 py-0.5 rounded-full"
                  style={{
                    fontSize: 11,
                    background: "var(--neu-accent-soft)",
                    color: "var(--neu-accent-primary)",
                  }}
                >
                  #{t}
                </span>
              ))}
            </div>
          )}

          {/* Actions */}
          <div
            className="flex items-center justify-between mt-4 py-3"
            style={{
              borderTop: "1px solid var(--neu-border)",
              borderBottom: "1px solid var(--neu-border)",
            }}
          >
            <div className="flex gap-4">
              <motion.button
                whileTap={{ scale: 0.85 }}
                onClick={toggleLike}
                aria-label="点赞"
                className="flex items-center gap-1.5 outline-none"
                style={{
                  color: isLiked
                    ? "var(--neu-accent-primary)"
                    : "var(--neu-text-muted)",
                }}
              >
                <Heart
                  style={{ width: 19, height: 19 }}
                  strokeWidth={1.5}
                  fill={isLiked ? "currentColor" : "none"}
                />
                <span style={{ fontSize: 13 }}>{likesCount}</span>
              </motion.button>
              <div
                className="flex items-center gap-1.5"
                style={{ color: "var(--neu-text-muted)" }}
              >
                <MessageCircle
                  style={{ width: 19, height: 19 }}
                  strokeWidth={1.5}
                />
                <span style={{ fontSize: 13 }}>{commentsCount}</span>
              </div>
            </div>
            <div className="flex gap-3">
              <motion.button
                whileTap={{ scale: 0.85 }}
                onClick={toggleFavorite}
                aria-label="收藏"
                className="outline-none"
                style={{
                  color: isFavorited
                    ? "var(--neu-accent-primary)"
                    : "var(--neu-text-muted)",
                }}
              >
                <Bookmark
                  style={{ width: 19, height: 19 }}
                  strokeWidth={1.5}
                  fill={isFavorited ? "currentColor" : "none"}
                />
              </motion.button>
              <motion.button
                whileTap={{ scale: 0.85 }}
                onClick={handleShare}
                aria-label="分享"
                className="outline-none"
                style={{ color: "var(--neu-text-muted)" }}
              >
                {" "}
                <Share2 style={{ width: 19, height: 19 }} strokeWidth={1.5} />
              </motion.button>
            </div>
          </div>

          {/* Comments */}
          <p
            className="uppercase mt-4 mb-3"
            style={{
              fontSize: 10,
              letterSpacing: "0.14em",
              color: "var(--neu-text-muted)",
            }}
          >
            评论 · {commentsCount}
          </p>
          <div className="flex flex-col gap-4">
            {comments.map((c) => (
              <div key={c.comment_id} className="flex gap-3">
                <div
                  className="rounded-full overflow-hidden flex-shrink-0"
                  style={{ width: 34, height: 34 }}
                >
                  {c.user.avatar_url ? (
                    <img
                      src={c.user.avatar_url}
                      alt={c.user.nickname}
                      className="w-full h-full object-cover"
                    />
                  ) : (
                    <div
                      className="w-full h-full flex items-center justify-center"
                      style={{ background: "var(--neu-accent-soft)" }}
                    >
                      <span
                        style={{
                          fontSize: 13,
                          color: "var(--neu-accent-primary)",
                        }}
                      >
                        {c.user.nickname.charAt(0)}
                      </span>
                    </div>
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-baseline gap-2 min-w-0">
                    <span
                      className="block truncate"
                      style={{ fontSize: 13, color: "var(--neu-text)" }}
                    >
                      {c.user.nickname}
                    </span>
                    <span
                      style={{ fontSize: 11, color: "var(--neu-text-muted)" }}
                    >
                      {new Date(c.created_at).toLocaleDateString("zh-CN", {
                        month: "short",
                        day: "numeric",
                      })}
                    </span>
                  </div>
                  <p
                    style={{
                      fontSize: 13,
                      color: "var(--neu-text-muted)",
                      lineHeight: 1.6,
                      marginTop: 2,
                    }}
                  >
                    {c.content}
                  </p>
                  <div className="flex items-center gap-1 mt-1.5">
                    <motion.button
                      whileTap={{ scale: 0.85 }}
                      onClick={() =>
                        toggleCommentLike(c.comment_id, !!c.is_liked)
                      }
                      aria-label="点赞评论"
                      className="flex items-center gap-1 outline-none"
                      style={{
                        color: c.is_liked
                          ? "var(--neu-accent-primary)"
                          : "var(--neu-text-muted)",
                      }}
                    >
                      <Heart
                        style={{ width: 12, height: 12 }}
                        strokeWidth={1.5}
                        fill={c.is_liked ? "currentColor" : "none"}
                      />
                      <span style={{ fontSize: 11 }}>{c.likes_count}</span>
                    </motion.button>
                    <button
                      onClick={() =>
                        setReplyTo({ id: c.comment_id, name: c.user.nickname })
                      }
                      aria-label="回复"
                      className="outline-none bg-transparent"
                      style={{
                        fontSize: 11,
                        color: "var(--neu-text-muted)",
                        marginLeft: 8,
                        cursor: "pointer",
                      }}
                    >
                      回复
                    </button>
                  </div>
                </div>
              </div>
            ))}
            {comments.length === 0 && (
              <div className="flex flex-col items-center py-8 gap-2">
                <MessageCircle
                  style={{
                    width: 28,
                    height: 28,
                    color: "var(--neu-text-muted)",
                    opacity: 0.3,
                  }}
                  strokeWidth={1.25}
                />
                <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
                  暂无评论，来写第一条吧
                </p>
              </div>
            )}
          </div>
          <div style={{ height: 80 }} />
        </div>
      </div>

      {/* Input bar */}
      <div
        className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderTop: "1px solid var(--neu-border)",
        }}
      >
        <div
          className="rounded-full overflow-hidden flex-shrink-0"
          style={{ width: 30, height: 30 }}
        >
          {currentAccount?.avatar_url ? (
            <img
              src={currentAccount.avatar_url}
              alt="我的头像"
              className="w-full h-full object-cover"
            />
          ) : (
            <div
              className="w-full h-full flex items-center justify-center"
              style={{ background: "var(--neu-accent-soft)" }}
            >
              <span
                style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}
              >
                {(currentAccount?.nickname || "我").charAt(0)}
              </span>
            </div>
          )}
        </div>
        <div
          className="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}
        >
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleComment();
              if (e.key === "Escape") setReplyTo(null);
            }}
            placeholder={replyTo ? `回复 @${replyTo.name}…` : "写下你的评论…"}
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
        </div>
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={handleComment}
          aria-label="发送评论"
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{
            width: 36,
            height: 36,
            background: input.trim()
              ? "var(--neu-accent-primary)"
              : "var(--neu-bg)",
            boxShadow: input.trim()
              ? "0 4px 14px var(--neu-glow)"
              : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
            transition: "all 0.2s",
          }}
        >
          <Send
            style={{
              width: 14,
              height: 14,
              color: input.trim()
                ? "var(--text-on-accent, white)"
                : "var(--neu-text-muted)",
              transform: "translateX(1px)",
            }}
            strokeWidth={2}
          />
        </motion.button>
      </div>
    </div>
  );
}
