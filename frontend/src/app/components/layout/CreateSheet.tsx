import React, { useState } from "react";
import {
  Globe2,
  Hash,
  ImagePlus,
  Lock,
  MapPin,
  Smile,
  Users,
  X,
} from "lucide-react";
import { motion } from "motion/react";
import { ActionIconButton } from "../shared/ActionIconButton";
import { Avatar } from "../shared/Avatar";
import { useAuth } from "../../contexts/AuthContext";
import { useDefaultLocation } from "../../hooks/useDefaultLocation";
import { createMoment } from "../../../api";

const MOOD_TAGS = ["松弛", "漫游", "咖啡", "展览", "自然", "夜色"];
const LOCATIONS = ["徐州 · 云龙湖附近", "徐州 · 回龙窝", "徐州 · 彭城广场"];

const AUDIENCES = [
  { id: "nearby", label: "附近可见", hint: "进入附近动态", Icon: MapPin },
  { id: "friends", label: "好友可见", hint: "仅你的好友", Icon: Users },
  { id: "private", label: "仅自己", hint: "保存在我的瞬间", Icon: Lock },
] as const;

type AudienceId = (typeof AUDIENCES)[number]["id"];

interface Props {
  onClose: () => void;
}

/** Shared composer used by the phone sheet, tablet sheet, and desktop dialog. */
export function CreateSheet({ onClose }: Props) {
  const { user } = useAuth();
  const { reference_point } = useDefaultLocation();
  const [text, setText] = useState("");
  const [selectedMoods, setSelectedMoods] = useState<Set<string>>(new Set());
  const [hasImage, setHasImage] = useState(false);
  const [locationIndex, setLocationIndex] = useState(0);
  const [audience, setAudience] = useState<AudienceId>("nearby");
  const [topicActive, setTopicActive] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const toggleMood = (tag: string) =>
    setSelectedMoods((previous) => {
      const next = new Set(previous);
      next.has(tag) ? next.delete(tag) : next.add(tag);
      return next;
    });

  const appendText = (snippet: string) => {
    setText((previous) => {
      if (previous.includes(snippet)) return previous;
      const separator = previous.trim().length ? "\n" : "";
      return `${previous}${separator}${snippet}`.slice(0, 280);
    });
  };

  const toggleTopic = () => {
    const topic = "#云龙湖漫游";
    setText((previous) =>
      topicActive ? previous.replace(topic, "").replace(/\n\n+/g, "\n").trimEnd() : appendTopic(previous, topic)
    );
    setTopicActive((active) => !active);
  };

  const canPost = text.trim().length > 0 && !submitting;
  const activeAudience = AUDIENCES.find((item) => item.id === audience) ?? AUDIENCES[0];

  const handleSubmit = async () => {
    if (!canPost) return;
    setSubmitting(true);
    try {
      const res = await createMoment({
        description: text.trim(),
        tags: Array.from(selectedMoods),
        // spec: location requires point + location_name. Point from ClientConfig default_location.
        location: reference_point
          ? { point: reference_point, location_name: LOCATIONS[locationIndex] }
          : { point: { latitude: 0, longitude: 0 }, location_name: LOCATIONS[locationIndex] },
        // spec PrivacyLevel: nearby→public, friends→friends, private→private
        privacy_level: audience === "nearby" ? "public" : audience,
      });
      if (res.code === 0) {
        onClose();
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form
      className="flex h-full w-full flex-col overflow-hidden bg-[var(--neu-bg)] text-[var(--neu-text)]"
      onSubmit={(event) => {
        event.preventDefault();
        void handleSubmit();
      }}
    >
      <header className="flex-none border-b border-[var(--neu-border)] px-5 pb-3 pt-4">
        <div className="mb-3 flex justify-center" aria-hidden="true">
          <span className="h-1 w-9 rounded-full bg-[var(--neu-border-pressed)]" />
        </div>
        <div className="flex items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="neu-eyebrow">附近动态</p>
            <h1 id="create-composer-title" className="mt-1 text-[20px] font-medium leading-none tracking-[-0.01em] text-[var(--neu-text)]">
              记录此刻
            </h1>
          </div>
          <ActionIconButton icon={X} label="关闭发布器" onClick={onClose} size={36} />
        </div>
      </header>

      <div className="no-scrollbar flex-1 overflow-y-auto px-5 py-4">
        <div className="mx-auto flex w-full max-w-[560px] flex-col gap-4">
          <div className="flex items-center justify-between gap-3">
            <div className="flex min-w-0 items-center gap-2.5">
              <Avatar src={user?.avatar_url} name={user?.nickname} size={36}
                className="border border-[var(--neu-border)]" />
              <div className="min-w-0">
                <p className="text-[13px] font-medium text-[var(--neu-text)]">{user?.nickname || "用户"}</p>
                <button
                  type="button"
                  onClick={() => setLocationIndex((index) => (index + 1) % LOCATIONS.length)}
                  className="mt-0.5 flex items-center gap-1 rounded-md text-left text-[11px] text-[var(--neu-text-muted)] outline-none transition-colors hover:text-[var(--neu-accent-primary)] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)]"
                  aria-label="切换发布位置"
                >
                  <MapPin size={12} className="text-[var(--neu-accent-primary)]" strokeWidth={1.75} />
                  <span className="truncate">{LOCATIONS[locationIndex]}</span>
                </button>
              </div>
            </div>
            <span className="rounded-full border border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] px-2.5 py-1 text-[10px] font-medium text-[var(--neu-accent-primary)]">
              正在发布
            </span>
          </div>

          <div className="neu-input-surface rounded-[20px] p-4">
            <textarea
              value={text}
              onChange={(event) => setText(event.target.value.slice(0, 280))}
              placeholder="写下此刻发生的事，或分享一个想被附近的人看见的瞬间…"
              className="min-h-[122px] w-full resize-none bg-transparent text-[15px] leading-7 text-[var(--neu-text)] outline-none placeholder:text-[var(--neu-text-muted)]"
              aria-label="动态内容"
            />
            <div className="mt-3 flex items-center justify-between border-t border-[var(--neu-border)] pt-3">
              <div className="flex items-center gap-1">
                <ComposerTool
                  icon={ImagePlus}
                  label={hasImage ? "已添加照片" : "添加照片"}
                  active={hasImage}
                  onClick={() => setHasImage((value) => !value)}
                />
                <ComposerTool icon={Smile} label="添加心情" onClick={() => appendText("✨")}/>
                <ComposerTool icon={Hash} label="添加话题" active={topicActive} onClick={toggleTopic} />
              </div>
              <span className={`text-[11px] tabular-nums ${text.length >= 260 ? "text-[var(--destructive)]" : "text-[var(--neu-text-muted)]"}`}>
                {text.length}/280
              </span>
            </div>
          </div>

          {hasImage && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              className="relative overflow-hidden rounded-[18px] neu-ring-border bg-[var(--page-bg)]"
            >
              <div style={{ overflow: "hidden", borderRadius: 18 }}>
                <img
                src="https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=600"
                alt="待发布的动态配图：云龙湖边的傍晚"
                className="h-36 w-full object-cover"
              />
              <span className="absolute bottom-2.5 left-2.5 rounded-full bg-black/55 px-2 py-1 text-[10px] text-white backdrop-blur-sm">1 张照片</span>
              <button
                type="button"
                onClick={() => setHasImage(false)}
                className="absolute right-2.5 top-2.5 flex h-7 w-7 items-center justify-center rounded-full bg-black/55 text-white outline-none backdrop-blur-sm transition-transform hover:scale-105 focus-visible:ring-2 focus-visible:ring-white"
                aria-label="移除照片"
              >
                <X size={14} strokeWidth={2} />
              </button>
              </div>
            </motion.div>
          )}

          <section aria-labelledby="mood-label">
            <div className="mb-2 flex items-center justify-between">
              <p id="mood-label" className="neu-eyebrow">此刻氛围</p>
              {selectedMoods.size > 0 && <span className="text-[10px] text-[var(--neu-text-muted)]">已选 {selectedMoods.size} 项</span>}
            </div>
            <div className="flex flex-wrap gap-2">
              {MOOD_TAGS.map((tag) => {
                const active = selectedMoods.has(tag);
                return (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => toggleMood(tag)}
                    aria-pressed={active}
                    className={`rounded-full border px-3 py-1.5 text-[12px] outline-none transition-all focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${
                      active
                        ? "border-[var(--neu-accent-soft-border)] bg-[var(--neu-accent-soft)] font-medium text-[var(--neu-accent-primary)] hover:border-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]"
                        : "border-[var(--neu-border)] bg-transparent text-[var(--neu-text-muted)] hover:border-[var(--neu-border-pressed)] hover:text-[var(--neu-text)]"
                    }`}
                  >
                    {tag}
                  </button>
                );
              })}
            </div>
          </section>

          <section aria-labelledby="audience-label" className="rounded-[18px] ring-1 ring-[var(--neu-border)] bg-[color-mix(in_srgb,var(--neu-bg)_94%,var(--page-bg)_6%)] p-2">
            <div className="flex items-center justify-between px-2 pb-2 pt-1">
              <p id="audience-label" className="neu-eyebrow">谁可以看到</p>
              <span className="text-[10px] text-[var(--neu-text-muted)]">{activeAudience.hint}</span>
            </div>
            <div className="grid grid-cols-3 gap-1">
              {AUDIENCES.map(({ id, label, Icon }) => {
                const active = audience === id;
                return (
                  <button
                    key={id}
                    type="button"
                    onClick={() => setAudience(id)}
                    aria-pressed={active}
                    className={`flex min-w-0 flex-col items-center gap-1 rounded-[12px] px-1 py-2 text-center outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${
                      active
                        ? "bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]"
                        : "text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)]/50"
                    }`}
                  >
                    <Icon size={14} strokeWidth={active ? 2 : 1.5} />
                    <span className="truncate text-[10px] font-medium">{label}</span>
                  </button>
                );
              })}
            </div>
          </section>
        </div>
      </div>

      <footer className="flex flex-none items-center gap-3 border-t border-[var(--neu-border)] bg-[var(--neu-bg)] px-5 py-3">
        <div className="min-w-0 flex-1">
          <p className="truncate text-[11px] text-[var(--neu-text-muted)]">
            {selectedMoods.size ? `将以 ${Array.from(selectedMoods).join(" · ")} 的氛围出现` : "选择氛围，让附近的人更容易找到你"}
          </p>
        </div>
        <motion.button
          whileTap={canPost ? { scale: 0.97 } : {}}
          type="submit"
          disabled={!canPost}
          className="shrink-0 rounded-full px-5 py-2.5 text-[13px] font-medium outline-none transition-all focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] disabled:cursor-not-allowed disabled:border disabled:border-[var(--neu-border)] disabled:bg-transparent disabled:text-[var(--neu-text-muted)] enabled:bg-[var(--neu-accent-primary)] enabled:text-white enabled:shadow-[0_6px_18px_var(--neu-glow)]"
        >
          发布动态
        </motion.button>
      </footer>
    </form>
  );
}

function appendTopic(text: string, topic: string) {
  if (text.includes(topic)) return text;
  const separator = text.trim().length ? "\n" : "";
  return `${text}${separator}${topic}`.slice(0, 280);
}

function ComposerTool({
  icon: Icon,
  label,
  active = false,
  onClick,
}: {
  icon: React.ElementType;
  label: string;
  active?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      aria-pressed={active}
      className={`flex h-8 w-8 items-center justify-center rounded-full outline-none transition-colors focus-visible:ring-2 focus-visible:ring-[var(--neu-accent-primary)] ${
        active
          ? "bg-[var(--neu-accent-soft)] text-[var(--neu-accent-primary)] hover:bg-[var(--neu-active-hover)]"
          : "text-[var(--neu-text-muted)] hover:bg-[var(--neu-accent-soft)] hover:text-[var(--neu-accent-primary)]"
      }`}
    >
      <Icon size={16} strokeWidth={active ? 2 : 1.6} />
    </button>
  );
}
