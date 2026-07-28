/**
 * IdempotencyKey format validation.
 * Spec: {client_id}-{timestamp_ms}-{random}, length 32–64,
 * charset [a-zA-Z0-9-_].
 */

import { describe, it, expect } from '@ohos/hypium';
import { util } from '@kit.ArkTS';

function generateIdempotencyKey(): string {
  const clientId = util.generateRandomUUID(false).split('-').join('').substring(0, 8);
  const timestampMs = Date.now().toString();
  const randomPart = util.generateRandomUUID(false).split('-').join('').substring(0, 12);
  return `${clientId}-${timestampMs}-${randomPart}`;
}

const VALID_CHARS = /^[a-zA-Z0-9\-_]+$/;

export default function idempotencyKeyTest() {
  describe('IdempotencyKey_Format', () => {

    it('length_is_within_32_to_64', () => {
      const key = generateIdempotencyKey();
      expect(key.length >= 32 && key.length <= 64).assertTrue();
    });

    it('contains_two_hyphens', () => {
      const key = generateIdempotencyKey();
      const hyphens = (key.match(/-/g) || []).length;
      expect(hyphens).assertEqual(2);
    });

    it('only_allowed_charset', () => {
      const key = generateIdempotencyKey();
      expect(VALID_CHARS.test(key)).assertTrue();
    });

    it('same_ts_produces_different_keys', () => {
      // Two keys generated in rapid succession should differ.
      const k1 = generateIdempotencyKey();
      const k2 = generateIdempotencyKey();
      expect(k1 === k2).assertFalse();
    });

    it('starts_with_client_id_prefix', () => {
      const key = generateIdempotencyKey();
      // First segment before '-' is client_id (8 hex-ish chars).
      const firstSegment = key.split('-')[0];
      expect(firstSegment.length).assertEqual(8);
    });
  });
}
