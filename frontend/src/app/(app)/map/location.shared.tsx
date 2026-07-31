import React from "react";
import { motion } from "motion/react";
import type { PinType, Pin } from "./location.data";
import markerUserUrl from "@/assets/markers/marker_user.svg";
import markerGroupUrl from "@/assets/markers/marker_group.svg";
import markerMomentUrl from "@/assets/markers/marker_moment.svg";
import markerUserClusterUrl from "@/assets/markers/marker_user_cluster.svg";
import markerGroupClusterUrl from "@/assets/markers/marker_group_cluster.svg";
import markerMomentClusterUrl from "@/assets/markers/marker_moment_cluster.svg";

export {
  markerUserUrl,
  markerGroupUrl,
  markerMomentUrl,
  markerUserClusterUrl,
  markerGroupClusterUrl,
  markerMomentClusterUrl,
};

export const SNAP_MIN = 148;
export const SNAP_MID = 318;
export const SNAP_MAX = 540;
export type SnapState = "min" | "mid" | "max";
export const SNAP_HEIGHTS: Record<SnapState, number> = { min: SNAP_MIN, mid: SNAP_MID, max: SNAP_MAX };

export const PIN_COLORS: Record<PinType, string> = { user: "var(--map-user)", group: "var(--map-group)", post: "var(--map-post)" };
export const PIN_LABELS: Record<PinType, string> = { user: "用户", group: "群组", post: "动态" };
export const pinTone = (type: PinType, amount: number) => `color-mix(in srgb, ${PIN_COLORS[type]} ${amount}%, transparent)`;
export const floatBtnStyle: React.CSSProperties = { background: "var(--neu-bg)", border: "1px solid var(--neu-border)", boxShadow: "0 2px 12px var(--neu-shadow-dark), 0 1px 3px var(--neu-shadow-dark)" };

function pinToPercent(lat: number, lng: number, bounds: { minLat: number; maxLat: number; minLng: number; maxLng: number }) {
  return { x: ((lng - bounds.minLng) / (bounds.maxLng - bounds.minLng)) * 100, y: ((bounds.maxLat - lat) / (bounds.maxLat - bounds.minLat)) * 100 };
}
export function getMarkerUrl(type: PinType, count: number) {
  if (type === "user") return count >= 3 ? markerUserClusterUrl : markerUserUrl;
  if (type === "group") return count >= 3 ? markerGroupClusterUrl : markerGroupUrl;
  return count >= 3 ? markerMomentClusterUrl : markerMomentUrl;
}
export function buildMapData(center: { lat: number; lng: number }, zoom: number) {
  const bounds = { minLat: center.lat - zoom, maxLat: center.lat + zoom, minLng: center.lng - zoom, maxLng: center.lng + zoom };
  return { url: `https://www.openstreetmap.org/export/embed.html?bbox=${bounds.minLng},${bounds.minLat},${bounds.maxLng},${bounds.maxLat}&layer=mapnik`, bounds };
}
export function MapPinMarker({ pin, bounds, isSelected, onSelect }: { pin: Pin; bounds: ReturnType<typeof buildMapData>["bounds"]; isSelected: boolean; onSelect: (event: React.MouseEvent) => void }) {
  const pos = pinToPercent(pin.lat, pin.lng, bounds);
  return <motion.button className="absolute outline-none" style={{ left: `${pos.x}%`, top: `${pos.y}%`, transform: "translate(-50%, -100%)", zIndex: isSelected ? 30 : 10 }} onClick={onSelect} animate={{ scale: isSelected ? 1.32 : 1, y: isSelected ? -5 : 0 }} transition={{ type: "spring", stiffness: 500, damping: 28 }}>
    {(pin.count ?? 1) > 1 && <div className="absolute -top-1 -right-1 flex items-center justify-center rounded-full z-10" style={{ width: 16, height: 16, background: PIN_COLORS[pin.type], border: "1.5px solid white", fontSize: 9, color: "white", fontWeight: 700 }}>{pin.count}</div>}
    <img src={getMarkerUrl(pin.type, pin.count ?? 1)} alt={pin.type} style={{ width: 36, height: 48, display: "block" }} />
    {isSelected && <motion.div className="absolute rounded-full pointer-events-none" style={{ inset: "8px 2px", background: PIN_COLORS[pin.type], opacity: .3, filter: "blur(7px)" }} animate={{ scale: [1, 1.8], opacity: [.4, 0] }} transition={{ duration: 1.1, repeat: Infinity }} />}
  </motion.button>;
}
