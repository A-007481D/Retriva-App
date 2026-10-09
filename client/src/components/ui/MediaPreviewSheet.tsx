import React from "react";
import { motion, AnimatePresence } from "framer-motion";
import { X, CheckCircle2 } from "lucide-react";
import { GlassCard } from "./GlassCard";

interface MediaItem {
  id: string;
  title: string;
  thumbnail?: string;
  duration?: string;
}

interface MediaPreviewSheetProps {
  isOpen: boolean;
  onClose: () => void;
  items: MediaItem[];
  selectedIds: Set<string>;
  onToggleSelect: (id: string) => void;
  onSelectAll?: () => void;
  onDeselectAll?: () => void;
  onDownload: () => void;
}

export const MediaPreviewSheet: React.FC<MediaPreviewSheetProps> = ({
  isOpen,
  onClose,
  items,
  selectedIds,
  onToggleSelect,
  onSelectAll,
  onDeselectAll,
  onDownload,
}) => {
  const allSelected = items.length > 0 && selectedIds.size === items.length;

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 z-40 bg-black/60 backdrop-blur-md"
          />

          {/* Sheet */}
          <motion.div
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{ type: "spring", damping: 28, stiffness: 220 }}
            className="fixed bottom-0 left-0 right-0 z-50 h-[85vh] max-h-[900px] bg-[#0c0d14]/95 backdrop-blur-3xl border-t border-white/10 rounded-t-3xl shadow-2xl flex flex-col"
          >
            {/* Handle */}
            <div className="w-full flex justify-center pt-3 pb-1 cursor-grab">
              <div className="w-12 h-1.5 rounded-full bg-white/20" />
            </div>

            {/* Header */}
            <div className="px-6 py-4 flex flex-wrap items-center justify-between gap-3 border-b border-white/5">
              <div>
                <h3 className="text-xl font-bold text-white flex items-center gap-2">
                  <span>Highlight Stories</span>
                  <span className="text-xs px-2.5 py-0.5 rounded-full bg-teal-500/20 text-teal-300 border border-teal-500/30 font-medium">
                    {items.length} stories found
                  </span>
                </h3>
                <p className="text-xs text-white/50 mt-0.5">
                  Choose 1 or multiple stories to download to your vault
                </p>
              </div>

              {/* Header Actions */}
              <div className="flex items-center gap-3">
                <div className="flex items-center gap-2 bg-white/5 p-1 rounded-xl border border-white/10 text-xs">
                  <button
                    type="button"
                    onClick={allSelected ? onDeselectAll : onSelectAll}
                    className="px-3 py-1 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-all font-medium"
                  >
                    {allSelected ? "Deselect All" : "Select All"}
                  </button>
                  {selectedIds.size > 0 && !allSelected && (
                    <button
                      type="button"
                      onClick={onDeselectAll}
                      className="px-3 py-1 rounded-lg text-white/80 hover:text-white hover:bg-white/10 transition-all font-medium"
                    >
                      Clear ({selectedIds.size})
                    </button>
                  )}
                </div>

                <button
                  type="button"
                  onClick={onClose}
                  className="w-8 h-8 rounded-full bg-white/10 flex items-center justify-center text-white/70 hover:text-white hover:bg-white/20 transition-all"
                  aria-label="Close"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Scrollable Grid */}
            <div className="flex-1 overflow-y-auto px-6 py-6 pb-28">
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
                {items.map((item, idx) => {
                  const isSelected = selectedIds.has(item.id);
                  return (
                    <GlassCard
                      key={item.id}
                      variant={isSelected ? "light" : "heavy"}
                      className={`relative aspect-[9/16] p-0 cursor-pointer overflow-hidden rounded-2xl transition-all duration-200 select-none ${
                        isSelected
                          ? "ring-2 ring-teal-400 shadow-[0_0_20px_rgba(45,212,191,0.25)] scale-[0.98]"
                          : "hover:border-white/20 hover:scale-[1.01]"
                      }`}
                      onClick={() => onToggleSelect(item.id)}
                    >
                      {item.thumbnail ? (
                        <img
                          src={item.thumbnail}
                          alt={item.title}
                          referrerPolicy="no-referrer"
                          crossOrigin="anonymous"
                          className="absolute inset-0 w-full h-full object-cover transition-opacity duration-300"
                          style={{ opacity: isSelected ? 0.9 : 0.65 }}
                          onError={(e) => {
                            // If direct image loading fails due to CDN hotlinking
                            (e.target as HTMLElement).style.display = "none";
                          }}
                        />
                      ) : (
                        <div className="absolute inset-0 bg-gradient-to-br from-teal-900/30 via-slate-900 to-indigo-950/40" />
                      )}

                      {/* Dark gradient overlay for text readability */}
                      <div className="absolute inset-0 bg-gradient-to-t from-black/85 via-black/20 to-black/40 pointer-events-none" />

                      {/* Index badge */}
                      <div className="absolute top-2.5 left-2.5 px-2 py-0.5 rounded-lg bg-black/60 backdrop-blur-md border border-white/10 flex items-center justify-center text-[11px] font-bold text-white">
                        #{idx + 1}
                      </div>

                      {/* Selection checkbox indicator */}
                      <div className="absolute top-2.5 right-2.5">
                        {isSelected ? (
                          <div className="w-6 h-6 rounded-full bg-teal-400 text-black flex items-center justify-center shadow-lg">
                            <CheckCircle2 className="w-5 h-5 fill-current" />
                          </div>
                        ) : (
                          <div className="w-6 h-6 rounded-full border-2 border-white/40 bg-black/30 backdrop-blur-sm hover:border-white transition-colors" />
                        )}
                      </div>

                      {/* Item Info Footer */}
                      <div className="absolute bottom-0 left-0 right-0 p-3 pointer-events-none">
                        <p className="text-xs font-semibold text-white line-clamp-1 leading-snug">
                          {item.title}
                        </p>
                        <div className="flex items-center justify-between mt-1 text-[10px] text-white/60">
                          <span>{item.duration || "Video"}</span>
                          <span className="font-mono text-white/40">Story #{idx + 1}</span>
                        </div>
                      </div>
                    </GlassCard>
                  );
                })}
              </div>
            </div>

            {/* Sticky Action Footer */}
            <div className="absolute bottom-0 left-0 right-0 p-4 sm:p-5 bg-[#0c0d14]/90 backdrop-blur-2xl border-t border-white/10 flex items-center justify-between gap-4">
              <div className="text-sm">
                <span className="text-white font-medium">{selectedIds.size}</span>
                <span className="text-white/50"> of {items.length} selected</span>
              </div>

              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-5 h-11 rounded-xl bg-white/10 text-white/80 hover:text-white hover:bg-white/15 font-semibold text-sm transition-all"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={onDownload}
                  disabled={selectedIds.size === 0}
                  className="px-6 h-11 rounded-xl bg-gradient-to-r from-teal-400 to-emerald-400 text-black font-bold text-sm disabled:opacity-30 disabled:cursor-not-allowed hover:opacity-95 transition-all shadow-lg shadow-teal-500/20"
                >
                  {selectedIds.size === 0
                    ? "Select stories to download"
                    : selectedIds.size === 1
                    ? "Download 1 Story"
                    : `Download ${selectedIds.size} Stories`}
                </button>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
