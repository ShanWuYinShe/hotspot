'use client'
import React, { useState, useRef, useEffect } from "react";
import { ArrowLeft, Phone, Video, Send, Smile, CheckCheck, MoreHorizontal } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { getPrivateMessages, sendPrivateMessage } from "@/api";
import { useAuth } from "@/contexts/AuthContext";
import { Avatar } from "@/components/shared/Avatar";

/* ── ChatView ─────────────────────────────────────────────────────── */
// spec: private chats are addressed by the peer's user_id (target_user_id).
// History comes from private_messages/get_history; sends go to
// private_messages/send. Message direction is derived from sender.user_id
// vs the current user.

interface ChatContact {
  name: string;
  avatar: string;
  online: boolean;
  lastSeen?: string;
}

interface ChatMessage {
  id: string;
  text: string;
  sent: boolean;   // true = outgoing (from current user)
  time: string;
}

interface ChatViewProps {
  /** The peer's user_id (spec: target_user_id). */
  peerUserId: string;
  /** Optional peer display info from the chat list; fetched if absent. */
  contact?: ChatContact;
  showBackButton?: boolean;
  onBack?: () => void;
  compact?: boolean; // for desktop split-pane (smaller header)
}

function formatTime(iso?: string) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

export function ChatView({ peerUserId, contact: contactProp, showBackButton = false, onBack, compact = false }: ChatViewProps) {
  const { user } = useAuth();
  const [contact, setContact] = useState<ChatContact | null>(contactProp ?? null);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputVal, setInputVal] = useState("");
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      // Fetch history by target_user_id; derive direction from sender.
      const res = await getPrivateMessages({ target_user_id: peerUserId, pagination: { limit: 50 } })
        .catch(() => ({ code: -1, data: { items: [] } }));
      if (cancelled) return;
      const items = (res.code === 0 && res.data ? (res.data as any).items : []) as any[];
      const mapped: ChatMessage[] = items.map((m: any) => ({
        id: String(m.message_id ?? ""),
        text: m.content ?? m.text ?? "",
        sent: m.sender?.user_id ? m.sender.user_id === (user?.user_id ?? "__me__") : Boolean(m.sent),
        time: formatTime(m.sent_at ?? m.created_at),
      }));
      setMessages(mapped);
      setLoading(false);

      // If no peer display info was passed, try to read it from the latest
      // incoming message's sender (or any message sender that isn't me).
      if (!contactProp) {
        const peer = items
          .map((m: any) => m.sender)
          .find((s: any) => s && s.user_id !== (user?.user_id ?? "__me__"));
        if (peer) {
          setContact({
            name: peer.nickname ?? peer.name ?? "",
            avatar: peer.avatar_url ?? peer.avatar ?? "",
            online: peer.presence_status === "online",
          });
        }
      }
    }
    if (peerUserId) load();
    else { setMessages([]); setLoading(false); }
    return () => { cancelled = true; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [peerUserId, user?.user_id]);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [loading]);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  }, [messages]);

  const peerName = contact?.name ?? "";

  const handleSend = async () => {
    const text = inputVal.trim();
    if (!text || sending || !peerUserId) return;
    setInputVal("");
    setSending(true);
    // Optimistic append so the user sees their message instantly.
    const tempId = `temp-${Date.now()}`;
    setMessages((prev) => [...prev, { id: tempId, text, sent: true, time: formatTime(new Date().toISOString()) }]);
    try {
      const res = await sendPrivateMessage({ target_user_id: peerUserId, type: "text", content: text });
      if (res.code !== 0) {
        // Roll back the optimistic message on failure.
        setMessages((prev) => prev.filter((m) => m.id !== tempId));
      }
    } catch {
      setMessages((prev) => prev.filter((m) => m.id !== tempId));
    } finally {
      setSending(false);
    }
  };

  const lastSentId = messages.filter((m) => m.sent).map((m) => m.id).pop();

  return (
    <div className="flex min-h-0 flex-col h-full bg-[var(--neu-bg)]">
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4 bg-[var(--neu-bg)]"
        style={{
          height: compact ? 52 : 60,
          borderBottom: "1px solid var(--neu-border)",
        }}
      >
        {showBackButton && (
          <motion.button
            whileTap={{ scale: 0.88 }}
            onClick={onBack}
            className="flex items-center justify-center rounded-full outline-none flex-shrink-0 bg-[var(--neu-bg)]"
            style={{
              width: 36, height: 36,
              boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
        )}

        {/* Avatar */}
        <Avatar
          src={contact?.avatar}
          name={peerName}
          size={compact ? 34 : 38}
          style={{
            border: "1.5px solid var(--neu-border)",
            boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)",
          }}
          badge={
            contact?.online ? (
              <span
                className="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]"
                style={{ width: 9, height: 9, border: "2px solid var(--neu-bg)" }}
              />
            ) : undefined
          }
        />

        {/* Name + status */}
        <div className="flex-1 min-w-0">
          <p className="truncate text-[var(--neu-text)]" style={{ fontSize: compact ? 13 : 14 }}>
            {peerName || "会话"}
          </p>
          <p className={contact?.online ? "text-[var(--semantic-online)]" : "text-[var(--neu-text-muted)]"} style={{ fontSize: 11 }}>
            {contact ? (contact.online ? "在线" : (contact.lastSeen ? `活跃于${contact.lastSeen}` : "离线")) : ""}
          </p>
        </div>

        {/* Actions */}
        <div className="flex gap-1.5">
          {([Phone, Video] as const).map((Icon, i) => (
            <motion.button
              key={i}
              whileTap={{ scale: 0.88 }}
              className="flex items-center justify-center rounded-full outline-none bg-[var(--neu-bg)]"
              style={{
                width: compact ? 32 : 36, height: compact ? 32 : 36,
                boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <Icon style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
          ))}
          <motion.button
            whileTap={{ scale: 0.88 }}
            className="flex items-center justify-center rounded-full outline-none bg-[var(--neu-bg)]"
            style={{
              width: compact ? 32 : 36, height: compact ? 32 : 36,
              boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <MoreHorizontal style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
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
          {/* Date pill */}
          <div className="flex justify-center mb-2">
            <span
              className="px-3 py-1 rounded-full bg-[var(--neu-bg)] text-[var(--neu-text-muted)]"
              style={{
                fontSize: 10,
                border: "1px solid var(--neu-border)",
                letterSpacing: "0.05em",
                opacity: 0.8,
              }}
            >
              今天
            </span>
          </div>

          <AnimatePresence initial={false}>
            {messages.map((msg) => (
              <motion.div
                key={msg.id}
                initial={{ opacity: 0, y: 8, scale: 0.96 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
                className={`flex ${msg.sent ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={msg.sent ? "bg-[var(--neu-accent-primary)]" : "bg-[var(--neu-bg)]"}
                  style={{
                    maxWidth: "76%",
                    padding: "10px 14px",
                    borderRadius: msg.sent ? "18px 4px 18px 18px" : "4px 18px 18px 18px",
                    ...(msg.sent
                      ? {
                          boxShadow: "0 4px 14px var(--neu-glow)",
                        }
                      : {
                          boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                          border: "1px solid var(--neu-border)",
                        }),
                  }}
                >
                  <p className={msg.sent ? "text-white" : "text-[var(--neu-text)]"} style={{ fontSize: 13, lineHeight: 1.55 }}>
                    {msg.text}
                  </p>
                  <p
                    className={msg.sent ? "text-[rgba(255,255,255,0.65)]" : "text-[var(--neu-text-muted)]"}
                    style={{
                      fontSize: 10, marginTop: 4,
                      display: "flex", alignItems: "center", gap: 3,
                      justifyContent: msg.sent ? "flex-end" : "flex-start",
                    }}
                  >
                    {msg.time}
                    {msg.sent && msg.id === lastSentId && (
                      <CheckCheck style={{ width: 12, height: 12, opacity: 0.8 }} strokeWidth={2} />
                    )}
                  </p>
                </div>
              </motion.div>
            ))}
          </AnimatePresence>
        </motion.div>
      </div>

      {/* Input bar */}
      <div
        className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--neu-bg)]"
        style={{ borderTop: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0 bg-[var(--neu-bg)]"
          style={{
            width: 36, height: 36,
            boxShadow: "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <Smile style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
        </motion.button>

        <div
          className="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface"
          style={{ height: 38 }}
        >
          <input
            type="text"
            value={inputVal}
            onChange={(e) => setInputVal(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSend()}
            placeholder="发消息…"
            className="flex-1 bg-transparent outline-none text-[var(--neu-text)]"
            style={{ fontSize: 13 }}
          />
        </div>

        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={handleSend}
          className={`flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-all duration-200 ${inputVal.trim() ? "bg-[var(--neu-accent-primary)]" : "bg-[var(--neu-bg)]"}`}
          style={{
            width: 36, height: 36,
            boxShadow: inputVal.trim()
              ? "0 4px 14px var(--neu-glow)"
              : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <Send
            style={{
              width: 14, height: 14,
              color: inputVal.trim() ? "white" : "var(--neu-text-muted)",
              transform: "translateX(1px)",
            }}
            strokeWidth={2}
          />
        </motion.button>
      </div>
    </div>
  );
}