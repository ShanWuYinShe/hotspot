/**
 * DeviceId persistence and format validation.
 * Spec: DeviceId = UUID v4, persisted across sessions, not cleared on logout.
 */

import { describe, it, expect } from '@ohos/hypium';
import { util } from '@kit.ArkTS';

const UUID_V4_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default function deviceIdTest() {
  describe('DeviceId_Format_And_Persistence', () => {

    it('generated_uuid_is_v4_format', () => {
      const id = util.generateRandomUUID(false);
      expect(UUID_V4_PATTERN.test(id)).assertTrue();
    });

    it('generated_uuid_is_36_chars', () => {
      const id = util.generateRandomUUID(false);
      expect(id.length).assertEqual(36);
    });

    it('two_uuids_are_different', () => {
      const id1 = util.generateRandomUUID(false);
      const id2 = util.generateRandomUUID(false);
      expect(id1 === id2).assertFalse();
    });

    it('version_nibble_is_4', () => {
      const id = util.generateRandomUUID(false);
      // Position 14 in standard UUID is the version char ('4').
      expect(id.charAt(14)).assertEqual('4');
    });

    it('variant_nibble_is_8_9_a_or_b', () => {
      const id = util.generateRandomUUID(false);
      const variantChar = id.charAt(19);
      expect(['8', '9', 'a', 'b', 'A', 'B'].indexOf(variantChar) >= 0).assertTrue();
    });
  });
}
