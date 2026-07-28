import React, { useState } from "react";
import { motion } from "motion/react";
import { useHoverConfig } from "../../hooks/useHoverConfig";
import { useAuth } from "../../contexts/AuthContext";

// Shared auth form used by both the full-page Auth route and the LoginPromptSheet.
// Three modes: login / register / guest, plus a "forgot password" sub-flow
// (request_password_reset → reset_password). Matches harmonyapp LoginPage rules:
//   login_id 3–64, password 8–128 with letter+digit, nickname 1–30.

export type AuthMode = "login" | "register" | "guest" | "forgot" | "reset";

type FormState = "idle" | "submitting" | "error";

interface Props {
  /** When true the form is rendered inside a compact sheet (no hero/header). */
  compact?: boolean;
  /** Called after a successful auth; the parent decides where to go. */
  onSuccess?: () => void;
  /** Initial mode. */
  initialMode?: AuthMode;
}

function validateLoginId(v: string): string | null {
  if (!v) return "请输入账号";
  if (v.length < 3 || v.length > 64) return "账号长度需 3–64";
  return null;
}
function validatePassword(v: string): string | null {
  if (!v) return "请输入密码";
  if (v.length < 8 || v.length > 128) return "密码长度需 8–128";
  if (!/[a-zA-Z]/.test(v) || !/\d/.test(v)) return "密码需包含字母和数字";
  return null;
}
function validateNickname(v: string): string | null {
  if (!v) return "请输入昵称";
  if (v.length < 1 || v.length > 30) return "昵称长度需 1–30";
  return null;
}

