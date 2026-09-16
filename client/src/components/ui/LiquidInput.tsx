import React, { useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { Search, X, Download, ClipboardPaste } from "lucide-react";
import { cn } from "../../lib/utils";

interface LiquidInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  onSubmitUrl?: (url: string) => void;
}

export const LiquidInput = React.forwardRef<HTMLInputElement, LiquidInputProps>(
  ({ className, onSubmitUrl, ...props }, ref) => {
    const [isFocused, setIsFocused] = useState(false);
    const [value, setValue] = useState("");

    const handleSubmit = (e: React.FormEvent) => {
      e.preventDefault();
      if (value.trim() && onSubmitUrl) {
        onSubmitUrl(value.trim());
      }
    };

    const handlePaste = async () => {
      try {
        const text = await navigator.clipboard.readText();
        setValue(text);
        setIsFocused(true);
      } catch (err) {
        console.error("Failed to read clipboard contents: ", err);
      }
    };

    return (
      <div className={cn("relative w-full max-w-3xl mx-auto", className)}>
        {/* Glow / Liquid backplate */}
        <motion.div
          className="absolute -inset-1 rounded-[1.5rem] bg-gradient-to-r from-mesh-1 via-mesh-2 to-mesh-3 blur-md opacity-20 pointer-events-none"
          animate={{
            opacity: isFocused ? 0.6 : 0.2,
            scale: isFocused ? 1.01 : 1,
          }}
          transition={{ duration: 0.4, ease: "easeOut" }}
        />

        <form
          onSubmit={handleSubmit}
          className="relative flex items-center w-full h-[64px] glass-panel !rounded-[1.5rem] px-2 overflow-hidden bg-[#18181b]/60 border-white/10"
        >
          <input
            ref={ref}
            type="url"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onFocus={() => setIsFocused(true)}
            onBlur={() => setIsFocused(false)}
            placeholder="Paste the story or profile link here..."
            className="w-full h-full bg-transparent border-none outline-none text-white px-4 text-[16px] placeholder-white/40 font-medium"
            {...props}
          />

          <AnimatePresence>
            {value.length > 0 && (
              <motion.button
                type="button"
                initial={{ opacity: 0, scale: 0.5 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.5 }}
                onClick={() => setValue("")}
                className="w-6 h-6 flex items-center justify-center rounded-full bg-white/10 hover:bg-white/20 text-white mr-2 flex-shrink-0 transition-colors"
              >
                <X className="w-3.5 h-3.5" />
              </motion.button>
            )}
          </AnimatePresence>

          <div className="flex items-center gap-2 flex-shrink-0 ml-1">
            <button
              type="button"
              onClick={handlePaste}
              className="h-[44px] px-4 rounded-xl bg-white/5 hover:bg-white/10 text-white text-sm font-medium flex items-center gap-2 transition-colors border border-white/5 hidden sm:flex"
            >
              <ClipboardPaste className="w-4 h-4 text-white/70" />
              Paste
            </button>
            
            <button
              type="submit"
              disabled={!value.trim()}
              className={cn(
                "h-[44px] px-6 rounded-xl text-black text-sm font-bold flex items-center gap-2 transition-all",
                value.trim() 
                  ? "bg-gradient-to-r from-teal-400 to-emerald-400 hover:opacity-90 active:scale-95 cursor-pointer shadow-[0_0_15px_rgba(45,212,191,0.4)]" 
                  : "bg-white/10 text-white/40 cursor-not-allowed"
              )}
            >
              <Download className="w-4 h-4" />
              <span className="hidden sm:inline">Download</span>
            </button>
          </div>
        </form>
      </div>
    );
  }
);
LiquidInput.displayName = "LiquidInput";
