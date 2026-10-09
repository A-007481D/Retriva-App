import React from "react";

interface RetrivaLogoProps {
  size?: number;
  showText?: boolean;
  className?: string;
  variant?: "emerald" | "monochrome";
}

export const RetrivaLogo: React.FC<RetrivaLogoProps> = ({
  size = 32,
  showText = true,
  className = "",
  variant = "emerald",
}) => {
  const gradientId = "retriva-logo-gradient";

  return (
    <div className={`flex items-center gap-2.5 select-none ${className}`}>
      {/* Icon Mark */}
      <svg
        width={size}
        height={size}
        viewBox="0 0 40 40"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="flex-shrink-0 transition-transform duration-300 hover:scale-105"
      >
        <defs>
          <linearGradient id={gradientId} x1="4" y1="4" x2="36" y2="36" gradientUnits="userSpaceOnUse">
            <stop offset="0%" stopColor="#34d399" />
            <stop offset="100%" stopColor="#059669" />
          </linearGradient>
          <filter id="retriva-glow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="2" result="blur" />
            <feComposite in="SourceGraphic" in2="blur" operator="over" />
          </filter>
        </defs>

        {/* Outer Hex-Shield / Chamfered Container */}
        <rect
          x="3"
          y="3"
          width="34"
          height="34"
          rx="10"
          className={
            variant === "emerald"
              ? "fill-[#0e1713] stroke-emerald-500/30"
              : "fill-[#18181b] stroke-white/20"
          }
          strokeWidth="1.5"
        />

        {/* Minimalist Retriva 'R' + Arrow Monogram */}
        {/* Vertical Left Stem */}
        <path
          d="M12 11V29"
          stroke={variant === "emerald" ? "url(#retriva-logo-gradient)" : "#ffffff"}
          strokeWidth="3.5"
          strokeLinecap="round"
        />

        {/* Upper Loop of 'R' */}
        <path
          d="M13.5 13H21.5C24.5376 13 27 15.4624 27 18.5C27 21.5376 24.5376 24 21.5 24H13.5"
          stroke={variant === "emerald" ? "url(#retriva-logo-gradient)" : "#ffffff"}
          strokeWidth="3.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {/* Diagonal Downward Dynamic Leg / Ingest Chevron */}
        <path
          d="M19 22L27.5 28.5"
          stroke={variant === "emerald" ? "#34d399" : "#ffffff"}
          strokeWidth="3.5"
          strokeLinecap="round"
        />

        {/* Micro Tech Pip */}
        <circle
          cx="28"
          cy="12"
          r="1.75"
          className={variant === "emerald" ? "fill-emerald-400" : "fill-white"}
        />
      </svg>

      {/* Wordmark */}
      {showText && (
        <div className="flex flex-col">
          <div className="flex items-center gap-1.5 leading-none">
            <span className="font-black tracking-wider text-white text-[17px] font-sans">
              RETRIVA
            </span>
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]" />
          </div>
          <span className="text-[10px] font-medium tracking-widest text-zinc-400 uppercase font-mono mt-0.5">
            Media Vault
          </span>
        </div>
      )}
    </div>
  );
};
