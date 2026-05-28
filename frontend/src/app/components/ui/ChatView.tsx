import { useState, useRef, useEffect } from "react";
import {
  ArrowLeft,
  Phone,
  Video,
  Send,
  Smile,
  CheckCheck,
  MoreHorizontal,
} from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useChatData } from "@/lib/hooks/use-chat-data";
import { normalizeAvatar } from "@/lib/utils";

function formatTime(dateStr: string) {
  const d = new Date(dateStr);
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

/* ── ChatView ─────────────────────────────────────────────────────── */
interface ChatViewProps {
  chatId: string;
  showBackButton?: boolean;
  onBack?: () => void;
  compact?: boolean;
}

export function ChatView({
  chatId,
  showBackButton = false,
  onBack,
  compact = false,
}: ChatViewProps) {
  const {
    chat,
    messages,
    loading,
    sending,
    error,
    hasMore,
    sendMessage,
    loadMore,
    refresh,
  } = useChatData(chatId);
  const [inputVal, setInputVal] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);
  const currentUserIdRef = useRef<string | null>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [messages]);

  const handleSend = async () => {
    const text = inputVal.trim();
    if (!text || sending) return;
    const ok = await sendMessage(text);
    if (ok) setInputVal("");
  };

  const nextCursor = hasMore;

  if (loading) {
    return (
      <div
        className="flex flex-col items-center justify-center h-full gap-3"
        role="status"
        aria-busy="true"
      >
        <p style={{ color: "var(--neu-text-muted)", fontSize: 14 }}>加载中…</p>
      </div>
    );
  }

  if (error && !chat) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3">
        <p style={{ color: "var(--neu-text-muted)", fontSize: 14 }}>{error}</p>
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
      </div>
    );
  }

  if (!chat) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3">
        <p style={{ color: "var(--neu-text-muted)", fontSize: 14 }}>
          会话不存在
        </p>
      </div>
    );
  }

  const otherUser = chat.other_user;
  const isOnline = chat.presence_status === "online";

  return (
    <div
      className="flex flex-col h-full"
      style={{ backgroundColor: "var(--neu-bg)" }}
    >
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{
          height: compact ? 52 : 60,
          backgroundColor: "var(--neu-bg)",
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        {showBackButton && (
          <motion.button
            whileTap={{ scale: 0.88 }}
            onClick={onBack}
            aria-label="返回"
            className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
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
        )}

        <div className="relative flex-shrink-0">
          <div
            className="rounded-full overflow-hidden"
            style={{
              width: compact ? 34 : 38,
              height: compact ? 34 : 38,
              border: "1.5px solid var(--neu-border)",
              boxShadow:
                "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)",
            }}
          >
            <img
              src={normalizeAvatar(otherUser.avatar_url)}
              alt={otherUser.nickname}
              className="w-full h-full object-cover"
            />
          </div>
          {isOnline && (
            <span
              className="absolute bottom-0 right-0 rounded-full"
              style={{
                width: 9,
                height: 9,
                background: "#22c55e",
                border: "2px solid var(--neu-bg)",
              }}
            />
          )}
        </div>

        <div className="flex-1 min-w-0">
          <p
            className="truncate"
            style={{ fontSize: compact ? 13 : 14, color: "var(--neu-text)" }}
          >
            {otherUser.nickname}
          </p>
          <p
            style={{
              fontSize: 11,
              color: isOnline ? "#22c55e" : "var(--neu-text-muted)",
            }}
          >
            {isOnline
              ? "在线"
              : `活跃于${chat.last_activity_at ? formatTime(chat.last_activity_at) : ""}`}
          </p>
        </div>

        <div className="flex gap-1.5">
          {([Phone, Video] as const).map((Icon, i) => (
            <motion.button
              key={i}
              aria-label={i === 0 ? "语音通话" : "视频通话"}
              whileTap={{ scale: 0.88 }}
              className="flex items-center justify-center rounded-full outline-none"
              style={{
                width: compact ? 32 : 36,
                height: compact ? 32 : 36,
                background: "var(--neu-bg)",
                boxShadow:
                  "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <Icon
                style={{
                  width: 14,
                  height: 14,
                  color: "var(--neu-text-muted)",
                }}
                strokeWidth={1.5}
              />
            </motion.button>
          ))}
          <motion.button
            whileTap={{ scale: 0.88 }}
            aria-label="更多操作"
            className="flex items-center justify-center rounded-full outline-none"
            style={{
              width: compact ? 32 : 36,
              height: compact ? 32 : 36,
              background: "var(--neu-bg)",
              boxShadow:
                "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <MoreHorizontal
              style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }}
              strokeWidth={1.5}
            />
          </motion.button>
        </div>
      </div>

      {/* Messages */}
      <div
        ref={scrollRef}
        className="flex-1 overflow-y-auto no-scrollbar"
        style={{ overscrollBehavior: "contain" }}
      >
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.2 }}
          className="flex flex-col gap-2 px-4 pt-4 pb-3"
        >
          {/* Load more button */}
          {nextCursor && (
            <div className="flex justify-center mb-2">
              <button
                onClick={loadMore}
                className="px-3 py-1 rounded-full"
                style={{
                  fontSize: 10,
                  background: "var(--neu-bg)",
                  color: "var(--neu-text-muted)",
                  border: "1px solid var(--neu-border)",
                }}
              >
                加载更多
              </button>
            </div>
          )}

          {/* Date pill */}
          <div className="flex justify-center mb-2">
            <span
              className="px-3 py-1 rounded-full"
              style={{
                fontSize: 10,
                background: "var(--neu-bg)",
                color: "var(--neu-text-muted)",
                border: "1px solid var(--neu-border)",
                letterSpacing: "0.05em",
                opacity: 0.8,
              }}
            >
              今天
            </span>
          </div>

          <AnimatePresence initial={false}>
            {messages.map((msg) => {
              const isSent =
                msg.sender.user_id === currentUserIdRef.current ||
                msg.sender.user_id !== otherUser.user_id;
              return (
                <motion.div
                  key={msg.message_id}
                  initial={{ opacity: 0, y: 8, scale: 0.96 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
                  className={`flex ${isSent ? "justify-end" : "justify-start"}`}
                >
                  <div
                    style={{
                      maxWidth: "76%",
                      padding: "10px 14px",
                      borderRadius: isSent
                        ? "18px 4px 18px 18px"
                        : "4px 18px 18px 18px",
                      ...(isSent
                        ? {
                            background: "var(--neu-accent-primary)",
                            boxShadow: "0 4px 14px var(--neu-glow)",
                          }
                        : {
                            background: "var(--neu-bg)",
                            boxShadow:
                              "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                            border: "1px solid var(--neu-border)",
                          }),
                    }}
                  >
                    {msg.recalled_at ? (
                      <p
                        style={{
                          fontSize: 13,
                          lineHeight: 1.55,
                          color: "var(--neu-text-muted)",
                          fontStyle: "italic",
                        }}
                      >
                        消息已撤回
                      </p>
                    ) : (
                      <>
                        <p
                          style={{
                            fontSize: 13,
                            lineHeight: 1.55,
                            color: isSent
                              ? "var(--text-on-accent, white)"
                              : "var(--neu-text)",
                          }}
                        >
                          {msg.content || "[媒体消息]"}
                        </p>
                        <p
                          style={{
                            fontSize: 10,
                            marginTop: 4,
                            color: isSent
                              ? "rgba(255,255,255,0.65)"
                              : "var(--neu-text-muted)",
                            display: "flex",
                            alignItems: "center",
                            gap: 3,
                            justifyContent: isSent ? "flex-end" : "flex-start",
                          }}
                        >
                          {formatTime(msg.sent_at)}
                          {isSent && msg.is_read && (
                            <CheckCheck
                              style={{ width: 12, height: 12, opacity: 0.8 }}
                              strokeWidth={2}
                            />
                          )}
                        </p>
                      </>
                    )}
                  </div>
                </motion.div>
              );
            })}
          </AnimatePresence>
        </motion.div>
      </div>

      {/* Input bar */}
      <div
        className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3"
        style={{
          backgroundColor: "var(--neu-bg)",
          borderTop: "1px solid var(--neu-border)",
        }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          aria-label="表情"
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{
            width: 36,
            height: 36,
            background: "var(--neu-bg)",
            boxShadow:
              "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <Smile
            style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }}
            strokeWidth={1.5}
          />
        </motion.button>

        <div
          className="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}
        >
          <input
            type="text"
            aria-label="消息"
            value={inputVal}
            onChange={(e) => setInputVal(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && !e.shiftKey && handleSend()}
            placeholder="发消息…"
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
        </div>

        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={handleSend}
          aria-label="发送消息"
          disabled={sending}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-all duration-200"
          style={{
            width: 36,
            height: 36,
            background: inputVal.trim()
              ? "var(--neu-accent-primary)"
              : "var(--neu-bg)",
            boxShadow: inputVal.trim()
              ? "0 4px 14px var(--neu-glow)"
              : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <Send
            style={{
              width: 14,
              height: 14,
              color: inputVal.trim()
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
