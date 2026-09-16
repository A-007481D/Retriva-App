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
  onDownload: () => void;
}

export const MediaPreviewSheet: React.FC<MediaPreviewSheetProps> = ({
  isOpen,
  onClose,
  items,
  selectedIds,
  onToggleSelect,
  onDownload,
}) => {
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
            className="fixed inset-0 z-40 bg-black/40 backdrop-blur-sm"
          />

          {/* Sheet */}
          <motion.div
            initial={{ y: "100%" }}
            animate={{ y: 0 }}
            exit={{ y: "100%" }}
            transition={{ type: "spring", damping: 25, stiffness: 200 }}
            className="fixed bottom-0 left-0 right-0 z-50 h-[80vh] bg-[#0f0c29]/80 backdrop-blur-3xl border-t border-white/10 rounded-t-3xl shadow-2xl flex flex-col"
          >
            {/* Handle */}
            <div className="w-full flex justify-center pt-3 pb-2">
              <div className="w-12 h-1.5 rounded-full bg-white/20" />
            </div>

            {/* Header */}
            <div className="px-6 py-4 flex items-center justify-between border-b border-white/5">
              <div>
                <h3 className="text-lg font-bold text-white">Select Media</h3>
                <p className="text-sm text-white/50">{items.length} items found</p>
              </div>
              <button
                onClick={onClose}
                className="w-8 h-8 rounded-full bg-white/10 flex items-center justify-center text-white/70 hover:text-white"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Scrollable Grid */}
            <div className="flex-1 overflow-y-auto px-6 py-6 pb-24">
              <div className="grid grid-cols-2 gap-4">
                {items.map((item, idx) => {
                  const isSelected = selectedIds.has(item.id);
                  return (
                    <GlassCard
                      key={item.id}
                      variant={isSelected ? "light" : "heavy"}
                      className={`relative aspect-[4/5] p-0 cursor-pointer overflow-hidden transition-all duration-300 ${
                        isSelected ? "ring-2 ring-cyan-400" : ""
                      }`}
                      onClick={() => onToggleSelect(item.id)}
                    >
                      {item.thumbnail ? (
                        <img
                          src={item.thumbnail}
                          alt={item.title}
                          className="absolute inset-0 w-full h-full object-cover opacity-60"
                        />
                      ) : (
                        <div className="absolute inset-0 bg-gradient-to-br from-mesh-1/20 to-mesh-3/20" />
                      )}
                      
                      <div className="absolute top-2 left-2 w-6 h-6 rounded-full glass flex items-center justify-center text-[10px] font-bold">
                        {idx + 1}
                      </div>

                      {isSelected && (
                        <div className="absolute top-2 right-2 text-cyan-400">
                          <CheckCircle2 className="w-6 h-6 fill-black" />
                        </div>
                      )}

                      <div className="absolute bottom-0 left-0 right-0 p-3 bg-gradient-to-t from-black/80 to-transparent">
                        <p className="text-xs font-medium text-white line-clamp-2 leading-tight">
                          {item.title}
                        </p>
                        {item.duration && (
                          <p className="text-[10px] text-white/60 mt-1">{item.duration}</p>
                        )}
                      </div>
                    </GlassCard>
                  );
                })}
              </div>
            </div>

            {/* Sticky Action Footer */}
            <div className="absolute bottom-0 left-0 right-0 p-6 glass-panel rounded-none border-x-0 border-b-0 rounded-t-3xl">
              <button
                onClick={onDownload}
                disabled={selectedIds.size === 0}
                className="w-full h-12 rounded-xl bg-white text-black font-bold disabled:opacity-50 disabled:bg-white/20 disabled:text-white/50 transition-all hover:bg-cyan-50"
              >
                Download Selected ({selectedIds.size})
              </button>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  );
};
