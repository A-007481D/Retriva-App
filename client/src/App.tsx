import { Routes, Route, Navigate } from "react-router-dom";
import { Navbar } from "./components/ui/Navbar";
import { BottomTabs } from "./components/ui/BottomTabs";
import { Home } from "./pages/Home";
import { Vault } from "./pages/Vault";

function App() {
  return (
    <div className="relative min-h-screen w-full bg-[#09090b] text-zinc-100 font-sans selection:bg-emerald-500/30 selection:text-emerald-200">
      {/* Subtle tech grid background */}
      <div className="fixed inset-0 z-0 bg-grid-subtle opacity-70 pointer-events-none" />

      {/* Top Navbar */}
      <Navbar />

      {/* Main Content Area */}
      <main className="relative z-10 min-h-screen">
        <Routes>
          <Route path="/home" element={<Home />} />
          <Route path="/library" element={<Vault />} />
          <Route path="/vault" element={<Navigate to="/library" replace />} />
          <Route path="*" element={<Navigate to="/home" replace />} />
        </Routes>
      </main>

      {/* Mobile Floating Tab Navigation */}
      <BottomTabs />
    </div>
  );
}

export default App;
