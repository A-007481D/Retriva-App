import React from 'react'
import ReactDOM from 'react-dom/client'
import './index.css'

// Placeholder app — full routing and pages added in Phase 11
function App() {
  return (
    <div className="flex items-center justify-center min-h-dvh">
      <div className="text-center animate-fade-in">
        <div className="text-6xl mb-4">▼</div>
        <h1 className="text-4xl font-bold text-indigo-400 mb-2 tracking-tight">
          Retriva
        </h1>
        <p className="text-slate-400 text-sm">
          Personal universal media downloader
        </p>
      </div>
    </div>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
