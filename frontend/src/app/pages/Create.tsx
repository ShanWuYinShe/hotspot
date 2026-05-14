import React, { useState } from "react";
import { ImagePlus, MapPin, Send, Smile, Hash, X } from "lucide-react";
import { motion } from "motion/react";

const USER_AVATAR =
  "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400";

const MOOD_TAGS = ["Calm", "Adventure", "Foodie", "Culture", "Nature", "Urban"];

export function Create() {
  const [text, setText] = useState("");
  const [selectedMoods, setSelectedMoods] = useState<Set<string>>(new Set());
  const [hasImage, setHasImage] = useState(false);

  const toggleMood = (tag: string) =>
    setSelectedMoods((prev) => {
      const next = new Set(prev);
      next.has(tag) ? next.delete(tag) : next.add(tag);
      return next;
    });

  const canPost = text.trim().length > 0;

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="flex flex-col w-full px-5 pt-14 pb-4"
    >
      {/* ── Header ────────────────────────────────────── */}
      <div className="mb-4">
        <h1 className="text-[24px] tracking-tight" style={{ color: "var(--neu-text)" }}>
          New Post
        </h1>
        <p className="text-[12px] mt-0.5" style={{ color: "var(--neu-text-muted)" }}>
          Share what's around you
        </p>
      </div>

      {/* ── Author Row ────────────────────────────────── */}
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
          <img src={USER_AVATAR} alt="You" className="w-full h-full object-cover" />
        </div>
        <div>
          <p className="text-[13px]" style={{ color: "var(--neu-text)" }}>
            Elara Vance
          </p>
          <div className="flex items-center gap-1">
            <MapPin style={{ width: 11, height: 11, color: "#D4845A" }} strokeWidth={1.5} />
            <span className="text-[11px]" style={{ color: "var(--neu-text-muted)" }}>
              Xuzhou, Jiangsu
            </span>
          </div>
        </div>
      </div>

      {/* ── Text Area ─────────────────────────────────── */}
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
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="What's on your mind?"
          className="w-full flex-1 bg-transparent outline-none resize-none text-[13px] leading-relaxed"
          style={{
            color: "var(--neu-text)",
            minHeight: 90,
          }}
        />
        {/* Toolbar row */}
        <div className="flex justify-between items-center mt-2">
          <div className="flex gap-2">
            {/* Emoji button — small neumorphic pill */}
            <motion.button
              whileTap={{ scale: 0.88 }}
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

            {/* Hashtag button — small neumorphic pill */}
            <motion.button
              whileTap={{ scale: 0.88 }}
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
              color: text.length > 240 ? "#D4845A" : "var(--neu-text-muted)",
            }}
          >
            {text.length}/280
          </span>
        </div>
      </div>

      {/* ── Image Upload ──────────────────────────────── */}
      {hasImage ? (
        <div
          className="relative rounded-[22px] overflow-hidden mb-4"
          style={{
            aspectRatio: "16/9",
            boxShadow:
              "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <img
            src="https://images.unsplash.com/photo-1762113396314-8e9e5417ed17?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800"
            alt="Selected"
            className="w-full h-full object-cover"
          />
          <motion.button
            whileTap={{ scale: 0.9 }}
            onClick={() => setHasImage(false)}
            className="absolute top-2.5 right-2.5 rounded-full flex items-center justify-center outline-none"
            style={{
              width: 26,
              height: 26,
              background: "rgba(0,0,0,0.45)",
              backdropFilter: "blur(4px)",
            }}
          >
            <X style={{ width: 14, height: 14, color: "white" }} strokeWidth={2} />
          </motion.button>
        </div>
      ) : (
        <motion.button
          whileTap={{ scale: 0.98 }}
          onClick={() => setHasImage(true)}
          className="flex flex-col items-center justify-center gap-1.5 rounded-[22px] mb-4 outline-none w-full"
          style={{
            height: 80,
            boxShadow:
              "inset 4px 4px 8px var(--neu-shadow-dark), inset -4px -4px 8px var(--neu-shadow-light)",
            border: "1.5px dashed var(--neu-border-pressed)",
          }}
        >
          <ImagePlus
            style={{ width: 22, height: 22, color: "var(--neu-text-muted)" }}
            strokeWidth={1.25}
          />
          <span className="text-[12px]" style={{ color: "var(--neu-text-muted)" }}>
            Add photo
          </span>
        </motion.button>
      )}

      {/* ── Vibe Tags ─────────────────────────────────── */}
      <div className="mb-5">
        <p
          className="text-[11px] uppercase tracking-widest mb-2.5"
          style={{ color: "var(--neu-text-muted)" }}
        >
          Vibe
        </p>
        <div className="flex flex-wrap gap-2">
          {MOOD_TAGS.map((tag) => {
            const active = selectedMoods.has(tag);
            return (
              <motion.button
                key={tag}
                whileTap={{ scale: 0.92 }}
                onClick={() => toggleMood(tag)}
                className="px-4 py-1.5 rounded-full text-[12px] outline-none transition-all duration-250"
                style={
                  active
                    ? {
                        background: "linear-gradient(135deg, #E8A87C, #D4845A)",
                        color: "white",
                        boxShadow: "0 4px 12px rgba(212,132,90,0.3)",
                        border: "1px solid transparent",
                      }
                    : {
                        background: "var(--neu-bg)",
                        boxShadow:
                          "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
                        border: "1px solid var(--neu-border)",
                        color: "var(--neu-text-muted)",
                      }
                }
              >
                {tag}
              </motion.button>
            );
          })}
        </div>
      </div>

      {/* ── Post Button ───────────────────────────────── */}
      <motion.button
        whileTap={canPost ? { scale: 0.97 } : {}}
        disabled={!canPost}
        className="flex items-center justify-center gap-2 w-full py-3.5 rounded-[22px] text-[14px] outline-none transition-all duration-300"
        style={
          canPost
            ? {
                background: "linear-gradient(135deg, #E8A87C, #D4845A)",
                color: "white",
                boxShadow: "0 8px 24px rgba(212,132,90,0.4)",
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
        <span>Share Post</span>
        <Send style={{ width: 16, height: 16 }} strokeWidth={1.75} />
      </motion.button>
    </motion.div>
  );
}