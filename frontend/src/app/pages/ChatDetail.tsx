import React from "react";
import { useNavigate, useParams } from "react-router";
import { ChatView } from "../features/messaging/ChatView";

export function ChatDetail() {
  // spec: private chats are addressed by the peer's user_id (target_user_id),
  // not a synthetic chat id. The mobile route carries that user_id here.
  const { chatId } = useParams<{ chatId: string }>();
  const navigate = useNavigate();

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      <ChatView
        peerUserId={chatId ?? ""}
        showBackButton
        onBack={() => navigate(-1)}
      />
    </div>
  );
}
