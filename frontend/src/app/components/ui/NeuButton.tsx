import React from "react";
import { cn } from "../../../lib/utils";
import { motion, HTMLMotionProps } from "motion/react";
import { useHoverConfig } from "../../hooks/useHoverConfig";

interface NeuButtonProps extends HTMLMotionProps<"button"> {
  variant?: "flat" | "pressed" | "icon" | "glow";
  size?: "sm" | "md" | "lg" | "icon";
  active?: boolean;
}

export const NeuButton = React.forwardRef<HTMLButtonElement, NeuButtonProps>(
  (
    {
      className,
      variant = "flat",
      size = "md",
      active = false,
      children,
      ...props
    },
    ref,
  ) => {
    const hover = useHoverConfig();
    return (
      <motion.button
        ref={ref}
        whileTap={hover.tap}
        whileHover={
          !active
            ? {
                ...hover.btn,
                y: -2,
              }
            : {}
        }
        className={cn(
          "relative flex items-center justify-center rounded-[28px] outline-none select-none",
          "text-neu font-medium neu-pill-hover",
          size === "sm" && "px-4 py-2 text-sm",
          size === "md" && "px-8 py-3.5",
          size === "lg" && "px-10 py-4 text-lg",
          size === "icon" && "p-4 rounded-full aspect-square",
          variant === "flat" && !active && "neu-flat",
          variant === "pressed" && "neu-pressed",
          variant === "glow" && !active && "neu-button-glow",
          variant === "icon" && !active && "neu-flat text-neu-accent",
          active && "neu-pressed text-neu-accent",
          className,
        )}
        {...props}
      >
        {children}
      </motion.button>
    );
  },
);
NeuButton.displayName = "NeuButton";
