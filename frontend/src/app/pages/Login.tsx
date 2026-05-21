import { useState, useCallback } from "react";
import { useNavigate } from "react-router";
import { motion } from "motion/react";
import { User, Lock, UserPlus, LogIn } from "lucide-react";
import { toast } from "sonner";
import { useAuthActions } from "@/lib/hooks/use-auth-actions";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
export function Login() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [nickname, setNickname] = useState("");
  const { loading, login, register } = useAuthActions();
  useDocumentTitle("登录");

  const handleSubmit = useCallback(
    async (e: { preventDefault: () => void }) => {
      e.preventDefault();

      if (!username.trim() || !password.trim()) {
        toast.error("请填写用户名和密码");
        return;
      }
      if (mode === "register" && !nickname.trim()) {
        toast.error("请填写昵称");
        return;
      }

      let ok: boolean;
      if (mode === "login") {
        ok = await login(username.trim(), password.trim());
        if (ok) toast.success("登录成功");
        else toast.error("登录失败");
      } else {
        ok = await register(username.trim(), nickname.trim(), password.trim());
        if (ok) toast.success("注册成功");
        else toast.error("注册失败");
      }
      if (ok) navigate("/", { replace: true });
    },
    [mode, username, password, nickname, navigate, login, register],
  );

  return (
    <div
      className="min-h-[100dvh] flex items-center justify-center px-5"
      style={{ backgroundColor: "var(--page-bg)" }}
    >
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
        className="w-full flex flex-col items-center"
        style={{ maxWidth: 400 }}
      >
        <div className="mb-8 text-center">
          <h1 className="text-[28px] tracking-tight" style={{ color: "var(--neu-text)" }}>
            HotSpot
          </h1>
          <p className="mt-1.5 text-[13px]" style={{ color: "var(--neu-text-muted)" }}>
            {mode === "login" ? "发现附近的热点" : "创建你的账号"}
          </p>
        </div>

        <div
          className="w-full rounded-[22px] overflow-hidden p-6"
          style={{
            background: "var(--neu-bg)",
            boxShadow: "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-1.5">
                <User style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                <label style={{ fontSize: 11, color: "var(--neu-text-muted)", textTransform: "uppercase", letterSpacing: "0.1em" }}>
                  用户名
                </label>
              </div>
              <div className="px-4 py-3 rounded-[18px] neu-input-surface">
                <input
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="输入用户名"
                  autoComplete="username"
                  aria-label="用户名"
                  className="w-full bg-transparent outline-none"
                  style={{ fontSize: 14, color: "var(--neu-text)" }}
                />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <div className="flex items-center gap-1.5">
                <Lock style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                <label style={{ fontSize: 11, color: "var(--neu-text-muted)", textTransform: "uppercase", letterSpacing: "0.1em" }}>
                  密码
                </label>
              </div>
              <div className="px-4 py-3 rounded-[18px] neu-input-surface">
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="输入密码"
                  autoComplete={mode === "login" ? "current-password" : "new-password"}
                  aria-label="密码"
                  className="w-full bg-transparent outline-none"
                  style={{ fontSize: 14, color: "var(--neu-text)" }}
                />
              </div>
            </div>

            {mode === "register" && (
              <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }} className="flex flex-col gap-1.5">
                <div className="flex items-center gap-1.5">
                  <UserPlus style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                  <label style={{ fontSize: 11, color: "var(--neu-text-muted)", textTransform: "uppercase", letterSpacing: "0.1em" }}>
                    昵称
                  </label>
                </div>
                <div className="px-4 py-3 rounded-[18px] neu-input-surface">
                  <input
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                    placeholder="给自己取个名字"
                    aria-label="昵称"
                    className="w-full bg-transparent outline-none"
                    style={{ fontSize: 14, color: "var(--neu-text)" }}
                  />
                </div>
              </motion.div>
            )}

            <motion.button
              whileTap={{ scale: 0.97 }}
              type="submit"
              disabled={loading}
              aria-label={mode === "login" ? "登录" : "注册"}
              className="flex items-center justify-center gap-2 w-full py-3 rounded-[16px] outline-none mt-2"
              style={{
                background: "var(--neu-accent-primary)",
                color: "var(--text-on-accent, white)",
                fontSize: 15,
                fontWeight: 500,
                boxShadow: "0 4px 14px var(--neu-glow)",
                border: "none",
                opacity: loading ? 0.7 : 1,
              }}
            >
              <LogIn style={{ width: 15, height: 15 }} strokeWidth={2} />
              {loading ? "处理中..." : mode === "login" ? "登录" : "注册"}
            </motion.button>
          </form>

          <div className="flex items-center justify-center mt-5 gap-1.5">
            <span style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
              {mode === "login" ? "没有账号？" : "已有账号？"}
            </span>
            <motion.button
              whileTap={{ scale: 0.93 }}
              type="button"
              onClick={() => { setMode(mode === "login" ? "register" : "login"); setNickname(""); }}
              aria-label={mode === "login" ? "切换到注册" : "切换到登录"}
              className="outline-none"
              style={{ fontSize: 13, color: "var(--neu-accent-primary)", fontWeight: 500 }}
            >
              {mode === "login" ? "注册" : "登录"}
            </motion.button>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
