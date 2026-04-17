/**
 * Mock data for MSW
 */

export const MOCK_CURRENT_USER_ID = 'user_mock_001';

export const MOCK_OTHER_USER_ID = 'user_mock_002';

export const mockClientConfig = {
	features: {
		enable_public_registration: true,
		enable_guest_login: true,
		enable_moments: true,
		enable_nearby_people: true
	},
	limits: {
		max_group_members: 500,
		max_moment_tags: 10,
		max_moment_images: 9,
		moment_description_max_length: 2000,
		comment_max_length: 500,
		search_radius_max: 50000
	},
	contact: {
		support_email: 'mock@example.com',
		privacy_policy_url: 'https://example.com/privacy',
		terms_of_service_url: 'https://example.com/terms',
		official_website: 'https://example.com'
	},
	defaults: {
		default_location: {
			point: { latitude: 39.9042, longitude: 116.4074 },
			location_name: '北京市'
		},
		default_user_avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=default',
		default_group_avatar: 'https://api.dicebear.com/7.x/identicon/svg?seed=default'
	}
};

export const mockMyProfile = {
	user_id: MOCK_CURRENT_USER_ID,
	nickname: '测试用户',
	user_type: 'normal' as const,
	avatar_url: 'https://api.dicebear.com/7.x/avataaars/svg?seed=mock_user_1',
	bio: '这是一条测试动态',
	tags: ['测试'],
	gender: 'male' as const,
	created_at: '2025-01-01T00:00:00Z',
	account_status: 'normal' as const,
	location: {
		point: { latitude: 39.9042, longitude: 116.4074 },
		location_name: '北京市'
	},
	presence_status: 'online' as const,
	last_online: new Date().toISOString(),
	stats: {
		moments_count: 10,
		friends_count: 20,
		followers_count: 100,
		following_count: 50,
		groups_count: 3
	}
};

export const mockOtherUser = {
	user_id: MOCK_OTHER_USER_ID,
	nickname: '其他用户',
	user_type: 'normal' as const,
	avatar_url: 'https://api.dicebear.com/7.x/avataaars/svg?seed=mock_user_2',
	bio: '其他用户的动态',
	tags: [],
	gender: 'female' as const,
	created_at: '2025-01-01T00:00:00Z',
	account_status: 'normal' as const,
	location: {
		point: { latitude: 39.9142, longitude: 116.4174 },
		location_name: '北京市东城区'
	},
	presence_status: 'online' as const,
	last_online: new Date().toISOString(),
	stats: {
		moments_count: 25,
		friends_count: 30,
		followers_count: 200,
		following_count: 80,
		groups_count: 5
	}
};

// In-memory store for moments (mutable for testing)
export interface MockMoment {
	moment_id: string;
	user_id: string;
	description: string;
	location: {
		point: { latitude: number; longitude: number };
		location_name: string;
	};
	tags: string[];
	images: { url: string; file_id: string }[];
	privacy_level: 'public' | 'private';
	created_at: string;
	updated_at: string;
	likes_count: number;
	comments_count: number;
	my_state?: { is_liked?: boolean; is_favorited?: boolean };
}

export const mockMomentsStore: Map<string, MockMoment> = new Map();
export const mockGroupsStore: Map<string, Record<string, unknown>> = new Map();
export const mockUsersStore: Map<string, Record<string, unknown>> = new Map();

// Pre-seed a moment owned by the mock user (for edit testing)
const seedMomentId = 'moment_mock_001';
mockMomentsStore.set(seedMomentId, {
	moment_id: seedMomentId,
	user_id: MOCK_CURRENT_USER_ID,
	description: '这是我的一条测试动态，用来验证编辑功能。标签：#测试 #美食',
	location: {
		point: { latitude: 39.9042, longitude: 116.4074 },
		location_name: '北京市朝阳区'
	},
	tags: ['测试', '美食'],
	images: [],
	privacy_level: 'public',
	created_at: new Date(Date.now() - 3600 * 1000).toISOString(),
	updated_at: new Date(Date.now() - 3600 * 1000).toISOString(),
	likes_count: 5,
	comments_count: 2,
	my_state: { is_liked: false, is_favorited: false }
});

export function createMockMomentId(): string {
	return `moment_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
}

export function momentToItem(m: MockMoment, currentUserId: string): Record<string, unknown> {
	return {
		moment_id: m.moment_id,
		user: {
			user_id: m.user_id,
			nickname: m.user_id === MOCK_CURRENT_USER_ID ? mockMyProfile.nickname : mockOtherUser.nickname,
			avatar_url: m.user_id === MOCK_CURRENT_USER_ID ? mockMyProfile.avatar_url : mockOtherUser.avatar_url,
			user_type: 'normal'
		},
		description: m.description,
		location: m.location,
		tags: m.tags,
		images: m.images,
		privacy_level: m.privacy_level,
		created_at: m.created_at,
		updated_at: m.updated_at,
		likes_count: m.likes_count,
		comments_count: m.comments_count,
		my_state: {
			is_liked: m.user_id === currentUserId ? false : (m.my_state?.is_liked ?? false),
			is_favorited: m.my_state?.is_favorited ?? false
		}
	};
}
