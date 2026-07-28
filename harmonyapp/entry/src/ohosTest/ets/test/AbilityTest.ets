/**
 * Hypium test suite entry point.
 * Run via DevEco Studio or hdc: hdc shell aa test...
 *
 * Covers:
 *   - ErrorCode classification (10100/10201/10500/22000-22002)
 *   - Reaction derive logic (reacted_by_me → is_liked, is_favorited → saved)
 *   - IdempotencyKey format ({clientId}-{ts}-{rand})
 *   - DeviceId UUID v4 format
 *   - Suggest nested parse (UserSuggestItem/GroupSuggestItem)
 */

import { describe, beforeAll, afterAll } from '@ohos/hypium';
import errorCodeTest from './ErrorCode.test';
import reactionDeriveTest from './ReactionDerive.test';
import idempotencyKeyTest from './IdempotencyKey.test';
import deviceIdTest from './DeviceId.test';
import suggestParseTest from './SuggestParse.test';

export default function abilityTest() {
  describe('HotSpot_Spec_Conformance_Suite', () => {
    beforeAll(() => {
      console.info('=== HotSpot spec conformance tests starting ===');
    });

    afterAll(() => {
      console.info('=== HotSpot spec conformance tests complete ===');
    });

    errorCodeTest();
    reactionDeriveTest();
    idempotencyKeyTest();
    deviceIdTest();
    suggestParseTest();
  });
}
