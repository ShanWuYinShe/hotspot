/**
 * AccountValidation —— 账号输入校验纯函数（登录/注册/资料页共享）。
 * 从 LoginPage / ProfilePage 抽取的重复实现；语义以 spec user.md / auth.md 为准。
 */

/** 密码需同时含字母与数字（spec auth.md 密码策略） */
export function hasLetterAndDigit(value: string): boolean {
  let hasLetter: boolean = false;
  let hasDigit: boolean = false;
  for (let i = 0; i < value.length; i++) {
    const code = value.charCodeAt(i);
    if ((code >= 65 && code <= 90) || (code >= 97 && code <= 122)) {
      hasLetter = true;
    }
    if (code >= 48 && code <= 57) {
      hasDigit = true;
    }
    if (hasLetter && hasDigit) {
      return true;
    }
  }
  return false;
}

/** 密码长度 8-128 且同时含字母与数字 */
export function isValidPassword(value: string): boolean {
  return value.length >= 8 && value.length <= 128 && hasLetterAndDigit(value);
}

/** 昵称：trim 后 1-30 字符 */
export function isValidNickname(nickname: string): boolean {
  const trimmed = nickname.trim();
  return trimmed.length >= 1 && trimmed.length <= 30;
}

/** 访客昵称可选：留空合法（提交 null，服务端自动生成），非空时限 30 字符 */
export function isValidOptionalNickname(nickname: string): boolean {
  const trimmed = nickname.trim();
  return trimmed.length === 0 || trimmed.length <= 30;
}

/** 手机号（纯数字）或邮箱（含 '@'）型 login_id 需验证码（spec public.md → auth/register） */
export function isPhoneOrEmailLoginId(loginId: string): boolean {
  const trimmed = loginId.trim();
  if (trimmed.includes('@')) {
    return true;
  }
  return trimmed.length > 0 && /^[0-9]+$/.test(trimmed);
}

/**
 * 用户名型 login_id（upgrade 场景）：拒绝 '@' 与纯数字（手机号/邮箱需验证码服务，
 * 接入前拒绝），长度 3-64。登录场景允许手机号/邮箱，勿用本函数。
 */
export function isValidUsernameLoginId(loginId: string): boolean {
  const trimmed = loginId.trim();
  if (trimmed.includes('@') || /^\d+$/.test(trimmed)) {
    return false;
  }
  return trimmed.length >= 3 && trimmed.length <= 64;
}
