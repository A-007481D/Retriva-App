import React, { useState, useEffect } from "react";
import { LiquidInput } from "../components/ui/LiquidInput";
import { MediaPreviewSheet, type MediaItem } from "../components/ui/MediaPreviewSheet";
import { createJob, inspectUrl, getVault, getMediaFileUrl, type Media } from "../lib/api";
import {
  Sparkles,
  Camera,
  MonitorPlay,
  Music,
  Hash,
  PlaySquare,
  ShieldCheck,
  FolderArchive,
  Layers,
  ArrowRight,
  Loader2,
  CheckCircle2,
  AlertCircle,
  Video,
  Image as ImageIcon,
} from "lucide-react";
import { NavLink } from "react-router-dom";
import { cn } from "../lib/utils";

const PLATFORMS = [
  { id: "instagram", label: "Instagram Stories & Highlights", icon: Camera, color: "text-pink-400" },
  { id: "reels", label: "Reels & Posts", icon: PlaySquare, color: "text-purple-400" },
  { id: "youtube", label: "YouTube", icon: MonitorPlay, color: "text-red-400" },
  { id: "tiktok", label: "TikTok", icon: Music, color: "text-cyan-400" },
  { id: "twitter", label: "X (Twitter)", icon: Hash, color: "text-blue-400" },
];

