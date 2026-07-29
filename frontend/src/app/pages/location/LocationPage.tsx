import { usePlatform } from "../../contexts/PlatformContext";
import { LocationDesktop } from "./LocationDesktop";
import { LocationMobile } from "./LocationMobile";

/** Platform boundary only — desktop/mobile implementations live alongside in this folder. */
export function LocationPage() {
  const { platform } = usePlatform();
  return platform === "desktop" ? <LocationDesktop /> : <LocationMobile />;
}
