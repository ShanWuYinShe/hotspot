-- 注册验证码（spec api_spec_public.md auth/send_register_code）：login_id 在发送
-- 验证码时尚未注册，故不 FK 用户/账号表（参照 auth_password_reset_requests 结构，
-- 二者差异仅在归属：注册码属于「未来账号」，重置码属于「既有账号」）。
-- 无论 login_id 是否已被注册均落一条记录（防枚举：已占用的 login_id 对应验证码
-- 不生效，注册最终以 10400 拒绝）。
CREATE TABLE IF NOT EXISTS auth_register_verification_codes (
  request_id TEXT PRIMARY KEY,
  login_id TEXT NOT NULL,
  verification_code TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  consumed_at TIMESTAMPTZ NULL
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_auth_register_codes_login_id_created_at
  ON auth_register_verification_codes(login_id, created_at DESC);
