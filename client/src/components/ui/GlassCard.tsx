import React from "react";
import { cn } from "../../lib/utils";

interface GlassCardProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  variant?: "light" | "heavy";
}

export const GlassCard = React.forwardRef<HTMLDivElement, GlassCardProps>(
  ({ children, className, variant = "heavy", ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          variant === "heavy" ? "glass-panel" : "glass",
          className
        )}
        {...props}
      >
        {children}
      </div>
    );
  }
);
GlassCard.displayName = "GlassCard";
