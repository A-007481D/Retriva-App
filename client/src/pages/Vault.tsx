import React, { useEffect, useState, useMemo, useCallback } from "react";
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
  Trash2,
  CheckSquare,
  Square,
  AlertTriangle,
  Loader2,
  CheckCheck,
  ChevronLeft,
  ChevronRight,
  Calendar,
} from "lucide-react";
import { getVault, getMediaFileUrl, deleteMedia, batchDeleteMedia, type Media } from "../lib/api";
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

function formatTimeOnly(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    return d.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

function getDateKey(dateStr: string): string {
  try {
    const d = new Date(dateStr);
    const year = d.getFullYear();
    const month = String(d.getMonth() + 1).padStart(2, "0");
    const day = String(d.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
  } catch {
    return (dateStr || "").substring(0, 10);
  }
}

function formatDateHeader(dateKey: string): {
  title: string;
  subtitle: string;
  isToday: boolean;
  isYesterday: boolean;
} {
  try {
    const parts = dateKey.split("-");
    const d = new Date(parseInt(parts[0], 10), parseInt(parts[1], 10) - 1, parseInt(parts[2], 10));
    const now = new Date();

    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const target = new Date(d.getFullYear(), d.getMonth(), d.getDate());
    const diffDays = Math.round((today.getTime() - target.getTime()) / (1000 * 60 * 60 * 24));

    const formattedLong = d.toLocaleDateString("en-US", {
      weekday: "short",
      month: "short",
      day: "numeric",
      year: "numeric",
    });

    if (diffDays === 0) {
      return { title: "Today", subtitle: formattedLong, isToday: true, isYesterday: false };
    } else if (diffDays === 1) {
      return { title: "Yesterday", subtitle: formattedLong, isToday: false, isYesterday: true };
    } else if (diffDays > 1 && diffDays < 7) {
      const weekday = d.toLocaleDateString("en-US", { weekday: "long" });
      return { title: weekday, subtitle: formattedLong, isToday: false, isYesterday: false };
    } else {
      const title = d.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" });
      const subtitle = d.toLocaleDateString("en-US", { weekday: "long" });
      return { title, subtitle, isToday: false, isYesterday: false };
    }
  } catch {
    return { title: dateKey, subtitle: "", isToday: false, isYesterday: false };
  }
}

type DeleteTarget =
  | { type: "single"; item: Media }
  | { type: "batch"; ids: string[]; count: number; totalBytes: number };

interface DateGroup {
  dateKey: string;
  title: string;
  subtitle: string;
  isToday: boolean;
  isYesterday: boolean;
  items: Media[];
  totalBytes: number;
  videoCount: number;
  photoCount: number;
}

export const Vault: React.FC = () => {
  const [items, setItems] = useState<Media[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<"all" | "video" | "photo">("all");
  const [dateFilter, setDateFilter] = useState<string>("all");
  const [selectedMedia, setSelectedMedia] = useState<Media | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  // Selection & Batch Delete state
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  // Delete confirmation modal state
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [totalCount, setTotalCount] = useState<number>(0);

  const fetchItems = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await getVault();
      setItems(res.items || []);
      setTotalCount(res.total ?? (res.items ? res.items.length : 0));
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

  // Distinct dates present in the vault for the dropdown selector
  const availableDates = useMemo(() => {
    const map = new Map<string, number>();
    for (const it of items) {
      const k = getDateKey(it.createdAt);
      map.set(k, (map.get(k) || 0) + 1);
    }
    return Array.from(map.entries())
      .sort(([a], [b]) => b.localeCompare(a))
      .map(([dateKey, count]) => {
        const header = formatDateHeader(dateKey);
        return {
          dateKey,
          label: `${header.title} (${count})`,
          count,
        };
      });
  }, [items]);

  const filteredItems = useMemo(() => {
    return items.filter((it) => {
      const isVideo = it.mimeType.startsWith("video");
      const isPhoto = it.mimeType.startsWith("image");

      if (filter === "video" && !isVideo) return false;
      if (filter === "photo" && !isPhoto) return false;

      // Date Filter
      if (dateFilter !== "all") {
        const itemDateKey = getDateKey(it.createdAt);
        if (dateFilter === "today") {
          const todayKey = getDateKey(new Date().toISOString());
          if (itemDateKey !== todayKey) return false;
        } else if (dateFilter === "yesterday") {
          const yesterday = new Date();
          yesterday.setDate(yesterday.getDate() - 1);
          const yestKey = getDateKey(yesterday.toISOString());
          if (itemDateKey !== yestKey) return false;
        } else if (dateFilter === "week") {
          const itemTime = new Date(it.createdAt).getTime();
          const weekAgo = Date.now() - 7 * 24 * 60 * 60 * 1000;
          if (itemTime < weekAgo) return false;
        } else if (dateFilter === "month") {
          const itemTime = new Date(it.createdAt).getTime();
          const monthAgo = Date.now() - 30 * 24 * 60 * 60 * 1000;
          if (itemTime < monthAgo) return false;
        } else {
          // Specific dateKey (e.g. "2026-10-27")
          if (itemDateKey !== dateFilter) return false;
        }
      }

      if (search.trim()) {
        const q = search.toLowerCase();
        const matchName = it.filename.toLowerCase().includes(q);
        const matchSource = (it.sourceUrl || "").toLowerCase().includes(q);
        const matchId = it.id.toLowerCase().includes(q);
        return matchName || matchSource || matchId;
      }
      return true;
    });
  }, [items, filter, dateFilter, search]);

  // Group filtered items by date phase
  const dateGroups = useMemo<DateGroup[]>(() => {
    const groupsMap = new Map<string, Media[]>();

    for (const item of filteredItems) {
      const key = getDateKey(item.createdAt);
      let arr = groupsMap.get(key);
      if (!arr) {
        arr = [];
        groupsMap.set(key, arr);
      }
      arr.push(item);
    }

    // Sort descending: newest phase first
    const sortedKeys = Array.from(groupsMap.keys()).sort((a, b) => b.localeCompare(a));

    return sortedKeys.map((key) => {
      const groupItems = groupsMap.get(key)!;
      const header = formatDateHeader(key);
      const groupTotalBytes = groupItems.reduce((acc, it) => acc + (it.sizeBytes || 0), 0);
      const groupVideos = groupItems.filter((it) => it.mimeType.startsWith("video")).length;
      const groupPhotos = groupItems.filter((it) => it.mimeType.startsWith("image")).length;

      return {
        dateKey: key,
        ...header,
        items: groupItems,
        totalBytes: groupTotalBytes,
        videoCount: groupVideos,
        photoCount: groupPhotos,
      };
    });
  }, [filteredItems]);

  const selectedSize = useMemo(() => {
    let sum = 0;
    for (const it of items) {
      if (selectedIds.has(it.id)) {
        sum += it.sizeBytes || 0;
      }
    }
    return sum;
  }, [items, selectedIds]);

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

  const toggleSelect = (id: string, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
    if (!selectionMode) {
      setSelectionMode(true);
    }
  };

  const toggleSelectDateGroup = (groupItems: Media[]) => {
    const groupIds = groupItems.map((it) => it.id);
    const allSelected = groupIds.length > 0 && groupIds.every((id) => selectedIds.has(id));

    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (allSelected) {
        groupIds.forEach((id) => next.delete(id));
      } else {
        groupIds.forEach((id) => next.add(id));
      }
      return next;
    });

    if (!allSelected && !selectionMode) {
      setSelectionMode(true);
    }
  };

  const handleSelectAllFiltered = () => {
    const allFilteredIds = filteredItems.map((it) => it.id);
    const allSelected = allFilteredIds.length > 0 && allFilteredIds.every((id) => selectedIds.has(id));

    if (allSelected) {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        allFilteredIds.forEach((id) => next.delete(id));
        return next;
      });
    } else {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        allFilteredIds.forEach((id) => next.add(id));
        return next;
      });
      setSelectionMode(true);
    }
  };

  const handleClearSelection = () => {
    setSelectedIds(new Set());
    setSelectionMode(false);
  };

  const openSingleDelete = (item: Media, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    setDeleteError(null);
    setDeleteTarget({ type: "single", item });
  };

  const openBatchDelete = () => {
    if (selectedIds.size === 0) return;
    setDeleteError(null);
    setDeleteTarget({
      type: "batch",
      ids: Array.from(selectedIds),
      count: selectedIds.size,
      totalBytes: selectedSize,
    });
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setIsDeleting(true);
    setDeleteError(null);

    try {
      if (deleteTarget.type === "single") {
        const targetId = deleteTarget.item.id;
        await deleteMedia(targetId);

        setItems((prev) => prev.filter((it) => it.id !== targetId));
        setTotalCount((prev) => Math.max(0, prev - 1));

        if (selectedMedia?.id === targetId) {
          setSelectedMedia(null);
        }
        setSelectedIds((prev) => {
          const next = new Set(prev);
          next.delete(targetId);
          return next;
        });
      } else {
        const ids = deleteTarget.ids;
        const res = await batchDeleteMedia(ids);
        const deletedSet = new Set(res.deleted);

        setItems((prev) => prev.filter((it) => !deletedSet.has(it.id)));
        setTotalCount((prev) => Math.max(0, prev - res.deleted.length));

        if (selectedMedia && deletedSet.has(selectedMedia.id)) {
          setSelectedMedia(null);
        }
        setSelectedIds((prev) => {
          const next = new Set(prev);
          res.deleted.forEach((id) => next.delete(id));
          return next;
        });

        if (res.failed && res.failed.length > 0) {
          setDeleteError(`Deleted ${res.deleted.length} file(s), but ${res.failed.length} failed.`);
          return;
        }
        setSelectionMode(false);
      }

      setDeleteTarget(null);
    } catch (err: any) {
      setDeleteError(err.message || "Failed to delete file(s)");
    } finally {
      setIsDeleting(false);
    }
  };

  // Lightbox keyboard navigation
  const currentIndex = useMemo(() => {
    if (!selectedMedia) return -1;
    return filteredItems.findIndex((it) => it.id === selectedMedia.id);
  }, [selectedMedia, filteredItems]);

  const handlePrev = useCallback(() => {
    if (filteredItems.length === 0) return;
    if (currentIndex <= 0) {
      setSelectedMedia(filteredItems[filteredItems.length - 1]);
    } else {
      setSelectedMedia(filteredItems[currentIndex - 1]);
    }
  }, [currentIndex, filteredItems]);

  const handleNext = useCallback(() => {
    if (filteredItems.length === 0) return;
    if (currentIndex === -1 || currentIndex >= filteredItems.length - 1) {
      setSelectedMedia(filteredItems[0]);
    } else {
      setSelectedMedia(filteredItems[currentIndex + 1]);
    }
  }, [currentIndex, filteredItems]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (deleteTarget) return;

      const target = e.target as HTMLElement;
      if (
        target &&
        (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)
      ) {
        return;
      }

      if (selectedMedia) {
        if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
          e.preventDefault();
          handlePrev();
        } else if (e.key === "ArrowRight" || e.key === "ArrowDown") {
          e.preventDefault();
          handleNext();
        } else if (e.key === "Escape") {
          e.preventDefault();
          setSelectedMedia(null);
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [selectedMedia, deleteTarget, handlePrev, handleNext]);

  const allVisibleSelected =
    filteredItems.length > 0 && filteredItems.every((it) => selectedIds.has(it.id));

  // Render individual media card
  const renderCard = (item: Media) => {
    const isVideo = item.mimeType.startsWith("video");
    const fileUrl = getMediaFileUrl(item.id);
    const isSelected = selectedIds.has(item.id);

    return (
      <div
        key={item.id}
        onClick={(e) => {
          if (selectionMode) {
            toggleSelect(item.id, e);
          } else {
            setSelectedMedia(item);
          }
        }}
        className={cn(
          "group relative aspect-[9/16] rounded-2xl overflow-hidden bg-[#121318] border transition-all duration-200 cursor-pointer flex flex-col justify-between hover:shadow-xl hover:shadow-black/60 select-none",
          isSelected
            ? "ring-2 ring-emerald-400 border-emerald-400/80 shadow-[0_0_20px_rgba(52,211,153,0.25)] scale-[0.98]"
            : "border-white/[0.08] hover:border-emerald-500/40 hover:scale-[1.01]"
        )}
      >
        {/* Media Preview Media */}
        {isVideo ? (
          <div className="absolute inset-0 bg-black flex items-center justify-center">
            <video
              src={fileUrl}
              preload="metadata"
              className="w-full h-full object-cover opacity-80 group-hover:opacity-100 transition-opacity"
            />
            {!selectionMode && (
              <div className="absolute inset-0 flex items-center justify-center">
                <div className="w-10 h-10 rounded-full bg-black/60 backdrop-blur-md border border-white/20 flex items-center justify-center text-white group-hover:scale-110 group-hover:bg-emerald-500 group-hover:text-black transition-all">
                  <Play className="w-4 h-4 fill-current ml-0.5" />
                </div>
              </div>
            )}
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

        {/* Top Bar: Selection Checkbox & Type Badge */}
        <div className="relative z-10 p-2.5 flex items-center justify-between">
          {/* Select Checkbox */}
          <button
            type="button"
            onClick={(e) => toggleSelect(item.id, e)}
            className={cn(
              "w-6 h-6 rounded-lg flex items-center justify-center transition-all backdrop-blur-md border",
              isSelected
                ? "bg-emerald-500 text-black border-emerald-400 shadow-sm"
                : selectionMode
                ? "bg-black/60 border-white/30 text-white/40 hover:text-white hover:border-white/60"
                : "opacity-0 group-hover:opacity-100 bg-black/60 border-white/20 text-white/40 hover:text-white"
            )}
            title={isSelected ? "Deselect item" : "Select item"}
          >
            {isSelected ? (
              <Check className="w-4 h-4 stroke-[3]" />
            ) : (
              <Square className="w-3.5 h-3.5 opacity-60" />
            )}
          </button>

          <div className="flex items-center gap-1.5">
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
        </div>

        {/* Bottom Details */}
        <div className="relative z-10 p-3 bg-gradient-to-t from-black/95 to-transparent">
          <p className="text-xs font-semibold text-white truncate group-hover:text-emerald-300 transition-colors">
            {item.filename}
          </p>

          <div className="flex items-center justify-between text-[10px] text-zinc-400 mt-1 font-mono">
            <span className="flex items-center gap-1 truncate text-zinc-400">
              <Clock className="w-3 h-3 text-zinc-500 flex-shrink-0" />
              <span>{formatTimeOnly(item.createdAt)}</span>
            </span>

            {/* Quick Action Buttons: Copy, Download, Delete */}
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

              <button
                onClick={(e) => openSingleDelete(item, e)}
                className="w-6 h-6 rounded-lg bg-rose-500/15 hover:bg-rose-500/30 text-rose-300 hover:text-rose-100 flex items-center justify-center transition-all"
                title="Delete file permanently"
              >
                <Trash2 className="w-3 h-3" />
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-24 pb-36">
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
            All captured media is permanently preserved locally on your machine with zero expiration, grouped cleanly by download phase.
          </p>
        </div>

        {/* Stats Pill Deck */}
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2 px-3.5 py-2 rounded-xl bg-white/[0.03] border border-white/[0.08]">
            <FolderArchive className="w-4 h-4 text-emerald-400" />
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 font-mono uppercase">Total Files</span>
              <span className="text-xs font-bold text-white font-mono">{totalCount || items.length}</span>
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
            <Calendar className="w-4 h-4 text-emerald-400" />
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 font-mono uppercase">Active Phases</span>
              <span className="text-xs font-bold text-emerald-300 font-mono">
                {dateGroups.length} {dateGroups.length === 1 ? "Day" : "Days"}
              </span>
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
      <div className="flex flex-col lg:flex-row items-stretch lg:items-center justify-between gap-4 my-6">
        {/* Search */}
        <div className="relative w-full lg:w-72">
          <Search className="w-4 h-4 text-zinc-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search filenames, IDs, or links..."
            className="w-full h-10 pl-10 pr-4 rounded-xl bg-white/[0.04] border border-white/[0.08] text-sm text-white placeholder-zinc-500 outline-none focus:border-emerald-500/50 transition-colors"
          />
        </div>

        {/* Filter Toolbar: Media Type + Date Filter + Multi-Select */}
        <div className="flex flex-wrap items-center gap-2.5 justify-between lg:justify-end">
          {/* Media Type Filter Pills */}
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

          {/* Date Phase Filter Dropdown */}
          <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-white/[0.03] border border-white/[0.08] text-xs">
            <Calendar className="w-3.5 h-3.5 text-emerald-400" />
            <select
              value={dateFilter}
              onChange={(e) => setDateFilter(e.target.value)}
              className="bg-transparent text-white font-semibold outline-none cursor-pointer pr-1"
            >
              <option value="all" className="bg-[#161720] text-white">All Dates</option>
              <option value="today" className="bg-[#161720] text-white">Today</option>
              <option value="yesterday" className="bg-[#161720] text-white">Yesterday</option>
              <option value="week" className="bg-[#161720] text-white">Past 7 Days</option>
              <option value="month" className="bg-[#161720] text-white">Past 30 Days</option>
              {availableDates.length > 0 && (
                <optgroup label="Specific Days" className="bg-[#161720] text-zinc-400 font-bold">
                  {availableDates.map((d) => (
                    <option key={d.dateKey} value={d.dateKey} className="bg-[#161720] text-white">
                      {d.label}
                    </option>
                  ))}
                </optgroup>
              )}
            </select>
          </div>

          {/* Toggle Multi-Select Mode */}
          <button
            onClick={() => {
              if (selectionMode) {
                handleClearSelection();
              } else {
                setSelectionMode(true);
              }
            }}
            className={cn(
              "h-10 px-3.5 rounded-xl border text-xs font-bold flex items-center gap-2 transition-all",
              selectionMode
                ? "bg-emerald-500/20 border-emerald-500/40 text-emerald-300"
                : "bg-white/[0.04] border-white/[0.08] text-zinc-300 hover:text-white hover:bg-white/[0.08]"
            )}
            title={selectionMode ? "Exit Select Mode" : "Select Multiple Files"}
          >
            <CheckSquare className="w-4 h-4 text-emerald-400" />
            <span>{selectionMode ? "Done Selecting" : "Select"}</span>
          </button>

          {/* Select All Button when in selection mode */}
          {selectionMode && filteredItems.length > 0 && (
            <button
              onClick={handleSelectAllFiltered}
              className="h-10 px-3 rounded-xl bg-white/[0.04] border border-white/[0.08] hover:bg-white/[0.08] text-xs font-semibold text-zinc-300 hover:text-white flex items-center gap-1.5 transition-all"
            >
              <CheckCheck className="w-4 h-4 text-emerald-400" />
              <span>{allVisibleSelected ? "Deselect All" : "Select All"}</span>
            </button>
          )}

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

      {/* Main Content Area: Grouped & Separated by Date */}
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
            {search || dateFilter !== "all"
              ? "No files match your search query or date filter."
              : "Capture stories or videos using the Extractor and they will appear here."}
          </p>
        </div>
      ) : (
        <div className="space-y-12">
          {dateGroups.map((group) => {
            const groupAllSelected =
              group.items.length > 0 && group.items.every((it) => selectedIds.has(it.id));

            return (
              <div key={group.dateKey} className="relative">
                {/* Date Phase Section Header */}
                <div className="flex flex-wrap items-center justify-between gap-3 pb-3 mb-5 border-b border-white/[0.08]">
                  <div className="flex items-center gap-3">
                    <div
                      className={cn(
                        "w-9 h-9 rounded-xl flex items-center justify-center border font-mono text-xs font-bold transition-colors",
                        group.isToday
                          ? "bg-emerald-500/15 border-emerald-500/30 text-emerald-400"
                          : "bg-white/[0.04] border-white/[0.08] text-zinc-300"
                      )}
                    >
                      <Calendar className="w-4 h-4" />
                    </div>

                    <div>
                      <div className="flex items-center gap-2">
                        <h2 className="text-base sm:text-lg font-bold text-white tracking-tight">
                          {group.title}
                        </h2>
                        {group.isToday && (
                          <span className="px-2 py-0.5 rounded-full bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 text-[10px] font-mono font-bold uppercase tracking-wider">
                            Today
                          </span>
                        )}
                        {group.isYesterday && (
                          <span className="px-2 py-0.5 rounded-full bg-blue-500/20 text-blue-300 border border-blue-500/30 text-[10px] font-mono font-bold uppercase tracking-wider">
                            Yesterday
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-zinc-400 font-mono">
                        {group.subtitle}
                      </p>
                    </div>
                  </div>

                  {/* Phase Stats & Select Day Button */}
                  <div className="flex items-center gap-2">
                    <span className="px-2.5 py-1 rounded-lg bg-white/[0.04] border border-white/[0.08] text-xs font-mono font-semibold text-zinc-300">
                      <span className="text-white font-bold">{group.items.length}</span>{" "}
                      {group.items.length === 1 ? "download" : "downloads"}
                    </span>

                    <span className="px-2.5 py-1 rounded-lg bg-white/[0.04] border border-white/[0.08] text-xs font-mono font-semibold text-zinc-400">
                      {formatBytes(group.totalBytes)}
                    </span>

                    <button
                      type="button"
                      onClick={() => toggleSelectDateGroup(group.items)}
                      className={cn(
                        "px-2.5 py-1 rounded-lg border text-xs font-semibold transition-all flex items-center gap-1.5",
                        groupAllSelected
                          ? "bg-emerald-500/20 border-emerald-500/40 text-emerald-300 font-bold"
                          : "bg-white/[0.04] hover:bg-white/[0.08] border-white/[0.08] text-zinc-400 hover:text-white"
                      )}
                      title="Select or deselect all downloads from this day"
                    >
                      <CheckSquare className="w-3.5 h-3.5 text-emerald-400" />
                      <span>{groupAllSelected ? "Deselect Day" : "Select Day"}</span>
                    </button>
                  </div>
                </div>

                {/* Grid for this specific date */}
                <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
                  {group.items.map((item) => renderCard(item))}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Floating Bottom Batch Action Bar */}
      <AnimatePresence>
        {selectedIds.size > 0 && (
          <motion.div
            initial={{ opacity: 0, y: 40 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 40 }}
            className="fixed bottom-6 inset-x-0 mx-auto w-full max-w-xl z-40 px-4"
          >
            <div className="p-3 sm:px-5 rounded-2xl bg-[#14151b]/95 backdrop-blur-xl border border-white/15 shadow-2xl shadow-black/80 flex items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                <span className="w-6 h-6 rounded-lg bg-emerald-500 text-black flex items-center justify-center text-xs font-bold font-mono">
                  {selectedIds.size}
                </span>
                <div className="flex flex-col">
                  <span className="text-xs font-bold text-white">Selected</span>
                  <span className="text-[10px] text-zinc-400 font-mono">
                    {formatBytes(selectedSize)}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={handleSelectAllFiltered}
                  className="px-3 py-1.5 rounded-xl bg-white/10 hover:bg-white/15 text-xs font-semibold text-zinc-300 hover:text-white transition-all"
                >
                  {allVisibleSelected ? "Deselect" : "Select All"}
                </button>

                <button
                  type="button"
                  onClick={openBatchDelete}
                  className="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-bold flex items-center gap-1.5 shadow-lg shadow-rose-600/30 hover:shadow-rose-600/50 transition-all"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  <span>Delete ({selectedIds.size})</span>
                </button>

                <button
                  type="button"
                  onClick={handleClearSelection}
                  className="w-8 h-8 rounded-xl bg-white/5 hover:bg-white/10 text-zinc-400 hover:text-white flex items-center justify-center transition-all ml-1"
                  title="Cancel selection"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

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
                <div className="truncate pr-4 flex items-center gap-3">
                  <div className="truncate">
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm sm:text-base font-bold text-white truncate">
                        {selectedMedia.filename}
                      </h3>
                      {currentIndex >= 0 && (
                        <span className="px-2 py-0.5 rounded-md bg-white/10 text-[11px] font-mono text-zinc-300 font-semibold flex-shrink-0">
                          {currentIndex + 1} / {filteredItems.length}
                        </span>
                      )}
                    </div>
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
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => openSingleDelete(selectedMedia)}
                    className="h-9 px-3 rounded-xl bg-rose-500/20 text-rose-300 hover:bg-rose-500/30 border border-rose-500/30 font-bold text-xs flex items-center gap-1.5 transition-all"
                    title="Delete this file"
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                    <span className="hidden sm:inline">Delete</span>
                  </button>

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
              <div className="relative flex-1 overflow-hidden bg-black flex items-center justify-center p-2 min-h-[360px] group/viewer">
                {/* Previous Button */}
                {filteredItems.length > 1 && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handlePrev();
                    }}
                    className="absolute left-3.5 z-20 w-11 h-11 rounded-full bg-black/60 hover:bg-black/90 text-white/80 hover:text-white border border-white/20 backdrop-blur-md flex items-center justify-center transition-all opacity-70 hover:opacity-100 hover:scale-105 shadow-lg shadow-black/50"
                    title="Previous media (Left Arrow ←)"
                  >
                    <ChevronLeft className="w-6 h-6 -ml-0.5" />
                  </button>
                )}

                {selectedMedia.mimeType.startsWith("video") ? (
                  <video
                    key={selectedMedia.id}
                    src={getMediaFileUrl(selectedMedia.id)}
                    controls
                    autoPlay
                    className="max-h-[70vh] w-auto max-w-full rounded-xl"
                  />
                ) : (
                  <img
                    key={selectedMedia.id}
                    src={getMediaFileUrl(selectedMedia.id)}
                    alt={selectedMedia.filename}
                    className="max-h-[70vh] w-auto max-w-full object-contain rounded-xl"
                  />
                )}

                {/* Next Button */}
                {filteredItems.length > 1 && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleNext();
                    }}
                    className="absolute right-3.5 z-20 w-11 h-11 rounded-full bg-black/60 hover:bg-black/90 text-white/80 hover:text-white border border-white/20 backdrop-blur-md flex items-center justify-center transition-all opacity-70 hover:opacity-100 hover:scale-105 shadow-lg shadow-black/50"
                    title="Next media (Right Arrow →)"
                  >
                    <ChevronRight className="w-6 h-6 -mr-0.5" />
                  </button>
                )}
              </div>

              {/* Modal Footer / Source & Keyboard Hints */}
              <div className="p-3 sm:px-6 bg-[#0c0d10] border-t border-white/[0.06] flex items-center justify-between text-xs text-zinc-400">
                <div className="flex items-center gap-3 truncate mr-2">
                  <div className="hidden sm:flex items-center gap-1.5 text-[11px] text-zinc-400 font-mono">
                    <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white border border-white/10 font-mono text-[10px]">←</kbd>
                    <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white border border-white/10 font-mono text-[10px]">→</kbd>
                    <span className="text-zinc-500">navigate</span>
                    <span className="text-zinc-600 ml-1">•</span>
                    <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-white border border-white/10 font-mono text-[10px] ml-1">ESC</kbd>
                    <span className="text-zinc-500">close</span>
                  </div>
                  {selectedMedia.sourceUrl && (
                    <span className="font-mono text-zinc-500 truncate hidden md:inline">
                      Source: {selectedMedia.sourceUrl}
                    </span>
                  )}
                </div>

                {selectedMedia.sourceUrl && (
                  <a
                    href={selectedMedia.sourceUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-emerald-400 hover:text-emerald-300 flex items-center gap-1 font-semibold flex-shrink-0"
                  >
                    <span>Original Post</span>
                    <ExternalLink className="w-3 h-3" />
                  </a>
                )}
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>

      {/* Delete Confirmation Modal */}
      <AnimatePresence>
        {deleteTarget && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            {/* Backdrop */}
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              onClick={() => {
                if (!isDeleting) setDeleteTarget(null);
              }}
              className="absolute inset-0 bg-black/80 backdrop-blur-sm"
            />

            {/* Dialog Card */}
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: 10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: 10 }}
              className="relative z-10 w-full max-w-md bg-[#161720] border border-white/15 rounded-3xl p-6 shadow-2xl text-left"
            >
              <div className="flex items-center gap-3.5 mb-4">
                <div className="w-11 h-11 rounded-2xl bg-rose-500/20 border border-rose-500/30 flex items-center justify-center text-rose-400 flex-shrink-0">
                  <AlertTriangle className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-white">
                    {deleteTarget.type === "single"
                      ? "Delete Media File?"
                      : `Delete ${deleteTarget.count} Media Files?`}
                  </h3>
                  <p className="text-xs text-zinc-400 mt-0.5">
                    This action will permanently remove the file(s) from disk.
                  </p>
                </div>
              </div>

              {deleteTarget.type === "single" ? (
                <div className="p-3 rounded-xl bg-white/[0.03] border border-white/[0.06] mb-5">
                  <p className="text-xs font-semibold text-white truncate">
                    {deleteTarget.item.filename}
                  </p>
                  <p className="text-[11px] text-zinc-400 font-mono mt-0.5">
                    Size: {formatBytes(deleteTarget.item.sizeBytes)}
                  </p>
                </div>
              ) : (
                <div className="p-3 rounded-xl bg-white/[0.03] border border-white/[0.06] mb-5">
                  <p className="text-xs font-semibold text-white">
                    {deleteTarget.count} files selected for permanent deletion
                  </p>
                  <p className="text-[11px] text-zinc-400 font-mono mt-0.5">
                    Total disk space to free: {formatBytes(deleteTarget.totalBytes)}
                  </p>
                </div>
              )}

              {deleteError && (
                <div className="mb-4 p-2.5 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-300 text-xs font-medium">
                  {deleteError}
                </div>
              )}

              <div className="flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  disabled={isDeleting}
                  onClick={() => setDeleteTarget(null)}
                  className="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/15 text-xs font-semibold text-zinc-300 hover:text-white transition-all disabled:opacity-50"
                >
                  Cancel
                </button>

                <button
                  type="button"
                  disabled={isDeleting}
                  onClick={confirmDelete}
                  className="px-4 py-2 rounded-xl bg-rose-600 hover:bg-rose-500 text-white text-xs font-bold flex items-center gap-1.5 shadow-md shadow-rose-600/30 hover:shadow-rose-600/50 transition-all disabled:opacity-50"
                >
                  {isDeleting ? (
                    <>
                      <Loader2 className="w-3.5 h-3.5 animate-spin" />
                      <span>Deleting...</span>
                    </>
                  ) : (
                    <>
                      <Trash2 className="w-3.5 h-3.5" />
                      <span>Delete Permanently</span>
                    </>
                  )}
                </button>
              </div>
            </motion.div>
          </div>
        )}
      </AnimatePresence>
    </div>
  );
};
