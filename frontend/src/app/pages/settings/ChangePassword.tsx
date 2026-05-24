import { useState } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Loader2 } from "lucide-react";
import { motion } from "motion/react";
import { toast } from "sonner";

import { authService } from "@/lib/api/services";
import { isSuccess } from "@/lib/api/client";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";

const hasLetterAndDigit = (v: string) => /[A-Za-z]/.test(v) && /\d/.test(v);

function validate(value: string, rules: Array<{ test: (v: string) => string | undefined }>): string | undefined {
  for (const rule of rules) {
    const err = rule.test(value);
    if (err) return err;
  }
  return undefined;
}

export function ChangePassword() {
  const navigate = useNavigate();
  useDocumentTitle("修改密码");

  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [touched, setTouched] = useState<Record<string, boolean>>({});

  const markTouched = (field: string) => setTouched((prev) => ({ ...prev, [field]: true }));
  const markAllTouched = () => setTouched({ oldPassword: true, newPassword: true, confirmPassword: true });

  const oldPasswordError = touched.oldPassword
    ? validate(oldPassword, [{ test: (v) => (!v ? "请输入当前密码" : undefined) }])
    : undefined;

  const newPasswordError = touched.newPassword
    ? validate(newPassword, [
        { test: (v) => (!v ? "请输入新密码" : undefined) },
        { test: (v) => (v.length < 8 ? "新密码至少 8 个字符" : undefined) },
        { test: (v) => (v.length > 128 ? "新密码最多 128 个字符" : undefined) },
        { test: (v) => (v && !hasLetterAndDigit(v) ? "新密码需同时包含字母和数字" : undefined) },
      ])
    : undefined;

  const confirmPasswordError = touched.confirmPassword
    ? validate(confirmPassword, [
        { test: (v) => (!v ? "请确认新密码" : undefined) },
        { test: (v) => (v && v !== newPassword ? "两次密码不一致" : undefined) },
      ])
    : undefined;

  const hasErrors = () =>
    !!validate(oldPassword, [{ test: (v) => (!v ? "x" : undefined) }]) ||
    !!validate(newPassword, [
      { test: (v) => (!v ? "x" : undefined) },
      { test: (v) => (v.length < 8 ? "x" : undefined) },
      { test: (v) => (v && !hasLetterAndDigit(v) ? "x" : undefined) },
    ]) ||
    !!validate(confirmPassword, [
      { test: (v) => (!v ? "x" : undefined) },
      { test: (v) => (v !== newPassword ? "x" : undefined) },
    ]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    markAllTouched();
    if (hasErrors()) return;

    setIsSaving(true);
    try {
      const response = await authService.changePassword({
        old_password: oldPassword,
        new_password: newPassword,
      });
      if (isSuccess(response)) {
        toast.success("密码已更新");
        setOldPassword("");
        setNewPassword("");
        setConfirmPassword("");
        setTouched({});
      } else {
        toast.error(response.message || "修改失败");
      }
    } catch {
      toast.error("修改失败");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <>
      {/* Header */}
      <div
        className="flex-shrink-0 flex items-center gap-3 px-4"
        style={{ height: 56, borderBottom: "1px solid var(--neu-border)" }}
      >
        <motion.button
          whileTap={{ scale: 0.88 }}
          onClick={() => navigate(-1)}
          className="flex items-center justify-center rounded-full outline-none"
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
        <p style={{ fontSize: 16, fontWeight: 600, color: "var(--neu-text)" }}>
          修改密码
        </p>
      </div>

      {/* Content */}
      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
        className="flex flex-col px-5 pt-4 pb-10 gap-5"
        style={{ maxWidth: 560, marginLeft: "auto", marginRight: "auto", width: "100%" }}
      >
        <div
          className="rounded-[22px] p-5"
          style={{
            background: "var(--neu-bg)",
            boxShadow:
              "7px 7px 16px var(--neu-shadow-dark), -7px -7px 16px var(--neu-shadow-light)",
            border: "1px solid var(--neu-border)",
          }}
        >
          <p
            style={{
              fontSize: 14,
              fontWeight: 600,
              color: "var(--neu-text)",
              marginBottom: 16,
            }}
          >
            账户安全
          </p>
          <p
            style={{
              fontSize: 12,
              color: "var(--neu-text-muted)",
              marginBottom: 16,
            }}
          >
            请确保新密码足够安全，避免与其他平台重复
          </p>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {/* Old password */}
            <div className="flex flex-col gap-2">
              <label htmlFor="old-password" style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
                当前密码
              </label>
              <input
                id="old-password"
                type="password"
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                onBlur={() => markTouched("oldPassword")}
                className="w-full rounded-[18px] px-4 py-2.5 outline-none"
                style={{
                  border: `1px solid ${oldPasswordError ? "#EF4444" : "var(--neu-border)"}`,
                  background: "var(--neu-bg)",
                  color: "var(--neu-text)",
                }}
              />
              {oldPasswordError && (
                <p style={{ fontSize: 12, color: "#EF4444" }}>{oldPasswordError}</p>
              )}
            </div>

            {/* New password */}
            <div className="flex flex-col gap-2">
              <label htmlFor="new-password" style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
                新密码
              </label>
              <input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                onBlur={() => markTouched("newPassword")}
                className="w-full rounded-[18px] px-4 py-2.5 outline-none"
                style={{
                  border: `1px solid ${newPasswordError ? "#EF4444" : "var(--neu-border)"}`,
                  background: "var(--neu-bg)",
                  color: "var(--neu-text)",
                }}
              />
              {newPasswordError && (
                <p style={{ fontSize: 12, color: "#EF4444" }}>{newPasswordError}</p>
              )}
            </div>

            {/* Confirm password */}
            <div className="flex flex-col gap-2">
              <label
                htmlFor="confirm-password"
                style={{ fontSize: 13, color: "var(--neu-text-muted)" }}
              >
                确认新密码
              </label>
              <input
                id="confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                onBlur={() => markTouched("confirmPassword")}
                className="w-full rounded-[18px] px-4 py-2.5 outline-none"
                style={{
                  border: `1px solid ${confirmPasswordError ? "#EF4444" : "var(--neu-border)"}`,
                  background: "var(--neu-bg)",
                  color: "var(--neu-text)",
                }}
              />
              {confirmPasswordError && (
                <p style={{ fontSize: 12, color: "#EF4444" }}>{confirmPasswordError}</p>
              )}
            </div>

            {/* Submit */}
            <motion.button
              whileTap={{ scale: 0.96 }}
              type="submit"
              disabled={isSaving}
              className="flex h-11 w-full items-center justify-center rounded-full text-sm font-medium transition-opacity hover:opacity-90 disabled:opacity-50"
              style={{
                background: "var(--neu-accent-primary)",
                color: "var(--text-on-accent, white)",
                boxShadow: "0 4px 14px var(--neu-glow)",
              }}
            >
              {isSaving && (
                <Loader2
                  style={{ width: 16, height: 16, marginRight: 8 }}
                  className="animate-spin"
                />
              )}
              保存修改
            </motion.button>
          </form>
        </div>
      </motion.div>
    </>
  );
}
