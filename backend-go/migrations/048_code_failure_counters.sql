-- 验证码防爆破：失败计数落码行（登录侧 FailedLoginTracker 的 PG 等价）。
-- ResetPassword/VerifyRegisterCode 每次验码失败给 login_id 最新有效码
-- failed_attempts + 1，满 5 次直接作废该码（静默烧码，对外仍返回原错误，
-- 不新增可探测口径）。计数随码行消亡，无需清理任务；多实例天然共享。
ALTER TABLE auth_password_reset_requests
  ADD COLUMN IF NOT EXISTS failed_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE auth_register_verification_codes
  ADD COLUMN IF NOT EXISTS failed_attempts INTEGER NOT NULL DEFAULT 0;
