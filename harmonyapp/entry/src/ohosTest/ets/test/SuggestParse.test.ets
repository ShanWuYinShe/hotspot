/**
 * Suggest parse logic — verifies that search/suggest results correctly
 * extract display info from nested spec shapes.
 *
 * Spec: UserSuggestItem = { user: UserInfo, bio?, location? }
 *       GroupSuggestItem = { group: GroupInfo, description?, location? }
 */

import { describe, it, expect } from '@ohos/hypium';

interface UserInfo { user_id: string; nickname: string; avatar_url?: string; }
interface GroupInfo { group_id: string; name: string; avatar_url?: string; members_count: number; }

function parseUserSuggest(raw: Record<string, Object>): string {
  const user = (raw.user ?? raw) as Record<string, Object>;
  return (user.nickname ?? user.user_id ?? '') as string;
}

function parseGroupSuggest(raw: Record<string, Object>): string {
  const group = (raw.group ?? raw) as Record<string, Object>;
  return (group.name ?? group.group_id ?? '') as string;
}

export default function suggestParseTest() {
  describe('Suggest_Parse_Nested', () => {

    it('extracts_nickname_from_nested_user', () => {
      const suggest: Record<string, Object> = {
        user: { user_id: 'u1', nickname: '鹿鸣' } as Object,
        bio: 'something',
      };
      expect(parseUserSuggest(suggest)).assertEqual('鹿鸣');
    });

    it('falls_back_to_user_id_when_nickname_missing', () => {
      const suggest: Record<string, Object> = {
        user: { user_id: 'u2', nickname: '' } as Object,
      };
      expect(parseUserSuggest(suggest)).assertEqual('u2');
    });

    it('flat_layout_also_works', () => {
      const suggest: Record<string, Object> = {
        user_id: 'u3', nickname: '野原',
      } as unknown as Record<string, Object>;
      expect(parseUserSuggest(suggest)).assertEqual('野原');
    });

    it('extracts_name_from_nested_group', () => {
      const suggest: Record<string, Object> = {
        group: { group_id: 'g1', name: '回龙窝胶片会', members_count: 42 } as Object,
      };
      expect(parseGroupSuggest(suggest)).assertEqual('回龙窝胶片会');
    });

    it('flat_group_also_works', () => {
      const suggest: Record<string, Object> = {
        group_id: 'g2', name: '云龙湖晨跑团', members_count: 89,
      } as unknown as Record<string, Object>;
      expect(parseGroupSuggest(suggest)).assertEqual('云龙湖晨跑团');
    });
  });
}
