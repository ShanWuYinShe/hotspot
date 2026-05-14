import React, { useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router";
import {
  ArrowLeft, Search, MapPin, Navigation, X,
  Users, ChevronRight, Clock,
  Star, TrendingUp, SlidersHorizontal, Crosshair,
  Heart, Check,
} from "lucide-react";
import {
  motion, AnimatePresence,
  useMotionValue, useTransform, animate,
} from "motion/react";
import { usePlatform } from "../contexts/PlatformContext";

/* ── SVG Marker imports ─────────────────────────────────────────── */
import markerUserUrl from "../../assets/markers/marker_user.svg";
import markerGroupUrl from "../../assets/markers/marker_group.svg";
import markerMomentUrl from "../../assets/markers/marker_moment.svg";
import markerUserClusterUrl from "../../assets/markers/marker_user_cluster.svg";
import markerGroupClusterUrl from "../../assets/markers/marker_group_cluster.svg";
import markerMomentClusterUrl from "../../assets/markers/marker_moment_cluster.svg";

/* ══════════════════════════════════════════════════════════════════
   Constants & Types
══════════════════════════════════════════════════════════════════ */
/*
  Mobile sheet snap heights (px):
  MIN  = handle(20) + search(52) + confirm(76) = 148  → no list content visible
  MID  = ~320 → list content visible (~partial)
  MAX  = ~540 → full content scrollable
*/
const SNAP_MIN = 148;
const SNAP_MID = 318;
const SNAP_MAX = 540;

type PinType = "user" | "group" | "post";
type SnapState = "min" | "mid" | "max";

interface Pin {
  id: number; type: PinType; lat: number; lng: number;
  count?: number;
  name?: string; avatar?: string; distance?: string; online?: boolean;
  members?: number; tag?: string; title?: string; likes?: number; image?: string;
}
interface LocationResult {
  displayName: string; lat: number; lng: number;
  emoji?: string; desc?: string; users?: number;
}

const PIN_COLORS: Record<PinType, string> = { user: "#2563EB", group: "#059669", post: "#EA580C" };
const PIN_LABELS: Record<PinType, string>  = { user: "用户", group: "群组", post: "动态" };
const SNAP_HEIGHTS: Record<SnapState, number> = { min: SNAP_MIN, mid: SNAP_MID, max: SNAP_MAX };

/* ══════════════════════════════════════════════════════════════════
   Mock Data
══════════════════════════════════════════════════════════════════ */
const XUZHOU_CENTER = { lat: 34.2604, lng: 117.1855 };

const MOCK_PINS: Pin[] = [
  { id: 1,  type: "user",  lat: 34.268, lng: 117.190, count: 1, name: "Liam",    distance: "246m", online: true,  avatar: "https://images.unsplash.com/photo-1619961311005-3cf76b2bcabc?w=80",  tag: "摄影" },
  { id: 2,  type: "user",  lat: 34.255, lng: 117.180, count: 1, name: "Sophia",  distance: "380m", online: true,  avatar: "https://images.unsplash.com/photo-1614339804578-8e82dd7f3174?w=80",  tag: "旅行" },
  { id: 3,  type: "user",  lat: 34.272, lng: 117.175, count: 1, name: "Marco",   distance: "520m", online: false, avatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=80",  tag: "音乐" },
  { id: 10, type: "user",  lat: 34.248, lng: 117.195, count: 5, name: "5人聚集" },
  { id: 11, type: "user",  lat: 34.275, lng: 117.200, count: 3, name: "3人聚集" },
  { id: 20, type: "group", lat: 34.262, lng: 117.185, count: 1, name: "摄影爱好者", members: 28, tag: "摄影" },
  { id: 21, type: "group", lat: 34.253, lng: 117.200, count: 1, name: "美食打卡群", members: 89, tag: "美食" },
  { id: 25, type: "group", lat: 34.265, lng: 117.165, count: 4, name: "4个群组" },
  { id: 40, type: "post",  lat: 34.270, lng: 117.195, count: 1, title: "植物园晨曦", likes: 124, image: "https://images.unsplash.com/photo-1768776179834-93e6cafc6d97?w=200",   tag: "自然" },
  { id: 41, type: "post",  lat: 34.258, lng: 117.172, count: 1, title: "云龙湖日落", likes: 201, image: "https://images.unsplash.com/photo-1593536488177-1eb3c2d4e3d2?w=200",   tag: "风景" },
  { id: 42, type: "post",  lat: 34.245, lng: 117.188, count: 1, title: "老街咖啡",   likes: 56,  image: "https://images.unsplash.com/photo-1748737349771-e2b6605e5961?w=200",   tag: "生活" },
  { id: 45, type: "post",  lat: 34.260, lng: 117.208, count: 7, title: "7条动态" },
];

const RECENT_LOCATIONS: LocationResult[] = [
  { displayName: "云龙湖风景区", lat: 34.2444, lng: 117.1989, desc: "今天 10:24",  emoji: "🏞️" },
  { displayName: "彭城广场",     lat: 34.2568, lng: 117.1952, desc: "昨天 18:55",  emoji: "🏙️" },
  { displayName: "徐州植物园",   lat: 34.2750, lng: 117.1700, desc: "3天前",        emoji: "🌿" },
];

const HOT_LOCATIONS: LocationResult[] = [
  { displayName: "云龙湖风景区", lat: 34.2444, lng: 117.1989, emoji: "🏞️", desc: "国家5A级旅游景区", users: 42 },
  { displayName: "彭城广场",     lat: 34.2568, lng: 117.1952, emoji: "🏙️", desc: "徐州市中心地标",   users: 38 },
  { displayName: "汉文化景区",   lat: 34.2520, lng: 117.2045, emoji: "🏯", desc: "汉代遗址公园",     users: 23 },
  { displayName: "徐州博物馆",   lat: 34.2682, lng: 117.1874, emoji: "🏛️", desc: "历史文化展馆",    users: 15 },
  { displayName: "徐州植物园",   lat: 34.2750, lng: 117.1700, emoji: "🌿", desc: "城市绿肺·天然氧吧", users: 12 },
];

const SEARCH_DB: LocationResult[] = [
  { displayName: "云龙湖风景区",   lat: 34.2444, lng: 117.1989 },
  { displayName: "徐州市中心广场", lat: 34.2604, lng: 117.1855 },
  { displayName: "徐州植物园",     lat: 34.2750, lng: 117.1700 },
  { displayName: "彭城广场",       lat: 34.2568, lng: 117.1952 },
  { displayName: "汉文化景区",     lat: 34.2520, lng: 117.2045 },
  { displayName: "徐州博物馆",     lat: 34.2682, lng: 117.1874 },
  { displayName: "徐州火车站",     lat: 34.2842, lng: 117.2001 },
  { displayName: "泉山森林公园",   lat: 34.2300, lng: 117.1600 },
];

/* ══════════════════════════════════════════════════════════════════
   Helpers
══════════════════════════════════════════════════════════════════ */
function pinToPercent(
  lat: number, lng: number,
  b: { minLat: number; maxLat: number; minLng: number; maxLng: number }
) {
  return {
    x: ((lng - b.minLng) / (b.maxLng - b.minLng)) * 100,
    y: ((b.maxLat - lat) / (b.maxLat - b.minLat)) * 100,
  };
}

function getMarkerUrl(type: PinType, count: number): string {
  if (type === "user")  return count >= 3 ? markerUserClusterUrl  : markerUserUrl;
  if (type === "group") return count >= 3 ? markerGroupClusterUrl : markerGroupUrl;
  return count >= 3 ? markerMomentClusterUrl : markerMomentUrl;
}

function buildMapData(center: { lat: number; lng: number }, zoom: number) {
  const bounds = {
    minLat: center.lat - zoom, maxLat: center.lat + zoom,
    minLng: center.lng - zoom, maxLng: center.lng + zoom,
  };
  return {
    url: `https://www.openstreetmap.org/export/embed.html?bbox=${bounds.minLng},${bounds.minLat},${bounds.maxLng},${bounds.maxLat}&layer=mapnik`,
    bounds,
  };
}

/* ══════════════════════════════════════════════════════════════════
   Shared: MapPinMarker
══════════════════════════════════════════════════════════════════ */
function MapPinMarker({ pin, bounds, isSelected, onSelect }: {
  pin: Pin;
  bounds: { minLat: number; maxLat: number; minLng: number; maxLng: number };
  isSelected: boolean;
  onSelect: (e: React.MouseEvent) => void;
}) {
  const pos = pinToPercent(pin.lat, pin.lng, bounds);
  if (pos.x < 2 || pos.x > 98 || pos.y < 2 || pos.y > 98) return null;
  return (
    <motion.button
      className="absolute outline-none"
      style={{
        left: `${pos.x}%`, top: `${pos.y}%`,
        transform: "translate(-50%, -100%)",
        zIndex: isSelected ? 30 : 10,
      }}
      onClick={onSelect}
      animate={{ scale: isSelected ? 1.32 : 1, y: isSelected ? -5 : 0 }}
      transition={{ type: "spring", stiffness: 500, damping: 28 }}>
      {(pin.count ?? 1) > 1 && (
        <div className="absolute -top-1 -right-1 flex items-center justify-center rounded-full z-10"
          style={{
            width: 16, height: 16,
            background: PIN_COLORS[pin.type],
            border: "1.5px solid white",
            fontSize: 9, color: "white", fontWeight: 700,
          }}>
          {pin.count}
        </div>
      )}
      <img src={getMarkerUrl(pin.type, pin.count ?? 1)} alt={pin.type}
        style={{ width: 36, height: 48, display: "block" }} />
      {isSelected && (
        <motion.div className="absolute rounded-full pointer-events-none"
          style={{ inset: "8px 2px", background: PIN_COLORS[pin.type], opacity: 0.3, filter: "blur(7px)" }}
          animate={{ scale: [1, 1.8], opacity: [0.4, 0] }}
          transition={{ duration: 1.1, repeat: Infinity }} />
      )}
    </motion.button>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Shared: floating icon button style (theme-aware)
══════════════════════════════════════════════════════════════════ */
const floatBtnStyle: React.CSSProperties = {
  background: "var(--neu-bg)",
  border: "1px solid var(--neu-border)",
  boxShadow: "0 2px 12px var(--neu-shadow-dark), 0 1px 3px var(--neu-shadow-dark)",
};

/* ══════════════════════════════════════════════════════════════════
   MOBILE — Draggable bottom sheet
══════════════════════════════════════════════════════════════════ */
function LocationMobile() {
  const navigate = useNavigate();

  /* ── Sheet drag ──────────────────────────────────────────────── */
  const sheetH      = useMotionValue(SNAP_MID);
  const snapRef     = useRef<SnapState>("mid");
  const [snap, setSnap] = useState<SnapState>("mid");
  const drag = useRef({ active: false, startY: 0, startH: SNAP_MID, velY: 0, prevY: 0, prevT: 0 });

  /*
    Right controls bottom offset = sheet height + gap
    This makes the control column always sit just above the sheet,
    independent of the filter legend (which is anchored near the top).
  */
  const rightCtrlBottom = useTransform(sheetH, h => h + 10);

  /* ── Content state ───────────────────────────────────────────── */
  const [showFilter, setShowFilter]       = useState(false);
  const [activeFilters, setActiveFilters] = useState<Set<PinType>>(new Set(["user", "group", "post"]));
  const [searchQuery, setSearchQuery]     = useState("");
  const [selectedPin, setSelectedPin]     = useState<Pin | null>(null);
  const [selectedLoc, setSelectedLoc]     = useState<LocationResult | null>(null);
  const [zoom, setZoom]                   = useState(0.045);
  const [mapCenter, setMapCenter]         = useState(XUZHOU_CENTER);
  const [recentItems, setRecentItems]     = useState(RECENT_LOCATIONS);
  const [activeTab, setActiveTab]         = useState<"recent" | "hot">("recent");

  const { url: embedUrl, bounds } = buildMapData(mapCenter, zoom);
  const filteredPins = MOCK_PINS.filter(p => activeFilters.has(p.type));
  const searchResults: LocationResult[] = searchQuery.trim()
    ? SEARCH_DB.filter(s => s.displayName.includes(searchQuery)).slice(0, 5) : [];

  /* ── Snap helpers ────────────────────────────────────────────── */
  function snapTo(target: SnapState) {
    snapRef.current = target;
    setSnap(target);
    animate(sheetH, SNAP_HEIGHTS[target], { type: "spring", stiffness: 380, damping: 36 });
  }

  function resolveSnap(h: number, vel: number): SnapState {
    const proj = h + vel * 90;                         // 90ms lookahead
    if (proj < (SNAP_MIN + SNAP_MID) / 2) return "min";
    if (proj < (SNAP_MID + SNAP_MAX) / 2) return "mid";
    return "max";
  }

  /* ── Drag pointer events ─────────────────────────────────────── */
  const onHandlePointerDown = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
    const now = Date.now();
    drag.current = { active: true, startY: e.clientY, startH: sheetH.get(), velY: 0, prevY: e.clientY, prevT: now };

    function onMove(ev: PointerEvent) {
      if (!drag.current.active) return;
      const t = Date.now();
      const dt = t - drag.current.prevT || 1;
      drag.current.velY  = (drag.current.prevY - ev.clientY) / dt; // +ve = dragging up
      drag.current.prevY = ev.clientY;
      drag.current.prevT = t;
      const delta = drag.current.startY - ev.clientY;
      const h = Math.max(SNAP_MIN - 10, Math.min(SNAP_MAX + 10, drag.current.startH + delta));
      sheetH.set(h);
    }

    function onUp() {
      drag.current.active = false;
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup",   onUp);
      const target = resolveSnap(sheetH.get(), drag.current.velY);
      snapTo(target);
    }

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup",   onUp);
  }, []);

  /* ── Map tap → collapse to min ───────────────────────────────── */
  function handleMapTap() {
    setSelectedPin(null);
    setShowFilter(false);
    snapTo("min");
  }

  /* ── Location pick ───────────────────────────────────────────── */
  function handleSelectLoc(loc: LocationResult) {
    setSelectedLoc(loc);
    setSearchQuery(loc.displayName);
    setMapCenter({ lat: loc.lat, lng: loc.lng });
    setSelectedPin(null);
    if (snap === "max") snapTo("mid");
  }

  function handleConfirm() {
    const loc = selectedLoc ?? { displayName: "当前位置", lat: XUZHOU_CENTER.lat, lng: XUZHOU_CENTER.lng };
    try { localStorage.setItem("hotspot_last_location", JSON.stringify([loc.lat, loc.lng])); } catch {}
    navigate(-1);
  }

  function toggleFilter(type: PinType) {
    setActiveFilters(prev => {
      const next = new Set(prev);
      next.has(type) ? next.delete(type) : next.add(type);
      return next;
    });
  }

  /* ── Render ──────────────────────────────────────────────────── */
  return (
    <div style={{ position: "relative", width: "100%", height: "100%", overflow: "hidden", background: "var(--neu-bg)" }}>

      {/* ══ MAP (full screen, tappable) ════════════════════════ */}
      <div style={{ position: "absolute", inset: 0, zIndex: 0 }} onClick={handleMapTap}>
        <iframe
          src={embedUrl}
          title="地图"
          style={{ width: "100%", height: "100%", border: 0, pointerEvents: "none" }}
        />

        {/* Current location pulsing dot */}
        <div style={{ position: "absolute", left: "50%", top: "46%", transform: "translate(-50%,-50%)", zIndex: 5, pointerEvents: "none" }}>
          <div style={{
            width: 18, height: 18, borderRadius: "50%",
            background: "#2563EB",
            border: "3px solid var(--neu-bg)",
            boxShadow: "0 2px 10px rgba(37,99,235,0.55)",
            position: "relative",
          }}>
            <motion.div
              style={{
                position: "absolute", inset: -7, borderRadius: "50%",
                background: "rgba(37,99,235,0.2)",
                border: "1px solid rgba(37,99,235,0.3)",
              }}
              animate={{ scale: [1, 1.8], opacity: [0.55, 0] }}
              transition={{ duration: 2, repeat: Infinity, ease: "easeOut" }} />
          </div>
        </div>

        {/* Marker pins */}
        {filteredPins.map(pin => (
          <MapPinMarker
            key={pin.id} pin={pin} bounds={bounds}
            isSelected={selectedPin?.id === pin.id}
            onSelect={e => {
              e.stopPropagation();
              setSelectedPin(prev => prev?.id === pin.id ? null : pin);
              if (snap === "min") snapTo("mid");
            }}
          />
        ))}

        {/* Attribution */}
        <div style={{ position: "absolute", bottom: 4, left: 6, fontSize: 9, color: "var(--neu-text-muted)", opacity: 0.4, pointerEvents: "none", zIndex: 1 }}>
          © OpenStreetMap contributors
        </div>
      </div>

      {/* ══ TOP BAR (fixed, floating over map) ════════════════ */}
      <div style={{ position: "absolute", top: 0, left: 0, right: 0, zIndex: 50, padding: "12px 14px 0" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          {/* Back */}
          <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }} onClick={() => navigate(-1)}
            className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
            style={{ width: 42, height: 42, ...floatBtnStyle }}>
            <ArrowLeft style={{ width: 17, height: 17, color: "var(--neu-text)" }} strokeWidth={2} />
          </motion.button>

          {/* Filter toggle */}
          <motion.button
            whileTap={{ scale: 0.93 }}
            onClick={e => { e.stopPropagation(); setShowFilter(v => !v); }}
            className="flex items-center gap-2 rounded-full outline-none"
            style={{
              padding: "9px 16px", fontSize: 14,
              background: showFilter ? "var(--neu-accent-primary)" : "var(--neu-bg)",
              color: showFilter ? "white" : "var(--neu-text)",
              border: showFilter ? "1px solid transparent" : "1px solid var(--neu-border)",
              boxShadow: showFilter
                ? "0 4px 16px var(--neu-glow)"
                : "0 2px 12px var(--neu-shadow-dark), 0 1px 3px var(--neu-shadow-dark)",
            }}>
            <SlidersHorizontal style={{ width: 14, height: 14 }} strokeWidth={showFilter ? 2.5 : 1.75} />
            筛选
          </motion.button>
        </div>

        {/* ── Filter legend panel — anchored BELOW the filter button ── */}
        <AnimatePresence>
          {showFilter && (
            <motion.div
              initial={{ opacity: 0, y: -10, scale: 0.9 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -10, scale: 0.9 }}
              transition={{ type: "spring", stiffness: 400, damping: 30 }}
              style={{
                position: "absolute",
                top: 62, right: 14,
                width: 136,
                borderRadius: 16,
                overflow: "hidden",
                background: "var(--neu-bg)",
                border: "1px solid var(--neu-border)",
                boxShadow: "0 8px 28px var(--neu-shadow-dark)",
                zIndex: 60,
              }}
              onClick={e => e.stopPropagation()}>
              {([
                ["user",  "用户", markerUserUrl,   "#2563EB"],
                ["group", "群组", markerGroupUrl,  "#059669"],
                ["post",  "动态", markerMomentUrl, "#EA580C"],
              ] as const).map(([type, label, icon, color], i) => {
                const active = activeFilters.has(type as PinType);
                return (
                  <React.Fragment key={type}>
                    {i > 0 && <div style={{ height: 1, background: "var(--neu-border)" }} />}
                    <motion.button
                      whileTap={{ scale: 0.95 }}
                      onClick={() => toggleFilter(type as PinType)}
                      className="w-full flex items-center gap-2 px-3 py-3 outline-none"
                      style={{ opacity: active ? 1 : 0.38 }}>
                      <img src={icon} alt={label} style={{ width: 18, height: 24, flexShrink: 0 }} />
                      <span style={{ flex: 1, fontSize: 13, color: "var(--neu-text)", textAlign: "left" }}>{label}</span>
                      <div style={{
                        width: 13, height: 13, borderRadius: "50%", flexShrink: 0,
                        background: active ? color : "transparent",
                        border: `2px solid ${active ? color : "var(--neu-border)"}`,
                        boxShadow: active ? `0 0 6px ${color}55` : "none",
                        transition: "all 0.18s",
                      }} />
                    </motion.button>
                  </React.Fragment>
                );
              })}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      {/* ══ RIGHT CONTROLS — bottom tracks sheet, independent of filter ══ */}
      <motion.div
        style={{
          position: "absolute",
          right: 12,
          bottom: rightCtrlBottom,     // ← only follows the sheet
          zIndex: 35,
          display: "flex",
          flexDirection: "column",
          gap: 8,
          alignItems: "center",
        }}>
        {/* GPS re-center */}
        <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }}
          onClick={() => { setMapCenter(XUZHOU_CENTER); setSelectedLoc(null); setSearchQuery(""); }}
          className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
          style={{ width: 42, height: 42, ...floatBtnStyle }}>
          <Crosshair style={{ width: 18, height: 18, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
        </motion.button>

        {/* My location dot */}
        <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }}
          className="flex items-center justify-center rounded-full outline-none neu-circle-hover"
          style={{ width: 42, height: 42, ...floatBtnStyle }}>
          <div style={{ width: 14, height: 14, borderRadius: "50%", background: "#2563EB", border: "2.5px solid var(--neu-bg)", boxShadow: "0 0 8px rgba(37,99,235,0.65)" }} />
        </motion.button>

        {/* Zoom is handled via pinch gesture on touch + the map's intrinsic
            controls — duplicate explicit zoom buttons removed per UX cleanup. */}
      </motion.div>

      {/* ══ BOTTOM SHEET (draggable) ════════════════════════════ */}
      <motion.div
        style={{
          position: "absolute", bottom: 0, left: 0, right: 0,
          height: sheetH,
          zIndex: 20,
          borderRadius: "22px 22px 0 0",
          background: "var(--neu-bg)",
          border: "1px solid var(--neu-border)",
          borderBottom: "none",
          boxShadow: "0 -6px 28px var(--neu-shadow-dark)",
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}>

        {/* ── Drag handle ──────────────────────────────────────── */}
        <div
          style={{ flexShrink: 0, display: "flex", flexDirection: "column", alignItems: "center", paddingTop: 10, paddingBottom: 6, cursor: "grab", touchAction: "none", userSelect: "none" }}
          onPointerDown={onHandlePointerDown}>
          <div style={{ width: 38, height: 4, borderRadius: 2, background: "var(--neu-border)" }} />
        </div>

        {/* ── Search bar ───────────────────────────────────────── */}
        <div style={{ flexShrink: 0, padding: "0 14px 8px" }}>
          <div className="flex items-center gap-2.5 px-3.5 rounded-[15px] neu-input-surface"
            style={{ height: 44 }}>
            <Search style={{ width: 15, height: 15, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
            <input
              value={searchQuery}
              onChange={e => {
                setSearchQuery(e.target.value);
                setSelectedLoc(null);
                if (snap === "min") snapTo("mid");
              }}
              onFocus={() => { if (snap === "min") snapTo("mid"); }}
              placeholder="搜索地点、地标或坐标"
              className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 14, color: "var(--neu-text)" }}
            />
            {searchQuery ? (
              <motion.button whileTap={{ scale: 0.85 }}
                onClick={() => { setSearchQuery(""); setSelectedLoc(null); }}
                className="outline-none">
                <X style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={2} />
              </motion.button>
            ) : selectedLoc ? (
              <Check style={{ width: 14, height: 14, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
            ) : null}
          </div>
        </div>

        {/* ── Scrollable content — ONLY shown when snap is not "min" ── */}
        {snap !== "min" && (
          <div style={{ flex: 1, minHeight: 0, overflowY: "auto" }} className="no-scrollbar">
            <AnimatePresence mode="wait">
              {/* Search results */}
              {searchQuery && searchResults.length > 0 && (
                <motion.div key="search" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                  {searchResults.map((r, i) => (
                    <motion.button key={i} whileTap={{ scale: 0.98 }}
                      onClick={() => handleSelectLoc(r)}
                      className="flex items-center gap-3 w-full text-left px-5 py-3 outline-none"
                      style={{ borderBottom: i < searchResults.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                      <div className="flex items-center justify-center rounded-full flex-shrink-0"
                        style={{ width: 34, height: 34, background: "var(--neu-accent-primary)", opacity: 0.12 }}>
                        <MapPin style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                      </div>
                      <span style={{ fontSize: 14, color: "var(--neu-text)", flex: 1 }}>{r.displayName}</span>
                      {selectedLoc?.displayName === r.displayName && (
                        <Check style={{ width: 14, height: 14, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
                      )}
                    </motion.button>
                  ))}
                </motion.div>
              )}

              {/* No search result */}
              {searchQuery && searchResults.length === 0 && (
                <motion.div key="empty" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                  className="flex flex-col items-center py-6">
                  <MapPin style={{ width: 26, height: 26, color: "var(--neu-text-muted)", opacity: 0.28 }} strokeWidth={1} />
                  <p style={{ fontSize: 13, color: "var(--neu-text-muted)", marginTop: 8 }}>未找到匹配地点</p>
                </motion.div>
              )}

              {/* Default: selected pin + tabs + list */}
              {!searchQuery && (
                <motion.div key="main" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                  {/* Selected pin mini-card */}
                  <AnimatePresence>
                    {selectedPin && (
                      <motion.div
                        initial={{ opacity: 0, y: 6, scale: 0.97 }}
                        animate={{ opacity: 1, y: 0, scale: 1 }}
                        exit={{ opacity: 0, y: 4, scale: 0.97 }}
                        className="flex items-center gap-3 mx-4 mb-2 p-3 rounded-[15px]"
                        style={{
                          background: `${PIN_COLORS[selectedPin.type]}10`,
                          border: `1px solid ${PIN_COLORS[selectedPin.type]}30`,
                        }}>
                        {selectedPin.type === "user" && selectedPin.avatar ? (
                          <div className="relative flex-shrink-0">
                            <img src={selectedPin.avatar} className="rounded-full object-cover" style={{ width: 42, height: 42 }} />
                            {selectedPin.online && (
                              <span className="absolute bottom-0 right-0 rounded-full"
                                style={{ width: 10, height: 10, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />
                            )}
                          </div>
                        ) : selectedPin.type === "post" && selectedPin.image ? (
                          <img src={selectedPin.image} className="rounded-[10px] object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
                        ) : (
                          <img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt=""
                            style={{ width: 22, height: 29, flexShrink: 0 }} />
                        )}
                        <div className="flex-1 min-w-0">
                          <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{selectedPin.name || selectedPin.title}</p>
                          <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }}>
                            {PIN_LABELS[selectedPin.type]}
                            {selectedPin.distance ? ` · ${selectedPin.distance}` : ""}
                            {selectedPin.tag ? ` · #${selectedPin.tag}` : ""}
                          </p>
                        </div>
                        <motion.button whileTap={{ scale: 0.9 }}
                          className="px-2.5 py-1.5 rounded-full outline-none flex-shrink-0"
                          style={{ background: PIN_COLORS[selectedPin.type], color: "white", fontSize: 11 }}>
                          查看
                        </motion.button>
                        <motion.button whileTap={{ scale: 0.85 }} onClick={() => setSelectedPin(null)} className="outline-none flex-shrink-0 ml-0.5">
                          <X style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={2} />
                        </motion.button>
                      </motion.div>
                    )}
                  </AnimatePresence>

                  {/* Tab row */}
                  <div className="flex items-center gap-2 px-4 pb-2">
                    {([["recent", "最近使用", Clock], ["hot", "热门地点", Star]] as const).map(([id, label, Icon]) => (
                      <motion.button key={id} whileTap={{ scale: 0.92 }} onClick={() => setActiveTab(id)}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none"
                        style={activeTab === id
                          ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)" }
                          : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)" }
                        }>
                        <Icon style={{ width: 11, height: 11 }} strokeWidth={activeTab === id ? 2.5 : 1.5} />
                        {label}
                      </motion.button>
                    ))}
                    {activeTab === "recent" && recentItems.length > 0 && (
                      <motion.button whileTap={{ scale: 0.9 }} onClick={() => setRecentItems([])}
                        className="ml-auto outline-none" style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>
                        清空
                      </motion.button>
                    )}
                  </div>

                  {/* List items */}
                  <AnimatePresence mode="wait">
                    {activeTab === "recent" && (
                      <motion.div key="rec" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                        {recentItems.length > 0 ? recentItems.map((loc, i) => (
                          <motion.button key={i} whileTap={{ scale: 0.98 }} onClick={() => handleSelectLoc(loc)}
                            className="flex items-center gap-3 w-full text-left px-4 py-2.5 outline-none"
                            style={{ borderBottom: i < recentItems.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                            <div className="flex items-center justify-center rounded-full flex-shrink-0"
                              style={{ width: 36, height: 36, background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)", fontSize: 18 }}>
                              {loc.emoji}
                            </div>
                            <div className="flex-1 min-w-0">
                              <p style={{ fontSize: 14, color: "var(--neu-text)" }} className="truncate">{loc.displayName}</p>
                              <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }}>{loc.desc}</p>
                            </div>
                            {selectedLoc?.displayName === loc.displayName
                              ? <Check style={{ width: 14, height: 14, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
                              : <ChevronRight style={{ width: 13, height: 13, color: "var(--neu-border)", flexShrink: 0 }} strokeWidth={1.5} />
                            }
                          </motion.button>
                        )) : (
                          <p style={{ fontSize: 13, color: "var(--neu-text-muted)", padding: "12px 20px" }}>暂无最近记录</p>
                        )}
                      </motion.div>
                    )}
                    {activeTab === "hot" && (
                      <motion.div key="hot" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                        {HOT_LOCATIONS.map((loc, i) => (
                          <motion.button key={i} whileTap={{ scale: 0.98 }} onClick={() => handleSelectLoc(loc)}
                            className="flex items-center gap-3 w-full text-left px-4 py-2.5 outline-none"
                            style={{ borderBottom: i < HOT_LOCATIONS.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                            <div className="flex items-center justify-center rounded-[13px] flex-shrink-0"
                              style={{ width: 36, height: 36, background: "var(--neu-accent-primary)", fontSize: 18, opacity: 0.88 }}>
                              {loc.emoji}
                            </div>
                            <div className="flex-1 min-w-0">
                              <p style={{ fontSize: 14, color: "var(--neu-text)" }} className="truncate">{loc.displayName}</p>
                              <p style={{ fontSize: 11, color: "var(--neu-text-muted)", marginTop: 1 }}>{loc.desc}</p>
                            </div>
                            <div className="flex items-center gap-1 flex-shrink-0">
                              <Users style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                              <span style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}>{loc.users}</span>
                            </div>
                          </motion.button>
                        ))}
                      </motion.div>
                    )}
                  </AnimatePresence>
                </motion.div>
              )}
            </AnimatePresence>
          </div>
        )}

        {/* ── Confirm button — always at bottom of sheet ───────── */}
        <div style={{ flexShrink: 0, padding: "8px 14px 18px" }}>
          <motion.button whileTap={{ scale: 0.97 }} onClick={handleConfirm}
            className="w-full flex items-center justify-center gap-2 rounded-full outline-none"
            style={{
              height: 50, fontSize: 15, color: "white",
              background: "var(--neu-accent-primary)",
              boxShadow: "0 6px 20px var(--neu-glow)",
            }}>
            <MapPin style={{ width: 15, height: 15 }} strokeWidth={2} />
            {selectedLoc
              ? `确认：${selectedLoc.displayName.slice(0, 11)}`
              : "设为浏览锚点"}
          </motion.button>
        </div>
      </motion.div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   DESKTOP — optimized 2-panel layout
══════════════════════════════════════════════════════════════════ */
function LocationDesktop() {
  const navigate = useNavigate();

  const [searchQuery, setSearchQuery]   = useState("");
  const [selectedPin, setSelectedPin]   = useState<Pin | null>(null);
  const [selectedLoc, setSelectedLoc]   = useState<LocationResult | null>(null);
  const [activeFilters, setActiveFilters] = useState<Set<PinType>>(new Set(["user", "group", "post"]));
  const [activeTab, setActiveTab]       = useState<"nearby" | "hot" | "recent">("nearby");
  const [showFilter, setShowFilter]     = useState(true);
  const [zoom]                          = useState(0.045);
  const [mapCenter, setMapCenter]       = useState(XUZHOU_CENTER);

  function toggleFilter(type: PinType) {
    setActiveFilters(prev => {
      const next = new Set(prev);
      next.has(type) ? next.delete(type) : next.add(type);
      return next;
    });
  }

  const searchResults: LocationResult[] = searchQuery.trim()
    ? SEARCH_DB.filter(s => s.displayName.includes(searchQuery)).slice(0, 6) : [];
  const filteredPins = MOCK_PINS.filter(p => activeFilters.has(p.type));
  const nearbyPins   = MOCK_PINS.filter(p => (p.count ?? 1) === 1).filter(p => activeFilters.has(p.type));

  const { url: embedUrl, bounds } = buildMapData(mapCenter, zoom);

  const userCount  = MOCK_PINS.filter(p => p.type === "user").reduce((s, p) => s + (p.count ?? 1), 0);
  const groupCount = MOCK_PINS.filter(p => p.type === "group").reduce((s, p) => s + (p.count ?? 1), 0);
  const postCount  = MOCK_PINS.filter(p => p.type === "post").reduce((s, p) => s + (p.count ?? 1), 0);

  function handleSelectLoc(loc: LocationResult) {
    setSelectedLoc(loc);
    setSearchQuery(loc.displayName);
    setMapCenter({ lat: loc.lat, lng: loc.lng });
    setSelectedPin(null);
  }

  function handleConfirm() {
    if (selectedLoc) {
      try { localStorage.setItem("hotspot_last_location", JSON.stringify([selectedLoc.lat, selectedLoc.lng])); } catch {}
    }
    navigate(-1);
  }

  return (
    <div className="flex h-full" style={{ background: "var(--neu-bg)" }}>

      {/* ══ LEFT SIDEBAR ══════════════════════════════════════════ */}
      <div className="flex flex-col flex-shrink-0" style={{ width: 340, borderRight: "1px solid var(--neu-border)" }}>

        {/* Header */}
        <div className="flex-shrink-0 px-5 pt-4 pb-3" style={{ borderBottom: "1px solid var(--neu-border)" }}>
          <div className="flex items-center gap-3 mb-3">
            <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }} onClick={() => navigate(-1)}
              className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-circle-hover"
              style={{ width: 36, height: 36, background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)" }}>
              <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
            <div className="flex-1 min-w-0">
              <p style={{ fontSize: 16, color: "var(--neu-text)" }}>位置选择</p>
              <p style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>江苏·徐州</p>
            </div>
            <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }}
              onClick={() => { setMapCenter(XUZHOU_CENTER); setSelectedLoc(null); setSearchQuery(""); }}
              className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-circle-hover"
              style={{ width: 36, height: 36, background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)" }}>
              <Crosshair style={{ width: 15, height: 15, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
            </motion.button>
          </div>

          {/* Type stat chips */}
          <div className="flex gap-2">
            {([
              ["user",  userCount,  "#2563EB", "用户", markerUserUrl],
              ["group", groupCount, "#059669", "群组", markerGroupUrl],
              ["post",  postCount,  "#EA580C", "动态", markerMomentUrl],
            ] as const).map(([type, count, color, label, icon]) => {
              const active = activeFilters.has(type as PinType);
              return (
                <motion.button key={type} whileTap={{ scale: 0.92 }}
                  onClick={() => toggleFilter(type as PinType)}
                  className="flex-1 flex items-center gap-1.5 rounded-[13px] px-2 py-2 outline-none"
                  style={{
                    background: active ? `${color}14` : "var(--neu-bg)",
                    border: `1px solid ${active ? color + "40" : "var(--neu-border)"}`,
                    boxShadow: active
                      ? `0 2px 8px ${color}25`
                      : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
                  }}>
                  <img src={icon} alt={label} style={{ width: 14, height: 19, flexShrink: 0 }} />
                  <div className="flex-1 min-w-0">
                    <p style={{ fontSize: 15, color: active ? color : "var(--neu-text)", lineHeight: 1 }}>{count}</p>
                    <p style={{ fontSize: 10, color: "var(--neu-text-muted)", marginTop: 1 }}>{label}</p>
                  </div>
                </motion.button>
              );
            })}
          </div>
        </div>

        {/* Search */}
        <div className="flex-shrink-0 px-4 pt-3 pb-2 relative">
          <div className="flex items-center gap-2.5 px-3.5 rounded-[14px] neu-input-surface"
            style={{ height: 42 }}>
            <Search style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />
            <input
              value={searchQuery}
              onChange={e => { setSearchQuery(e.target.value); setSelectedLoc(null); }}
              placeholder="搜索地点或地标…"
              className="flex-1 bg-transparent outline-none"
              style={{ fontSize: 13, color: "var(--neu-text)" }} />
            {searchQuery && (
              <motion.button whileTap={{ scale: 0.85 }} onClick={() => setSearchQuery("")} className="outline-none">
                <X style={{ width: 13, height: 13, color: "var(--neu-text-muted)" }} strokeWidth={2} />
              </motion.button>
            )}
          </div>
          <AnimatePresence>
            {searchResults.length > 0 && (
              <motion.div
                initial={{ opacity: 0, y: -6, scale: 0.97 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -6, scale: 0.97 }}
                className="absolute top-full left-4 right-4 z-30 mt-1 rounded-[16px] overflow-hidden"
                style={{ background: "var(--neu-bg)", boxShadow: "0 10px 36px var(--neu-shadow-dark)", border: "1px solid var(--neu-border)" }}>
                {searchResults.map((r, i) => (
                  <motion.button key={i} whileTap={{ scale: 0.98 }} onClick={() => handleSelectLoc(r)}
                    className="flex items-center gap-3 w-full px-4 py-3 outline-none text-left"
                    style={{ borderBottom: i < searchResults.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                    <MapPin style={{ width: 13, height: 13, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={1.5} />
                    <span style={{ fontSize: 13, color: "var(--neu-text)", flex: 1 }}>{r.displayName}</span>
                    {selectedLoc?.displayName === r.displayName && (
                      <Check style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={2.5} />
                    )}
                  </motion.button>
                ))}
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Tabs */}
        <div className="flex-shrink-0 flex gap-1.5 px-4 pb-3">
          {([
            ["nearby",  "附近",  TrendingUp],
            ["hot",     "热门",  Star],
            ["recent",  "最近",  Clock],
          ] as const).map(([id, label, Icon]) => (
            <motion.button key={id} whileTap={{ scale: 0.92 }} onClick={() => setActiveTab(id)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none"
              style={activeTab === id
                ? { background: "var(--neu-accent-primary)", color: "white", fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)" }
                : { background: "var(--neu-bg)", color: "var(--neu-text-muted)", fontSize: 12, border: "1px solid var(--neu-border)" }
              }>
              <Icon style={{ width: 11, height: 11 }} strokeWidth={activeTab === id ? 2.5 : 1.5} />
              {label}
            </motion.button>
          ))}
        </div>

        {/* List */}
        <div className="flex-1 min-h-0 overflow-y-auto no-scrollbar px-3 pb-3">
          <AnimatePresence mode="wait">
            {activeTab === "nearby" && (
              <motion.div key="nearby" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="flex flex-col gap-2">
                {/* Current location card */}
                <div className="flex items-center gap-3 p-3.5 rounded-[16px]"
                  style={{ background: "linear-gradient(135deg, var(--neu-accent-primary), rgba(139,92,246,0.85))", color: "white" }}>
                  <div className="flex items-center justify-center rounded-full flex-shrink-0"
                    style={{ width: 42, height: 42, background: "rgba(255,255,255,0.2)" }}>
                    <Navigation style={{ width: 17, height: 17 }} strokeWidth={2} />
                  </div>
                  <div className="flex-1">
                    <p style={{ fontSize: 13 }}>当前位置</p>
                    <p style={{ fontSize: 11, opacity: 0.8, marginTop: 1 }}>江苏·徐州 · 精度 ±12m</p>
                  </div>
                  <div style={{ width: 8, height: 8, borderRadius: "50%", background: "#4ade80", boxShadow: "0 0 6px #4ade80", flexShrink: 0 }} />
                </div>

                {nearbyPins.map(pin => (
                  <motion.button key={pin.id} whileTap={{ scale: 0.97 }}
                    onClick={() => { setSelectedPin(pin); setMapCenter({ lat: pin.lat, lng: pin.lng }); }}
                    className="flex items-center gap-3 w-full text-left p-3 rounded-[15px] outline-none"
                    style={{
                      background: "var(--neu-bg)",
                      boxShadow: selectedPin?.id === pin.id
                        ? `0 4px 16px ${PIN_COLORS[pin.type]}30, 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)`
                        : "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                      border: `1px solid ${selectedPin?.id === pin.id ? PIN_COLORS[pin.type] + "40" : "var(--neu-border)"}`,
                    }}>
                    {pin.type === "user" && pin.avatar ? (
                      <div className="relative flex-shrink-0">
                        <img src={pin.avatar} className="rounded-full object-cover" style={{ width: 42, height: 42 }} />
                        {pin.online && <span className="absolute bottom-0 right-0 rounded-full" style={{ width: 10, height: 10, background: "#22c55e", border: "2px solid var(--neu-bg)" }} />}
                      </div>
                    ) : pin.type === "post" && pin.image ? (
                      <img src={pin.image} className="rounded-[11px] object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
                    ) : (
                      <div className="flex items-center justify-center rounded-[12px] flex-shrink-0"
                        style={{ width: 42, height: 42, background: `${PIN_COLORS[pin.type]}12` }}>
                        <img src={getMarkerUrl(pin.type, 1)} alt={pin.type} style={{ width: 20, height: 27 }} />
                      </div>
                    )}
                    <div className="flex-1 min-w-0">
                      <p style={{ fontSize: 13, color: "var(--neu-text)" }}>{pin.name || pin.title}</p>
                      <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                        <span style={{ fontSize: 10, padding: "1px 6px", borderRadius: 4, background: `${PIN_COLORS[pin.type]}15`, color: PIN_COLORS[pin.type] }}>
                          {PIN_LABELS[pin.type]}
                        </span>
                        {pin.tag && <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>#{pin.tag}</span>}
                        {pin.distance && (
                          <div className="flex items-center gap-0.5">
                            <MapPin style={{ width: 9, height: 9, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{pin.distance}</span>
                          </div>
                        )}
                        {pin.members && <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{pin.members}人</span>}
                        {pin.likes && (
                          <div className="flex items-center gap-0.5">
                            <Heart style={{ width: 9, height: 9, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                            <span style={{ fontSize: 11, color: "var(--neu-text-muted)" }}>{pin.likes}</span>
                          </div>
                        )}
                      </div>
                    </div>
                    {pin.type === "user" && pin.online && (
                      <div style={{ width: 7, height: 7, borderRadius: "50%", background: "#22c55e", flexShrink: 0 }} />
                    )}
                  </motion.button>
                ))}
              </motion.div>
            )}

            {activeTab === "hot" && (
              <motion.div key="hot" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="flex flex-col gap-2">
                {HOT_LOCATIONS.map((loc, i) => (
                  <motion.button key={i} whileTap={{ scale: 0.97 }} onClick={() => handleSelectLoc(loc)}
                    className="flex items-center gap-3 w-full text-left p-3.5 rounded-[16px] outline-none"
                    style={{
                      background: "var(--neu-bg)",
                      boxShadow: selectedLoc?.displayName === loc.displayName
                        ? "0 0 0 1px var(--neu-accent-soft-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)"
                        : "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                      border: `1px solid ${selectedLoc?.displayName === loc.displayName ? "var(--neu-accent-soft-border)" : "var(--neu-border)"}`,
                    }}>
                    <div className="flex items-center justify-center rounded-[14px] flex-shrink-0"
                      style={{ width: 46, height: 46, fontSize: 22, background: "var(--neu-accent-primary)", opacity: 0.88 }}>
                      {loc.emoji}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{loc.displayName}</p>
                      <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>{loc.desc}</p>
                    </div>
                    <div className="flex flex-col items-end flex-shrink-0 gap-0.5">
                      <div className="flex items-center gap-1">
                        <Users style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                        <span style={{ fontSize: 12, color: "var(--neu-accent-primary)" }}>{loc.users}</span>
                      </div>
                      <span style={{ fontSize: 10, color: "var(--neu-text-muted)" }}>在线</span>
                    </div>
                  </motion.button>
                ))}
              </motion.div>
            )}

            {activeTab === "recent" && (
              <motion.div key="recent" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="flex flex-col gap-2">
                {RECENT_LOCATIONS.map((loc, i) => (
                  <motion.button key={i} whileTap={{ scale: 0.97 }} onClick={() => handleSelectLoc(loc)}
                    className="flex items-center gap-3 w-full text-left p-3.5 rounded-[16px] outline-none"
                    style={{ background: "var(--neu-bg)", boxShadow: "4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)", border: "1px solid var(--neu-border)" }}>
                    <div className="flex items-center justify-center rounded-full flex-shrink-0"
                      style={{ width: 44, height: 44, background: "var(--neu-bg)", boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", fontSize: 20 }}>
                      {loc.emoji}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p style={{ fontSize: 14, color: "var(--neu-text)" }}>{loc.displayName}</p>
                      <p style={{ fontSize: 12, color: "var(--neu-text-muted)", marginTop: 2 }}>{loc.desc}</p>
                    </div>
                    {selectedLoc?.displayName === loc.displayName && (
                      <Check style={{ width: 14, height: 14, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
                    )}
                  </motion.button>
                ))}
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Confirm */}
        <div style={{ flexShrink: 0, padding: "10px 12px 16px", borderTop: "1px solid var(--neu-border)" }}>
          <AnimatePresence>
            {selectedLoc && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: "auto" }}
                exit={{ opacity: 0, height: 0 }}
                className="overflow-hidden mb-2 rounded-[13px] px-3 py-2.5 flex items-center gap-2"
                style={{ background: "var(--neu-accent-primary)", opacity: 0.1, border: "1px solid var(--neu-border)" }}>
              </motion.div>
            )}
          </AnimatePresence>
          <AnimatePresence>
            {selectedLoc && (
              <motion.div
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 4 }}
                className="flex items-center gap-2 mb-2 px-3 py-2.5 rounded-[13px]"
                style={{ background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)" }}>
                <Check style={{ width: 13, height: 13, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
                <p style={{ fontSize: 12, color: "var(--neu-accent-primary)" }} className="flex-1 truncate">{selectedLoc.displayName}</p>
                <motion.button whileTap={{ scale: 0.85 }} onClick={() => { setSelectedLoc(null); setSearchQuery(""); }} className="outline-none">
                  <X style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={2} />
                </motion.button>
              </motion.div>
            )}
          </AnimatePresence>
          <motion.button whileTap={{ scale: 0.97 }} onClick={handleConfirm}
            className="w-full flex items-center justify-center gap-2 rounded-full outline-none"
            style={{ height: 48, background: "var(--neu-accent-primary)", color: "white", fontSize: 14, boxShadow: "0 6px 20px var(--neu-glow)" }}>
            <MapPin style={{ width: 14, height: 14 }} strokeWidth={2} />
            {selectedLoc ? "确认此位置" : "设为浏览锚点"}
          </motion.button>
        </div>
      </div>

      {/* ══ MAP PANEL ═══════════════════════════════════════════════ */}
      <div className="flex-1 relative overflow-hidden" onClick={() => setSelectedPin(null)}>
        <iframe src={embedUrl} className="w-full h-full border-0" title="地图" style={{ pointerEvents: "none" }} />

        {/* Current location dot */}
        <div style={{ position: "absolute", left: "50%", top: "50%", transform: "translate(-50%,-50%)", pointerEvents: "none", zIndex: 5 }}>
          <div style={{ width: 18, height: 18, borderRadius: "50%", background: "#2563EB", border: "3px solid var(--neu-bg)", boxShadow: "0 2px 10px rgba(37,99,235,0.55)", position: "relative" }}>
            <motion.div
              style={{ position: "absolute", inset: -8, borderRadius: "50%", background: "rgba(37,99,235,0.18)", border: "1px solid rgba(37,99,235,0.25)" }}
              animate={{ scale: [1, 1.8], opacity: [0.55, 0] }}
              transition={{ duration: 2, repeat: Infinity, ease: "easeOut" }} />
          </div>
        </div>

        {/* Pin markers */}
        {filteredPins.map(pin => (
          <MapPinMarker key={pin.id} pin={pin} bounds={bounds} isSelected={selectedPin?.id === pin.id}
            onSelect={e => { e.stopPropagation(); setSelectedPin(prev => prev?.id === pin.id ? null : pin); setMapCenter({ lat: pin.lat, lng: pin.lng }); }} />
        ))}

        {/* Top-left: 筛选 (multi-select, consistent with mobile/tablet) */}
        <div style={{ position: "absolute", top: 14, left: 14, zIndex: 20 }} onClick={e => e.stopPropagation()}>
          <motion.button whileTap={{ scale: 0.93 }} onClick={() => setShowFilter(v => !v)}
            className="flex items-center gap-2 rounded-full outline-none"
            style={{
              padding: "9px 16px", fontSize: 13,
              background: showFilter ? "var(--neu-accent-primary)" : "var(--neu-bg)",
              color:      showFilter ? "white"                    : "var(--neu-text)",
              border:     showFilter ? "1px solid transparent"    : "1px solid var(--neu-border)",
              boxShadow:  showFilter
                ? "0 4px 16px var(--neu-glow)"
                : "0 4px 16px var(--neu-shadow-dark)",
            }}>
            <SlidersHorizontal style={{ width: 13, height: 13 }} strokeWidth={showFilter ? 2.5 : 1.75} />
            筛选
            {activeFilters.size < 3 && (
              <span style={{
                marginLeft: 2,
                fontSize: 10,
                padding: "1px 6px",
                borderRadius: 999,
                background: showFilter ? "rgba(255,255,255,0.25)" : "var(--neu-accent-soft)",
                color:      showFilter ? "white"                  : "var(--neu-accent-primary)",
                fontWeight: 500,
              }}>{activeFilters.size}</span>
            )}
          </motion.button>
          <AnimatePresence>
            {showFilter && (
              <motion.div
                initial={{ opacity: 0, y: -8, scale: 0.9 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: -8, scale: 0.9 }}
                className="mt-2 rounded-[16px] overflow-hidden"
                style={{ background: "var(--neu-bg)", boxShadow: "0 8px 28px var(--neu-shadow-dark)", border: "1px solid var(--neu-border)", width: 200 }}>
                {([
                  ["user",  "用户",  markerUserUrl,   markerUserClusterUrl,   "#2563EB", userCount],
                  ["group", "群组",  markerGroupUrl,  markerGroupClusterUrl,  "#059669", groupCount],
                  ["post",  "动态",  markerMomentUrl, markerMomentClusterUrl, "#EA580C", postCount],
                ] as const).map(([type, label, icon, clusterIcon, color, count], i) => {
                  const active = activeFilters.has(type as PinType);
                  return (
                    <React.Fragment key={type}>
                      {i > 0 && <div style={{ height: 1, background: "var(--neu-border)" }} />}
                      <motion.button whileTap={{ scale: 0.97 }}
                        onClick={() => toggleFilter(type as PinType)}
                        className="w-full flex items-center gap-2 px-3 py-2.5 outline-none"
                        style={{ opacity: active ? 1 : 0.45 }}>
                        <img src={icon} alt="" style={{ width: 15, height: 21 }} />
                        <img src={clusterIcon} alt="" style={{ width: 15, height: 21, opacity: 0.65 }} />
                        <span style={{ flex: 1, fontSize: 12, color: "var(--neu-text)", textAlign: "left" }}>{label}</span>
                        <span style={{ fontSize: 12, color, fontWeight: 600 }}>{count}</span>
                        <div style={{
                          width: 14, height: 14, borderRadius: 4, flexShrink: 0,
                          background: active ? color : "transparent",
                          border: `1.5px solid ${active ? color : "var(--neu-border-pressed)"}`,
                          display: "flex", alignItems: "center", justifyContent: "center",
                          transition: "all 0.18s",
                        }}>
                          {active && <Check style={{ width: 10, height: 10, color: "white" }} strokeWidth={3} />}
                        </div>
                      </motion.button>
                    </React.Fragment>
                  );
                })}
                <div style={{ height: 1, background: "var(--neu-border)" }} />
                <div className="px-3 py-2 flex items-center gap-2">
                  <div style={{ width: 13, height: 13, borderRadius: "50%", background: "#2563EB", border: "2px solid var(--neu-bg)", boxShadow: "0 0 5px rgba(37,99,235,0.5)" }} />
                  <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>我的位置</span>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Top-right zoom + compass removed — zoom uses the map's own
            mechanism, and recenter already exists in the left sidebar. */}

        {/* Selected pin popup */}
        <AnimatePresence>
          {selectedPin && (
            <motion.div
              initial={{ opacity: 0, y: 14, scale: 0.94 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: 14, scale: 0.94 }}
              transition={{ type: "spring", stiffness: 420, damping: 32 }}
              style={{
                position: "absolute", bottom: 20, left: 20, right: 20, zIndex: 25,
                background: "var(--neu-bg)", borderRadius: 20, padding: 16,
                boxShadow: "0 16px 48px var(--neu-shadow-dark)", border: "1px solid var(--neu-border)",
              }}
              onClick={e => e.stopPropagation()}>
              <div className="flex items-center gap-3">
                {selectedPin.type === "user" && selectedPin.avatar ? (
                  <div className="relative flex-shrink-0">
                    <img src={selectedPin.avatar} className="rounded-full object-cover" style={{ width: 52, height: 52 }} />
                    {selectedPin.online && <span className="absolute bottom-0 right-0 rounded-full" style={{ width: 12, height: 12, background: "#22c55e", border: "2.5px solid var(--neu-bg)" }} />}
                  </div>
                ) : selectedPin.type === "post" && selectedPin.image ? (
                  <img src={selectedPin.image} className="rounded-[14px] object-cover flex-shrink-0" style={{ width: 52, height: 52 }} />
                ) : (
                  <div className="flex items-center justify-center rounded-[15px] flex-shrink-0"
                    style={{ width: 52, height: 52, background: `${PIN_COLORS[selectedPin.type]}14` }}>
                    <img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt="" style={{ width: 26, height: 35 }} />
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p style={{ fontSize: 15, color: "var(--neu-text)" }}>{selectedPin.name || selectedPin.title}</p>
                    {(selectedPin.count ?? 1) > 1 && (
                      <span style={{ fontSize: 10, padding: "1px 6px", borderRadius: 4, background: PIN_COLORS[selectedPin.type], color: "white" }}>
                        {selectedPin.count}个
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    <span style={{ fontSize: 11, padding: "1px 7px", borderRadius: 4, background: `${PIN_COLORS[selectedPin.type]}15`, color: PIN_COLORS[selectedPin.type] }}>
                      {PIN_LABELS[selectedPin.type]}
                    </span>
                    {selectedPin.tag && <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>#{selectedPin.tag}</span>}
                    {selectedPin.distance && (
                      <div className="flex items-center gap-1">
                        <MapPin style={{ width: 10, height: 10, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                        <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{selectedPin.distance}</span>
                      </div>
                    )}
                    {selectedPin.members && <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{selectedPin.members}名成员</span>}
                    {selectedPin.likes && (
                      <div className="flex items-center gap-1">
                        <Heart style={{ width: 10, height: 10, color: "#f43f5e" }} strokeWidth={1.5} />
                        <span style={{ fontSize: 12, color: "var(--neu-text-muted)" }}>{selectedPin.likes}</span>
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex gap-2 flex-shrink-0">
                  <motion.button whileTap={{ scale: 0.93 }}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none"
                    style={{ background: "var(--neu-accent-primary)", color: "white", fontSize: 13, boxShadow: "0 4px 14px var(--neu-glow)" }}>
                    查看详情
                    <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={2} />
                  </motion.button>
                  <motion.button whileTap={{ scale: 0.88 }} onClick={() => setSelectedPin(null)}
                    className="outline-none flex items-center justify-center rounded-full"
                    style={{ width: 36, height: 36, background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)" }}>
                    <X style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={2} />
                  </motion.button>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Attribution */}
        <div style={{ position: "absolute", bottom: 4, left: 8, fontSize: 9, color: "var(--neu-text-muted)", opacity: 0.38, pointerEvents: "none", zIndex: 1 }}>
          © OpenStreetMap contributors
        </div>
      </div>
    </div>
  );
}

/* ══════════════════════════════════════════════════════════════════
   Export
══════════════════════════════════════════════════════════════════ */
export function Location() {
  const { platform } = usePlatform();
  if (platform !== "desktop") return <LocationMobile />;
  return <LocationDesktop />;
}
