import { Toaster as Sonner, type ToasterProps } from "sonner";
import { useTheme } from "@/lib/theme/theme-context";

export function Toaster({ ...props }: ToasterProps) {
  const { isDark } = useTheme();

  return (
    <Sonner
      theme={isDark ? "dark" : "light"}
      className="toaster group"
      toastOptions={{
        style: {
          background: "var(--neu-bg)",
          color: "var(--neu-text)",
          border: "1px solid var(--neu-border)",
          boxShadow:
            "6px 6px 14px var(--neu-shadow-dark), -6px -6px 14px var(--neu-shadow-light)",
          borderRadius: 14,
          fontSize: 13,
        },
      }}
      {...props}
    />
  );
}
