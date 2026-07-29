import { motion } from "motion/react";
import { useNavigate } from "react-router";
import { AuthForm } from "../../components/auth/AuthForm";
import { ROUTES } from "../../lib/routes";

// Full-page auth route at /login. Covers login / register / guest / forgot+reset password.
// On success from the route-level entry we go home; when entered via requireAuth
// the AuthContext already holds the pending action and replays it on success.

export function AuthPage() {
  const navigate = useNavigate();
  return (
    <div className="flex min-h-screen flex-col items-center justify-center px-5 py-10" style={{ background: "var(--neu-bg)" }}>
      <motion.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: [0.22, 1, 0.36, 1] }}
        className="w-full"
        style={{ maxWidth: 380 }}
      >
        <div className="mb-7 text-center">
          <h1 className="text-[26px] font-semibold tracking-tight" style={{ color: "var(--neu-text)" }}>
            热点
          </h1>
          <p className="mt-1.5 text-[13px]" style={{ color: "var(--neu-text-muted)" }}>
            发现身边的人与故事
          </p>
        </div>
        <div
          className="rounded-[22px] px-5 py-6"
          style={{ background: "var(--neu-surface, var(--neu-bg))", border: "1px solid var(--neu-border)" }}
        >
          <AuthForm
            onSuccess={() => {
              // Route-level entry: go home. Prompt-driven entries replay their
              // pending action via AuthContext; either way, leave /login.
              navigate(ROUTES.home, { replace: true });
            }}
          />
        </div>
      </motion.div>
    </div>
  );
}
