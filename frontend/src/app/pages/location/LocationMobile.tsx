'use client'
import React, { useState, useRef, useCallback, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Search, MapPin, Navigation, X, Users, ChevronRight, Clock, Star, TrendingUp, SlidersHorizontal, Crosshair, Heart, Check } from "lucide-react";
import { motion, AnimatePresence, useMotionValue, useTransform, animate } from "motion/react";
import { PageHeader } from "@/components/shared/PageHeader";
import type { PinType, Pin, LocationResult } from "@/app/pages/location/location.data";
import { XUZHOU_CENTER, MOCK_PINS, RECENT_LOCATIONS, HOT_LOCATIONS, SEARCH_DB } from "@/app/pages/location/location.data";
import { SNAP_MIN, SNAP_MID, SNAP_MAX, SNAP_HEIGHTS, type SnapState, PIN_COLORS, PIN_LABELS, pinTone, floatBtnStyle, buildMapData, MapPinMarker, getMarkerUrl, markerUserUrl, markerGroupUrl, markerMomentUrl } from "@/app/pages/location/location.shared";
import { useDismissibleLayer } from "@/hooks/useDismissibleLayer";

function LocationMobile() {
  const router = useRouter();

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
  useDismissibleLayer({ open: showFilter, onDismiss: () => setShowFilter(false) });
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
    const proj = h + vel * 90;
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
      drag.current.velY  = (drag.current.prevY - ev.clientY) / dt;
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
    router.back();
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
    <div className="bg-[var(--neu-bg)]" style={{ position: "relative", width: "100%", height: "100%", overflow: "hidden" }}>

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
            background: PIN_COLORS.user,
            border: "3px solid var(--neu-bg)",
            boxShadow: `0 2px 10px ${pinTone("user", 55)}`,
            position: "relative",
          }}>
            <motion.div
              style={{
                position: "absolute", inset: -7, borderRadius: "50%",
                background: pinTone("user", 20),
                border: `1px solid ${pinTone("user", 30)}`,
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
        <div className="text-[var(--neu-text-muted)]" style={{ position: "absolute", bottom: 4, left: 6, fontSize: 9, opacity: 0.4, pointerEvents: "none", zIndex: 1 }}>
          © OpenStreetMap contributors
        </div>
      </div>

      {/* ══ TOP BAR ════════════════════════════════ */}
      <div style={{ position: "absolute", top: 0, left: 0, right: 0, zIndex: 50 }}>
        <PageHeader title="地图探索" onBack={() => router.back()}>
          <motion.button
            whileTap={{ scale: 0.93 }}
            onClick={e => { e.stopPropagation(); setShowFilter(v => !v); }}
            className={`flex items-center gap-2 rounded-full outline-none ${showFilter ? "bg-[var(--neu-accent-primary)] text-white" : "bg-[var(--neu-bg)] text-[var(--neu-text)]"}`}
            style={{
              padding: "9px 16px",
              fontSize: 14,
              border: showFilter ? "1px solid transparent" : "1px solid var(--neu-border)",
              boxShadow: showFilter
                ? "0 4px 16px var(--neu-glow)"
                : "0 2px 12px var(--neu-shadow-dark), 0 1px 3px var(--neu-shadow-dark)",
            }}>
            <SlidersHorizontal style={{ width: 14, height: 14 }} strokeWidth={showFilter ? 2.5 : 1.75} />
            筛选
          </motion.button>
        </PageHeader>

        {/* ── Filter legend panel — anchored BELOW the filter button ── */}
        <AnimatePresence>
          {showFilter && (
            <motion.div
              initial={{ opacity: 0, y: -10, scale: 0.9 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -10, scale: 0.9 }}
              transition={{ type: "spring", stiffness: 400, damping: 30 }}
              className="bg-[var(--neu-bg)]"
              style={{
                position: "absolute",
                top: 62, right: 14,
                width: 136,
                borderRadius: 16,
                overflow: "hidden",
                border: "1px solid var(--neu-border)",
                boxShadow: "0 8px 28px var(--neu-shadow-dark)",
                zIndex: 60,
              }}
              onClick={e => e.stopPropagation()}>
              {([
                ["user",  "用户", markerUserUrl,   PIN_COLORS.user],
                ["group", "群组", markerGroupUrl,  PIN_COLORS.group],
                ["post",  "动态", markerMomentUrl, PIN_COLORS.post],
              ] as const).map(([type, label, icon, color], i) => {
                const active = activeFilters.has(type as PinType);
                return (
                  <React.Fragment key={type}>
                    {i > 0 && <div className="bg-[var(--neu-border)]" style={{ height: 1 }} />}
                    <motion.button
                      whileTap={{ scale: 0.95 }}
                      onClick={() => toggleFilter(type as PinType)}
                      className="w-full flex items-center gap-2 px-3 py-3 outline-none"
                      style={{ opacity: active ? 1 : 0.38 }}>
                      <img src={icon} alt={label} style={{ width: 18, height: 24, flexShrink: 0 }} />
                      <span className="text-[var(--neu-text)]" style={{ flex: 1, fontSize: 13, textAlign: "left" }}>{label}</span>
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
          bottom: rightCtrlBottom,
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
          <div style={{ width: 14, height: 14, borderRadius: "50%", background: PIN_COLORS.user, border: "2.5px solid var(--neu-bg)", boxShadow: `0 0 8px ${pinTone("user", 65)}` }} />
        </motion.button>

        {/* Zoom is handled via pinch gesture on touch + the map's intrinsic
            controls — duplicate explicit zoom buttons removed per UX cleanup. */}
      </motion.div>

      {/* ══ BOTTOM SHEET (draggable) ════════════════════════════ */}
      <motion.div
        className="bg-[var(--neu-bg)]"
        style={{
          position: "absolute", bottom: 0, left: 0, right: 0,
          height: sheetH,
          zIndex: 20,
          borderRadius: "22px 22px 0 0",
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
          <div className="bg-[var(--neu-border)]" style={{ width: 38, height: 4, borderRadius: 2 }} />
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
              className="flex-1 bg-transparent outline-none text-[var(--neu-text)] text-sm"
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
                      data-active={selectedLoc?.displayName === r.displayName || undefined}
                      aria-pressed={selectedLoc?.displayName === r.displayName}
                      className="app-list-row app-list-row--standard neu-row-hover px-5 outline-none"
                      style={{ borderBottom: i < searchResults.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                      <div className="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-accent-primary)]"
                        style={{ width: 34, height: 34, opacity: 0.12 }}>
                        <MapPin style={{ width: 14, height: 14, color: "var(--neu-accent-primary)" }} strokeWidth={1.75} />
                      </div>
                      <span className="text-[var(--neu-text)]" style={{ fontSize: 14, flex: 1 }}>{r.displayName}</span>
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
                  <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 13, marginTop: 8 }}>未找到匹配地点</p>
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
                          background: pinTone(selectedPin.type, 10),
                          border: `1px solid ${pinTone(selectedPin.type, 30)}`,
                        }}>
                        {selectedPin.type === "user" && selectedPin.avatar ? (
                          <div className="relative flex-shrink-0">
                            <img src={selectedPin.avatar} className="rounded-full object-cover" style={{ width: 42, height: 42 }} />
                            {selectedPin.online && (
                              <span className="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]"
                                style={{ width: 10, height: 10, border: "2px solid var(--neu-bg)" }} />
                            )}
                          </div>
                        ) : selectedPin.type === "post" && selectedPin.image ? (
                          <img src={selectedPin.image} className="rounded-[10px] object-cover flex-shrink-0" style={{ width: 42, height: 42 }} />
                        ) : (
                          <img src={getMarkerUrl(selectedPin.type, selectedPin.count ?? 1)} alt=""
                            style={{ width: 22, height: 29, flexShrink: 0 }} />
                        )}
                        <div className="flex-1 min-w-0">
                          <p className="text-[var(--neu-text)] text-[13px]">{selectedPin.name || selectedPin.title}</p>
                          <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, marginTop: 1 }}>
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
                        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full outline-none ${activeTab === id ? "bg-[var(--neu-accent-primary)] text-white" : "bg-[var(--neu-bg)] text-[var(--neu-text-muted)]"}`}
                        style={activeTab === id
                          ? { fontSize: 12, boxShadow: "0 3px 10px var(--neu-glow)" }
                          : { fontSize: 12, border: "1px solid var(--neu-border)" }
                        }>
                        <Icon style={{ width: 11, height: 11 }} strokeWidth={activeTab === id ? 2.5 : 1.5} />
                        {label}
                      </motion.button>
                    ))}
                    {activeTab === "recent" && recentItems.length > 0 && (
                      <motion.button whileTap={{ scale: 0.9 }} onClick={() => setRecentItems([])}
                        className="ml-auto outline-none text-[var(--neu-text-muted)] text-xs">
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
                            data-active={selectedLoc?.displayName === loc.displayName || undefined}
                            aria-pressed={selectedLoc?.displayName === loc.displayName}
                            className="app-list-row app-list-row--standard neu-row-hover px-4 outline-none"
                            style={{ borderBottom: i < recentItems.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                            <div className="flex items-center justify-center rounded-full flex-shrink-0 bg-[var(--neu-bg)]"
                              style={{ width: 36, height: 36, border: "1px solid var(--neu-border)", boxShadow: "3px 3px 6px var(--neu-shadow-dark), -3px -3px 6px var(--neu-shadow-light)", fontSize: 18 }}>
                              {loc.emoji}
                            </div>
                            <div className="flex-1 min-w-0">
                              <p className="truncate text-[var(--neu-text)] text-sm">{loc.displayName}</p>
                              <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, marginTop: 1 }}>{loc.desc}</p>
                            </div>
                            {selectedLoc?.displayName === loc.displayName
                              ? <Check style={{ width: 14, height: 14, color: "var(--neu-accent-primary)", flexShrink: 0 }} strokeWidth={2.5} />
                              : <ChevronRight style={{ width: 13, height: 13, color: "var(--neu-border)", flexShrink: 0 }} strokeWidth={1.5} />
                            }
                          </motion.button>
                        )) : (
                          <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 13, padding: "12px 20px" }}>暂无最近记录</p>
                        )}
                      </motion.div>
                    )}
                    {activeTab === "hot" && (
                      <motion.div key="hot" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                        {HOT_LOCATIONS.map((loc, i) => (
                          <motion.button key={i} whileTap={{ scale: 0.98 }} onClick={() => handleSelectLoc(loc)}
                            data-active={selectedLoc?.displayName === loc.displayName || undefined}
                            aria-pressed={selectedLoc?.displayName === loc.displayName}
                            className="app-list-row app-list-row--standard neu-row-hover px-4 outline-none"
                            style={{ borderBottom: i < HOT_LOCATIONS.length - 1 ? "1px solid var(--neu-border)" : "none" }}>
                            <div className="flex items-center justify-center rounded-[13px] flex-shrink-0 bg-[var(--neu-accent-primary)]"
                              style={{ width: 36, height: 36, fontSize: 18, opacity: 0.88 }}>
                              {loc.emoji}
                            </div>
                            <div className="flex-1 min-w-0">
                              <p className="truncate text-[var(--neu-text)] text-sm">{loc.displayName}</p>
                              <p className="text-[var(--neu-text-muted)]" style={{ fontSize: 11, marginTop: 1 }}>{loc.desc}</p>
                            </div>
                            <div className="flex items-center gap-1 flex-shrink-0">
                              <Users style={{ width: 11, height: 11, color: "var(--neu-accent-primary)" }} strokeWidth={1.5} />
                              <span className="text-[var(--neu-accent-primary)] text-xs">{loc.users}</span>
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
            className="w-full flex items-center justify-center gap-2 rounded-full outline-none bg-[var(--neu-accent-primary)] text-white"
            style={{
              height: 50, fontSize: 15,
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


export { LocationMobile };
