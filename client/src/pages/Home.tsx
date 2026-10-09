import { useState } from "react";
import { LiquidInput } from "../components/ui/LiquidInput";
import { MediaPreviewSheet } from "../components/ui/MediaPreviewSheet";
import { createJob, inspectUrl } from "../lib/api";
import { Check, MonitorPlay, Camera, Hash, Music, PlaySquare, Image as ImageIcon, Loader2 } from "lucide-react";
import { cn } from "../lib/utils";

const FEATURES = [
  "High Quality",
  "No Watermarks",
  "Secure & Private",
  "Unlimited Usage"
];

const PLATFORMS = [
  { id: "youtube", label: "YouTube", icon: MonitorPlay },
  { id: "instagram", label: "Instagram", icon: Camera },
  { id: "tiktok", label: "TikTok", icon: Music },
  { id: "twitter", label: "X (Twitter)", icon: Hash },
  { id: "reels", label: "Reels/Shorts", icon: PlaySquare },
  { id: "photo", label: "Photo", icon: ImageIcon },
];

export const Home = () => {
  const [activePlatform, setActivePlatform] = useState("youtube");
  const [isSheetOpen, setIsSheetOpen] = useState(false);
  const [previewItems, setPreviewItems] = useState<{ id: string; title: string; thumbnail?: string; duration?: string }[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [currentUrl, setCurrentUrl] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const handleDownload = async (url: string) => {
    if (!url.trim()) return;
    setIsLoading(true);
    setStatusMessage("Checking link...");
    setCurrentUrl(url);

    try {
      // Check if it's a highlight or playlist
      if (url.includes("/stories/highlights/") || url.includes("playlist")) {
        setStatusMessage("Inspecting highlight stories...");
        const res = await inspectUrl(url);
        if (res.items && res.items.length > 1) {
          const items = res.items.map((it) => ({
            id: String(it.index),
            title: it.title || `Story #${it.index}`,
            thumbnail: it.thumbnail,
            duration: it.duration ? `${Math.round(it.duration)}s` : undefined,
          }));
          setPreviewItems(items);
          setSelectedIds(new Set()); // start empty so user explicitly chooses
          setIsSheetOpen(true);
          setStatusMessage(null);
          setIsLoading(false);
          return;
        }
      }

      // Single item download
      setStatusMessage("Queuing download...");
      await createJob(url);
      setStatusMessage("Download started! Media will never expire.");
      setTimeout(() => setStatusMessage(null), 4000);
    } catch (err: any) {
      console.error(err);
      setStatusMessage(`Error: ${err.message}`);
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

  const handleSelectAll = () => {
    setSelectedIds(new Set(previewItems.map((i) => i.id)));
  };

  const handleDeselectAll = () => {
    setSelectedIds(new Set());
  };

  const handleDownloadSelected = async () => {
    if (selectedIds.size === 0) return;
    setIsLoading(true);
    setStatusMessage(`Queuing ${selectedIds.size} stories...`);
    try {
      await createJob(currentUrl, Array.from(selectedIds));
      setIsSheetOpen(false);
      setStatusMessage(`Successfully queued ${selectedIds.size} downloads!`);
      setTimeout(() => setStatusMessage(null), 5000);
    } catch (err: any) {
      console.error(err);
      setStatusMessage(`Error: ${err.message}`);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="w-full h-full flex flex-col items-center pt-24 px-4 pb-32 overflow-y-auto">
      <div className="w-full max-w-4xl mx-auto flex flex-col items-center space-y-8">
        
        {/* Hero Headers */}
        <div className="text-center space-y-4">
          <h1 className="text-4xl md:text-5xl lg:text-6xl font-black tracking-tight text-white drop-shadow-xl">
            Universal <span className="text-transparent bg-clip-text bg-gradient-to-r from-teal-400 to-emerald-400">Media Extractor</span>
          </h1>
          <p className="text-base md:text-lg text-white/60 font-medium max-w-xl mx-auto">
            Capture high-quality content instantly. No limits, totally secure, and extremely fast.
          </p>
        </div>

        {/* Input Section */}
        <div className="w-full space-y-6 pt-4">
          <LiquidInput onSubmitUrl={handleDownload} />

          {statusMessage && (
            <div className="flex items-center justify-center gap-2 p-3 rounded-xl bg-white/10 border border-teal-400/30 text-teal-300 text-sm font-medium">
              {isLoading && <Loader2 className="w-4 h-4 animate-spin text-teal-400" />}
              <span>{statusMessage}</span>
            </div>
          )}

          {/* Feature Tags */}
          <div className="flex flex-wrap items-center justify-center gap-3">
            {FEATURES.map((feature) => (
              <div 
                key={feature} 
                className="flex items-center gap-1.5 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-white/70 text-xs font-medium"
              >
                <Check className="w-3 h-3 text-teal-400" />
                {feature}
              </div>
            ))}
          </div>

          <p className="text-center text-xs text-white/40 font-medium mt-4">
            One tool for every platform — paste a <strong className="text-white/70 font-semibold">YouTube, TikTok, Instagram, X (Twitter),</strong> or <strong className="text-white/70 font-semibold">Facebook</strong> link.
          </p>
        </div>

        {/* Platform Selector Cards */}
        <div className="w-full pt-8 pb-4">
          <div className="flex overflow-x-auto hide-scrollbar gap-3 sm:gap-4 pb-4 snap-x snap-mandatory px-4 md:px-0 justify-start md:justify-center">
            {PLATFORMS.map((platform) => {
              const Icon = platform.icon;
              const isActive = activePlatform === platform.id;
              return (
                <button
                  key={platform.id}
                  onClick={() => setActivePlatform(platform.id)}
                  className={cn(
                    "relative flex flex-col items-center justify-center gap-2 min-w-[80px] sm:min-w-[90px] h-[80px] sm:h-[90px] rounded-2xl transition-all duration-300 snap-center shrink-0",
                    isActive 
                      ? "bg-white/10 border-teal-400/50 shadow-[0_0_20px_rgba(45,212,191,0.15)]" 
                      : "bg-white/5 border-white/5 hover:bg-white/10",
                    "border border-solid"
                  )}
                >
                  <Icon 
                    className={cn(
                      "w-6 h-6 sm:w-7 sm:h-7 transition-colors duration-300",
                      isActive ? "text-teal-400" : "text-white/60"
                    )} 
                  />
                  <span className={cn(
                    "text-[10px] sm:text-xs font-semibold transition-colors duration-300",
                    isActive ? "text-white" : "text-white/50"
                  )}>
                    {platform.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {/* How it works (Skeleton for now) */}
        <div className="w-full max-w-4xl mt-16 space-y-6">
          <div className="flex items-center gap-3">
            <div className="h-[2px] w-8 bg-teal-400/50" />
            <h2 className="text-[10px] font-bold tracking-[0.2em] text-teal-400 uppercase">
              Three Steps
            </h2>
          </div>
          <h3 className="text-2xl font-bold text-white">How it works</h3>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 pt-4">
            {[1, 2, 3].map((step) => (
              <div key={step} className="glass-panel h-64 rounded-2xl relative overflow-hidden flex flex-col p-6 border-white/5">
                <div className="text-5xl font-black text-white/5 absolute -top-4 -right-2">0{step}</div>
                <div className="mt-auto space-y-2">
                  <h4 className="text-lg font-bold text-white">
                    {step === 1 ? "Copy the link" : step === 2 ? "Paste into tool" : "Download media"}
                  </h4>
                  <p className="text-sm text-white/50 font-medium">
                    {step === 1 
                      ? "Find the media you want to save and copy its URL." 
                      : step === 2 
                      ? "Paste the copied URL into our search bar above." 
                      : "Click download to instantly save it to your device."}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <MediaPreviewSheet
          isOpen={isSheetOpen}
          onClose={() => setIsSheetOpen(false)}
          items={previewItems}
          selectedIds={selectedIds}
          onToggleSelect={handleToggleSelect}
          onSelectAll={handleSelectAll}
          onDeselectAll={handleDeselectAll}
          onDownload={handleDownloadSelected}
        />

      </div>
    </div>
  );
};
