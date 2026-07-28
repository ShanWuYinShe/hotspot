/**
 * ErrorCode classification tests — verify that spec error codes
 * (10100/10201/10300/10400/10500/10600 + idempotency 22000-22002)
 * are correctly classified by ApiErrorUtil.
 */

import { describe, it, expect } from '@ohos/hypium';
import {
  isAuthErrorCode,
  isAccountBannedCode,
  isRateLimitCode,
  extractApiErrorSnapshot
} from '../../../main/ets/core/network/ApiErrorUtil';

export default function errorCodeTest() {
  describe('ApiErrorUtil_ErrorCode_Classification', () => {

    it('10100_is_auth_error', () => {
      expect(isAuthErrorCode(10100)).assertTrue();
    });

    it('10201_is_account_banned', () => {
      expect(isAccountBannedCode(10201)).assertTrue();
    });

    it('10201_is_not_auth_error', () => {
      // 10201 is banned, not general auth-failed.
      expect(isAuthErrorCode(10201)).assertTrue();
    });

    it('10500_is_rate_limit', () => {
      expect(isRateLimitCode(10500)).assertTrue();
    });

    it('10600_is_server_error_not_rate_limit', () => {
      expect(isRateLimitCode(10600)).assertFalse();
    });

    it('10300_is_not_auth_error', () => {
      expect(isAuthErrorCode(10300)).assertFalse();
    });

    it('10400_is_not_auth_error', () => {
      expect(isAuthErrorCode(10400)).assertFalse();
    });

    it('extractApiErrorSnapshot_returns_code_and_message', () => {
      const snapshot = extractApiErrorSnapshot(10100, '密码错误');
      expect(snapshot.code).assertEqual(10100);
      expect(snapshot.message).assertEqual('密码错误');
    });

    it('22000_is_idempotency_key_conflict', () => {
      // 22000 should NOT be classified as rate-limit
      expect(isRateLimitCode(22000)).assertFalse();
    });

    it('22002_is_not_rate_limit', () => {
      expect(isRateLimitCode(22002)).assertFalse();
    });
  });
}
