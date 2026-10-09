import React, { useEffect, useState } from "react";
import { NavLink } from "react-router-dom";
import { RetrivaLogo } from "./RetrivaLogo";
import { DownloadCloud, FolderArchive, ShieldCheck } from "lucide-react";
import { getVault } from "../../lib/api";

export const Navbar: React.FC = () => {
  const [vaultCount, setVaultCount] = useState<number | null>(null);

  useEffect(() => {
    let isMounted = true;
    const fetchCount = async () => {
      try {
        const res = await getVault();
        if (isMounted) {
          setVaultCount(res.total ?? (res.items ? res.items.length : 0));
        }
      } catch {
        // silent fail
      }
    };
    fetchCount();
    const interval = setInterval(fetchCount, 15000);
    return () => {
      isMounted = false;
      clearInterval(interval);
    };
  }, []);

  return (
    <header className="fixed top-0 left-0 right-0 z-40 h-16 pro-glass border-b border-white/[0.07] px-4 md:px-8 flex items-center justify-between">
      {/* Brand */}
      <NavLink to="/home" className="flex items-center gap-3 group">
        <RetrivaLogo size={32} />
      </NavLink>

      {/* Navigation Links */}
      <nav className="flex items-center gap-1 bg-white/[0.04] p-1 rounded-xl border border-white/[0.08]">
        <NavLink
          to="/home"
          className={({ isActive }) =>
            `flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-all ${
              isActive
                ? "bg-white/[0.12] text-white shadow-sm shadow-black/40"
                : "text-zinc-400 hover:text-zinc-200 hover:bg-white/[0.04]"
            }`
          }
        >
          <DownloadCloud className="w-3.5 h-3.5" />
          <span>Extractor</span>
        </NavLink>

        <NavLink
          to="/library"
          className={({ isActive }) =>
            `flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-semibold transition-all ${
              isActive
                ? "bg-white/[0.12] text-white shadow-sm shadow-black/40"
                : "text-zinc-400 hover:text-zinc-200 hover:bg-white/[0.04]"
            }`
          }
        >
          <FolderArchive className="w-3.5 h-3.5" />
          <span>Vault</span>
          {vaultCount !== null && (
            <span className="px-1.5 py-0.2 rounded bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 text-[10px] font-mono font-bold">
              {vaultCount}
            </span>
          )}
        </NavLink>
      </nav>

      {/* Status Badge */}
      <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-[11px] font-mono text-emerald-300">
        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
        <ShieldCheck className="w-3.5 h-3.5 text-emerald-400" />
        <span>Permanent Retention</span>
      </div>
    </header>
  );
};
