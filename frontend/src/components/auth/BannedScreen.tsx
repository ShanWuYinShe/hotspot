'use client'
import { motion } from "motion/react";
import { ShieldAlert } from "lucide-react";

// Full-screen overlay shown when the account is banned (spec error code 10201).
// Rendered globally by AuthContext when bannedMessage is non-null. The only exit
// is acknowledging — the session has already been dropped client-side.

interface Props {
  message: string | null;
  onAcknowledge: () => void;
}

export function BannedScreen({ message, onAcknowledge }: Props) {
  return (
    <div
      className="fixed inset-0 z-[var(--z-toast)] flex items-center justify-center p-6"
      style={{ background: "var(--neu-bg)" }}
    >
      <motion.div
        initial={{ opacity: 0, scale: 0.96 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col items-center text-center"
        style={{ maxWidth: 360, gap: 16 }}
      >
        <div
          className="flex items-center justify-center rounded-full"
          style={{
            width: 64, height: 64,
            background: "color-mix(in srgb, var(--semantic-hot) 14%, transparent)",
          }}
        >
          <ShieldAlert size={30} color="var(--semantic-hot)" strokeWidth={1.6} />
        </div>
        <h1 className="text-[20px] font-semibold tracking-tight" style={{ color: "var(--neu-text)" }}>
          账号已被封禁
        </h1>
        <p className="text-[14px] leading-relaxed" style={{ color: "var(--neu-text-muted)" }}>
          {message ?? "你的账号因违规操作被封禁。如有疑问，请联系客服。"}
        </p>
        <motion.button
          type="button"
          whileTap={{ scale: 0.97 }}
          onClick={onAcknowledge}
          className="mt-2 rounded-[14px] px-8 py-2.5 text-[15px] font-medium"
          style={{ background: "var(--neu-accent-primary)", color: "white" }}
        >
          知道了
        </motion.button>
      </motion.div>
    </div>
  );
}
