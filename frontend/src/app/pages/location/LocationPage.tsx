'use client'
import { usePlatform } from "@/contexts/PlatformContext";
import { LocationDesktop } from "@/app/pages/location/LocationDesktop";
import { LocationMobile } from "@/app/pages/location/LocationMobile";

/** Platform boundary only — desktop/mobile implementations live alongside in this folder. */
export function LocationPage() {
  const { platform } = usePlatform();
  return platform === "desktop" ? <LocationDesktop /> : <LocationMobile />;
}
