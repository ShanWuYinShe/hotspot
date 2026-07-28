import { LoginPromptSheet } from "./LoginPromptSheet";
import { BannedScreen } from "./BannedScreen";
import { useAuth } from "../../contexts/AuthContext";

// Global auth overlays: the action-gated login prompt and the account-banned
// screen. Mounted once inside AuthProvider so they cover every route regardless
// of which layout (mobile/tablet/desktop) is active.

export function AuthGate() {
  const { loginPromptOpen, pendingAction, closeLoginPrompt, bannedMessage, dismissBanned } = useAuth();
  return (
    <>
      <LoginPromptSheet
        open={loginPromptOpen}
        actionLabel={pendingAction?.action ?? null}
        onClose={closeLoginPrompt}
        onSuccess={closeLoginPrompt}
      />
      {bannedMessage && (
        <BannedScreen message={bannedMessage} onAcknowledge={dismissBanned} />
      )}
    </>
  );
}
