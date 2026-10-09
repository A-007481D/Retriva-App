import React, { useState } from "react";
import { X, ArrowRight, ClipboardPaste, Link2 } from "lucide-react";
import { cn } from "../../lib/utils";

interface LiquidInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  onSubmitUrl?: (url: string) => void;
  isLoading?: boolean;
}

export const LiquidInput = React.forwardRef<HTMLInputElement, LiquidInputProps>(
  ({ className, onSubmitUrl, isLoading, ...props }, ref) => {
    const [isFocused, setIsFocused] = useState(false);
    const [value, setValue] = useState("");

    const handleSubmit = (e: React.FormEvent) => {
      e.preventDefault();
      if (value.trim() && onSubmitUrl && !isLoading) {
        onSubmitUrl(value.trim());
      }
    };

    const handlePaste = async () => {
      try {
        const text = await navigator.clipboard.readText();
        if (text) {
          setValue(text);
          setIsFocused(true);
        }
      } catch (err) {
        console.error("Failed to read clipboard contents: ", err);
      }
    };

    return (
      <div className={cn("relative w-full max-w-3xl mx-auto", className)}>
        <form
          onSubmit={handleSubmit}
          className={cn(
            "relative flex items-center w-full h-[62px] rounded-2xl px-2.5 bg-[#121318] border transition-all duration-200 shadow-xl shadow-black/40",
            isFocused
              ? "border-emerald-500/60 ring-1 ring-emerald-500/30 shadow-[0_0_25px_rgba(16,185,129,0.12)]"
              : "border-white/[0.1] hover:border-white/[0.18]"
          )}
        >
          {/* Leading Icon */}
          <div className="pl-2 pr-1 text-zinc-500 flex items-center justify-center">
            <Link2 className="w-5 h-5 text-zinc-400" />
          </div>

          {/* Text Input */}
          <input
            ref={ref}
            type="url"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            placeholder="Paste Instagram story, highlight, post, or video URL..."
            disabled={isLoading}
            className="w-full h-full bg-transparent border-none outline-none text-white px-3 text-[15px] placeholder-zinc-500 font-medium"
            {...props}
          />

          {/* Clear button */}
          {value.length > 0 && !isLoading && (
            <button
              type="button"
              onClick={() => setValue("")}
              className="w-7 h-7 flex items-center justify-center rounded-lg bg-white/[0.06] hover:bg-white/[0.12] text-zinc-400 hover:text-white mr-2 flex-shrink-0 transition-colors"
              title="Clear input"
            >
              <X className="w-4 h-4" />
            </button>
          )}

          {/* Action buttons */}
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* Quick Paste Button */}
            <button
              type="button"
              onClick={handlePaste}
              className="h-10 px-3.5 rounded-xl bg-white/[0.05] hover:bg-white/[0.1] text-zinc-300 hover:text-white text-xs font-semibold flex items-center gap-1.5 transition-all border border-white/[0.06] hidden sm:flex"
            >
              <ClipboardPaste className="w-3.5 h-3.5 text-zinc-400" />
              <span>Paste</span>
            </button>

            {/* Submit / Extract Button */}
            <button
              type="submit"
              disabled={!value.trim() || isLoading}
              className={cn(
                "h-10 px-5 rounded-xl text-black text-xs font-bold flex items-center gap-2 transition-all",
                value.trim() && !isLoading
                  ? "bg-emerald-400 hover:bg-emerald-300 active:scale-95 cursor-pointer shadow-md shadow-emerald-500/25"
                  : "bg-white/[0.08] text-zinc-500 cursor-not-allowed"
              )}
            >
              <span>{isLoading ? "Inspecting..." : "Extract"}</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </button>
          </div>
        </form>
      </div>
    );
  }
);
LiquidInput.displayName = "LiquidInput";
