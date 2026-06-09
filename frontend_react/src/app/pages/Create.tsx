import { useState } from "react";
import { useNavigate } from "react-router";
import { MapPin, Send, Smile, Hash, X } from "lucide-react";
import { motion } from "motion/react";

import { useAuthStore } from "@/lib/stores/auth";
import { useLocationStore } from "@/lib/stores/location";
import { useCreatePost } from "@/lib/hooks/use-create-post";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";
import { toast } from "sonner";

export function Create() {
  const navigate = useNavigate();
  const currentAccount = useAuthStore((s) => s.currentAccount);
  const profile = useAuthStore((s) => s.profile);
  const geoPoint = useLocationStore((s) => s.geoPoint);
  const { submitting, createPost } = useCreatePost();
  useDocumentTitle("发布");

  const [text, setText] = useState("");
  const [selectedTags, setSelectedTags] = useState<Set<string>>(new Set());

  const toggleTag = (tag: string) =>
    setSelectedTags((prev) => {
      const next = new Set(prev);
      next.has(tag) ? next.delete(tag) : next.add(tag);
      return next;
    });

  const canPost = text.trim().length > 0 && !submitting;

  const handleSubmit = async () => {
    if (!canPost || !geoPoint) return;

    const ok = await createPost({
      description: text.trim(),
      point: geoPoint,
      locationName: profile?.location?.location_name || "当前位置",
      tags: selectedTags.size > 0 ? Array.from(selectedTags) : undefined,
    });
    if (ok) {
      goBack(navigate);
    } else {
      toast.error("发布失败，请重试");
    }
  };

  const avatarUrl = currentAccount?.avatar_url;
  const nickname = currentAccount?.nickname || "我";
  const locationName = profile?.location?.location_name || "未知位置";

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="flex flex-col w-full px-5 pt-14 pb-4"
    >
      {/* Header */}
      <div className="mb-4">
        <h1
          className="text-[24px] tracking-tight"
          style={{ color: "var(--neu-text)" }}
        >
          发布动态
        </h1>
        <p
          className="text-[12px] mt-0.5"
          style={{ color: "var(--neu-text-muted)" }}
        >
          分享你此刻的所见所想
        </p>
      </div>

      {/* Author Row */}
      <div className="flex items-center gap-3 mb-4">
        <div
          className="rounded-full overflow-hidden flex-shrink-0"
          style={{
            width: 40,
            height: 40,
            border: "1.5px solid var(--neu-border)",
            boxShadow:
              "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
          }}
        >
          {avatarUrl ? (
            <img
              src={avatarUrl}
              alt="我的头像"
              className="w-full h-full object-cover"
            />
          ) : (
            <div
              className="w-full h-full flex items-center justify-center"
              style={{ background: "var(--neu-accent-primary)" }}
            >
              <span
                style={{ fontSize: 16, color: "var(--text-on-accent, white)" }}
              >
                {nickname.charAt(0)}
              </span>
            </div>
          )}
        </div>
        <div>
          <p className="text-[13px]" style={{ color: "var(--neu-text)" }}>
            {nickname}
          </p>
          <div className="flex items-center gap-1">
            <MapPin
              style={{
                width: 11,
                height: 11,
                color: "var(--neu-accent-primary)",
              }}
              strokeWidth={1.5}
            />
            <span
              className="text-[11px]"
              style={{ color: "var(--neu-text-muted)" }}
            >
              {locationName}
            </span>
          </div>
        </div>
      </div>

      {/* Text Area */}
      <div
        className="rounded-[22px] p-4 mb-4"
        style={{
          boxShadow:
            "inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light)",
          border: "1px solid var(--neu-border-pressed)",
          minHeight: 130,
          display: "flex",
          flexDirection: "column",
        }}
      >
        <textarea
          aria-label="发布内容"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="此刻的想法…"
          className="w-full flex-1 bg-transparent outline-none resize-none text-[13px] leading-relaxed"
          style={{
            color: "var(--neu-text)",
            minHeight: 90,
          }}
        />
        <div className="flex justify-between items-center mt-2">
          <div className="flex gap-2">
            <motion.button
              whileTap={{ scale: 0.88 }}
              aria-label="表情"
              className="flex items-center justify-center rounded-full outline-none transition-all duration-200"
              style={{
                width: 30,
                height: 30,
                background: "var(--neu-bg)",
                boxShadow:
                  "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
                color: "var(--neu-text-muted)",
              }}
            >
              <Smile style={{ width: 15, height: 15 }} strokeWidth={1.5} />
            </motion.button>
            <motion.button
              whileTap={{ scale: 0.88 }}
              aria-label="添加标签"
              className="flex items-center justify-center rounded-full outline-none transition-all duration-200"
              style={{
                width: 30,
                height: 30,
                background: "var(--neu-bg)",
                boxShadow:
                  "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
                color: "var(--neu-text-muted)",
              }}
            >
              <Hash style={{ width: 15, height: 15 }} strokeWidth={1.5} />
            </motion.button>
          </div>
          <span
            className="text-[11px]"
            style={{
              color:
                text.length > 240
                  ? "var(--neu-accent-primary)"
                  : "var(--neu-text-muted)",
            }}
          >
            {text.length}/280
          </span>
        </div>
      </div>

      {/* Tags */}
      {selectedTags.size > 0 && (
        <div className="mb-4">
          <div className="flex flex-wrap gap-2">
            {Array.from(selectedTags).map((tag) => (
              <motion.button
                key={tag}
                whileTap={{ scale: 0.94 }}
                onClick={() => toggleTag(tag)}
                className="flex items-center gap-1 px-3 py-1 rounded-full text-[12px] outline-none"
                style={{
                  background: "var(--neu-accent-soft)",
                  color: "var(--neu-accent-primary)",
                  border: "1px solid var(--neu-accent-soft-border)",
                }}
              >
                #{tag}
                <X style={{ width: 10, height: 10 }} strokeWidth={2} />
              </motion.button>
            ))}
          </div>
        </div>
      )}

      {/* Post Button */}
      <motion.button
        whileTap={canPost ? { scale: 0.97 } : {}}
        disabled={!canPost}
        onClick={handleSubmit}
        className="flex items-center justify-center gap-2 w-full py-3.5 rounded-[22px] text-[14px] outline-none transition-all duration-300"
        style={
          canPost
            ? {
                background: "var(--neu-accent-primary)",
                color: "var(--text-on-accent, white)",
                boxShadow: "0 8px 24px var(--neu-glow)",
                border: "1px solid transparent",
              }
            : {
                background: "var(--neu-bg)",
                boxShadow:
                  "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
                color: "var(--neu-text-muted)",
                opacity: 0.6,
                cursor: "not-allowed",
              }
        }
      >
        <span>{submitting ? "发布中…" : "发布"}</span>
        {!submitting && (
          <Send style={{ width: 16, height: 16 }} strokeWidth={1.75} />
        )}
      </motion.button>
    </motion.div>
  );
}
