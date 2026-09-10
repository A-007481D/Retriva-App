import { Routes, Route, NavLink } from 'react-router-dom'
import { DownloadCloud, History, LayoutGrid } from 'lucide-react'

// Placeholder views for Phase 10 (we will build them properly in later phases)
const Home = () => (
  <div className="flex flex-col items-center mt-16">
    <div className="glass max-w-2xl w-full p-10 text-center">
      <h1 className="text-4xl font-bold mb-4">Download Media</h1>
      <p className="text-slate-400 mb-8">Paste any supported URL to download a temporary server copy, then save to your device.</p>
      
      <form className="flex gap-2">
        <input 
          type="url" 
          placeholder="https://..." 
          className="input flex-1"
          required
        />
        <button type="submit" className="btn btn-primary">
          <DownloadCloud className="w-5 h-5 mr-2" />
          Download
        </button>
      </form>
    </div>
  </div>
)

const Vault = () => (
  <div className="mt-8">
    <h2 className="text-3xl font-bold mb-6">Your Vault</h2>
    <p className="text-slate-400">Active media files currently stored on the server.</p>
  </div>
)

const HistoryView = () => (
  <div className="mt-8">
    <h2 className="text-3xl font-bold mb-6">Download History</h2>
    <div className="glass p-6">
      <p className="text-slate-400">No history yet.</p>
    </div>
  </div>
)

function App() {
  return (
    <>
      <header className="py-6 border-b border-white/5 mb-8">
        <div className="container mx-auto px-4 flex justify-between items-center">
          <NavLink to="/" className="text-2xl font-bold flex items-center gap-2 text-gradient">
            Retriva
          </NavLink>
          <nav className="flex gap-6">
            <NavLink 
              to="/" 
              className={({isActive}) => `font-medium transition-colors ${isActive ? 'text-slate-50' : 'text-slate-400 hover:text-slate-200'}`}
            >
              <span className="flex items-center gap-2">
                <DownloadCloud className="w-4 h-4" /> Home
              </span>
            </NavLink>
            <NavLink 
              to="/vault" 
              className={({isActive}) => `font-medium transition-colors ${isActive ? 'text-slate-50' : 'text-slate-400 hover:text-slate-200'}`}
            >
              <span className="flex items-center gap-2">
                <LayoutGrid className="w-4 h-4" /> Vault
              </span>
            </NavLink>
            <NavLink 
              to="/history" 
              className={({isActive}) => `font-medium transition-colors ${isActive ? 'text-slate-50' : 'text-slate-400 hover:text-slate-200'}`}
            >
              <span className="flex items-center gap-2">
                <History className="w-4 h-4" /> History
              </span>
            </NavLink>
          </nav>
        </div>
      </header>

      <main className="container mx-auto px-4 flex-1">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/vault" element={<Vault />} />
          <Route path="/history" element={<HistoryView />} />
        </Routes>
      </main>
      
      <footer className="mt-16 py-8 border-t border-white/5 text-center text-slate-500 text-sm">
        &copy; {new Date().getFullYear()} Retriva. Personal Universal Media Downloader.
      </footer>
    </>
  )
}

export default App
