/**
 * 客户端 Cookie 工具
 */

'use client';

const ACCESS_TOKEN_KEY = 'access_token';

export function setAccessTokenCookie(token: string, expiresAt?: string) {
  if (typeof document === 'undefined') return;
  const expires = expiresAt ? `; Expires=${new Date(expiresAt).toUTCString()}` : '';
  document.cookie = `${ACCESS_TOKEN_KEY}=${encodeURIComponent(token)}; Path=/; SameSite=Lax${expires}`;
}

export function clearAccessTokenCookie() {
  if (typeof document === 'undefined') return;
  document.cookie = `${ACCESS_TOKEN_KEY}=; Path=/; Max-Age=0; SameSite=Lax`;
}