export const Home: React.FC = () => {
  const [isSheetOpen, setIsSheetOpen] = useState(false);
  const [previewItems, setPreviewItems] = useState<MediaItem[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [currentUrl, setCurrentUrl] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [status, setStatus] = useState<{ type: "info" | "success" | "error"; message: string } | null>(null);
  const [recentVault, setRecentVault] = useState<Media[]>([]);

  const fetchRecentVault = async () => {
    try {
      const res = await getVault();
      if (res.items) {
        setRecentVault(res.items.slice(0, 4));
      }
    } catch {
      // ignore
    }
  };

  useEffect(() => {
    fetchRecentVault();
  }, []);

  const handleDownload = async (url: string) => {
    if (!url.trim()) return;
    setIsLoading(true);
    setStatus({ type: "info", message: "Inspecting source URL..." });
    setCurrentUrl(url);

    try {
      // Check if it's a highlight or playlist
      if (url.includes("/stories/highlights/") || url.includes("playlist")) {
        setStatus({ type: "info", message: "Fetching highlight metadata and stories..." });
        const res = await inspectUrl(url);
        if (res.items && res.items.length > 0) {
          const items: MediaItem[] = res.items.map((it) => ({
            id: String(it.index),
            type: it.type || (it.duration ? "video" : "photo"),
            title: it.title || `Story #${it.index}`,
            thumbnail: it.thumbnail,
            duration: it.duration ? `${Math.round(it.duration)}s` : undefined,
          }));
          setPreviewItems(items);
          setSelectedIds(new Set()); // start clean so user explicitly chooses
          setIsSheetOpen(true);
          setStatus(null);
          setIsLoading(false);
          return;
        }
      }

      // Single item download
      setStatus({ type: "info", message: "Queuing media download..." });
      await createJob(url);
      setStatus({ type: "success", message: "Download queued successfully! Media saved to permanent vault." });
      setTimeout(() => {
        setStatus(null);
        fetchRecentVault();
      }, 4000);
    } catch (err: any) {
      console.error(err);
      setStatus({ type: "error", message: err.message || "Failed to process URL" });
    } finally {
      setIsLoading(false);
    }
  };

  const handleToggleSelect = (id: string) => {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setSelectedIds(next);
  };

  const handleSelectMultiple = (ids: string[]) => {
    const next = new Set(selectedIds);
    ids.forEach((id) => next.add(id));
    setSelectedIds(next);
  };

  const handleDeselectMultiple = (ids: string[]) => {
    const next = new Set(selectedIds);
    ids.forEach((id) => next.delete(id));
    setSelectedIds(next);
  };

  const handleClearAll = () => {
    setSelectedIds(new Set());
  };

  const handleDownloadSelected = async () => {
    if (selectedIds.size === 0) return;
    setIsLoading(true);
    setStatus({ type: "info", message: `Queuing ${selectedIds.size} stories...` });
    try {
      await createJob(currentUrl, Array.from(selectedIds));
      setIsSheetOpen(false);
      setStatus({
        type: "success",
        message: `Queued ${selectedIds.size} downloads! Saving permanently to your Vault.`,
      });
      setTimeout(() => {
        setStatus(null);
        fetchRecentVault();
      }, 5000);
    } catch (err: any) {
      console.error(err);
      setStatus({ type: "error", message: err.message || "Failed to queue selected downloads" });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="w-full flex flex-col items-center pt-24 px-4 pb-32">
      <div className="w-full max-w-4xl mx-auto flex flex-col items-center space-y-10">
        
        {/* Header / Hero */}
        <div className="text-center space-y-4 pt-4">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs font-mono font-medium">
            <Sparkles className="w-3.5 h-3.5 text-emerald-400" />
            <span>High-Fidelity Media Archival</span>
          </div>

          <h1 className="text-4xl sm:text-5xl md:text-6xl font-black tracking-tight text-white">
            Retrieve & Preserve <br className="hidden sm:inline" />
            <span className="text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-teal-300">
              Any Media Locally
            </span>
          </h1>

          <p className="text-sm sm:text-base text-zinc-400 font-medium max-w-xl mx-auto leading-relaxed">
            Extract Instagram stories, highlights, videos, and photos directly to your machine. No cloud telemetry, no watermarks, permanent local retention.
          </p>
        </div>

        {/* Input Bar */}
        <div className="w-full space-y-4">
          <LiquidInput onSubmitUrl={handleDownload} isLoading={isLoading} />

          {/* Inline Status Message */}
          {status && (
            <div
              className={cn(
                "flex items-center justify-center gap-2.5 p-3 rounded-xl border text-xs font-medium max-w-xl mx-auto transition-all animate-in fade-in",
                status.type === "info" && "bg-zinc-900 border-zinc-700 text-zinc-300",
                status.type === "success" && "bg-emerald-950/40 border-emerald-500/30 text-emerald-300",
                status.type === "error" && "bg-rose-950/40 border-rose-500/30 text-rose-300"
              )}
            >
              {status.type === "info" && <Loader2 className="w-4 h-4 animate-spin text-emerald-400" />}
              {status.type === "success" && <CheckCircle2 className="w-4 h-4 text-emerald-400" />}
              {status.type === "error" && <AlertCircle className="w-4 h-4 text-rose-400" />}
              <span>{status.message}</span>
            </div>
          )}

          {/* Supported Sources Pill Deck */}
          <div className="flex flex-wrap items-center justify-center gap-2 pt-2">
            {PLATFORMS.map((p) => {
              const Icon = p.icon;
              return (
                <div
                  key={p.id}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white/[0.03] border border-white/[0.06] text-zinc-400 text-xs font-medium hover:border-white/[0.12] transition-colors"
                >
                  <Icon className={cn("w-3.5 h-3.5", p.color)} />
                  <span>{p.label}</span>
                </div>
              );
            })}
          </div>
        </div>

        {/* Recent Vault Additions Strip */}
        {recentVault.length > 0 && (
          <div className="w-full pt-4 space-y-3">
            <div className="flex items-center justify-between px-1">
              <div className="flex items-center gap-2">
                <FolderArchive className="w-4 h-4 text-emerald-400" />
                <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-300 font-mono">
                  Recent Vault Captures
                </h3>
              </div>
              <NavLink
                to="/library"
                className="text-xs text-emerald-400 hover:text-emerald-300 font-semibold flex items-center gap-1 transition-colors"
              >
                <span>View All In Vault</span>
                <ArrowRight className="w-3 h-3" />
              </NavLink>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {recentVault.map((m) => {
                const isVideo = m.mimeType.startsWith("video");
                const fileUrl = getMediaFileUrl(m.id);
                return (
                  <NavLink
                    key={m.id}
                    to="/library"
                    className="group relative aspect-[4/3] rounded-xl overflow-hidden bg-[#121318] border border-white/[0.08] hover:border-emerald-500/40 transition-all p-2 flex flex-col justify-between"
                  >
                    {isVideo ? (
                      <video
                        src={fileUrl}
                        preload="metadata"
                        className="absolute inset-0 w-full h-full object-cover opacity-60 group-hover:opacity-85 transition-opacity"
                      />
                    ) : (
                      <img
                        src={fileUrl}
                        alt={m.filename}
                        referrerPolicy="no-referrer"
                        className="absolute inset-0 w-full h-full object-cover opacity-60 group-hover:opacity-85 transition-opacity"
                      />
                    )}
                    <div className="absolute inset-0 bg-gradient-to-t from-black/90 via-black/20 to-transparent pointer-events-none" />

                    <div className="relative z-10 flex items-center justify-between">
                      <span className="px-1.5 py-0.5 rounded-md bg-black/70 backdrop-blur-md text-[9px] font-mono font-bold text-zinc-300 flex items-center gap-1">
                        {isVideo ? <Video className="w-2.5 h-2.5 text-teal-400" /> : <ImageIcon className="w-2.5 h-2.5 text-amber-400" />}
                        <span>{isVideo ? "MP4" : "JPG"}</span>
                      </span>
                    </div>

                    <div className="relative z-10">
                      <p className="text-[11px] font-semibold text-white truncate group-hover:text-emerald-300 transition-colors">
                        {m.filename}
                      </p>
                    </div>
                  </NavLink>
                );
              })}
            </div>
          </div>
        )}

        {/* Pro Architecture Features */}
        <div className="w-full pt-8 grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="p-5 rounded-2xl bg-white/[0.02] border border-white/[0.06] space-y-2">
            <div className="w-8 h-8 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 mb-3">
              <ShieldCheck className="w-4 h-4" />
            </div>
            <h4 className="text-sm font-bold text-white">Permanent Local Retention</h4>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Configured with 0-day expiry policy. Files are permanently cataloged in SQLite and stored locally on your filesystem.
            </p>
          </div>

          <div className="p-5 rounded-2xl bg-white/[0.02] border border-white/[0.06] space-y-2">
            <div className="w-8 h-8 rounded-xl bg-teal-500/10 border border-teal-500/20 flex items-center justify-center text-teal-400 mb-3">
              <Layers className="w-4 h-4" />
            </div>
            <h4 className="text-sm font-bold text-white">Stories & Highlights Selector</h4>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Native Instagram reel resolver pulls all photo & video stories. Select individual cards or use filters before downloading.
            </p>
          </div>

          <div className="p-5 rounded-2xl bg-white/[0.02] border border-white/[0.06] space-y-2">
            <div className="w-8 h-8 rounded-xl bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400 mb-3">
              <FolderArchive className="w-4 h-4" />
            </div>
            <h4 className="text-sm font-bold text-white">Built-in Media Vault</h4>
            <p className="text-xs text-zinc-400 leading-relaxed">
              Instant in-app video playback, full-screen lightbox for photos, searchable filenames, and direct offline downloads.
            </p>
          </div>
        </div>

      </div>

      {/* Story & Highlight Selector Modal */}
      <MediaPreviewSheet
        isOpen={isSheetOpen}
        onClose={() => setIsSheetOpen(false)}
        items={previewItems}
        selectedIds={selectedIds}
        onToggleSelect={handleToggleSelect}
        onSelectMultiple={handleSelectMultiple}
        onDeselectMultiple={handleDeselectMultiple}
        onClearAll={handleClearAll}
        onDownload={handleDownloadSelected}
      />
    </div>
  );
};
