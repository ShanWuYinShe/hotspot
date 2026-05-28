/* ── Location Page Mock Data ────────────────────────────────────── */
/*
 * Figma Make 重新生成 Location.tsx 后，执行 3 步重连：
 *   1. import { XUZHOU_CENTER, MOCK_PINS, MOCK_RECENT_LOCATIONS, MOCK_HOT_LOCATIONS, MOCK_SEARCH_DB } from "./Location.mock"
 *   2. 后续接入 mapService.getHeatCanvas() + feedService.getUsers/Groups/Moments() 构造 pins
 *   3. 用 API 返回的 GeoPoint 代替硬编码坐标
 */

export interface Pin {
  id: number;
  type: "user" | "group" | "post";
  lat: number;
  lng: number;
  count?: number;
  name?: string;
  avatar?: string;
  distance?: string;
  online?: boolean;
  members?: number;
  tag?: string;
  title?: string;
  likes?: number;
  image?: string;
}

export interface LocationResult {
  displayName: string;
  lat: number;
  lng: number;
  emoji?: string;
  desc?: string;
  users?: number;
}

export const XUZHOU_CENTER = { lat: 34.2604, lng: 117.1855 };

export const MOCK_PINS: Pin[] = [
  {
    id: 1,
    type: "user",
    lat: 34.268,
    lng: 117.19,
    count: 1,
    name: "Liam",
    distance: "246m",
    online: true,
    avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",
    tag: "摄影",
  },
  {
    id: 2,
    type: "user",
    lat: 34.255,
    lng: 117.18,
    count: 1,
    name: "Sophia",
    distance: "380m",
    online: true,
    avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",
    tag: "旅行",
  },
  {
    id: 3,
    type: "user",
    lat: 34.272,
    lng: 117.175,
    count: 1,
    name: "Marco",
    distance: "520m",
    online: false,
    avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",
    tag: "音乐",
  },
  {
    id: 10,
    type: "user",
    lat: 34.248,
    lng: 117.195,
    count: 5,
    name: "5人聚集",
  },
  { id: 11, type: "user", lat: 34.275, lng: 117.2, count: 3, name: "3人聚集" },
  {
    id: 20,
    type: "group",
    lat: 34.262,
    lng: 117.185,
    count: 1,
    name: "摄影爱好者",
    members: 28,
    tag: "摄影",
  },
  {
    id: 21,
    type: "group",
    lat: 34.253,
    lng: 117.2,
    count: 1,
    name: "美食打卡群",
    members: 89,
    tag: "美食",
  },
  {
    id: 25,
    type: "group",
    lat: 34.265,
    lng: 117.165,
    count: 4,
    name: "4个群组",
  },
  {
    id: 40,
    type: "post",
    lat: 34.27,
    lng: 117.195,
    count: 1,
    title: "植物园晨曦",
    likes: 124,
    image: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=200",
    tag: "自然",
  },
  {
    id: 41,
    type: "post",
    lat: 34.258,
    lng: 117.172,
    count: 1,
    title: "云龙湖日落",
    likes: 201,
    image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=200",
    tag: "风景",
  },
  {
    id: 42,
    type: "post",
    lat: 34.245,
    lng: 117.188,
    count: 1,
    title: "老街咖啡",
    likes: 56,
    image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=200",
    tag: "生活",
  },
  {
    id: 45,
    type: "post",
    lat: 34.26,
    lng: 117.208,
    count: 7,
    title: "7条动态",
  },
];

export const MOCK_RECENT_LOCATIONS: LocationResult[] = [
  {
    displayName: "云龙湖风景区",
    lat: 34.2444,
    lng: 117.1989,
    desc: "今天 10:24",
    emoji: "🏞️",
  },
  {
    displayName: "彭城广场",
    lat: 34.2568,
    lng: 117.1952,
    desc: "昨天 18:55",
    emoji: "🏙️",
  },
  {
    displayName: "徐州植物园",
    lat: 34.275,
    lng: 117.17,
    desc: "3天前",
    emoji: "🌿",
  },
];

export const MOCK_HOT_LOCATIONS: LocationResult[] = [
  {
    displayName: "云龙湖风景区",
    lat: 34.2444,
    lng: 117.1989,
    emoji: "🏞️",
    desc: "国家5A级旅游景区",
    users: 42,
  },
  {
    displayName: "彭城广场",
    lat: 34.2568,
    lng: 117.1952,
    emoji: "🏙️",
    desc: "徐州市中心地标",
    users: 38,
  },
  {
    displayName: "汉文化景区",
    lat: 34.252,
    lng: 117.2045,
    emoji: "🏯",
    desc: "汉代遗址公园",
    users: 23,
  },
  {
    displayName: "徐州博物馆",
    lat: 34.2682,
    lng: 117.1874,
    emoji: "🏛️",
    desc: "历史文化展馆",
    users: 15,
  },
  {
    displayName: "徐州植物园",
    lat: 34.275,
    lng: 117.17,
    emoji: "🌿",
    desc: "城市绿肺·天然氧吧",
    users: 12,
  },
];

export const MOCK_SEARCH_DB: LocationResult[] = [
  { displayName: "云龙湖风景区", lat: 34.2444, lng: 117.1989 },
  { displayName: "徐州市中心广场", lat: 34.2604, lng: 117.1855 },
  { displayName: "徐州植物园", lat: 34.275, lng: 117.17 },
  { displayName: "彭城广场", lat: 34.2568, lng: 117.1952 },
  { displayName: "汉文化景区", lat: 34.252, lng: 117.2045 },
  { displayName: "徐州博物馆", lat: 34.2682, lng: 117.1874 },
  { displayName: "徐州火车站", lat: 34.2842, lng: 117.2001 },
  { displayName: "泉山森林公园", lat: 34.23, lng: 117.16 },
];
