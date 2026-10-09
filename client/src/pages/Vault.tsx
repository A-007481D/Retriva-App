import React, { useEffect, useState, useMemo } from "react";
import {
  FolderArchive,
  Search,
  Video,
  Image as ImageIcon,
  Download,
  Copy,
  Check,
  Play,
  X,
  ExternalLink,
  HardDrive,
  RefreshCw,
  Clock,
  ShieldCheck,
} from "lucide-react";
import { getVault, getMediaFileUrl, type Media } from "../lib/api";
import { cn } from "../lib/utils";
import { motion, AnimatePresence } from "framer-motion";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function formatDate(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return d.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

export const Vault: React.FC = () => {
  const [items, setItems] = useState<Media[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<"all" | "video" | "photo">("all");
  const [selectedMedia, setSelectedMedia] = useState<Media | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const fetchItems = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await getVault();
      setItems(res.items || []);
    } catch (err: any) {
      setError(err.message || "Failed to load vault media");
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchItems();
  }, []);

  const totalSize = useMemo(() => {
    return items.reduce((acc, it) => acc + (it.sizeBytes || 0), 0);
  }, [items]);

  const videoCount = useMemo(() => {
    return items.filter((it) => it.mimeType.startsWith("video")).length;
  }, [items]);

  const photoCount = useMemo(() => {
    return items.filter((it) => it.mimeType.startsWith("image")).length;
  }, [items]);

  const filteredItems = useMemo(() => {
    return items.filter((it) => {
      const isVideo = it.mimeType.startsWith("video");
      const isPhoto = it.mimeType.startsWith("image");

      if (filter === "video" && !isVideo) return false;
      if (filter === "photo" && !isPhoto) return false;

      if (search.trim()) {
        const q = search.toLowerCase();
        const matchName = it.filename.toLowerCase().includes(q);
        const matchSource = (it.sourceUrl || "").toLowerCase().includes(q);
        const matchId = it.id.toLowerCase().includes(q);
        return matchName || matchSource || matchId;
      }
      return true;
    });
  }, [items, filter, search]);

  const handleCopyLink = async (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    const url = window.location.origin + getMediaFileUrl(id);
    try {
      await navigator.clipboard.writeText(url);
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    } catch {
      // fallback
    }
  };

  return (
    <div className="w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-24 pb-32">
      {/* Page Title & Stats */}
      <div className="flex flex-col md:flex-row md:items-end justify-between gap-6 pb-8 border-b border-white/[0.08]">
        <div>
          <div className="flex items-center gap-2.5 mb-2">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-400 shadow-[0_0_10px_rgba(52,211,153,0.8)]" />
            <span className="text-xs font-mono uppercase tracking-widest text-emerald-400 font-semibold">
              Local Vault Storage
            </span>
          </div>
          <h1 className="text-3xl sm:text-4xl font-black tracking-tight text-white">
            Media Vault
          </h1>
          <p className="text-sm text-zinc-400 mt-1 max-w-xl">
            All captured media is permanently preserved locally on your machine with zero expiration.
          </p>
        </div>

        {/* Stats Pill Deck */}
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white/[0.03] border border-white/[0.08]">
            <FolderArchive className="w-4 h-4 text-emerald-400" />
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 font-mono uppercase">Total Files</span>
              <span className="text-xs font-bold text-white font-mono">{items.length}</span>
            </div>
          </div>

          <div className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white/[0.03] border border-white/[0.08]">
            <HardDrive className="w-4 h-4 text-cyan-400" />
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 font-mono uppercase">Vault Size</span>
              <span className="text-xs font-bold text-white font-mono">{formatBytes(totalSize)}</span>
            </div>
          </div>

          <div className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white/[0.03] border border-white/[0.08]">
            <ShieldCheck className="w-4 h-4 text-emerald-400" />
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 font-mono uppercase">Retention</span>
              <span className="text-xs font-bold text-emerald-300 font-mono">Permanent</span>
            </div>
          </div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col sm:flex-row items-center justify-between gap-4 my-6">
        {/* Search */}
        <div className="relative w-full sm:w-80">
          <Search className="w-4 h-4 text-zinc-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search filenames, IDs, or links..."
            className="w-full h-10 pl-10 pr-4 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-zinc-500 outline-none focus:border-emerald-500/50 transition-colors"
          />
        </div>

        {/* Filter Pills */}
        <div className="flex items-center gap-2 w-full sm:w-auto justify-between sm:justify-start">
          <div className="flex items-center gap-1 p-1 rounded-xl bg-white/[0.03] border border-white/[0.08] text-xs">
            <button
              onClick={() => setFilter("all")}
              className={cn(
                "px-3 py-1.5 rounded-lg font-semibold transition-all flex items-center gap-1.5",
                filter === "all"
                  ? "bg-white/[0.12] text-white shadow-sm"
                  : "text-zinc-400 hover:text-white"
              )}
            >
              <span>All</span>
              <span className="px-1.5 py-0.2 rounded bg-white/[0.08] text-[10px] font-mono">
                {items.length}
              </span>
            </button>

            <button
              onClick={() => setFilter("video")}
              className={cn(
                "px-3 py-1.5 rounded-lg font-semibold transition-all flex items-center gap-1.5",
                filter === "video"
                  ? "bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 shadow-sm"
                  : "text-zinc-400 hover:text-white"
              )}
            >
              <Video className="w-3.5 h-3.5 text-emerald-400" />
              <span>Videos</span>
              <span className="px-1.5 py-0.2 rounded bg-white/[0.08] text-[10px] font-mono">
                {videoCount}
              </span>
            </button>

            <button
              onClick={() => setFilter("photo")}
              className={cn(
                "px-3 py-1.5 rounded-lg font-semibold transition-all flex items-center gap-1.5",
                filter === "photo"
                  ? "bg-amber-500/20 text-amber-300 border border-amber-500/30 shadow-sm"
                  : "text-zinc-400 hover:text-white"
              )}
            >
              <ImageIcon className="w-3.5 h-3.5 text-amber-400" />
              <span>Photos</span>
              <span className="px-1.5 py-0.2 rounded bg-white/[0.08] text-[10px] font-mono">
                {photoCount}
              </span>
            </button>
          </div>

          <button
            onClick={fetchItems}
            disabled={isLoading}
            className="w-10 h-10 rounded-xl bg-white/[0.04] border border-white/[0.08] flex items-center justify-center text-zinc-400 hover:text-white hover:bg-white/[0.08] transition-all disabled:opacity-50"
            title="Refresh Vault"
          >
            <RefreshCw className={cn("w-4 h-4", isLoading && "animate-spin")} />
          </button>
        </div>
      </div>

      {/* Media Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4 animate-pulse">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((n) => (
            <div key={n} className="aspect-[4/5] rounded-2xl bg-white/[0.03] border border-white/[0.06]" />
          ))}
        </div>
      ) : error ? (
        <div className="p-12 text-center rounded-2xl bg-rose-500/5 border border-rose-500/20 text-rose-300">
          <p className="font-semibold">{error}</p>
          <button
            onClick={fetchItems}
            className="mt-4 px-4 py-2 rounded-xl bg-rose-500/20 hover:bg-rose-500/30 font-bold text-xs"
          >
            Retry
          </button>
        </div>
      ) : filteredItems.length === 0 ? (
        <div className="py-24 text-center rounded-3xl bg-white/[0.02] border border-white/[0.06] flex flex-col items-center justify-center">
          <FolderArchive className="w-12 h-12 text-zinc-600 mb-3" />
          <h3 className="text-lg font-bold text-white">No items found</h3>
          <p className="text-xs text-zinc-500 max-w-sm mt-1">
            {search
              ? "No files match your search query."
              : "Capture stories or videos using the Extractor and they will appear here."}
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {filteredItems.map((item) => {
            const isVideo = item.mimeType.startsWith("video");
            const fileUrl = getMediaFileUrl(item.id);

            return (
              <div
                key={item.id}
                onClick={() => setSelectedMedia(item)}
                className="group relative aspect-[9/16] rounded-2xl overflow-hidden bg-[#121318] border border-white/[0.08] hover:border-emerald-500/40 transition-all duration-300 cursor-pointer flex flex-col justify-between hover:shadow-xl hover:shadow-black/60"
              >
                {/* Media Preview Media */}
                {isVideo ? (
                  <div className="absolute inset-0 bg-black flex items-center justify-center">
                    <video
                      src={fileUrl}
                      preload="metadata"
                      className="w-full h-full object-cover opacity-80 group-hover:opacity-100 transition-opacity"
                    />
                    <div className="absolute inset-0 flex items-center justify-center">
                      <div className="w-10 h-10 rounded-full bg-black/60 backdrop-blur-md border border-white/20 flex items-center justify-center text-white group-hover:scale-110 group-hover:bg-emerald-500 group-hover:text-black transition-all">
                        <Play className="w-4 h-4 fill-current ml-0.5" />
                      </div>
                    </div>
                  </div>
                ) : (
                  <img
                    src={fileUrl}
                    alt={item.filename}
                    referrerPolicy="no-referrer"
                    className="absolute inset-0 w-full h-full object-cover opacity-85 group-hover:opacity-100 group-hover:scale-105 transition-all duration-300"
                  />
                )}

                {/* Dark Gradient Overlay */}
                <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/20 to-black/60 pointer-events-none" />

                {/* Top Badges */}
                <div className="relative z-10 p-2.5 flex items-center justify-between">
                  <span
                    className={cn(
                      "px-2 py-0.5 rounded-lg text-[10px] font-bold font-mono uppercase tracking-wider backdrop-blur-md border flex items-center gap-1",
                      isVideo
                        ? "bg-teal-500/20 text-teal-300 border-teal-500/30"
                        : "bg-amber-500/20 text-amber-300 border-amber-500/30"
                    )}
                  >
                    {isVideo ? <Video className="w-2.5 h-2.5" /> : <ImageIcon className="w-2.5 h-2.5" />}
                    <span>{isVideo ? "MP4" : "JPG"}</span>
                  </span>

                  <span className="px-2 py-0.5 rounded-lg bg-black/60 backdrop-blur-md border border-white/10 text-[10px] font-mono text-zinc-300 font-semibold">
                    {formatBytes(item.sizeBytes)}
                  </span>
                </div>

                {/* Bottom Details */}
                <div className="relative z-10 p-3 bg-gradient-to-t from-black/95 to-transparent">
                  <p className="text-xs font-semibold text-white truncate group-hover:text-emerald-300 transition-colors">
                    {item.filename}
                  </p>

                  <div className="flex items-center justify-between text-[10px] text-zinc-400 mt-1 font-mono">
                    <span className="flex items-center gap-1 truncate">
                      <Clock className="w-3 h-3 text-zinc-500 flex-shrink-0" />
                      <span>{formatDate(item.createdAt).split(",")[0]}</span>
                    </span>

                    {/* Quick Copy Link Button */}
                    <div className="flex items-center gap-1">
                      <button
                        onClick={(e) => handleCopyLink(item.id, e)}
                        className="w-6 h-6 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center transition-all"
                        title="Copy direct file link"
                      >
                        {copiedId === item.id ? (
                          <Check className="w-3 h-3 text-emerald-400" />
                        ) : (
                          <Copy className="w-3 h-3" />
                        )}
                      </button>
                      <a
                        href={fileUrl}
                        download={item.filename}
                        onClick={(e) => e.stopPropagation()}
                        className="w-6 h-6 rounded-lg bg-white/10 hover:bg-white/20 text-white/70 hover:text-white flex items-center justify-center transition-all"
                        title="Download file directly"
                      >
                        <Download className="w-3 h-3" />
                      </a>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Lightbox / Video Player Modal */}
      <AnimatePresence>
        {selectedMedia && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 md:p-10">
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => setSelectedMedia(null)}
              className="absolute inset-0 bg-black/85 backdrop-blur-md"
            />

            {/* Modal Body */}
            <motion.div
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.95 }}
              className="relative z-10 w-full max-w-4xl max-h-[90vh] bg-[#111216] border border-white/[0.12] rounded-3xl shadow-2xl overflow-hidden flex flex-col"
            >
              {/* Modal Header */}
              <div className="p-4 sm:px-6 flex items-center justify-between border-b border-white/[0.08]">
                <div className="truncate pr-4">
                  <h3 className="text-sm sm:text-base font-bold text-white truncate">
                    {selectedMedia.filename}
                  </h3>
                  <div className="flex items-center gap-3 text-xs text-zinc-400 mt-0.5 font-mono">
                    <span>{formatBytes(selectedMedia.sizeBytes)}</span>
                    <span>•</span>
                    <span>{formatDate(selectedMedia.createdAt)}</span>
                    <span>•</span>
                    <span className="text-emerald-400 flex items-center gap-1">
                      <ShieldCheck className="w-3 h-3" /> Permanent Retention
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <a
                    href={getMediaFileUrl(selectedMedia.id)}
                    download={selectedMedia.filename}
                    className="h-9 px-3.5 rounded-xl bg-emerald-500 text-black hover:bg-emerald-400 font-bold text-xs flex items-center gap-1.5 transition-all shadow-md shadow-emerald-500/20"
                  >
                    <Download className="w-3.5 h-3.5" />
                    <span className="hidden sm:inline">Save File</span>
                  </a>
                  <button
                    onClick={() => setSelectedMedia(null)}
                    className="w-9 h-9 rounded-xl bg-white/10 hover:bg-white/20 text-zinc-300 hover:text-white flex items-center justify-center transition-all"
                  >
                    <X className="w-5 h-5" />
                  </button>
                </div>
              </div>

              {/* Modal Viewer */}
              <div className="flex-1 overflow-auto bg-black flex items-center justify-center p-2 min-h-[360px]">
                {selectedMedia.mimeType.startsWith("video") ? (
                  <video
                    src={getMediaFileUrl(selectedMedia.id)}
                    controls
                    autoPlay
                    className="max-h-[70vh] w-auto max-w-full rounded-xl"
                  />
                ) : (
                  <img
                    src={getMediaFileUrl(selectedMedia.id)}
                    alt={selectedMedia.filename}
                    className="max-h-[70vh] w-auto max-w-full object-contain rounded-xl"
                  />
                )}
              </div>

              {/* Modal Footer / Source */}
              {selectedMedia.sourceUrl && (
                <div className="p-3 sm:px-6 bg-[#0c0d10] border-t border-white/[0.06] flex items-center justify-between text-xs text-zinc-400">
                  <span className="font-mono text-zinc-500 truncate mr-2">
                    Source: {selectedMedia.sourceUrl}
                  </span>
                  <a
                    href={selectedMedia.sourceUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-emerald-400 hover:text-emerald-300 flex items-center gap-1 font-semibold flex-shrink-0"
                  >
                    <span>Original Post</span>
                    <ExternalLink className="w-3 h-3" />
                  </a>
                </div>
              )}
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
};
