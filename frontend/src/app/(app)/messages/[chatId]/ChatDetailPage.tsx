'use client'
import React from "react";
import { useRouter } from "next/navigation"
import { useParams } from "next/navigation";
import { ChatView } from "../ChatView";

export function ChatDetailPage() {
  // spec: private chats are addressed by the peer's user_id (target_user_id),
  // not a synthetic chat id. The mobile route carries that user_id here.
  const { chatId } = useParams<{ chatId: string }>();
  const router = useRouter();

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      <ChatView
        peerUserId={chatId ?? ""}
        showBackButton
        onBack={() => router.back()}
      />
    </div>
  );
}
