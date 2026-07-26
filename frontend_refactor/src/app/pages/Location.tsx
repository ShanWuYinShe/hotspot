import { usePlatform } from "../contexts/PlatformContext";
import { LocationDesktop } from "../features/location/LocationDesktop";
import { LocationMobile } from "../features/location/LocationMobile";

/** Platform boundary only — feature implementations live in features/location. */
export function Location() {
  const { platform } = usePlatform();
  return platform === "desktop" ? <LocationDesktop /> : <LocationMobile />;
}
