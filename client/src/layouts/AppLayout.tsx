import { Outlet } from "react-router-dom"
import { Sidebar } from "../components/Sidebar"

export function AppLayout() {
  return (
    <div className="flex h-screen bg-slate-900 overflow-hidden text-slate-50 relative selection:bg-indigo-500/30">
      {/* Animated background blob */}
      <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/20 rounded-full blur-[120px] pointer-events-none" />
      <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-indigo-600/20 rounded-full blur-[120px] pointer-events-none" />

      {/* Sidebar */}
      <Sidebar />

      {/* Main Content Area */}
      <main className="flex-1 relative z-10 overflow-y-auto overflow-x-hidden">
        <div className="p-10 max-w-7xl mx-auto w-full">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
