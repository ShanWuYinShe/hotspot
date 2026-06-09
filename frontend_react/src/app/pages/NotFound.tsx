import { useNavigate } from "react-router";
import { goBack } from "@/lib/utils/navigation";
import { motion } from "motion/react";
import { ArrowLeft, Home } from "lucide-react";

export function NotFound() {
  const navigate = useNavigate();

  return (
    <div
      className="flex flex-col items-center justify-center gap-6"
      style={{ height: "100%", backgroundColor: "var(--neu-bg)" }}
    >
      <span
        style={{
          fontSize: 72,
          fontWeight: 700,
          color: "var(--neu-accent-primary)",
          lineHeight: 1,
          opacity: 0.6,
        }}
      >
        404
      </span>
      <div className="flex flex-col items-center gap-2">
        <p style={{ fontSize: 18, color: "var(--neu-text)", fontWeight: 500 }}>
          页面不存在
        </p>
        <p style={{ fontSize: 13, color: "var(--neu-text-muted)" }}>
          你访问的页面可能已被移动或删除
        </p>
      </div>
      <div className="flex gap-3">
        <motion.button
          whileTap={{ scale: 0.94 }}
          onClick={() => goBack(navigate)}
          aria-label="返回上页"
          className="flex items-center gap-1.5 px-4 py-2.5 rounded-full outline-none"
          style={{
            background: "var(--neu-bg)",
            color: "var(--neu-text-muted)",
            border: "1px solid var(--neu-border)",
            fontSize: 13,
          }}
        >
          <ArrowLeft style={{ width: 14, height: 14 }} strokeWidth={1.5} />
          返回上页
        </motion.button>
        <motion.button
          whileTap={{ scale: 0.97 }}
          onClick={() => navigate("/")}
          aria-label="回到首页"
          className="flex items-center gap-1.5 px-4 py-2.5 rounded-full outline-none"
          style={{
            background: "var(--neu-accent-primary)",
            color: "var(--text-on-accent, white)",
            fontSize: 13,
            boxShadow: "0 4px 14px var(--neu-glow)",
          }}
        >
          <Home style={{ width: 14, height: 14 }} strokeWidth={1.5} />
          回到首页
        </motion.button>
      </div>
    </div>
  );
}
