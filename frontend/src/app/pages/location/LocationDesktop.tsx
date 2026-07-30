import React, { useState, useRef, useCallback } from "react";
import { useNavigate } from "react-router";
import { ArrowLeft, Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, SlidersHorizontal, Crosshair, Heart, Check } from "lucide-react";
import { motion, AnimatePresence, useMotionValue, useTransform, animate } from "motion/react";
import type { PinType, Pin, LocationResult } from "./location.data";
import { XUZHOU_CENTER, MOCK_PINS, RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB } from "./location.data";
import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, type SnapState, PIN_COLORS, PIN_LABELS, pinTone, floatBtnStyle, buildMapData, MapPinMarker, getMarkerUrl, markerUserUrl, markerGroupUrl, markerMomentUrl, markerUserClusterUrl, markerGroupClusterUrl, markerMomentClusterUrl } from "./location.shared";
import { useDismissibleLayer } from "../../hooks/useDismissibleLayer";

function LocationDesktop() {
  const navigate = useNavigate();

  const [searchQuery, setSearchQuery]   = useState("");
  const [selectedPin, setSelectedPin]   = useState<Pin | null>(null);
  const [selectedLoc, setSelectedLoc]   = useState<LocationResult | null>(null);
  const [activeFilters, setActiveFilters] = useState<Set<PinType>>(new Set(["user", "group", "post"]));
  const [activeTab, setActiveTab]       = useState<"nearby" | "hot" | "recent">("nearby");
  const [hoveredListId, setHoveredListId] = useState<string | null>(null);
  const [showFilter, setShowFilter]     = useState(true);
  const [zoom]                          = useState(0.045);

  useDismissibleLayer({ open: showFilter, onDismiss: () => setShowFilter(false) });
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
    <div className="flex h-full bg-[var(--neu-bg)]">

      {/* ══ LEFT SIDEBAR ══════════════════════════════════════════ */}
      <div className="flex flex-col flex-shrink-0" style={{ width: 340, borderRight: "1px solid var(--neu-border)" }}>

        {/* Header */}
        <div className="flex-shrink-0 px-5 pt-4 pb-3" style={{ borderBottom: "1px solid var(--neu-border)" }}>
          <div className="flex items-center gap-3 mb-3">
            <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }} onClick={() => navigate(-1)}
              className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-circle-hover bg-[var(--neu-bg)]"
              style={{ width: 36, height: 36, border: "1px solid var(--neu-border)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)" }}>
              <ArrowLeft style={{ width: 15, height: 15, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
            </motion.button>
            <div className="flex-1 min-w-0">
              <p className="text-[var(--neu-text)]" style={{ fontSize: 16 }}>位置选择</p>
              <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>江苏·徐州</p>
            </div>
            <motion.button whileTap={{ scale: 0.88, transition: { type: "spring", stiffness: 600, damping: 28 } }} whileHover={{ scale: 1.14, transition: { type: "spring", stiffness: 400, damping: 16 } }}
              onClick={() => { setMapCenter(XUZHOU_CENTER); setSelectedLoc(null); setSearchQuery(""); }}
              className="flex items-center justify-center rounded-full outline-none flex-shrink-0 neu-circle-hover bg-[var(--neu-bg)]"
              style={{ width: 36, height: 36, border: "1px solid var(--neu-border)", boxShadow: "4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light)" }}>
              <Crosshair style={{ width: 15, height: 15, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
            </motion.button>
          </div>

          {/* Type stat chips */}
          <div className="flex gap-2">
            {([
              ["user",  userCount,  PIN_COLORS.user, "用户", markerUserUrl],
              ["group", groupCount, PIN_COLORS.group, "群组", markerGroupUrl],
              ["post",  postCount,  PIN_COLORS.post, "动态", markerMomentUrl],
            ] as const).map(([type, count, color, label, icon]) => {
              const active = activeFilters.has(type as PinType);
              return (
                <motion.button key={type} whileTap={{ scale: 0.92 }}
                  onClick={() => toggleFilter(type as PinType)}
                  className="flex-1 flex items-center gap-1.5 rounded-[13px] px-2 py-2 outline-none"
                  style={{
                    background: active ? pinTone(type as PinType, 14) : "var(--neu-bg)",
                    border: `1px solid ${active ? pinTone(type as PinType, 40) : "var(--neu-border)"}`,
                    boxShadow: active
                      ? `0 2px 8px ${pinTone(type as PinType, 25)}`
                      : "4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)",
                  }}>
                  <img src={icon} alt={label} style={{ width: 14, height: 19, flexShrink: 0 }} />
                  <div className="flex-1 min-w-0">
                    <p style={{ fontSize: 15, color: active ? color : "var(--neu-text)", lineHeight: 1 }}>{count}</p>
                    <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 10, marginTop: 1 }}>{label}</p>
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
              className="flex-1 bg-transparent outline-none text-[var(--neu-text)]"
              style={{ fontSize: 13 }} />
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
                className="absolute top-full left-4 right-4 z-30 mt-1 rounded-[16px] bg-[var(--neu-bg)]"
                style={{ boxShadow: "0 0 0 1px var(--neu-border), 0 10px 36px var(--neu-shadow-dark)" }}>
                <div style={{ overflow: "hidden", borderRadius: 15 }}>
                {searchResults.map((r, i) => (
                  <motion.button key={i} whileTap={{ scale: 0.98 }} whileHover={selectedLoc?.displayName === r.displayName ? undefined : { x: 2 }}
                    onHoverStart={() => setHoveredListId(`search-${i}`)}
                    onHoverEnd={() => setHoveredListId(null)}
                    onClick={() => handleSelectLoc(r)}
                    data-active={selectedLoc?.displayName === r.displayName || undefined}
                    aria-pressed={selectedLoc?.displayName === r.displayName}
                    className={`neu-row-hover flex w-full items-center gap-3 px-4 py-3 text-left outline-none ${hoveredListId === `search-${i}` ? "bg-[var(--neu-accent-soft)]" : "bg-[var(--neu-bg)]"}`}
                    style={{ borderBottom: i < searchResults.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                    <MapPin style={{ width: 13, height: 13, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={1.5} />
                    <span className="text-[var(--neu-text)]" style={{ fontSize: 13, flex: 1 }}>{r.displayName}</span>
                    {selectedLoc?.displayName === r.displayName && (
                      <Check style={{ width: 13, height: 13, color: "var(--neu-accent-primary)" }} strokeWidth={2.5} />
                    )}
                  </motion.button>
                ))}
                </div>
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
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none ${activeTab === id ? "bg-[var(--neu-accent-primary)] text-white" : "bg-[var(--neu-bg)] text-[var(--neu-text-muted)]"}`}
              style={activeTab === id
                ? { fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)" }
                : { fontSize: 12, border: "1px solid var(--neu-border)" }
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
                <div className="flex items-center gap-3 p-3.5 rounded-[16px] bg-[var(--neu-accent-primary)] text-white">
                  <div className="flex items-center justify-center rounded-full flex-shrink-0"
                    style={{ width: 42, height: 42, background: "rgba(255,255,255,0.2)" }}>
                    <Navigation style={{ width: 17, height: 17 }} strokeWidth={2} />
                  </div>
                  <div className="flex-1">
                    <p style={{ fontSize: 13 }}>当前位置</p>
                    <p style={{ fontSize: 11, opacity: 0.8, marginTop: 1 }}>江苏·徐州 · 精度 ±12m</p>
                  </div>
                  <div className="bg-[var(--semantic-online)]" style={{ width: 8, height: 8, borderRadius: "50%", boxShadow: "0 0 6px var(--semantic-online)", flexShrink: 0 }} />
                </div>

                {nearbyPins.map(pin => (
                  <motion.button key={pin.id} whileTap={{ scale: 0.97 }} whileHover={selectedPin?.id === pin.id ? undefined : { y: -2 }}
                    onHoverStart={() => setHoveredListId(`pin-${pin.id}`)}
                    onHoverEnd={() => setHoveredListId(null)}
                    onClick={() => { setSelectedPin(pin); setMapCenter({ lat: pin.lat, lng: pin.lng }); }}
                    data-active={selectedPin?.id === pin.id || undefined}
                    aria-pressed={selectedPin?.id === pin.id}
                    className="app-list-row app-list-row--rich neu-row-hover rounded-[15px] outline-none"
                    style={{
                      background: selectedPin?.id === pin.id ? pinTone(pin.type, 10) : hoveredListId === `pin-${pin.id}` ? pinTone(pin.type, 6) : "var(--neu-bg)",
                      boxShadow: selectedPin?.id === pin.id
                        ? `0 0 0 1px ${pinTone(pin.type, 52)}, 0 6px 18px ${pinTone(pin.type, 32)}, 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)`
                        : hoveredListId === `pin-${pin.id}`
                          ? `0 0 0 1px ${pinTone(pin.type, 25)}, 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)`
                          : "0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                    }}>
                    {pin.type === "user" && pin.avatar ? (
                      <div className="relative flex-shrink-0">
                        <img src={pin.avatar} className="rounded-full object-cover" style={{ width: 42, height: 42 }} />
                        {pin.online && <span className="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style={{ width: 10, height: 10, border: "2px solid var(--neu-bg)" }} />}
                      </div>
                    ) : pin.type === "post" && pin.image ? (
                      <img src={pin.image} className="rounded-[11px] object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
                    ) : (
                      <div className="flex items-center justify-center rounded-[12px] flex-shrink-0"
                        style={{ width: 42, height: 42, background: pinTone(pin.type, 12) }}>
                        <img src={getMarkerUrl(pin.type, 1)} alt={pin.type} style={{ width: 20, height: 27 }} />
                      </div>
                    )}
                    <div className="flex-1 min-w-0">
                      <p className="text-[var(--neu-text)]" style={{ fontSize: 13 }}>{pin.name || pin.title}</p>
                      <div className="flex items-center gap-2 mt-0.5 flex-wrap">
                        <span style={{ fontSize: 10, padding: "1px 6px", borderRadius: 4, background: pinTone(pin.type, 15), color: PIN_COLORS[pin.type] }}>
                          {PIN_LABELS[pin.type]}
                        </span>
                        {pin.tag && <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>#{pin.tag}</span>}
                        {pin.distance && (
                          <div className="flex items-center gap-0.5">
                            <MapPin style={{ width: 9, height: 9, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                            <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>{pin.distance}</span>
                          </div>
                        )}
                        {pin.members && <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>{pin.members}人</span>}
                        {pin.likes && (
                          <div className="flex items-center gap-0.5">
                            <Heart style={{ width: 9, height: 9, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                            <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 11 }}>{pin.likes}</span>
                          </div>
                        )}
                      </div>
                    </div>
                    {selectedPin?.id === pin.id ? (
                      <span className="rounded-full px-2 py-1" style={{ fontSize: 10, color: "white", background: PIN_COLORS[pin.type], flexShrink: 0 }}>已定位</span>
                    ) : pin.type === "user" && pin.online ? (
                      <div className="bg-[var(--semantic-online)]" style={{ width: 7, height: 7, borderRadius: "50%", flexShrink: 0 }} />
                    ) : <ChevronRight style={{ width: 14, height: 14, color: "var(--neu-text-muted)", flexShrink: 0 }} strokeWidth={1.5} />}
                  </motion.button>
                ))}
              </motion.div>
            )}

            {activeTab === "hot" && (
              <motion.div key="hot" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="flex flex-col gap-2">
                {HOT_LOCATIONS.map((loc, i) => (
                  <motion.button key={i} whileTap={{ scale: 0.97 }} whileHover={selectedLoc?.displayName === loc.displayName ? undefined : { y: -2 }}
                    onHoverStart={() => setHoveredListId(`hot-${i}`)}
                    onHoverEnd={() => setHoveredListId(null)}
                    onClick={() => handleSelectLoc(loc)}
                    data-active={selectedLoc?.displayName === loc.displayName || undefined}
                    aria-pressed={selectedLoc?.displayName === loc.displayName}
                    className="app-list-row app-list-row--rich neu-row-hover rounded-[16px] outline-none"
                    style={{
                      background: selectedLoc?.displayName === loc.displayName ? "var(--neu-accent-soft)" : hoveredListId === `hot-${i}` ? "color-mix(in srgb, var(--neu-accent-soft) 62%, var(--neu-bg))" : "var(--neu-bg)",
                      boxShadow: selectedLoc?.displayName === loc.displayName
                        ? "0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)"
                        : hoveredListId === `hot-${i}`
                          ? "0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)"
                          : "0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                    }}>
                    <div className="flex items-center justify-center rounded-[14px] flex-shrink-0 bg-[var(--neu-accent-primary)]"
                      style={{ width: 46, height: 46, fontSize: 22, opacity: 0.88 }}>
                      {loc.emoji}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-[var(--neu-text)]" style={{ fontSize: 14 }}>{loc.displayName}</p>
                      <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, marginTop: 2 }}>{loc.desc}</p>
                    </div>
                    <div className="flex flex-col items-end flex-shrink-0 gap-0.5">
                      <div className="flex items-center gap-1">
                        <Users style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                        <span className="text-[var(--neu-accent-primary)]" style={{ fontSize: 12 }}>{loc.users}</span>
                      </div>
                      <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 10 }}>在线</span>
                    </div>
                  </motion.button>
                ))}
              </motion.div>
            )}

            {activeTab === "recent" && (
              <motion.div key="recent" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="flex flex-col gap-2">
                {RECENT_LOCATIONS.map((loc, i) => (
                  <motion.button key={i} whileTap={{ scale: 0.97 }} whileHover={selectedLoc?.displayName === loc.displayName ? undefined : { y: -2 }}
                    onHoverStart={() => setHoveredListId(`recent-${i}`)}
                    onHoverEnd={() => setHoveredListId(null)}
                    onClick={() => handleSelectLoc(loc)}
                    data-active={selectedLoc?.displayName === loc.displayName || undefined}
                    aria-pressed={selectedLoc?.displayName === loc.displayName}
                    className="app-list-row app-list-row--rich neu-row-hover rounded-[16px] outline-none"
                    style={{
                      background: selectedLoc?.displayName === loc.displayName ? "var(--neu-accent-soft)" : hoveredListId === `recent-${i}` ? "color-mix(in srgb, var(--neu-accent-soft) 62%, var(--neu-bg))" : "var(--neu-bg)",
                      boxShadow: selectedLoc?.displayName === loc.displayName || hoveredListId === `recent-${i}`
                        ? "0 0 0 1px var(--neu-accent-soft-border), 6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)"
                        : "0 0 0 1px var(--neu-border), 4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light)",
                    }}>
                    <div className="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-bg)]"
                      style={{ width: 44, height: 44, boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)", border: "1px solid var(--neu-border)", fontSize: 20 }}>
                      {loc.emoji}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-[var(--neu-text)]" style={{ fontSize: 14 }}>{loc.displayName}</p>
                      <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 12, marginTop: 2 }}>{loc.desc}</p>
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
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 4 }}
                className="flex items-center gap-2 mb-2 px-3 py-2.5 rounded-[13px] bg-[var(--neu-bg)]"
                style={{ boxShadow: "0 0 0 1px var(--neu-border), inset 2px 2px 5px var(--neu-shadow-dark), inset -2px -2px 5px var(--neu-shadow-light)" }}>
                <Check style={{ width: 13, height: 13, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
                <p className="flex-1 truncate text-[var(--neu-accent-primary)]" style={{ fontSize: 12 }}>{selectedLoc.displayName}</p>
                <motion.button whileTap={{ scale: 0.85 }} onClick={() => { setSelectedLoc(null); setSearchQuery(""); }} className="outline-none">
                  <X style={{ width: 12, height: 12, color: "var(--neu-text-muted)" }} strokeWidth={2} />
                </motion.button>
              </motion.div>
            )}
          </AnimatePresence>
          <motion.button whileTap={{ scale: 0.97 }} onClick={handleConfirm}
            className="w-full flex items-center justify-center gap-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
            style={{ height: 48, fontSize: 14, boxShadow: "0 6px 20px var(--neu-glow)" }}>
            <MapPin style={{ width: 14, height: 14 }} strokeWidth={2} />
            {selectedLoc ? "确认此位置" : "设为浏览锚点"}
          </motion.button>
        </div>
      </div>

      {/* ══ MAP PANEL ═══════════════════════════════════════════════ */}
      <div className="flex-1 relative overflow-hidden" onClick={() => { setSelectedPin(null); setShowFilter(false); }}>
        <iframe src={embedUrl} className="w-full h-full border-0" title="地图" style={{ pointerEvents: "none" }} />

        {/* Current location dot */}
        <div style={{ position: "absolute", left: "50%", top: "50%", transform: "translate(-50%,-50%)", pointerEvents: "none", zIndex: 5 }}>
          <div style={{ width: 18, height: 18, borderRadius: "50%", background: PIN_COLORS.user, border: "3px solid var(--neu-bg)", boxShadow: `0 2px 10px ${pinTone("user", 55)}`, position: "relative" }}>
            <motion.div
              style={{ position: "absolute", inset: -8, borderRadius: "50%", background: pinTone("user", 18), border: `1px solid ${pinTone("user", 25)}` }}
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
            className={`flex items-center gap-2 rounded-full outline-none ${showFilter ? "bg-[var(--neu-accent-primary)] text-white" : "bg-[var(--neu-bg)] text-[var(--neu-text)]"}`}
            style={{
              padding: "9px 16px", fontSize: 13,
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
                className="mt-2 rounded-[16px] bg-[var(--neu-bg)]"
                style={{ boxShadow: "0 0 0 1px var(--neu-border), 0 8px 28px var(--neu-shadow-dark)", width: 200 }}>
                <div style={{ overflow: "hidden", borderRadius: 15 }}>
                {([
                  ["user",  "用户",  markerUserUrl,   markerUserClusterUrl,   PIN_COLORS.user, userCount],
                  ["group", "群组",  markerGroupUrl,  markerGroupClusterUrl,  PIN_COLORS.group, groupCount],
                  ["post",  "动态",  markerMomentUrl, markerMomentClusterUrl, PIN_COLORS.post, postCount],
                ] as const).map(([type, label, icon, clusterIcon, color, count], i) => {
                  const active = activeFilters.has(type as PinType);
                  return (
                    <React.Fragment key={type}>
                      {i > 0 && <div className="bg-[var(--neu-border)]" style={{ height: 1 }} />}
                      <motion.button whileTap={{ scale: 0.97 }}
                        onClick={() => toggleFilter(type as PinType)}
                        className="w-full flex items-center gap-2 px-3 py-2.5 outline-none"
                        style={{ opacity: active ? 1 : 0.45 }}>
                        <img src={icon} alt="" style={{ width: 15, height: 21 }} />
                        <img src={clusterIcon} alt="" style={{ width: 15, height: 21, opacity: 0.65 }} />
                        <span className="text-[var(--neu-text)]" style={{ flex: 1, fontSize: 12, textAlign: "left" }}>{label}</span>
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
                <div className="bg-[var(--neu-border)]" style={{ height: 1 }} />
                <div className="px-3 py-2 flex items-center gap-2">
                  <div style={{ width: 13, height: 13, borderRadius: "50%", background: PIN_COLORS.user, border: "2px solid var(--neu-bg)", boxShadow: `0 0 5px ${pinTone("user", 50)}` }} />
                  <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>我的位置</span>
                </div>
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
                className="bg-[var(--neu-bg)]"
                style={{
                  position: "absolute", bottom: 20, left: 20, right: 20, zIndex: 25,
                  borderRadius: 20, padding: 16,
                  boxShadow: "0 0 0 1px var(--neu-border), 0 16px 48px var(--neu-shadow-dark)",
                }}
              onClick={e => e.stopPropagation()}>
              <div className="flex items-center gap-3">
                {selectedPin.type === "user" && selectedPin.avatar ? (
                  <div className="relative flex-shrink-0">
                    <img src={selectedPin.avatar} className="rounded-full object-cover" style={{ width: 52, height: 52 }} />
                    {selectedPin.online && <span className="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style={{ width: 12, height: 12, border: "2.5px solid var(--neu-bg)" }} />}
                  </div>
                ) : selectedPin.type === "post" && selectedPin.image ? (
                  <img src={selectedPin.image} className="rounded-[14px] object-cover flex-shrink-0" style={{ width: 52, height: 52 }} />
                ) : (
                  <div className="flex items-center justify-center rounded-[15px] flex-shrink-0"
                    style={{ width: 52, height: 52, background: pinTone(selectedPin.type, 14) }}>
                    <img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt="" style={{ width: 26, height: 35 }} />
                  </div>
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <p className="text-[var(--neu-text)]" style={{ fontSize: 15 }}>{selectedPin.name || selectedPin.title}</p>
                    {(selectedPin.count ?? 1) > 1 && (
                      <span style={{ fontSize: 10, padding: "1px 6px", borderRadius: 4, background: PIN_COLORS[selectedPin.type], color: "white" }}>
                        {selectedPin.count}个
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 mt-1 flex-wrap">
                    <span style={{ fontSize: 11, padding: "1px 7px", borderRadius: 4, background: pinTone(selectedPin.type, 15), color: PIN_COLORS[selectedPin.type] }}>
                      {PIN_LABELS[selectedPin.type]}
                    </span>
                    {selectedPin.tag && <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>#{selectedPin.tag}</span>}
                    {selectedPin.distance && (
                      <div className="flex items-center gap-1">
                        <MapPin style={{ width: 10, height: 10, color: "var(--neu-text-muted)" }} strokeWidth={1.5} />
                        <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>{selectedPin.distance}</span>
                      </div>
                    )}
                    {selectedPin.members && <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>{selectedPin.members}名成员</span>}
                    {selectedPin.likes && (
                      <div className="flex items-center gap-1">
                        <Heart style={{ width: 10, height: 10, color: "var(--semantic-hot)" }} strokeWidth={1.5} />
                        <span className="text-[var(--neu-text-muted)]" style={{ fontSize: 12 }}>{selectedPin.likes}</span>
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex gap-2 flex-shrink-0">
                  <motion.button whileTap={{ scale: 0.93 }}
                    className="flex items-center gap-1.5 px-4 py-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
                    style={{ fontSize: 13, boxShadow: "0 4px 14px var(--neu-glow)" }}>
                    查看详情
                    <ChevronRight style={{ width: 13, height: 13 }} strokeWidth={2} />
                  </motion.button>
                  <motion.button whileTap={{ scale: 0.88 }} onClick={() => setSelectedPin(null)}
                    className="outline-none flex items-center justify-center rounded-full bg-[var(--neu-bg)]"
                    style={{ width: 36, height: 36, border: "1px solid var(--neu-border)", boxShadow: "3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light)" }}>
                    <X style={{ width: 14, height: 14, color: "var(--neu-text-muted)" }} strokeWidth={2} />
                  </motion.button>
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Attribution */}
        <div className="text-[var(--neu-text-muted)]" style={{ position: "absolute", bottom: 4, left: 8, fontSize: 9, opacity: 0.38, pointerEvents: "none", zIndex: 1 }}>
          © OpenStreetMap contributors
        </div>
      </div>
    </div>
  );
}


export { LocationDesktop };
