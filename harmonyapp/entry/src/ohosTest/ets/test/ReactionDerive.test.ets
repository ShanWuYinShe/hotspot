/**
 * Reaction derive logic — verifies that liked/saved state is correctly
 * derived from reactions[].reacted_by_me and my_state.is_favorited.
 *
 * Spec: MomentItem.likes_count coexists with reactions aggregate; like
 * state comes from reactions[].reacted_by_me, NOT from a separate is_liked
 * boolean.
 */

import { describe, it, expect } from '@ohos/hypium';

interface Reaction {
  emoji: string;
  count: number;
  reacted_by_me: boolean;
}

interface MomentLikeState {
  moment_id: string;
  reactions?: Reaction[];
  my_state?: { is_favorited: boolean };
}

function deriveLiked(moment: MomentLikeState): boolean {
  return (moment.reactions || []).some(r => r.reacted_by_me);
}

function deriveSaved(moment: MomentLikeState): boolean {
  return moment.my_state?.is_favorited ?? false;
}

export default function reactionDeriveTest() {
  describe('Reaction_Derive_Logic', () => {

    it('no_reactions_means_not_liked', () => {
      expect(deriveLiked({ moment_id: 'm1' })).assertFalse();
    });

    it('reacted_by_me_true_means_liked', () => {
      expect(deriveLiked({
        moment_id: 'm1',
        reactions: [{ emoji: '👍', count: 10, reacted_by_me: true }]
      })).assertTrue();
    });

    it('reacted_by_me_false_means_not_liked', () => {
      expect(deriveLiked({
        moment_id: 'm1',
        reactions: [{ emoji: '👍', count: 10, reacted_by_me: false }]
      })).assertFalse();
    });

    it('multiple_reactions_one_with_me_is_liked', () => {
      expect(deriveLiked({
        moment_id: 'm2',
        reactions: [
          { emoji: '👍', count: 30, reacted_by_me: false },
          { emoji: '❤️', count: 5, reacted_by_me: true },
        ]
      })).assertTrue();
    });

    it('my_state_is_favorited_means_saved', () => {
      expect(deriveSaved({
        moment_id: 'm1',
        my_state: { is_favorited: true },
      })).assertTrue();
    });

    it('no_my_state_means_not_saved', () => {
      expect(deriveSaved({ moment_id: 'm1' })).assertFalse();
    });

    it('is_favorited_false_means_not_saved', () => {
      expect(deriveSaved({
        moment_id: 'm1',
        my_state: { is_favorited: false },
      })).assertFalse();
    });
  });
}
