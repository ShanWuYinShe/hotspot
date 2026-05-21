import { useNavigate, useParams } from "react-router";
import { ChatView } from "../components/ui/ChatView";
import { useDocumentTitle } from "@/lib/hooks/use-document-title";
import { goBack } from "@/lib/utils/navigation";

export function ChatDetail() {
  const { chatId } = useParams<{ chatId: string }>();
  const navigate = useNavigate();
  useDocumentTitle("聊天");

  return (
    <div className="flex flex-col" style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}>
      <ChatView
        chatId={chatId ?? ""}
        showBackButton
        onBack={() => goBack(navigate)}
      />
    </div>
  );
}
