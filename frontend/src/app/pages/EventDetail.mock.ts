/* ── EventDetail Page Mock Data ─────────────────────────────────── */
/*
 * Figma Make 重新生成 EventDetail.tsx 后，暂无 API 可对接（等待后端 event 接口）
 * 重连时保持 mock 导入即可：
 *   1. import { MOCK_EVENTS, MOCK_DEFAULT_EVENT } from "./EventDetail.mock"
 *   2. 后续接入真实 API hook
 */

export interface MockParticipant {
  id: number; name: string; avatar: string;
}

export interface MockEvent {
  id: number; title: string; tag: string; color: string; bg: string;
  cover: string; description: string; location: string; date: string;
  time: string; host: { name: string; avatar: string };
  participants: MockParticipant[];
  maxParticipants: number; distance: string;
  highlights: string[];
}

export const MOCK_EVENTS: Record<number, MockEvent> = {
  1: {
    id: 1, title: "晨间咖啡会", tag: "社交", color: "#F59E0B", bg: "rgba(245,158,11,0.12)",
    cover: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=1080",
    description: "每周六早上九点，我们聚集在老街咖啡馆，分享一周的所见所闻。无论是设计师、程序员还是艺术家，都欢迎加入！带上你的好奇心就够了。",
    location: "老街咖啡馆·徐州市鼓楼区", date: "2026年4月18日 周六", time: "09:00 - 11:00",
    host: { name: "Aria Sol", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
    participants: [
      { id: 1, name: "Liam",   avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
      { id: 2, name: "Sophia", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
      { id: 3, name: "Marco",  avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" },
      { id: 4, name: "Aiko",   avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=80" },
    ],
    maxParticipants: 20, distance: "0.3km",
    highlights: ["免费咖啡 ☕", "结识新朋友", "分享创意", "每周固定"],
  },
  2: {
    id: 2, title: "街头音乐节", tag: "音乐", color: "#8B5CF6", bg: "rgba(139,92,246,0.12)",
    cover: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=1080",
    description: "在城市的中心，用音乐连接每一颗心。本次街头音乐节将邀请本地独立乐队和歌手进行现场表演，免费入场，欢迎所有人来感受音乐的力量。",
    location: "彭城广场·徐州市云龙区", date: "2026年4月19日 周日", time: "14:00 - 20:00",
    host: { name: "本地向导", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
    participants: [
      { id: 1, name: "Noah",    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
      { id: 2, name: "Isabella",avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
      { id: 3, name: "Kai",     avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" },
    ],
    maxParticipants: 200, distance: "0.7km",
    highlights: ["免费入场 🎸", "现场表演", "多个乐队", "本地艺术家"],
  },
};

export const MOCK_DEFAULT_EVENT = MOCK_EVENTS[1];
