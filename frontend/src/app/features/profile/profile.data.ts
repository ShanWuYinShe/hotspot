/* Profile mock data — extracted from Profile.tsx to keep the page file lean. */
import type { ProfileSection } from "../../lib/routes";

export const USER_AVATAR =
  "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400";

const GALLERY_POST_SEED = [
  { img: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 124, comments: 18 },
  { img: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 89,  comments: 7 },
  { img: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 312, comments: 44 },
  { img: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 56,  comments: 3 },
  { img: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 201, comments: 28 },
  { img: "https://images.unsplash.com/photo-1758613172074-7275c3455034?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 77,  comments: 9 },
  { img: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 143, comments: 21 },
  { img: "https://images.unsplash.com/photo-1759734282634-b4958ea9a6e0?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 98,  comments: 12 },
  { img: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?crop=entropy&cs=tinysrgb&fit=max&fm=jpg&q=80&w=400", likes: 65,  comments: 5 },
];

// Mock pagination: a real profile owns many more moments than fit on screen,
// so the seed images are cycled with id/like offsets to give the BSP canvas
// enough cards to exercise incremental loading + virtualization. Replaced by
// the API once pagination is wired through.
export const GALLERY_POSTS = Array.from({ length: 72 }, (_, i) => {
  const seed = GALLERY_POST_SEED[i % GALLERY_POST_SEED.length];
  const cycle = Math.floor(i / GALLERY_POST_SEED.length);
  return { id: i + 1, img: seed.img, likes: seed.likes + cycle * 13, comments: seed.comments + cycle * 2 };
});

export const LIKED_POSTS = [...GALLERY_POSTS].reverse();
export const SAVED_POSTS = GALLERY_POSTS.slice(0, 36);

export const STATS: { label: string; value: string; section: ProfileSection | null }[] = [
  { label: "帖子",   value: "48",  section: "moments" },
  { label: "关注者", value: "312", section: "followers" },
  { label: "足迹",   value: "27",  section: null },
];

const MOCK_GROUP_SEED = [
  { name: "回龙窝胶片会", members: 128, online: 14, avatar: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=100", banner: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=400", desc: "记录生活中每一个美好瞬间", role: "管理员", lastActive: "2分钟前" },
  { name: "徐州骑行圈",   members: 256, online: 31, avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=100", banner: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=400", desc: "用骑行丈量城市的每一条街道", role: "成员",   lastActive: "1小时前" },
  { name: "城市探索组",   members: 89,  online: 8,  avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=100", banner: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=400", desc: "发现城市中隐藏的惊喜角落",   role: "成员",   lastActive: "3小时前" },
  { name: "老街手冲地图", members: 412, online: 67, avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", banner: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=400", desc: "跟着味蕾去旅行，探索每一道美食", role: "管理员", lastActive: "刚刚" },
];

// Mock pagination: a user's group directory is rarely just a handful. Cycled
// with id/member offsets so "我的群组" can scroll-load like the real backend.
// Replaced by the API once pagination is wired through.
export const MOCK_GROUPS = Array.from({ length: 48 }, (_, i) => {
  const seed = MOCK_GROUP_SEED[i % MOCK_GROUP_SEED.length];
  const cycle = Math.floor(i / MOCK_GROUP_SEED.length);
  return {
    ...seed,
    id: i + 1,
    name: cycle === 0 ? seed.name : `${seed.name} · 第${cycle + 1}分部`,
    members: seed.members + cycle * 41,
    online: seed.online + cycle * 5,
  };
});

export const MOCK_FRIENDS = [
  { id: 1, name: "青砚",   handle: "@qingyan.xz", avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", online: true,  mutual: 12, tag: "摄影" },
  { id: 2, name: "阿禾",   handle: "@ahe.wander", avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", online: true,  mutual: 8,  tag: "旅行" },
  { id: 3, name: "野原",   handle: "@yeyuan.m",   avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100", online: false, mutual: 5,  tag: "音乐" },
  { id: 4, name: "Mika",   handle: "@mika.t",     avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=100", online: true,  mutual: 3,  tag: "美食" },
  { id: 5, name: "林一",   handle: "@linyi",      avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=100", online: false, mutual: 15, tag: "户外" },
  { id: 6, name: "南枝",   handle: "@nanzhi",     avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", online: true,  mutual: 6,  tag: "艺术" },
];

export const MOCK_FOLLOWING = [
  { id: 1, name: "Studio Ghibli Art", handle: "@ghibli_art",  avatar: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=100", desc: "独立艺术工作室 · 动画爱好者",  posts: 234, latestImg: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=80" },
  { id: 2, name: "Urban Explorer",    handle: "@urban_exp",   avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=100", desc: "城市探索达人 · 街头摄影师",    posts: 189, latestImg: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=80" },
  { id: 3, name: "Foodie Diary",      handle: "@foodie_d",    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", desc: "美食博主 · 环球美食探索",      posts: 412, latestImg: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80" },
  { id: 4, name: "Photo Master",      handle: "@photo_m",     avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", desc: "风光摄影师 · 全球旅行",         posts: 876, latestImg: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80" },
  { id: 5, name: "City Nights",       handle: "@city_nights", avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100", desc: "夜拍爱好者 · 城市光影",         posts: 156, latestImg: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80" },
];

export const MOCK_FANS = [
  { id: 1, name: "小满",   handle: "@xiaoman.life", avatar: "https://images.unsplash.com/photo-1758613172074-7275c3455034?w=100", followBack: false, joinedDays: 2 },
  { id: 2, name: "阿川",   handle: "@achuan",       avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=100", followBack: true,  joinedDays: 5 },
  { id: 3, name: "宋悦",   handle: "@song.yue",     avatar: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=100", followBack: false, joinedDays: 14 },
  { id: 4, name: "Yuki",   handle: "@yuki.xz",      avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=100", followBack: true,  joinedDays: 30 },
  { id: 5, name: "吴远",   handle: "@wu.yuan",      avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=100", followBack: false, joinedDays: 7 },
  { id: 6, name: "陈芊",   handle: "@chen.qian",    avatar: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=100", followBack: false, joinedDays: 1 },
  { id: 7, name: "林杰",   handle: "@lin.jie",      avatar: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=100", followBack: true,  joinedDays: 60 },
];
