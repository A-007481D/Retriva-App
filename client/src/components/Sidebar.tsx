import { NavLink } from "react-router-dom"
import { motion } from "framer-motion"
import { Download, LayoutGrid, History, Settings, User } from "lucide-react"
import { cn } from "../lib/utils"

const navItems = [
  { icon: Download, label: "Downloads", path: "/" },
  { icon: LayoutGrid, label: "Vault", path: "/vault" },
  { icon: History, label: "History", path: "/history" },
  { icon: Settings, label: "Settings", path: "/settings" },
  { icon: User, label: "Profile", path: "/profile" },
]

export function Sidebar() {
  return (
    <div className="w-64 h-screen flex flex-col glass border-r border-white/5 relative z-10">
      {/* Brand */}
      <div className="p-6 flex items-center gap-3">
        <div className="w-8 h-8 rounded-xl bg-gradient-to-br from-blue-400 to-indigo-600 flex items-center justify-center text-white font-bold text-xl shadow-lg shadow-indigo-500/20">
          R
        </div>
        <span className="text-xl font-bold tracking-tight text-white">RETRIVA</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-4 py-2 space-y-1">
        {navItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 px-4 py-3 rounded-2xl transition-all duration-300 relative group overflow-hidden",
                isActive ? "text-white" : "text-slate-400 hover:text-slate-200 hover:bg-white/5"
              )
            }
          >
            {({ isActive }) => (
              <>
                {isActive && (
                  <motion.div
                    layoutId="activeTab"
                    className="absolute inset-0 bg-gradient-to-r from-blue-500/20 to-indigo-600/20 backdrop-blur-md border border-white/10 rounded-2xl"
                    initial={false}
                    transition={{ type: "spring", stiffness: 300, damping: 30 }}
                  />
                )}
                <item.icon className={cn("w-5 h-5 relative z-10", isActive && "text-blue-400")} />
                <span className="font-medium relative z-10">{item.label}</span>
              </>
            )}
          </NavLink>
        ))}
      </nav>

      {/* User Profile Snippet */}
      <div className="p-6 mt-auto">
        <div className="flex items-center gap-3 p-3 rounded-2xl hover:bg-white/5 transition-colors cursor-pointer">
          <div className="w-10 h-10 rounded-full bg-slate-800 border border-white/10 flex items-center justify-center text-slate-300">
            <User className="w-5 h-5" />
          </div>
          <div>
            <div className="text-sm font-medium text-white">User Profile</div>
            <div className="text-xs text-slate-500">Local Instance</div>
          </div>
        </div>
      </div>
    </div>
  )
}
