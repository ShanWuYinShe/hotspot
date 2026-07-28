import { AnimatePresence, motion } from "motion/react";
import { X } from "lucide-react";
import { AuthForm } from "./AuthForm";

// Bottom sheet shown when requireAuth() intercepts a write action by an
// anonymous user. Embeds the shared AuthForm; on success AuthContext replays
// the deferred action automatically.

interface Props {
  open: boolean;
  actionLabel: string | null;
  onClose: () => void;
  onSuccess: () => void;
}

export function LoginPromptSheet({ open, actionLabel, onClose, onSuccess }: Props) {
  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="fixed inset-0 z-[90]"
            style={{ background: "rgba(0,0,0,.4)", backdropFilter: "blur(2px)" }}
            onClick={onClose}
          />
          <motion.div
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{ type: "spring", damping: 30, stiffness: 320 }}
            className="fixed bottom-0 left-1/2 z-[91] w-full max-w-[460px] -translate-x-1/2 rounded-t-[28px] px-5 pb-8 pt-3"
            style={{ background: "var(--neu-bg)", boxShadow: "0 -8px 30px rgba(0,0,0,.12)" }}
            role="dialog"
            aria-modal="true"
          >
            <div className="mb-3 flex justify-center" aria-hidden="true">
              <span className="h-1 w-9 rounded-full" style={{ background: "var(--neu-border-pressed)" }} />
            </div>
            <div className="mb-4 flex items-start justify-between">
              <div>
                <h2 className="text-[17px] font-semibold tracking-tight" style={{ color: "var(--neu-text)" }}>
                  登录后继续
                </h2>
                {actionLabel && (
                  <p className="mt-0.5 text-[12px]" style={{ color: "var(--neu-text-muted)" }}>
                    请先登录后再「{actionLabel}」
                  </p>
                )}
              </div>
              <button
                type="button"
                onClick={onClose}
                aria-label="关闭"
                className="flex items-center justify-center rounded-full transition-opacity hover:opacity-70"
                style={{ width: 32, height: 32, color: "var(--neu-text-muted)" }}
              >
                <X size={18} />
              </button>
            </div>
            <AuthForm compact initialMode="login" onSuccess={onSuccess} />
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
}
