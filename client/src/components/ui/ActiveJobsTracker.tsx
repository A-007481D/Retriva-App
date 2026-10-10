import React from "react";
import {
  Loader2,
  CheckCircle2,
  AlertCircle,
  X,
  RotateCw,
  Download,
  FolderArchive,
  ArrowDownCircle,
  Video,
} from "lucide-react";
import { type Job, getMediaFileUrl } from "../../lib/api";
import { cn } from "../../lib/utils";
import { NavLink } from "react-router-dom";

interface ActiveJobsTrackerProps {
  jobs: Job[];
  onDismiss: (id: string) => void;
  onRetry: (job: Job) => void;
  onClearFinished: () => void;
}

export const ActiveJobsTracker: React.FC<ActiveJobsTrackerProps> = ({
  jobs,
  onDismiss,
  onRetry,
  onClearFinished,
}) => {
  if (jobs.length === 0) return null;

  const activeCount = jobs.filter(
    (j) => j.status === "queued" || j.status === "resolving" || j.status === "downloading"
  ).length;

  const finishedCount = jobs.length - activeCount;

  return (
    <div className="w-full max-w-4xl mx-auto space-y-3 animate-in fade-in slide-in-from-bottom-2 duration-300">
      {/* Header Bar */}
      <div className="flex items-center justify-between px-1">
        <div className="flex items-center gap-2.5">
          <div className="relative flex items-center justify-center">
            {activeCount > 0 ? (
              <>
                <span className="animate-ping absolute inline-flex h-3 w-3 rounded-full bg-emerald-400 opacity-75" />
                <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-400" />
              </>
            ) : (
              <span className="inline-flex rounded-full h-2 w-2 bg-zinc-500" />
            )}
          </div>
          <h3 className="text-xs font-bold uppercase tracking-wider text-zinc-200 font-mono flex items-center gap-2">
            <span>Transfers & Downloads</span>
            {activeCount > 0 ? (
              <span className="px-2 py-0.5 rounded-full text-[10px] bg-emerald-500/15 border border-emerald-500/30 text-emerald-400 font-sans font-semibold">
                {activeCount} active
              </span>
            ) : (
              <span className="px-2 py-0.5 rounded-full text-[10px] bg-zinc-800 border border-zinc-700 text-zinc-400 font-sans font-medium">
                All finished
              </span>
            )}
          </h3>
        </div>

        {finishedCount > 0 && (
          <button
            onClick={onClearFinished}
            className="text-[11px] text-zinc-400 hover:text-zinc-200 font-mono font-medium transition-colors cursor-pointer"
          >
            Clear finished ({finishedCount})
          </button>
        )}
      </div>

      {/* Jobs Cards Container */}
      <div className="space-y-2.5">
        {jobs.map((job) => {
          const isPending =
            job.status === "queued" ||
            job.status === "resolving" ||
            job.status === "downloading";
          const isSuccess = job.status === "completed";
          const isFailed = job.status === "failed";

          const displayName =
            job.filename && job.filename !== "resolving..."
              ? job.filename
              : job.sourceUrl || `Task #${job.id.slice(-6)}`;

          const isVideo =
            displayName.endsWith(".mp4") ||
            displayName.endsWith(".mkv") ||
            displayName.endsWith(".webm");

          return (
            <div
              key={job.id}
              className={cn(
                "relative rounded-xl border p-3.5 sm:p-4 transition-all duration-200 backdrop-blur-md",
                isPending && "bg-[#111319]/90 border-emerald-500/30 shadow-[0_0_15px_rgba(16,185,129,0.05)]",
                isSuccess && "bg-[#101416]/80 border-emerald-500/20",
                isFailed && "bg-[#181113]/80 border-rose-500/30"
              )}
            >
              {/* Main Info Row */}
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3 min-w-0">
                  {/* Media / Task Icon */}
                  <div
                    className={cn(
                      "w-8 h-8 rounded-lg flex items-center justify-center shrink-0 border",
                      isPending && "bg-emerald-500/10 border-emerald-500/20 text-emerald-400",
                      isSuccess && "bg-emerald-500/15 border-emerald-500/30 text-emerald-300",
                      isFailed && "bg-rose-500/15 border-rose-500/30 text-rose-400"
                    )}
                  >
                    {isPending ? (
                      job.status === "resolving" ? (
                        <Loader2 className="w-4 h-4 animate-spin text-sky-400" />
                      ) : (
                        <ArrowDownCircle className="w-4 h-4 animate-pulse text-emerald-400" />
                      )
                    ) : isSuccess ? (
                      isVideo ? (
                        <Video className="w-4 h-4 text-emerald-400" />
                      ) : (
                        <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                      )
                    ) : (
                      <AlertCircle className="w-4 h-4 text-rose-400" />
                    )}
                  </div>

                  {/* Title & Status */}
                  <div className="min-w-0">
                    <p
                      className="text-xs sm:text-sm font-semibold text-white truncate max-w-[200px] sm:max-w-md md:max-w-lg"
                      title={displayName}
                    >
                      {displayName}
                    </p>
                    <div className="flex items-center gap-2 mt-0.5">
                      {job.status === "queued" && (
                        <span className="text-[11px] text-amber-400 flex items-center gap-1 font-mono">
                          <span className="inline-block w-1.5 h-1.5 rounded-full bg-amber-400" />
                          Queued in worker pool...
                        </span>
                      )}
                      {job.status === "resolving" && (
                        <span className="text-[11px] text-sky-400 flex items-center gap-1 font-mono">
                          <Loader2 className="w-3 h-3 animate-spin inline" />
                          Resolving stream URL...
                        </span>
                      )}
                      {job.status === "downloading" && (
                        <span className="text-[11px] text-emerald-400 flex items-center gap-1.5 font-mono font-medium">
                          <span>Downloading</span>
                          <span>—</span>
                          <span className="text-white font-bold">
                            {Math.round(job.progress)}%
                          </span>
                        </span>
                      )}
                      {isSuccess && (
                        <span className="text-[11px] text-emerald-400 font-mono flex items-center gap-1">
                          <CheckCircle2 className="w-3 h-3 text-emerald-400" />
                          Saved to Vault
                        </span>
                      )}
                      {isFailed && (
                        <span className="text-[11px] text-rose-400 font-mono font-medium flex items-center gap-1">
                          <AlertCircle className="w-3 h-3 text-rose-400" />
                          Download Failed
                        </span>
                      )}
                    </div>
                  </div>
                </div>

                {/* Action Buttons */}
                <div className="flex items-center gap-2 shrink-0">
                  {isSuccess && job.mediaId && (
                    <>
                      <a
                        href={getMediaFileUrl(job.mediaId)}
                        download
                        className="px-2.5 py-1 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
                        title="Download file to computer"
                      >
                        <Download className="w-3.5 h-3.5" />
                        <span className="hidden sm:inline">Download</span>
                      </a>
                      <NavLink
                        to="/library"
                        className="px-2.5 py-1 rounded-lg bg-white/[0.04] hover:bg-white/[0.08] border border-white/[0.1] text-zinc-300 text-xs font-medium flex items-center gap-1.5 transition-colors"
                        title="View in Vault"
                      >
                        <FolderArchive className="w-3.5 h-3.5 text-zinc-400" />
                        <span className="hidden sm:inline">Vault</span>
                      </NavLink>
                    </>
                  )}

                  {isFailed && (
                    <button
                      onClick={() => onRetry(job)}
                      className="px-2.5 py-1 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 border border-rose-500/30 text-rose-300 text-xs font-semibold flex items-center gap-1.5 transition-colors cursor-pointer"
                      title="Retry this download"
                    >
                      <RotateCw className="w-3.5 h-3.5" />
                      <span>Retry</span>
                    </button>
                  )}

                  <button
                    onClick={() => onDismiss(job.id)}
                    className="p-1 rounded-md text-zinc-500 hover:text-zinc-300 hover:bg-white/[0.06] transition-colors cursor-pointer"
                    title="Dismiss"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Progress Bar (when downloading or queued) */}
              {isPending && (
                <div className="mt-3 space-y-1.5">
                  <div className="w-full bg-zinc-800/80 rounded-full h-1.5 overflow-hidden">
                    <div
                      className={cn(
                        "h-full rounded-full transition-all duration-300 ease-out",
                        job.status === "downloading"
                          ? "bg-gradient-to-r from-emerald-500 via-teal-400 to-emerald-400"
                          : "bg-gradient-to-r from-amber-500 to-sky-400 w-1/4 animate-pulse"
                      )}
                      style={{
                        width:
                          job.status === "downloading"
                            ? `${Math.max(job.progress, 5)}%`
                            : undefined,
                      }}
                    />
                  </div>
                </div>
              )}

              {/* Detailed Error Box (when failed) */}
              {isFailed && (
                <div className="mt-2.5 text-[11px] font-mono text-rose-300 bg-rose-950/30 border border-rose-500/20 rounded-lg p-2.5 flex items-start gap-2 break-all">
                  <AlertCircle className="w-3.5 h-3.5 text-rose-400 mt-0.5 shrink-0" />
                  <div className="space-y-0.5">
                    <span className="font-semibold text-rose-200">Error Details: </span>
                    <span>{job.error || "The download operation timed out or was rejected by source server."}</span>
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};
