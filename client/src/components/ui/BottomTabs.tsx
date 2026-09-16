import { NavLink } from "react-router-dom";
import { motion } from "framer-motion";
import { Home, Compass, Settings } from "lucide-react";

export const BottomTabs = () => {
  const tabs = [
    { path: "/home", icon: Home, label: "Home" },
    { path: "/library", icon: Compass, label: "Explore" },
    { path: "/settings", icon: Settings, label: "Settings" },
  ];

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 w-[90%] max-w-sm z-50">
      <div className="glass-panel flex items-center justify-between px-6 py-4">
        {tabs.map((tab) => (
          <NavLink
            key={tab.path}
            to={tab.path}
            className={({ isActive }) =>
              `relative flex flex-col items-center gap-1 transition-colors duration-300 ${
                isActive ? "text-cyan-400" : "text-white/60 hover:text-white/80"
              }`
            }
          >
            {({ isActive }) => (
              <>
                <tab.icon className="w-6 h-6" strokeWidth={isActive ? 2.5 : 2} />
                <span className="text-[10px] font-medium tracking-wide">
                  {tab.label}
                </span>
                
                {/* Active Indicator Line */}
                {isActive && (
                  <motion.div
                    layoutId="activeTabIndicator"
                    className="absolute -bottom-2 w-4 h-[3px] bg-cyan-400 rounded-full shadow-[0_0_8px_rgba(34,211,238,0.6)]"
                    transition={{ type: "spring", stiffness: 300, damping: 25 }}
                  />
                )}
              </>
            )}
          </NavLink>
        ))}
      </div>
    </div>
  );
};