export function AuthForm({ compact, onSuccess, initialMode = "login" }: Props) {
  const hover = useHoverConfig();
  const { login, register, guest } = useAuth();
  const [mode, setMode] = useState<AuthMode>(initialMode);
  const [loginId, setLoginId] = useState("");
  const [password, setPassword] = useState("");
  const [nickname, setNickname] = useState("");
  const [code, setCode] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [state, setState] = useState<FormState>("idle");
  const [error, setError] = useState<string | null>(null);
  const [resetLoginId, setResetLoginId] = useState("");

  const reset = () => {
    setError(null);
    setState("idle");
  };

  const switchMode = (m: AuthMode) => {
    setMode(m);
    reset();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Client-side validation aligned with harmonyapp LoginPage.
    let fieldErr: string | null = null;
    if (mode === "login") fieldErr = validateLoginId(loginId) ?? validatePassword(password);
    else if (mode === "register") fieldErr = validateLoginId(loginId) ?? validateNickname(nickname) ?? validatePassword(password);
    else if (mode === "guest") fieldErr = validateNickname(nickname);
    else if (mode === "forgot") fieldErr = validateLoginId(resetLoginId);
    else if (mode === "reset") fieldErr = validateLoginId(resetLoginId) ?? (code ? null : "请输入验证码") ?? validatePassword(newPassword);
    if (fieldErr) { setError(fieldErr); return; }

    setState("submitting");
    try {
      let res;
      if (mode === "login") res = await login({ login_id: loginId, password });
      else if (mode === "register") res = await register({ login_id: loginId, nickname, password });
      else if (mode === "guest") res = await guest({ nickname });
      else if (mode === "forgot") {
        res = await import("../../../api").then(({ requestPasswordReset }) =>
          requestPasswordReset({ login_id: resetLoginId, client_type: "web" }));
        if (res.code === 0) {
          // Spec: always returns success (anti-enumeration) → move to reset step.
          setMode("reset");
          setState("idle");
          return;
        }
      } else if (mode === "reset") {
        res = await import("../../../api").then(({ resetPassword }) =>
          resetPassword({ login_id: resetLoginId, verification_code: code, new_password: newPassword, client_type: "web" }));
      }
      if (res && res.code === 0) {
        onSuccess?.();
        return;
      }
      // Business failure: surface the server message (spec: message is for display).
      setError(res?.message ?? "操作失败，请重试");
      setState("error");
    } catch {
      setError("网络错误，请稍后重试");
      setState("error");
    }
  };

  const submitting = state === "submitting";

  return (
    <form onSubmit={handleSubmit} className="flex flex-col" style={{ gap: compact ? 12 : 16 }}>
      {mode === "register" && (
        <Field label="昵称" value={nickname} onChange={setNickname} placeholder="1–30 个字符" type="text" disabled={submitting} />
      )}
      {mode === "guest" && (
        <Field label="昵称" value={nickname} onChange={setNickname} placeholder="给自己起个名字" type="text" disabled={submitting} />
      )}

      {(mode === "login" || mode === "register") && (
        <>
          <Field label="账号" value={loginId} onChange={setLoginId} placeholder="手机号 / 邮箱 / 用户名" type="text" disabled={submitting} />
          <Field label="密码" value={password} onChange={setPassword} placeholder="8–128 位，含字母和数字" type="password" disabled={submitting} />
        </>
      )}

      {(mode === "forgot" || mode === "reset") && (
        <>
          <Field label="账号" value={resetLoginId} onChange={setResetLoginId} placeholder="注册时的手机号 / 邮箱" type="text" disabled={submitting} />
          {mode === "reset" && (
            <>
              <Field label="验证码" value={code} onChange={setCode} placeholder="输入收到的验证码" type="text" disabled={submitting} />
              <Field label="新密码" value={newPassword} onChange={setNewPassword} placeholder="8–128 位，含字母和数字" type="password" disabled={submitting} />
            </>
          )}
        </>
      )}

      {error && (
        <p style={{ fontSize: 12, color: "var(--semantic-hot)" }} role="alert">{error}</p>
      )}

      <motion.button
        type="submit"
        whileTap={hover.tap as any}
        disabled={submitting}
        className="w-full rounded-[14px] py-3 text-[15px] font-medium transition-opacity"
        style={{
          background: "var(--neu-accent-primary)",
          color: "white",
          opacity: submitting ? 0.6 : 1,
        }}
      >
        {submitting ? "处理中…" : submitLabel(mode)}
      </motion.button>

      {/* Mode switcher */}
      <div className="flex flex-wrap items-center justify-center" style={{ gap: 14, marginTop: 4 }}>
        {mode !== "login" && mode !== "forgot" && mode !== "reset" && (
          <LinkBtn onClick={() => switchMode("login")}>已有账号？登录</LinkBtn>
        )}
        {mode === "login" && (
          <>
            <LinkBtn onClick={() => switchMode("register")}>注册新账号</LinkBtn>
            <LinkBtn onClick={() => switchMode("forgot")}>忘记密码</LinkBtn>
          </>
        )}
        {mode === "register" && <LinkBtn onClick={() => switchMode("login")}>已有账号？登录</LinkBtn>}
        {mode === "forgot" && <LinkBtn onClick={() => switchMode("login")}>返回登录</LinkBtn>}
        {mode === "reset" && <LinkBtn onClick={() => switchMode("login")}>返回登录</LinkBtn>}
        {mode !== "guest" && mode !== "forgot" && mode !== "reset" && (
          <LinkBtn onClick={() => switchMode("guest")}>随便逛逛（访客）</LinkBtn>
        )}
      </div>
    </form>
  );
}

function submitLabel(mode: AuthMode): string {
  switch (mode) {
    case "login": return "登录";
    case "register": return "注册并登录";
    case "guest": return "以访客身份继续";
    case "forgot": return "发送验证码";
    case "reset": return "重置密码并登录";
  }
}

function Field({ label, value, onChange, placeholder, type, disabled }: {
  label: string; value: string; onChange: (v: string) => void;
  placeholder?: string; type: "text" | "password"; disabled?: boolean;
}) {
  return (
    <label className="flex flex-col" style={{ gap: 6 }}>
      <span style={{ fontSize: 11, letterSpacing: "0.08em", color: "var(--neu-text-muted)" }}>{label}</span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        autoComplete={type === "password" ? "current-password" : "off"}
        className="rounded-[12px] px-3.5 py-2.5 text-[15px] outline-none transition-shadow"
        style={{
          background: "var(--neu-bg)",
          color: "var(--neu-text)",
          border: "1px solid var(--neu-border)",
        }}
      />
    </label>
  );
}

function LinkBtn({ onClick, children }: { onClick: () => void; children: React.ReactNode }) {
  return (
    <button type="button" onClick={onClick}
      className="text-[12px] transition-opacity hover:opacity-70"
      style={{ color: "var(--neu-accent-primary)" }}>
      {children}
    </button>
  );
}
