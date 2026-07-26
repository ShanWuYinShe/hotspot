import React from "react";
import { useNavigate, useParams } from "react-router";
import { ChatView } from "../features/messaging/ChatView";

export function ChatDetail() {
  const { chatId } = useParams<{ chatId: string }>();
  const navigate = useNavigate();
  const id = Number(chatId);

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      <ChatView
        chatId={id}
        showBackButton
        onBack={() => navigate(-1)}
      />
    </div>
  );
}
