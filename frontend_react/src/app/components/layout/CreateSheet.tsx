import { useState } from "react";
import { X, ImagePlus, MapPin, Send, Smile, Hash } from "lucide-react";
import { motion } from "motion/react";
import { momentsService } from "@/lib/api/services/moments";
import { useLocationStore } from "@/lib/stores/location";
import { toast } from "sonner";

const USER_AVATAR =
  "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400";

const MOOD_TAGS = ["平静", "冒险", "美食", "文化", "自然", "都市"];

interface Props {
  onClose: () => void;
}

export function CreateSheet({ onClose }: Props) {
  const [text, setText] = useState("");
  const [selectedMoods, setSelectedMoods] = useState<Set<string>>(new Set());
  const [hasImage, setHasImage] = useState(false);
  const [isPosting, setIsPosting] = useState(false);
  const location = useLocationStore((s) => s.current);

  const toggleMood = (tag: string) =>
    setSelectedMoods((prev) => {
      const next = new Set(prev);
      next.has(tag) ? next.delete(tag) : next.add(tag);
      return next;
    });

  const canPost = text.trim().length > 0 && !isPosting;

  const handlePost = async () => {
    if (!canPost) return;
    setIsPosting(true);
    try {
      const tags = Array.from(selectedMoods);
      const description = text.trim();
      // 若 location store 为空，回退到浏览器 geolocation（拿不到时给个默认 location_name）
      let point = location?.point ?? { latitude: 0, longitude: 0 };
      let locationName = location?.location_name ?? "";
      if (!location && typeof navigator !== "undefined" && navigator.geolocation) {
        try {
          const pos = await new Promise<GeolocationPosition>((resolve, reject) =>
            navigator.geolocation.getCurrentPosition(resolve, reject, { timeout: 3000 }),
          );
          point = { latitude: pos.coords.latitude, longitude: pos.coords.longitude };
          locationName = "当前位置";
        } catch {
          locationName = "未知位置";
        }
      }
      if (point.latitude === 0 && point.longitude === 0 && locationName === "") {
        locationName = "未知位置";
      }
      await momentsService.create({
        description,
        location: {
          point,
          location_name: locationName,
        },
        tags: tags.length > 0 ? tags : undefined,
        privacy_level: "public",
      });
      toast.success("发布成功");
      setText("");
      setSelectedMoods(new Set());
      onClose();
    } catch (err: unknown) {
      const message =
        err && typeof err === "object" && "message" in err
          ? String((err as { message: unknown }).message)
          : "发布失败";
      toast.error(message);
    } finally {
      setIsPosting(false);
    }
  };

  return (
    <div
      className="flex flex-col h-full w-full overflow-hidden"
      style={{ backgroundColor: "var(--neu-bg)" }}
    >
      {/* ── Sheet Handle + Header ───────────────────────── */}
      <div
        className="flex-shrink-0 px-5 pt-4 pb-3"
        style={{
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        {/* Drag handle */}
        <div className="flex justify-center mb-4">
          <div
            className="rounded-full"
            style={{
              width: 36,
              height: 4,
              background: "var(--neu-border-pressed)",
            }}
          />
        </div>

        {/* Title row */}
        <div className="flex items-center justify-between">
          <div>
            <h1
              className="text-[20px] tracking-tight"
              style={{ color: "var(--neu-text)" }}
            >
              发布动态
            </h1>
            <p
              className="text-[12px] mt-0.5"
              style={{ color: "var(--neu-text-muted)" }}
            >
              分享你身边的新鲜事
            </p>
          </div>

          {/* Close button */}
          <motion.button
            aria-label="关闭"
            whileTap={{ scale: 0.88 }}
            onClick={onClose}
            className="flex items-center justify-center rounded-full outline-none"
            style={{
              width: 38,
              height: 38,
              background: "var(--neu-bg)",
              boxShadow:
                "5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <X
              style={{ width: 16, height: 16, color: "var(--neu-text-muted)" }}
              strokeWidth={1.75}
            />
          </motion.button>
        </div>
      </div>

      {/* ── Scrollable Body ─────────────────────────────── */}
      <div
        className="flex-1 overflow-y-auto no-scrollbar px-5 py-4 flex flex-col gap-4"
        style={{ overscrollBehavior: "contain" }}
      >
        {/* Author Row */}
        <div className="flex items-center gap-3">
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
            <img
              src={USER_AVATAR}
              alt="我的头像"
              className="w-full h-full object-cover"
            />
          </div>
          <div>
            <p className="text-[13px]" style={{ color: "var(--neu-text)" }}>
              Elara Vance
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
                徐州, 江苏
              </span>
            </div>
          </div>
        </div>

        {/* Text Area */}
        <div
          className="rounded-[22px] p-4"
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
            placeholder="此刻在想什么？"
            className="w-full flex-1 bg-transparent outline-none resize-none text-[13px] leading-relaxed"
            style={{ color: "var(--neu-text)", minHeight: 90 }}
          />
          <div className="flex justify-between items-center mt-2">
            <div className="flex gap-2">
              <motion.button
                aria-label="表情"
                whileTap={{ scale: 0.88 }}
                className="flex items-center justify-center rounded-full outline-none"
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
                aria-label="话题标签"
                whileTap={{ scale: 0.88 }}
                className="flex items-center justify-center rounded-full outline-none"
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

        {/* Image Upload */}
        {hasImage ? (
          <div
            className="relative rounded-[22px] overflow-hidden"
            style={{
              height: 120,
              boxShadow:
                "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <img
              src="https://images.unsplash.com/photo-1762113396314-8e9e5417ed17?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=800"
              alt="已选照片"
              className="w-full h-full object-cover"
            />
            <motion.button
              aria-label="删除图片"
              whileTap={{ scale: 0.85 }}
              onClick={() => setHasImage(false)}
              className="absolute top-2.5 right-2.5 rounded-full flex items-center justify-center outline-none"
              style={{
                width: 26,
                height: 26,
                background: "rgba(0,0,0,0.45)",
                backdropFilter: "blur(4px)",
              }}
            >
              <X
                style={{ width: 14, height: 14, color: "white" }}
                strokeWidth={2}
              />
            </motion.button>
          </div>
        ) : (
          <motion.button
            whileTap={{ scale: 0.98 }}
            onClick={() => setHasImage(true)}
            className="flex flex-col items-center justify-center gap-1.5 rounded-[22px] outline-none w-full"
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
            <span
              className="text-[12px]"
              style={{ color: "var(--neu-text-muted)" }}
            >
              添加照片
            </span>
          </motion.button>
        )}

        {/* Vibe Tags */}
        <div>
          <p
            className="text-[11px] uppercase tracking-widest mb-2.5"
            style={{ color: "var(--neu-text-muted)" }}
          >
            心情
          </p>
          <div className="flex flex-wrap gap-2">
            {MOOD_TAGS.map((tag) => {
              const active = selectedMoods.has(tag);
              return (
                <motion.button
                  key={tag}
                  whileTap={{ scale: 0.94 }}
                  onClick={() => toggleMood(tag)}
                  className="px-4 py-1.5 rounded-full text-[12px] outline-none transition-all duration-250"
                  style={
                    active
                      ? {
                          background: "var(--neu-accent-primary)",
                          color: "var(--text-on-accent, white)",
                          boxShadow: "0 4px 12px rgba(0,0,0,0.15)",
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

        {/* Post Button */}
        <motion.button
          whileTap={canPost ? { scale: 0.97 } : {}}
          disabled={!canPost}
          onClick={handlePost}
          className="flex items-center justify-center gap-2 w-full py-3.5 rounded-[22px] text-[14px] outline-none transition-all duration-300"
          style={
            canPost
              ? {
                  background: "var(--neu-accent-primary)",
                  color: "var(--text-on-accent, white)",
                  boxShadow: "0 8px 24px rgba(0,0,0,0.18)",
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
          <span>发布</span>
          <Send style={{ width: 16, height: 16 }} strokeWidth={1.75} />
        </motion.button>

        {/* Bottom breathing room */}
        <div style={{ height: 8 }} />
      </div>
    </div>
  );
}
