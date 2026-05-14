import React, { useState, useRef, useEffect } from "react";
import { ArrowLeft, Phone, Video, Send, Smile, CheckCheck, MoreHorizontal } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";

/* ── Data ───────────────────────────────────────────────────────────── */
export const CONTACTS: Record<number, { name: string; avatar: string; online: boolean; lastSeen?: string }> = {
  1: { name: "Noah",        avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", online: true },
  2: { name: "设计小队",    avatar: "https://images.unsplash.com/photo-1763467669450-e69029c1484d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", online: false, lastSeen: "2小时前" },
  3: { name: "Isabella",    avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", online: true },
  4: { name: "Kai",         avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", online: false, lastSeen: "3小时前" },
};

export const CONVERSATIONS: Record<number, Array<{ id: number; text: string; sent: boolean; time: string }>> = {
  1: [
    { id: 1, text: "今天怎么样？", sent: false, time: "13:18" },
    { id: 2, text: "还不错！刚在植物园附近发现一家超棒的咖啡馆 ☕", sent: true, time: "13:19" },
    { id: 3, text: "哪家呀，听起来很赞", sent: false, time: "13:21" },
    { id: 4, text: "就那家有拱形窗户、灯光很柔和的那个，你一定喜欢", sent: true, time: "13:22" },
    { id: 5, text: "这周末一定要去！", sent: false, time: "13:24" },
    { id: 6, text: "一起去吧 🌿", sent: true, time: "13:25" },
    { id: 7, text: "嘿，我们今晚的计划还在吗？", sent: false, time: "13:27" },
  ],
  2: [
    { id: 1, text: "刚把更新后的组件规范推到共享白板了", sent: true, time: "09:28" },
    { id: 2, text: "新的原型看起来超棒！", sent: false, time: "09:38" },
    { id: 3, text: "谢谢！新拟态卡片的阴影效果真的很满意", sent: true, time: "09:39" },
    { id: 4, text: "阴影层次感拿捏得刚刚好 👌", sent: false, time: "09:41" },
    { id: 5, text: "明早要开个评审会吗？", sent: true, time: "09:43" },
  ],
  3: [
    { id: 1, text: "昨天的日落看到了吗？", sent: true, time: "17:30" },
    { id: 2, text: "看到了！！美翻了 🌅", sent: false, time: "17:33" },
    { id: 3, text: "我试着拍了但照片真的不够好看", sent: true, time: "17:34" },
    { id: 4, text: "给你发张照片 📷", sent: false, time: "17:36" },
    { id: 5, text: "哇这也太美了！", sent: true, time: "17:38" },
  ],
  4: [
    { id: 1, text: "快出发了吗？", sent: false, time: "11:18" },
    { id: 2, text: "快了，给我10分钟", sent: true, time: "11:20" },
    { id: 3, text: "出门了！", sent: true, time: "11:22" },
    { id: 4, text: "在老地方见！", sent: false, time: "11:23" },
  ],
};

function formatTime(d: Date) {
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

/* ── ChatView ─────────────────────────────────────────────────────── */
interface ChatViewProps {
  chatId: number;
  showBackButton?: boolean;
  onBack?: () => void;
  compact?: boolean; // for desktop split-pane (smaller header)
}

export function ChatView({ chatId, showBackButton = false, onBack, compact = false }: ChatViewProps) {
  const contact = CONTACTS[chatId];
  const [messages, setMessages] = useState(() =>
    CONVERSATIONS[chatId] ? [...CONVERSATIONS[chatId]] : []
  );
  const [inputVal, setInputVal] = useState("");
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (el) el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  }, [messages]);

  if (!contact) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-3">
        <p style={{ color: "var(--neu-text-muted)", fontSize: 14 }}>会话不存在</p>
      </div>
    );
  }

  const handleSend = () => {
    const text = inputVal.trim();
    if (!text) return;
    setMessages((prev) => [
      ...prev,
      { id: prev.length + 1, text, sent: true, time: formatTime(new Date()) },
    ]);
    setInputVal("");
  };

  const lastSentId = Math.max(...messages.filter((m) => m.sent).map((m) => m.id), -Infinity);

  return (
    <div className="flex flex-col h-full" style={{ backgroundColor: "var(--neu-bg)" }}>
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
            className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
            style={{
              width: 36, height: 36,
              background: "var(--neu-bg)",
              boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
              border: "1px solid var(--neu-border)",
            }}
          >
            <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
          </motion.button>
        )}

        {/* Avatar */}
        <div className="relative flex-shrink-0">
          <div
            className="rounded-full overflow-hidden"
            style={{
              width: compact ? 34 : 38, height: compact ? 34 : 38,
              border: "1.5px solid var(--neu-border)",
              boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)",
            }}
          >
            <img src={contact.avatar} alt={contact.name} className="w-full h-full object-cover" />
          </div>
          {contact.online && (
            <span
              className="absolute bottom-0 right-0 rounded-full"
              style={{ width: 9, height: 9, background: "#22c55e", border: "2px solid var(--neu-bg)" }}
            />
          )}
        </div>

        {/* Name + status */}
        <div className="flex-1 min-w-0">
          <p className="truncate" style={{ fontSize: compact ? 13 : 14, color: "var(--neu-text)" }}>
            {contact.name}
          </p>
          <p style={{ fontSize: 11, color: contact.online ? "#22c55e" : "var(--neu-text-muted)" }}>
            {contact.online ? "在线" : `活跃于${contact.lastSeen}`}
          </p>
        </div>

        {/* Actions */}
        <div className="flex gap-1.5">
          {([Phone, Video] as const).map((Icon, i) => (
            <motion.button
              key={i}
              whileTap={{ scale: 0.88 }}
              className="flex items-center justify-center rounded-full outline-none"
              style={{
                width: compact ? 32 : 36, height: compact ? 32 : 36,
                background: "var(--neu-bg)",
                boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)",
                border: "1px solid var(--neu-border)",
              }}
            >
              <Icon style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
          ))}
          <motion.button
            whileTap={{ scale: 0.88 }}
            className="flex items-center justify-center rounded-full outline-none"
            style={{
              width: compact ? 32 : 36, height: compact ? 32 : 36,
              background: "var(--neu-bg)",
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
            {messages.map((msg) => (
              <motion.div
                key={msg.id}
                initial={{ opacity: 0, y: 8, scale: 0.96 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
                className={`flex ${msg.sent ? "justify-end" : "justify-start"}`}
              >
                <div
                  style={{
                    maxWidth: "76%",
                    padding: "10px 14px",
                    borderRadius: msg.sent ? "18px 4px 18px 18px" : "4px 18px 18px 18px",
                    ...(msg.sent
                      ? {
                          background: "var(--neu-accent-primary)",
                          boxShadow: "0 4px 14px var(--neu-glow)",
                        }
                      : {
                          background: "var(--neu-bg)",
                          boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                          border: "1px solid var(--neu-border)",
                        }),
                  }}
                >
                  <p style={{ fontSize: 13, lineHeight: 1.55, color: msg.sent ? "white" : "var(--neu-text)" }}>
                    {msg.text}
                  </p>
                  <p
                    style={{
                      fontSize: 10, marginTop: 4,
                      color: msg.sent ? "rgba(255,255,255,0.65)" : "var(--neu-text-muted)",
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
        className="flex-shrink-0 flex items-center gap-2.5 px-4 py-3"
        style={{ backgroundColor: "var(--neu-bg)", borderTop: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0"
          style={{
            width: 36, height: 36,
            background: "var(--neu-bg)",
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
            className="flex-1 bg-transparent outline-none"
            style={{ fontSize: 13, color: "var(--neu-text)" }}
          />
        </div>

        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={handleSend}
          className="flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-all duration-200"
          style={{
            width: 36, height: 36,
            background: inputVal.trim() ? "var(--neu-accent-primary)" : "var(--neu-bg)",
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