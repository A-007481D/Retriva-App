import { Routes, Route, Navigate } from "react-router-dom";
import DotField from "./components/ui/DotField";
import { BottomTabs } from "./components/ui/BottomTabs";

import { Home } from "./pages/Home";

// Placeholders for views
const Library = () => <div className="p-6 pt-16 h-full text-center text-white">Library View</div>;
const Settings = () => <div className="p-6 pt-16 h-full text-center text-white">Settings View</div>;

function App() {
  return (
    <div className="relative min-h-screen w-full font-sans selection:bg-mesh-2/30">
      {/* Glowing aurora background */}
      <div className="fixed inset-0 z-0 overflow-hidden">
        <div className="glow-orb-1 top-[-10%] left-[-10%]" />
        <div className="glow-orb-2 bottom-[-5%] right-[-10%]" />
        <div className="glow-orb-3 top-[30%] left-[40%]" />
      </div>

      {/* Dot field on top of glow */}
      <div className="fixed inset-0 z-[1]">
        <DotField
          dotRadius={1.5}
          dotSpacing={14}
          bulgeStrength={67}
          glowRadius={160}
          sparkle={false}
          waveAmplitude={0}
        />
      </div>

      {/* Main Content Area */}
      <main className="pb-24 min-h-screen relative z-10">
        <Routes>
          <Route path="/home" element={<Home />} />
          <Route path="/library" element={<Library />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/home" replace />} />
        </Routes>
      </main>

      {/* Floating Navigation */}
      <BottomTabs />
    </div>
  );
}

export default App;
