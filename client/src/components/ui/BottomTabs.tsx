import { NavLink } from "react-router-dom";
import { DownloadCloud, FolderArchive } from "lucide-react";

export const BottomTabs = () => {
  const tabs = [
    { path: "/home", icon: DownloadCloud, label: "Extractor" },
    { path: "/library", icon: FolderArchive, label: "Vault" },
  ];

  return (
    <div className="fixed bottom-5 left-1/2 -translate-x-1/2 z-40 md:hidden w-auto">
      <div className="pro-glass rounded-2xl flex items-center gap-1 p-1.5 shadow-2xl border border-white/[0.1]">
        {tabs.map((tab) => (
          <NavLink
            key={tab.path}
            to={tab.path}
            className={({ isActive }) =>
              `flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold transition-all ${
                isActive
                  ? "bg-emerald-500 text-black shadow-md shadow-emerald-500/25 font-bold"
                  : "text-zinc-400 hover:text-white hover:bg-white/[0.05]"
              }`
            }
          >
            <tab.icon className="w-4 h-4" />
            <span>{tab.label}</span>
          </NavLink>
        ))}
      </div>
    </div>
  );
};
